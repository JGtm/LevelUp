package teammates

// teammates_service_emprise_test.go — le bloc « Emprise » (lot L4 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) publié par GetPage : nominal, et les chemins
// dégradés de L4.4 (titre sans film, capability non supportée, lecture en échec) — chaque
// source dégrade seule, la page ne tombe jamais.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakeFeuilleEmprise — la feuille de match (frags aux armes spéciales).
type fakeFeuilleEmprise struct {
	rows []squademprise.PowerKillRow
	err  error
}

func (f *fakeFeuilleEmprise) LoadPowerWeaponKills(_ context.Context, _ []string) ([]squademprise.PowerKillRow, error) {
	return f.rows, f.err
}

var _ port.SquadEmpriseRepository = (*fakeFeuilleEmprise)(nil)

// filmEnErreur — un résumé d'usage dont la lecture des films échoue.
type filmEnErreur struct {
	mockTeammatesUsageRepo
	err error
}

func (f *filmEnErreur) LoadUsageFilms(_ context.Context, _ []string) (map[string]sessionusage.FilmRow, error) {
	return nil, f.err
}

func feuilleM1() *fakeFeuilleEmprise {
	k := func(v int) *int { return &v }
	return &fakeFeuilleEmprise{rows: []squademprise.PowerKillRow{
		{MatchID: "m1", XUID: "player-xuid", TeamID: teammatesUsageTeam(0), Kills: k(4)},
		{MatchID: "m1", XUID: "x1", TeamID: teammatesUsageTeam(0), Kills: k(1)},
		{MatchID: "m1", XUID: "e1", TeamID: teammatesUsageTeam(1), Kills: k(3)},
	}}
}

func pageEmprise(t *testing.T, usage port.SessionUsageRepository, feuille port.SquadEmpriseRepository) *domain.SquadEmpriseBlock {
	t.Helper()
	repo, _ := usageFixture(time.Now().UTC().Add(-time.Hour))
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Main")
	if usage != nil {
		svc = svc.WithUsageSummary(usage)
	}
	if feuille != nil {
		svc = svc.WithEmprise(feuille)
	}
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("GetPage : %v — une source de l'Emprise en panne ne doit jamais faire tomber la page", err)
	}
	if resp.SquadEmprise == nil {
		t.Fatal("bloc Emprise absent alors que le périmètre a un match")
	}
	return resp.SquadEmprise
}

func TestTeammatesService_GetPage_PublieLEmprise(t *testing.T) {
	t.Parallel()
	_, usage := usageFixture(time.Now().UTC().Add(-time.Hour))
	b := pageEmprise(t, usage, feuilleM1())
	if b.MatchesTotal != 1 || b.MatchesMeasured != 1 || b.FilmUnavailable != "" || b.SheetUnavailable != "" {
		t.Fatalf("bloc = %+v, attendu le périmètre D2 (m1), filmé, feuille lue", b)
	}
	if len(b.Players) != 2 || b.Players[0].Gamertag != "Main" || b.Players[1].Gamertag != "Ally1" {
		t.Errorf("joueurs des fiches = %+v, attendu Main puis Ally1", b.Players)
	}
	if pwk := b.Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{Us: 5, Them: 3}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 5 / 3", pwk)
	}
	if b.Habit == nil {
		t.Error("habitude absente alors qu'un coéquipier est sélectionné")
	}
}

// TestTeammatesService_GetPage_EmpriseSansFilm — D10 : un titre sans `film.usage_summary` (le
// résumé d'usage n'est pas câblé) ne publie que la feuille de match.
func TestTeammatesService_GetPage_EmpriseSansFilm(t *testing.T) {
	t.Parallel()
	b := pageEmprise(t, nil, feuilleM1())
	if b.FilmUnavailable != domain.EmpriseFilmUnsupported || b.MatchesMeasured != 0 || b.Habit != nil {
		t.Fatalf("bloc = %+v, attendu film non supporté, sans habitude", b)
	}
	if len(b.Resources) != 0 || len(b.Objects) != 0 {
		t.Errorf("grandeurs du film publiées sans film : %+v / %+v", b.Resources, b.Objects)
	}
	if len(b.Production) != 1 || b.Production[0].Resource != domain.EmpriseResourcePowerWeapon {
		t.Errorf("production = %+v, attendu les frags aux armes spéciales seuls", b.Production)
	}
}

// TestTeammatesService_GetPage_EmpriseCapabilityNonSupportee — ErrCapabilityNotSupported, d'un
// côté ou de l'autre, retire sa source et garde l'autre.
func TestTeammatesService_GetPage_EmpriseCapabilityNonSupportee(t *testing.T) {
	t.Parallel()
	pasLa := fmt.Errorf("lecture : %w", games.ErrCapabilityNotSupported)
	_, usage := usageFixture(time.Now().UTC().Add(-time.Hour))
	b := pageEmprise(t, usage, &fakeFeuilleEmprise{err: pasLa})
	if b.SheetUnavailable != domain.EmpriseSheetUnsupported || b.Matches[0].PowerWeaponKills != nil {
		t.Errorf("feuille non supportée : %+v", b)
	}
	if b.MatchesMeasured != 1 {
		t.Errorf("le film doit rester servi : %+v", b)
	}
	b = pageEmprise(t, &filmEnErreur{mockTeammatesUsageRepo: *usage, err: pasLa}, feuilleM1())
	if b.FilmUnavailable != domain.EmpriseFilmUnsupported || b.Matches[0].PowerWeaponKills == nil {
		t.Errorf("film non supporté : %+v, attendu la feuille seule", b)
	}
}

// TestTeammatesService_GetPage_EmpriseLecturesEnEchec — une lecture en échec est dite, jamais
// confondue avec « non supporté », et la page reste servie.
func TestTeammatesService_GetPage_EmpriseLecturesEnEchec(t *testing.T) {
	t.Parallel()
	panne := errors.New("base indisponible")
	_, usage := usageFixture(time.Now().UTC().Add(-time.Hour))
	b := pageEmprise(t, &filmEnErreur{mockTeammatesUsageRepo: *usage, err: panne}, &fakeFeuilleEmprise{err: panne})
	if b.FilmUnavailable != domain.EmpriseFilmLoadFailed || b.SheetUnavailable != domain.EmpriseSheetLoadFailed {
		t.Errorf("raisons = %q / %q, attendu les deux lectures en échec", b.FilmUnavailable, b.SheetUnavailable)
	}
	if b.MatchesTotal != 1 || len(b.Matches) != 1 {
		t.Errorf("le périmètre doit rester publié : %+v", b)
	}
}
