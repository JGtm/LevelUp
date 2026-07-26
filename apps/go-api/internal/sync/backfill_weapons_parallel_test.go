//go:build integration

// Package sync — backfill_weapons_parallel_test.go : tests TDD pour la
// parallélisation de `collectWeaponKillsChunk` (plan Phase 3.0).
//
// Cf. .ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md §3.0 + handoff §3 priorité 1.
// Audit Agent 3 a mesuré que `collectWeaponKillsChunk` séquentiel coûte
// ~210-275s par cycle Madina (21 films × ~10-13s/match). Paralléliser via
// errgroup.SetLimit(8) doit diviser ce coût par 5-7 sans changer les outputs.
//
// **Mode TDD strict** (directive utilisateur) : ces tests définissent le
// contrat de sortie ATTENDU avant toute modification du code. Le test perf
// (LatencyParallelFasterThanSequential) doit ÉCHOUER sur le code séquentiel
// actuel et PASSER après l'impl errgroup.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// weaponLatencyClient simule un GetMatchFilm avec latence configurable. Permet
// de mesurer le gain perf de la parallélisation vs sequentielle.
type weaponLatencyClient struct {
	weaponTestClient                                  // hérite des autres méthodes mock
	latency          time.Duration                    // simule le RTT du download film
	filmsByMatch     map[string]bool                  // filmPresent par match_id (default false = noFilm)
	chunksByMatch    map[string]map[int]FilmChunkData // optionnel : data si film present
	callsByMatch     sync.Map                         // map[string]*atomic.Int64 — nb d'appels par match_id
}

func (w *weaponLatencyClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]FilmChunkData, bool, error) {
	// Tracking : 1 call par match_id
	v, _ := w.callsByMatch.LoadOrStore(matchID, new(atomic.Int64))
	v.(*atomic.Int64).Add(1)

	// Simule latence réseau (respecte ctx cancel)
	if w.latency > 0 {
		select {
		case <-time.After(w.latency):
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}

	present := w.filmsByMatch[matchID]
	var chunks map[int]FilmChunkData
	if present {
		chunks = w.chunksByMatch[matchID]
	}
	return chunks, present, nil
}

func newWeaponLatencyClient(latency time.Duration, presentMatches []string) *weaponLatencyClient {
	films := make(map[string]bool, len(presentMatches))
	for _, m := range presentMatches {
		films[m] = true
	}
	return &weaponLatencyClient{
		latency:      latency,
		filmsByMatch: films,
	}
}

// seedMatchesInRegistry insère N matchs vides dans match_registry pour
// satisfaire les FK / index. Retourne la liste des match_ids créés.
func seedMatchesInRegistry(t *testing.T, db *sql.DB, n int, prefix string) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s-%04d", prefix, i)
		if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES (?)`, id); err != nil {
			t.Fatalf("seed match %q: %v", id, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// TestProcessWeaponKillsInline_Concurrent_NoRace : 100 matchs (mix film
// présent / absent), latence faible, run avec -race. Verify done + noFilm
// totaux + 1 call par match (pas de doublon ni perte).
func TestProcessWeaponKillsInline_Concurrent_NoRace(t *testing.T) {
	db := openWeaponDB(t)
	matchIDs := seedMatchesInRegistry(t, db, 100, "race")

	// 60 films présent, 40 absents
	present := matchIDs[:60]
	client := newWeaponLatencyClient(1*time.Millisecond, present)

	done, noFilm, err := weaponKillsInline(context.Background(), db, client, "xuid1", matchIDs)
	if err != nil {
		t.Fatalf("weaponKillsInline: %v", err)
	}
	if done != 60 {
		t.Errorf("done = %d, want 60", done)
	}
	if noFilm != 40 {
		t.Errorf("noFilm = %d, want 40", noFilm)
	}

	// Chaque match doit avoir été appelé exactement 1 fois (pas de retry caché
	// ni dedupe inattendue).
	for _, id := range matchIDs {
		v, ok := client.callsByMatch.Load(id)
		if !ok {
			t.Errorf("match %s: never called", id)
			continue
		}
		if got := v.(*atomic.Int64).Load(); got != 1 {
			t.Errorf("match %s: called %d times, want 1", id, got)
		}
	}
}

// TestProcessWeaponKillsInline_LatencyParallelFasterThanSequential :
// 16 matchs × 100ms latence simulée. La version SÉQUENTIELLE prend ~1.6s.
// La version PARALLÈLE (weaponBackfillParallelism=24 post-phase 3.6) doit
// prendre ~100ms + overhead (1 vague, tous matchs fit dans 24 slots).
//
// **Test perf qui ÉCHOUE sur le code séquentiel actuel** et passe après
// errgroup.SetLimit(weaponBackfillParallelism).
func TestProcessWeaponKillsInline_LatencyParallelFasterThanSequential(t *testing.T) {
	db := openWeaponDB(t)
	matchIDs := seedMatchesInRegistry(t, db, 16, "perf")

	// Tous films présents (pas un test sur le tri found/noFilm, juste sur le wall-time).
	client := newWeaponLatencyClient(100*time.Millisecond, matchIDs)

	start := time.Now()
	_, _, err := weaponKillsInline(context.Background(), db, client, "xuid1", matchIDs)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("weaponKillsInline: %v", err)
	}

	// Séquentielle attendue : 16 × 100ms = 1600ms.
	// Parallèle (SetLimit=24, phase 3.6) : 1 vague × 100ms = 100ms + overhead = ~200ms.
	// On exige < 700ms pour passer (3× speedup mini), plus tolérant que les 5-15×
	// estimés théoriques pour absorber la variance CI.
	const maxAcceptable = 700 * time.Millisecond
	if elapsed > maxAcceptable {
		t.Errorf("wall-time %v > %v (parallélisation absente ou défaillante — "+
			"séquentiel théorique = 1.6s pour 16×100ms, parallèle attendu < 700ms)",
			elapsed, maxAcceptable)
	} else {
		t.Logf("wall-time = %v (gain parallélisation confirmé)", elapsed)
	}
}

// TestProcessWeaponKillsInline_Idempotent : 2 runs consécutifs sur les mêmes
// match_ids produisent le même résultat. Important car certains heal loops
// peuvent appeler `collectWeaponKillsChunk` sur des matchs déjà traités.
func TestProcessWeaponKillsInline_Idempotent(t *testing.T) {
	db := openWeaponDB(t)
	matchIDs := seedMatchesInRegistry(t, db, 10, "idem")

	client := newWeaponLatencyClient(1*time.Millisecond, matchIDs[:5])

	done1, noFilm1, err := weaponKillsInline(context.Background(), db, client, "xuid1", matchIDs)
	if err != nil {
		t.Fatalf("1er run: %v", err)
	}
	done2, noFilm2, err := weaponKillsInline(context.Background(), db, client, "xuid1", matchIDs)
	if err != nil {
		t.Fatalf("2e run: %v", err)
	}

	if done1 != done2 || noFilm1 != noFilm2 {
		t.Errorf("idempotence violée : run1=(done=%d,noFilm=%d) vs run2=(done=%d,noFilm=%d)",
			done1, noFilm1, done2, noFilm2)
	}
}

// TestProcessWeaponKillsInline_CancelMidRun : ctx.Cancel à mi-parcours doit
// retourner ctx.Err() sans crasher. Les matchs déjà traités restent committés.
//
// 100 matchs × 50ms : même avec weaponBackfillParallelism=24 (phase 3.6),
// au moins 4 vagues = 200ms minimum → cancel à 75ms catche mid-run.
func TestProcessWeaponKillsInline_CancelMidRun(t *testing.T) {
	db := openWeaponDB(t)
	matchIDs := seedMatchesInRegistry(t, db, 100, "cancel")

	client := newWeaponLatencyClient(50*time.Millisecond, matchIDs)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel après 75ms (assez pour que 1-2 vagues parallèles commencent mais
	// pas pour que tout soit fini, même avec parallelism=24).
	go func() {
		time.Sleep(75 * time.Millisecond)
		cancel()
	}()

	_, _, err := weaponKillsInline(ctx, db, client, "xuid1", matchIDs)
	if err == nil {
		t.Error("attendu une erreur ctx.Err() après cancel, got nil")
	}
}
