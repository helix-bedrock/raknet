package internal

import (
	"context"
	"log/slog"
)

// DiscardHandler implements a slog.Handler that is always disabled. Each of its
// methods return immediately without any code running.
type DiscardHandler struct{}

// Enabled always reports false.
func (d DiscardHandler) Enabled(context.Context, slog.Level) bool { return false }

// Handle discards the record.
func (d DiscardHandler) Handle(context.Context, slog.Record) error { return nil }

// WithAttrs returns the handler unchanged.
func (d DiscardHandler) WithAttrs([]slog.Attr) slog.Handler { return d }

// WithGroup returns the handler unchanged.
func (d DiscardHandler) WithGroup(string) slog.Handler { return d }
