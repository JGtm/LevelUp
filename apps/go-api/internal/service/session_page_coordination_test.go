package service

// session_page_coordination_test.go — L'ATTACHEMENT du bloc « Coordination » à la page
// détail de session (lot S, 2026-09-22).
//
// Ce que ces tests cadenassent :
//   - le MIROIR de la session comparée est servi quand le drawer est ouvert, et jamais
//     autrement (publier un bloc que rien ne rend serait du code mort à chaque requête) ;
//   - les deux blocs sont CALCULÉS SÉPARÉMENT (jamais le même pointeur) ;
//   - le repère d'habituel vient de la PÉRIODE DE RÉFÉRENCE, et il est OMIS quand cette
//     référence est tautologique — sans même ouvrir le journal des morts une fois de plus ;
//   - les trois scopes lisent le journal et les appuis UNE fois par match (lot L5a,
//     2026-09-23), et la lecture partagée rend les MÊMES blocs que trois lectures séparées.

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// tacticalRepoParScope — un journal des morts qui RÉPOND AU SCOPE DEMANDÉ, et garde la
// trace de chaque appel. Le stub du producteur (`tacticalRepoStub`) rend toujours le même
// univers : ici c'est justement la différence entre les scopes qui est en cause. `source`
// nil = `lectureDeTest`.
type tacticalRepoParScope struct {
	port.TacticalRepository
	appels [][]string
	source func() domain.TacticalKillEvents
}

func (s *tacticalRepoParScope) KillEvents(
	_ context.Context, q domain.TacticalQuery,
) (domain.TacticalKillEvents, error) {
	ids := q.Matchs.IDs()
	s.appels = append(s.appels, ids)
	source := s.source
	if source == nil {
		source = lectureDeTest
	}
	return restreindreLectureDeTest(source(), ids), nil
}

// restreindreLectureDeTest projette une lecture sur les matchs demandés.
func restreindreLectureDeTest(src domain.TacticalKillEvents, ids []string) domain.TacticalKillEvents {
	garde := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		garde[id] = struct{}{}
	}
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

// lectureAvecReference — lectureDeTest plus m3, un match de la période de référence hors
// des deux sessions : E1 me tue à 1 s, et personne ne le venge dans la fenêtre (je le tue
// moi-même à 8 s).
func lectureAvecReference() domain.TacticalKillEvents {
	l := lectureDeTest()
	l.Univers.Matchs = append(l.Univers.Matchs, domain.TacticalMatch{MatchID: "m3", Mesure: true})
	l.Univers.Equipes["m3"] = map[string]int{"P": 0, "A": 0, "E1": 1}
	l.Events = append(l.Events,
		domain.KillEvent{MatchID: "m3", KillerXUID: "E1", VictimXUID: "P", TimeMs: 1000},
		domain.KillEvent{MatchID: "m3", KillerXUID: "P", VictimXUID: "E1", TimeMs: 8000},
	)
	return l
}

// appuisAvecReference — les appuis de m1 (un de mes frags préparé sur deux) et de m3 (mon
// frag, sans assistant).
func appuisAvecReference() []domain.CoordinationAppuiRow {
	return []domain.CoordinationAppuiRow{
		{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
		{MatchID: "m1", AssistXUID: "", KillerXUID: "P", Nombre: 1},
		{MatchID: "m3", AssistXUID: "", KillerXUID: "P", Nombre: 1},
	}
}

func serviceAvecReference(tactical *tacticalRepoParScope, appuis *appuisRepoStub) *SessionPageService {
	return NewSessionPageService(nil).
		WithSessionUsage(usageTestRepoMock(), "P", nil, "").
		WithSessionCoordination(tactical, appuis, games.CapabilityMap{games.CapFilmKillSource: games.CapSupported})
}

// scopeAvecReference : la session m1, la session comparée m2, la référence m1 + m2 + m3.
func scopeAvecReference() sessionBlocksScope {
	return sessionBlocksScope{
		Matches: matchsDeTest("m1"), CompareMatches: matchsDeTest("m2"),
		ReferenceMatches: matchsDeTest("m1", "m2", "m3"),
	}
}

// TestAttachSessionCoordination_UneLectureParMatch — D5a.3 (lot L5a du plan perf,
// 2026-09-23) : le journal des morts et les appuis des TROIS scopes sont lus une fois par
// match. Deux lectures au plus : les deux sessions ensemble, puis le SEUL complément de la
// référence — jamais m1 ni m2 une seconde fois.
func TestAttachSessionCoordination_UneLectureParMatch(t *testing.T) {
	tactical := &tacticalRepoParScope{source: lectureAvecReference}
	appuis := &appuisRepoStub{rows: appuisAvecReference()}
	var resp domain.SessionPageResponse
	serviceAvecReference(tactical, appuis).attachSessionCoordination(context.Background(), &resp,
		scopeAvecReference(), map[string]int{"m1": 4}, map[string]int{"m2": 4})

	want := fmt.Sprint([][]string{{"m1", "m2"}, {"m3"}})
	if got := fmt.Sprint(tactical.appels); got != want {
		t.Errorf("lectures du journal = %s, attendu %s : chaque match lu une fois", got, want)
	}
	if got := fmt.Sprint(appuis.appels); got != want {
		t.Errorf("lectures des appuis = %s, attendu %s : chaque match lu une fois", got, want)
	}
	if resp.Coordination == nil || resp.Coordination.Riposte.HabituelPct == nil {
		t.Fatalf("coordination = %+v : la référence (non tautologique) devait poser un habituel", resp.Coordination)
	}
}

// TestAttachSessionCoordination_PariteAvecLesLecturesSeparees — la lecture partagée rend
// EXACTEMENT les blocs que rendaient trois lectures séparées : effectifs de camp propres à
// chaque scope, couverture, cases par match et repères d'habituel. L'ORACLE est le chemin
// d'avant le lot L5a : un scope, une lecture (buildCoordinationBlock).
func TestAttachSessionCoordination_PariteAvecLesLecturesSeparees(t *testing.T) {
	ctx := context.Background()
	sc := scopeAvecReference()
	teamSize, compareTeamSize := map[string]int{"m1": 4}, map[string]int{"m2": 3}
	var resp domain.SessionPageResponse
	serviceAvecReference(&tacticalRepoParScope{source: lectureAvecReference},
		&appuisRepoStub{rows: appuisAvecReference()}).
		attachSessionCoordination(ctx, &resp, sc, teamSize, compareTeamSize)

	oracle := serviceAvecReference(&tacticalRepoParScope{source: lectureAvecReference},
		&appuisRepoStub{rows: appuisAvecReference()})
	unScope := func(ids []string, effectifs map[string]int) *domain.CoordinationBlock {
		q := oracle.lecteursDeCoordination()
		q.MatchIDs, q.TeamSize = ids, effectifs
		return buildCoordinationBlock(ctx, q)
	}
	wantCourant := unScope([]string{"m1"}, teamSize)
	wantCompare := unScope([]string{"m2"}, compareTeamSize)
	ref := unScope([]string{"m1", "m2", "m3"}, nil)
	for _, bloc := range []*domain.CoordinationBlock{wantCourant, wantCompare} {
		bloc.Riposte.HabituelPct = tauxOuRien(ref.Riposte.JeSuisCouvert)
		bloc.Appui.HabituelPct = tauxOuRien(ref.Appui.OnMePrepare)
	}

	if !reflect.DeepEqual(resp.Coordination, wantCourant) {
		t.Errorf("session affichée :\n got %+v\nwant %+v", resp.Coordination, wantCourant)
	}
	if !reflect.DeepEqual(resp.CompareCoordination, wantCompare) {
		t.Errorf("session comparée :\n got %+v\nwant %+v", resp.CompareCoordination, wantCompare)
	}
	// Garde-fou de l'oracle lui-même : sur la référence, une de mes deux morts est vengée
	// (50 %), un de mes trois frags mesurés m'a été préparé.
	if got := resp.Coordination.Riposte.HabituelPct; got == nil || *got != 50 {
		t.Errorf("riposte.habituel_pct = %v, attendu 50", got)
	}
	if got := resp.CompareCoordination.Appui.HabituelPct; got == nil || *got < 33.3 || *got > 33.4 {
		t.Errorf("appui.habituel_pct = %v, attendu 33,3", got)
	}
}
