package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/legacymatch"
)

// fakeSessionObjectiveRepo : port.ObjectiveTeamRepository en mémoire ; compte les lectures pour prouver
// qu'il n'y en a qu'UNE pour tous les matchs de la session.
type fakeSessionObjectiveRepo struct {
	rows  map[string][]narrative.ObjectiveTeamRow
	err   error
	calls int
	asked []string
}

func (f *fakeSessionObjectiveRepo) LoadObjectiveTeamRows(_ context.Context, matchIDs []string) (map[string][]narrative.ObjectiveTeamRow, error) {
	f.calls++
	f.asked = matchIDs
	return f.rows, f.err
}

func (f *fakeSessionObjectiveRepo) XUIDsByGamertag(context.Context, []string) (map[string][]string, error) {
	return nil, nil
}

// L'axe Objectif d'une session est la note moyenne des matchs, lue en UNE requête pour tous les matchs ;
// repo absent, xuid inconnu ou lecture en échec : aucun match mesuré, l'axe est retiré (jamais un zéro).
func TestSessionPageService_ObjectiveForNoteMoyenneEnUneLecture(t *testing.T) {
	ctf := func(xuid string, team int, captures float64) narrative.ObjectiveTeamRow {
		return narrative.ObjectiveTeamRow{
			XUID: xuid, TeamID: team, Family: narrative.FamilyCTF, TimePlayedSeconds: 600,
			Values: map[string]float64{colFlagCaptures: captures},
		}
	}
	repo := &fakeSessionObjectiveRepo{rows: map[string][]narrative.ObjectiveTeamRow{
		"m1": {ctf("me", 0, 2), ctf("mate", 0, 2), ctf("foe", 1, 9)}, // part équitable : 50
		"m2": {ctf("me", 0, 3), ctf("mate", 0, 0), ctf("foe", 1, 9)}, // tout l'objectif de l'équipe : 100
	}}
	matches := []legacymatch.StatsMatchRow{{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}}
	svc := &SessionPageService{objectiveTeam: repo, objectiveXUID: "me"}

	got := svc.objectiveFor(context.Background(), matches)
	if got.Matches != 2 || got.Score != 75 {
		t.Fatalf("(%v, %d matchs), want (75, 2) : m3 sans ligne n'est pas mesuré", got.Score, got.Matches)
	}
	if repo.calls != 1 || len(repo.asked) != 3 {
		t.Errorf("lectures = %d pour %d matchs, want 1 lecture pour les 3 matchs", repo.calls, len(repo.asked))
	}

	for name, s := range map[string]*SessionPageService{
		"repo absent":      {objectiveXUID: "me"},
		"xuid inconnu":     {objectiveTeam: repo},
		"lecture en échec": {objectiveTeam: &fakeSessionObjectiveRepo{err: errors.New("panne")}, objectiveXUID: "me"},
	} {
		if got := s.objectiveFor(context.Background(), matches); got.Matches != 0 {
			t.Errorf("%s : %+v, want aucun match mesuré (axe retiré)", name, got)
		}
	}
}
