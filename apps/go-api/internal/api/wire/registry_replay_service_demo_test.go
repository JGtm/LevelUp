package wire

// registry_replay_service_demo_test.go — en démo, le service de rejeu lit les DONNÉES
// VERSIONNÉES (fonds de carte, mappings, libellés, zones, règles de tiers) à la racine du
// dépôt, et les ARTEFACTS dans les rejeux FIGÉS de la disposition démo
// (title.DemoLayout.ReplayArtifactsDir), joueurs réels masqués par l'index (décisions D-1/D-2
// du plan des recommandations du 2026-10-09).
//
// Régression de B5 (constat R2-1) : le service était enraciné tout entier sur la racine
// d'exécution de la démo. En démo, l'onglet Tactique répondait 404 sur les fonds de carte, et
// les catalogues du rejeu étaient introuvables.
//
// Le test passe par le VRAI handler HTTP et par la construction de production
// (replayServiceFrom), sans base : seules les identités de carte sont bouchonnées.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

const (
	cleFondVersionne = "fond-versionne"
	matchRejeuDemo   = "demo0001"
)

// identitesCarte : la carte du match est désignée par son map_id, clé du fond versionné.
type identitesCarte struct{}

func (identitesCarte) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{MapID: cleFondVersionne}, nil
}

func (identitesCarte) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{MapID: cleFondVersionne}, nil
}

func ecrireFichier(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s : %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("écriture %s : %v", path, err)
	}
}

// fondVersionne pose, sous la racine du DÉPÔT, un fond de carte (calage + image) comme la
// cuisson les écrit dans data/titles/<slug>/reference/map_backgrounds/.
func fondVersionne(t *testing.T, repo string) {
	t.Helper()
	res := title.NewPathResolver(repo)
	sidecar := `{"schemaVersion":1,"module":"` + cleFondVersionne + `","image":"` + cleFondVersionne + `.png",` +
		`"source":"test","generatedAt":"2026-09-27T10:00:00Z","style":"jeu",` +
		`"calibration":{"metersPerPixel":0.092,"originX":65.6,"originY":112.43,` +
		`"widthPx":1332,"heightPx":1287,"convention":"x = originX + (px+0.5)*mpp"},` +
		`"stats":{"anchors":4,"anchorsInFrame":4,"anchorsWithGround":4}}`
	ecrireFichier(t, res.MapBackgroundMetaPath(title.DefaultSlug, cleFondVersionne), sidecar)
	ecrireFichier(t, res.MapBackgroundPath(title.DefaultSlug, cleFondVersionne), "\x89PNG\r\n\x1a\nfond")
}

// routeurRejeu monte le handler de rejeu sur le service construit comme en production.
func routeurRejeu(cfg *config.AppConfig) *chi.Mux {
	svc := replayServiceFrom(cfg, title.DefaultSlug, identitesCarte{})
	h := handlers.NewReplayHandler(func(context.Context, string) (port.ReplayService, error) { return svc, nil })
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

func get(r *chi.Mux, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// artefactDemo : un rejeu minimal dont la piste et la table d'identités nomment un joueur RÉEL.
const (
	xuidReel     = "2533274899990001"
	gamertagReel = "VraiJoueur"
	artefactDemo = `{"schemaVersion":1,"matchId":"` + matchRejeuDemo + `","tracks":[{"slot":3,"team":0,` +
		`"xuid":"` + xuidReel + `","name":"` + gamertagReel + `","points":[]}],"identity":{"players":[` +
		`{"filmIndex":0,"xuid":"` + xuidReel + `","name":"` + gamertagReel + `","link":{}}],"coverage":{}}}`
	indexDemo = `{"matches":[{"match_id":"` + matchRejeuDemo + `","mode":"ctf","schema_version":1}],` +
		`"identities":[{"xuid":"` + xuidReel + `","demo_xuid":"0000000000000001","demo_gamertag":"DemoPlayer2"}]}`
)

func TestReplayService_Demo_DonneesVersionneesEtRejeuxFigesMasques(t *testing.T) {
	repo, demo := t.TempDir(), t.TempDir()
	fondVersionne(t, repo)
	layout := title.NewDemoLayout(demo)
	ecrireFichier(t, layout.ReplayArtifactPath(title.DefaultSlug, matchRejeuDemo), artefactDemo)
	ecrireFichier(t, layout.ReplayIndexPath(title.DefaultSlug), indexDemo)
	r := routeurRejeu(&config.AppConfig{DemoMode: true, RepoRoot: repo, DemoFixturesDir: demo})

	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay/background.png"); w.Code != http.StatusOK {
		t.Errorf("démo : fond de carte versionné, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay")
	if w.Code != http.StatusOK {
		t.Fatalf("démo : rejeu figé, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	corps := w.Body.String()
	if strings.Contains(corps, xuidReel) || strings.Contains(corps, gamertagReel) {
		t.Errorf("démo : identité RÉELLE dans la réponse : %s", corps)
	}
	if !strings.Contains(corps, `"0000000000000001"`) || !strings.Contains(corps, `"DemoPlayer2"`) {
		t.Errorf("démo : identité démo absente de la réponse : %s", corps)
	}
	// Leurres : un artefact du DÉPÔT, ou de la racine d'exécution de la démo, n'est jamais servi.
	ecrireFichier(t, title.NewPathResolver(repo).ReplayArtifactPath(title.DefaultSlug, "depot002"), artefactDemo)
	ecrireFichier(t, layout.RuntimePaths().ReplayArtifactPath(title.DefaultSlug, "runtime3"), artefactDemo)
	for _, id := range []string{"depot002", "runtime3"} {
		if w := get(r, "/players/p/matches/"+id+"/replay"); w.Code != http.StatusNotFound {
			t.Errorf("démo : artefact %s hors rejeux figés servi, statut %d, attendu 404", id, w.Code)
		}
	}
}

func TestReplayService_Demo_IndexAbsentRefuseLeRejeu(t *testing.T) {
	repo, demo := t.TempDir(), t.TempDir()
	layout := title.NewDemoLayout(demo)
	ecrireFichier(t, layout.ReplayArtifactPath(title.DefaultSlug, matchRejeuDemo), artefactDemo)
	r := routeurRejeu(&config.AppConfig{DemoMode: true, RepoRoot: repo, DemoFixturesDir: demo})
	// Sans index, pas de masque : servir le document publierait les noms réels.
	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay"); w.Code == http.StatusOK ||
		strings.Contains(w.Body.String(), gamertagReel) {
		t.Errorf("démo sans index : statut %d, attendu un refus sans identité réelle — corps %s", w.Code, w.Body.String())
	}
}

func TestReplayService_HorsDemo_ToutSousLeDepot(t *testing.T) {
	repo := t.TempDir()
	fondVersionne(t, repo)
	ecrireFichier(t, title.NewPathResolver(repo).ReplayArtifactPath(title.DefaultSlug, matchRejeuDemo), `{"schemaVersion":1}`)
	r := routeurRejeu(&config.AppConfig{RepoRoot: repo})

	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay/background.png"); w.Code != http.StatusOK {
		t.Errorf("hors démo : fond de carte, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay"); w.Code != http.StatusOK {
		t.Errorf("hors démo : artefact, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
}
