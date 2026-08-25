//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// TestGamertagRepo_ResolveGamertags valide la résolution batch xuid → gamertag
// via le chokepoint v_gamertag_lookup (sur SharedReader, vue bare), utilisée par
// l'enrichissement des identités de la timeline d'events (MatchEventsService).
func TestGamertagRepo_ResolveGamertags(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x1", "Alpha")
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x2", "Beta")

	repo := NewGamertagRepo(pdb.SharedReadDB())

	// Dédup (x1 deux fois) + vide ignoré ; xMissing absent du référentiel → exclu.
	got, err := repo.ResolveGamertags(ctx, []string{"x1", "x2", "x1", "", "xMissing"})
	if err != nil {
		t.Fatalf("ResolveGamertags: %v", err)
	}
	if got["x1"] != "Alpha" || got["x2"] != "Beta" {
		t.Errorf("résolution inattendue: %+v", got)
	}
	if _, ok := got["xMissing"]; ok {
		t.Errorf("xuid orphelin ne doit PAS apparaître (masqué au rendu front): %+v", got)
	}

	empty, err := repo.ResolveGamertags(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("xuids vide → map vide sans erreur, got %v err=%v", empty, err)
	}
}

// TestGamertagRepo_ResolveXUIDsByGamertags valide le chemin INVERSE, celui de la
// présence en jeu : la liste `friend_gamertags` des Réglages est saisie À LA MAIN
// (casse arbitraire) et doit retrouver les xuids pour l'appel batch Xbox.
func TestGamertagRepo_ResolveXUIDsByGamertags(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x1", "Alpha")
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x2", "Beta")
	// Bot officiel : jamais un ami, doit rester hors résolution.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "bid(1.2)", "Gamma")

	repo := NewGamertagRepo(pdb.SharedReadDB())

	got, err := repo.ResolveXUIDsByGamertags(ctx,
		[]string{"alpha", "BETA", "Gamma", "JamaisCroise", ""})
	if err != nil {
		t.Fatalf("ResolveXUIDsByGamertags: %v", err)
	}
	// Clés = les gamertags DEMANDÉS (casse d'origine préservée pour l'appelant).
	if got["alpha"] != "x1" {
		t.Errorf("résolution insensible à la casse attendue pour \"alpha\": %+v", got)
	}
	if got["BETA"] != "x2" {
		t.Errorf("résolution insensible à la casse attendue pour \"BETA\": %+v", got)
	}
	if _, ok := got["Gamma"]; ok {
		t.Errorf("un bot ne doit pas être résolu comme ami: %+v", got)
	}
	if _, ok := got["JamaisCroise"]; ok {
		t.Errorf("gamertag inconnu ne doit pas apparaître: %+v", got)
	}

	empty, err := repo.ResolveXUIDsByGamertags(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("gamertags vide → map vide sans erreur, got %v err=%v", empty, err)
	}
}
