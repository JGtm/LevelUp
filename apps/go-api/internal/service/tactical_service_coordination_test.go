package service

// tactical_service_coordination_test.go — LA SECTION « COORDINATION D'EQUIPE » (lot F,
// 2026-09-13, maquette 034b1915).
//
// Ce que ces tests cadenassent, et le defaut que chacun attrape :
//
//	servie sur TOUTES les questions   la reserver a « ou je meurs isole » obligeait a
//	                                  changer de question pour lire un chiffre qui ne
//	                                  change pas ;
//	une SEULE lecture des morts       « ou je meurs isole » la pose lui-meme, depuis la
//	                                  lecture qu'il fait deja — une seconde requete pour la
//	                                  meme table serait un cout pur ;
//	deux rayons, jamais leur moyenne  un filtre qui melange Arene (18 m) et BTB (24 m)
//	                                  melange DEUX REGLES DU JEU.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func lireQuestion(t *testing.T, svc *TacticalService, question string, ids ...string) domain.TacticalRaster {
	t.Helper()
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: question, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids},
	})
	if err != nil {
		t.Fatalf("lecture %s: %v", question, err)
	}
	return out
}

// svcCoordination monte le service avec les memes morts servies aux DEUX substrats : le
// journal des positions (lectures morts/kills/gagne) et le contexte des morts.
func svcCoordination(univ domain.TacticalUnivers, morts ...domain.MortContexte) *TacticalService {
	repo := &mockTacticalRepo{
		univ:  univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: morts},
	}
	repo.pos.Univers = univ
	for _, mc := range morts {
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: mc.MatchID, KillerXUID: tsAdv, VictimXUID: mc.VictimXUID,
			KillerX: 9, KillerY: 9, VictimX: mc.X, VictimY: mc.Y, TimeMs: 1000,
		})
	}
	return NewTacticalService(repo, capsCompletes(), tsMoi).
		WithRadarRange(map[string]int{"Slayer:Arena": 18, "BTB:Slayer": 24})
}

// TestCoordination_ServieSurUneQuestionQuiNEstPasIsole : « ou je meurs » publie la section
// ET le taux d'isolement. Sans ce test, la version precedente (isolement reserve a la
// question « isole ») repasserait sans qu'aucun rouge n'apparaisse.
func TestCoordination_ServieSurUneQuestionQuiNEstPasIsole(t *testing.T) {
	svc := svcCoordination(universVariantes(map[string]string{"arene": "Slayer:Arena"}),
		mortContexte("arene", tsMoi, 2, 3, m(5), 1, 0),
		mortContexte("arene", tsMoi, 4, 5, m(25), 1, 0))

	out := lireQuestion(t, svc, domain.TacticalQuestionMorts, "arene")

	if out.Coordination == nil {
		t.Fatal("aucune section de coordination servie sur la question « ou je meurs »")
	}
	if out.Isolement == nil {
		t.Fatal("aucun taux d'isolement servi hors de la question « isole »")
	}
	if out.Coordination.NDistances != 2 {
		t.Errorf("n_distances = %d, attendu 2", out.Coordination.NDistances)
	}
	if out.Coordination.DistanceMedianeM == nil || *out.Coordination.DistanceMedianeM != 15 {
		t.Errorf("mediane = %v, attendu 15 (moyenne de 5 et 25)", out.Coordination.DistanceMedianeM)
	}
	if out.Coordination.FenetreEchangeSecondes != 5 {
		t.Errorf("fenetre d'echange = %d s, attendu 5", out.Coordination.FenetreEchangeSecondes)
	}
	if len(out.Coordination.RayonsM) != 1 || out.Coordination.RayonsM[0] != 18 {
		t.Errorf("rayons = %v, attendu [18]", out.Coordination.RayonsM)
	}
}

// TestCoordination_DeuxFormatsDeuxRayons : le filtre melange Arene et BTB — les DEUX
// portees sortent, triees, jamais leur moyenne (21 m ne serait la regle d'aucun match).
func TestCoordination_DeuxFormatsDeuxRayons(t *testing.T) {
	svc := svcCoordination(universVariantes(map[string]string{
		"arene": "Slayer:Arena", "btb": "BTB:Slayer",
	}),
		mortContexte("arene", tsMoi, 2, 3, m(19), 1, 0),
		mortContexte("btb", tsMoi, 2, 3, m(19), 1, 0))

	out := lireQuestion(t, svc, domain.TacticalQuestionMorts, "arene", "btb")

	if out.Coordination == nil {
		t.Fatal("aucune section de coordination servie")
	}
	got := out.Coordination.RayonsM
	if len(got) != 2 || got[0] != 18 || got[1] != 24 {
		t.Fatalf("rayons = %v, attendu [18 24] tries", got)
	}
}

// TestCoordination_IsoleNeRelitPasLesMorts : la question « ou je meurs isole » pose la
// section depuis SA PROPRE lecture. Une seconde requete pour la meme table serait un cout
// pur — et se verrait ici, la derniere requete vue portant alors deux fois le meme scope.
func TestCoordination_IsoleServiSansSecondeLecture(t *testing.T) {
	univ := universVariantes(map[string]string{"arene": "Slayer:Arena"})
	repo := &mockTacticalRepo{
		univ: univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{
			mortContexte("arene", tsMoi, 2, 3, m(9), 1, 0),
		}},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).
		WithRadarRange(map[string]int{"Slayer:Arena": 18})

	out := lireQuestion(t, svc, domain.TacticalQuestionIsole, "arene")

	if out.Coordination == nil {
		t.Fatal("« ou je meurs isole » ne publie pas la section de coordination")
	}
	if out.Coordination.NDistances != 1 {
		t.Errorf("n_distances = %d, attendu 1", out.Coordination.NDistances)
	}
}

// TestCoordination_MortSansCoequipierVisible : elle est ISOLEE (elle compte au taux) mais
// n'a AUCUNE distance. La verser dans le dernier intervalle inventerait une mesure.
func TestCoordination_MortSansCoequipierVisible(t *testing.T) {
	svc := svcCoordination(universVariantes(map[string]string{"arene": "Slayer:Arena"}),
		mortContexte("arene", tsMoi, 2, 3, nil, 0, 2))

	out := lireQuestion(t, svc, domain.TacticalQuestionMorts, "arene")

	if out.Coordination == nil {
		t.Fatal("aucune section de coordination servie")
	}
	if out.Coordination.NDistances != 0 {
		t.Errorf("n_distances = %d, attendu 0", out.Coordination.NDistances)
	}
	if out.Coordination.MortsSansDistance != 1 {
		t.Errorf("morts_sans_distance = %d, attendu 1", out.Coordination.MortsSansDistance)
	}
	if out.Coordination.DistanceMedianeM != nil {
		t.Errorf("mediane = %v, attendu nil", *out.Coordination.DistanceMedianeM)
	}
	if out.Isolement == nil || out.Isolement.Brut != 1 {
		t.Errorf("isolement = %+v, attendu 1 mort isolee : l'absence de distance ne l'efface pas", out.Isolement)
	}
}
