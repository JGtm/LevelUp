package service

// timeseries_service_emprise_test.go — L'ONGLET « USAGES » DES SÉRIES TEMPORELLES : l'Emprise sur
// les matchs de la fenêtre, un seul joueur des fiches, chaque source qui dégrade seule (jamais une
// erreur de page), et les lectures du résumé d'usage faites UNE fois pour tous les blocs qui les
// lisent (ADR 0036 I4).

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
	"levelup/go-api/internal/games/canonical"
)

// feuilleFixe — la feuille de match de l'Emprise : des lignes, ou une erreur.
type feuilleFixe struct {
	rows []squademprise.PowerKillRow
	err  error
}

func (f feuilleFixe) LoadPowerWeaponKills(context.Context, []string) ([]squademprise.PowerKillRow, error) {
	return f.rows, f.err
}

// usageCompte — le résumé d'usage (et le grain socles des formes) qui compte ses lectures de films.
type usageCompte struct {
	*mockSessionUsageRepo
	films int
}

func (u *usageCompte) LoadUsageFilms(ctx context.Context, ids []string) (map[string]sessionusage.FilmRow, error) {
	u.films++
	return u.mockSessionUsageRepo.LoadUsageFilms(ctx, ids)
}

// emblemesFixes — le chargeur d'emblèmes.
type emblemesFixes map[string]string

func (e emblemesFixes) LoadEmblemURLs(context.Context, string, []string) map[string]string { return e }

// ligneCanon — un match de la fenêtre sur une carte, avec son résultat.
func ligneCanon(id, carte string, issue canonical.Outcome, minute int) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID: id, StartedAtUTC: time.Date(2026, 7, 28, 20, minute, 0, 0, time.UTC),
			Map: &canonical.AssetReference{ID: "k-" + carte, DefaultLabel: carte},
		},
		Self: canonical.MatchParticipant{Outcome: issue},
	}
}

func fenetreDeTest() []canonical.PlayerMatchRow {
	return []canonical.PlayerMatchRow{
		ligneCanon("m1", "Bazaar", canonical.OutcomeWin, 0),
		ligneCanon("m2", "Bazaar", canonical.OutcomeLoss, 10),
	}
}

func feuilleDeTest() feuilleFixe {
	return feuilleFixe{rows: []squademprise.PowerKillRow{
		{MatchID: "m2", XUID: "P", TeamID: teamp(0), Kills: entierDeTest(3)},
		{MatchID: "m2", XUID: "E1", TeamID: teamp(1), Kills: entierDeTest(1)},
	}}
}

func entierDeTest(v int) *int { return &v }

func serviceEmprise(usage *mockSessionUsageRepo, feuille feuilleFixe) *TimeseriesService {
	svc := NewTimeseriesService(nil).
		WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
		WithHighlightEventsRepo(nil, "P").
		WithEmprise(feuille)
	if usage != nil {
		svc = svc.WithUsageSummary(usage, "")
	}
	return svc
}

func TestAttachEmprise_FenetreUnJoueurCartesEtEquipement(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	serviceEmprise(usageTestRepoMock(), feuilleDeTest()).attachEmprise(context.Background(), &resp, fenetreDeTest(), "fr", nil)
	e := resp.Emprise
	if e == nil || e.MatchesTotal != 2 || e.FilmUnavailable != "" || e.SheetUnavailable != "" {
		t.Fatalf("emprise = %+v, attendu 2 matchs, film et feuille lus", e)
	}
	if len(e.Players) != 1 || e.Players[0].XUID != "P" || e.Players[0].Gamertag != "Papa" {
		t.Errorf("fiches = %+v, attendu le seul joueur consulté (pas de coéquipier sur une page solo)", e.Players)
	}
	if e.Habit != nil || e.Placement != nil {
		t.Errorf("habitude %+v, placement %+v : attendus absents sur la page solo", e.Habit, e.Placement)
	}
	if len(e.Maps) != 1 || e.Maps[0].MapLabel != "Bazaar" || e.Maps[0].Wins != 1 || e.Maps[0].Losses != 1 {
		t.Errorf("grille par carte = %+v, attendu Bazaar, 1 V, 1 D", e.Maps)
	}
	if e.Maps[0].PowerWeaponKills == nil || *e.Maps[0].PowerWeaponKills != (domain.SquadEmpriseCount{Us: 3, Them: 1}) {
		t.Errorf("frags aux armes spéciales de Bazaar = %+v, attendu 3 / 1", e.Maps[0].PowerWeaponKills)
	}
	if e.Equipment == nil || e.Equipment.MatchesMeasured != 1 {
		t.Errorf("équipement = %+v, attendu servi sur le match filmé", e.Equipment)
	}
}

// Halo 5 (titre sans résumé d'usage) : seule la feuille de match, le bloc dit pourquoi.
func TestAttachEmprise_SansFilmSeuleLaFeuille(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	serviceEmprise(nil, feuilleDeTest()).attachEmprise(context.Background(), &resp, fenetreDeTest(), "fr", nil)
	e := resp.Emprise
	if e == nil || e.FilmUnavailable != domain.EmpriseFilmUnsupported || e.Equipment != nil || e.MatchesMeasured != 0 {
		t.Fatalf("emprise = %+v, attendu film_unsupported, pas d'équipement", e)
	}
	if len(e.Production) != 1 || e.Production[0].Resource != domain.EmpriseResourcePowerWeapon {
		t.Errorf("production = %+v, attendu les seuls frags aux armes spéciales", e.Production)
	}
}

// Chaque source dégrade seule, avec sa raison ; jamais une erreur de page.
func TestAttachEmprise_DegradationsNommees(t *testing.T) {
	for _, cas := range []struct {
		nom         string
		usage       *mockSessionUsageRepo
		feuille     feuilleFixe
		film, sheet string
	}{
		{"feuille non supportée", usageTestRepoMock(), feuilleFixe{err: fmt.Errorf("x : %w", games.ErrCapabilityNotSupported)}, "", domain.EmpriseSheetUnsupported},
		{"feuille en échec", usageTestRepoMock(), feuilleFixe{err: errors.New("base indisponible")}, "", domain.EmpriseSheetLoadFailed},
		{"film en échec", &mockSessionUsageRepo{filmsErr: errors.New("base indisponible")}, feuilleDeTest(), domain.EmpriseFilmLoadFailed, ""},
	} {
		var resp domain.TimeseriesPageResponse
		serviceEmprise(cas.usage, cas.feuille).attachEmprise(context.Background(), &resp, fenetreDeTest(), "fr", nil)
		if resp.Emprise == nil || resp.Emprise.FilmUnavailable != cas.film || resp.Emprise.SheetUnavailable != cas.sheet {
			t.Errorf("%s : emprise = %+v, attendu film %q, feuille %q", cas.nom, resp.Emprise, cas.film, cas.sheet)
		}
	}
}

func TestAttachEmprise_FenetreVide(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	serviceEmprise(usageTestRepoMock(), feuilleDeTest()).attachEmprise(context.Background(), &resp, nil, "fr", nil)
	if resp.Emprise != nil {
		t.Errorf("emprise = %+v sur une fenêtre vide, attendu nil", resp.Emprise)
	}
}

// Les lectures du résumé d'usage se font UNE fois pour les formes et l'Emprise.
func TestAttachMigratedSections_UneLectureDuResumeDUsage(t *testing.T) {
	usage := &usageCompte{mockSessionUsageRepo: usageTestRepoMock()}
	svc := NewTimeseriesService(nil).
		WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
		WithHighlightEventsRepo(nil, "P").
		WithEmprise(feuilleDeTest()).
		WithUsageSummary(usage, "").
		WithSquadFormes(usage, nil)
	var resp domain.TimeseriesPageResponse
	svc.attachMigratedSections(context.Background(), &resp, fenetreDeTest(), "fr", equipesDuScope{})
	if usage.films != 1 {
		t.Errorf("%d lecture(s) des films du résumé d'usage, attendu 1 partagée", usage.films)
	}
	if resp.Emprise == nil || resp.Emprise.MatchesMeasured != 1 || resp.SquadFormes == nil {
		t.Errorf("blocs = emprise %+v, formes %v : la lecture partagée doit les nourrir tous",
			resp.Emprise, resp.SquadFormes != nil)
	}
}

func TestAttachEmblem(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	svc := serviceEmprise(nil, feuilleFixe{}).WithEmblemLoader(emblemesFixes{"Papa": "/emblem/papa.png"})
	svc.attachEmblem(context.Background(), &resp)
	if resp.PlayerEmblemURL != "/emblem/papa.png" {
		t.Errorf("emblème = %q, attendu celui du joueur consulté", resp.PlayerEmblemURL)
	}
	var sans domain.TimeseriesPageResponse
	serviceEmprise(nil, feuilleFixe{}).attachEmblem(context.Background(), &sans)
	if sans.PlayerEmblemURL != "" {
		t.Errorf("sans chargeur : %q, attendu vide (initiale côté web)", sans.PlayerEmblemURL)
	}
}
