package wire

// replay_cadence_http_test.go — TÉMOIN de bout en bout de la réponse HTTP du document de rejeu :
// vrai service, vrai handler, vrai fichier d'artefact. Il fige les en-têtes et le corps d'un document
// au pas de cuisson courant et d'un document à un ancien pas, pour que la décision de fraîcheur de
// cadence puisse changer de couche sans changer un octet de la réponse.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/service"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func TestReplayHTTP_CadenceCourantePuisAncienPas(t *testing.T) {
	const matchID = "c0de0c01"
	cas := []struct {
		nom     string
		courant int
		pas     int
		stale   string
	}{
		{"document au pas courant", 33, 33, "false"},
		{"document a un ancien pas", 33, 100, "true"},
		{"document sans pas, lu a 100 ms", 33, 0, "true"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			defer replaybuild.ForceCookFrameIntervalMS(c.courant)()
			root := t.TempDir()
			doc := `{"schemaVersion":115,"matchId":"` + matchID + `","titleSlug":"` + title.DefaultSlug + `",` +
				`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[]`
			if c.pas != 0 {
				doc += `,"frameIntervalMs":` + strconv.Itoa(c.pas)
			}
			doc += `}`
			ecrireFichier(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, matchID), doc)

			svc := service.NewReplayService(title.DefaultSlug, root, nil, nil)
			factory := func(context.Context, string) (port.ReplayService, error) { return svc, nil }
			r := chi.NewRouter()
			h := handlers.NewReplayHandler(factory, nil)
			r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })

			req := httptest.NewRequest(http.MethodGet, "/players/joueur/matches/"+matchID+"/replay", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("attendu 200, obtenu %d : %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("X-Replay-Cadence-Stale"); got != c.stale {
				t.Errorf("X-Replay-Cadence-Stale = %q, attendu %q", got, c.stale)
			}
			if got, want := w.Header().Get("X-Replay-Latest-Schema-Version"), strconv.Itoa(replay.SchemaVersion); got != want {
				t.Errorf("X-Replay-Latest-Schema-Version = %q, attendu %q", got, want)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := w.Body.String(); got != doc+"\n" { // l'encodeur JSON termine le corps par un saut de ligne
				t.Errorf("corps = %s, attendu %s", got, doc)
			}
		})
	}
}
