// Package handlers_test — media_serve_confinement_test.go : la route des médias ne sert
// que des fichiers média (liste blanche d'extensions) et, sous le dossier interne d'un
// joueur, seulement son sous-dossier captures/. Une base joueur, son WAL ou un réglage
// JSON posés dans une racine de service répondent 404, quelle que soit la racine.
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

const confinementSlug = "JGtm"

// mediaRootFixture décrit une racine de service : où poser un fichier nommé name sur
// disque, et le chemin d'URL (après /media/files/) qui l'atteint par cette racine.
type mediaRootFixture struct {
	name   string
	onDisk func(repoRoot, capturesBase, name string) string
	urlFor func(name string) string
}

func mediaRootFixtures() []mediaRootFixture {
	return []mediaRootFixture{
		{
			name: "base des captures, format canonique {owner}/{rel}",
			onDisk: func(_, capturesBase, name string) string {
				return filepath.Join(capturesBase, confinementSlug, name)
			},
			urlFor: func(name string) string { return confinementSlug + "/" + name },
		},
		{
			name: "base des captures, legacy sans préfixe owner",
			onDisk: func(_, capturesBase, name string) string {
				return filepath.Join(capturesBase, confinementSlug, "legacy", name)
			},
			urlFor: func(name string) string { return "legacy/" + name },
		},
		{
			name: "base média interne data/media",
			onDisk: func(repoRoot, _, name string) string {
				return filepath.Join(titlePkg.NewPathResolver(repoRoot).MediaDataDir(), confinementSlug, name)
			},
			urlFor: func(name string) string { return confinementSlug + "/" + name },
		},
		{
			name: "dossier interne du joueur, sous-dossier captures/",
			onDisk: func(repoRoot, _, name string) string {
				return filepath.Join(titlePkg.NewPathResolver(repoRoot).PlayerCapturesDir(titlePkg.DefaultSlug, confinementSlug), name)
			},
			urlFor: func(name string) string { return "captures/" + name },
		},
	}
}

// newConfinementRouter monte ServeMediaFile avec une base de captures configurée et un
// repoRoot réel (les quatre racines sont actives).
func newConfinementRouter(t *testing.T, repoRoot, capturesBase string) *chi.Mux {
	t.Helper()
	store := writeSettingsJSON(t, capturesBase)
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return nil, nil }
	h := handlers.NewMediaHandler(factory, nil, repoRoot).WithSettingsStore(store)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/media/files/*", h.ServeMediaFile)
	})
	return r
}

func writeFixtureFile(t *testing.T, abs string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("DATA"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func serveConfined(r http.Handler, urlPath string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/players/"+confinementSlug+"/media/files/"+urlPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestServeMediaFile_NonMediaFilesAre404OnEveryRoot : une base DuckDB, son WAL et un
// fichier JSON présents dans une racine de service ne sont jamais servis.
func TestServeMediaFile_NonMediaFilesAre404OnEveryRoot(t *testing.T) {
	names := []string{"stats.duckdb", "stats.duckdb.wal", "stats.wal", "media_audio_config.json"}
	for _, root := range mediaRootFixtures() {
		for _, name := range names {
			t.Run(root.name+"/"+name, func(t *testing.T) {
				repoRoot := t.TempDir()
				capturesBase := filepath.Join(repoRoot, "captures_base")
				writeFixtureFile(t, root.onDisk(repoRoot, capturesBase, name))
				r := newConfinementRouter(t, repoRoot, capturesBase)

				w := serveConfined(r, root.urlFor(name))
				if w.Code != http.StatusNotFound {
					t.Fatalf("status = %d, attendu 404 (fichier non média servi : %q)", w.Code, w.Body.String())
				}
			})
		}
	}
}

// TestServeMediaFile_MediaFilesAreServedOnEveryRoot : les formats réellement produits
// (vidéos, images envoyées, miniatures, arbre HLS) restent servis par chaque racine.
func TestServeMediaFile_MediaFilesAreServedOnEveryRoot(t *testing.T) {
	names := []string{
		"clip.mp4", "clip.webm", "clip.mov", "CLIP.MP4",
		"shot.png", "shot.jpg", "shot.jpeg", "shot.bmp", "shot.gif",
		"thumbs/clip.webp", "thumbs/clip.gif",
		"hls/clip/master.m3u8", "hls/clip/stream_0.m3u8", "hls/clip/init_0.mp4",
		"hls/clip/seg_0_000.m4s", "hls/clip/seg_000.ts",
	}
	for _, root := range mediaRootFixtures() {
		for _, name := range names {
			t.Run(root.name+"/"+name, func(t *testing.T) {
				repoRoot := t.TempDir()
				capturesBase := filepath.Join(repoRoot, "captures_base")
				writeFixtureFile(t, root.onDisk(repoRoot, capturesBase, filepath.FromSlash(name)))
				r := newConfinementRouter(t, repoRoot, capturesBase)

				w := serveConfined(r, root.urlFor(name))
				if w.Code != http.StatusOK {
					t.Fatalf("status = %d, attendu 200 (média légitime refusé)", w.Code)
				}
			})
		}
	}
}

// TestServeMediaFile_PlayerDirServesOnlyCaptures : sous le dossier interne du joueur,
// seul captures/ est une racine de service ; un média rangé ailleurs (archive, racine
// du dossier) répond 404.
func TestServeMediaFile_PlayerDirServesOnlyCaptures(t *testing.T) {
	for _, rel := range []string{"stats.duckdb", "media_audio_config.json", "clip.mp4", "archive/clip.mp4", "backup/thumbs/clip.webp"} {
		t.Run(rel, func(t *testing.T) {
			repoRoot := t.TempDir()
			capturesBase := filepath.Join(repoRoot, "captures_base")
			playerDir := titlePkg.NewPathResolver(repoRoot).PlayerDir(titlePkg.DefaultSlug, confinementSlug)
			writeFixtureFile(t, filepath.Join(playerDir, filepath.FromSlash(rel)))
			r := newConfinementRouter(t, repoRoot, capturesBase)

			w := serveConfined(r, rel)
			if w.Code != http.StatusNotFound {
				t.Fatalf("status = %d, attendu 404 (hors de captures/)", w.Code)
			}
		})
	}
}
