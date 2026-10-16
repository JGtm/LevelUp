package sync

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

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

// Le recalcul d'une exclusion n'acquiert jamais le writer de la base partagée : il la lit
// par le lecteur du fournisseur (ouverture en lecture seule, toute écriture y échouerait)
// et n'écrit qu'en base joueur.
func TestMatchRecomputer_NeverAcquiresTheSharedWriter(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedRecomputerShared(t, sharedPath, 12)

	base, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	spy := &spySharedProvider{Provider: base}

	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", spy)
	if err := rec.RecomputeAfterExclusion(context.Background(), "00000000-0000-0000-0000-000000000001"); err != nil {
		t.Fatalf("RecomputeAfterExclusion: %v", err)
	}

	if got := spy.writers.Load(); got != 0 {
		t.Fatalf("writer partagé acquis %d fois, attendu 0", got)
	}
	if got := spy.readers.Load(); got < 1 {
		t.Fatalf("lecteur du fournisseur jamais utilisé (%d)", got)
	}

	// Les résultats sont bien écrits côté joueur.
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	var scored int
	if err := player.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`).Scan(&scored); err != nil {
		t.Fatalf("lecture des notes: %v", err)
	}
	if scored == 0 {
		t.Fatal("aucune note de performance écrite en base joueur : le recalcul n'a rien produit")
	}
	if base.State() != sharedprovider.StateRO {
		t.Fatalf("la base partagée n'est pas restée en lecture seule: %v", base.State())
	}
}

// Sans fournisseur (mode historique), la base partagée est lue par OpenReadForQuery : le
// recalcul réussit sans jamais l'ouvrir en écriture ni prendre son bail d'écriture.
func TestMatchRecomputer_NoProviderReadsSharedReadOnly(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedRecomputerShared(t, sharedPath, 12)

	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", nil)
	if err := rec.RecomputeAfterExclusion(context.Background(), "00000000-0000-0000-0000-000000000001"); err != nil {
		t.Fatalf("RecomputeAfterExclusion: %v", err)
	}

	// Le bail d'écriture de la base partagée est resté libre.
	lease, err := dblease.AcquireWriterCtx(context.Background(), nil, sharedPath, dblease.KindSharedMatches)
	if err != nil {
		t.Fatalf("le bail d'écriture partagé n'est pas libre après le recalcul: %v", err)
	}
	lease.Release()
}
