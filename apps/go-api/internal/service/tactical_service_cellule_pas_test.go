package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// Les sidecars sont cuits a 0,5 m une fois pour toutes ; une lecture « temps passe » ou
// « routes » a 1 ou 2 m (pas adaptatif, lot 3.2) adresse donc ses clics sur une grille plus
// grosse que celle du sidecar. `celluleVisee` doit readresser chaque cellule fine vers la
// cellule demandee, en s'ancrant sur l'origine du monde — bornes negatives comprises. Ecrit
// apres la revue de la vague 3 (2026-09-10) : jusque-la, ces deux questions n'etaient
// exercees qu'au pas par defaut, ou le readressage est l'identite. Mutation prouvee :
// comparer directement (col, lig) sans readresser fait rougir les deux cas.

func sidecarTemps(entrees []domain.TacticalRasterEntree) *mockRasterStore {
	return &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": {
			SchemaVersion: domain.TacticalRasterSchemaVersion, MatchID: "m1", ShortID: "m1",
			PasM: domain.TacticalRasterPasM, PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
			FrameIntervalMs: 100,
			Joueurs:         []domain.TacticalRasterJoueur{{XUID: tsMoi, PremieresEntrees: entrees}},
		},
	}}
}

func demandeAuPas(repo *mockTacticalRepo, question string, col, lig int, pasM float64) domain.TacticalCelluleRequest {
	req := celluleDemande(repo, question, domain.TacticalQuiMoi, col, lig)
	req.PasM = pasM
	return req
}

// TestCellule_Temps_SuitLePasDeLaLecture : a 2 m, la cellule (1,1) rassemble les cellules
// fines dont le CENTRE tombe dans [2,4) x [2,4) — (4,6) et (7,7) oui, (8,6) non (x = 4,25 m).
func TestCellule_Temps_SuitLePasDeLaLecture(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	store := sidecarTemps([]domain.TacticalRasterEntree{
		{Col: 4, Lig: 6, Frame: 30},
		{Col: 7, Lig: 7, Frame: 50},
		{Col: 8, Lig: 6, Frame: 5}, // cellule voisine a 2 m : ne doit pas apparaitre
	})
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)

	got, err := svc.Cellule(context.Background(), demandeAuPas(repo, domain.TacticalQuestionTemps, 1, 1, 2))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions = %+v, want les deux entrees de la cellule de 2 m (1,1)", got.Contributions)
	}
	if got.Contributions[0].InstantMs != 30*100 || got.Contributions[1].InstantMs != 50*100 {
		t.Errorf("instants = %d, %d ; want 3000 puis 5000", got.Contributions[0].InstantMs, got.Contributions[1].InstantMs)
	}
	// La cellule voisine (2,1) ne recoit que l'entree (8,6).
	voisine, err := svc.Cellule(context.Background(), demandeAuPas(repo, domain.TacticalQuestionTemps, 2, 1, 2))
	if err != nil {
		t.Fatalf("Cellule voisine: %v", err)
	}
	if len(voisine.Contributions) != 1 || voisine.Contributions[0].InstantMs != 5*100 {
		t.Fatalf("voisine = %+v, want la seule entree (8,6)", voisine.Contributions)
	}
}

// TestCellule_Temps_BornesNegativesAuPasGrossier : l'ancrage est l'origine du monde. A 2 m,
// (-1,-1) couvre [-2,0) x [-2,0) : les cellules fines (-1,-1) et (-4,-4) y tombent, (-5,-1)
// non (x = -2,25 m). Une adresse relative aux bornes de la lecture donnerait un autre resultat.
func TestCellule_Temps_BornesNegativesAuPasGrossier(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	store := sidecarTemps([]domain.TacticalRasterEntree{
		{Col: -1, Lig: -1, Frame: 10},
		{Col: -4, Lig: -4, Frame: 20},
		{Col: -5, Lig: -1, Frame: 30},
	})
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)

	got, err := svc.Cellule(context.Background(), demandeAuPas(repo, domain.TacticalQuestionTemps, -1, -1, 2))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions = %+v, want (-1,-1) et (-4,-4) seulement", got.Contributions)
	}
}

// TestCellule_Routes_SuitLePasDeLaLecture : une route qui traverse deux cellules fines de la
// MEME cellule de 2 m contribue UNE fois ; a 2 m la cellule voisine ne la voit pas.
func TestCellule_Routes_SuitLePasDeLaLecture(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": {
			SchemaVersion: domain.TacticalRasterSchemaVersion, MatchID: "m1", ShortID: "m1",
			PasM: domain.TacticalRasterPasM, PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
			FrameIntervalMs: 200,
			Joueurs: []domain.TacticalRasterJoueur{{
				XUID: tsMoi,
				Routes: []domain.TacticalRasterRoute{{
					DebutFrame: 10,
					Cases:      []domain.TacticalRasterCase{{Col: 4, Lig: 6}, {Col: 5, Lig: 6}},
				}},
			}},
		},
	}}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)

	got, err := svc.Cellule(context.Background(), demandeAuPas(repo, domain.TacticalQuestionRoutes, 1, 1, 2))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].InstantMs != 10*200 {
		t.Fatalf("contributions = %+v, want une seule route, au debut de vie", got.Contributions)
	}
	voisine, err := svc.Cellule(context.Background(), demandeAuPas(repo, domain.TacticalQuestionRoutes, 2, 1, 2))
	if err != nil {
		t.Fatalf("Cellule voisine: %v", err)
	}
	if len(voisine.Contributions) != 0 {
		t.Fatalf("voisine = %+v, want aucune contribution", voisine.Contributions)
	}
}
