package httpserver

import (
	"encoding/json"
	"net/http"
	"time"
)

type Server struct {
	handler http.Handler
}

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func New() *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", livez)
	mux.HandleFunc("GET /healthz", healthz)
	return &Server{handler: mux}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func livez(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
