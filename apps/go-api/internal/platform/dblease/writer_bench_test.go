package dblease

import (
	"testing"
	"time"
)

// BenchmarkAcquireRelease_Uncontended mesure l'overhead pur d'un cycle
// acquire/release sans contention. Cible : < 1 µs.
func BenchmarkAcquireRelease_Uncontended(b *testing.B) {
	path := "bench://uncontended/" + time.Now().Format("150405.000000000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, err := AcquireWriter(nil, path, KindPlayer, time.Second)
		if err != nil {
			b.Fatal(err)
		}
		w.Release()
	}
}

// BenchmarkAcquireRelease_Contended_2 mesure le coût en concurrence à 2
// goroutines. Cible : < 100 µs en moyenne.
func BenchmarkAcquireRelease_Contended_2(b *testing.B) {
	path := "bench://contended-2/" + time.Now().Format("150405.000000000")
	b.SetParallelism(1) // GOMAXPROCS * 1 = nombre de CPU ; suffit pour ~2 goroutines simultanées sur la plupart des dev machines
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w, err := AcquireWriter(nil, path, KindPlayer, 10*time.Second)
			if err != nil {
				b.Fatal(err)
			}
			w.Release()
		}
	})
}

// BenchmarkAcquireRelease_Contended_10 mesure le coût sous forte contention.
// Cible : pas de starvation visible (P99 < 10× P50, à vérifier manuellement).
func BenchmarkAcquireRelease_Contended_10(b *testing.B) {
	path := "bench://contended-10/" + time.Now().Format("150405.000000000")
	b.SetParallelism(10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w, err := AcquireWriter(nil, path, KindPlayer, 30*time.Second)
			if err != nil {
				b.Fatal(err)
			}
			w.Release()
		}
	})
}
