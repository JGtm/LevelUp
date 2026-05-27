// cmd/fetch_film_chunks — télécharge les chunks REPLICATION_DATA manquants
// depuis les manifests Python hérités (data/cache/film_manifests/*.json).
//
// Les blobs sont sur le CDN Azure Halo (pré-signés, pas d'auth nécessaire).
// Les fichiers sont décompressés zlib avant d'être sauvegardés.
//
// Usage (depuis apps/go-api/) :
//
//	go run ./cmd/fetch_film_chunks/
//	go run ./cmd/fetch_film_chunks/ -cache ../../data/cache -workers 8 -dry-run
package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	chunkTypeReplicationData = 2

	defaultCacheDir = "../../data/cache"
	defaultWorkers  = 8
	httpTimeout     = 30 * time.Second
)

// cachedManifest reflète le format JSON des manifests Python hérités.
type cachedManifest struct {
	BlobPrefix string         `json:"blob_prefix"`
	Chunks     []cachedChunk  `json:"chunks"`
}

type cachedChunk struct {
	Index            int    `json:"index"`
	ChunkType        int    `json:"chunk_type"`
	StartMS          int    `json:"start_ms"`
	DurationMS       int    `json:"duration_ms"`
	FileRelativePath string `json:"file_relative_path"`
}

// downloadTask représente un chunk à télécharger.
type downloadTask struct {
	shortID          string
	chunkIndex       int
	url              string
	destPath         string
}

func main() {
	cacheDir := flag.String("cache", defaultCacheDir, "Répertoire racine du cache (contient film_manifests/ et film_chunks/)")
	workers := flag.Int("workers", defaultWorkers, "Nombre de téléchargements parallèles")
	dryRun := flag.Bool("dry-run", false, "Lister les chunks manquants sans télécharger")
	verbose := flag.Bool("v", false, "Logs verbeux par chunk")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	manifDir := filepath.Join(*cacheDir, "film_manifests")
	chunksDir := filepath.Join(*cacheDir, "film_chunks")

	entries, err := os.ReadDir(manifDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lecture film_manifests: %v\n", err)
		os.Exit(1)
	}

	var tasks []downloadTask
	totalExpected := 0

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		shortID := strings.TrimSuffix(e.Name(), ".json")

		raw, err := os.ReadFile(filepath.Join(manifDir, e.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "lecture %s: %v\n", e.Name(), err)
			continue
		}
		var m cachedManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", e.Name(), err)
			continue
		}

		matchChunksDir := filepath.Join(chunksDir, shortID)

		for _, ch := range m.Chunks {
			if ch.ChunkType != chunkTypeReplicationData {
				continue
			}
			totalExpected++
			destPath := filepath.Join(matchChunksDir, fmt.Sprintf("chunk_%02d.bin", ch.Index))
			if _, err := os.Stat(destPath); err == nil {
				continue // déjà présent
			}
			relPath := strings.TrimLeft(ch.FileRelativePath, "/")
			blobPrefix := m.BlobPrefix
			if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
				blobPrefix += "/"
			}
			tasks = append(tasks, downloadTask{
				shortID:    shortID,
				chunkIndex: ch.Index,
				url:        blobPrefix + relPath,
				destPath:   destPath,
			})
		}
	}

	fmt.Printf("Manifests scannés : %d\n", len(entries))
	fmt.Printf("Chunks REPLICATION_DATA attendus : %d\n", totalExpected)
	fmt.Printf("Chunks manquants à télécharger   : %d\n", len(tasks))

	if *dryRun || len(tasks) == 0 {
		if *dryRun && len(tasks) > 0 {
			fmt.Println("\n--- dry-run : premiers 20 chunks manquants ---")
			n := 20
			if n > len(tasks) {
				n = len(tasks)
			}
			for _, t := range tasks[:n] {
				fmt.Printf("  %s/chunk_%02d  %s\n", t.shortID, t.chunkIndex, t.url)
			}
		}
		return
	}

	// Création des répertoires de destination en avance.
	seen := make(map[string]bool)
	for _, t := range tasks {
		dir := filepath.Dir(t.destPath)
		if !seen[dir] {
			seen[dir] = true
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", dir, err)
			}
		}
	}

	client := &http.Client{Timeout: httpTimeout}

	taskCh := make(chan downloadTask, len(tasks))
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var (
		downloaded int64
		skipped    int64
		failed     int64
	)

	start := time.Now()
	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				err := downloadChunk(ctx, client, task)
				if err != nil {
					if isGone(err) {
						atomic.AddInt64(&skipped, 1)
						if *verbose {
							fmt.Fprintf(os.Stderr, "SKIP (expiré) %s/chunk_%02d\n", task.shortID, task.chunkIndex)
						}
					} else {
						atomic.AddInt64(&failed, 1)
						fmt.Fprintf(os.Stderr, "ERREUR %s/chunk_%02d: %v\n", task.shortID, task.chunkIndex, err)
					}
				} else {
					n := atomic.AddInt64(&downloaded, 1)
					if *verbose || n%100 == 0 {
						fmt.Printf("  [%d/%d] %s/chunk_%02d OK\n", n, len(tasks), task.shortID, task.chunkIndex)
					}
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\n=== Résultat ===\n")
	fmt.Printf("Téléchargés : %d\n", downloaded)
	fmt.Printf("Expirés/404 : %d\n", skipped)
	fmt.Printf("Erreurs     : %d\n", failed)
	fmt.Printf("Durée       : %s\n", elapsed.Round(time.Second))
}

// downloadChunk télécharge un chunk CDN, le décompresse zlib, et l'écrit sur disque.
func downloadChunk(ctx context.Context, client *http.Client, task downloadTask) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return &expiredError{status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("zlib header: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return fmt.Errorf("zlib decompress: %w", err)
	}

	if err := os.WriteFile(task.destPath, decompressed, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", task.destPath, err)
	}
	return nil
}

type expiredError struct{ status int }

func (e *expiredError) Error() string { return fmt.Sprintf("HTTP %d (CDN expiré)", e.status) }

func isGone(err error) bool {
	_, ok := err.(*expiredError)
	return ok
}
