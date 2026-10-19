//go:build integration

// Package ops — seed_demo_film_tables_integration_test.go : les tables DÉRIVÉES DU FILM
// (filmDerivedTables) traversent la chaîne complète SeedDemo — extraction filtrée sur le corpus,
// anonymisation à la copie, migrations rejouées, vues `_latest` — et la base de démo produite les
// sert comme la production.
//
// Ce que le test exige, table par table (la table de cas ci-dessous est l'inventaire) :
//  1. la table est copiée, FILTRÉE sur le corpus (m3, hors corpus, n'y est pas) ;
//  2. toutes ses colonnes d'identité sont anonymisées : aucune valeur réelle ne subsiste
//     (scanDemoIdentityLeaks, le contrôle des valeurs du seed, rejoué sur la base produite) ;
//  3. sa vue `_latest` — l'unique chemin de lecture — se lie dans la base produite et rend la
//     DERNIÈRE passe seule, jamais la périmée ;
//  4. une identité de démo remplace l'identité réelle sur la ligne attendue.
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	filmSourceXUID = "1111111111111111" // le joueur source (JGtm) : roster principal, xuid démo 0
	filmOtherXUID  = "2222222222222222" // l'autre participant de m1
	oldPassAt      = "TIMESTAMP '2026-10-01 10:00:00'"
	newPassAt      = "TIMESTAMP '2026-10-02 10:00:00'"
)

// filmTableCase : une table dérivée du film, ses lignes sources et ce que la démo doit en servir.
type filmTableCase struct {
	table string
	// inserts : les lignes sources (m1 en deux passes quand la table en porte, plus m3 hors corpus).
	inserts []string
	// raw : lignes attendues dans la table copiée (m1 toutes passes ; m3 exclu).
	raw int
	// view / latest : la vue de lecture et ses lignes attendues (dernière passe de m1 seule) ;
	// view vide = table sans vue (clé naturelle).
	view   string
	latest int
}

func filmTableCases() []filmTableCase {
	src, oth := "'"+filmSourceXUID+"'", "'"+filmOtherXUID+"'"
	return []filmTableCase{
		{table: "match_lives", raw: 2, view: "match_lives_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', 'rev', ` + oldPassAt + `, ` + src + `, 0, 60000, 'death', 'kill_feed')`,
			`(2, 'm1', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 0, 61000, 'death', 'kill_feed')`,
			`(3, 'm3', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 0, 61000, 'death', 'kill_feed')`}},
		{table: "match_death_context", raw: 2, view: "match_death_context_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', 'rev', ` + oldPassAt + `, ` + src + `, 30000, 12.5, 1, 0, 0, 0, 1)`,
			`(2, 'm1', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 30000, 12.5, 1, 0, 0, 0, 1)`,
			`(3, 'm3', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 30000, 12.5, 1, 0, 0, 0, 1)`}},
		// DEUX colonnes d'identité : la victime et le coéquipier.
		{table: "match_death_mates", raw: 2, view: "match_death_mates_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', 'rev', ` + oldPassAt + `, ` + src + `, 30000, ` + oth + `, 'visible', 1.0, 2.0, 12.5)`,
			`(2, 'm1', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 30000, ` + oth + `, 'visible', 1.0, 2.0, 12.5)`,
			`(3, 'm3', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 30000, ` + oth + `, 'visible', 1.0, 2.0, 12.5)`}},
		{table: "match_life_placement", raw: 2, view: "match_life_placement_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', 'rev', ` + oldPassAt + `, ` + src + `, 0, 60000, 60000, 60000, 9.0, 0, 12.0, 0, 0, 0, 0, 1)`,
			`(2, 'm1', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 0, 60000, 60000, 60000, 9.0, 0, 12.0, 0, 0, 0, 0, 1)`,
			`(3, 'm3', 'p2', 'rev', ` + newPassAt + `, ` + src + `, 0, 60000, 60000, 60000, 9.0, 0, 12.0, 0, 0, 0, 0, 1)`}},
		{table: "kill_openings", raw: 2, view: "kill_openings_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', ` + src + `, 30000, 1, 2, 3, 4, 5, 6, ` + oldPassAt + `)`,
			`(2, 'm1', 'p2', ` + src + `, 30000, 1, 2, 3, 4, 5, 6, ` + newPassAt + `)`,
			`(3, 'm3', 'p2', ` + src + `, 30000, 1, 2, 3, 4, 5, 6, ` + newPassAt + `)`}},
		{table: "match_usage_films", raw: 2, view: "match_usage_films_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'u1', 'rev', 70, ` + oldPassAt + `, 100, 600, 60000, 3, 3, 0, '{}', '{}')`,
			`(2, 'm1', 'u2', 'rev', 70, ` + newPassAt + `, 100, 600, 60000, 3, 3, 0, '{}', '{}')`,
			`(3, 'm3', 'u2', 'rev', 70, ` + newPassAt + `, 100, 600, 60000, 3, 3, 0, '{}', '{}')`}},
		// La vue des joueurs suit la passe de la vue des FILMS (jointure), pas la sienne.
		{table: "match_usage_players", raw: 2, view: "match_usage_players_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'u1', 'rev', ` + oldPassAt + `, ` + src + usageCounters + `)`,
			`(2, 'm1', 'u2', 'rev', ` + newPassAt + `, ` + src + usageCounters + `)`,
			`(3, 'm3', 'u2', 'rev', ` + newPassAt + `, ` + src + usageCounters + `)`}},
		{table: "match_pad_pickups_by_tier", raw: 2, view: "match_pad_pickups_by_tier_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', ` + src + `, 'power', 'f1', 1, 1, 2, false, ` + oldPassAt + `)`,
			`(2, 'm1', 'p2', ` + src + `, 'power', 'f1', 1, 1, 2, false, ` + newPassAt + `)`,
			`(3, 'm3', 'p2', ` + src + `, 'power', 'f1', 1, 1, 2, false, ` + newPassAt + `)`}},
		{table: "match_vehicle_takes", raw: 2, view: "match_vehicle_takes_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', ` + oldPassAt + `, 'take', 0, ` + src + `, 'warthog', 2, 30000, 1, 0, 3` + vehicleCoverage + `)`,
			`(2, 'm1', 'p2', ` + newPassAt + `, 'take', 0, ` + src + `, 'warthog', 2, 30000, 1, 0, 3` + vehicleCoverage + `)`,
			`(3, 'm3', 'p2', ` + newPassAt + `, 'take', 0, ` + src + `, 'warthog', 2, 30000, 1, 0, 3` + vehicleCoverage + `)`}},
		{table: "match_flag_grabs_net", raw: 2, view: "match_flag_grabs_net_latest", latest: 1, inserts: []string{
			`(1, 'm1', 'p1', ` + src + `, 2, 1, 1, 5000, ` + oldPassAt + `)`,
			`(2, 'm1', 'p2', ` + src + `, 2, 1, 1, 5000, ` + newPassAt + `)`,
			`(3, 'm3', 'p2', ` + src + `, 2, 1, 1, 5000, ` + newPassAt + `)`}},
		// Une ligne par (match, joueur) : la vue rend la plus récente.
		{table: "match_bomb_stats", raw: 2, view: "match_bomb_stats_latest", latest: 1, inserts: []string{
			`(1, 'm1', ` + src + `, 1, 2, 3, 4.0, 5, ` + oldPassAt + `)`,
			`(2, 'm1', ` + src + `, 2, 2, 3, 4.0, 5, ` + newPassAt + `)`,
			`(3, 'm3', ` + src + `, 2, 2, 3, 4.0, 5, ` + newPassAt + `)`}},
		// Clé naturelle (match_id, seq), aucune vue : les deux tables sont lues telles quelles.
		{table: "match_objective_events", raw: 1, inserts: []string{
			`('m1', 1, 1000, 'bomb', 'bomb_armed', 0, 1, 1, 'film', 'high', '{}', ` + newPassAt + `)`,
			`('m3', 1, 1000, 'bomb', 'bomb_armed', 0, 1, 1, 'film', 'high', '{}', ` + newPassAt + `)`}},
		{table: "match_objective_event_players", raw: 1, inserts: []string{
			`('m1', 1, ` + oth + `, 'armer', ` + newPassAt + `)`,
			`('m3', 1, ` + oth + `, 'armer', ` + newPassAt + `)`}},
		{table: "match_player_positions", raw: 2, view: "match_player_positions_latest", latest: 1, inserts: []string{
			`(1, 'm1', 20000, 1.0, 2.0, 3.0, 0, ` + oldPassAt + `, 'pos1')`,
			`(2, 'm1', 20000, 1.0, 2.0, 3.0, 0, ` + newPassAt + `, 'pos2')`,
			`(3, 'm3', 20000, 1.0, 2.0, 3.0, 0, ` + newPassAt + `, 'pos2')`}},
	}
}

// filmInsertColumns : la liste de colonnes de chaque INSERT ci-dessus (les autres prennent leur
// défaut). Séparée des lignes pour que celles-ci restent lisibles.
var filmInsertColumns = map[string]string{
	"match_lives":                   `(id, match_id, decode_pass, decoder_rev, written_at, xuid, start_ms, end_ms, end_cause, named_by)`,
	"match_death_context":           `(id, match_id, decode_pass, decoder_rev, written_at, victim_xuid, time_ms, nearest_teammate_m, teammates_visible, teammates_waiting, teammates_out_of_sight, teammates_left, teammates_total)`,
	"match_death_mates":             `(id, match_id, decode_pass, decoder_rev, written_at, victim_xuid, time_ms, mate_xuid, state, x, y, distance_m)`,
	"match_life_placement":          `(id, match_id, decode_pass, decoder_rev, written_at, xuid, start_ms, end_ms, duration_ms, measured_ms, median_m, beyond_ms, radar_m, carrier_ms, team_down_ms, unplaced_ms, teammate_unplaced_ms, kills)`,
	"kill_openings":                 `(id, match_id, decode_pass, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z, written_at)`,
	"match_usage_films":             `(id, match_id, summary_pass, summary_rev, artifact_schema, written_at, frame_interval_ms, frame_count, duration_ms, pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)`,
	"match_usage_players":           `(id, match_id, summary_pass, summary_rev, written_at, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills, overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown, pad_pickups, pad_pickups_json)`,
	"match_pad_pickups_by_tier":     `(id, match_id, decode_pass, xuid, tier, weapon_family, pickups, pads_confirmed, pads_total, random_starts, written_at)`,
	"match_vehicle_takes":           `(id, match_id, decode_pass, written_at, row_kind, camp, xuid, family, takes, aboard_ms, episodes, proximity_episodes, frags, measured, unmeasured_reason, doc_schema, episodes_read, episodes_unnamed, episodes_no_camp, frags_read, frags_reason, frags_total, frags_unmatched)`,
	"match_flag_grabs_net":          `(id, match_id, decode_pass, xuid, flag_grabs_raw, flag_grabs_net, openings, juggle_window_ms, written_at)`,
	"match_bomb_stats":              `(id, match_id, xuid, bomb_detonations, bomb_arms, bomb_grabs, time_as_bomb_carrier_seconds, bomb_carriers_killed, written_at)`,
	"match_objective_events":        `(match_id, seq, time_ms, objective_type, event_type, team_id, objective_id, value, source, confidence, details, written_at)`,
	"match_objective_event_players": `(match_id, seq, xuid, role, written_at)`,
	"match_player_positions":        `(id, match_id, time_ms, x, y, z, team, written_at, positions_pass)`,
}

// seedFilmRowsInSource pose les lignes de filmTableCases dans la base partagée SOURCE.
func seedFilmRowsInSource(t *testing.T, srcShared string) {
	t.Helper()
	db, err := sql.Open("duckdb", srcShared)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, tc := range filmTableCases() {
		cols, ok := filmInsertColumns[tc.table]
		if !ok {
			t.Fatalf("%s : colonnes d'insertion non déclarées", tc.table)
		}
		for _, row := range tc.inserts {
			mustExec(t, db, `INSERT INTO `+tc.table+` `+cols+` VALUES `+row)
		}
	}
}

func TestSeedDemo_TablesDeriveesDuFilm_ExtraitesAnonymiseesEtLisibles(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	seedFilmRowsInSource(t, srcShared)
	outDir := filepath.Join(tmpDir, "out")

	// MaxMatches 2 : m1 + m2 (les plus récents) ; m3 reste hors corpus.
	res, err := SeedDemo(context.Background(), SeedDemoOptions{
		SourcePlayerDB: srcPlayer, SourceSharedDB: srcShared, SourceMetaDB: srcMeta,
		SourceXUID: filmSourceXUID, OutDir: outDir, MaxMatches: 2, SourceLabel: "JGtm", ServiceTag: "SPTA",
	})
	if err != nil {
		t.Fatalf("SeedDemo (le contrôle des valeurs de fin de seed a joué) : %v", err)
	}
	outShared := filepath.Join(outDir, "warehouse", "shared_matches_v2.duckdb")

	db, err := sql.Open("duckdb", outShared+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range filmTableCases() {
		if got := res.SharedRows[tc.table]; got != tc.raw {
			t.Errorf("%s : %d ligne(s) extraite(s), attendu %d (m1 toutes passes ; m3 hors corpus exclu)", tc.table, got, tc.raw)
		}
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ` + tc.table + ` WHERE match_id = 'm3'`).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s : %d ligne(s) de m3 (hors corpus), err %v — l'extraction n'est pas filtrée", tc.table, n, err)
		}
		if tc.view == "" {
			continue
		}
		if err := db.QueryRow(`SELECT count(*) FROM ` + tc.view).Scan(&n); err != nil {
			t.Errorf("%s illisible dans la démo produite : %v", tc.view, err)
		} else if n != tc.latest {
			t.Errorf("%s : %d ligne(s), attendu %d (la dernière passe seule)", tc.view, n, tc.latest)
		}
	}
	verifyFilmTablesAnonymised(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// (2) Le contrôle des valeurs, rejoué sur la base produite avec les identités réelles de la fixture.
	real := realIdentities{gamertags: map[string]bool{"JGtm": true, "Other": true}}
	real.add(filmSourceXUID, "")
	real.add(filmOtherXUID, "")
	leaks, err := scanDemoIdentityLeaks(context.Background(), outShared, real)
	if err != nil {
		t.Fatalf("scan des fuites : %v", err)
	}
	for _, l := range leaks {
		t.Errorf("FUITE d'identité réelle dans la démo produite : %s", l.where())
	}
}

// verifyFilmTablesAnonymised : l'identité de démo remplace la réelle sur la ligne attendue —
// y compris la SECONDE colonne de match_death_mates (le coéquipier) et celle d'une table à clé
// naturelle (les joueurs d'un fait daté).
func verifyFilmTablesAnonymised(t *testing.T, db *sql.DB) {
	t.Helper()
	var victim, mate string
	if err := db.QueryRow(`SELECT victim_xuid, mate_xuid FROM match_death_mates_latest`).Scan(&victim, &mate); err != nil {
		t.Fatalf("match_death_mates_latest : %v", err)
	}
	if victim != DefaultDemoXUID {
		t.Errorf("match_death_mates.victim_xuid = %q, attendu l'identité démo du joueur source %q", victim, DefaultDemoXUID)
	}
	if mate == filmOtherXUID || mate == "" || mate == victim {
		t.Errorf("match_death_mates.mate_xuid = %q : doit être l'identité démo du coéquipier, distincte de la victime", mate)
	}
	var player string
	if err := db.QueryRow(`SELECT xuid FROM match_objective_event_players WHERE match_id = 'm1'`).Scan(&player); err != nil {
		t.Fatalf("match_objective_event_players : %v", err)
	}
	if player != mate {
		t.Errorf("match_objective_event_players.xuid = %q, attendu la même identité démo que mate_xuid (%q) pour le même joueur réel", player, mate)
	}
}

// usageCounters : les colonnes NOT NULL de match_usage_players sans défaut (compteurs à zéro,
// documents JSON vides) — la suite de la liste de colonnes de filmInsertColumns.
const usageCounters = `, 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}'`

// vehicleCoverage : les colonnes de couverture NOT NULL de match_vehicle_takes (mesuré, schéma 70,
// épisodes lus, frags lus) — la suite de la liste de colonnes de filmInsertColumns.
const vehicleCoverage = `, true, '', 70, 1, 0, 0, true, '', 3, 0`
