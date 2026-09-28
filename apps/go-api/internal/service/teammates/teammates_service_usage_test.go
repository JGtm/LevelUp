package teammates

// teammates_service_usage_test.go — les blocs d'usage de la page Teammates (POST
// /pages/teammates, le SEUL endpoint que la page Escouade réelle appelle) : périmètre D2
// (plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), coéquipiers SÉLECTIONNÉS, lectures
// partagées. Le bloc « servi ou gâché » (`equipment_usage`) n'est plus publié ici depuis le
// lot L5.4 (plus de lecteur) : le périmètre se vérifie sur le bloc « formes retenues ».

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

// mockTeammatesUsageRepo — mock minimal de port.SessionUsageRepository (et de la source du
// bloc « formes retenues ») pour les tests des blocs d'usage de la page Teammates.
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

func (m *mockTeammatesUsageRepo) LoadUsageFilmPads(context.Context, []string) (map[string]squadformes.FilmPads, error) {
	return nil, nil
}

var (
	_ port.SessionUsageRepository     = (*mockTeammatesUsageRepo)(nil)
	_ port.SquadFormesUsageRepository = (*mockTeammatesUsageRepo)(nil)
)

func teammatesUsageTeam(v int) *int { return &v }

// usageFixture — deux matchs filtrés : m1 joué avec Ally1, m2 sans lui. Les deux ont un film et
// les deux joueurs principaux y ont des lignes d'usage.
func usageFixture(t0 time.Time) (*mockSquadRepo, *mockTeammatesUsageRepo) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "x1", Gamertag: "Ally1", GamesTogether: 3, WinsTogether: 2, WinRate: 0.66, AvgKDA: 1.2},
		},
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 3},
			{MatchID: "m2", StartTime: t0.Add(15 * time.Minute), Outcome: domain.OutcomeLoss, Kills: 4, Deaths: 8},
		},
		squadRows: []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin}},
	}
	usageRepo := &mockTeammatesUsageRepo{
		films: map[string]sessionusage.FilmRow{
			"m1": {MatchID: "m1", DurationMS: 600000}, "m2": {MatchID: "m2", DurationMS: 600000},
		},
		players: []sessionusage.PlayerRow{
			{MatchID: "m1", XUID: "player-xuid", PadPickups: 2,
				TakenByFamily: map[string]int{"wall": 2}, KeptByFamily: map[string]int{"wall": 2}},
			{MatchID: "m1", XUID: "x1", PadPickups: 1,
				TakenByFamily: map[string]int{"wall": 1}, KeptByFamily: map[string]int{"wall": 1}},
			{MatchID: "m2", XUID: "player-xuid", PadPickups: 1,
				TakenByFamily: map[string]int{"wall": 1}, KeptByFamily: map[string]int{"wall": 1}},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "player-xuid", Gamertag: "Main", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "x1", Gamertag: "Ally1", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "player-xuid", Gamertag: "Main", TeamID: teammatesUsageTeam(0), PresentAtCompletion: true},
		},
	}
	return repo, usageRepo
}

// TestTeammatesService_GetPage_PublieLesBlocsDUsageSurLePerimetreEscouade — D2 (plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : avec un coéquipier SÉLECTIONNÉ, le périmètre est
// la composition exacte ∩ les matchs filtrés (m1 seul : m2 a été joué sans Ally1), et le
// coéquipier sélectionné (pas un ami configuré) entre dans l'escouade du bloc. Vérifié sur les
// deux blocs publiés : « formes retenues » et l'Emprise.
func TestTeammatesService_GetPage_PublieLesBlocsDUsageSurLePerimetreEscouade(t *testing.T) {
	t.Parallel()
	repo, usageRepo := usageFixture(time.Now().UTC().Add(-time.Hour))
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Main").
		WithUsageSummary(usageRepo).
		WithSquadFormes(usageRepo, nil, "")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	block := resp.SquadFormes
	if block == nil || !block.Available {
		t.Fatalf("bloc formes = %+v, attendu disponible", block)
	}
	if block.MatchesMeasured != 1 || block.MatchesTotal != 1 {
		t.Errorf("couverture = %d/%d, attendu 1/1 (m2, joué sans Ally1, hors périmètre)", block.MatchesMeasured, block.MatchesTotal)
	}
	if len(block.Squad) != 2 || block.Squad[1].Gamertag != "Ally1" {
		t.Fatalf("escouade = %+v, attendu [Main, Ally1] (le coéquipier SÉLECTIONNÉ)", block.Squad)
	}
	if e := resp.SquadEmprise; e == nil || e.MatchesTotal != 1 {
		t.Errorf("Emprise = %+v, attendu le même périmètre (1 match)", e)
	}
	if resp.TotalMatches != 2 {
		t.Errorf("TotalMatches = %d, attendu 2 (le compteur de la page reste celui des matchs filtrés)", resp.TotalMatches)
	}
}

// TestTeammatesService_GetPage_SansSelectionLePerimetreEstLeScopeFiltre — D2, repli : sans
// coéquipier sélectionné, les blocs d'usage lisent tous les matchs filtrés du joueur.
func TestTeammatesService_GetPage_SansSelectionLePerimetreEstLeScopeFiltre(t *testing.T) {
	t.Parallel()
	repo, usageRepo := usageFixture(time.Now().UTC().Add(-time.Hour))
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Main").
		WithUsageSummary(usageRepo).
		WithSquadFormes(usageRepo, nil, "")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if b := resp.SquadFormes; b == nil || b.MatchesTotal != 2 {
		t.Fatalf("bloc formes = %+v, attendu les 2 matchs filtrés", b)
	}
	if e := resp.SquadEmprise; e == nil || e.MatchesTotal != 2 {
		t.Fatalf("Emprise = %+v, attendu les 2 matchs filtrés", e)
	}
	if resp.SquadObjectiveHistory != nil {
		t.Errorf("historique d'objectif publié sans composition : %+v", resp.SquadObjectiveHistory)
	}
}

// TestTeammatesService_GetPage_SansRepoLeBlocDitPourquoi — repo non câblé
// (titre sans film.usage_summary) : réponse partielle propre, jamais un 500 — le bloc formes
// dit « non supporté », l'Emprise ne publie que la feuille de match.
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
	if resp.SquadFormes == nil || resp.SquadFormes.Available ||
		resp.SquadFormes.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc formes = %+v, attendu indisponible/unsupported", resp.SquadFormes)
	}
	if resp.SquadEmprise == nil || resp.SquadEmprise.FilmUnavailable != domain.EmpriseFilmUnsupported {
		t.Errorf("Emprise = %+v, attendu film_unsupported", resp.SquadEmprise)
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
// blocs du résumé d'usage (formes retenues, Emprise) publient leur section en ne lisant qu'UNE
// fois les films, les joueurs et les participants du scope.
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
		WithUsageSummary(usage).
		WithSquadFormes(usage, nil, "")
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if resp.SquadFormes == nil || !resp.SquadFormes.Available || resp.SquadEmprise == nil {
		t.Fatalf("les deux blocs doivent être publiés : formes %+v, Emprise %v", resp.SquadFormes, resp.SquadEmprise != nil)
	}
	for _, lecture := range []string{"films", "players", "participants", "pads"} {
		if usage.lectures[lecture] != 1 {
			t.Errorf("lecture %s faite %d fois, attendu 1 (lectures %v)", lecture, usage.lectures[lecture], usage.lectures)
		}
	}
}
