package duckdb

// career_repo_fenetres_test.go — LES LECTURES D'HISTORIQUE COMPLET SONT PAYÉES AUX MATCHS DU
// JOUEUR (lot B du plan perf « compaction et périmètre joueur », 2026-09-27).
//
// Ce que ces tests verrouillent, et pourquoi chacun peut échouer :
//
//  1. Carrière rencontres (Q26), rivaux (Q27), Relations (Q28 sur l'historique) et Tactique
//     « morts par carte » sans liste blanche lisent la liste des matchs du joueur
//     (QMatchsDuJoueurTpl) et la lient sous CHAQUE fenêtre `_latest` : rouge si une fenêtre
//     voit une ligne d'un match où le joueur n'a pas joué (cf. fenetres_perimetre_helpers_test.go
//     — le défaut ne change aucun chiffre, seul le nombre de lignes vues par la fenêtre le montre) ;
//  2. les rivaux sont lus UNE fois pour les deux classements (rouge si la page relit l'agrégat) ;
//  3. la liste exclut la Campagne (Halo 5) : un duel d'un match de Campagne n'entre ni dans les
//     rivaux ni dans les frags échangés des rencontres.
// Invariant I2 de l'ADR 0036 (docs/adr/0036-page-reads-are-scoped.md).

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// matchsDuJoueurFenetres / matchsDesAutresFenetres : le corpus — des matchs de MOI, et des matchs
// où je n'ai pas joué (deux tiers entre eux), qui ne doivent entrer dans aucune fenêtre.
const (
	matchsDuJoueurFenetres  = 4
	matchsDesAutresFenetres = 6
)

// seedMatchsDesAutres monte `n` matchs de deux tiers (ami contre tiers), chacun avec
// mortsParMatchFenetres morts, leurs positions et leur contexte.
func seedMatchsDesAutres(t *testing.T, pdb *PlayerDB, n int) {
	t.Helper()
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("g%02d", i)
		tacMatch(t, pdb, id, tacCarteA, base.Add(time.Duration(i)*time.Hour))
		tacParticipant(t, pdb, id, tacXUIDAmi, 0, domain.OutcomeWin)
		tacParticipant(t, pdb, id, tacXUIDTier, 1, domain.OutcomeLoss)
		for k := 0; k < mortsParMatchFenetres; k++ {
			ts := 1000 * (k + 1)
			tacKill(t, pdb, id, tacXUIDAmi, tacXUIDTier, ts, true)
			tacPos(t, pdb, id, tacXUIDAmi, ts, 1.0, 1.0, 2.0, 2.0)
			tacContexte(t, pdb, id, tacXUIDTier, ts, 4.0)
		}
	}
}

// TestLecturesHistorique_FenetresBorneesAuxMatchsDuJoueur : quatre matchs à moi sur dix ; chaque
// fenêtre de chaque requête des quatre lectures ne voit que les lignes de ces quatre-là.
func TestLecturesHistorique_FenetresBorneesAuxMatchsDuJoueur(t *testing.T) {
	b := newBaseNotee(t)
	seedFenetresTactiques(t, b.pdb, matchsDuJoueurFenetres)
	seedMatchsDesAutres(t, b.pdb, matchsDesAutresFenetres)
	ctx := context.Background()
	repo := NewCareerRepo(b.pdb)
	borne := matchsDuJoueurFenetres * mortsParMatchFenetres
	b.carnet.vider()

	enc, _, err := repo.GetTopEncountersGlobal(ctx, nil)
	if err != nil {
		t.Fatalf("GetTopEncountersGlobal: %v", err)
	}
	if len(enc) != 1 || enc[0].XUID != tacXUIDAdv || enc[0].KillsDealt == nil || *enc[0].KillsDealt != 8 {
		t.Fatalf("rencontres = %+v, want l'adversaire seul, 8 frags infligés", enc)
	}
	exigerFenetresBornees(t, b, "GetTopEncountersGlobal", borne, 1)

	nem, vic, err := repo.GetRivals(ctx)
	if err != nil {
		t.Fatalf("GetRivals: %v", err)
	}
	if len(nem) != 1 || nem[0].Deaths != 4 || len(vic) != 1 || vic[0].Frags != 8 {
		t.Fatalf("rivaux = %+v / %+v, want l'adversaire, 4 morts subies et 8 frags", nem, vic)
	}
	exigerUneLectureDesRivaux(t, b)
	exigerFenetresBornees(t, b, "GetRivals", borne, 1)

	rel, err := repo.GetRelations(ctx, nil)
	if err != nil {
		t.Fatalf("GetRelations: %v", err)
	}
	if len(rel) != 1 || rel[0].KillsDealt != 8 || rel[0].DeathsSuffered != 4 {
		t.Fatalf("relations = %+v, want l'adversaire, 8 frags / 4 morts", rel)
	}
	exigerFenetresBornees(t, b, "GetRelations", borne, 1)

	// Le périmètre de filtres de la page (deux de mes matchs) : même requête, autre liste.
	rel, err = repo.GetRelations(ctx, []string{"f01", "f02"})
	if err != nil {
		t.Fatalf("GetRelations(scopé): %v", err)
	}
	if len(rel) != 1 || rel[0].KillsDealt != 4 || rel[0].DeathsSuffered != 2 {
		t.Fatalf("relations scopées = %+v, want l'adversaire, 4 frags / 2 morts", rel)
	}
	exigerFenetresBornees(t, b, "GetRelations (scopé)", 2*mortsParMatchFenetres, 1)

	parCarte, err := NewTacticalRepo(b.pdb).MortsParCarte(ctx, domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MortsParCarte: %v", err)
	}
	if len(parCarte[tacCarteA]) != matchsDuJoueurFenetres {
		t.Fatalf("MortsParCarte = %+v, want une mort à moi par match", parCarte)
	}
	exigerFenetresBornees(t, b, "MortsParCarte (sans liste blanche)", borne, 1)
}

// exigerUneLectureDesRivaux : l'agrégat Q27 (reconnu à sa colonne `match_rencontre`) est lu une
// seule fois par GetRivals ; les requêtes notées sont remises au carnet pour la suite du test.
func exigerUneLectureDesRivaux(t *testing.T, b baseNotee) {
	t.Helper()
	requetes := b.carnet.vider()
	lectures := 0
	for _, r := range requetes {
		if strings.Contains(r.sql, "AS match_rencontre") {
			lectures++
		}
		b.carnet.noter(r.sql, r.args)
	}
	if lectures != 1 {
		t.Fatalf("GetRivals a lu l'agrégat Q27 %d fois, want 1 (les deux classements se trient en Go)", lectures)
	}
}

// TestLecturesHistorique_CampagneHorsDeLaListe : un joueur Halo 5 a un duel dans un match de
// Campagne et un autre dans un match ordinaire. La liste des matchs du joueur exclut la Campagne :
// seul le duel ordinaire compte, dans les rivaux comme dans les frags échangés des rencontres.
func TestLecturesHistorique_CampagneHorsDeLaListe(t *testing.T) {
	b := newBaseNotee(t)
	campagne := analysis.CampaignExcludedVariantIDs("halo_5")
	if len(campagne) == 0 {
		t.Fatal("aucun mode Campagne connu pour halo_5 : le test ne mesurerait rien")
	}
	b.pdb.TitleSlug = "halo_5"
	debut := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatchVariant(t, b.pdb, "c1", tacCarteA, debut, campagne[0])
	tacMatch(t, b.pdb, "n1", tacCarteA, debut.Add(time.Hour))
	for _, id := range []string{"c1", "n1"} {
		tacParticipant(t, b.pdb, id, tacXUIDMoi, 0, domain.OutcomeWin)
		tacParticipant(t, b.pdb, id, tacXUIDAdv, 1, domain.OutcomeLoss)
		tacKill(t, b.pdb, id, tacXUIDMoi, tacXUIDAdv, 1000, true)
	}
	ctx := context.Background()
	repo := NewCareerRepo(b.pdb)

	_, vic, err := repo.GetRivals(ctx)
	if err != nil {
		t.Fatalf("GetRivals: %v", err)
	}
	if len(vic) != 1 || vic[0].Frags != 1 || vic[0].MatchCount != 1 {
		t.Fatalf("souffre-douleur = %+v, want l'adversaire, 1 frag sur 1 match (Campagne exclue)", vic)
	}
	enc, _, err := repo.GetTopEncountersGlobal(ctx, nil)
	if err != nil {
		t.Fatalf("GetTopEncountersGlobal: %v", err)
	}
	if len(enc) != 1 || enc[0].KillsDealt == nil || *enc[0].KillsDealt != 1 || enc[0].CountTogether != 1 {
		t.Fatalf("rencontres = %+v, want l'adversaire, 1 match, 1 frag infligé (Campagne exclue)", enc)
	}
}
