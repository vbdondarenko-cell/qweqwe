package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/guardian"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/onboarding"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/places"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type Dependencies struct {
	Accounts                *account.Service
	Blocks                  *blocklist.Service
	Slots                   *slot.Service
	Chats                   *chat.Service
	CityContext             *citycontext.Service
	Capabilities            *capability.Service
	Map                     *citymap.Service
	Places                  *places.Service
	Realtime                RealtimeFeed
	CityRealtime            CityRealtimeFeed
	Monetization            *monetization.Service
	Push                    *push.Service
	NotificationPreferences *notification.PreferencesService
	Bump                    *bump.Service
	Friends                 *friend.Service
	Guardians               *guardian.Service
	Onboarding              *onboarding.Service
	Telegram                onboarding.ContactPrompter
	TelegramWebhookSecret   string
	Ready                   func(context.Context) error
	AuthLimiter             *ratelimit.Limiter
	UserLimiter             *ratelimit.Limiter
	// MonetizationLimiter enforces docs/LINKUP_PLUS_MONETIZATION.md §8.2's
	// entitlement-sensitive rate limits (referral binding, rewarded-view
	// submission, purchase verification) — a separate, tighter budget from
	// UserLimiter's general per-authenticated-request limit, since these
	// specific endpoints can create or influence entitlement.
	MonetizationLimiter *ratelimit.Limiter
}

type Server struct {
	handler http.Handler
	deps    Dependencies
}

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type problem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type authContext struct {
	User      account.User
	SessionID string
	RawToken  string
}

type contextKey string

const authKey contextKey = "auth"
const requestIDKey contextKey = "request_id"

func New(deps Dependencies) *Server {
	s := &Server{deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", s.livez)
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.Handle("POST /v1/auth/register", s.authRateLimit(http.HandlerFunc(s.startOnboarding)))
	mux.Handle("POST /v1/auth/register/status", s.authRateLimit(http.HandlerFunc(s.onboardingStatus)))
	mux.Handle("POST /v1/auth/register/complete", s.authRateLimit(http.HandlerFunc(s.completeOnboarding)))
	mux.HandleFunc("POST /v1/integrations/telegram/onboarding", s.telegramOnboardingWebhook)
	mux.Handle("POST /v1/auth/login", s.authRateLimit(http.HandlerFunc(s.login)))
	mux.Handle("POST /v1/auth/recovery/request", s.authRateLimit(http.HandlerFunc(s.requestPasswordRecovery)))
	mux.Handle("POST /v1/auth/recovery/reset", s.authRateLimit(http.HandlerFunc(s.resetPassword)))
	mux.Handle("POST /v1/auth/logout", s.requireAuth(http.HandlerFunc(s.logout)))
	mux.Handle("GET /v1/me", s.requireAuth(http.HandlerFunc(s.getMe)))
	mux.Handle("GET /v1/capabilities", s.requireAuth(http.HandlerFunc(s.capabilities)))
	mux.Handle("PATCH /v1/me", s.requireAuth(http.HandlerFunc(s.patchMe)))
	mux.Handle("GET /v1/me/blocks", s.requireAuth(http.HandlerFunc(s.listBlocks)))
	mux.Handle("PUT /v1/me/blocks/{userID}", s.requireAuth(http.HandlerFunc(s.blockUser)))
	mux.Handle("DELETE /v1/me/blocks/{userID}", s.requireAuth(http.HandlerFunc(s.unblockUser)))
	mux.Handle("GET /v1/me/monetization", s.requireAuth(http.HandlerFunc(s.getMonetization)))
	mux.Handle("PUT /v1/me/referral", s.requireAuth(s.monetizationRateLimit(http.HandlerFunc(s.bindReferral))))
	mux.Handle("POST /v1/me/monetization/rewarded-views", s.requireAuth(s.monetizationRateLimit(http.HandlerFunc(s.submitRewardedView))))
	mux.Handle("POST /v1/me/monetization/purchases", s.requireAuth(s.monetizationRateLimit(http.HandlerFunc(s.verifyPurchase))))
	mux.Handle("PUT /v1/me/push/android", s.requireAuth(s.requireCapability(capability.Notifications, http.HandlerFunc(s.registerAndroidPush))))
	mux.Handle("DELETE /v1/me/push/android/{installationID}", s.requireAuth(s.requireCapability(capability.Notifications, http.HandlerFunc(s.revokeAndroidPush))))
	mux.Handle("GET /v1/me/notifications/preferences", s.requireAuth(s.requireCapability(capability.Notifications, http.HandlerFunc(s.getNotificationPreferences))))
	mux.Handle("PUT /v1/me/notifications/preferences", s.requireAuth(s.requireCapability(capability.Notifications, http.HandlerFunc(s.updateNotificationPreferences))))
	mux.Handle("POST /v1/slots/{slotID}/bump/challenge", s.requireAuth(s.requireCapability(capability.Bump, http.HandlerFunc(s.issueBumpChallenge))))
	mux.Handle("POST /v1/slots/{slotID}/bump/confirm", s.requireAuth(s.requireCapability(capability.Bump, http.HandlerFunc(s.confirmBump))))
	mux.Handle("GET /v1/me/reliability", s.requireAuth(s.requireCapability(capability.Bump, http.HandlerFunc(s.getReliability))))
	mux.Handle("GET /v1/me/bump-vault", s.requireAuth(s.requireCapability(capability.Bump, http.HandlerFunc(s.getBumpVault))))
	mux.Handle("GET /v1/users/{userID}/reliability-band", s.requireAuth(s.requireCapability(capability.Bump, http.HandlerFunc(s.getUserReliabilityBand))))
	mux.Handle("POST /v1/me/friends/requests/{userID}", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.sendFriendRequest))))
	mux.Handle("DELETE /v1/me/friends/requests/{userID}", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.cancelFriendRequest))))
	mux.Handle("POST /v1/me/friends/requests/{userID}/accept", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.acceptFriendRequest))))
	mux.Handle("POST /v1/me/friends/requests/{userID}/reject", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.rejectFriendRequest))))
	mux.Handle("GET /v1/me/friends/requests/incoming", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.listIncomingFriendRequests))))
	mux.Handle("GET /v1/me/friends/requests/outgoing", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.listOutgoingFriendRequests))))
	mux.Handle("GET /v1/me/friends", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.listFriends))))
	mux.Handle("DELETE /v1/me/friends/{userID}", s.requireAuth(s.requireCapability(capability.Friends, http.HandlerFunc(s.removeFriend))))

	mux.Handle("GET /v1/realtime/events", s.requireAuth(s.requireCapability(capability.Realtime, http.HandlerFunc(s.realtimeEvents))))
	mux.Handle("GET /v1/realtime/cursor", s.requireAuth(s.requireCapability(capability.Realtime, http.HandlerFunc(s.realtimeCursor))))
	mux.Handle("GET /v1/realtime/city", s.requireAuth(s.requireCapability(capability.Realtime, s.requireCapability(capability.CityContext, http.HandlerFunc(s.realtimeCity)))))
	mux.Handle("GET /v1/realtime/city/cursor", s.requireAuth(s.requireCapability(capability.Realtime, s.requireCapability(capability.CityContext, http.HandlerFunc(s.realtimeCityCursor)))))
	mux.Handle("GET /v1/city-context", s.requireAuth(s.requireCapability(capability.CityContext, http.HandlerFunc(s.getCityContext))))
	mux.Handle("POST /v1/city-context/resolve", s.requireAuth(s.requireCapability(capability.CityContext, http.HandlerFunc(s.resolveCityContext))))
	mux.Handle("GET /v1/places/search", s.requireAuth(http.HandlerFunc(s.searchPlaces)))
	mux.Handle("GET /v1/map", s.requireAuth(s.requireCapability(capability.Map, s.requireCapability(capability.CityContext, http.HandlerFunc(s.mapViewport)))))
	mux.Handle("GET /v1/map/places/{placeID}/slots", s.requireAuth(s.requireCapability(capability.Map, s.requireCapability(capability.CityContext, http.HandlerFunc(s.mapPlaceSlots)))))
	mux.Handle("POST /v1/slots/drafts", s.requireAuth(http.HandlerFunc(s.createDraftSlot)))
	mux.Handle("POST /v1/slots/{slotID}/publish", s.requireAuth(http.HandlerFunc(s.publishDraftSlot)))
	mux.Handle("POST /v1/slots", s.requireAuth(http.HandlerFunc(s.createSlot)))
	mux.Handle("GET /v1/slots/{slotID}", s.requireAuth(http.HandlerFunc(s.getSlot)))
	mux.Handle("PATCH /v1/slots/{slotID}", s.requireAuth(http.HandlerFunc(s.editSlot)))
	mux.Handle("POST /v1/slots/{slotID}/cancel", s.requireAuth(http.HandlerFunc(s.cancelSlot)))
	mux.Handle("POST /v1/slots/{slotID}/members/{userID}/remove", s.requireAuth(http.HandlerFunc(s.removeMember)))
	mux.Handle("GET /v1/slots/{slotID}/accepted", s.requireAuth(http.HandlerFunc(s.listAccepted)))
	mux.Handle("GET /v1/me/slots", s.requireAuth(http.HandlerFunc(s.listMySlots)))
	mux.Handle("GET /v1/pulse", s.requireAuth(http.HandlerFunc(s.listPulse)))
	mux.Handle("POST /v1/slots/{slotID}/join", s.requireAuth(http.HandlerFunc(s.joinSlot)))
	mux.Handle("POST /v1/slots/{slotID}/request", s.requireAuth(http.HandlerFunc(s.requestSlot)))
	mux.Handle("POST /v1/slots/{slotID}/leave", s.requireAuth(http.HandlerFunc(s.leaveSlot)))
	mux.Handle("GET /v1/slots/{slotID}/requests", s.requireAuth(http.HandlerFunc(s.listPendingRequests)))
	mux.Handle("POST /v1/slots/{slotID}/requests/{userID}/approve", s.requireAuth(http.HandlerFunc(s.approveRequest)))
	mux.Handle("POST /v1/slots/{slotID}/requests/{userID}/reject", s.requireAuth(http.HandlerFunc(s.rejectRequest)))
	mux.Handle("POST /v1/slots/{slotID}/start", s.requireAuth(http.HandlerFunc(s.startSlot)))
	mux.Handle("POST /v1/slots/{slotID}/complete", s.requireAuth(http.HandlerFunc(s.completeSlot)))
	mux.Handle("GET /v1/slots/{slotID}/chat/messages", s.requireAuth(http.HandlerFunc(s.listChatMessages)))
	mux.Handle("POST /v1/slots/{slotID}/chat/messages", s.requireAuth(http.HandlerFunc(s.sendChatMessage)))
	mux.Handle("POST /v1/slots/{slotID}/bill-split", s.requireAuth(http.HandlerFunc(s.splitBill)))
	mux.Handle("POST /v1/slots/{slotID}/guardian-links", s.requireAuth(http.HandlerFunc(s.createGuardianLink)))
	mux.Handle("DELETE /v1/slots/{slotID}/guardian-links/{linkID}", s.requireAuth(http.HandlerFunc(s.revokeGuardianLink)))
	mux.Handle("GET /v1/guardian-links/{token}", s.guardianAccessRateLimit(http.HandlerFunc(s.accessGuardianLink)))

	s.handler = s.requestMeta(mux)
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) livez(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if s.deps.Ready != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.deps.Ready(ctx); err != nil {
			writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "service dependencies are not ready")
			return
		}
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.Accounts == nil {
			writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "account service is unavailable")
			return
		}
		raw, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		u, sid, err := s.deps.Accounts.Authenticate(r.Context(), raw)
		if err != nil {
			if errors.Is(err, account.ErrUnauthorized) || errors.Is(err, account.ErrNotFound) {
				writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
			} else {
				writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "authentication service is temporarily unavailable")
			}
			return
		}
		if s.deps.UserLimiter != nil {
			allowed, retry := s.deps.UserLimiter.Allow(u.ID)
			if !allowed {
				writeRetryAfter(w, retry)
				writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "too many authenticated requests")
				return
			}
		}
		ctx := context.WithValue(r.Context(), authKey, authContext{User: u, SessionID: sid, RawToken: raw})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) authRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.AuthLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}
		key := remoteIP(r.RemoteAddr) + ":" + r.URL.Path
		allowed, retry := s.deps.AuthLimiter.Allow(key)
		if !allowed {
			writeRetryAfter(w, retry)
			writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "too many authentication attempts")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// monetizationRateLimit applies docs/LINKUP_PLUS_MONETIZATION.md §8.2's
// entitlement-sensitive abuse budget, independent of the general UserLimiter
// applied to every authenticated request. Must run after requireAuth (it
// keys on the authenticated user, not the remote address, since a shared
// NAT/family network legitimately produces many distinct users from one IP
// — §8.3's own explicit caveat against treating that as fraud).
func (s *Server) monetizationRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.MonetizationLimiter != nil {
			auth, ok := authFrom(r)
			if !ok {
				writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			allowed, retry := s.deps.MonetizationLimiter.Allow(auth.User.ID)
			if !allowed {
				writeRetryAfter(w, retry)
				writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "too many monetization requests")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// guardianAccessRateLimit throttles unauthenticated guardian-link token
// lookups by remote IP, guarding against token brute-forcing. It reuses
// AuthLimiter rather than adding a new dependency/config tier — this
// endpoint is the same shape of risk (unauthenticated, secret-bearing
// lookup) as login/recovery. It deliberately keys on a fixed logical
// suffix, not r.URL.Path: unlike authRateLimit's fixed-path routes, this
// route's path contains the very token being guessed, so keying on the
// literal path would give every guessed token its own fresh rate-limit
// bucket and defeat the limiter entirely.
func (s *Server) guardianAccessRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.AuthLimiter != nil {
			key := remoteIP(r.RemoteAddr) + ":guardian-access"
			allowed, retry := s.deps.AuthLimiter.Allow(key)
			if !allowed {
				writeRetryAfter(w, retry)
				writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "too many guardian link attempts")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeRetryAfter(w http.ResponseWriter, retry time.Duration) {
	seconds := int((retry + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}

func remoteIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func bearerToken(v string) (string, bool) {
	parts := strings.Fields(v)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
		return parts[1], true
	}
	return "", false
}

func authFrom(r *http.Request) (authContext, bool) {
	v, ok := r.Context().Value(authKey).(authContext)
	return v, ok
}

func (s *Server) requestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier.NewUUID()
		if err != nil {
			id = "unavailable"
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))
		slog.Info("http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("expected one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	id, _ := r.Context().Value(requestIDKey).(string)
	writeJSON(w, status, problem{Code: code, Message: message, RequestID: id})
}
