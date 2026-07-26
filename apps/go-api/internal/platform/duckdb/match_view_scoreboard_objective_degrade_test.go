//go:build integration

// Package duckdb — match_view_scoreboard_objective_degrade_test.go : G1.
//
// Régression prod du 25/07 : Q12 portait un LEFT JOIN INCONDITIONNEL sur
// match_objective_stats_latest. Sur une DB où la vue manquait (fenêtre entre
// v7.2.0 et sa migration), DuckDB répondait Catalog Error → GetMatchScoreboard
// échouait → TOUTE la vue Match affichait « le match n'a pas pu être chargé »,
// alors que seule la section objectifs était concernée.
//
// Contrat vérifié ici : vue absente ⇒ scoreboard SERVI, section objectifs VIDE,
// WARN structuré émis AVANT la dégradation (CLAUDE.md règle n°3 — jamais
// d'erreur avalée en silence).
package duckdb

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// captureSlog redirige le logger par défaut vers un buffer JSON pour la durée du
// test et le restaure ensuite.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// hasWarnContaining indique si le buffer porte une ligne WARN dont le message
// contient chaque fragment attendu.
func hasWarnContaining(t *testing.T, buf *bytes.Buffer, fragments ...string) bool {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if lvl, _ := entry["level"].(string); lvl != "WARN" {
			continue
		}
		msg, _ := entry["msg"].(string)
		ok := true
		for _, f := range fragments {
			if !strings.Contains(msg, f) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// TestGetMatchScoreboard_MissingObjectiveView_ServesScoreboardAndWarns : la vue
// match_objective_stats_latest est absente → le scoreboard est servi entier, la
// section objectifs est vide, un WARN explicite est loggé.
func TestGetMatchScoreboard_MissingObjectiveView_ServesScoreboardAndWarns(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// SharedReader = LegacySharedReader(pdb.Player) : c'est CETTE conn que Q12
	// interroge. On y supprime la vue pour reproduire une DB non migrée.
	// NB : le message d'erreur ne répète PAS le littéral « VIEW <nom de la vue » —
	// le garde-rail archlint/no_inline_objective_latest_view_test.go l'interdit hors
	// de la source unique du DDL.
	if _, err := pdb.Player.Exec(ctx, `DROP VIEW IF EXISTS match_objective_stats_latest`); err != nil {
		t.Fatalf("suppression de la vue objectifs (fixture DB non migrée): %v", err)
	}

	buf := captureSlog(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	rows, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard doit RESTER servi sans la vue objectifs, erreur reçue: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("scoreboard attendu 1 joueur, obtenu %d — la dégradation a vidé le scoreboard", len(rows))
	}
	if rows[0].Obj.HasObjective() {
		t.Errorf("section objectifs attendue VIDE sans la vue, obtenu %+v", rows[0].Obj)
	}
	if !hasWarnContaining(t, buf, "objectifs") {
		t.Errorf("aucun WARN sur la dégradation de la section objectifs ; logs=%s", buf.String())
	}
}

// TestGetMatchScoreboard_ObjectiveViewPresent_PopulatesObjective : chemin nominal
// — avec la vue, les compteurs d'objectifs sont bien joints par xuid (la requête
// séparée Q12bObjectiveStats ne perd pas de données par rapport au LEFT JOIN).
func TestGetMatchScoreboard_ObjectiveViewPresent_PopulatesObjective(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_objective_stats
			(id, match_id, xuid, flag_captures, flag_grabs, time_as_flag_carrier_seconds, written_at)
		VALUES (1, ?, ?, 3, 5, 42.5, now())`, "m1", pTestXUID,
	); err != nil {
		t.Fatalf("INSERT match_objective_stats: %v", err)
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	rows, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 joueur, obtenu %d", len(rows))
	}
	obj := rows[0].Obj
	if !obj.HasCTF() {
		t.Fatalf("bloc CTF attendu non-nil, obtenu %+v", obj)
	}
	if obj.FlagCaptures == nil || *obj.FlagCaptures != 3 {
		t.Errorf("flag_captures = %v, want 3", obj.FlagCaptures)
	}
	if obj.FlagGrabs == nil || *obj.FlagGrabs != 5 {
		t.Errorf("flag_grabs = %v, want 5", obj.FlagGrabs)
	}
	if obj.TimeAsFlagCarrierSeconds == nil || *obj.TimeAsFlagCarrierSeconds != 42.5 {
		t.Errorf("time_as_flag_carrier_seconds = %v, want 42.5", obj.TimeAsFlagCarrierSeconds)
	}
}
