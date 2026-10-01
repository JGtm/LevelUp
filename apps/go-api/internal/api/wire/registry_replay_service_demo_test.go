package wire

// registry_replay_service_demo_test.go — en démo, le service de rejeu lit les DONNÉES
// VERSIONNÉES (fonds de carte, mappings, libellés, zones, règles de tiers) à la racine du
// dépôt, et les ARTEFACTS D'EXÉCUTION (rejeux) sous `<démo>/runtime/` (revue adversariale du
// lot B5, constat R2-1 ; lot B-C7 du backlog 2026-09-26).
//
// Régression de B5 : le service était enraciné tout entier sur cfg.RuntimePaths(). En démo,
// l'onglet Tactique répondait 404 sur les fonds de carte, et les catalogues du rejeu étaient
// introuvables.
//
// Le test passe par le VRAI handler HTTP et par la construction de production
// (replayServiceFrom), sans base : seules les identités de carte sont bouchonnées.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestReplayService_Demo_DonneesVersionneesEtArtefactsRuntime(t *testing.T) {
	repo, demo := t.TempDir(), t.TempDir()
	fondVersionne(t, repo)
	// L'artefact de rejeu vit sous <démo>/runtime/, et là seulement.
	runtime := title.NewDemoLayout(demo).RuntimePaths()
	ecrireFichier(t, runtime.ReplayArtifactPath(title.DefaultSlug, matchRejeuDemo), `{"schemaVersion":1}`)
	r := routeurRejeu(&config.AppConfig{DemoMode: true, RepoRoot: repo, DemoFixturesDir: demo})

	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay/background.png"); w.Code != http.StatusOK {
		t.Errorf("démo : fond de carte versionné, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	if w := get(r, "/players/p/matches/"+matchRejeuDemo+"/replay"); w.Code != http.StatusOK {
		t.Errorf("démo : artefact sous runtime, statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	// Leurre : un artefact du DÉPÔT n'est jamais servi en démo.
	ecrireFichier(t, title.NewPathResolver(repo).ReplayArtifactPath(title.DefaultSlug, "depot002"), `{"schemaVersion":1}`)
	if w := get(r, "/players/p/matches/depot002/replay"); w.Code != http.StatusNotFound {
		t.Errorf("démo : artefact du dépôt servi, statut %d, attendu 404", w.Code)
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
