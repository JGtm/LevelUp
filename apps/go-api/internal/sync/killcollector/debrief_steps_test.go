package killcollector

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// TestSignalerPasseFilm_SeulesLesPassesFaites : « morts lues » est capté quand la passe film est
// écrite ou que le film a été lu en entier sans kill-feed — jamais pour une lecture à refaire,
// dont l'instant n'est pas celui de l'étape.
func TestSignalerPasseFilm_SeulesLesPassesFaites(t *testing.T) {
	observability.EnableMatchSteps(true)
	t.Cleanup(func() { observability.EnableMatchSteps(false) })
	_ = observability.DrainMatchSteps()
	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")

	signalerPasseFilm(ctx, "ecrit", OutcomeWritten)
	signalerPasseFilm(ctx, "sans-kf", OutcomeNoKillFeed)
	signalerPasseFilm(ctx, "a-relire", OutcomeSansKillFeedARelire)
	signalerPasseFilm(ctx, "absent", OutcomeNoFilm)

	got := map[string]string{}
	for _, ev := range observability.DrainMatchSteps() {
		if ev.Step != domain.MatchStepKillSourceWritten || ev.TitleSlug != "halo_infinite" {
			t.Fatalf("instant inattendu : %+v", ev)
		}
		got[ev.MatchID] = ev.Detail
	}
	if len(got) != 2 || got["ecrit"] != "" || got["sans-kf"] != "sans_killfeed" {
		t.Fatalf("instants captés : %v, attendu ecrit et sans-kf seulement", got)
	}
}
