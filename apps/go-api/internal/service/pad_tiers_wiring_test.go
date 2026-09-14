package service

// pad_tiers_wiring_test.go — LE GARDE-RAIL DE LA LECTURE PRODUIT DES NIVEAUX D'ARMES.
//
// # POURQUOI CE FICHIER EXISTE
//
// La revue des prises nettes a releve le meme defaut trois fois (constats C1/M6/M7) : une
// grandeur PRODUITE, une table REMPLIE, un calcul TESTE — et aucun ecran qui la lit, parce que
// le cablage entre le repo et le bloc manquait sans que rien ne rougisse. Un test de calcul ne
// l'attrape pas : il appelle la fonction directement.
//
// Ces tests-ci passent par le SERVICE, avec un repo qui rend des lignes, et verifient que le
// bloc arrive au contrat. Debrancher `attachPadTiers` (page Sessions) ou `attacherNiveauxDArmes`
// (Escouade / Timeseries) les fait rougir.

import (
	"context"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// usageTestXUID / usageTestMatchID : le joueur et le match du temoin partage
// (`session_page_usage_test.go`).
const (
	usageTestXUID    = "P"
	usageTestMatchID = "m1"
)

// repoUsageTemoin rend le repo temoin de la page Sessions, avec des lignes de niveaux.
func repoUsageTemoin() *mockSessionUsageRepo {
	repo := usageTestRepoMock()
	repo.padTiers = lignesNiveaux(usageTestMatchID)
	return repo
}

// construireBlocUsage fait tourner la page Sessions sur un repo donne et rend le bloc.
func construireBlocUsage(t *testing.T, repo *mockSessionUsageRepo) *domain.SessionUsageBlock {
	t.Helper()
	svc := NewSessionPageService(nil).WithSessionUsage(repo, usageTestXUID, nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSolo, "fr")
	if resp.Usage == nil {
		t.Fatal("bloc usage absent")
	}
	return resp.Usage
}

// blocUsageAvecNiveaux : le bloc de la page Sessions pour un jeu de lignes donne.
func blocUsageAvecNiveaux(t *testing.T, rows []sessionusage.PadTierRow) *domain.SessionUsageBlock {
	t.Helper()
	repo := usageTestRepoMock()
	repo.padTiers = rows
	return construireBlocUsage(t, repo)
}

// lireSource lit un fichier source du paquet (ou d un paquet voisin) pour les garde-rails de
// cablage — la seule facon de prouver qu un APPEL existe, ce qu un test de calcul ne fait pas.
func lireSource(chemin string) (string, error) {
	b, err := os.ReadFile(chemin)
	return string(b), err
}

// lignesNiveaux : une passe minimale mais COMPLETE — un niveau, un zero mesure, et les valeurs
// de match qui disent que la mesure a eu lieu.
func lignesNiveaux(matchID string) []sessionusage.PadTierRow {
	return []sessionusage.PadTierRow{
		{
			MatchID: matchID, XUID: usageTestXUID, Tier: "puissance",
			WeaponFamily: "9d6aaed2", Pickups: 2, PadsConfirmed: 4, PadsTotal: 5,
		},
		{
			MatchID: matchID, XUID: "autre", Tier: "aucune_prise",
			PadsConfirmed: 4, PadsTotal: 5,
		},
	}
}

// TestSessionPage_LitLesNiveauxDArmes — la page Sessions publie le bloc.
func TestSessionPage_LitLesNiveauxDArmes(t *testing.T) {
	block := blocUsageAvecNiveaux(t, lignesNiveaux(usageTestMatchID))
	if block.PadTiers == nil {
		t.Fatal("le bloc des niveaux d'armes n'est pas publie : la lecture produit est debranchee " +
			"(la table se remplirait sans qu'aucun ecran ne la lise)")
	}
	if block.PadTiers.MatchesMeasured != 1 || block.PadTiers.MatchesTiersEstablished != 1 {
		t.Errorf("denominateurs = %d mesures / %d etablis, attendu 1 et 1",
			block.PadTiers.MatchesMeasured, block.PadTiers.MatchesTiersEstablished)
	}
	if len(block.PadTiers.Tiers) != 1 || block.PadTiers.Tiers[0].Tier != "puissance" {
		t.Fatalf("niveaux publies : %+v", block.PadTiers.Tiers)
	}
	if block.PadTiers.Tiers[0].PlayerTotal != 2 {
		t.Errorf("prises du joueur = %v, attendu 2", block.PadTiers.Tiers[0].PlayerTotal)
	}
}

// TestSessionPage_SansLigneAucunBloc — « pas encore mesure » ne se publie pas comme « zero ».
func TestSessionPage_SansLigneAucunBloc(t *testing.T) {
	if block := blocUsageAvecNiveaux(t, nil); block.PadTiers != nil {
		t.Errorf("bloc servi sans aucune ligne : %+v", block.PadTiers)
	}
}

// TestSessionPage_LectureEnEchecNeCassePasLaPage — best-effort, et le reste du bloc est servi.
func TestSessionPage_LectureEnEchecNeCassePasLaPage(t *testing.T) {
	repo := repoUsageTemoin()
	repo.padTiersErr = errTestLectureNiveaux
	block := construireBlocUsage(t, repo)
	if block.PadTiers != nil {
		t.Errorf("bloc servi malgre une lecture en echec : %+v", block.PadTiers)
	}
	if !block.Available {
		t.Error("la page entiere a ete degradee par l'echec d'une seule lecture best-effort")
	}
}

// TestNiveauxDArmes_LeCablageExisteDesDeuxCotes — le second cote (Escouade / Timeseries).
//
// Le bloc d'equipement est produit par `squadagg.BuildEquipmentUsageBlock`, un autre chemin que
// la page Sessions. Il doit lire la MEME table par le MEME calcul.
func TestNiveauxDArmes_LeCablageExisteDesDeuxCotes(t *testing.T) {
	for _, cas := range []struct{ fichier, attendu string }{
		{"session_page_usage.go", "s.attachPadTiers(ctx, &block, ids, tc)"},
		{"session_page_usage.go", "s.sessionUsageRepo.LoadPadTiers(ctx, matchIDs)"},
		{"squadagg/equipment_usage.go", "attacherNiveauxDArmes(ctx, &block, q, tc)"},
		{"squadagg/equipment_usage.go", "q.Repo.LoadPadTiers(ctx, q.MatchIDs)"},
	} {
		src, err := lireSource(cas.fichier)
		if err != nil {
			t.Fatalf("lecture de %s : %v", cas.fichier, err)
		}
		if !strings.Contains(src, cas.attendu) {
			t.Errorf("%s : %q absent — la lecture des niveaux d'armes est debranchee de ce chemin",
				cas.fichier, cas.attendu)
		}
	}
}

// TestNiveauxDArmes_LectureParLaVueLatest — ADR 0026, et le motif de la table l'exige.
//
// Une lecture BRUTE servirait des niveaux calcules sous une reference de cartes PERIMEE a cote
// des niveaux recalcules : le total d'un niveau compterait deux fois la meme prise.
func TestNiveauxDArmes_LectureParLaVueLatest(t *testing.T) {
	src, err := lireSource("../platform/duckdb/session_usage_repo.go")
	if err != nil {
		t.Fatalf("lecture du repo : %v", err)
	}
	if !strings.Contains(src, "FROM match_pad_pickups_by_tier_latest") {
		t.Error("la lecture ne passe pas par la vue _latest")
	}
	if strings.Contains(src, "FROM match_pad_pickups_by_tier\n") ||
		strings.Contains(src, "FROM match_pad_pickups_by_tier ") {
		t.Error("lecture BRUTE de match_pad_pickups_by_tier — elle servirait une passe supplantee")
	}
}

var errTestLectureNiveaux = context.DeadlineExceeded
