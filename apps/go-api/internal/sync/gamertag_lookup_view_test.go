//go:build integration

// Package sync — gamertag_lookup_view_test.go : garde-rail anti-régression sur
// le résolveur xuid → gamertag (analysis.GamertagLookupViewSQL).
//
// Contexte (fix 2026-05-30) : deux définitions divergentes de v_gamertag_lookup
// coexistaient (sync/schema.go simplifiée avec `WHERE gamertag IS NOT NULL` vs
// migration/steps_shared.go robuste). La version simplifiée — recréée à CHAQUE
// boot — gagnait, et des xuids bruts réapparaissaient à l'affichage. Désormais
// boot + migrations partagent analysis.GamertagLookupViewSQL.
//
// Ce test vérifie l'INVARIANT métier : la colonne gamertag de la vue ne contient
// JAMAIS un identifiant au format xuid brut (numérique long ou "xuid(...)").
//
// CGO requis (driver DuckDB) → tag integration.
// Lancer : go test -tags=integration ./internal/sync/ -run VGamertagLookup -v

package sync

import (
	"database/sql"
	"regexp"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/migration"
)

// rawXuidRe détecte un identifiant au format xuid brut joueur : soit "xuid(...)",
// soit une suite purement numérique de >= 15 chiffres (xuid Xbox décimal).
// Les bots "bid(N.0)" ne sont PAS visés (résolus à part via BotSQLCase).
var rawXuidRe = regexp.MustCompile(`(?i)^xuid\(|^[0-9]{15,}$`)

func TestGamertagLookupView_NeverLeaksRawXuid(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR,
			last_seen TIMESTAMP
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR
		);
		CREATE TABLE killer_victim_pairs (
			match_id VARCHAR,
			killer_xuid VARCHAR,
			killer_gamertag VARCHAR,
			victim_xuid VARCHAR,
			victim_gamertag VARCHAR
		);
	`); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	// Jeu de données hétérogène couvrant chaque branche du CASE :
	//  - alias renseigné (priorité 2)
	//  - alias vide mais participant renseigné (priorité 3)
	//  - participant gamertag vide (chaîne) et xuid absent des alias → masqué
	//  - participant gamertag NULL et xuid absent des alias → masqué
	//  - bot connu → nom officiel
	if _, err := db.Exec(`
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen) VALUES
			('2533274800000001', 'RealPlayer', '2026-05-30'),
			('2533274800000004', '',           '2026-05-30');
		INSERT INTO match_participants (match_id, xuid, gamertag) VALUES
			('m1', '2533274800000001', 'RealPlayer'),
			('m1', '2533274800000002', ''),
			('m1', '2533274800000003', NULL),
			('m1', '2533274800000004', 'FromParticipant'),
			('m1', '2533274800000005', ''),
			('m1', '2533274800000006', ''),
			('m1', 'bid(3.0)',         NULL);
		-- Adversaires sans alias ni gamertag participant, mais résolus depuis les
		-- films (kill-feed) : ...005 comme tueur, ...006 comme victime (priorité 4).
		INSERT INTO killer_victim_pairs (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag) VALUES
			('m1', '2533274800000005', 'FromKillFeed', '2533274800000006', 'VictimName');
	`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	// match_kill_events : le résolveur y lit le kill-feed canonique depuis le 2026-08-02,
	// EN PLUS de killer_victim_pairs. DuckDB bind les vues à leur création : la table doit
	// exister, même vide — et vide, elle prouve que la jambe historique tient toujours.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatalf("ensure match_kill_events: %v", err)
	}
	if _, err := db.Exec(analysis.GamertagLookupViewSQL()); err != nil {
		t.Fatalf("create v_gamertag_lookup: %v", err)
	}

	cases := []struct {
		name     string
		xuid     string
		expected string
	}{
		{"alias renseigné", "2533274800000001", "RealPlayer"},
		{"participant vide → masqué", "2533274800000002", "Joueur 0002"},
		{"participant NULL → masqué", "2533274800000003", "Joueur 0003"},
		{"alias vide, participant renseigné", "2533274800000004", "FromParticipant"},
		{"résolu via killer_victim_pairs (tueur)", "2533274800000005", "FromKillFeed"},
		{"résolu via killer_victim_pairs (victime)", "2533274800000006", "VictimName"},
		{"bot connu", "bid(3.0)", "343 Ellis"},
	}

	for _, c := range cases {
		var got string
		if err := db.QueryRow(
			`SELECT gamertag FROM v_gamertag_lookup WHERE xuid = ?`, c.xuid,
		).Scan(&got); err != nil {
			t.Errorf("%s: query xuid %s: %v", c.name, c.xuid, err)
			continue
		}
		if got != c.expected {
			t.Errorf("%s: xuid %s → got %q, want %q", c.name, c.xuid, got, c.expected)
		}
		if rawXuidRe.MatchString(got) {
			t.Errorf("%s: xuid %s → gamertag %q a un FORMAT XUID BRUT (fuite)", c.name, c.xuid, got)
		}
	}

	// Invariant global : aucune ligne de la vue n'expose un xuid brut.
	rows, err := db.Query(`SELECT xuid, gamertag FROM v_gamertag_lookup`)
	if err != nil {
		t.Fatalf("scan view: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var xuid, gt string
		if err := rows.Scan(&xuid, &gt); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		if rawXuidRe.MatchString(gt) {
			t.Errorf("ligne xuid=%s : gamertag %q au format xuid brut", xuid, gt)
		}
	}
}
