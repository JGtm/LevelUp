package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/port"
)

// L9-5 : un refus 4xx ou 5xx ne recopie pas le texte de l'erreur du service ; le client
// reçoit le code et le message fixe de ce code.
func TestClientErrors_DoNotCarryServiceErrorText(t *testing.T) {
	const secret = "open C:/data/titles/halo_infinite/players/JGtm/stats.duckdb: IO Error"
	failing := func(_ context.Context, _ string) (port.AchievementsService, error) {
		return nil, errors.New(secret)
	}
	r := newAchievementsRouter(failing)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/pages/achievements", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("statut = %d, attendu 404 : %s", w.Code, w.Body.String())
	}
	for _, leaked := range []string{"duckdb", "IO Error", "C:/data"} {
		if strings.Contains(w.Body.String(), leaked) {
			t.Errorf("la réponse laisse fuir %q : %s", leaked, w.Body.String())
		}
	}
	if !strings.Contains(w.Body.String(), `"code":"player_not_found"`) ||
		!strings.Contains(w.Body.String(), `"message":"player not found"`) {
		t.Errorf("code ou message fixe attendu, corps : %s", w.Body.String())
	}
}
