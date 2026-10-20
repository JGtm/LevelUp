// Package handlers - match_view_cancellation_e2e_test.go : E6.2a-F3, de bout en bout. Une requete
// abandonnee pendant le chargement de la vue match (vrai service, vrai handler) donne 499
// `client_closed` : ni 200 partiel parseme de raisons `_unavailable`, ni 500.
package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

type repoQuiAnnule struct {
	repoDuelsEnPanne
	cancel context.CancelFunc
}

func (r *repoQuiAnnule) GetMatchScoreboard(ctx context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	r.cancel()
	return nil, fmt.Errorf("tableau des scores: %w", ctx.Err())
}

func TestMatchViewHandler_RequeteAbandonneePendantLeChargement_499(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := &repoQuiAnnule{cancel: cancel}
	factory := func(context.Context, string) (port.MatchViewService, error) {
		return service.NewMatchViewService(repo, "xuid1"), nil
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m1", nil).WithContext(ctx)
	newMatchViewRouter(factory).ServeHTTP(w, req)
	if w.Code != 499 {
		t.Fatalf("499 client_closed attendu, obtenu %d : %s", w.Code, w.Body.String())
	}
}
