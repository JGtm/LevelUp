package replayartifacts

// selection_mode_non_gere_test.go — LE REJEU NE FABRIQUE PAS LE FIREFIGHT : ni le chemin des
// matchs inseres (selectBuildWork) ni la requete du rattrapage ne le retiennent.

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
)

func registreModesDeTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("duckdb en memoire : %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR, map_id VARCHAR,
		start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, backfill_completed INTEGER, is_firefight BOOLEAN)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry VALUES
		('pvp', 'Recharge', 'a', TIMESTAMP '2026-10-01 10:00:00', TIMESTAMP '2026-10-01 10:00:00', 0, FALSE),
		('pvp-sans-drapeau', 'Recharge', 'a', TIMESTAMP '2026-10-02 10:00:00', TIMESTAMP '2026-10-02 10:00:00', 0, NULL),
		('pve', 'Cole Protocol', 'b', TIMESTAMP '2026-10-03 10:00:00', TIMESTAMP '2026-10-03 10:00:00', 0, TRUE)`); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestSelectBuildWork_ExclutLeModeNonGere — les matchs inseres de mode non gere ne sont jamais
// retenus.
func TestSelectBuildWork_ExclutLeModeNonGere(t *testing.T) {
	db := registreModesDeTest(t)
	work := selectBuildWork(context.Background(), db, nil, []string{"pvp", "pvp-sans-drapeau", "pve"}, 0)
	got := map[string]bool{}
	for _, w := range work {
		got[w.matchID] = true
	}
	if len(got) != 2 || !got["pvp"] || !got["pvp-sans-drapeau"] || got["pve"] {
		t.Fatalf("retenus = %v, attendu pvp et pvp-sans-drapeau, jamais pve", got)
	}
}

// TestRequeteQueueRecente_ExclutLeModeNonGere — le rattrapage lit la file du predicat partage :
// le Firefight n'en fait pas partie, et la requete s'execute (arguments dans l'ordre).
func TestRequeteQueueRecente_ExclutLeModeNonGere(t *testing.T) {
	db := registreModesDeTest(t)
	sqlQueue, args := requeteQueueRecente(0)
	if !strings.Contains(sqlQueue, analysis.SQLModeNonGere("r")) {
		t.Fatalf("requete = %q : le predicat de mode non gere manque", sqlQueue)
	}
	rows, err := db.QueryContext(context.Background(), sqlQueue, append(args, BacklogHorizon)...)
	if err != nil {
		t.Fatalf("requete : %v", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		var name, mapID sql.NullString
		if err := rows.Scan(&id, &name, &mapID); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if len(ids) != 2 || ids[0] != "pvp-sans-drapeau" || ids[1] != "pvp" {
		t.Fatalf("file = %v, attendu [pvp-sans-drapeau pvp] (du plus recent au plus vieux, sans pve)", ids)
	}
}
