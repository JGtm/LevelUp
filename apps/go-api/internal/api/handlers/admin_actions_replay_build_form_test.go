package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// replayBuildFormTraps : identifiants qui n'ont pas la forme d'un identifiant de match
// (title.IsValidMatchID) ; chacun deviendrait un segment de chemin du rejeu.
var replayBuildFormTraps = []string{
	"..", "../../etc/passwd", `a\b`, "a/b", "zz", "000d5950 ", "000d5950%2F", "\x00",
	strings.Repeat("a", 65),
}

// L'action admin de cuisson (construction locale et mise en file) refuse en 400
// invalid_match_id un identifiant hors forme, avant de créer un job ou de toucher la file.
func TestAdminReplayBuild_MatchIDOutsideForm(t *testing.T) {
	for _, id := range replayBuildFormTraps {
		body, _ := json.Marshal(map[string]string{"match_id": id})
		for _, route := range []string{"run", "enqueue"} {
			t.Run(route+"/"+id, func(t *testing.T) {
				store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
				ran, queued := false, false
				h := NewAdminReplayBuildActionHandler(func(context.Context, string, string) (map[string]any, error) {
					ran = true
					return nil, nil
				}, store, context.Background()).WithEnqueuer(func(context.Context, string, string) (domain.BuildQueueJob, bool, error) {
					queued = true
					return domain.BuildQueueJob{}, true, nil
				})
				req := httptest.NewRequest(http.MethodPost, "/admin/actions/replay-build/"+route, strings.NewReader(string(body)))
				req.Header.Set("Content-Type", "application/json")
				rec := serveAdminReplayBuild(h, req)

				if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_match_id"`) {
					t.Fatalf("status = %d body=%s ; attendu 400 invalid_match_id", rec.Code, rec.Body.String())
				}
				if queued || ran || len(store.List(0)) != 0 {
					t.Fatalf("identifiant hors forme : job créé ou mis en file (run=%v, file=%v, jobs=%d)", ran, queued, len(store.List(0)))
				}
			})
		}
	}
}
