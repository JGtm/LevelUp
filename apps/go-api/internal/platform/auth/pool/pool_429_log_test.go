package pool

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// Un 429 absorbé par la mise à l'écart d'un token est un INFO compté ; une pause plus longue que
// perToken429WarnCooldown reste un WARN.
func TestPoolOn429ForToken_NiveauSelonLaDuree(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()
	avant := perToken429Absorbed.Value()

	p.On429ForToken("A", 5*time.Second)
	out := buf.String()
	if !strings.Contains(out, "level=INFO msg=\"pool: token rate-limité (429)") {
		t.Errorf("un 429 court devrait être un INFO :\n%s", out)
	}
	if strings.Contains(out, "level=WARN") {
		t.Errorf("WARN écrit pour un 429 court absorbé :\n%s", out)
	}

	buf.Reset()
	p.On429ForToken("B", perToken429WarnCooldown+time.Second)
	if !strings.Contains(buf.String(), "level=WARN msg=\"pool: token rate-limité (429)") {
		t.Errorf("une pause longue devrait rester un WARN :\n%s", buf.String())
	}
	if got := perToken429Absorbed.Value() - avant; got != 2 {
		t.Errorf("compteur des 429 absorbés +%d, attendu +2", got)
	}
}
