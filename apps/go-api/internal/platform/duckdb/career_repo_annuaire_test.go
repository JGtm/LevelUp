//go:build integration

// Package duckdb — career_repo_annuaire_test.go : LA PARITÉ DES NOMS du lot perf L7.
//
// Q26 (GetTopEncountersGlobal) ne joint plus v_gamertag_lookup : ses noms viennent de l'annuaire
// de la lecture (squad_repo_annuaire.go), sur les matchs de l'historique du joueur. La
// référence est l'ANCIENNE expression des lecteurs sur la VRAIE vue canonique
// (nomSelonLaVue, squad_repo_annuaire_test.go), plus le nom de chaque niveau écrit en clair.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/observability/timing"
)

// TestCareerRepo_Annuaire_Q26MemeNomsQueLaVue : les huit joueurs croisés sur ma1 / ma2 (un par
// niveau de la cascade, alliés et ennemi) portent le nom que la jointure leur donnait.
func TestCareerRepo_Annuaire_Q26MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	got, _, err := NewCareerRepo(pdb).GetTopEncountersGlobal(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetTopEncountersGlobal : %v", err)
	}
	// Croisés non-bots sur ma1 + ma2 : x_alias, x_alias_vide, x_part, x_triple, x_kfl, x_ennemi,
	// x_kvp, x_rien (Q26 écarte les bots).
	if len(got) != 8 {
		t.Fatalf("%d rencontres, attendu 8 : %+v", len(got), got)
	}
	for _, e := range got {
		verifierNom(t, pdb, "Q26", e.XUID, e.Gamertag)
	}
}

// TestCareerRepo_Annuaire_EcartNomme_NomHorsHistorique : l'ÉCART ASSUMÉ (D7.1 du plan perf).
// L'annuaire lit les participants sur les matchs de l'historique du joueur, la vue sur toute la
// base : un croisé SANS alias, sans nom dans les matchs du joueur mais nommé AILLEURS, reçoit le
// libellé masqué là où la jointure rendait ce nom. Copie de production du 2026-09-23 : 8 cas sur
// 58 353 joueurs croisés ou affrontés par les cinq joueurs suivis (tous chez Nuzzles, dont les
// participants n'ont pas de gamertag), aucun dans une ligne servie.
func TestCareerRepo_Annuaire_EcartNomme_NomHorsHistorique(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	ctx := context.Background()
	participant := `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, 0)`
	execOnSharedDBs(t, pdb, ctx, participant, "ma1", "x_ailleurs", "")
	execOnSharedDBs(t, pdb, ctx, participant, "mb1", "x_ailleurs", "NomAilleurs") // match d'un autre joueur
	got, _, err := NewCareerRepo(pdb).GetTopEncountersGlobal(ctx, nil)
	if err != nil {
		t.Fatalf("GetTopEncountersGlobal : %v", err)
	}
	for _, e := range got {
		if e.XUID != "x_ailleurs" {
			continue
		}
		if e.Gamertag != "Joueur eurs" {
			t.Errorf("x_ailleurs nommé %q, attendu le libellé masqué « Joueur eurs »", e.Gamertag)
		}
		if vue := nomSelonLaVue(t, pdb, e.XUID); vue != "NomAilleurs" {
			t.Errorf("la jointure rendait %q ; l'écart documenté suppose « NomAilleurs »", vue)
		}
		return
	}
	t.Fatal("x_ailleurs absent des rencontres")
}

// TestCareerRepo_Annuaire_SectionsDeDuree : D7.4 — la lecture et son annuaire ont chacun leur
// section (des feuilles, cf. observability/timing).
func TestCareerRepo_Annuaire_SectionsDeDuree(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	ctx, chrono := timing.WithTimings(context.Background())
	if _, _, err := NewCareerRepo(pdb).GetTopEncountersGlobal(ctx, nil); err != nil {
		t.Fatalf("GetTopEncountersGlobal : %v", err)
	}
	appels := map[string]int{}
	for _, s := range chrono.Snapshot() {
		appels[s.Name] = s.Calls
	}
	for nom, n := range map[string]int{"top_encounters": 1, "top_encounters_annuaire": 1} {
		if appels[nom] != n {
			t.Errorf("section %q : %d appel(s), attendu %d (sections : %v)", nom, appels[nom], n, appels)
		}
	}
}
