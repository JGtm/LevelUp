// Package duckdb — tactical_repo_perimetre_test.go : LE PERIMETRE des lectures
// tactiques (phase 4 bis, 2026-09-06).
//
// Ce que ces tests verrouillent, et pourquoi chacun peut echouer :
//
//  1. la LISTE BLANCHE restreint reellement l'univers — sans elle, la barre de
//     filtres de l'onglet n'aurait aucun effet sur les chiffres ;
//  2. une liste VIDE rend AUCUN match, jamais « tous » : c'est l'etat normal d'un
//     filtre qui ne retient rien, et le confondre avec « pas de restriction »
//     servirait l'historique entier a qui vient d'en demander une tranche vide ;
//  3. la COMPOSITION exige que TOUS les coequipiers aient ete DANS MON EQUIPE — un
//     match ou l'un d'eux jouait EN FACE est exclu (sinon l'axe « escouade » peindrait
//     les points d'un adversaire du jour) ;
//  4. carte optionnelle ET liste blanche s'appliquent ENSEMBLE — la neutralisation du
//     predicat de carte ne doit pas emporter le reste du perimetre ;
//  5. les trois lectures partagent le MEME perimetre (grille, positions, journal).
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// seedTacticalRetournement ajoute m5 : carte A, MOI et l'AMI dans des equipes
// OPPOSEES. C'est le cas que la contrainte de composition doit ecarter — un
// coequipier habituel qui, ce jour-la, jouait en face.
func seedTacticalRetournement(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	base := time.Date(2026, 8, 1, 20, 0, 0, 0, time.UTC)
	tacMatch(t, pdb, "m5", tacCarteA, base)
	tacParticipant(t, pdb, "m5", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "m5", tacXUIDAmi, 1, domain.OutcomeLoss)
	tacKill(t, pdb, "m5", tacXUIDMoi, tacXUIDAmi, 1000, true)
	tacPos(t, pdb, "m5", tacXUIDMoi, 1000, 3.0, 3.0, 5.0, 5.0)
}

// TestTacticalRepo_ListeBlanche_RestreintUnivers : la liste blanche est LE
// perimetre. m1 et m2 sont tous deux sur la carte A ; ne poser que m2 doit laisser
// m1 dehors, avec ses deux points.
func TestTacticalRepo_ListeBlanche_RestreintUnivers(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	q := tacQuery(tacCarteA)
	q.Matchs = domain.RestreindreAux([]string{"m2"})

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m2"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v", matchIDs(got.Univers.Matchs), want)
	}
	if len(got.Points) != 0 {
		t.Errorf("points = %+v, want 0 (m1 est hors liste blanche)", got.Points)
	}
}

// TestTacticalRepo_ListeBlanche_HorsUnivers_Ignore : un match_id qui n'est pas
// celui du joueur (ou pas sur la carte) ne fait PAS entrer le match. La liste
// blanche RESTREINT, elle n'ouvre rien — sinon un identifiant devine dans une URL
// donnerait a lire les matchs d'un autre.
func TestTacticalRepo_ListeBlanche_HorsUnivers_Ignore(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	q := tacQuery(tacCarteA)
	// m4 : carte A mais joue par un TIERS. m3 : le mien, mais sur la carte B.
	q.Matchs = domain.RestreindreAux([]string{"m1", "m3", "m4"})

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m1"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (m3 est une autre carte, m4 n'est pas a moi)",
			matchIDs(got.Univers.Matchs), want)
	}
}

// TestTacticalRepo_ListeBlancheVide_AucunMatch : LE test du lot. Une liste vide
// rend zero match sur les TROIS lectures — jamais l'historique entier.
func TestTacticalRepo_ListeBlancheVide_AucunMatch(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	repo := NewTacticalRepo(pdb)
	ctx := context.Background()

	q := tacQuery(tacCarteA)
	q.Matchs = domain.RestreindreAux(nil)

	pos, err := repo.KillPositions(ctx, q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(pos.Univers.Matchs) != 0 || len(pos.Points) != 0 {
		t.Errorf("KillPositions = %d matchs / %d points, want 0/0 — une liste vide ne veut pas dire « tous »",
			len(pos.Univers.Matchs), len(pos.Points))
	}

	ev, err := repo.KillEvents(ctx, q)
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if len(ev.Univers.Matchs) != 0 || len(ev.Events) != 0 {
		t.Errorf("KillEvents = %d matchs / %d evenements, want 0/0",
			len(ev.Univers.Matchs), len(ev.Events))
	}

	cartes, err := repo.MapsPlayed(ctx, domain.TacticalQuery{
		PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(nil),
	})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(cartes) != 0 {
		t.Errorf("MapsPlayed = %+v, want aucune carte", cartes)
	}
}

// TestTacticalRepo_SansListeBlanche_ToutLHistorique : le zero-value du perimetre ne
// restreint RIEN — c'est ce dont la page Escouade depend (elle lit le journal des
// morts sur tout l'historique, puis resserre en Go). Le pendant exact du test
// precedent : les deux etats doivent rester distincts.
func TestTacticalRepo_SansListeBlanche_ToutLHistorique(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if want := []string{"m1", "m2", "m3"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (aucune liste blanche posee)", matchIDs(got.Univers.Matchs), want)
	}
}

// TestTacticalRepo_Composition_ExigeMonEquipe : un match ou le coequipier demande
// jouait EN FACE (m5) est exclu, comme celui ou il n'etait pas la (m2). Seul m1 —
// ou il est de mon cote — reste.
func TestTacticalRepo_Composition_ExigeMonEquipe(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	seedTacticalRetournement(t, pdb)

	q := tacQuery(tacCarteA)
	q.Matchs = domain.RestreindreAux([]string{"m1", "m2", "m5"})
	q.Coequipiers = []string{tacXUIDAmi}

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m1"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (m2 : l'ami est absent ; m5 : il est EN FACE)",
			matchIDs(got.Univers.Matchs), want)
	}
}

// TestTacticalRepo_Composition_TousExiges : avec DEUX coequipiers, un match ou un
// seul des deux est present ne compte pas. La contrainte est un ET, pas un OU —
// « cette composition » n'est pas « au moins l'un d'eux ».
func TestTacticalRepo_Composition_TousExiges(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	q := tacQuery(tacCarteA)
	q.Coequipiers = []string{tacXUIDAmi, tacXUIDTier} // le tiers n'a joue que m4

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(got.Univers.Matchs) != 0 {
		t.Errorf("univers = %v, want vide (m1 porte l'ami mais pas le tiers)",
			matchIDs(got.Univers.Matchs))
	}
}

// TestTacticalRepo_SansCarte_ListeBlancheEtCompositionAppliquees : le predicat de
// carte neutralise (page Escouade, journal des morts) ne desactive PAS le reste du
// perimetre. Les deux cartes sont dans la liste blanche ; la composition ne laisse
// passer que m1.
func TestTacticalRepo_SansCarte_ListeBlancheEtCompositionAppliquees(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	q := tacQuery("")
	q.Matchs = domain.RestreindreAux([]string{"m1", "m2", "m3"})

	repo := NewTacticalRepo(pdb)
	got, err := repo.KillEvents(context.Background(), q)
	if err != nil {
		t.Fatalf("KillEvents sans carte: %v", err)
	}
	if want := []string{"m1", "m2", "m3"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (la carte est neutre, la liste blanche non)",
			matchIDs(got.Univers.Matchs), want)
	}

	q.Coequipiers = []string{tacXUIDAmi}
	got, err = repo.KillEvents(context.Background(), q)
	if err != nil {
		t.Fatalf("KillEvents sans carte, avec composition: %v", err)
	}
	if want := []string{"m1"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (seul m1 porte l'ami dans mon equipe)",
			matchIDs(got.Univers.Matchs), want)
	}
}

// TestTacticalRepo_MapsPlayed_Perimetre : la grille d'entree lit le MEME perimetre
// que les rasters — sans quoi une carte proposee a l'ecran pourrait n'avoir aucun
// match une fois ouverte.
func TestTacticalRepo_MapsPlayed_Perimetre(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(), domain.TacticalQuery{
		PlayerXUID: tacXUIDMoi,
		Matchs:     domain.RestreindreAux([]string{"m2", "m3"}),
	})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("cartes = %+v, want les deux cartes (m2 sur A, m3 sur B)", rows)
	}
	for _, r := range rows {
		if r.Matchs != 1 {
			t.Errorf("carte %s = %d matchs, want 1 (la liste blanche n'en pose qu'un par carte)",
				r.MapID, r.Matchs)
		}
	}
}

// TestTacticalRepo_MapsPlayed_Composition : la composition resserre AUSSI la
// grille. Sans l'ami, la carte B (ou il n'a pas joue) disparait.
func TestTacticalRepo_MapsPlayed_Composition(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(), domain.TacticalQuery{
		PlayerXUID:  tacXUIDMoi,
		Coequipiers: []string{tacXUIDAmi},
	})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 1 || rows[0].MapID != tacCarteA || rows[0].Matchs != 1 {
		t.Fatalf("cartes = %+v, want la seule carte A a 1 match (m1)", rows)
	}
}
