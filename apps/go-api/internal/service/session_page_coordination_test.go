package service

// session_page_coordination_test.go — L'ATTACHEMENT du bloc « Coordination » à la page
// détail de session (lot S, 2026-09-22).
//
// Ce que ces tests cadenassent :
//   - le MIROIR de la session comparée est servi quand le drawer est ouvert, et jamais
//     autrement (publier un bloc que rien ne rend serait du code mort à chaque requête) ;
//   - les deux blocs sont CALCULÉS SÉPARÉMENT (jamais le même pointeur) ;
//   - le repère d'habituel vient de la PÉRIODE DE RÉFÉRENCE, et il est OMIS quand cette
//     référence est tautologique — sans même ouvrir le journal des morts une fois de plus.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// tacticalRepoParScope — un journal des morts qui RÉPOND AU SCOPE DEMANDÉ, et garde la
// trace de chaque appel. Le stub du producteur (`tacticalRepoStub`) rend toujours le même
// univers : ici c'est justement la différence entre les scopes qui est en cause.
type tacticalRepoParScope struct {
	port.TacticalRepository
	appels [][]string
}

func (s *tacticalRepoParScope) KillEvents(
	_ context.Context, q domain.TacticalQuery,
) (domain.TacticalKillEvents, error) {
	ids := q.Matchs.IDs()
	s.appels = append(s.appels, ids)
	return restreindreLectureDeTest(ids), nil
}

// restreindreLectureDeTest projette `lectureDeTest` sur les matchs demandés.
func restreindreLectureDeTest(ids []string) domain.TacticalKillEvents {
	garde := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		garde[id] = struct{}{}
	}
	src := lectureDeTest()
	out := domain.TacticalKillEvents{
		Univers: domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}},
	}
	for _, m := range src.Univers.Matchs {
		if _, ok := garde[m.MatchID]; ok {
			out.Univers.Matchs = append(out.Univers.Matchs, m)
			out.Univers.Equipes[m.MatchID] = src.Univers.Equipes[m.MatchID]
		}
	}
	for _, e := range src.Events {
		if _, ok := garde[e.MatchID]; ok {
			out.Events = append(out.Events, e)
		}
	}
	return out
}

func serviceDeCoordination(tactical port.TacticalRepository) *SessionPageService {
	return NewSessionPageService(nil).
		WithSessionUsage(usageTestRepoMock(), "P", nil, "").
		WithSessionCoordination(tactical, &appuisRepoStub{rows: []domain.CoordinationAppuiRow{
			{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
			{MatchID: "m1", AssistXUID: "", KillerXUID: "P", Nombre: 1},
		}}, games.CapabilityMap{games.CapFilmKillSource: games.CapSupported})
}

func matchsDeTest(ids ...string) []legacymatch.StatsMatchRow {
	rows := make([]legacymatch.StatsMatchRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, legacymatch.StatsMatchRow{MatchID: id})
	}
	return rows
}

func TestAttachSessionCoordination_MiroirServiQuandLaComparaisonEstDemandee(t *testing.T) {
	svc := serviceDeCoordination(&tacticalRepoParScope{})
	var resp domain.SessionPageResponse
	svc.attachSessionCoordination(context.Background(), &resp, sessionBlocksScope{
		Matches: matchsDeTest("m1"), CompareMatches: matchsDeTest("m2"),
	}, map[string]int{"m1": 4}, map[string]int{"m2": 4})

	if resp.Coordination == nil || resp.CompareCoordination == nil {
		t.Fatalf("coordination = %v, compare = %v : les deux colonnes doivent porter un bloc",
			resp.Coordination, resp.CompareCoordination)
	}
	if resp.Coordination == resp.CompareCoordination {
		t.Fatal("les deux colonnes partagent le même bloc")
	}
	if resp.CompareCoordination.MatchesTotal != 1 {
		t.Errorf("compare matches_total = %d, attendu 1 : le miroir porte les matchs de la session comparée",
			resp.CompareCoordination.MatchesTotal)
	}
	// La session comparée (m2) est celle où JE meurs et où un coéquipier riposte.
	if got := resp.CompareCoordination.Riposte.JeSuisCouvert; got.N != 1 || got.Brut != 1 {
		t.Errorf("compare je suis couvert = %d/%d, attendu 1/1", got.Brut, got.N)
	}
}

func TestAttachSessionCoordination_AucunMiroirHorsComparaison(t *testing.T) {
	svc := serviceDeCoordination(&tacticalRepoParScope{})
	var resp domain.SessionPageResponse
	svc.attachSessionCoordination(context.Background(), &resp, sessionBlocksScope{
		Matches: matchsDeTest("m1", "m2"),
	}, map[string]int{"m1": 4, "m2": 4}, nil)

	if resp.Coordination == nil {
		t.Fatal("coordination = nil, attendu un bloc pour la session affichée")
	}
	if resp.CompareCoordination != nil {
		t.Errorf("compare_coordination = %+v, attendu nil hors comparaison", resp.CompareCoordination)
	}
}

func TestAttachSessionCoordination_HabituelDeLaPeriodeDeReference(t *testing.T) {
	svc := serviceDeCoordination(&tacticalRepoParScope{})
	var resp domain.SessionPageResponse
	svc.attachSessionCoordination(context.Background(), &resp, sessionBlocksScope{
		Matches:          matchsDeTest("m1"),
		ReferenceMatches: matchsDeTest("m1", "m2"),
	}, map[string]int{"m1": 4}, nil)

	if resp.Coordination == nil {
		t.Fatal("coordination = nil")
	}
	// Sur la référence (m1 + m2) je meurs une fois, et cette mort est ripostée : 100 %.
	if got := resp.Coordination.Riposte.HabituelPct; got == nil || *got != 100 {
		t.Errorf("riposte.habituel_pct = %v, attendu 100", got)
	}
	// Sur la référence, un de mes deux frags mesurés m'a été préparé : 50 %.
	if got := resp.Coordination.Appui.HabituelPct; got == nil || *got != 50 {
		t.Errorf("appui.habituel_pct = %v, attendu 50", got)
	}
}

func TestAttachSessionCoordination_PasDHabituelQuandLaReferenceEstTautologique(t *testing.T) {
	tactical := &tacticalRepoParScope{}
	svc := serviceDeCoordination(tactical)
	var resp domain.SessionPageResponse
	svc.attachSessionCoordination(context.Background(), &resp, sessionBlocksScope{
		Matches:          matchsDeTest("m1", "m2"),
		ReferenceMatches: matchsDeTest("m2", "m1"),
	}, map[string]int{"m1": 4, "m2": 4}, nil)

	if resp.Coordination == nil {
		t.Fatal("coordination = nil")
	}
	if resp.Coordination.Riposte.HabituelPct != nil || resp.Coordination.Appui.HabituelPct != nil {
		t.Errorf("habituel = %v / %v, attendu absent : la référence se réduit au scope mesuré",
			resp.Coordination.Riposte.HabituelPct, resp.Coordination.Appui.HabituelPct)
	}
	if len(tactical.appels) != 1 {
		t.Errorf("%d lectures du journal des morts, attendu 1 : une référence tautologique ne se lit pas",
			len(tactical.appels))
	}
}
