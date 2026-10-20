// cmd/fetch_film_chunks — complète les films du cache dont des chunks manquent, depuis les
// manifests Python hérités (data/cache/film_manifests/*.json, qui portent le `blob_prefix` CDN).
//
// Les blobs sont sur le CDN Azure Halo (pré-signés, pas d'auth nécessaire). Les fichiers sont
// décompressés zlib avant d'être sauvegardés.
//
// UN SEUL ÉCRIVAIN DU CACHE (J2.3 du plan de suite de l'audit du décodeur de film, 2026-09-26).
// L'outil écrivait ses chunks lui-même, en place, sous un nom recomposé ; il passe désormais par
// `filmcache.Write` (écriture atomique, chunk présent adopté seulement à la bonne taille). Le
// writer ne valide qu'un film FINALISÉ et COMPLET : un film incomplet est donc complété EN
// ENTIER — chunks présents relus par `filmcache.LireChunk`, chunks manquants (ou tronqués)
// téléchargés, quel que soit leur type — ou pas du tout (un chunk expiré au CDN laisse le film
// tel quel). Le manifeste historique, finalisé, est conservé par le writer.
//
// Usage (depuis apps/go-api/) :
//
//	go run ./cmd/fetch_film_chunks/
//	go run ./cmd/fetch_film_chunks/ -cache ../../data/cache -workers 8 -dry-run
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

const (
	defaultCacheDir = "../../data/cache"
	defaultWorkers  = 8
	httpTimeout     = 30 * time.Second
)

// cachedManifest reflète le format JSON des manifests Python hérités.
type cachedManifest struct {
	BlobPrefix string        `json:"blob_prefix"`
	Chunks     []cachedChunk `json:"chunks"`
}

type cachedChunk struct {
	Index            int    `json:"index"`
	ChunkType        int    `json:"chunk_type"`
	StartMS          int    `json:"start_ms"`
	DurationMS       int    `json:"duration_ms"`
	FileRelativePath string `json:"file_relative_path"`
}

// filmIncomplet : un film du cache dont au moins un chunk déclaré manque sur disque.
type filmIncomplet struct {
	shortID   string
	manifest  cachedManifest
	manquants int
}

// bilan : les compteurs de la passe, partagés par les workers.
type bilan struct {
	completes   atomic.Int64
	telecharges atomic.Int64
	expires     atomic.Int64
	erreurs     atomic.Int64
}

func main() {
	cacheDir := flag.String("cache", defaultCacheDir, "Répertoire racine du cache (contient film_manifests/ et film_chunks/)")
	workers := flag.Int("workers", defaultWorkers, "Nombre de films complétés en parallèle")
	dryRun := flag.Bool("dry-run", false, "Lister les films incomplets sans télécharger")
	verbose := flag.Bool("v", false, "Logs verbeux par film")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	films, scannes, err := filmsIncomplets(*cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lecture des manifestes: %v\n", err)
		os.Exit(1)
	}
	totalManquants := 0
	for _, f := range films {
		totalManquants += f.manquants
	}
	fmt.Printf("Manifests scannés  : %d\n", scannes)
	fmt.Printf("Films incomplets   : %d\n", len(films))
	fmt.Printf("Chunks manquants   : %d\n", totalManquants)

	if *dryRun || len(films) == 0 {
		if *dryRun {
			for _, f := range films[:min(20, len(films))] {
				fmt.Printf("  %s : %d chunk(s) manquant(s)\n", f.shortID, f.manquants)
			}
		}
		return
	}

	start := time.Now()
	b := completerTout(context.Background(), *cacheDir, films, *workers, *verbose)
	fmt.Printf("\n=== Résultat ===\n")
	fmt.Printf("Films complétés : %d\n", b.completes.Load())
	fmt.Printf("Chunks reçus    : %d\n", b.telecharges.Load())
	fmt.Printf("Expirés/404     : %d\n", b.expires.Load())
	fmt.Printf("Erreurs         : %d\n", b.erreurs.Load())
	fmt.Printf("Durée           : %s\n", time.Since(start).Round(time.Second))
}

// filmsIncomplets lit chaque manifeste du cache et rend les films dont un chunk déclaré manque
// sur disque. Un manifeste illisible est signalé et sauté.
func filmsIncomplets(root string) ([]filmIncomplet, int, error) {
	shorts, err := filmcache.ListShortIDs(root)
	if err != nil {
		return nil, 0, err
	}
	var out []filmIncomplet
	for _, shortID := range shorts {
		raw, err := os.ReadFile(filmcache.ManifestPath(root, shortID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "lecture %s: %v\n", shortID, err)
			continue
		}
		var m cachedManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", shortID, err)
			continue
		}
		dir := filmcache.ChunkDir(root, shortID)
		manquants := 0
		for _, ch := range m.Chunks {
			if _, err := os.Stat(filmcache.CheminDuChunk(dir, ch.Index)); err != nil {
				manquants++
			}
		}
		if manquants > 0 {
			out = append(out, filmIncomplet{shortID: shortID, manifest: m, manquants: manquants})
		}
	}
	return out, len(shorts), nil
}

// completerTout complète les films sur `workers` workers.
func completerTout(ctx context.Context, root string, films []filmIncomplet, workers int, verbose bool) *bilan {
	b := &bilan{}
	client := &http.Client{Timeout: httpTimeout}
	filmCh := make(chan filmIncomplet, len(films))
	for _, f := range films {
		filmCh <- f
	}
	close(filmCh)

	var wg sync.WaitGroup
	for range max(1, workers) {
		wg.Go(func() {
			for f := range filmCh {
				recus, err := completerFilm(ctx, client, root, f)
				b.telecharges.Add(int64(recus))
				switch {
				case isGone(err):
					b.expires.Add(1)
					if verbose {
						fmt.Fprintf(os.Stderr, "SKIP (expiré) %s: %v\n", f.shortID, err)
					}
				case err != nil:
					b.erreurs.Add(1)
					fmt.Fprintf(os.Stderr, "ERREUR %s: %v\n", f.shortID, err)
				default:
					if n := b.completes.Add(1); verbose || n%100 == 0 {
						fmt.Printf("  [%d/%d] %s OK (%d chunk(s) reçu(s))\n", n, len(films), f.shortID, recus)
					}
				}
			}
		})
	}
	wg.Wait()
	return b
}

// completerFilm relit les chunks présents, télécharge les manquants ou tronqués, et confie le
// film ENTIER à `filmcache.Write`. Rend le nombre de chunks téléchargés.
func completerFilm(ctx context.Context, client *http.Client, root string, f filmIncomplet) (int, error) {
	chunks := make([]filmcache.WriteChunk, 0, len(f.manifest.Chunks))
	recus := 0
	for _, ch := range f.manifest.Chunks {
		data, err := filmcache.LireChunk(root, f.shortID, ch.Index)
		if err != nil {
			if !chunkARetelecharger(err) {
				return recus, err
			}
			if data, err = downloadChunk(ctx, client, chunkURL(f.manifest.BlobPrefix, ch.FileRelativePath)); err != nil {
				return recus, fmt.Errorf("chunk %d: %w", ch.Index, err)
			}
			recus++
		}
		chunks = append(chunks, filmcache.WriteChunk{Index: ch.Index, ChunkType: ch.ChunkType,
			StartMS: ch.StartMS, DurationMS: ch.DurationMS, Data: data})
	}
	return recus, filmcache.Write(ctx, root, f.shortID, chunks)
}

// chunkARetelecharger : absent du disque, ou tronqué (taille différente du manifeste).
func chunkARetelecharger(err error) bool {
	var tronque *filmcache.ErrChunkTronque
	return errors.Is(err, os.ErrNotExist) || errors.As(err, &tronque)
}

func chunkURL(blobPrefix, fileRelativePath string) string {
	if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
		blobPrefix += "/"
	}
	return blobPrefix + strings.TrimLeft(fileRelativePath, "/")
}

// downloadChunk télécharge un chunk CDN et le rend décompressé.
func downloadChunk(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, &expiredError{status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return decfilm.Decompresser(raw)
}

type expiredError struct{ status int }

func (e *expiredError) Error() string { return fmt.Sprintf("HTTP %d (CDN expiré)", e.status) }

func isGone(err error) bool {
	var gone *expiredError
	return errors.As(err, &gone)
}
