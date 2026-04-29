package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

func TestContextHandler_AttachesRequestID(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewContextHandler(inner))

	ctx := ctxkeys.WithRequestID(context.Background(), "req-abc-123")
	logger.InfoContext(ctx, "test event", "match_id", "m-001")

	var rec map[string]any
	if err := json.NewDecoder(&buf).Decode(&rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec["request_id"] != "req-abc-123" {
		t.Errorf("request_id = %v, want req-abc-123", rec["request_id"])
	}
	if rec["match_id"] != "m-001" {
		t.Errorf("match_id = %v, want m-001", rec["match_id"])
	}
	if rec["msg"] != "test event" {
		t.Errorf("msg = %v, want 'test event'", rec["msg"])
	}
}

func TestContextHandler_NoRequestID_NoAttribute(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewContextHandler(inner))

	logger.InfoContext(context.Background(), "background event")

	if strings.Contains(buf.String(), "request_id") {
		t.Errorf("background event should NOT contain request_id, got: %s", buf.String())
	}
}

func TestContextHandler_PropagatesEnabled(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewContextHandler(inner)

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug devrait être disabled (level=warn)")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("error devrait être enabled (level=warn)")
	}
}

func TestContextHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewContextHandler(inner)
	withAttrs := h.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})
	logger := slog.New(withAttrs)

	ctx := ctxkeys.WithRequestID(context.Background(), "req-xyz")
	logger.InfoContext(ctx, "with attrs")

	var rec map[string]any
	if err := json.NewDecoder(&buf).Decode(&rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec["service"] != "test-svc" {
		t.Errorf("service = %v, want test-svc", rec["service"])
	}
	if rec["request_id"] != "req-xyz" {
		t.Errorf("request_id = %v, want req-xyz", rec["request_id"])
	}
}
