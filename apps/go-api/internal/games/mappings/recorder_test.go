package mappings

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func newRecorderWithBuffer(t *testing.T) (*LookupRecorder, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return NewLookupRecorder(logger), buf
}

func TestLookupRecorder_FirstSeenLogsWarn(t *testing.T) {
	t.Parallel()
	rec, buf := newRecorderWithBuffer(t)
	rec.Record("halo_infinite", "kills", "fr")

	if !strings.Contains(buf.String(), "field_lookup_missing") {
		t.Errorf("first record devrait émettre Warn field_lookup_missing")
	}
	stored, dropped := rec.Stats()
	if stored != 1 || dropped != 0 {
		t.Errorf("stored=%d dropped=%d, want 1/0", stored, dropped)
	}
}

func TestLookupRecorder_SecondSeenSuppressed(t *testing.T) {
	t.Parallel()
	rec, buf := newRecorderWithBuffer(t)
	rec.Record("halo_infinite", "kills", "fr")
	logsAfterFirst := buf.String()

	// 100 records identiques → 0 nouveau log, 100 dropped
	for i := 0; i < 100; i++ {
		rec.Record("halo_infinite", "kills", "fr")
	}
	if buf.String() != logsAfterFirst {
		t.Errorf("records répétés ne devraient pas émettre de nouveau log")
	}
	stored, dropped := rec.Stats()
	if stored != 1 {
		t.Errorf("stored = %d, want 1", stored)
	}
	if dropped != 100 {
		t.Errorf("dropped = %d, want 100", dropped)
	}
}

func TestLookupRecorder_DistinctKeysAllLogged(t *testing.T) {
	t.Parallel()
	rec, buf := newRecorderWithBuffer(t)
	rec.Record("halo_infinite", "kills", "fr")
	rec.Record("halo_infinite", "deaths", "fr")
	rec.Record("halo_infinite", "kills", "en") // même key, locale différente
	rec.Record("halo_2", "kills", "fr")        // autre titre

	count := strings.Count(buf.String(), "field_lookup_missing")
	if count != 4 {
		t.Errorf("4 couples uniques → %d logs, want 4", count)
	}
	stored, _ := rec.Stats()
	if stored != 4 {
		t.Errorf("stored = %d, want 4", stored)
	}
}

func TestLookupRecorder_BoundedDoesNotExplode(t *testing.T) {
	t.Parallel()
	rec, _ := newRecorderWithBuffer(t)
	rec.WithBound(3)

	// 5 keys distinctes, borne à 3 → seulement 3 stockées, 2 droppées
	for i := 0; i < 5; i++ {
		rec.Record("halo_infinite", string(rune('a'+i)), "fr")
	}
	stored, dropped := rec.Stats()
	if stored != 3 {
		t.Errorf("borné stored = %d, want 3", stored)
	}
	if dropped != 2 {
		t.Errorf("borné dropped = %d, want 2", dropped)
	}
}

func TestLookupRecorder_FlushDropped_Resets(t *testing.T) {
	t.Parallel()
	rec, buf := newRecorderWithBuffer(t)
	rec.Record("halo_infinite", "kills", "fr")
	for i := 0; i < 10; i++ {
		rec.Record("halo_infinite", "kills", "fr")
	}
	if d := rec.FlushDropped(); d != 10 {
		t.Errorf("FlushDropped = %d, want 10", d)
	}
	if !strings.Contains(buf.String(), "mappings_lookup_throttled") {
		t.Errorf("FlushDropped > 0 devrait émettre mappings_lookup_throttled")
	}
	// Second flush → 0 drops (reset)
	if d := rec.FlushDropped(); d != 0 {
		t.Errorf("FlushDropped après reset = %d, want 0", d)
	}
}

func TestLookupRecorder_FlushDropped_NoLogIfZero(t *testing.T) {
	t.Parallel()
	rec, buf := newRecorderWithBuffer(t)
	rec.Record("halo_infinite", "kills", "fr")
	// Pas de répétition → dropped = 0
	if d := rec.FlushDropped(); d != 0 {
		t.Errorf("FlushDropped sans drops = %d, want 0", d)
	}
	if strings.Contains(buf.String(), "mappings_lookup_throttled") {
		t.Errorf("pas de log throttled si dropped=0")
	}
}

func TestLookupRecorder_NilLogger_FallbacksToDefault(t *testing.T) {
	t.Parallel()
	rec := NewLookupRecorder(nil)
	if rec.logger == nil {
		t.Errorf("logger devrait être slog.Default si nil passé")
	}
	// Sanity check : Record ne panique pas
	rec.Record("x", "y", "z")
}

func TestLookupRecorder_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	rec, _ := newRecorderWithBuffer(t)
	rec.WithBound(100)
	rec.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rec.Record("halo_infinite", "key_"+string(rune('a'+(i%5))), "fr")
			}
		}(i)
	}
	wg.Wait()

	stored, dropped := rec.Stats()
	if stored != 5 {
		t.Errorf("stored sur 5 keys distinctes = %d, want 5", stored)
	}
	if stored+dropped != 5000 {
		t.Errorf("stored(%d) + dropped(%d) = %d, want 5000 (50 goroutines × 100 records)", stored, dropped, stored+dropped)
	}
}
