package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

const notificationConnector = "notifications"

// pusher is the minimal capability NotificationProjector needs from
// internal/push. *push.Service satisfies it structurally; tests can
// substitute a recorder without wiring a real Firebase sender.
type pusher interface {
	NotifyUser(ctx context.Context, userID string, message push.Message) error
}

// NotificationProjector implements README §6.8's canonical pipeline from
// "notification projector/worker" onward. The upstream half ("domain
// transaction -> transactional outbox") already exists
// (db/migrations/000014_v11_realtime_outbox.sql); the downstream adapter
// ("FCM/APNs adapter") already exists (internal/push.Service.NotifyUser).
// This type is everything in between: it reads domain_outbox_events through
// the same durable connector-cursor mechanism realtime delivery already
// uses (RealtimeOutboxStore), recognizes a small, explicit set of event
// types, dedupes on notification_deliveries.source_event_id, and resolves
// each recipient's decision via internal/notification.Decide before ever
// calling the adapter.
type NotificationProjector struct {
	pool               *pgxpool.Pool
	outbox             *RealtimeOutboxStore
	capabilities       *capability.Service
	pusher             pusher
	ttl                time.Duration
	frequencyCapMax    int
	frequencyCapWindow time.Duration
	// groupWindow is README §6.8's grouping/collapse window (see
	// internal/notification.Type.Groupable/Decide's own doc comments).
	// <= 0 disables grouping entirely, same "not configured" convention
	// frequencyCapMax already uses.
	groupWindow time.Duration
	now         func() time.Time
}

func NewNotificationProjector(pool *pgxpool.Pool, capabilities *capability.Service, sender pusher, ttl, frequencyCapWindow time.Duration, frequencyCapMax int, groupWindow time.Duration) (*NotificationProjector, error) {
	if pool == nil || capabilities == nil || ttl <= 0 {
		return nil, errors.New("invalid notification projector dependencies")
	}
	outbox, err := NewRealtimeOutboxStore(pool)
	if err != nil {
		return nil, err
	}
	return &NotificationProjector{
		pool: pool, outbox: outbox, capabilities: capabilities, pusher: sender,
		ttl: ttl, frequencyCapMax: frequencyCapMax, frequencyCapWindow: frequencyCapWindow,
		groupWindow: groupWindow,
		now:         func() time.Time { return time.Now().UTC() },
	}, nil
}

// candidate is what buildCandidate resolves a relevant outbox event into,
// before any capability/preference/quiet-hours/frequency-cap decision is
// applied to it.
type candidate struct {
	recipientID string
	typ         notification.Type
	slotID      *string
	title       string
	body        string
}

// ProcessBatch reads up to limit unconsumed outbox events, projects the
// ones this package recognizes into notification_deliveries (subject to
// capability/dedupe/TTL/quiet-hours/frequency-cap), attempts push delivery
// for anything Decide() clears, and advances the durable connector cursor
// exactly like any other outbox connector in this codebase. It returns the
// number of events consumed (recognized or not — an unrecognized event type
// still advances the cursor as Skipped, matching how connector semantics
// already work for the realtime feed).
//
// Callers are expected to serialize calls (the cmd/api polling loop calls
// this on a single ticker goroutine); this method does not itself guard
// against concurrent invocation racing the same connector cursor.
func (p *NotificationProjector) ProcessBatch(ctx context.Context, limit int) (int, error) {
	after, err := p.outbox.Cursor(ctx, notificationConnector)
	if err != nil {
		return 0, err
	}
	events, err := p.outbox.ListAfter(ctx, after, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, event := range events {
		if err := p.projectOne(ctx, event); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (p *NotificationProjector) projectOne(ctx context.Context, event realtime.Event) error {
	candidates, err := p.buildCandidates(ctx, event)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return p.outbox.Checkpoint(ctx, notificationConnector, event, realtime.OutcomeSkipped)
	}
	// One outbox event can now fan out to multiple recipients (a chat
	// message notifies every other participant) — the cursor still
	// advances exactly once for the event itself, after every recipient's
	// row is written, not once per recipient.
	for _, cand := range candidates {
		if err := p.projectCandidate(ctx, event, cand); err != nil {
			return err
		}
	}
	return p.outbox.Checkpoint(ctx, notificationConnector, event, realtime.OutcomeDelivered)
}

func (p *NotificationProjector) projectCandidate(ctx context.Context, event realtime.Event, cand *candidate) error {
	// Fail-closed by capability, per recipient (README §6.8: "capability
	// registry controls notifications and remains fail-closed by
	// default"). Disabled means no notification_deliveries row at all,
	// not a suppressed one — the feature does not exist for this
	// recipient, the same way other v1.1 capability gates behave.
	if !p.capabilities.Enabled(ctx, cand.recipientID, capability.Notifications) {
		return nil
	}

	now := p.now()
	prefs, err := p.loadPreferences(ctx, cand.recipientID)
	if err != nil {
		return err
	}
	var recentCount int
	if cand.typ.FrequencyCapped() && p.frequencyCapMax > 0 {
		if recentCount, err = p.recentFrequencyCappedCount(ctx, cand.recipientID, now); err != nil {
			return err
		}
	}
	var groupAnchor *time.Time
	if cand.typ.Groupable() && p.groupWindow > 0 {
		if groupAnchor, err = p.lastGroupAnchor(ctx, cand.recipientID, cand.slotID, cand.typ, now); err != nil {
			return err
		}
	}
	expiresAt := now.Add(p.ttl)
	outcome := notification.Decide(notification.DecisionInput{
		Type: cand.typ, Now: now, ExpiresAt: expiresAt, Preferences: prefs,
		RecentFrequencyCappedCount: recentCount, FrequencyCapMax: p.frequencyCapMax,
		LastGroupAnchorAt: groupAnchor, GroupWindow: p.groupWindow,
	})
	var deliveredAt any
	if outcome == "" {
		outcome = p.attemptPush(ctx, cand)
	}
	if outcome == notification.OutcomeSent {
		deliveredAt = now
	}

	id, err := identifier.NewUUID()
	if err != nil {
		return err
	}
	deepLink := notification.DeepLink(cand.typ, slotIDValue(cand.slotID))
	// ON CONFLICT DO NOTHING is the dedupe boundary README §6.8 requires
	// ("idempotency/dedupe is required per logical notification"): if this
	// event was already projected by an earlier run that crashed before
	// checkpointing its cursor, this INSERT is a harmless no-op. Keyed on
	// (source_event_id, user_id) rather than source_event_id alone
	// (migration 000036): one outbox event can now fan out to several
	// recipients, each still deduped independently.
	//
	// created_at is set explicitly to p.now() (this same decision's own
	// clock) rather than left to the column's DEFAULT now(): a real bug
	// found while building grouping (which reads created_at back to find
	// an anchor) — relying on the database's own wall clock is
	// indistinguishable from p.now() in production (both real time), but
	// makes deterministic time-travel tests (and this decision's own
	// internal consistency with expiresAt, which already uses p.now())
	// silently wrong. This changes no real-world behavior, only what a
	// test controlling p.now() actually observes.
	_, err = p.pool.Exec(ctx, `
		INSERT INTO notification_deliveries (
			id,user_id,notification_type,source_event_id,slot_id,deep_link,title,body,created_at,expires_at,push_outcome,delivered_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (source_event_id,user_id) DO NOTHING`,
		id, cand.recipientID, string(cand.typ), event.ID, cand.slotID, deepLink, cand.title, cand.body,
		now, expiresAt, string(outcome), deliveredAt,
	)
	return err
}

func (p *NotificationProjector) attemptPush(ctx context.Context, cand *candidate) notification.PushOutcome {
	if p.pusher == nil {
		// No push adapter wired at all in this deployment (push
		// registration/delivery not configured, see cfg.push.LoadRuntimeConfig).
		// The notification_deliveries row itself is still the correct
		// in-app/domain truth; only the push leg did not happen.
		return notification.OutcomeSkippedNoDevice
	}
	if err := p.pusher.NotifyUser(ctx, cand.recipientID, push.Message{
		Title: cand.title,
		Body:  cand.body,
		Data:  map[string]string{"type": string(cand.typ), "deepLink": notification.DeepLink(cand.typ, slotIDValue(cand.slotID))},
	}); err != nil {
		return notification.OutcomeSendFailed
	}
	// push.Service.NotifyUser also returns nil (indistinguishable from a
	// real send) when the recipient has zero registered device tokens;
	// OutcomeSent is therefore optimistic in that specific case rather
	// than a distinct SKIPPED_NO_DEVICE. Splitting that apart would
	// require extending push.Service's return contract, out of scope for
	// this block; the in-app notification record is correct either way.
	return notification.OutcomeSent
}

// buildCandidates recognizes exactly the outbox event types this
// codebase's notification work has wired through so far. Every other
// event type returns (nil, nil): the cursor still advances past it
// (Skipped), but no notification_deliveries row is created. This is a
// deliberately small, explicit allow-list — extending it to the remaining
// README §6.8 canonical types (EVENT_RECOMMENDATION, admin PROMO
// campaigns) is future work, not silently assumed done by this switch
// existing. FRIEND_REQUEST/FRIEND_ACCEPTED (internal/friend,
// friend_store.go) and slot.chat_message_created (MESSAGE, the one event
// type here that fans out to more than one recipient) were added in the
// same block as this comment's last edit.
func (p *NotificationProjector) buildCandidates(ctx context.Context, event realtime.Event) ([]*candidate, error) {
	switch event.Type {
	case "slot.membership_added":
		return oneOrNone(p.buildMembershipAddedCandidate(ctx, event))
	case "slot.request_created":
		return oneOrNone(p.buildRequestCreatedCandidate(ctx, event))
	case "slot.starting_soon":
		return oneOrNone(p.buildStartingSoonCandidate(ctx, event))
	case "friend.requested":
		return oneOrNone(p.buildFriendRequestedCandidate(ctx, event))
	case "friend.accepted":
		return oneOrNone(p.buildFriendAcceptedCandidate(ctx, event))
	case "slot.chat_message_created":
		return p.buildChatMessageCandidates(ctx, event)
	default:
		return nil, nil
	}
}

// oneOrNone wraps a single-candidate builder's (nil-safe) return into the
// slice shape buildCandidates now needs uniformly, without having to
// touch every existing single-recipient builder function's own signature.
func oneOrNone(c *candidate, err error) ([]*candidate, error) {
	if err != nil || c == nil {
		return nil, err
	}
	return []*candidate{c}, nil
}

// buildMembershipAddedCandidate covers every path that inserts a
// slot_memberships row: host approval, INSTANT/WAITLIST auto-admit, and
// WAITLIST promotion (see internal/postgres's chat SYSTEM-message
// equivalent, emitSystemChatMessageTx, which fires on the same set of call
// sites for the same reason: one INSERT INTO slot_memberships, one signal).
func (p *NotificationProjector) buildMembershipAddedCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	if event.SubjectUserID == nil || event.SlotID == nil {
		return nil, nil
	}
	var title string
	err := p.pool.QueryRow(ctx, `SELECT title FROM slots WHERE id=$1`, *event.SlotID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Slot no longer exists; nothing left to notify about.
	}
	if err != nil {
		return nil, err
	}
	return &candidate{
		recipientID: *event.SubjectUserID,
		typ:         notification.TypeEvent,
		slotID:      event.SlotID,
		title:       "You're in!",
		body:        "You joined " + title + ".",
	}, nil
}

// buildRequestCreatedCandidate notifies the host, not the requester
// (event.SubjectUserID), that someone wants to join. It deliberately does
// not attempt to cover request rejection/withdrawal from the generic
// slot.request_removed event: that event's trigger has no way to know
// whether a request row was removed by host rejection or by the
// requester's own withdrawal (slot_requests carries no "removed by" actor),
// so a notification built from it alone could tell a user who withdrew
// their own request that it was "not approved" — wrong information is
// worse than none, so that path is left unbuilt rather than built wrong.
func (p *NotificationProjector) buildRequestCreatedCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	if event.SubjectUserID == nil || event.SlotID == nil {
		return nil, nil
	}
	var hostID, slotTitle string
	err := p.pool.QueryRow(ctx, `SELECT host_id::text,title FROM slots WHERE id=$1`, *event.SlotID).Scan(&hostID, &slotTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var requesterName string
	err = p.pool.QueryRow(ctx, `SELECT display_name FROM app_users WHERE id=$1`, *event.SubjectUserID).Scan(&requesterName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &candidate{
		recipientID: hostID,
		typ:         notification.TypeEvent,
		slotID:      event.SlotID,
		title:       "New request",
		body:        requesterName + " wants to join " + slotTitle + ".",
	}, nil
}

// buildStartingSoonCandidate handles the one event type in this file that a
// DB trigger never emits: ReminderScanner (internal/postgres/
// reminder_scanner.go) emits it directly, already scoped to exactly one
// recipient per event (event.SubjectUserID), so unlike the trigger-driven
// candidates above there is no separate "who gets notified" decision to
// make here — only what to say.
func (p *NotificationProjector) buildStartingSoonCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	if event.SubjectUserID == nil || event.SlotID == nil {
		return nil, nil
	}
	var title string
	err := p.pool.QueryRow(ctx, `SELECT title FROM slots WHERE id=$1`, *event.SlotID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Slot no longer exists; nothing left to notify about.
	}
	if err != nil {
		return nil, err
	}
	return &candidate{
		recipientID: *event.SubjectUserID,
		typ:         notification.TypeEventReminder,
		slotID:      event.SlotID,
		title:       "Starting soon",
		body:        title + " is starting soon.",
	}, nil
}

// buildFriendRequestedCandidate notifies the target (event.SubjectUserID)
// that someone sent them a friend request. event.AggregateID is the
// friend_requests.id (FriendStore.Request emits this in the same
// transaction as the INSERT), so the requester's name is looked up through
// that row rather than needing it duplicated into the event payload.
func (p *NotificationProjector) buildFriendRequestedCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	if event.SubjectUserID == nil {
		return nil, nil
	}
	var requesterName string
	err := p.pool.QueryRow(ctx, `
		SELECT u.display_name FROM friend_requests r JOIN app_users u ON u.id=r.requester_id
		WHERE r.id=$1`, event.AggregateID).Scan(&requesterName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Request row (or requester) no longer exists.
	}
	if err != nil {
		return nil, err
	}
	return &candidate{
		recipientID: *event.SubjectUserID,
		typ:         notification.TypeFriendRequest,
		title:       "New friend request",
		body:        requesterName + " wants to connect.",
	}, nil
}

// buildFriendAcceptedCandidate notifies the ORIGINAL requester
// (event.SubjectUserID, set by FriendStore's acceptRequestRowTx) that their
// request was accepted — never the person who clicked accept, who already
// knows. The target's name (who accepted) is looked up the same way.
func (p *NotificationProjector) buildFriendAcceptedCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	if event.SubjectUserID == nil {
		return nil, nil
	}
	var targetName string
	err := p.pool.QueryRow(ctx, `
		SELECT u.display_name FROM friend_requests r JOIN app_users u ON u.id=r.target_id
		WHERE r.id=$1`, event.AggregateID).Scan(&targetName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &candidate{
		recipientID: *event.SubjectUserID,
		typ:         notification.TypeFriendAccepted,
		title:       "Friend request accepted",
		body:        targetName + " accepted your friend request.",
	}, nil
}

// buildChatMessageCandidates is this codebase's first MESSAGE-type
// notification candidate, and its first fan-out builder (one event, many
// recipients): a real gap found by reading buildCandidates' own
// allow-list before assuming it was already covered — no candidate
// builder ever produced notification.TypeMessage, so a chat message never
// generated a push notification at all despite README §6.8 listing
// "coordination events" among the required evidence. Recipients are every
// current chat-authorized participant of the Slot (host + accepted
// members) except the sender, with the same block exclusion
// ChatStore.ListRecent already applies — a blocked pair should not
// notify each other any more than they should see each other's messages.
// event.SubjectUserID is the trigger's NEW.author_id (migration 000014):
// nil for a SYSTEM message (author_id is NULL there), which this
// intentionally skips — attributing a notification to "nobody" would be
// wrong, and every SYSTEM chat event already has its own EVENT-type
// candidate elsewhere (buildMembershipAddedCandidate, etc.) covering the
// same underlying occurrence.
//
// The body deliberately never echoes the message's own text: this is a
// zero-trace chat product (README §6.7), and a push notification is
// exactly the kind of externally-visible, provider-retained surface that
// principle is meant to keep sensitive content out of.
func (p *NotificationProjector) buildChatMessageCandidates(ctx context.Context, event realtime.Event) ([]*candidate, error) {
	if event.SubjectUserID == nil || event.SlotID == nil {
		return nil, nil
	}
	var slotTitle string
	err := p.pool.QueryRow(ctx, `SELECT title FROM slots WHERE id=$1`, *event.SlotID).Scan(&slotTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Slot (and, by cascade, its messages) no longer exists.
	}
	if err != nil {
		return nil, err
	}
	var senderName string
	err = p.pool.QueryRow(ctx, `SELECT display_name FROM app_users WHERE id=$1`, *event.SubjectUserID).Scan(&senderName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, `
		SELECT participant_id FROM (
			SELECT host_id AS participant_id FROM slots WHERE id=$1
			UNION
			SELECT user_id AS participant_id FROM slot_memberships WHERE slot_id=$1
		) participants
		WHERE participant_id<>$2
		  AND NOT EXISTS (
			SELECT 1 FROM user_blocks b
			WHERE (b.blocker_id=participants.participant_id AND b.blocked_id=$2)
			   OR (b.blocker_id=$2 AND b.blocked_id=participants.participant_id)
		  )`, *event.SlotID, *event.SubjectUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candidates []*candidate
	for rows.Next() {
		var recipientID string
		if err := rows.Scan(&recipientID); err != nil {
			return nil, err
		}
		candidates = append(candidates, &candidate{
			recipientID: recipientID,
			typ:         notification.TypeMessage,
			slotID:      event.SlotID,
			title:       "New message",
			body:        senderName + " sent a message in " + slotTitle + ".",
		})
	}
	return candidates, rows.Err()
}

func (p *NotificationProjector) loadPreferences(ctx context.Context, userID string) (notification.Preferences, error) {
	var social, eventEnabled, recommendation, promo bool
	var startMin, endMin int
	var tzName string
	err := p.pool.QueryRow(ctx, `
		SELECT social_enabled,event_enabled,recommendation_enabled,promo_enabled,
		       quiet_hours_start_minute,quiet_hours_end_minute,timezone_name
		FROM notification_preferences WHERE user_id=$1`, userID).
		Scan(&social, &eventEnabled, &recommendation, &promo, &startMin, &endMin, &tzName)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.DefaultPreferences(), nil
	}
	if err != nil {
		return notification.Preferences{}, err
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		// A stored zone name that no longer resolves (or never validated)
		// fails closed to UTC rather than failing the whole projection —
		// quiet hours become UTC-relative for this one delivery instead of
		// blocking every notification this user would otherwise receive.
		loc = time.UTC
	}
	return notification.Preferences{
		SocialEnabled: social, EventEnabled: eventEnabled, RecommendationEnabled: recommendation, PromoEnabled: promo,
		QuietHoursStartMinute: startMin, QuietHoursEndMinute: endMin, Location: loc,
	}, nil
}

func (p *NotificationProjector) recentFrequencyCappedCount(ctx context.Context, userID string, now time.Time) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `
		SELECT count(*) FROM notification_deliveries
		WHERE user_id=$1
		  AND notification_type IN ('EVENT_RECOMMENDATION','PROMO','ADVERTISEMENT')
		  AND push_outcome='SENT'
		  AND created_at>$2`, userID, now.Add(-p.frequencyCapWindow)).Scan(&count)
	return count, err
}

// lastGroupAnchor finds when this recipient's most recent NON-grouped
// (i.e. actually decided, not itself a product of an earlier grouping
// suppression) notification of this exact (Slot, Type) pair was created —
// see DecisionInput.LastGroupAnchorAt's own doc comment for why "not
// itself grouped" matters (a fixed, non-overlapping window per burst, not
// a rolling one that could suppress forever under sustained traffic). A
// nil slotID (no Slot identity to group by at all) always returns no
// anchor; every candidate this codebase currently builds for a groupable
// Type always carries one.
func (p *NotificationProjector) lastGroupAnchor(ctx context.Context, userID string, slotID *string, typ notification.Type, now time.Time) (*time.Time, error) {
	if slotID == nil {
		return nil, nil
	}
	var anchor time.Time
	err := p.pool.QueryRow(ctx, `
		SELECT created_at FROM notification_deliveries
		WHERE user_id=$1 AND slot_id=$2 AND notification_type=$3
		  AND push_outcome<>'SUPPRESSED_GROUPED'
		ORDER BY created_at DESC LIMIT 1`, userID, *slotID, string(typ)).Scan(&anchor)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	anchor = anchor.UTC()
	return &anchor, nil
}

func slotIDValue(slotID *string) string {
	if slotID == nil {
		return ""
	}
	return *slotID
}
