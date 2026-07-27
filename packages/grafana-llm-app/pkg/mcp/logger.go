package mcp

import (
	"context"
	"log/slog"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// NewSlogLogger returns the *slog.Logger expected by mcp-go, backed by Grafana's
// backend logger.
func NewSlogLogger() *slog.Logger {
	return slog.New(&slogHandler{})
}

// slogHandler is a minimal slog.Handler that forwards records to the Grafana
// backend logger. It bridges MCP logging to Grafana's logging system.
type slogHandler struct {
	// attrs are the attributes accumulated via WithAttrs, already prefixed with
	// any groups that were open when they were added.
	attrs []slog.Attr
	// groups are the currently open group names, joined as a prefix for the keys
	// of any subsequent attributes.
	groups []string
}

// Enabled reports whether the handler handles records at the given level. All
// levels are forwarded; Grafana's logger applies its own level filtering.
func (h *slogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	args := make([]any, 0, 2*(len(h.attrs)+r.NumAttrs()))
	for _, a := range h.attrs {
		args = append(args, a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, h.key(a.Key), a.Value.Any())
		return true
	})

	switch {
	case r.Level >= slog.LevelError:
		log.DefaultLogger.Error(r.Message, args...)
	case r.Level >= slog.LevelWarn:
		log.DefaultLogger.Warn(r.Message, args...)
	case r.Level >= slog.LevelInfo:
		log.DefaultLogger.Info(r.Message, args...)
	default:
		log.DefaultLogger.Debug(r.Message, args...)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := h.clone()
	for _, a := range attrs {
		next.attrs = append(next.attrs, slog.Attr{Key: h.key(a.Key), Value: a.Value})
	}
	return next
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

// key qualifies an attribute key with the currently open groups, e.g. "a.b.key".
func (h *slogHandler) key(key string) string {
	for i := len(h.groups) - 1; i >= 0; i-- {
		key = h.groups[i] + "." + key
	}
	return key
}

func (h *slogHandler) clone() *slogHandler {
	return &slogHandler{
		attrs:  append(h.attrs[:len(h.attrs):len(h.attrs)], nil...),
		groups: append(h.groups[:len(h.groups):len(h.groups)], nil...),
	}
}
