//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
)

// TestCampaignExclusion_FiltersCampaignMatch — preuve COMPORTEMENTALE (item H1) :
// sur une DuckDB en mémoire reproduisant match_registry ⨝ match_participants, une
// requête de stats portant le token d'exclusion résolu pour Halo 5 EXCLUT le match
// de mode Campagne (identifié par game_variant_id) tout en gardant le match
// d'arène ; résolue pour Infinite (no-op), elle renvoie les deux (title-aware).
func TestCampaignExclusion_FiltersCampaignMatch(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	// Schéma minimal (colonnes utilisées par la requête témoin).
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE match_registry (match_id VARCHAR, game_variant_id VARCHAR);
		CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	campGUID := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	const arenaGUID = "aaaaaaaa-0000-0000-0000-000000000001"
	const xuid = "2533274800000001"

	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry VALUES ('arena1', ?), ('camp1', ?)`, arenaGUID, campGUID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_participants VALUES ('arena1', ?), ('camp1', ?)`, xuid, xuid); err != nil {
		t.Fatalf("seed participants: %v", err)
	}

	// Requête témoin : même squelette JOIN + token que les requêtes de stats
	// per-player réelles (Q22/Q23/Q26/...).
	base := `
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry r ON r.match_id = mp.match_id
		WHERE mp.xuid = ? ` + campaignExclusionToken + `
		ORDER BY mp.match_id`

	// Halo 5 : le match Campagne doit disparaître.
	got := queryMatchIDs(t, db, resolveCampaignExclusion(base, "halo_5", "r"), xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("halo_5 : attendu %v (Campagne exclue), obtenu %v", want, got)
	}

	// Infinite : no-op, les deux matchs remontent (le GUID Campagne n'existe pas
	// pour Infinite → aucune régression sur les autres titres).
	got = queryMatchIDs(t, db, resolveCampaignExclusion(base, "halo_infinite", "r"), xuid)
	if want := []string{"arena1", "camp1"}; !equalStrings(got, want) {
		t.Errorf("halo_infinite : attendu %v (no-op), obtenu %v", want, got)
	}

	// Forme SOUS-REQUÊTE (participants-only, SANS jointure registre — chemin
	// weapon_kills / compare / count / leaderboard).
	sub := `SELECT mp.match_id FROM match_participants mp WHERE mp.xuid = ?` +
		excludeCampaignByMatchID("halo_5", "mp.match_id") + ` ORDER BY mp.match_id`
	got = queryMatchIDs(t, db, sub, xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("sous-requête halo_5 : attendu %v (Campagne exclue), obtenu %v", want, got)
	}

	// Variante title-agnostic (GamertagRepo) : mêmes GUID, no-op si absents.
	subAll := `SELECT mp.match_id FROM match_participants mp WHERE mp.xuid = ?` +
		excludeAllCampaignByMatchID("mp.match_id") + ` ORDER BY mp.match_id`
	got = queryMatchIDs(t, db, subAll, xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("sous-requête title-agnostic : attendu %v, obtenu %v", want, got)
	}
}

func queryMatchIDs(t *testing.T, db *sql.DB, query, xuid string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), query, xuid)
	if err != nil {
		t.Fatalf("query: %v\n%s", err, query)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
