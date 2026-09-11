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
	now                func() time.Time
}

func NewNotificationProjector(pool *pgxpool.Pool, capabilities *capability.Service, sender pusher, ttl, frequencyCapWindow time.Duration, frequencyCapMax int) (*NotificationProjector, error) {
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
		now: func() time.Time { return time.Now().UTC() },
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
	cand, err := p.buildCandidate(ctx, event)
	if err != nil {
		return err
	}
	if cand == nil {
		return p.outbox.Checkpoint(ctx, notificationConnector, event, realtime.OutcomeSkipped)
	}
	// Fail-closed by capability, per recipient (README §6.8: "capability
	// registry controls notifications and remains fail-closed by
	// default"). Disabled means no notification_deliveries row at all,
	// not a suppressed one — the feature does not exist for this
	// recipient, the same way other v1.1 capability gates behave.
	if !p.capabilities.Enabled(ctx, cand.recipientID, capability.Notifications) {
		return p.outbox.Checkpoint(ctx, notificationConnector, event, realtime.OutcomeSkipped)
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
	expiresAt := now.Add(p.ttl)
	outcome := notification.Decide(notification.DecisionInput{
		Type: cand.typ, Now: now, ExpiresAt: expiresAt, Preferences: prefs,
		RecentFrequencyCappedCount: recentCount, FrequencyCapMax: p.frequencyCapMax,
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
	// checkpointing its cursor, this INSERT is a harmless no-op and the
	// cursor still advances below.
	if _, err := p.pool.Exec(ctx, `
		INSERT INTO notification_deliveries (
			id,user_id,notification_type,source_event_id,slot_id,deep_link,title,body,expires_at,push_outcome,delivered_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (source_event_id) DO NOTHING`,
		id, cand.recipientID, string(cand.typ), event.ID, cand.slotID, deepLink, cand.title, cand.body,
		expiresAt, string(outcome), deliveredAt,
	); err != nil {
		return err
	}
	return p.outbox.Checkpoint(ctx, notificationConnector, event, realtime.OutcomeDelivered)
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

// buildCandidate recognizes exactly the outbox event types this codebase's
// notification work has wired through so far. Every other event type
// returns (nil, nil): the cursor still advances past it (Skipped), but no
// notification_deliveries row is created. This is a deliberately small,
// explicit allow-list — extending it to more of README §6.8's canonical
// types (EVENT_RECOMMENDATION, MESSAGE grouping, admin PROMO campaigns) is
// future work, not silently assumed done by this switch existing.
// FRIEND_REQUEST/FRIEND_ACCEPTED (internal/friend, friend_store.go) were
// added in the same block as this comment's last edit.
func (p *NotificationProjector) buildCandidate(ctx context.Context, event realtime.Event) (*candidate, error) {
	switch event.Type {
	case "slot.membership_added":
		return p.buildMembershipAddedCandidate(ctx, event)
	case "slot.request_created":
		return p.buildRequestCreatedCandidate(ctx, event)
	case "slot.starting_soon":
		return p.buildStartingSoonCandidate(ctx, event)
	case "friend.requested":
		return p.buildFriendRequestedCandidate(ctx, event)
	case "friend.accepted":
		return p.buildFriendAcceptedCandidate(ctx, event)
	default:
		return nil, nil
	}
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

func slotIDValue(slotID *string) string {
	if slotID == nil {
		return ""
	}
	return *slotID
}
