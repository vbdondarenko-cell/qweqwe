package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
)

type Dependencies struct {
	Accounts    *account.Service
	Ready       func(context.Context) error
	AuthLimiter *ratelimit.Limiter
}

type Server struct { handler http.Handler; deps Dependencies }

type healthResponse struct { Status string `json:"status"`; Time string `json:"time"` }
type problem struct { Code string `json:"code"`; Message string `json:"message"`; RequestID string `json:"requestId,omitempty"` }
type authContext struct { User account.User; SessionID string; RawToken string }
type contextKey string
const authKey contextKey="auth"
const requestIDKey contextKey="request_id"

func New(deps Dependencies) *Server {
	s:=&Server{deps:deps}
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /livez",s.livez)
	mux.HandleFunc("GET /healthz",s.healthz)
	mux.Handle("POST /v1/auth/register",s.authRateLimit(http.HandlerFunc(s.register)))
	mux.Handle("POST /v1/auth/login",s.authRateLimit(http.HandlerFunc(s.login)))
	mux.Handle("POST /v1/auth/recovery/request",s.authRateLimit(http.HandlerFunc(s.requestPasswordRecovery)))
	mux.Handle("POST /v1/auth/recovery/reset",s.authRateLimit(http.HandlerFunc(s.resetPassword)))
	mux.Handle("POST /v1/auth/logout",s.requireAuth(http.HandlerFunc(s.logout)))
	mux.Handle("GET /v1/me",s.requireAuth(http.HandlerFunc(s.getMe)))
	mux.Handle("PATCH /v1/me",s.requireAuth(http.HandlerFunc(s.patchMe)))
	s.handler=s.requestMeta(mux)
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) livez(w http.ResponseWriter,_ *http.Request) { writeJSON(w,http.StatusOK,healthResponse{Status:"ok",Time:time.Now().UTC().Format(time.RFC3339)}) }
func (s *Server) healthz(w http.ResponseWriter,r *http.Request) { if s.deps.Ready!=nil { ctx,cancel:=context.WithTimeout(r.Context(),2*time.Second); defer cancel(); if err:=s.deps.Ready(ctx); err!=nil { writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","service dependencies are not ready"); return } }; writeJSON(w,http.StatusOK,healthResponse{Status:"ok",Time:time.Now().UTC().Format(time.RFC3339)}) }

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if s.deps.Accounts==nil { writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","account service is unavailable"); return }
		raw,ok:=bearerToken(r.Header.Get("Authorization")); if !ok { writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required"); return }
		u,sid,err:=s.deps.Accounts.Authenticate(r.Context(),raw); if err!=nil { writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required"); return }
		ctx:=context.WithValue(r.Context(),authKey,authContext{User:u,SessionID:sid,RawToken:raw})
		next.ServeHTTP(w,r.WithContext(ctx))
	})
}

func (s *Server) authRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.AuthLimiter == nil { next.ServeHTTP(w, r); return }
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

func bearerToken(v string)(string,bool){ parts:=strings.Fields(v); returnValue:=""; if len(parts)==2 && strings.EqualFold(parts[0],"Bearer") { returnValue=parts[1] }; return returnValue,returnValue!="" }
func authFrom(r *http.Request)(authContext,bool){ v,ok:=r.Context().Value(authKey).(authContext); return v,ok }

func (s *Server) requestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ id,err:=identifier.NewUUID(); if err!=nil { id="unavailable" }; w.Header().Set("X-Request-ID",id); ctx:=context.WithValue(r.Context(),requestIDKey,id); start:=time.Now(); rw:=&statusWriter{ResponseWriter:w,status:http.StatusOK}; next.ServeHTTP(rw,r.WithContext(ctx)); slog.Info("http request","request_id",id,"method",r.Method,"path",r.URL.Path,"status",rw.status,"duration_ms",time.Since(start).Milliseconds()) })
}

type statusWriter struct { http.ResponseWriter; status int }
func (w *statusWriter) WriteHeader(code int){ w.status=code; w.ResponseWriter.WriteHeader(code) }

func decodeJSON(w http.ResponseWriter,r *http.Request,dst any) error { r.Body=http.MaxBytesReader(w,r.Body,64<<10); dec:=json.NewDecoder(r.Body); dec.DisallowUnknownFields(); return dec.Decode(dst) }
func writeJSON(w http.ResponseWriter,status int,body any){ w.Header().Set("Content-Type","application/json; charset=utf-8"); w.WriteHeader(status); if body!=nil { _=json.NewEncoder(w).Encode(body) } }
func writeProblem(w http.ResponseWriter,r *http.Request,status int,code,message string){ id,_:=r.Context().Value(requestIDKey).(string); writeJSON(w,status,problem{Code:code,Message:message,RequestID:id}) }
