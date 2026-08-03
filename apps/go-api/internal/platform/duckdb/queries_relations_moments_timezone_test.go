// queries_relations_moments_timezone_test.go — non-régression V7.3 lot 2 (item 1.3) :
// Q29RelationsHeatmapTpl doit rendre hour_local / dow_local dans le FUSEAU DE SESSION
// DuckDB (SET TimeZone = cfg.UserTimezone), pas en UTC.
//
// Avant le correctif, la projection était
// `EXTRACT(hour FROM (COALESCE(start_time_utc, start_time) AT TIME ZONE 'UTC'))` :
// le `AT TIME ZONE 'UTC'` posé APRÈS le COALESCE reconvertit le TIMESTAMPTZ en
// TIMESTAMP naïf UTC, ce qui annule le fuseau de session. Ces tests échouent sur
// l'ancienne forme (heures UTC 22/22/20) et passent sur la forme canonique.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// heatmapTZFixture décrit un match du jeu d'essai + l'heure/jour locale attendue.
type heatmapTZFixture struct {
	matchID     string
	label       string
	wantHourTZ  int // heure attendue en Europe/Paris
	wantDowTZ   int // jour attendu en Europe/Paris (0=dimanche)
	wantHourUTC int // heure attendue quand la session est en UTC
	wantDowUTC  int // jour attendu quand la session est en UTC
}

// Jeu d'essai : trois matchs communs avec la même relation (le HAVING >= 2 de la
// requête impose au moins deux matchs pour qu'une relation entre dans le top-N).
//
//	m1 : 22h30 UTC un 10 juillet — heure d'ÉTÉ (Paris = UTC+2) → 00h30 le 11, samedi.
//	m2 : 22h30 UTC un 10 janvier — heure d'HIVER (Paris = UTC+1) → 23h30 le 10, samedi.
//	m3 : start_time_utc NULL, fallback sur start_time (TIMESTAMP naïf stocké en UTC)
//	     → doit être interprété en UTC par le fragment canonique, puis rendu en Paris.
var heatmapTZFixtures = []heatmapTZFixture{
	{matchID: "m1", label: "ete (DST +2)", wantHourTZ: 0, wantDowTZ: 6, wantHourUTC: 22, wantDowUTC: 5},
	{matchID: "m2", label: "hiver (+1)", wantHourTZ: 23, wantDowTZ: 6, wantHourUTC: 22, wantDowUTC: 6},
	{matchID: "m3", label: "fallback start_time naif", wantHourTZ: 0, wantDowTZ: 6, wantHourUTC: 22, wantDowUTC: 5},
}

// openHeatmapTZDB ouvre une DuckDB :memory: avec la timezone de session appliquée
// par le MÊME chemin qu'en production (openSQLDBFor → applyDuckSessionInit →
// SET TimeZone sur chaque connexion), puis sème le schéma minimal de Q29.
func openHeatmapTZDB(t *testing.T, timezone string) *DB {
	t.Helper()
	sqlDB, err := openSQLDBFor(":memory:", timezone, "test", ":memory:")
	if err != nil {
		t.Fatalf("openSQLDBFor(:memory:, %s): %v", timezone, err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	seedHeatmapTZ(t, sqlDB)
	return newTestDB(sqlDB, ":memory:")
}

func seedHeatmapTZ(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER)`,
		`CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMPTZ, start_time TIMESTAMP)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM xuid_aliases`,
		`INSERT INTO match_participants VALUES
			('m1','xuidMe',0), ('m1','xuidAlly',0),
			('m2','xuidMe',0), ('m2','xuidAlly',0),
			('m3','xuidMe',0), ('m3','xuidAlly',0)`,
		// m1/m2 : instants UTC explicites. m3 : start_time_utc NULL → le fragment
		// canonique doit lire start_time comme un instant UTC (et non dans le
		// fuseau de session, piège du COALESCE mal parenthésé).
		`INSERT INTO match_registry VALUES
			('m1', TIMESTAMPTZ '2026-07-10 22:30:00+00', NULL),
			('m2', TIMESTAMPTZ '2026-01-10 22:30:00+00', NULL),
			('m3', NULL, TIMESTAMP '2026-07-10 22:30:00')`,
		`INSERT INTO xuid_aliases VALUES ('xuidMe','MePlayer'), ('xuidAlly','AllyPlayer')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seedHeatmapTZ: %v\nSQL: %s", err, s)
		}
	}
}

// queryHeatmapHoursByMatch réexécute la projection de Q29 match par match, pour
// pouvoir rattacher chaque (heure, jour) rendu à son match d'origine. L'expression
// testée est exactement celle de la requête servie (mêmes fragments).
func queryHeatmapHoursByMatch(t *testing.T, db *DB) map[string][2]int {
	t.Helper()
	sqlText := `SELECT r.match_id,
		EXTRACT(hour FROM ` + StartTimeCanonicalSQL("r") + `)::INTEGER,
		EXTRACT(dow FROM ` + StartTimeCanonicalSQL("r") + `)::INTEGER
		FROM match_registry r ORDER BY r.match_id`
	rows, err := db.Query(context.Background(), sqlText)
	if err != nil {
		t.Fatalf("projection heure locale: %v", err)
	}
	defer rows.Close()

	out := map[string][2]int{}
	for rows.Next() {
		var (
			id        string
			hour, dow int
		)
		if err := rows.Scan(&id, &hour, &dow); err != nil {
			t.Fatalf("scan projection: %v", err)
		}
		out[id] = [2]int{hour, dow}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return out
}

// TestQ29Heatmap_HoursInUserTimezone : avec une session Europe/Paris, hour_local
// doit être l'heure de Paris (décalage +2 en été, +1 en hiver), jamais l'heure UTC.
func TestQ29Heatmap_HoursInUserTimezone(t *testing.T) {
	db := openHeatmapTZDB(t, "Europe/Paris")
	got := queryHeatmapHoursByMatch(t, db)

	for _, f := range heatmapTZFixtures {
		cell, ok := got[f.matchID]
		if !ok {
			t.Fatalf("%s (%s) absent de la projection", f.matchID, f.label)
		}
		if cell[0] != f.wantHourTZ {
			t.Errorf("%s (%s) : hour_local=%d, attendu %d (Europe/Paris) — l'heure servie est encore en UTC (%d)",
				f.matchID, f.label, cell[0], f.wantHourTZ, f.wantHourUTC)
		}
		if cell[1] != f.wantDowTZ {
			t.Errorf("%s (%s) : dow_local=%d, attendu %d (Europe/Paris)",
				f.matchID, f.label, cell[1], f.wantDowTZ)
		}
	}
}

// TestQ29Heatmap_HoursFollowSessionTimezone : la MÊME requête sur les MÊMES données,
// session en UTC, doit rendre les heures UTC. Preuve que la projection est bien
// pilotée par cfg.UserTimezone et non par une constante figée.
func TestQ29Heatmap_HoursFollowSessionTimezone(t *testing.T) {
	db := openHeatmapTZDB(t, "UTC")
	got := queryHeatmapHoursByMatch(t, db)

	for _, f := range heatmapTZFixtures {
		cell, ok := got[f.matchID]
		if !ok {
			t.Fatalf("%s (%s) absent de la projection", f.matchID, f.label)
		}
		if cell[0] != f.wantHourUTC || cell[1] != f.wantDowUTC {
			t.Errorf("%s (%s) session UTC : (hour,dow)=(%d,%d), attendu (%d,%d)",
				f.matchID, f.label, cell[0], cell[1], f.wantHourUTC, f.wantDowUTC)
		}
	}
}

// TestQ29Heatmap_EndToEndLocalHours : la requête complète servie par le repo
// (GetRelationsHeatmap) rend bien des heures locales. Les trois matchs communs
// tombent sur deux heures locales distinctes en Europe/Paris (00h ×2, 23h ×1)
// alors qu'en UTC ils tomberaient tous à 22h — l'assertion discrimine donc
// directement l'ancien comportement.
func TestQ29Heatmap_EndToEndLocalHours(t *testing.T) {
	db := openHeatmapTZDB(t, "Europe/Paris")
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer", TitleSlug: "halo_infinite"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelationsHeatmap(context.Background(), nil, 8)
	if err != nil {
		t.Fatalf("GetRelationsHeatmap: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("aucune ligne de heatmap — le jeu d'essai doit produire la relation AllyPlayer")
	}

	countByHour := map[int]int{}
	for _, r := range rows {
		if r.Gamertag != "AllyPlayer" {
			t.Fatalf("relation inattendue %q", r.Gamertag)
		}
		if r.Dow != 6 {
			t.Errorf("dow=%d attendu 6 (samedi en Europe/Paris) pour l'heure %d", r.Dow, r.Hour)
		}
		countByHour[r.Hour] += r.Count
	}
	if countByHour[0] != 2 {
		t.Errorf("heure locale 00h : %d matchs, attendu 2 (m1 ete + m3 fallback) — heures rendues : %v",
			countByHour[0], countByHour)
	}
	if countByHour[23] != 1 {
		t.Errorf("heure locale 23h : %d matchs, attendu 1 (m2 hiver) — heures rendues : %v",
			countByHour[23], countByHour)
	}
	if countByHour[22] != 0 {
		t.Errorf("heure 22h non vide (%d) : la requête sert encore des heures UTC", countByHour[22])
	}
}
