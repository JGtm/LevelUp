package teammates

// teammates_service_emprise_vehicles_test.go — la ressource « véhicules » de l'Emprise publiée par
// GetPage (plan `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3), avec un dépôt simulé : une
// lecture par requête bornée par les matchs du périmètre (ADR 0036 I4), indépendante du film, et les
// dégradations (capability absente, dépôt non supporté, lecture en échec) — la page ne tombe jamais.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakeVehicules — le dépôt simulé : rend `read` / `err` et note chaque appel.
type fakeVehicules struct {
	read   squademprise.VehicleRead
	err    error
	appels [][]string
	xuids  []string
}

func (f *fakeVehicules) LoadVehicleUsage(_ context.Context, matchIDs []string, xuid string) (squademprise.VehicleRead, error) {
	f.appels = append(f.appels, append([]string(nil), matchIDs...))
	f.xuids = append(f.xuids, xuid)
	return f.read, f.err
}

var _ port.SquadVehicleRepository = (*fakeVehicules)(nil)

// vehiculesM1 : m1 mesuré, le joueur au camp 0 prend deux Warthog, un adversaire un Banshee.
func vehiculesM1() squademprise.VehicleRead {
	return squademprise.VehicleRead{
		Passes: []squademprise.VehiclePass{{MatchID: "m1", Measured: true, FragsRead: true}},
		Rows: []squademprise.VehicleRow{
			{MatchID: "m1", Camp: 0, XUID: "player-xuid", Family: "warthog", Takes: 2, AboardMS: 60_000},
			{MatchID: "m1", Camp: 1, XUID: "e1", Family: "banshee", Takes: 1, AboardMS: 30_000},
		},
		EventsRead: map[string]bool{},
		PlayerTeam: map[string]int{"m1": 0},
	}
}

// pageVehicules — GetPage sur le périmètre D2 de l'Emprise (m1, Main + Ally1), avec ou sans film,
// ressource câblée ou non (`repo` nil).
func pageVehicules(t *testing.T, avecFilm bool, repo port.SquadVehicleRepository) *domain.SquadEmpriseBlock {
	t.Helper()
	squad, usage := usageFixture(time.Now().UTC().Add(-time.Hour))
	svc := NewTeammatesService(squad, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(squad.synthRows, squad.synthErr), "halo_infinite", "Main").
		WithEmprise(feuilleM1())
	if avecFilm {
		svc = svc.WithUsageSummary(usage)
	}
	if repo != nil {
		svc = svc.WithVehicleUsage(repo)
	}
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("GetPage : %v — la ressource véhicules ne doit jamais faire tomber la page", err)
	}
	if resp.SquadEmprise == nil {
		t.Fatal("bloc Emprise absent")
	}
	return resp.SquadEmprise
}

func ressourceDuBloc(b *domain.SquadEmpriseBlock, res string) *domain.SquadEmpriseResource {
	for i := range b.Resources {
		if b.Resources[i].Resource == res {
			return &b.Resources[i]
		}
	}
	return nil
}

func TestGetPage_Vehicules_UneLectureBorneeParLePerimetre(t *testing.T) {
	repo := &fakeVehicules{read: vehiculesM1()}
	b := pageVehicules(t, true, repo)
	if len(repo.appels) != 1 {
		t.Fatalf("%d lecture(s) des véhicules, attendu une par requête", len(repo.appels))
	}
	if fmt.Sprint(repo.appels[0]) != "[m1]" || repo.xuids[0] != "player-xuid" {
		t.Errorf("lecture bornée par %v pour %q, attendu le périmètre [m1] et le joueur de la page", repo.appels[0], repo.xuids[0])
	}
	r := ressourceDuBloc(b, domain.EmpriseResourceVehicle)
	if r == nil || r.Taken != (domain.SquadEmpriseCount{Us: 2, Them: 1}) {
		t.Fatalf("ressource véhicules = %+v, attendu 2 / 1", r)
	}
	if b.Vehicles == nil || b.Vehicles.MatchesMeasured != 1 || b.Matches[0].Vehicles != domain.EmpriseVehiclesMeasured {
		t.Errorf("couverture = %+v, match = %q", b.Vehicles, b.Matches[0].Vehicles)
	}
}

// TestGetPage_Vehicules_IndependantsDuFilm — la ressource vient de l'artefact : sans résumé d'usage
// (titre sans `film.usage_summary`), elle se publie.
func TestGetPage_Vehicules_IndependantsDuFilm(t *testing.T) {
	b := pageVehicules(t, false, &fakeVehicules{read: vehiculesM1()})
	if b.FilmUnavailable != domain.EmpriseFilmUnsupported {
		t.Fatalf("le film devait être absent : %+v", b)
	}
	if r := ressourceDuBloc(b, domain.EmpriseResourceVehicle); r == nil || r.Taken.Us != 2 {
		t.Errorf("véhicules sans film = %+v, attendu publiés", r)
	}
}

// Capability absente : le lecteur n'est pas câblé (porte `film.vehicle_usage` du câblage) —
// ressource absente, ErrCapabilityNotSupported journalisé en Debug, rien d'autre ne bouge.
func TestGetPage_Vehicules_CapabilityAbsente(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	logs := withCapturedLogs(t, func() { b = pageVehicules(t, true, nil) })
	if b.Vehicles != nil || ressourceDuBloc(b, domain.EmpriseResourceVehicle) != nil || b.Matches[0].Vehicles != "" {
		t.Errorf("ressource publiée sans capability : %+v", b.Vehicles)
	}
	l := ligneDeLog(logs, "emprise_vehicules_capability_absente")
	if !strings.Contains(l, `"level":"DEBUG"`) || !strings.Contains(l, games.ErrCapabilityNotSupported.Error()) {
		t.Errorf("journal attendu en Debug avec ErrCapabilityNotSupported, obtenu : %q", l)
	}
	if b.MatchesMeasured != 1 {
		t.Errorf("le reste de l'Emprise doit rester servi : %+v", b)
	}
}

// Le dépôt dit la capability absente (table manquante) : même dégradation.
func TestGetPage_Vehicules_DepotNonSupporte(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	repo := &fakeVehicules{err: fmt.Errorf("lecture : %w", games.ErrCapabilityNotSupported)}
	logs := withCapturedLogs(t, func() { b = pageVehicules(t, true, repo) })
	l := ligneDeLog(logs, "emprise_vehicules_capability_absente")
	if b.Vehicles != nil || !strings.Contains(l, `"level":"DEBUG"`) {
		t.Errorf("couverture %+v, journal %q ; attendu absente et Debug", b.Vehicles, l)
	}
}

// Lecture en échec : ressource absente ET dite (jamais un zéro), journalisé en Error, la page reste servie.
func TestGetPage_Vehicules_LectureEnEchec(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	logs := withCapturedLogs(t, func() { b = pageVehicules(t, true, &fakeVehicules{err: errors.New("base indisponible")}) })
	l := ligneDeLog(logs, "emprise_vehicules_en_echec")
	if !strings.Contains(l, `"level":"ERROR"`) || !strings.Contains(l, "base indisponible") || !strings.Contains(l, `"page":"teammates"`) {
		t.Errorf("journal attendu en Error avec la cause, obtenu : %q", l)
	}
	if b.Vehicles == nil || b.Vehicles.Unavailable != domain.EmpriseVehiclesLoadFailed ||
		ressourceDuBloc(b, domain.EmpriseResourceVehicle) != nil {
		t.Errorf("couverture = %+v, attendu l'échec dit et aucune ressource", b.Vehicles)
	}
	if b.MatchesMeasured != 1 {
		t.Errorf("le reste de l'Emprise doit rester servi : %+v", b)
	}
}

// TestGetPage_Vehicules_LaLectureInclutLesSoireesDeLHabitude — revue L7.5, RV5 : les matchs des
// soirées précédentes comparables (`squademprise.HabitCandidates`) entrent dans la lecture, EN PLUS
// du périmètre, sans quoi l'habitude des véhicules se calcule sur rien. Le match `h1` d'une soirée
// antérieure, de la même famille de mode que `m1`, doit donc figurer dans l'appel du dépôt.
func TestGetPage_Vehicules_LaLectureInclutLesSoireesDeLHabitude(t *testing.T) {
	t0 := time.Now().UTC().Add(-time.Hour)
	squad, usage := usageFixture(t0)
	cur, prev := "soiree-courante", "soiree-precedente"
	squad.squadRows = []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin, PairName: "Slayer", SessionLabel: &cur},
		{MatchID: "h1", StartTime: t0.Add(-48 * time.Hour), Outcome: domain.OutcomeWin, PairName: "Slayer", SessionLabel: &prev},
	}
	repo := &fakeVehicules{read: vehiculesM1()}
	svc := NewTeammatesService(squad, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(squad.synthRows, squad.synthErr), "halo_infinite", "Main").
		WithEmprise(feuilleM1()).WithUsageSummary(usage).WithVehicleUsage(repo)
	if _, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	}); err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if len(repo.appels) != 1 {
		t.Fatalf("%d lecture(s) des véhicules, attendu une", len(repo.appels))
	}
	if got := fmt.Sprint(repo.appels[0]); got != "[m1 h1]" {
		t.Errorf("lecture bornée par %s, attendu le périmètre puis la soirée de l'habitude [m1 h1]", got)
	}
}
