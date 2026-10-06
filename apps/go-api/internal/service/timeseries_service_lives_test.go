package service

// timeseries_service_lives_test.go — « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL » : une lecture
// bornée par la fenêtre et le joueur, la portée COURANTE du radar par match, et chaque
// dégradation sans erreur de page (bloc absent, journalisé).

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// viesFixes — le dépôt des vies : une lecture ou une erreur ; note ce qu'on lui demande.
type viesFixes struct {
	lues   domain.ViesLues
	err    error
	appels int
	ids    []string
	xuid   string
}

func (v *viesFixes) LoadLivesNearTeammate(_ context.Context, ids []string, xuid string) (domain.ViesLues, error) {
	v.appels++
	v.ids, v.xuid = ids, xuid
	return v.lues, v.err
}

func distanceDeTest(m float64) *float64 { return &m }

// viesDeTest : m1 (Slayer:Arena, 18 m) une vie près (10 m, 1 frag) ; m2 (variante sans portée)
// une vie, écartée.
func viesDeTest() domain.ViesLues {
	zero, un := 0, 1
	return domain.ViesLues{
		Vies: []domain.VieLue{
			{MatchID: "m1", StartMS: 0, EndCause: coordination.CauseVieMort},
			{MatchID: "m2", StartMS: 0, EndCause: coordination.CauseVieMort},
		},
		Morts: []domain.MortSituee{
			{MatchID: "m1", TimeMS: 30_000, PlusProcheM: distanceDeTest(10)},
			{MatchID: "m2", TimeMS: 30_000, PlusProcheM: distanceDeTest(10)},
		},
		Frags:     []domain.FragLu{{MatchID: "m1", TimeMS: 10_000, CampTueur: &zero, CampVictime: &un}},
		Variantes: map[string]string{"m1": " Slayer:Arena", "m2": "Husky Raid:CTF"},
	}
}

func serviceVies(repo *viesFixes, radar map[string]int) *TimeseriesService {
	svc := NewTimeseriesService(nil).
		WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
		WithHighlightEventsRepo(nil, "P").
		WithRadarRange(radar)
	if repo != nil {
		svc = svc.WithLivesNearTeammate(repo)
	}
	return svc
}

func TestAttachLives_LectureBorneeEtPorteeCourante(t *testing.T) {
	repo := &viesFixes{lues: viesDeTest()}
	var resp domain.TimeseriesPageResponse
	serviceVies(repo, map[string]int{"Slayer:Arena": 18}).attachLives(context.Background(), &resp, fenetreDeTest())
	if repo.appels != 1 || !slices.Equal(repo.ids, []string{"m1", "m2"}) || repo.xuid != "P" {
		t.Errorf("lecture : %d appel(s) sur %v pour %q, attendu une lecture des matchs de la fenêtre pour P",
			repo.appels, repo.ids, repo.xuid)
	}
	want := domain.TimeseriesLivesNearTeammate{
		Near: domain.LivesSideCount{Lives: 1, Kills: 1}, ExcludedNoRadar: 1, MatchesRead: 2, MatchesWithoutRadar: 1,
	}
	if resp.LivesNearTeammate == nil || *resp.LivesNearTeammate != want {
		t.Errorf("vies = %+v, attendu %+v", resp.LivesNearTeammate, want)
	}
}

// Sans table des portées : toutes les vies sont écartées ET comptées, le bloc le dit.
func TestAttachLives_SansPorteeToutEstEcarteEtCompte(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	serviceVies(&viesFixes{lues: viesDeTest()}, nil).attachLives(context.Background(), &resp, fenetreDeTest())
	l := resp.LivesNearTeammate
	if l == nil || l.ExcludedNoRadar != 2 || l.Near.Lives+l.Alone.Lives != 0 || l.MatchesWithoutRadar != 2 {
		t.Errorf("vies = %+v, attendu 2 écartées faute de portée, rien de rangé", l)
	}
}

// Capability absente, dépôt non supporté, lecture en échec, aucune vie : pas de bloc, jamais
// d'erreur de page.
func TestAttachLives_Degradations(t *testing.T) {
	for _, cas := range []struct {
		nom  string
		repo *viesFixes
	}{
		{"capability absente", nil},
		{"dépôt non supporté", &viesFixes{err: fmt.Errorf("x : %w", games.ErrCapabilityNotSupported)}},
		{"lecture en échec", &viesFixes{err: errors.New("base indisponible")}},
		{"aucune vie", &viesFixes{}},
	} {
		var resp domain.TimeseriesPageResponse
		serviceVies(cas.repo, map[string]int{"Slayer:Arena": 18}).attachLives(context.Background(), &resp, fenetreDeTest())
		if resp.LivesNearTeammate != nil {
			t.Errorf("%s : bloc %+v, attendu absent", cas.nom, resp.LivesNearTeammate)
		}
	}
}

func TestAttachLives_FenetreVideNeLitRien(t *testing.T) {
	repo := &viesFixes{lues: viesDeTest()}
	var resp domain.TimeseriesPageResponse
	serviceVies(repo, nil).attachLives(context.Background(), &resp, nil)
	if repo.appels != 0 || resp.LivesNearTeammate != nil {
		t.Errorf("%d lecture(s), bloc %+v : attendu aucune lecture sur une fenêtre vide", repo.appels, resp.LivesNearTeammate)
	}
}

// La page pose le bloc : attachMigratedSections appelle la lecture des vies.
func TestAttachMigratedSections_PoseLesVies(t *testing.T) {
	var resp domain.TimeseriesPageResponse
	serviceVies(&viesFixes{lues: viesDeTest()}, map[string]int{"Slayer:Arena": 18}).
		attachMigratedSections(context.Background(), &resp, fenetreDeTest(), "fr", equipesDuScope{})
	if resp.LivesNearTeammate == nil || resp.LivesNearTeammate.Near.Lives != 1 {
		t.Errorf("vies = %+v, attendu le bloc posé par la page", resp.LivesNearTeammate)
	}
}
