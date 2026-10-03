package service

// timeseries_service_equipes_test.go — LES PARTICIPANTS DU SCOPE SONT LUS UNE FOIS (lot L5a
// du plan perf, 2026-09-23).
//
// La courbe d'équipe du profil d'intensité et l'effectif de camp de la coordination posaient
// la même question aux mêmes matchs : deux lectures identiques de `match_participants` par
// requête. Ce test cadenasse, sur la page ENTIÈRE :
//   - une seule section `participants`, appelée une fois ;
//   - la lecture NOURRIT les deux consommateurs (courbe d'équipe servie, parité servie) —
//     sans quoi « une lecture » pourrait vouloir dire « une lecture, et un consommateur
//     privé de données ».
//
// LA SECTION FAIT FOI, PAS LE COMPTEUR DU PORT : le bloc « formes retenues » (squadagg, hors
// du périmètre de ce lot) lit aussi les participants par le même port, sous sa propre
// section. Compter les appels du port mélangerait les deux lectures.

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// formesAvecCompteur : le port des « formes retenues » de la page, qui compte ses lectures de
// participants.
type formesAvecCompteur struct {
	*mockSessionUsageRepo
	lectures int
}

func (f *formesAvecCompteur) LoadParticipants(ctx context.Context, ids []string) ([]sessionusage.ParticipantRow, error) {
	f.lectures++
	return f.mockSessionUsageRepo.LoadParticipants(ctx, ids)
}

func (f *formesAvecCompteur) LoadUsageFilmPads(context.Context, []string) (map[string]squadformes.FilmPads, error) {
	return nil, nil
}

// highlightEventsFixes rend toujours les mêmes événements.
type highlightEventsFixes struct{ events []canonical.HighlightEvent }

func (h highlightEventsFixes) Load(context.Context, port.HighlightEventFilters) ([]canonical.HighlightEvent, error) {
	return h.events, nil
}

func fragDeTest(matchID, tueur string, ms int64) canonical.HighlightEvent {
	k := tueur
	return canonical.HighlightEvent{
		MatchID: matchID, EventType: string(canonical.EventKill), TimeMS: ms, XUID: tueur, KillerXUID: &k,
	}
}

func TestTimeseriesPage_ParticipantsLusUneFoisPourLesDeuxConsommateurs(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
	formes := &formesAvecCompteur{mockSessionUsageRepo: usageTestRepoMock()}
	svc := NewTimeseriesService(&mockTimeseriesRepo{matches: matches}).
		WithPlayerMatchesRepo(newStatsMockFromRows(matches, nil), "halo_infinite", "Test").
		WithHighlightEventsRepo(highlightEventsFixes{events: []canonical.HighlightEvent{
			fragDeTest("m1", "P", 1000), fragDeTest("m1", "A", 2000), fragDeTest("m2", "E1", 3000),
		}}, "P").
		WithSquadFormes(formes, nil).
		WithTimeseriesCoordination(&tacticalRepoParScope{}, &appuisRepoStub{},
			games.CapabilityMap{games.CapFilmKillSource: games.CapSupported})

	ctx, chrono := timing.WithTimings(context.Background())
	resp, err := svc.GetPage(ctx, domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}

	appels := map[string]int{}
	for _, s := range chrono.Snapshot() {
		appels[s.Name] = s.Calls
	}
	if appels["participants"] != 1 {
		t.Errorf("section participants = %d appel(s), attendu 1 : les participants du scope se lisent une fois",
			appels["participants"])
	}
	if n, reste := appels["team_sizes"]; reste {
		t.Errorf("section team_sizes = %d appel(s) : la coordination relit les participants", n)
	}
	if len(resp.IntensityRowsTeam) == 0 {
		t.Error("courbe d'équipe absente : la lecture partagée n'a pas nourri le profil d'intensité")
	}
	if resp.Coordination == nil || resp.Coordination.Riposte.ParityPct == nil {
		t.Errorf("coordination = %+v : la parité devait venir des effectifs de la lecture partagée", resp.Coordination)
	}
}
