package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// capturerLogs redirige slog vers un tampon le temps du test.
func capturerLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// E4.8b : les lectures qui avalaient leur erreur remontent ; le service les classe
// `section_unavailable` sans changement de sa part, et la vue reste servie.
func TestMatchView_LecturesDuLotB_EnPanne_SectionUnavailable(t *testing.T) {
	for _, methode := range []string{
		"GetMatchSkillRank", "GetMatchSharedCSRs", "GetMatchMedia", "GetMatchExpectedStats",
		"GetMatchBulkMedals", "GetMatchBulkWeaponKills", "GetHistoryForAvg",
	} {
		t.Run(methode, func(t *testing.T) {
			repo := repoVueComplete()
			repo.errs = map[string]error{methode: errors.New("lecture HS")}
			resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
				t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
			}
			if len(resp.TeamTab.Scoreboard) == 0 {
				t.Errorf("la vue doit rester servie")
			}
		})
	}
}

// Les enrichissements de la page (rang, enrichissement) passent par la même règle ; l'enrichissement
// rend une structure vide quand le match n'a pas de ligne (absence) : aucune raison.
func TestMatchView_EnrichissementEnPanne_SectionUnavailable(t *testing.T) {
	repo := repoVueComplete()
	repo.enrichErr = errors.New("vue HS")
	resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
		t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
	}
}

// Une absence (nil, sans erreur) n'est dite par aucune raison : c'est un résultat.
func TestMatchView_AbsencesDuLotB_AucuneRaison(t *testing.T) {
	resp, err := NewMatchViewService(repoVueComplete(), "xuid1").GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if resp.IsPartial || len(resp.PartialReasons) != 0 {
		t.Errorf("aucune raison attendue pour des absences, obtenu %v", resp.PartialReasons)
	}
}

// Les médailles servies AVEC une SectionDegradedError (libellés illisibles) disent
// `section_unavailable`, pas `medals_unavailable` : les médailles elles-mêmes sont là.
func TestMatchView_LibellesDeMedaillesEnPanne_MedaillesServiesEtPanneDite(t *testing.T) {
	repo := repoVueComplete()
	repo.medalsErr = &domain.SectionDegradedError{Section: "medal_labels", Err: errors.New("referentiel HS")}
	resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
		t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
	}
	if len(resp.SummaryTab.Medals) == 0 {
		t.Errorf("les médailles doivent rester servies")
	}
}

// vueAvecAmi : un tableau avec un ami tracké, un match assez long pour le modèle local.
func vueAvecAmi(repo *mockMatchViewRepo) {
	d := 600.0
	repo.meta.DurationSeconds = &d
	repo.board = []domain.ScoreboardRaw{
		{XUID: "xuid1", Gamertag: "Player1"},
		{XUID: "xuid2", Gamertag: "Friend"},
	}
}

func resolveurAmis(extras map[string]port.FriendMatchExtras, err error) port.FriendsExtrasResolver {
	return func(context.Context, string, string, []string) (map[string]port.FriendMatchExtras, error) {
		return extras, err
	}
}

// Une panne de lecture d'une DB amie est DITE (`section_unavailable`, journal ERREUR) et ne fait pas
// disparaître les extras des autres amis ; sans panne, aucune raison.
func TestMatchView_ExtrasDesAmisEnPanne_SectionUnavailable(t *testing.T) {
	buf := capturerLogs(t)
	repo := repoVueComplete()
	vueAvecAmi(repo)
	panne := resolveurAmis(map[string]port.FriendMatchExtras{"xuid2": {}}, errors.New("player DB de l'ami HS"))
	resp, err := NewMatchViewService(repo, "xuid1").WithFriendsExtras(panne).GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
		t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
	}
	if !strings.Contains(buf.String(), "level=ERROR") || !strings.Contains(buf.String(), "source=friends_extras") {
		t.Errorf("la panne doit être journalisée en ERREUR (source=friends_extras) :\n%s", buf.String())
	}

	ok := resolveurAmis(map[string]port.FriendMatchExtras{"xuid2": {}}, nil)
	resp, err = NewMatchViewService(repo, "xuid1").WithFriendsExtras(ok).GetMatchView(context.Background(), "m1")
	if err != nil || resp.IsPartial {
		t.Errorf("sans panne : vue complète attendue, err=%v raisons=%v", err, resp.PartialReasons)
	}
}

// L'historique des amis (lecture groupée, après le chargement parallèle) en panne est dit aussi.
func TestMatchView_HistoriqueGroupeDesAmisEnPanne_SectionUnavailable(t *testing.T) {
	repo := repoVueComplete()
	vueAvecAmi(repo)
	repo.errs = map[string]error{"GetHistoryForAvgBulk": errors.New("historique HS")}
	suivi := resolveurAmis(map[string]port.FriendMatchExtras{"xuid2": {}}, nil)
	resp, err := NewMatchViewService(repo, "xuid1").WithFriendsExtras(suivi).GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
		t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
	}
}

// Le modèle personnel d'assists attendus illisible est journalisé en ERREUR avant le repli
// populationnel ; la valeur de repli reste servie, aucune section n'est marquée.
func TestMatchView_ModeleDAssistsIllisible_JournaliseEnErreur(t *testing.T) {
	buf := capturerLogs(t)
	repo := repoVueComplete()
	gvn := "Slayer"
	repo.meta.GameVariantName = &gvn
	repo.errs = map[string]error{"GetPlayerAssistsModel": errors.New("modele HS")}
	if _, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1"); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !strings.Contains(buf.String(), "level=ERROR") || !strings.Contains(buf.String(), "modele HS") {
		t.Errorf("la panne du modèle doit être journalisée en ERREUR :\n%s", buf.String())
	}
}
