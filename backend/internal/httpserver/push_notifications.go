package httpserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
)

func (s *Server) notifyUser(userID string, message push.Message) {
	if s.deps.Push == nil || userID == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.deps.Push.NotifyUser(ctx, userID, message); err != nil {
			slog.Warn("push delivery failed", "user_id", userID, "type", message.Data["type"], "error", err)
		}
	}()
}
