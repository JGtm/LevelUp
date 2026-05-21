//go:build integration

package sharedprovider_test

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// setupSharedDBForBench équivalent de setupSharedDB pour les benchmarks
// (b.TempDir() au lieu de t.TempDir()).
func setupSharedDBForBench(b *testing.B) string {
	b.Helper()
	path := filepath.Join(b.TempDir(), "shared.duckdb")
	bootstrap, err := duckdb.OpenReadWrite(path)
	if err != nil {
		b.Fatalf("bootstrap OpenReadWrite: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(b.Context(), bootstrap.SQLDb()); err != nil {
		_ = bootstrap.Close()
		b.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := bootstrap.Close(); err != nil {
		b.Fatalf("bootstrap Close: %v", err)
	}
	return path
}

// BenchmarkProviderGet mesure l'overhead pur de Get + release sur le hot
// path en steady state RO, sans contention writer ni concurrence.
//
// Cible : < 1µs/op p99 sans contention (cf. critère go/no-go §5 du plan).
// Tout ce qui dépasse signale qu'on a introduit une indirection coûteuse
// (allocation par appel, mutex lourd, etc.) qui pénaliserait /health et
// les hot endpoints.
func BenchmarkProviderGet(b *testing.B) {
	path := setupSharedDBForBench(b)

	p, err := sharedprovider.New(path)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, release, err := p.Get(ctx)
		if err != nil {
			b.Fatalf("Get: %v", err)
		}
		release()
	}
}

// BenchmarkProviderGetParallel mesure le coût sous concurrence (10 readers).
// Le mutex p.mu pris dans Get devient le bottleneck — à comparer avec le
// solo pour quantifier l'impact.
func BenchmarkProviderGetParallel(b *testing.B) {
	path := setupSharedDBForBench(b)

	p, err := sharedprovider.New(path)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, release, err := p.Get(ctx)
			if err != nil {
				b.Fatalf("Get: %v", err)
			}
			release()
		}
	})
}
