//go:build integration

// Package duckdb — career_repo_annuaire_test.go : LA PARITÉ DES NOMS du lot perf L7.
//
// Q26 (GetTopEncountersGlobal) et Q27 (GetRivals) ne joignent plus v_gamertag_lookup : leurs
// noms viennent de l'annuaire de la lecture (squad_repo_annuaire.go), sur les matchs de l'historique
// du joueur. La référence est l'ANCIENNE expression des lecteurs sur la VRAIE vue canonique
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

// seedRivauxAnnuaire : un duel du joueur par niveau de la cascade, dont x_kfseul que SEUL le
// kill-feed connaît (aucune ligne participant, nulle part), affronté dans ma3 : un match du
// joueur où aucun autre xuid sans nom ne joue — seul le match du duel y mène la jambe kill-feed.
func seedRivauxAnnuaire(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry (match_id) VALUES ('ma3')`)
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id)
		VALUES ('ma3', ?, ?, 2, 0)`, pTestXUID, pTestGamertag)
	for _, k := range []struct{ match, tueur, nomTueur, victime, nomVictime string }{
		{"ma1", pTestXUID, pTestGamertag, "x_alias", "NomKFAlias"}, // l'alias gagne
		{"ma1", "x_part", "NomPart", pTestXUID, pTestGamertag},     // participant
		{"ma1", pTestXUID, pTestGamertag, "x_kfl", "NomKFL"},       // kill-feed canonique
		{"ma2", "x_kvp", "", pTestXUID, pTestGamertag},             // table historique seule
		{"ma2", pTestXUID, pTestGamertag, "x_rien", ""},            // libellé masqué
		{"ma3", pTestXUID, pTestGamertag, "x_kfseul", "NomKFSeul"}, // seul le kill-feed le nomme
	} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_kill_events_latest
			(match_id, feed_killer_xuid, feed_killer_gamertag, victim_xuid, victim_gamertag, time_ms)
			VALUES (?, ?, ?, ?, ?, 3000)`, k.match, k.tueur, k.nomTueur, k.victime, k.nomVictime)
	}
}

// TestCareerRepo_Annuaire_Q27MemeNomsQueLaVue : némésis et souffre-douleur portent le nom que la
// jointure leur donnait — y compris x_kfseul, qu'aucune ligne participant ne nomme : la jambe
// kill-feed le trouve dans le match du duel (`match_rencontre`).
func TestCareerRepo_Annuaire_Q27MemeNomsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	seedRivauxAnnuaire(t, pdb)
	nemeses, victimes, err := NewCareerRepo(pdb).GetRivals(context.Background())
	if err != nil {
		t.Fatalf("GetRivals : %v", err)
	}
	noms := map[string]string{}
	for _, r := range append(nemeses, victimes...) {
		verifierNom(t, pdb, "Q27", r.XUID, r.Gamertag)
		noms[r.XUID] = r.Gamertag
	}
	// Adversaires du joueur : x_alias, x_part, x_kfl, x_kvp, x_rien, x_kfseul (les duels de
	// seedAnnuaire, x_kfl et x_triple contre x_ennemi, ne le concernent pas).
	if len(noms) != 6 || len(nemeses) != 6 || len(victimes) != 6 {
		t.Fatalf("adversaires %v (%d némésis, %d souffre-douleur), attendu 6 chacun", noms, len(nemeses), len(victimes))
	}
	if noms["x_kfseul"] != "NomKFSeul" {
		t.Errorf("x_kfseul nommé %q, attendu NomKFSeul (kill-feed du match du duel)", noms["x_kfseul"])
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

// TestCareerRepo_Annuaire_SectionsDeDuree : D7.4 — chaque lecture et son annuaire ont leur
// section (des feuilles, cf. observability/timing) ; Q27 est lue deux fois (némésis puis
// souffre-douleur), son annuaire une seule.
func TestCareerRepo_Annuaire_SectionsDeDuree(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	seedRivauxAnnuaire(t, pdb)
	ctx, chrono := timing.WithTimings(context.Background())
	repo := NewCareerRepo(pdb)
	if _, _, err := repo.GetTopEncountersGlobal(ctx, nil); err != nil {
		t.Fatalf("GetTopEncountersGlobal : %v", err)
	}
	if _, _, err := repo.GetRivals(ctx); err != nil {
		t.Fatalf("GetRivals : %v", err)
	}
	appels := map[string]int{}
	for _, s := range chrono.Snapshot() {
		appels[s.Name] = s.Calls
	}
	for nom, n := range map[string]int{"top_encounters": 1, "top_encounters_annuaire": 1, "rivals": 2, "rivals_annuaire": 1} {
		if appels[nom] != n {
			t.Errorf("section %q : %d appel(s), attendu %d (sections : %v)", nom, appels[nom], n, appels)
		}
	}
}
