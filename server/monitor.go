package server

import (
	"context"
	"time"

	"gitlab.dev.ict/golang/go-ai/handlers"
)

func StartSSEMonitoring(ctx context.Context, handler *handlers.AppHandler, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handler.LogActiveSSEConnections()
			}
		}
	}()
}
