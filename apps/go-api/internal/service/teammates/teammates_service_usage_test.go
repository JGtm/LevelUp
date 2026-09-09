package teammates

// teammates_service_usage_test.go — étape E6.1bis : la page Teammates (POST
// /pages/teammates, le SEUL endpoint que la page Escouade réelle appelle) publie
// le bloc « servi ou gâché » de l'équipement sur son scope FILTRÉ, avec les
// coéquipiers SÉLECTIONNÉS comme « amis » — corrige la publication E6.1 posée à
// tort sur SquadPageV2Response (GET /pages/squad/v2, jamais fetché par la page).

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// mockTeammatesUsageRepo — mock minimal de port.SessionUsageRepository pour les
// tests du bloc équipement de la page Teammates.
type mockTeammatesUsageRepo struct {
	films        map[string]sessionusage.FilmRow
	players      []sessionusage.PlayerRow
	participants []sessionusage.ParticipantRow
}

func (m *mockTeammatesUsageRepo) LoadUsageFilms(_ context.Context, _ []string) (map[string]sessionusage.FilmRow, error) {
	return m.films, nil
}
func (m *mockTeammatesUsageRepo) LoadUsagePlayers(_ context.Context, _ []string) ([]sessionusage.PlayerRow, error) {
	return m.players, nil
}
func (m *mockTeammatesUsageRepo) LoadParticipants(_ context.Context, _ []string) ([]sessionusage.ParticipantRow, error) {
	return m.participants, nil
}

var _ port.SessionUsageRepository = (*mockTeammatesUsageRepo)(nil)

func teammatesUsageTeam(v int) *int { return &v }

// TestTeammatesService_GetPage_PublieLeBlocEquipementSurLeScopeFiltre —
// le coéquipier SÉLECTIONNÉ (pas un ami configuré) devient le joueur suivi du
// bloc, et le scope est celui de filteredMatches (le joueur principal), pas
// l'intersection escouade.
func TestTeammatesService_GetPage_PublieLeBlocEquipementSurLeScopeFiltre(t *testing.T) {
	t.Parallel()
	t0 := time.Now().UTC().Add(-time.Hour)
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "x1", Gamertag: "Ally1", GamesTogether: 3, WinsTogether: 2, WinRate: 0.66, AvgKDA: 1.2},
		},
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 3},
		},
	}
	usageRepo := &mockTeammatesUsageRepo{
		films: map[string]sessionusage.FilmRow{"m1": {MatchID: "m1", DurationMS: 600000}},
		players: []sessionusage.PlayerRow{
			{MatchID: "m1", XUID: "player-xuid", PadPickups: 2,
				TakenByFamily: map[string]int{"wall": 2}, KeptByFamily: map[string]int{"wall": 2}},
			{MatchID: "m1", XUID: "x1", PadPickups: 1,
				TakenByFamily: map[string]int{"wall": 1}, KeptByFamily: map[string]int{"wall": 1}},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "player-xuid", Gamertag: "Main", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "x1", Gamertag: "Ally1", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
		},
	}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Main").
		WithEquipmentUsage(usageRepo)

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	block := resp.EquipmentUsage
	if block == nil || !block.Available {
		t.Fatalf("bloc équipement = %+v, attendu disponible", block)
	}
	if block.MatchesMeasured != 1 || block.MatchesTotal != 1 {
		t.Errorf("couverture = %d/%d, attendu 1/1", block.MatchesMeasured, block.MatchesTotal)
	}
	if len(block.TrackedPlayers) != 1 || block.TrackedPlayers[0].Gamertag != "Ally1" {
		t.Fatalf("coéquipiers suivis = %+v, attendu [Ally1] (le coéquipier SÉLECTIONNÉ)", block.TrackedPlayers)
	}
}

// TestTeammatesService_GetPage_SansRepoLeBlocDitPourquoi — repo non câblé
// (titre sans film.usage_summary) : réponse partielle propre, jamais un 500.
func TestTeammatesService_GetPage_SansRepoLeBlocDitPourquoi(t *testing.T) {
	t.Parallel()
	repo := &mockSquadRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: time.Now().UTC(), Outcome: domain.OutcomeWin},
		},
	}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Main")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if resp.EquipmentUsage == nil || resp.EquipmentUsage.Available ||
		resp.EquipmentUsage.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc = %+v, attendu indisponible/unsupported", resp.EquipmentUsage)
	}
}
