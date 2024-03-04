package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type SimpleHandler struct {
	Writer io.Writer
	Level  slog.Leveler

	mu sync.Mutex
}

func (h *SimpleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level.Level()
}

func (h *SimpleHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.Writer.Write([]byte(fmt.Sprintf("%s: %s\n", record.Level.String(), record.Message)))
	return err
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// no support for attrs here
	return h
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	// no support for attrs here
	return h
}

var _ slog.Handler = (*SimpleHandler)(nil)
