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
	"levelup/go-api/internal/analysis/squadformes"
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
	padTiers     []sessionusage.PadTierRow
	padTiersErr  error
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

// LoadPadTiers — les PRISES DE SOCLE PAR NIVEAU D'ARME. `padTiers` nil = aucune ligne, donc
// « non mesure » : c'est l'etat par defaut de tous les temoins qui n'en parlent pas.
func (m *mockTeammatesUsageRepo) LoadPadTiers(_ context.Context, _ []string) ([]sessionusage.PadTierRow, error) {
	return m.padTiers, m.padTiersErr
}

// compteurUsageRepo compte les lectures du résumé d'usage ; il sert AUSSI de source au bloc
// « formes retenues » (LoadUsageFilmPads), comme le câblage de production.
type compteurUsageRepo struct {
	*mockTeammatesUsageRepo
	lectures map[string]int
}

func (c *compteurUsageRepo) LoadUsageFilms(ctx context.Context, ids []string) (map[string]sessionusage.FilmRow, error) {
	c.lectures["films"]++
	return c.mockTeammatesUsageRepo.LoadUsageFilms(ctx, ids)
}
func (c *compteurUsageRepo) LoadUsagePlayers(ctx context.Context, ids []string) ([]sessionusage.PlayerRow, error) {
	c.lectures["players"]++
	return c.mockTeammatesUsageRepo.LoadUsagePlayers(ctx, ids)
}
func (c *compteurUsageRepo) LoadParticipants(ctx context.Context, ids []string) ([]sessionusage.ParticipantRow, error) {
	c.lectures["participants"]++
	return c.mockTeammatesUsageRepo.LoadParticipants(ctx, ids)
}
func (c *compteurUsageRepo) LoadUsageFilmPads(context.Context, []string) (map[string]squadformes.FilmPads, error) {
	c.lectures["pads"]++
	return nil, nil
}

// TestTeammatesService_GetPage_UsageEtFormesPartagentLeursLectures (D2.6, lot perf L2) : les
// deux blocs du résumé d'usage publient leur section en ne lisant qu'UNE fois les films, les
// joueurs et les participants du scope (avant : deux fois chacun).
func TestTeammatesService_GetPage_UsageEtFormesPartagentLeursLectures(t *testing.T) {
	t.Parallel()
	t0 := time.Now().UTC().Add(-time.Hour)
	repo := &mockSquadRepo{
		topRows:   []domain.TopTeammateRow{{XUID: "x1", Gamertag: "Ally1", GamesTogether: 1}},
		synthRows: []legacymatch.SynthesisMatchRow{{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin}},
	}
	usage := &compteurUsageRepo{lectures: map[string]int{}, mockTeammatesUsageRepo: &mockTeammatesUsageRepo{
		films: map[string]sessionusage.FilmRow{"m1": {MatchID: "m1", DurationMS: 600000}},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "player-xuid", Gamertag: "Main", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
		},
	}}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, nil), "halo_infinite", "Main").
		WithEquipmentUsage(usage).
		WithSquadFormes(usage, nil, "")
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if resp.EquipmentUsage == nil || !resp.EquipmentUsage.Available || resp.SquadFormes == nil {
		t.Fatalf("les deux blocs doivent être publiés : usage %+v, formes %v", resp.EquipmentUsage, resp.SquadFormes != nil)
	}
	for _, lecture := range []string{"films", "players", "participants", "pads"} {
		if usage.lectures[lecture] != 1 {
			t.Errorf("lecture %s faite %d fois, attendu 1 (lectures %v)", lecture, usage.lectures[lecture], usage.lectures)
		}
	}
}
