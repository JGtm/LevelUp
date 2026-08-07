//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestPSAAppendOnly_LegacySwap_NoIdNoCreatedAt verrouille les deux bugs trouvés
// par le crash-test sur données RÉELLES (2026-06-21) : les vraies tables
// personal_score_awards / match_citations des player DBs Madina/JGtm n'ont NI
// colonne `id` NI colonne `created_at` (créées par un ancien schéma / pipeline
// Python). Les fixtures :memory: les avaient → le bug ne sortait qu'en réel.
//
// La conversion append-only doit donc : ajouter `id` si absent (via séquence),
// utiliser l horloge UTC pour written_at (pas COALESCE(created_at, …)), et
// préserver toutes les données.
func TestPSAAppendOnly_LegacySwap_NoIdNoCreatedAt(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Table LEGACY : pas de id, pas de created_at, pas de PK technique.
	if _, err := db.Exec(`
		CREATE TABLE personal_score_awards (
			match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
			award_name VARCHAR NOT NULL, award_category VARCHAR,
			award_count INTEGER DEFAULT 1, award_score INTEGER DEFAULT 0)`); err != nil {
		t.Fatalf("legacy psa: %v", err)
	}
	for _, r := range []struct {
		m, x, name, cat string
		cnt, score      int
	}{
		{"m1", "u1", "killjoy", "combat", 1, 100},
		{"m1", "u1", "perfection", "combat", 1, 200},
		{"m2", "u1", "objective", "objective", 2, 50},
	} {
		if _, err := db.Exec(`INSERT INTO personal_score_awards
			(match_id, xuid, award_name, award_category, award_count, award_score)
			VALUES (?, ?, ?, ?, ?, ?)`, r.m, r.x, r.name, r.cat, r.cnt, r.score); err != nil {
			t.Fatalf("seed psa: %v", err)
		}
	}

	if err := EnsurePersonalScoreAwardsAppendOnly(db); err != nil {
		t.Fatalf("EnsurePersonalScoreAwardsAppendOnly (legacy sans id/created_at): %v", err)
	}

	// id + generation_id + written_at ajoutés.
	for _, col := range []string{"id", "generation_id", "written_at"} {
		ok, err := columnExists(db, "personal_score_awards", col)
		if err != nil || !ok {
			t.Fatalf("colonne %q absente après conversion (err=%v)", col, err)
		}
	}
	// Données préservées + vue _latest opérationnelle.
	var phys, latest int
	db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&phys)
	db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards_latest`).Scan(&latest)
	if phys != 3 || latest != 3 {
		t.Fatalf("rows: physique=%d latest=%d, want 3/3 (zéro perte)", phys, latest)
	}
	// SUM lisible via _latest (le reader réel).
	var total int
	db.QueryRow(`SELECT COALESCE(SUM(award_score),0) FROM personal_score_awards_latest
		WHERE match_id='m1' AND xuid='u1'`).Scan(&total)
	if total != 300 {
		t.Errorf("SUM award_score m1/u1 = %d, want 300", total)
	}
	// Décision 2026-08-05 : le PostSwap ne recrée NI idx_psa_xuid (sélectivité nulle,
	// miroir d'idx_career_xuid) NI idx_psa_match_xuid (pur préfixe d'idx_psa_gen ; ce
	// PostSwap était sa SEULE autorité — divergence fraîche/convertie refermée). Les DB
	// qui les portent encore convergent via les steps drop_psa_*_art_index_v1.
	for _, idx := range []string{"idx_psa_xuid", "idx_psa_match_xuid"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, idx).Scan(&n); err != nil {
			t.Fatalf("duckdb_indexes(%s): %v", idx, err)
		}
		if n != 0 {
			t.Errorf("index %s recréé par la conversion — supprimé de toutes les autorités le 2026-08-05", idx)
		}
	}
}

func TestMatchCitationsAppendOnly_LegacySwap_NoIdNoCreatedAt(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Table LEGACY : PK composite (match_id, citation_name_norm), pas de id, pas de created_at.
	if _, err := db.Exec(`
		CREATE TABLE match_citations (
			match_id VARCHAR NOT NULL, citation_name_norm VARCHAR NOT NULL,
			value INTEGER NOT NULL DEFAULT 1,
			PRIMARY KEY (match_id, citation_name_norm))`); err != nil {
		t.Fatalf("legacy citations: %v", err)
	}
	for _, r := range []struct {
		m, name string
		v       int
	}{
		{"m1", "bulltrue", 3}, {"m1", "killjoy", 1}, {"m2", "_processed", 0},
	} {
		if _, err := db.Exec(`INSERT INTO match_citations (match_id, citation_name_norm, value)
			VALUES (?, ?, ?)`, r.m, r.name, r.v); err != nil {
			t.Fatalf("seed citations: %v", err)
		}
	}

	if err := EnsureMatchCitationsAppendOnly(db); err != nil {
		t.Fatalf("EnsureMatchCitationsAppendOnly (legacy sans id/created_at): %v", err)
	}

	for _, col := range []string{"id", "generation_id", "written_at"} {
		ok, err := columnExists(db, "match_citations", col)
		if err != nil || !ok {
			t.Fatalf("colonne %q absente après conversion (err=%v)", col, err)
		}
	}
	var phys, latest int
	db.QueryRow(`SELECT COUNT(*) FROM match_citations`).Scan(&phys)
	db.QueryRow(`SELECT COUNT(*) FROM match_citations_latest`).Scan(&latest)
	if phys != 3 || latest != 3 {
		t.Fatalf("rows: physique=%d latest=%d, want 3/3 (zéro perte)", phys, latest)
	}
	// La PK composite a bien été remplacée par un id technique.
	if ok, _ := hasPrimaryKey(db, "match_citations"); !ok {
		t.Error("match_citations devrait avoir une PK (id technique) après conversion")
	}
}
