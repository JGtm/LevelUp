//go:build integration

// match_recomputer_lusr_v2_integration_test.go : le recalcul d'une exclusion de match
// passe par le LUSR v2 (D-6 du plan backlog 2026-10-16).
//
// Avant ce lot, MatchRecomputer rejouait la v1 en force : chaque exclusion ajoutait, pour
// tous les matchs, une ligne rating_type='LUSR' plus récente que la ligne v2, et la vue
// match_skill_rank_latest servait alors la valeur v1 (le slot « LUSR » lu par l'interface
// est arbitré sur written_at). Le contrat prouvé ici : après une exclusion, la note servie
// est celle de la v2 rejouée SANS le match exclu.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// lusrV2State rend, pour une base partagée fermée, le mu et l'expérience de l'état v2 du joueur
// de test (une seule chaîne dans la fixture).
func lusrV2State(t *testing.T, sharedPath string) (mu float64, experience int) {
	t.Helper()
	db, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open shared %s: %v", sharedPath, err)
	}
	defer func() { _ = db.Close() }()
	if err := db.QueryRow(
		`SELECT mu, experience FROM player_skill_state_v2_latest WHERE xuid = ?`, recomputerTestXUID,
	).Scan(&mu, &experience); err != nil {
		t.Fatalf("lecture de l'état v2: %v", err)
	}
	return mu, experience
}

// lusrServed rend, par match, la valeur que sert la vue donnée (match_skill_rank_latest : la ligne que lit
// l'interface ; match_skill_rank_latest_by_type : la plus récente par type, que lit le graphe d'évolution)
// pour le type de notation donné.
func lusrServed(t *testing.T, playerPath, view, ratingType string) map[string]float64 {
	t.Helper()
	db, err := sql.Open("duckdb", playerPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open player %s: %v", playerPath, err)
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(
		`SELECT match_id, rating_value FROM `+view+` WHERE rating_type = ?`, ratingType)
	if err != nil {
		t.Fatalf("lecture %s(%s): %v", view, ratingType, err)
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var id string
		var v float64
		if err := rows.Scan(&id, &v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[id] = v
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// replayFullLUSRV2 rejoue le LUSR v2 complet du joueur de test sur des bases fermées ensuite.
func replayFullLUSRV2(t *testing.T, sharedPath, playerPath string) int {
	t.Helper()
	seedLUSRPlayer(t, playerPath)
	shared, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer shared.Close()
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	n, err := RecomputeLUSRCanonicalForPlayer(context.Background(), player.SQLDb(), shared.SQLDb(), recomputerTestXUID)
	if err != nil {
		t.Fatalf("RecomputeLUSRCanonicalForPlayer: %v", err)
	}
	return n
}

func TestMatchRecomputer_ExclusionServeLaNoteV2SansLeMatchExclu(t *testing.T) {

	const nMatches = 12
	excludedID := fmt.Sprintf("00000000-0000-0000-0000-%012d", 4)

	// Base testée : v2 complète, puis exclusion du match 4 et recalcul par le recalculateur.
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, nMatches)
	if n := replayFullLUSRV2(t, sharedPath, playerPath); n != nMatches {
		t.Fatalf("v2 initiale : %d matchs traités, attendu %d", n, nMatches)
	}
	if _, exp := lusrV2State(t, sharedPath); exp != nMatches {
		t.Fatalf("v2 initiale : expérience %d, attendu %d", exp, nMatches)
	}

	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	seedExcludedFlag(t, player.SQLDb(), excludedID)
	if err := player.Close(); err != nil {
		t.Fatalf("close player: %v", err)
	}

	provider, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", provider)
	if err := rec.RecomputeAfterExclusion(context.Background(), excludedID); err != nil {
		_ = provider.Close()
		t.Fatalf("RecomputeAfterExclusion: %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("close provider: %v", err)
	}

	// Référence : la même histoire sans le match exclu (jamais présent dans la base).
	refDir := t.TempDir()
	refShared := filepath.Join(refDir, "shared_matches_v2.duckdb")
	refPlayer := filepath.Join(refDir, "player", "stats.duckdb")
	seedLUSRShared(t, refShared, nMatches)
	dropMatchFromShared(t, refShared, excludedID)
	if n := replayFullLUSRV2(t, refShared, refPlayer); n != nMatches-1 {
		t.Fatalf("référence : %d matchs traités, attendu %d", n, nMatches-1)
	}

	// 1. La note servie est la v2, pas une ligne plus récente d'une autre origine : le graphe d'évolution
	// (match_skill_rank_latest_by_type, arbitré sur written_at) sert pour « LUSR » la valeur du slot d'audit
	// « LUSR_V2 » écrit dans la même transaction, et la vue lue par l'interface la même.
	audit := lusrServed(t, playerPath, "match_skill_rank_latest_by_type", "LUSR_V2")
	if len(audit) != nMatches {
		t.Fatalf("slot d'audit LUSR_V2 : %d matchs, attendu %d", len(audit), nMatches)
	}
	for _, view := range []string{"match_skill_rank_latest_by_type", "match_skill_rank_latest"} {
		served := lusrServed(t, playerPath, view, "LUSR")
		for id, want := range audit {
			got, ok := served[id]
			if !ok {
				t.Errorf("%s, match %s : pas de ligne LUSR servie", view, id)
				continue
			}
			if math.Abs(got-want) > 1e-9 {
				t.Errorf("%s, match %s : LUSR servi %.6f, la v2 a écrit %.6f (une ligne d'une autre origine masque la v2)",
					view, id, got, want)
			}
		}
	}

	// 2. Le match exclu n'entre pas dans la note : l'état v2 final est celui de la référence.
	mu, exp := lusrV2State(t, sharedPath)
	refMu, refExp := lusrV2State(t, refShared)
	if exp != refExp {
		t.Errorf("expérience v2 = %d, attendu %d (le match exclu ne doit pas compter)", exp, refExp)
	}
	if math.Abs(mu-refMu) > 1e-9 {
		t.Errorf("mu v2 = %.9f, la référence sans le match exclu donne %.9f", mu, refMu)
	}
}

// dropMatchFromShared retire un match (registre et participants) d'une base partagée fermée.
func dropMatchFromShared(t *testing.T, sharedPath, matchID string) {
	t.Helper()
	h, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer h.Close()
	mustExec(t, h.SQLDb(), `DELETE FROM match_participants WHERE match_id = ?`, matchID)
	mustExec(t, h.SQLDb(), `DELETE FROM match_registry WHERE match_id = ?`, matchID)
}
