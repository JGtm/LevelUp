package wire

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/sync/filmfetch"

	"github.com/JGtm/halo-api/infinite"
	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

const (
	genXUIDPage  = "xuid-du-joueur-de-la-page"
	genXUIDAutre = "xuid-dun-autre-profil"
	genSecret    = "SPARTAN-SECRET-A-NE-JAMAIS-JOURNALISER"
)

// faux client de téléchargement.
type fakeDownloader struct {
	chunks []filmfetch.FilmChunk
	found  bool
	err    error
	calls  int
}

func (d *fakeDownloader) GetFilmChunks(context.Context, string) ([]filmfetch.FilmChunk, bool, error) {
	d.calls++
	return d.chunks, d.found, d.err
}

// fetchRig : un téléchargeur dont toutes les dépendances sont des faux qui enregistrent leurs appels.
type fetchRig struct {
	mu          sync.Mutex
	complete    bool
	tokenCalls  []string
	tokenOf     map[string]*domain.HaloTokens // les jetons de TOUS les profils : seul celui du xuid demandé peut servir
	tokenErr    error
	clientsMade int
	client      *fakeDownloader
	written     []string
	writeErr    error
	absentFor   []string
	absentErr   error
	logs        *bytes.Buffer
}

func newFetchRig(t *testing.T) (*fetchRig, *filmDemandFetcher) {
	t.Helper()
	g := &fetchRig{
		tokenOf: map[string]*domain.HaloTokens{
			genXUIDPage:  {SpartanToken: genSecret, ClearanceToken: "clr"},
			genXUIDAutre: {SpartanToken: genSecret + "-AUTRE", ClearanceToken: "clr2"},
		},
		client: &fakeDownloader{found: true, chunks: []filmfetch.FilmChunk{
			{Index: 0, ChunkType: 1, Data: []byte("a")}, {Index: 1, ChunkType: 2, Data: []byte("b")},
		}},
		logs: &bytes.Buffer{},
	}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(g.logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return g, &filmDemandFetcher{
		archiveRoot: "/archive",
		complete:    func(string, string) (bool, error) { return g.complete, nil },
		tokens: func(_ context.Context, xuid string) (*domain.HaloTokens, error) {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.tokenCalls = append(g.tokenCalls, xuid)
			if g.tokenErr != nil {
				return nil, g.tokenErr
			}
			return g.tokenOf[xuid], nil
		},
		newClient: func(*domain.HaloTokens) filmDownloader { g.clientsMade++; return g.client },
		write: func(_ context.Context, _, short string, _ []filmarchive.WriteChunk) error {
			g.written = append(g.written, short)
			return g.writeErr
		},
		markAbsent: func(_ context.Context, matchID string) error {
			g.absentFor = append(g.absentFor, matchID)
			return g.absentErr
		},
	}
}

func genDemand() domain.ReplayGenerationDemand {
	return domain.ReplayGenerationDemand{
		TitleSlug: "halo_infinite", MatchID: "0a0a0a0a-1111-2222-3333-444444444444",
		PlayerSlug: "Joueur", XUID: genXUIDPage, Gamertag: "Joueur",
	}
}

// Un film complet à l'archive ne demande aucun jeton, ne bâtit aucun client, n'écrit rien.
func TestFilmDemande_FilmArchiveAucunJeton(t *testing.T) {
	g, f := newFetchRig(t)
	g.complete = true
	if err := f.fetch(context.Background(), genDemand()); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(g.tokenCalls) != 0 || g.clientsMade != 0 || len(g.written) != 0 || len(g.absentFor) != 0 {
		t.Fatalf("film archivé : jetons=%v clients=%d écritures=%v marqueurs=%v, attendu aucun",
			g.tokenCalls, g.clientsMade, g.written, g.absentFor)
	}
}

// Film absent de l'archive : UN SEUL appel de jeton, avec le xuid de la demande, alors qu'un autre profil a un jeton.
func TestFilmDemande_UnSeulAppelDeJetonAvecLeXuidDeLaDemande(t *testing.T) {
	g, f := newFetchRig(t)
	if err := f.fetch(context.Background(), genDemand()); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(g.tokenCalls) != 1 || g.tokenCalls[0] != genXUIDPage {
		t.Fatalf("appels de jeton = %v, attendu exactement [%s]", g.tokenCalls, genXUIDPage)
	}
	if len(g.written) != 1 || g.written[0] != "0a0a0a0a" || g.clientsMade != 1 {
		t.Fatalf("écritures=%v clients=%d, attendu un film archivé sous sa forme courte", g.written, g.clientsMade)
	}
	if strings.Contains(g.logs.String(), genSecret) {
		t.Fatalf("le journal ne doit jamais porter un jeton : %s", g.logs.String())
	}
}

// (nil, nil), erreur du résolveur, jeton vide : ErrReplayNoToken, aucun client, aucun conseil de recapture.
func TestFilmDemande_SansJetonUtilisable(t *testing.T) {
	cas := map[string]func(g *fetchRig){
		"aucun jeton": func(g *fetchRig) { delete(g.tokenOf, genXUIDPage) },
		"erreur":      func(g *fetchRig) { g.tokenErr = errors.New("échange refusé") },
		"jeton vide":  func(g *fetchRig) { g.tokenOf[genXUIDPage] = &domain.HaloTokens{} },
	}
	for nom, prepare := range cas {
		t.Run(nom, func(t *testing.T) {
			g, f := newFetchRig(t)
			prepare(g)
			err := f.fetch(context.Background(), genDemand())
			if !errors.Is(err, domain.ErrReplayNoToken) {
				t.Fatalf("err = %v, attendu ErrReplayNoToken", err)
			}
			if g.clientsMade != 0 || len(g.written) != 0 || len(g.absentFor) != 0 {
				t.Fatal("sans jeton : ni client, ni écriture, ni marqueur")
			}
			if len(g.tokenCalls) != 1 || g.tokenCalls[0] != genXUIDPage {
				t.Fatalf("appels de jeton = %v", g.tokenCalls)
			}
			for _, texte := range []string{err.Error(), g.logs.String()} {
				if strings.Contains(strings.ToLower(texte), "captur") {
					t.Fatalf("aucun conseil de recapture : %s", texte)
				}
			}
		})
	}
}

// Film disparu (404 typé, ou manifeste vide) : le marqueur est posé pour le match complet, ErrReplayFilmGone.
func TestFilmDemande_FilmDisparuPoseLeMarqueur(t *testing.T) {
	cas := map[string]func(g *fetchRig){
		"404 typé":       func(g *fetchRig) { g.client.err = &infinite.HTTPError{StatusCode: 404, URL: "u"} },
		"410 d'un blob":  func(g *fetchRig) { g.client.err = &infinite.BlobHTTPError{StatusCode: 410, URL: "u"} },
		"manifeste vide": func(g *fetchRig) { g.client.found, g.client.chunks = false, nil },
		"zéro morceau":   func(g *fetchRig) { g.client.chunks = nil },
	}
	for nom, prepare := range cas {
		t.Run(nom, func(t *testing.T) {
			g, f := newFetchRig(t)
			prepare(g)
			err := f.fetch(context.Background(), genDemand())
			if !errors.Is(err, domain.ErrReplayFilmGone) {
				t.Fatalf("err = %v, attendu ErrReplayFilmGone", err)
			}
			if len(g.absentFor) != 1 || g.absentFor[0] != genDemand().MatchID || len(g.written) != 0 {
				t.Fatalf("marqueurs=%v écritures=%v", g.absentFor, g.written)
			}
		})
	}
}

// Un échec du marqueur se journalise ; l'issue reste « film disparu ».
func TestFilmDemande_EchecDuMarqueurJournalise(t *testing.T) {
	g, f := newFetchRig(t)
	g.client.found = false
	g.absentErr = errors.New("écrivain occupé")
	if err := f.fetch(context.Background(), genDemand()); !errors.Is(err, domain.ErrReplayFilmGone) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(g.logs.String(), "level=WARN") || !strings.Contains(g.logs.String(), "marqueur film absent non posé") {
		t.Fatalf("l'échec du marqueur doit se journaliser : %s", g.logs.String())
	}
}

// Film non finalisé : ni marqueur ni écriture ; une panne transitoire (503) non plus.
func TestFilmDemande_NonFinaliseEtPanneTransitoireNeMarquentRien(t *testing.T) {
	cas := map[string]error{
		"non finalisé": fmt.Errorf("film x: %w", finalise.ErrFilmNonFinalise),
		"503":          &infinite.HTTPError{StatusCode: 503, URL: "u"},
		"réseau":       errors.New("connexion coupée"),
	}
	for nom, cause := range cas {
		t.Run(nom, func(t *testing.T) {
			g, f := newFetchRig(t)
			g.client.err = cause
			err := f.fetch(context.Background(), genDemand())
			if err == nil || errors.Is(err, domain.ErrReplayFilmGone) {
				t.Fatalf("err = %v, attendu une erreur qui n'est pas « film disparu »", err)
			}
			if errors.Is(cause, finalise.ErrFilmNonFinalise) != errors.Is(err, finalise.ErrFilmNonFinalise) {
				t.Fatalf("la cause du film non finalisé doit rester reconnaissable : %v", err)
			}
			if len(g.absentFor) != 0 || len(g.written) != 0 {
				t.Fatalf("marqueurs=%v écritures=%v, attendu aucun", g.absentFor, g.written)
			}
		})
	}
}

// Un échec d'archivage se rend et se journalise en Error.
func TestFilmDemande_EchecDArchivage(t *testing.T) {
	g, f := newFetchRig(t)
	g.writeErr = errors.New("disque plein")
	if err := f.fetch(context.Background(), genDemand()); err == nil || errors.Is(err, domain.ErrReplayFilmGone) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(g.logs.String(), "level=ERROR") {
		t.Fatalf("l'échec d'archivage doit se journaliser en Error : %s", g.logs.String())
	}
}

// Le profil de la demande est résolu AVANT le jeton, et une résolution en échec arrête tout sans appeler le jeton.
func TestFilmDemande_ProfilIntrouvableAvantLeJeton(t *testing.T) {
	g, f := newFetchRig(t)
	f.ensurePlayer = func(context.Context, domain.ReplayGenerationDemand) error { return errors.New("profil supprimé") }
	if err := f.fetch(context.Background(), genDemand()); err == nil {
		t.Fatal("une erreur est attendue")
	}
	if len(g.tokenCalls) != 0 {
		t.Fatalf("aucun jeton ne doit être demandé sans profil : %v", g.tokenCalls)
	}
}

// Archive désactivée : le film est retiré après la génération ; archive active : il reste.
func TestRetirerFilmSansArchive(t *testing.T) {
	for _, archive := range []bool{false, true} {
		root := t.TempDir()
		short := "0a0a0a0a"
		if err := os.MkdirAll(filmcache.ChunkDir(root, short), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(filmcache.ChunkDir(root, short), "0.bin"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(filmcache.ManifestPath(root, short)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filmcache.ManifestPath(root, short), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		retirerFilmSansArchive(context.Background(), root, "halo_infinite", "0a0a0a0a-1-2-3-4", archive)
		_, errManifest := os.Stat(filmcache.ManifestPath(root, short))
		if archive && errManifest != nil {
			t.Fatalf("archive active : le film doit rester (%v)", errManifest)
		}
		if !archive && !errors.Is(errManifest, os.ErrNotExist) {
			t.Fatalf("archive désactivée : le film doit être retiré (%v)", errManifest)
		}
	}
}
