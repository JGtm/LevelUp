//go:build integration

package sharedprovider_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_HealthLatency_p99_integration (T10 du plan) protège le hot
// path /health contre une régression d'overhead introduite par la couche
// provider. Cible : p99 < 50ms (SLA prod), p50 < 5ms.
//
// 1000 itérations Get + PRAGMA version + Release séquentielles, sans
// contention writer ni autre reader.
func TestProvider_HealthLatency_p99_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	const n = 1000
	durations := make([]time.Duration, n)

	// Warm-up : première query DuckDB peut être un peu plus lente (lazy
	// init pool conns). Ne pas compter dans les mesures.
	for i := 0; i < 10; i++ {
		db, release, err := p.Get(ctx)
		if err != nil {
			t.Fatalf("warmup Get: %v", err)
		}
		var v string
		_ = db.QueryRowContext(ctx, "SELECT version()").Scan(&v)
		release()
	}

	for i := 0; i < n; i++ {
		start := time.Now()
		db, release, err := p.Get(ctx)
		if err != nil {
			t.Fatalf("Get #%d: %v", i, err)
		}
		var v string
		if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
			release()
			t.Fatalf("PRAGMA #%d: %v", i, err)
		}
		release()
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50 := durations[n*50/100]
	p99 := durations[n*99/100]

	if p99 > 50*time.Millisecond {
		t.Errorf("p99 = %v, attendu < 50ms (SLA /health)", p99)
	}
	if p50 > 5*time.Millisecond {
		t.Errorf("p50 = %v, attendu < 5ms", p50)
	}
	t.Logf("perf /health : p50=%v p99=%v sur %d itérations", p50, p99, n)
}
