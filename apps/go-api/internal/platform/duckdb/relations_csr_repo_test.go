//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"
)

// seedRelationsCSR crée un schéma minimal match_csrs (+ vue append-only
// match_csrs_latest) et match_registry pour exercer GetLatestCSR. Deux joueurs :
//   - xuidRanked : deux matchs classés (ancien Diamant, récent Onyx) → le snapshot
//     retenu doit être le PLUS RÉCENT (Onyx, par start_time canonique).
//   - xuidPlacement : un match en « Placement » → non affichable (nil attendu).
//   - xuidSocial : aucune ligne CSR → nil attendu.
func seedRelationsCSR(t *testing.T, db *DB, now time.Time) {
	t.Helper()
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMPTZ, start_time TIMESTAMP)`,
		`CREATE TABLE match_csrs (
			match_id VARCHAR, xuid VARCHAR, rating_type VARCHAR,
			rating_value FLOAT, tier VARCHAR, sub_tier SMALLINT,
			tier_label VARCHAR, written_at TIMESTAMP)`,
		`CREATE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC) = 1`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("seedRelationsCSR DDL: %v\nSQL: %s", err, ddl)
		}
	}
	day := 24 * time.Hour
	if _, err := db.Exec(ctx, `INSERT INTO match_registry VALUES
		('mOld', ?, NULL),
		('mNew', ?, NULL),
		('mPlac', ?, NULL)`,
		now.Add(-30*day), now.Add(-2*day), now.Add(-5*day)); err != nil {
		t.Fatalf("seedRelationsCSR registry: %v", err)
	}
	// xuidRanked : Diamant 3 (ancien) puis Onyx 1523 (récent).
	// xuidPlacement : sentinelle « Placement ».
	if _, err := db.Exec(ctx, `INSERT INTO match_csrs VALUES
		('mOld','xuidRanked','CSR', 1450, 'Diamond', 3, 'Diamond 3', ?),
		('mNew','xuidRanked','CSR', 1523, 'Onyx',    0, 'Onyx 1523', ?),
		('mPlac','xuidPlacement','CSR', NULL, 'Placement', 0, 'Placement', ?)`,
		now, now, now); err != nil {
		t.Fatalf("seedRelationsCSR csrs: %v", err)
	}
}

// TestCareerRepo_GetLatestCSR : le snapshot CSR le plus récent est retourné pour un
// joueur classé ; nil (dégradation gracieuse) pour un joueur social, en placement,
// ou un xuid vide.
func TestCareerRepo_GetLatestCSR(t *testing.T) {
	now := time.Now().UTC()

	db := openMemDB(t)
	seedRelationsCSR(t, db, now)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)
	ctx := context.Background()

	// Joueur classé : le snapshot le PLUS RÉCENT (Onyx 1523) l'emporte sur l'ancien.
	csr, err := repo.GetLatestCSR(ctx, "xuidRanked")
	if err != nil {
		t.Fatalf("GetLatestCSR(ranked): %v", err)
	}
	if csr == nil || csr.Tier == nil || *csr.Tier != "Onyx" {
		t.Fatalf("ranked CSR=%+v want tier Onyx (le plus récent)", csr)
	}
	if csr.RatingValue == nil || *csr.RatingValue != 1523 {
		t.Fatalf("ranked CSR rating=%v want 1523", csr.RatingValue)
	}
	if csr.SubTier == nil || *csr.SubTier != 0 {
		t.Fatalf("ranked CSR sub_tier=%v want 0 (Onyx ouvert)", csr.SubTier)
	}

	// Joueur social : aucune ligne CSR → nil, pas d'erreur.
	social, err := repo.GetLatestCSR(ctx, "xuidSocial")
	if err != nil {
		t.Fatalf("GetLatestCSR(social): %v", err)
	}
	if social != nil {
		t.Fatalf("social CSR=%+v want nil (dégradation gracieuse)", social)
	}

	// Placement : palier non affichable → nil.
	plac, err := repo.GetLatestCSR(ctx, "xuidPlacement")
	if err != nil {
		t.Fatalf("GetLatestCSR(placement): %v", err)
	}
	if plac != nil {
		t.Fatalf("placement CSR=%+v want nil", plac)
	}

	// xuid vide → nil sans requête.
	empty, err := repo.GetLatestCSR(ctx, "")
	if err != nil {
		t.Fatalf("GetLatestCSR(empty): %v", err)
	}
	if empty != nil {
		t.Fatalf("empty xuid CSR=%+v want nil", empty)
	}
}
