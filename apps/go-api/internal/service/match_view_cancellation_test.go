package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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

// Client parti : les sites en aval ne journalisent JAMAIS une annulation en ERROR (règle
// observability.LevelUnlessCanceled : DEBUG), mais l'erreur reste remontée.
func TestMatchView_ClientParti_PasDErreurJournalisee(t *testing.T) {
	annule := fmt.Errorf("lecture: %w", context.Canceled)
	t.Run("ouverture de la player DB d'un ami", func(t *testing.T) {
		logs := logsNiveau(t)
		resolver := NewFriendsExtrasResolver(
			map[string]FriendProfile{"x1": {TitleSlug: "halo_infinite", Gamertag: "Ami"}},
			func(context.Context, string, string) (FriendMatchExtrasRepo, error) { return nil, annule },
			nil)
		if _, err := resolver(context.Background(), "m1", "", []string{"x1"}); err == nil {
			t.Error("la panne d'ouverture doit rester remontee")
		}
		assertAucuneErreurJournalisee(t, logs)
	})
	t.Run("lecture des extras d'un ami", func(t *testing.T) {
		logs := logsNiveau(t)
		if _, err := loadOneFriendExtras(context.Background(), &mockFriendRepo{enrichErr: annule}, "m1", "", "x1", nil); err == nil {
			t.Error("la panne de lecture doit rester remontee")
		}
		assertAucuneErreurJournalisee(t, logs)
	})
	t.Run("modele personnel d'assistances", func(t *testing.T) {
		logs := logsNiveau(t)
		repo := &mockMatchViewRepo{errs: map[string]error{"GetPlayerAssistsModel": annule}}
		computeExpectedAssists(context.Background(), repo, (port.MetadataRepository)(nil), "Slayer", &domain.PlayerMatchStatsRaw{})
		assertAucuneErreurJournalisee(t, logs)
	})
}

// E6.2b-C1 : un délai INTERNE expiré alors que la requête vit est une vraie panne : ERROR en
// aval, et ERROR + raison `scoreboard_unavailable` pour un chargeur.
func TestMatchView_DelaiInterneExpire_RequeteVivante_EstUnePanne(t *testing.T) {
	delai := fmt.Errorf("lecture: %w", context.DeadlineExceeded)
	t.Run("chargeur de la vue", func(t *testing.T) {
		logs := logsNiveau(t)
		repo := repoVueComplete()
		repo.boardErr = delai
		resp, err := NewMatchViewService(repo, "xuid1").GetMatchView(context.Background(), "m1")
		if err != nil {
			t.Fatalf("la vue est servie en panne partielle : %v", err)
		}
		if !slices.Contains(resp.PartialReasons, "scoreboard_unavailable") {
			t.Errorf("scoreboard_unavailable attendu, obtenu %v", resp.PartialReasons)
		}
		if !strings.Contains(logs.String(), "level=ERROR msg=\"match_view: source indisponible\"") {
			t.Errorf("ERROR « source indisponible » attendu : %s", logs.String())
		}
	})
	t.Run("lecture des extras d'un ami", func(t *testing.T) {
		logs := logsNiveau(t)
		_, _ = loadOneFriendExtras(context.Background(), &mockFriendRepo{enrichErr: delai}, "m1", "", "x1", nil)
		if !strings.Contains(logs.String(), "level=ERROR") {
			t.Errorf("ERROR attendu : %s", logs.String())
		}
	})
	t.Run("modele personnel d'assistances", func(t *testing.T) {
		logs := logsNiveau(t)
		repo := &mockMatchViewRepo{errs: map[string]error{"GetPlayerAssistsModel": delai}}
		computeExpectedAssists(context.Background(), repo, (port.MetadataRepository)(nil), "Slayer", &domain.PlayerMatchStatsRaw{})
		if !strings.Contains(logs.String(), "level=ERROR") {
			t.Errorf("ERROR attendu : %s", logs.String())
		}
	})
}
