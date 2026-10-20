// Package service — tactical_service_grille_test.go : LE PAS DE LA GRILLE, vu de la
// lecture qui le publie (lot 3.2, decision D6).
//
// Les tests du choix lui-meme vivent dans le paquet pur
// (analysis/tactical/pas_adaptatif_test.go) ; ceux-ci verifient ce que le SERVICE en fait :
// il publie le pas retenu, il ne bouge pas quand 0,5 m suffit, et il resout un detail de
// cellule DANS LE MEME REPERE que la lecture d'ou vient le clic.
package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// tsCarteClairsemee monte une carte a la forme d'Illusion apres le backfill des positions :
// les matchs sont la, mais dans chaque zone les trois morts tombent a 50 cm les unes des
// autres — trois cellules de 0,5 m d'UN SEUL match chacune. Aucune cellule ne passe le
// plancher avant le pas de 2 m, ou les trois se rejoignent.
func tsCarteClairsemee(zones int) *mockTacticalRepo {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	matchs := []string{"m1", "m2", "m3"}
	for _, id := range matchs {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
	}
	decalages := []float64{0.1, 0.6, 1.1}
	for z := 0; z < zones; z++ {
		for i, id := range matchs {
			repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
				MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
				KillerX: 100, KillerY: 100,
				VictimX: 4*float64(z) + decalages[i], VictimY: 0.1,
			})
		}
	}
	return repo
}

// TestTacticalService_PasAdaptatif_CarteClairsemee : le defaut du point 21. Avant ce lot, la
// lecture rendait 0 cellule au pas fixe de 0,5 m et la page annoncait « pas assez de matchs
// mesures » alors que les trois matchs etaient bien retenus. Le pas doit desormais grossir,
// et etre PUBLIE.
func TestTacticalService_PasAdaptatif_CarteClairsemee(t *testing.T) {
	repo := tsCarteClairsemee(tactical.CellulesLisiblesMin)
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	lecture, err := svc.Raster(context.Background(),
		tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if lecture.MatchsRetenus != 3 {
		t.Fatalf("les trois matchs sont mesures et retenus, recu %d", lecture.MatchsRetenus)
	}
	if lecture.PasM != 2 {
		t.Fatalf("pas publie = %v, attendu 2 m (la densite ne suffit ni a 0,5 ni a 1 m)", lecture.PasM)
	}
	if len(lecture.Cellules) < tactical.CellulesLisiblesMin {
		t.Fatalf("le plan doit porter au moins %d cellules, recu %d",
			tactical.CellulesLisiblesMin, len(lecture.Cellules))
	}
	if !lecture.Bornes.Valide {
		t.Fatalf("des cellules peintes exigent un cadre valide : %+v", lecture.Bornes)
	}
	// LE PLANCHER N'A PAS BAISSE : chaque cellule publiee compte bien ses trois matchs.
	for _, c := range lecture.Cellules {
		if c.Matchs < tactical.PlancherMatchsParCellule {
			t.Fatalf("cellule (%d,%d) publiee avec %d matchs distincts, plancher %d",
				c.Col, c.Lig, c.Matchs, tactical.PlancherMatchsParCellule)
		}
	}
}

// TestTacticalService_PasAdaptatif_CarteDejaLisible : l'invariant du lot cote service. Une
// carte lisible a 0,5 m rend EXACTEMENT le meme plan qu'avant — memes cellules, meme pas,
// memes bornes. Le pas adaptatif ne coute rien a ceux qui n'en ont pas besoin.
func TestTacticalService_PasAdaptatif_CarteDejaLisible(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	matchs := []string{"m1", "m2", "m3"}
	for _, id := range matchs {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
	}
	// Chaque zone : les trois matchs au MEME quart de metre carre — lisible des 0,5 m.
	for z := 0; z < tactical.CellulesLisiblesMin; z++ {
		for _, id := range matchs {
			repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
				MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
				KillerX: 100, KillerY: 100,
				VictimX: 2*float64(z) + 0.1, VictimY: 0.1,
			})
		}
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	lecture, err := svc.Raster(context.Background(),
		tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if lecture.PasM != tactical.PasParDefautM {
		t.Fatalf("pas publie = %v, attendu le pas par defaut %v", lecture.PasM, tactical.PasParDefautM)
	}
	if len(lecture.Cellules) != tactical.CellulesLisiblesMin {
		t.Fatalf("attendu %d cellules a 0,5 m, recu %d", tactical.CellulesLisiblesMin, len(lecture.Cellules))
	}
	for _, c := range lecture.Cellules {
		if c.Col%4 != 0 || c.Lig != 0 {
			t.Fatalf("cellule (%d,%d) : adresse d'une grille de 0,5 m attendue", c.Col, c.Lig)
		}
	}
}

// TestTacticalService_Cellule_SuitLePasDeLaLecture : le clic doit se resoudre au pas de la
// lecture d'ou il vient. Sans le pas dans la demande, la cellule (2, 0) d'une grille de 2 m
// etait cherchee sur la grille de 0,5 m — la ou elle designe un tout autre metre carre, et
// ou elle ne rend AUCUNE contribution.
func TestTacticalService_Cellule_SuitLePasDeLaLecture(t *testing.T) {
	repo := tsCarteClairsemee(tactical.CellulesLisiblesMin)
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	ctx := context.Background()

	lecture, err := svc.Raster(ctx, tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	visee := lecture.Cellules[0]

	detail, err := svc.Cellule(ctx, domain.TacticalCelluleRequest{
		MapID:    tsCarte,
		Question: domain.TacticalQuestionMorts,
		Qui:      domain.TacticalQuiMoi,
		Scope:    domain.TacticalScope{MatchIDs: tsPerimetreDu(repo)},
		Col:      visee.Col, Lig: visee.Lig,
		PasM: lecture.PasM,
	})
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	// La cellule de 2 m rassemble les trois morts de la zone, une par match.
	if len(detail.Contributions) != 3 {
		t.Fatalf("attendu les 3 morts de la cellule (%d,%d) au pas de %v m, recu %d : %+v",
			visee.Col, visee.Lig, lecture.PasM, len(detail.Contributions), detail.Contributions)
	}
}

// TestTacticalService_Cellule_PasAbsentVautPasParDefaut : un client d'une version anterieure
// n'envoie pas de pas. Il doit continuer a lire la grille de 0,5 m, jamais une erreur.
func TestTacticalService_Cellule_PasAbsentVautPasParDefaut(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
			KillerX: 100, KillerY: 100, VictimX: 10.0, VictimY: 10.0,
		})
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	detail, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID:    tsCarte,
		Question: domain.TacticalQuestionMorts,
		Qui:      domain.TacticalQuiMoi,
		Scope:    domain.TacticalScope{MatchIDs: tsPerimetreDu(repo)},
		Col:      20, Lig: 20, // (10,10) sur une grille de 0,5 m
	})
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(detail.Contributions) != 3 {
		t.Fatalf("attendu 3 contributions sur la grille par defaut, recu %d", len(detail.Contributions))
	}
}
