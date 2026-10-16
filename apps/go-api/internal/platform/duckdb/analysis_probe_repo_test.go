//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/matchflags"
)

// seedAnalysisProbe : registre + citations minimaux (vue _latest comme en prod).
//
//	m_vide      aucun bit, aucune citation                  -> rien de fait
//	m_armes     MBitWeaponKills, deltas de citations         -> source faite, citations faites
//	m_absent    MBitFilmAbsent, sentinelle seule             -> source faite, film absent
//	m_sansfeed  aucun bit, killsource_sans_killfeed_rev      -> source faite (film lu, muet)
//	m_ancien    première synchro NULL                        -> FirstSyncAt nil
//	m_gen       citations en deux générations                -> citations faites (vue _latest)
func seedAnalysisProbe(t *testing.T, db *DB, firstSync time.Time) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE match_registry (match_id VARCHAR, first_sync_at TIMESTAMP,
			backfill_completed INTEGER, killsource_sans_killfeed_rev VARCHAR)`,
		`CREATE TABLE match_citations (match_id VARCHAR, citation_name_norm VARCHAR, value INTEGER, generation_id BIGINT)`,
		`CREATE VIEW match_citations_latest AS
			SELECT * EXCLUDE (rk) FROM (
				SELECT *, DENSE_RANK() OVER (PARTITION BY match_id ORDER BY generation_id DESC) AS rk
				FROM match_citations) WHERE rk = 1`,
	}
	for _, s := range stmts {
		if _, err := db.SQLDb().Exec(s); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, s)
		}
	}
	if _, err := db.SQLDb().Exec(`INSERT INTO match_registry VALUES
		('m_vide', ?, NULL, NULL),
		('m_armes', ?, ?, NULL),
		('m_absent', ?, ?, NULL),
		('m_sansfeed', ?, 0, 'rev-1'),
		('m_ancien', NULL, 0, NULL),
		('m_gen', ?, 0, NULL)`,
		firstSync,
		firstSync, matchflags.MBitWeaponKills,
		firstSync, matchflags.MBitFilmAbsent,
		firstSync,
		firstSync,
	); err != nil {
		t.Fatalf("seed registre: %v", err)
	}
	if _, err := db.SQLDb().Exec(`INSERT INTO match_citations VALUES
		('m_armes', 'bulltrue', 2, 1), ('m_armes', 'killjoy', 1, 1),
		('m_absent', '_processed', 0, 1),
		('m_gen', 'bulltrue', 1, 1), ('m_gen', '_processed', 0, 2)`); err != nil {
		t.Fatalf("seed citations: %v", err)
	}
}

func TestAnalysisProbeRepo_LoadAnalysisProbes(t *testing.T) {
	db := openMemDB(t)
	firstSync := time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)
	seedAnalysisProbe(t, db, firstSync)
	repo := NewAnalysisProbeRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})

	got, err := repo.LoadAnalysisProbes(context.Background(),
		[]string{"m_vide", "m_armes", "m_absent", "m_sansfeed", "m_ancien", "m_gen", "inconnu"})
	if err != nil {
		t.Fatalf("LoadAnalysisProbes: %v", err)
	}
	want := map[string]domain.AnalysisProbe{
		"m_vide":     {},
		"m_armes":    {KillSourceDone: true, HasCitationRows: true},
		"m_absent":   {KillSourceDone: true, FilmAbsent: true, HasCitationRows: true},
		"m_sansfeed": {KillSourceDone: true},
		"m_ancien":   {},
		"m_gen":      {HasCitationRows: true},
	}
	for id, w := range want {
		g, ok := got[id]
		if !ok {
			t.Fatalf("%s absent de la map", id)
		}
		hasSync := g.FirstSyncAt != nil
		if hasSync == (id == "m_ancien") {
			t.Fatalf("%s : FirstSyncAt = %v", id, g.FirstSyncAt)
		}
		if hasSync && !g.FirstSyncAt.Equal(firstSync) {
			t.Fatalf("%s : FirstSyncAt = %v, want %v", id, g.FirstSyncAt, firstSync)
		}
		g.FirstSyncAt = nil
		if g != w {
			t.Fatalf("%s = %+v, want %+v", id, g, w)
		}
	}
	if _, ok := got["inconnu"]; ok {
		t.Fatalf("un match absent du registre doit être absent de la map")
	}
}

func TestAnalysisProbeRepo_EmptyBatch(t *testing.T) {
	repo := NewAnalysisProbeRepo(&PlayerDB{})
	got, err := repo.LoadAnalysisProbes(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("lot vide : (%v, %v), want (vide, nil)", got, err)
	}
}
