package service

// tactical_service_solde_test.go — la lecture « solde » (frags − morts) servie par le service :
// les deux faces de la cible, le plancher sur leur union, le dénominateur des matchs mesurés,
// l'échelle symétrique.

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// soldeCorpus : cinq matchs mesurés. En (2,2), je tue dans m1 et m2 et je meurs dans m3 ; m4 et m5
// n'y ont rien (m5 aucun point du tout : il reste au dénominateur). L'adversaire tombe en (6,6)
// sous mes coups et me tue depuis (14,14).
func soldeCorpus() *mockTacticalRepo {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3", "m4", "m5"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
	}
	for _, id := range []string{"m1", "m2"} {
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{MatchID: id,
			KillerXUID: tsMoi, VictimXUID: tsAdv, KillerX: 2.1, KillerY: 2.1, VictimX: 6.0, VictimY: 6.0})
	}
	repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{MatchID: "m3",
		KillerXUID: tsAdv, VictimXUID: tsMoi, KillerX: 14.0, KillerY: 14.0, VictimX: 2.1, VictimY: 2.1})
	return repo
}

func TestTacticalService_Solde_DeuxFacesPlancherSurLUnion(t *testing.T) {
	repo := soldeCorpus()
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	got, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionSolde, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster(solde): %v", err)
	}
	if len(got.Cellules) != 1 {
		t.Fatalf("cellules = %+v, attendu la seule cellule (2,2) : deux matchs de frags + un de morts", got.Cellules)
	}
	c := celluleEn(got.Cellules, 2.1, 2.1)
	if c == nil {
		t.Fatalf("cellule (2,2) absente : %+v", got.Cellules)
	}
	if c.Frags != 2 || c.Morts != 1 || c.Brut != 1 || c.Matchs != 3 {
		t.Fatalf("cellule = %+v, attendu 2 frags, 1 mort, brut 1, 3 matchs", *c)
	}
	if math.Abs(c.Valeur-0.2) > 1e-12 {
		t.Fatalf("Valeur = %v, attendu 0,2 ((2 − 1) / 5 matchs mesurés)", c.Valeur)
	}
	if !got.Echelle.Symetrique {
		t.Error("solde : l'échelle doit être symétrique (lecture signée)")
	}
	if got.MatchsRetenus != 5 || got.EvenementsLocalises != 3 {
		t.Errorf("retenus / localisés = %d / %d, attendu 5 / 3", got.MatchsRetenus, got.EvenementsLocalises)
	}
	if got.MatchsVictoire != 0 || got.MatchsDefaite != 0 {
		t.Errorf("solde : les côtés victoire / défaite n'ont aucun sens ici : %d / %d", got.MatchsVictoire, got.MatchsDefaite)
	}
}

// TestTacticalService_Solde_AxeAdversaires : la cible change, la règle non — les frags et les
// morts de l'AUTRE camp.
func TestTacticalService_Solde_AxeAdversaires(t *testing.T) {
	repo := soldeCorpus()
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	got, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionSolde, domain.TacticalQuiAdversaires))
	if err != nil {
		t.Fatalf("Raster(solde, adv): %v", err)
	}
	if celluleEn(got.Cellules, 2.1, 2.1) != nil {
		t.Errorf("la cellule de MES frags ne doit pas apparaître sous l'axe adversaires : %+v", got.Cellules)
	}
}

// TestTacticalService_Solde_QuestionAcceptee : « solde » est au vocabulaire servi.
func TestTacticalService_Solde_QuestionAcceptee(t *testing.T) {
	if err := validerLecture(tsCarte, domain.TacticalQuestionSolde, domain.TacticalQuiMoi, nil); err != nil {
		t.Fatalf("solde refusée : %v", err)
	}
}

// TestTacticalService_Solde_DeuxFacesAuJournal : la couverture (compterJournal) et le détail de
// cellule lisent les faces par facesDeLaQuestion — le solde y compte les frags ET les morts.
func TestTacticalService_Solde_DeuxFacesAuJournal(t *testing.T) {
	if victime, tueur := facesDeLaQuestion(domain.TacticalQuestionSolde); !victime || !tueur {
		t.Fatalf("faces du solde = (%v, %v), attendu les deux", victime, tueur)
	}
}
