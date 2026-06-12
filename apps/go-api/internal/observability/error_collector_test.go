package observability

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func mkRecord(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Now(), level, msg, 0)
	r.AddAttrs(attrs...)
	return r
}

func TestErrorCollector_AggregatesByMessage(t *testing.T) {
	c := newErrorCollector(8)

	// Même message, attributs (gamertag) variables → un seul bucket, count=3.
	c.record(mkRecord(slog.LevelError, "player_watcher: sync échoué", slog.String("gamertag", "A"), slog.String("err", "timeout A")))
	c.record(mkRecord(slog.LevelError, "player_watcher: sync échoué", slog.String("gamertag", "B"), slog.String("err", "timeout B")))
	c.record(mkRecord(slog.LevelError, "player_watcher: sync échoué", slog.String("gamertag", "C")))
	// Un autre message WARN.
	c.record(mkRecord(slog.LevelWarn, "pool: token expiré"))

	snap := c.snapshot()
	if len(snap) != 2 {
		t.Fatalf("buckets = %d (attendu 2)", len(snap))
	}
	// Trié Count desc → le sync échoué (3) avant le token expiré (1).
	if snap[0].Message != "player_watcher: sync échoué" || snap[0].Count != 3 {
		t.Fatalf("bucket[0] = %q count=%d (attendu sync échoué / 3)", snap[0].Message, snap[0].Count)
	}
	if snap[0].Module != "player_watcher" {
		t.Errorf("module = %q (attendu player_watcher)", snap[0].Module)
	}
	if snap[0].Level != "ERROR" {
		t.Errorf("level = %q (attendu ERROR)", snap[0].Level)
	}
	// Le dernier échantillon "err" non vide est conservé (le 3e record n'a pas
	// d'err → on garde "timeout B").
	if snap[0].LastDetail != "timeout B" {
		t.Errorf("last_detail = %q (attendu timeout B)", snap[0].LastDetail)
	}
}

func TestErrorCollector_EvictsOldestWhenFull(t *testing.T) {
	c := newErrorCollector(2)
	base := time.Now()
	rec := func(msg string, at time.Time) slog.Record {
		return slog.NewRecord(at, slog.LevelError, msg, 0)
	}
	c.record(rec("m1: a", base))
	c.record(rec("m2: b", base.Add(time.Second)))
	// 3e clé distincte → évince m1 (LastSeen le plus ancien).
	c.record(rec("m3: c", base.Add(2*time.Second)))

	snap := c.snapshot()
	if len(snap) != 2 {
		t.Fatalf("buckets = %d (attendu 2, cap)", len(snap))
	}
	for _, b := range snap {
		if b.Message == "m1: a" {
			t.Errorf("m1 aurait dû être évincé (LRU)")
		}
	}
}

func TestErrorCollectorHandler_OnlyWarnAndAbove(t *testing.T) {
	ResetErrorBuckets()
	t.Cleanup(ResetErrorBuckets)

	h := NewErrorCollectorHandler(slog.NewTextHandler(discardWriter{}, nil))
	ctx := context.Background()
	_ = h.Handle(ctx, mkRecord(slog.LevelInfo, "info: ignoré"))
	_ = h.Handle(ctx, mkRecord(slog.LevelDebug, "debug: ignoré"))
	_ = h.Handle(ctx, mkRecord(slog.LevelWarn, "warn: gardé"))
	_ = h.Handle(ctx, mkRecord(slog.LevelError, "error: gardé"))

	snap := ErrorBuckets()
	if len(snap) != 2 {
		t.Fatalf("buckets = %d (attendu 2 : warn+error, info/debug ignorés)", len(snap))
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
