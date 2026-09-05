package service

// session_page_usage_test.go — l'attachement du bloc usage à la page session :
// capability absente (repo nil) ⇒ Available=false avec raison machine, jamais
// d'échec ; erreur de lecture ⇒ load_failed ; contexte escouade ⇒ coéquipiers
// suivis + lignes squad ; sous-bloc objectifs via le loader optionnel.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

type mockSessionUsageRepo struct {
	films        map[string]sessionusage.FilmRow
	players      []sessionusage.PlayerRow
	participants []sessionusage.ParticipantRow
	filmsErr     error
}

func (m *mockSessionUsageRepo) LoadUsageFilms(_ context.Context, _ []string) (map[string]sessionusage.FilmRow, error) {
	return m.films, m.filmsErr
}
func (m *mockSessionUsageRepo) LoadUsagePlayers(_ context.Context, _ []string) ([]sessionusage.PlayerRow, error) {
	return m.players, nil
}
func (m *mockSessionUsageRepo) LoadParticipants(_ context.Context, _ []string) ([]sessionusage.ParticipantRow, error) {
	return m.participants, nil
}

// mockObjectiveIndexWithRoles implémente port.ObjectiveIndexRepository ET la
// capability optionnelle objectiveRoleRowsLoader (comme duckdb.ObjectiveStatsRepo).
type mockObjectiveIndexWithRoles struct {
	roleRows []sessionusage.ObjectiveRow
}

func (m *mockObjectiveIndexWithRoles) LoadObjectiveIndexInputs(_ context.Context, _, _ []string) (map[string]narrative.ObjectiveIndexInput, error) {
	return map[string]narrative.ObjectiveIndexInput{}, nil
}
func (m *mockObjectiveIndexWithRoles) LoadObjectiveIndexInputsByGamertag(_ context.Context, _ []string, _ string) (narrative.ObjectiveIndexInput, error) {
	return narrative.ObjectiveIndexInput{}, nil
}
func (m *mockObjectiveIndexWithRoles) LoadObjectiveRoleRows(_ context.Context, _ []string) ([]sessionusage.ObjectiveRow, error) {
	return m.roleRows, nil
}

func usageTestMatches() []legacymatch.StatsMatchRow {
	return []legacymatch.StatsMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
}

func teamp(v int) *int { return &v }

func usageTestRepoMock() *mockSessionUsageRepo {
	return &mockSessionUsageRepo{
		films: map[string]sessionusage.FilmRow{
			"m1": {MatchID: "m1", DurationMS: 600000, PadUnnamed: 2},
		},
		players: []sessionusage.PlayerRow{
			{MatchID: "m1", XUID: "P", PadPickups: 2},
			{MatchID: "m1", XUID: "A", PadPickups: 1},
			{MatchID: "m1", XUID: "E1", PadPickups: 1},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
		},
	}
}

func TestAttachSessionUsage_CapabilityAbsente(t *testing.T) {
	svc := NewSessionPageService(nil) // repo usage jamais câblé
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), domain.MatchContextSolo)
	if resp.Usage == nil {
		t.Fatal("Usage nil : le bloc doit être présent avec Available=false, pas absent")
	}
	if resp.Usage.Available || resp.Usage.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc = %+v, attendu Available=false, raison %q", resp.Usage, domain.SessionUsageUnsupported)
	}
	if resp.Usage.MatchesTotal != 2 {
		t.Errorf("matches_total = %d, attendu 2 (le dénominateur reste dit)", resp.Usage.MatchesTotal)
	}
}

func TestAttachSessionUsage_SessionSansMatch(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil)
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, nil, domain.MatchContextSolo)
	if resp.Usage != nil {
		t.Errorf("Usage = %+v, attendu nil (session sans match)", resp.Usage)
	}
}

func TestAttachSessionUsage_ErreurDeLectureDegrade(t *testing.T) {
	repo := usageTestRepoMock()
	repo.filmsErr = errors.New("boom")
	svc := NewSessionPageService(nil).WithSessionUsage(repo, "P", nil)
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), domain.MatchContextSolo)
	if resp.Usage == nil || resp.Usage.Available || resp.Usage.UnavailableReason != domain.SessionUsageLoadFailed {
		t.Errorf("bloc = %+v, attendu Available=false raison %q", resp.Usage, domain.SessionUsageLoadFailed)
	}
}

func TestAttachSessionUsage_ContexteEscouade(t *testing.T) {
	svc := NewSessionPageService(nil).
		WithSessionUsage(usageTestRepoMock(), "P", func(context.Context) []string { return []string{"Alpha"} })
	svc.objectiveIndex = &mockObjectiveIndexWithRoles{roleRows: []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 2},
		{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Take: 1},
	}}
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), domain.MatchContextSquad)
	u := resp.Usage
	if u == nil || !u.Available {
		t.Fatalf("Usage = %+v, attendu bloc disponible", u)
	}
	if u.MatchesMeasured != 1 || u.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 1/2", u.MatchesMeasured, u.MatchesTotal)
	}
	if len(u.SquadPlayers) != 1 || u.SquadPlayers[0].XUID != "A" || u.SquadPlayers[0].Gamertag != "Alpha" {
		t.Fatalf("squad_players = %+v, attendu [A/Alpha] (Echo est adverse)", u.SquadPlayers)
	}
	var pad *domain.SessionUsageMetric
	for i := range u.Metrics {
		if u.Metrics[i].Key == sessionusage.MetricPadPickups {
			pad = &u.Metrics[i]
		}
	}
	if pad == nil || len(pad.Squad) != 1 || pad.Squad[0].XUID != "A" || pad.Squad[0].Total != 1 {
		t.Errorf("lignes squad pad_pickups = %+v, attendu total 1 pour A", pad)
	}
	if u.Objectives == nil || len(u.Objectives.Roles) == 0 {
		t.Fatalf("objectifs = %+v, attendu sous-bloc via le loader optionnel", u.Objectives)
	}
	if u.Objectives.Roles[0].Squad == nil {
		t.Error("les rôles d'objectif doivent porter les lignes squad en contexte escouade")
	}
}

func TestAttachSessionUsage_ContexteSoloSansLigneSquad(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil)
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), domain.MatchContextSolo)
	u := resp.Usage
	if u == nil || !u.Available {
		t.Fatalf("Usage = %+v, attendu bloc disponible", u)
	}
	if len(u.SquadPlayers) != 0 {
		t.Errorf("squad_players = %+v, attendu vide en solo", u.SquadPlayers)
	}
	for _, m := range u.Metrics {
		if len(m.Squad) != 0 {
			t.Errorf("métrique %s porte des lignes squad en solo : %+v", m.Key, m.Squad)
		}
	}
	if u.Objectives != nil {
		t.Errorf("objectifs = %+v, attendu nil (aucun loader câblé)", u.Objectives)
	}
}
