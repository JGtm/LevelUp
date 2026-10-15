package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// logsNiveau capture le journal par defaut (tous niveaux) le temps du test.
func logsNiveau(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func assertAucuneErreurJournalisee(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, "level=ERROR") {
			t.Errorf("une annulation ne doit jamais etre journalisee en ERROR : %s", l)
		}
	}
}

// repoQuiAnnule : la requete est abandonnee PENDANT le chargement parallele (le client part
// pendant la lecture du tableau des scores).
type repoQuiAnnule struct {
	*mockMatchViewRepo
	cancel context.CancelFunc
}

func (r repoQuiAnnule) GetMatchScoreboard(ctx context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	r.cancel()
	return nil, fmt.Errorf("lecture du tableau des scores: %w", ctx.Err())
}

// E6.2a-F3 : requete abandonnee pendant le chargement -> l'erreur de contexte est rendue, la
// vue n'est PAS construite (donc aucune raison `_unavailable`, aucun ERROR en aval).
func TestMatchView_AnnulationPendantLeChargement_RendLErreurDeContexte(t *testing.T) {
	logs := logsNiveau(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := repoQuiAnnule{mockMatchViewRepo: repoVueComplete(), cancel: cancel}

	resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(ctx, "m1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur de contexte attendue, obtenu %v", err)
	}
	if len(resp.PartialReasons) != 0 || resp.HasUnavailableSource() {
		t.Errorf("aucune vue ne doit etre construite : raisons %v", resp.PartialReasons)
	}
	assertAucuneErreurJournalisee(t, logs)
}

// Les sites en aval journalisent une annulation en WARN (jamais ERROR).
func TestMatchView_AnnulationEnAval_JournaliseeEnWarn(t *testing.T) {
	annule := fmt.Errorf("lecture: %w", context.Canceled)
	delai := fmt.Errorf("lecture: %w", context.DeadlineExceeded)
	for _, e := range []error{annule, delai} {
		t.Run(e.Error(), func(t *testing.T) {
			t.Run("ouverture de la player DB d'un ami", func(t *testing.T) {
				logs := logsNiveau(t)
				resolver := NewFriendsExtrasResolver(
					map[string]FriendProfile{"x1": {TitleSlug: "halo_infinite", Gamertag: "Ami"}},
					func(context.Context, string, string) (FriendMatchExtrasRepo, error) { return nil, e },
					nil)
				if _, err := resolver(context.Background(), "m1", "", []string{"x1"}); err == nil {
					t.Error("la panne d'ouverture doit rester remontee")
				}
				assertAucuneErreurJournalisee(t, logs)
				if !strings.Contains(logs.String(), "level=WARN") {
					t.Errorf("WARN attendu : %s", logs.String())
				}
			})
			t.Run("lecture des extras d'un ami", func(t *testing.T) {
				logs := logsNiveau(t)
				_, err := loadOneFriendExtras(context.Background(), &mockFriendRepo{enrichErr: e}, "m1", "", "x1", nil)
				if err == nil {
					t.Error("la panne de lecture doit rester remontee")
				}
				assertAucuneErreurJournalisee(t, logs)
				if !strings.Contains(logs.String(), "level=WARN") {
					t.Errorf("WARN attendu : %s", logs.String())
				}
			})
			t.Run("modele personnel d'assistances", func(t *testing.T) {
				logs := logsNiveau(t)
				repo := &mockMatchViewRepo{errs: map[string]error{"GetPlayerAssistsModel": e}}
				computeExpectedAssists(context.Background(), repo, (port.MetadataRepository)(nil), "Slayer",
					&domain.PlayerMatchStatsRaw{})
				assertAucuneErreurJournalisee(t, logs)
				if !strings.Contains(logs.String(), "level=WARN") {
					t.Errorf("WARN attendu : %s", logs.String())
				}
			})
		})
	}
}
