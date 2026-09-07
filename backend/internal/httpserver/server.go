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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type Dependencies struct {
	Accounts    *account.Service
	Blocks      *blocklist.Service
	Slots       *slot.Service
	Chats       *chat.Service
	Ready       func(context.Context) error
	AuthLimiter *ratelimit.Limiter
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
	mux.Handle("POST /v1/auth/register", s.authRateLimit(http.HandlerFunc(s.register)))
	mux.Handle("POST /v1/auth/login", s.authRateLimit(http.HandlerFunc(s.login)))
	mux.Handle("POST /v1/auth/recovery/request", s.authRateLimit(http.HandlerFunc(s.requestPasswordRecovery)))
	mux.Handle("POST /v1/auth/recovery/reset", s.authRateLimit(http.HandlerFunc(s.resetPassword)))
	mux.Handle("POST /v1/auth/logout", s.requireAuth(http.HandlerFunc(s.logout)))
	mux.Handle("GET /v1/me", s.requireAuth(http.HandlerFunc(s.getMe)))
	mux.Handle("PATCH /v1/me", s.requireAuth(http.HandlerFunc(s.patchMe)))
	mux.Handle("GET /v1/me/blocks", s.requireAuth(http.HandlerFunc(s.listBlocks)))
	mux.Handle("PUT /v1/me/blocks/{userID}", s.requireAuth(http.HandlerFunc(s.blockUser)))
	mux.Handle("DELETE /v1/me/blocks/{userID}", s.requireAuth(http.HandlerFunc(s.unblockUser)))

	mux.Handle("POST /v1/slots", s.requireAuth(http.HandlerFunc(s.createSlot)))
	mux.Handle("GET /v1/slots/{slotID}", s.requireAuth(http.HandlerFunc(s.getSlot)))
	mux.Handle("PATCH /v1/slots/{slotID}", s.requireAuth(http.HandlerFunc(s.editSlot)))
	mux.Handle("POST /v1/slots/{slotID}/cancel", s.requireAuth(http.HandlerFunc(s.cancelSlot)))
	mux.Handle("GET /v1/slots/{slotID}/accepted", s.requireAuth(http.HandlerFunc(s.listAccepted)))
	mux.Handle("GET /v1/me/slots", s.requireAuth(http.HandlerFunc(s.listMySlots)))
	mux.Handle("GET /v1/pulse", s.requireAuth(http.HandlerFunc(s.listPulse)))
	mux.Handle("POST /v1/slots/{slotID}/request", s.requireAuth(http.HandlerFunc(s.requestSlot)))
	mux.Handle("POST /v1/slots/{slotID}/leave", s.requireAuth(http.HandlerFunc(s.leaveSlot)))
	mux.Handle("GET /v1/slots/{slotID}/requests", s.requireAuth(http.HandlerFunc(s.listPendingRequests)))
	mux.Handle("POST /v1/slots/{slotID}/requests/{userID}/approve", s.requireAuth(http.HandlerFunc(s.approveRequest)))
	mux.Handle("POST /v1/slots/{slotID}/requests/{userID}/reject", s.requireAuth(http.HandlerFunc(s.rejectRequest)))
	mux.Handle("POST /v1/slots/{slotID}/start", s.requireAuth(http.HandlerFunc(s.startSlot)))
	mux.Handle("POST /v1/slots/{slotID}/complete", s.requireAuth(http.HandlerFunc(s.completeSlot)))
	mux.Handle("GET /v1/slots/{slotID}/chat/messages", s.requireAuth(http.HandlerFunc(s.listChatMessages)))
	mux.Handle("POST /v1/slots/{slotID}/chat/messages", s.requireAuth(http.HandlerFunc(s.sendChatMessage)))

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
			seconds := int((retry + time.Second - 1) / time.Second)
			if seconds < 1 { seconds = 1 }
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "too many authentication attempts")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func remoteIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil && host != "" { return host }
	return strings.TrimSpace(remoteAddr)
}

func bearerToken(v string) (string, bool) {
	parts := strings.Fields(v)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" { return parts[1], true }
	return "", false
}

func authFrom(r *http.Request) (authContext, bool) {
	v, ok := r.Context().Value(authKey).(authContext)
	return v, ok
}

func (s *Server) requestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := identifier.NewUUID(); if err != nil { id = "unavailable" }
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))
		slog.Info("http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct { http.ResponseWriter; status int }
func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Consume the entire bounded body: a valid prefix must not hide another
	// command, malformed trailing bytes, or a payload beyond MaxBytesReader.
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil { return err }
		return errors.New("expected one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil { _ = json.NewEncoder(w).Encode(body) }
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	id, _ := r.Context().Value(requestIDKey).(string)
	writeJSON(w, status, problem{Code: code, Message: message, RequestID: id})
}
