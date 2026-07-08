package handlers

import (
	"context"
	"log/slog"
)

func logError(_ context.Context, handler string, err error) {
	if err != nil {
		slog.Error("handler error", "handler", handler, "error", err)
	}
}
