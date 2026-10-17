package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

const recomputerTestXUID = "2533274858283686"

// spySharedProvider compte les prises du writer partagé et les lecteurs en vol.
type spySharedProvider struct {
	sharedprovider.Provider
	writers atomic.Int32
	readers atomic.Int32
}

func (s *spySharedProvider) AcquireWriter(ctx context.Context) (*sharedprovider.WriterHandle, error) {
	s.writers.Add(1)
	return s.Provider.AcquireWriter(ctx)
}

func (s *spySharedProvider) Get(ctx context.Context) (*sql.DB, func(), error) {
	s.readers.Add(1)
	return s.Provider.Get(ctx)
}

// seedRecomputerShared crée une base partagée (schéma complet) avec n matchs sociaux du
// joueur de test, puis la referme : le fournisseur la rouvre en lecture seule.
func seedRecomputerShared(t *testing.T, path string, n int) {
	t.Helper()
	handle, err := OpenSharedDB(path)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	db := handle.SQLDb()
	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := range n {
		mid := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, start_time_utc, playlist_name, pair_name, is_ranked, is_firefight, duration_seconds)
			 VALUES (?, ?, ?, 'Quick Play', 'Slayer', FALSE, FALSE, 600)`,
			mid, base.Add(time.Duration(i)*time.Hour), base.Add(time.Duration(i)*time.Hour),
		); err != nil {
			t.Fatalf("insert match_registry: %v", err)
		}
		for j := range 8 {
			xuid, outcome, kills, deaths := recomputerTestXUID, 2, 15, 5
			if j > 0 {
				xuid, outcome, kills, deaths = fmt.Sprintf("opponent_%d", j), 3, 7+j, 12+j
			}
			if _, err := db.Exec(`
				INSERT INTO match_participants
					(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
					 damage_dealt, damage_taken, accuracy, team_id)
				VALUES (?, ?, ?, ?, ?, 3, ?, ?, ?, ?, 0.55, ?)`,
				mid, xuid, outcome, kills, deaths, float64(kills)*0.9, float64(deaths)*0.9,
				float64(kills)*200.0, float64(deaths)*200.0, j%2,
			); err != nil {
				t.Fatalf("insert match_participants: %v", err)
			}
		}
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close shared: %v", err)
	}
}

// seedLUSRShared crée la base partagée de la fixture par les migrations réelles (état v2 compris)
// puis y insère n matchs sociaux 4v4 du joueur de test, tous éligibles au LUSR v2.
func seedLUSRShared(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open shared %s: %v", path, err)
	}
	defer func() { _ = db.Close() }()
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(shared): %v", err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := range n {
		mid := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		start := base.Add(time.Duration(i) * time.Hour)
		exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, playlist_name, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES (?, ?, ?, 'Quick Play', 'pair:Slayer', FALSE, FALSE, 600)`, mid, start, start)
		for j := range 8 {
			xuid, outcome, kills, deaths := recomputerTestXUID, 2, 15+i%4, 5
			if j > 0 {
				xuid, outcome, kills, deaths = fmt.Sprintf("opponent_%d", j), 3, 7+j, 12+j
			}
			exec(`INSERT INTO match_participants
				(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
				 damage_dealt, damage_taken, accuracy, team_id)
				VALUES (?, ?, ?, ?, ?, 3, ?, ?, ?, ?, 0.55, ?)`,
				mid, xuid, outcome, kills, deaths, float64(kills)*0.9, float64(deaths)*0.9,
				float64(kills)*200.0, float64(deaths)*200.0, j%2)
		}
	}
}

// seedLUSRPlayer crée la base joueur de la fixture par le schéma de sync puis les migrations réelles
// (lusr_component_history et vues _latest comprises). Sans effet sur une base déjà créée.
func seedLUSRPlayer(t *testing.T, playerPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	db, err := sql.Open("duckdb", playerPath)
	if err != nil {
		t.Fatalf("open player %s: %v", playerPath, err)
	}
	defer func() { _ = db.Close() }()
	if err := EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
}

// burstSpyProvider observe l'accès du recalcul à la base partagée : nombre de prises du
// writer, et état du fournisseur au moment de chaque lecture (un recalcul qui tiendrait
// l'écrivain en continu lirait sous RW).
type burstSpyProvider struct {
	sharedprovider.Provider
	writers         atomic.Int32
	readers         atomic.Int32
	readsUnderWrite atomic.Int32
}

func (s *burstSpyProvider) AcquireWriter(ctx context.Context) (*sharedprovider.WriterHandle, error) {
	s.writers.Add(1)
	return s.Provider.AcquireWriter(ctx)
}

func (s *burstSpyProvider) Get(ctx context.Context) (*sql.DB, func(), error) {
	s.readers.Add(1)
	if s.Provider.State() != sharedprovider.StateRO {
		s.readsUnderWrite.Add(1)
	}
	return s.Provider.Get(ctx)
}

// Contrat d'accès du recalcul d'une exclusion (D-6, plan backlog 2026-10-16) : le rejeu LUSR v2
// écrit l'état du joueur dans la base partagée, donc le recalcul PREND l'écrivain, mais par rafales
// courtes — une pour la sentinelle de reset, une pour les 12 matchs de la fixture (au plus 50
// matchs ou 2 s par rafale) — et JAMAIS en continu : la lecture des notes de performance se fait
// base en lecture seule, et la base y revient dès la dernière rafale. Remplace le contrat
// précédent (« n'acquiert jamais l'écrivain »), qui valait tant que le recalcul rejouait la v1.
func TestMatchRecomputer_TakesTheSharedWriterInShortBurstsOnly(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, 12)
	seedLUSRPlayer(t, playerPath)

	base, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	spy := &burstSpyProvider{Provider: base}

	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", spy)
	if err := rec.RecomputeAfterExclusion(context.Background(), "00000000-0000-0000-0000-000000000001"); err != nil {
		t.Fatalf("RecomputeAfterExclusion: %v", err)
	}

	if got := spy.writers.Load(); got != 2 {
		t.Fatalf("écrivain partagé pris %d fois, attendu 2 (reset du filigrane + une rafale de rejeu)", got)
	}
	if got := spy.readers.Load(); got < 1 {
		t.Fatalf("lecteur du fournisseur jamais utilisé (%d)", got)
	}
	if got := spy.readsUnderWrite.Load(); got != 0 {
		t.Fatalf("%d lecture(s) ouverte(s) pendant que l'écrivain était tenu : le recalcul le tient en continu", got)
	}
	if base.State() != sharedprovider.StateRO {
		t.Fatalf("la base partagée n'est pas revenue en lecture seule: %v", base.State())
	}

	// Les résultats sont bien écrits : notes de performance et notes LUSR en base joueur.
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	var scored, rated int
	if err := player.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`).Scan(&scored); err != nil {
		t.Fatalf("lecture des notes: %v", err)
	}
	if scored == 0 {
		t.Fatal("aucune note de performance écrite en base joueur : le recalcul n'a rien produit")
	}
	if err := player.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type = 'LUSR'`).Scan(&rated); err != nil {
		t.Fatalf("lecture des notes LUSR: %v", err)
	}
	if rated != 12 {
		t.Fatalf("%d match(s) notés LUSR, attendu 12", rated)
	}
}

// Sans fournisseur (mode historique), l'écrivain partagé se prend par bail + ouverture directe,
// pour les mêmes rafales courtes, et le bail d'écriture est libre dès le recalcul terminé.
func TestMatchRecomputer_NoProviderReleasesTheSharedWriterLease(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, 12)
	seedLUSRPlayer(t, playerPath)

	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", nil)
	if err := rec.RecomputeAfterExclusion(context.Background(), "00000000-0000-0000-0000-000000000001"); err != nil {
		t.Fatalf("RecomputeAfterExclusion: %v", err)
	}

	lease, err := dblease.AcquireWriterCtx(context.Background(), nil, sharedPath, dblease.KindSharedMatches)
	if err != nil {
		t.Fatalf("le bail d'écriture partagé n'est pas libre après le recalcul: %v", err)
	}
	lease.Release()
}
