package teammates

// teammates_service_sections_test.go — lot perf L2, D2.7 : une requête annulée entre deux
// sections de GetPage rend l'erreur du contexte, jamais une page partielle, et ne lance plus
// aucune section — ni lecture ni calcul — pour un client parti.

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// annulation annule la requête pendant la n-ième lecture d'une méthode donnée, et compte les
// lectures vues par méthode.
type annulation struct {
	mu      sync.Mutex
	methode string
	rang    int
	annuler context.CancelFunc
	vues    map[string]int
}

func (a *annulation) lecture(methode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.vues[methode]++
	if methode == a.methode && a.vues[methode] == a.rang {
		a.annuler()
	}
}

// repoAnnulant : le lecteur Escouade de la fixture, qui annule la requête au point choisi.
type repoAnnulant struct {
	*countingSquadRepo
	a *annulation
}

func (r repoAnnulant) LoadTopTeammates(ctx context.Context, x string) ([]domain.TopTeammateRow, error) {
	r.a.lecture("LoadTopTeammates")
	return r.countingSquadRepo.LoadTopTeammates(ctx, x)
}

func (r repoAnnulant) LoadSquadMatches(ctx context.Context, x, tm string) ([]domain.SquadMatchRow, error) {
	r.a.lecture("LoadSquadMatches")
	return r.countingSquadRepo.LoadSquadMatches(ctx, x, tm)
}

func (r repoAnnulant) LoadMainTeamParticipants(ctx context.Context, x string, ids []string) ([]domain.AllyParticipant, error) {
	r.a.lecture("LoadMainTeamParticipants")
	return r.countingSquadRepo.LoadMainTeamParticipants(ctx, x, ids)
}

func (r repoAnnulant) LoadMapStatsForSquad(
	ctx context.Context, x string, squad, exclus []string,
) (map[string]domain.MapSquadStats, error) {
	r.a.lecture("LoadMapStatsForSquad")
	return r.countingSquadRepo.LoadMapStatsForSquad(ctx, x, squad, exclus)
}

// loaderAnnulant : le chargeur d'historiques de la fixture, idem.
type loaderAnnulant struct {
	*fakeSquadLoader
	a *annulation
}

func (l loaderAnnulant) LoadFor(
	ctx context.Context, slug, gt string, f port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	l.a.lecture("LoadFor")
	return l.fakeSquadLoader.LoadFor(ctx, slug, gt, f)
}

// usageAnnulant : le résumé d'usage, idem — sa première lecture (les films) compte sous « usage ».
type usageAnnulant struct {
	*mockTeammatesUsageRepo
	a *annulation
}

func (u usageAnnulant) LoadUsageFilms(ctx context.Context, ids []string) (map[string]sessionusage.FilmRow, error) {
	u.a.lecture("usage")
	return u.mockTeammatesUsageRepo.LoadUsageFilms(ctx, ids)
}

// serviceAnnulable : la page complète de la fixture (plus un second coéquipier, Bis), dont
// chaque lecteur signale ses lectures à l'annulation.
func serviceAnnulable(a *annulation) *TeammatesService {
	repo, loader := pageCompleteFixture()
	repo.topRows = append(repo.topRows, domain.TopTeammateRow{XUID: "x_bis", Gamertag: "Bis", GamesTogether: 3})
	return NewTeammatesService(repoAnnulant{countingSquadRepo: repo, a: a}, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, nil), "halo_infinite", "Main").
		WithSquadLoader(loaderAnnulant{fakeSquadLoader: loader, a: a}).
		WithEquipmentUsage(usageAnnulant{mockTeammatesUsageRepo: &mockTeammatesUsageRepo{}, a: a})
}

// sectionsVues : les sections de durée ouvertes par la requête.
func sectionsVues(tm *timing.Timings) map[string]bool {
	vues := map[string]bool{}
	for _, st := range tm.Snapshot() {
		vues[st.Name] = true
	}
	return vues
}

// TestGetPage_AnnuleeEntreDeuxSections (D2.7) : à chaque point d'annulation, GetPage rend
// l'erreur du contexte et une réponse vide ; la section en cours s'achève, aucune suivante ne
// démarre (section de durée absente), et la lecture en cours n'est pas refaite pour le membre
// ou le coéquipier suivant.
func TestGetPage_AnnuleeEntreDeuxSections(t *testing.T) {
	cas := []struct {
		nom, methode string
		selection    []string
		// enCours : la section pendant laquelle la requête meurt ; absentes : des sections
		// qui la suivent et ne doivent pas avoir démarré.
		enCours  string
		absentes []string
	}{
		{"avant l'historique du joueur", "LoadTopTeammates", []string{"Ally"},
			"top_teammates", []string{"player_matches", "teammate_rows"}},
		{"entre deux coéquipiers", "LoadSquadMatches", []string{"Ally", "Bis"},
			"teammate_rows", []string{"main_team_allies", "squad_members"}},
		{"après le dernier coéquipier", "LoadSquadMatches", []string{"Ally"},
			"teammate_rows", []string{"main_team_allies", "squad_members"}},
		{"avant le préchargement", "LoadMainTeamParticipants", []string{"Ally"},
			"main_team_allies", []string{"squad_members", "impact_events_shared", "enrich_assets"}},
		{"pendant le préchargement des membres", "LoadFor", []string{"Ally"},
			"squad_members", []string{"impact_events_shared", "enrich_assets", "map_stats"}},
		{"pendant une section de la population", "LoadMapStatsForSquad", []string{"Ally"},
			"map_stats", []string{"match_history", "session_timeline", "map_heatmap", "impact_matrix",
				"medal_digest", "briefing_header", "composition_sessions", "usage_shared"}},
		{"pendant les lectures d'usage", "usage", []string{"Ally"},
			"usage_shared", []string{"equipment_usage", "squad_formes"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ctx, annuler := context.WithCancel(context.Background())
			defer annuler()
			ctx, tm := timing.WithTimings(ctx)
			a := &annulation{methode: c.methode, rang: 1, annuler: annuler, vues: map[string]int{}}

			resp, err := serviceAnnulable(a).GetPage(ctx, "x_main",
				domain.TeammatesQueryRequest{SelectedGamertags: c.selection})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("erreur = %v, attendu l'annulation du contexte", err)
			}
			if !reflect.ValueOf(resp).IsZero() {
				t.Error("une requête annulée ne rend pas de page partielle")
			}
			if n := a.vues[c.methode]; n != 1 {
				t.Errorf("%s lu %d fois, attendu 1 (rien de relu après l'annulation)", c.methode, n)
			}
			vues := sectionsVues(tm)
			if !vues[c.enCours] {
				t.Fatalf("section %q absente : l'annulation n'est pas tombée où le cas l'attend (%v)", c.enCours, vues)
			}
			for _, nom := range c.absentes {
				if vues[nom] {
					t.Errorf("section %q lancée après l'annulation", nom)
				}
			}
		})
	}
}

// TestGetPage_NonAnnuleeToutesLesSections : témoin des cas ci-dessus — sans annulation, la
// même page lance bien chacune des sections qu'ils déclarent absentes.
func TestGetPage_NonAnnuleeToutesLesSections(t *testing.T) {
	ctx, tm := timing.WithTimings(context.Background())
	a := &annulation{vues: map[string]int{}}
	if _, err := serviceAnnulable(a).GetPage(ctx, "x_main",
		domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally", "Bis"}}); err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	vues := sectionsVues(tm)
	for _, nom := range []string{
		"top_teammates", "player_matches", "teammate_rows", "main_team_allies", "squad_members",
		"impact_events_shared", "enrich_assets", "map_stats", "match_history", "session_timeline",
		"map_heatmap", "impact_matrix", "medal_digest", "briefing_header",
		"composition_sessions", "usage_shared", "equipment_usage", "squad_formes",
	} {
		if !vues[nom] {
			t.Errorf("section %q absente d'une page non annulée", nom)
		}
	}
	if a.vues["LoadSquadMatches"] != 2 || a.vues["LoadFor"] != 3 {
		t.Errorf("lectures = %v, attendu deux coéquipiers et trois membres lus", a.vues)
	}
}

// TestLoadUsageBlocks_RequeteDejaAnnulee : les blocs d'usage ne lisent rien pour une requête
// déjà annulée.
func TestLoadUsageBlocks_RequeteDejaAnnulee(t *testing.T) {
	ctx, annuler := context.WithCancel(context.Background())
	annuler()
	a := &annulation{vues: map[string]int{}}
	repo, _ := pageCompleteFixture()
	svc := NewTeammatesService(repo, nil).
		WithEquipmentUsage(usageAnnulant{mockTeammatesUsageRepo: &mockTeammatesUsageRepo{}, a: a})
	equipement, formes := svc.loadUsageBlocks(ctx, "x_main", repo.synthRows, nil, domain.TeammatesQueryRequest{})
	if equipement != nil || formes != nil || a.vues["usage"] != 0 {
		t.Fatalf("blocs %v / %v, lectures %d : attendu aucun bloc et aucune lecture", equipement, formes, a.vues["usage"])
	}
}
