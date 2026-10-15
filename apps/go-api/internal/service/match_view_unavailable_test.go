package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// repoVueComplete : un repo dont toutes les sources du bandeau sont pleines.
func repoVueComplete() *mockMatchViewRepo {
	now := time.Now()
	bits := pBitMedals
	return &mockMatchViewRepo{
		meta:   &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now},
		stats:  &domain.PlayerMatchStatsRaw{OutcomeCode: 2, BackfillBits: &bits},
		board:  []domain.ScoreboardRaw{{XUID: "xuid1", Gamertag: "Player1"}},
		events: []domain.EventRaw{{EventType: "kill"}},
		medals: []domain.MedalRaw{{MedalID: 1, Count: 1}},
	}
}

// E4 / D-8 : un chargeur EN ERREUR dit `<source>_unavailable` et jamais `<source>_empty` ;
// un chargeur qui RÉUSSIT À VIDE garde `<source>_empty` (sens inchangé). Les codes sont
// écrits en littéral : le test fige le contrat servi au web.
func TestMatchView_ErreurDeChargeurDistincteDuVide(t *testing.T) {
	cas := []struct {
		nom         string
		panne       func(r *mockMatchViewRepo)
		vide        func(r *mockMatchViewRepo)
		unavailable string
		empty       string
	}{
		{"tableau des scores", func(r *mockMatchViewRepo) { r.boardErr = errors.New("lecteur HS") }, func(r *mockMatchViewRepo) { r.board = nil }, "scoreboard_unavailable", "scoreboard_empty"},
		{"evenements", func(r *mockMatchViewRepo) { r.eventsErr = errors.New("lecteur HS") }, func(r *mockMatchViewRepo) { r.events = nil }, "events_unavailable", "events_empty"},
		{"stats du joueur", func(r *mockMatchViewRepo) { r.statsErr = errors.New("lecteur HS") }, func(r *mockMatchViewRepo) { r.stats = nil }, "player_stats_unavailable", "player_stats_empty"},
		{"medailles", func(r *mockMatchViewRepo) { r.medalsErr = errors.New("lecteur HS") }, func(r *mockMatchViewRepo) { r.medals = nil; r.stats.BackfillBits = nil }, "medals_unavailable", "medals_empty"},
	}
	for _, c := range cas {
		t.Run(c.nom+"/erreur", func(t *testing.T) {
			repo := repoVueComplete()
			c.panne(repo)
			resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
			if err != nil {
				t.Fatalf("la panne d'un chargeur ne doit pas faire échouer la vue : %v", err)
			}
			if !slices.Contains(resp.PartialReasons, c.unavailable) {
				t.Errorf("raison %q attendue, obtenu %v", c.unavailable, resp.PartialReasons)
			}
			if slices.Contains(resp.PartialReasons, c.empty) {
				t.Errorf("une lecture en échec ne doit pas aussi dire %q : %v", c.empty, resp.PartialReasons)
			}
			if !resp.IsPartial || !resp.HasUnavailableSource() {
				t.Errorf("vue en échec : IsPartial=%v HasUnavailableSource=%v", resp.IsPartial, resp.HasUnavailableSource())
			}
		})
		t.Run(c.nom+"/vide", func(t *testing.T) {
			repo := repoVueComplete()
			c.vide(repo)
			resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if !slices.Contains(resp.PartialReasons, c.empty) {
				t.Errorf("raison %q attendue, obtenu %v", c.empty, resp.PartialReasons)
			}
			if resp.HasUnavailableSource() {
				t.Errorf("un chargeur vide n'est pas une panne : %v", resp.PartialReasons)
			}
		})
	}
}

// Les deux états simultanés : le tableau des scores en panne ET les événements vides
// donnent `scoreboard_unavailable` + `events_empty` (le web fait primer « indisponible »).
func TestMatchView_PanneEtVideSimultanes(t *testing.T) {
	repo := repoVueComplete()
	repo.boardErr = errors.New("lecteur HS")
	repo.events = nil
	resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := []string{"scoreboard_unavailable", "events_empty"}
	if !slices.Equal(resp.PartialReasons, want) {
		t.Errorf("raisons = %v, attendu %v", resp.PartialReasons, want)
	}
}

// E4 / D-9 : la panne ne survit pas au retour du chargeur. Même service, deux appels ;
// entre les deux le chargeur du tableau des scores est rétabli : la seconde vue est
// COMPLÈTE (tableau présent, aucune raison). Prouve qu'aucun état de la première vue n'est
// réutilisé par le service.
func TestMatchView_LaPanneNeSurvitPasAuRetourDuChargeur(t *testing.T) {
	repo := repoVueComplete()
	repo.boardErr = errors.New("lecteur HS")
	svc := NewMatchViewService(repo, "xuid1")

	premiere, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("1er appel : %v", err)
	}
	if !premiere.HasUnavailableSource() {
		t.Fatalf("1er appel : vue en échec attendue, raisons %v", premiere.PartialReasons)
	}

	repo.boardErr = nil // le chargeur est rétabli
	seconde, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("2e appel : %v", err)
	}
	if seconde.IsPartial || len(seconde.PartialReasons) != 0 {
		t.Errorf("2e appel : vue complète attendue, raisons %v", seconde.PartialReasons)
	}
	if len(seconde.TeamTab.Scoreboard) == 0 {
		t.Errorf("2e appel : le tableau des scores doit être servi")
	}
}

// D-8 : l'erreur est journalisée en ERREUR (source, match_id, err) AVANT la dégradation.
func TestMatchView_ErreurDeChargeurJournaliseeEnErreur(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	repo := repoVueComplete()
	repo.boardErr = errors.New("lecteur HS")
	if _, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1"); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	var ligne string
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, "source=scoreboard") {
			ligne = l
		}
	}
	if ligne == "" {
		t.Fatalf("aucune ligne de journal pour la source scoreboard :\n%s", buf.String())
	}
	for _, f := range []string{"level=ERROR", "match_id=m1", "lecteur HS"} {
		if !strings.Contains(ligne, f) {
			t.Errorf("la ligne %q doit contenir %q", ligne, f)
		}
	}
}
