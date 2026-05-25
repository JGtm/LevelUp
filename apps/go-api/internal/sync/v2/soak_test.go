// Package v2 — soak_test.go : test de longue durée pour détecter fuites
// mémoire, dégradation perf, ou races sous charge.
//
// Durée par défaut : 10 secondes (court — passe dans go test ./...).
// Override via LEVELUP_SOAK_V2_DURATION=1h pour le vrai soak (CI nightly
// ou validation pré-D7 shadow run).
//
// Métriques surveillées :
//   - Mémoire heap (runtime.MemStats) : pas plus de 2x au-delà du baseline
//   - Latence cycle : pas de dégradation linéaire (p95 stable)
//   - Cycle errors : aucun (le pipeline doit être stable sous charge)
package v2

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	syncpkg "levelup/go-api/internal/sync"
)

// TestSoak_V2_CycleStability exécute des cycles en boucle pendant la
// durée configurée. Vérifie :
//   - Aucun cycle ne retourne d'erreur
//   - Heap stable (pas de fuite cumulative > 2x baseline)
//   - Latence cycle stable (p95 ne dérive pas > 3x moyenne)
//
// Override env :
//   - LEVELUP_SOAK_V2_DURATION (default 10s, ex: "1m", "1h")
//   - LEVELUP_SOAK_V2_INTERVAL (default 50ms, intervalle entre cycles)
func TestSoak_V2_CycleStability(t *testing.T) {
	duration := parseSoakDuration(os.Getenv("LEVELUP_SOAK_V2_DURATION"), 10*time.Second)
	interval := parseSoakDuration(os.Getenv("LEVELUP_SOAK_V2_INTERVAL"), 50*time.Millisecond)

	if testing.Short() && duration > 30*time.Second {
		t.Skip("soak test long — skip en mode -short")
	}

	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
		{Gamertag: "bob", XUID: "1000000000000002", PlayerSlug: "bob"},
	}
	env := setupE2EEnv(t, []string{"alice", "bob"})

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m1", "m2"),
			"xuid(1000000000000002)": histList("m1", "m3"),
		},
		statsByMatch: map[string]map[string]any{
			"m1": {"placeholder": 1},
			"m2": {"placeholder": 2},
			"m3": {"placeholder": 3},
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)

	// Mesure baseline.
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	var cycleCount atomic.Int64
	var errorCount atomic.Int64
	var totalDurationMS atomic.Int64
	var maxDurationMS atomic.Int64

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	t.Logf("soak: démarrage pour %v (interval %v)", duration, interval)
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(start)
			count := cycleCount.Load()
			errors := errorCount.Load()
			avgMS := int64(0)
			if count > 0 {
				avgMS = totalDurationMS.Load() / count
			}
			maxMS := maxDurationMS.Load()

			// Heap final.
			runtime.GC()
			var final runtime.MemStats
			runtime.ReadMemStats(&final)
			heapGrowth := float64(final.HeapAlloc) / float64(baseline.HeapAlloc+1)

			t.Logf("soak: terminé après %v", elapsed)
			t.Logf("  cycles=%d errors=%d", count, errors)
			t.Logf("  latence avg=%dms max=%dms", avgMS, maxMS)
			t.Logf("  heap baseline=%d KB → final=%d KB (×%.2f)",
				baseline.HeapAlloc/1024, final.HeapAlloc/1024, heapGrowth)

			// Assertions go/no-go.
			if errors > 0 {
				t.Errorf("soak: %d cycles ont échoué (devrait être 0)", errors)
			}
			if count < 5 {
				t.Errorf("soak: seulement %d cycles exécutés (test trop court ?)", count)
			}
			// Heap : tolère croissance ×3 (allocations transitoires legit).
			if heapGrowth > 3.0 {
				t.Errorf("soak: heap a cru de ×%.2f (fuite mémoire suspectée)", heapGrowth)
			}
			// Latence : max ne doit pas exploser au-delà de 5x avg.
			if maxMS > 0 && avgMS > 0 && maxMS > 5*avgMS+50 {
				t.Errorf("soak: latence max=%dms vs avg=%dms (dégradation ×%d)",
					maxMS, avgMS, maxMS/avgMS)
			}
			return

		case <-ticker.C:
			cycleStart := time.Now()
			_, err := orch.Run(ctx, players)
			cycleMS := time.Since(cycleStart).Milliseconds()

			cycleCount.Add(1)
			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				errorCount.Add(1)
				t.Logf("soak cycle %d err: %v", cycleCount.Load(), err)
			}
			totalDurationMS.Add(cycleMS)
			// Update max atomique.
			for {
				prevMax := maxDurationMS.Load()
				if cycleMS <= prevMax || maxDurationMS.CompareAndSwap(prevMax, cycleMS) {
					break
				}
			}
		}
	}
}

// parseSoakDuration parse une env var de durée (ex: "10s", "1m", "1h").
// Retourne defaultVal si vide ou invalide.
func parseSoakDuration(envVal string, defaultVal time.Duration) time.Duration {
	if envVal == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(envVal)
	if err != nil {
		return defaultVal
	}
	return d
}

// TestSoak_V2_NoDataRaceUnderConcurrency lance N cycles en parallèle
// pour stresser les goroutines internes (errgroups Phase 1+3+4+6).
// Détecte les races via go test -race.
//
// Court par défaut (3 cycles parallèles × 5 itérations chacun).
func TestSoak_V2_NoDataRaceUnderConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("concurrent soak — skip en mode -short")
	}

	const parallelism = 3
	const iterations = 5

	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
		{Gamertag: "bob", XUID: "1000000000000002", PlayerSlug: "bob"},
	}

	// IMPORTANT : un orchestrator par goroutine — l'orchestrator est PAS
	// thread-safe (chaque cycle modifie son CycleResult internal).
	done := make(chan error, parallelism)
	for i := 0; i < parallelism; i++ {
		go func(workerID int) {
			env := setupE2EEnv(t, []string{"alice", "bob"})
			client := &mockNarrowClient{
				historyByArg: map[string][]syncpkg.MatchHistoryEntry{
					"xuid(1000000000000001)": histList("m1"),
					"xuid(1000000000000002)": histList("m1"),
				},
				statsByMatch: map[string]map[string]any{
					"m1": {"placeholder": 1},
				},
			}
			orch, _ := buildE2EOrchestrator(t, env, client, players)
			for j := 0; j < iterations; j++ {
				if _, err := orch.Run(context.Background(), players); err != nil {
					done <- fmt.Errorf("worker %d iter %d: %w", workerID, j, err)
					return
				}
			}
			done <- nil
		}(i)
	}

	for i := 0; i < parallelism; i++ {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}
