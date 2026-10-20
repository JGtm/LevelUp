// Package duckdb — tactical_repo_cartes_variantes_test.go : la grille d'entree de l'onglet
// Tactique rend UNE rangee par carte, quel que soit le nom que le registre porte match par
// match (vrai nom, NULL, ou map_id recopie par une sync sans traduction).
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// tacMatchNomme pose un match au registre avec un map_name EXPLICITE (nil = NULL).
func tacMatchNomme(t *testing.T, pdb *PlayerDB, matchID, mapID string, mapName any, start time.Time) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_registry
		(match_id, map_id, map_name, start_time, start_time_utc, playlist_name, pair_name)
		VALUES (?, ?, ?, ?, ?, 'Ranked Arena', 'Arena:Slayer')`,
		matchID, mapID, mapName, start, start)
}

// TestTacticalRepo_MapsPlayed_VariantesDeNomFusionnees : trois variantes du nom d'une meme
// carte (vrai nom, NULL, map_id recopie) donnent UNE rangee dont le compte est la somme, le
// nom celui du match nomme le plus recent, jamais le map_id.
func TestTacticalRepo_MapsPlayed_VariantesDeNomFusionnees(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	// Ancien vrai nom, puis NULL, puis map_id recopie (le plus recent), puis le vrai nom actuel.
	tacMatchNomme(t, pdb, "v1", tacCarteB, "Recharge (ancien)", base)
	tacMatchNomme(t, pdb, "v2", tacCarteB, nil, base.Add(time.Hour))
	tacMatchNomme(t, pdb, "v3", tacCarteB, "Recharge", base.Add(2*time.Hour))
	tacMatchNomme(t, pdb, "v4", tacCarteB, tacCarteB, base.Add(3*time.Hour))
	for i, id := range []string{"v1", "v2", "v3", "v4"} {
		outcome := domain.OutcomeWin
		if i%2 == 1 {
			outcome = domain.OutcomeLoss
		}
		tacParticipant(t, pdb, id, tacXUIDMoi, 0, outcome)
	}

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(),
		tacQuery(""))
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("cartes = %d, want 1 (une carte, quatre variantes de nom) : %+v", len(rows), rows)
	}
	got := rows[0]
	if got.MapID != tacCarteB || got.Matchs != 4 || got.Victoires != 2 || got.Defaites != 2 {
		t.Errorf("rangee = %+v, want %s a 4 matchs, 2 V / 2 D (compte fusionne)", got, tacCarteB)
	}
	if got.MapName != "Recharge" {
		t.Errorf("nom = %q, want %q (vrai nom du match nomme le plus recent, ni NULL ni map_id)",
			got.MapName, "Recharge")
	}
}

// TestTacticalRepo_MapsPlayed_SansVraiNom_LibelleDeLaTraduction : une carte dont AUCUN match
// ne porte de vrai nom au registre (NULL et map_id recopie) n'expose jamais le map_id comme
// nom : le libelle vient de la traduction de l'asset.
func TestTacticalRepo_MapsPlayed_SansVraiNom_LibelleDeLaTraduction(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatchNomme(t, pdb, "w1", tacCarteA, nil, base)
	tacMatchNomme(t, pdb, "w2", tacCarteA, tacCarteA, base.Add(time.Hour))
	tacParticipant(t, pdb, "w1", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "w2", tacXUIDMoi, 0, domain.OutcomeWin)

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(),
		tacQuery(""))
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 1 || rows[0].Matchs != 2 {
		t.Fatalf("rangees = %+v, want 1 carte a 2 matchs", rows)
	}
	if rows[0].MapName != "" {
		t.Errorf("nom = %q, want vide (ni NULL ni map_id ne sont un nom)", rows[0].MapName)
	}
	if rows[0].MapNameFR != "Les Rues" {
		t.Errorf("libelle = %q, want %q (traduction de l'asset)", rows[0].MapNameFR, "Les Rues")
	}
}
