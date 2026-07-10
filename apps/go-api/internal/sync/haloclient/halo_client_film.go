// Package sync - halo_client_film.go : film manifest + chunk download +
// highlight events extraction. Decoupe de halo_client.go (god-file split,
// refactor 2026-05-27).
package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/games"
)

// ─────────────────────────────────────────────────────────────────────────────
// Film API (Sprint 41 T2)
// ─────────────────────────────────────────────────────────────────────────────

const haloUGCHost = "https://discovery-infiniteugc.svc.halowaypoint.com"

// Constantes pour les types de chunks film Halo.
const (
	filmChunkTypeHeader          = 1
	filmChunkTypeReplicationData = 2
	filmChunkTypeHighlightEvents = 3
)

// filmChunkParallelism : nombre max de downloads CDN parallèles dans
// GetMatchFilm. Un film typique a 10-30 chunks REPLICATION_DATA, chaque
// download = 200-500ms RTT CDN. Avec parallelism=8, un film de 20 chunks
// passe de ~6s séquentiel à ~1s (3 vagues). Le rate limiter par-token cape
// l'ensemble du flux HTTP — bumper plus haut ne donnerait rien tant qu'on
// reste sous la limite ~15 RPS effectif.
//
// Plan stabilisation 2026-05-22 §3.1 — opportunité ratée initialement,
// reprise sur demande utilisateur 2026-05-23 (instruction initiale : "c'est
// tout le traitement et les calculs qu'il faut optimisier niveau perf").
const filmChunkParallelism = 8

// filmManifest représente la réponse JSON de l'endpoint /hi/films/matches/{id}/spectate.
// Structure validée contre spnkr/models/discovery_ugc.py (FilmCustomData + FilmChunk).
type filmManifest struct {
	BlobStoragePathPrefix string `json:"BlobStoragePathPrefix"`
	CustomData            struct {
		FilmMajorVersion int         `json:"FilmMajorVersion"`
		Chunks           []filmChunk `json:"Chunks"`
	} `json:"CustomData"`
}

// filmChunk décrit un segment binaire du film Halo.
type filmChunk struct {
	Index                            int    `json:"Index"`
	ChunkType                        int    `json:"ChunkType"`
	ChunkSize                        int    `json:"ChunkSize"`
	ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
	DurationMilliseconds             int    `json:"DurationMilliseconds"`
	FileRelativePath                 string `json:"FileRelativePath"`
}

// buildChunkURL construit l'URL complète d'un chunk depuis le prefix et le chemin relatif.
func buildChunkURL(blobPrefix, fileRelativePath string) string {
	name := strings.TrimLeft(fileRelativePath, "/")
	if name == "" {
		return blobPrefix
	}
	if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
		return blobPrefix + "/" + name
	}
	return blobPrefix + name
}

// fetchFilmManifest télécharge et décode le manifest film d'un match.
// Retourne (manifest, true, nil) si disponible, (nil, false, nil) si absent (404/410).
//
// Si un LocalFilmCache est configuré, on lit le manifest local avant l'API —
// le cache disque survit à l'expiration de l'endpoint manifest API (Halo
// purge les manifestes après quelques semaines/mois mais le cache local
// conserve les blob_prefixes valides plus longtemps via le CDN).
func (c *HaloAPIClient) fetchFilmManifest(ctx context.Context, matchID string) (*filmManifest, bool, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, false, fmt.Errorf("fetchFilmManifest: matchID invalide %q", matchID)
	}

	// 1. Cache disque (Python legacy).
	if cm, err := c.localFilmCache.LoadManifest(matchID); err == nil && cm != nil {
		manifest := &filmManifest{
			BlobStoragePathPrefix: cm.BlobPrefix,
		}
		manifest.CustomData.FilmMajorVersion = 0 // legacy cache n'a pas la version
		manifest.CustomData.Chunks = make([]filmChunk, 0, len(cm.Chunks))
		for _, ch := range cm.Chunks {
			manifest.CustomData.Chunks = append(manifest.CustomData.Chunks, filmChunk{
				Index:                            ch.Index,
				ChunkType:                        ch.ChunkType,
				ChunkStartTimeOffsetMilliseconds: ch.StartMS,
				DurationMilliseconds:             ch.DurationMS,
				FileRelativePath:                 ch.FileRelativePath,
			})
		}
		return manifest, true, nil
	}

	// 2. API Halo.
	endpoint := fmt.Sprintf("%s/%s/films/matches/%s/spectate", c.hostFor(ctx, games.EndpointUGCFilm, haloUGCHost), c.gamePrefix(ctx), url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("fetchFilmManifest(%s): %w", matchID, err)
	}
	var manifest filmManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, false, fmt.Errorf("fetchFilmManifest decode(%s): %w", matchID, err)
	}
	return &manifest, true, nil
}

// GetMatchFilm télécharge le manifest film d'un match et retourne les chunks REPLICATION_DATA.
// Seuls les chunks ChunkType==2 (REPLICATION_DATA) sont retournés — pour le weapon scanner.
// Retourne (chunks, true, nil) si le film est disponible.
// Retourne (nil, false, nil) si le film est absent (404/410) — normal pour vieux matchs.
//
// Phase 3.1 (plan stabilisation 2026-05-22, reprise 2026-05-23) : les chunks
// sont téléchargés en parallèle via errgroup.SetLimit(filmChunkParallelism=8).
// Avant : boucle séquentielle, ~6s pour un film de 20 chunks à 300ms RTT.
// Après : ~1s (3 vagues × 300ms), gain 5-6×. La fonction ne retourne qu'une
// fois TOUS les chunks téléchargés (errgroup.Wait), donc le caller
// BackfillWeaponKillsForMatch reçoit un map complet — aucune race possible
// avec le traitement aval (scan fire events, kills attribution).
//
// **Architecture sans mutex** : chaque goroutine écrit dans un slot pré-alloué
// du slice `dlResults` (indexé par la position dans `toDownload`, jamais par
// chunk.Index qui peut être sparse). L'assemblage final du map se fait
// séquentiellement après eg.Wait() — race-free par construction.
func (c *HaloAPIClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]FilmChunkData, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}

	result := make(map[int]FilmChunkData)

	// Phase 1 (séquentielle) : pré-filtre les chunks. Cache hits → écrits
	// directement dans result. Misses → accumulés dans toDownload pour la
	// phase 2 parallèle. Le cache check est rapide (fichier local), pas la
	// peine de paralléliser.
	type chunkToDownload struct {
		index            int
		startMS          int
		durationMS       int
		fileRelativePath string
	}
	var toDownload []chunkToDownload
	for _, chunk := range manifest.CustomData.Chunks {
		if chunk.ChunkType != filmChunkTypeReplicationData {
			continue
		}
		// Cache disque d'abord (Python legacy stocke les REPLICATION_DATA).
		if cached, cErr := c.localFilmCache.LoadChunk(matchID, chunk.Index); cErr == nil && cached != nil {
			result[chunk.Index] = FilmChunkData{
				Data:       cached,
				StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
				DurationMS: chunk.DurationMilliseconds,
			}
			continue
		}
		toDownload = append(toDownload, chunkToDownload{
			index:            chunk.Index,
			startMS:          chunk.ChunkStartTimeOffsetMilliseconds,
			durationMS:       chunk.DurationMilliseconds,
			fileRelativePath: chunk.FileRelativePath,
		})
	}

	if len(toDownload) == 0 {
		// Tout en cache (ou pas de REPLICATION_DATA). Court-circuit identique
		// au comportement séquentiel pré-paralléllisation.
		if len(result) == 0 {
			return nil, false, nil
		}
		return result, true, nil
	}

	// Phase 2 (parallèle) : downloads CDN concurrents. Chaque goroutine écrit
	// dans un slot pré-alloué (no mutex). errgroup.WithContext + SetLimit
	// borne la concurrence à filmChunkParallelism (8). Si une erreur tombe,
	// les autres goroutines abortent via egCtx.Done() (propagé à downloadBlob
	// via http.Request).
	type dlResult struct {
		index int
		data  FilmChunkData
	}
	dlResults := make([]dlResult, len(toDownload))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(filmChunkParallelism)
	for i, ch := range toDownload {
		i, ch := i, ch
		eg.Go(func() error {
			chunkURL := buildChunkURL(manifest.BlobStoragePathPrefix, ch.fileRelativePath)
			data, err := c.downloadBlob(egCtx, chunkURL)
			if err != nil {
				return fmt.Errorf("GetMatchFilm chunk %d(%s): %w", ch.index, matchID, err)
			}
			dlResults[i] = dlResult{
				index: ch.index,
				data: FilmChunkData{
					Data:       data,
					StartMS:    ch.startMS,
					DurationMS: ch.durationMS,
				},
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, false, err
	}

	// Phase 3 (séquentielle, post-Wait) : assemble le map final. Race-free
	// car eg.Wait() garantit que toutes les goroutines ont terminé leur
	// write avant ce point.
	for _, r := range dlResults {
		result[r.index] = r.data
	}

	if len(result) == 0 {
		return nil, false, nil
	}
	return result, true, nil
}

// GetHighlightEventsChunk télécharge le chunk highlight events (ChunkType=3) du film.
// Retourne (data, filmMajorVersion, true, nil) si disponible.
// Retourne (nil, 0, false, nil) si le film est absent ou sans chunk highlight events.
func (c *HaloAPIClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, 0, found, err
	}

	for _, chunk := range manifest.CustomData.Chunks {
		if chunk.ChunkType != filmChunkTypeHighlightEvents {
			continue
		}
		// Cache disque d'abord (rarement présent — Python ne cache que
		// REPLICATION_DATA — mais on tente).
		if cached, cErr := c.localFilmCache.LoadChunk(matchID, chunk.Index); cErr == nil && cached != nil {
			return cached, manifest.CustomData.FilmMajorVersion, true, nil
		}
		chunkURL := buildChunkURL(manifest.BlobStoragePathPrefix, chunk.FileRelativePath)
		data, err := c.downloadBlob(ctx, chunkURL)
		if err != nil {
			// Fallback gracieux : si le manifest vient du cache local et que
			// le blob CDN a expiré, on retourne (nil, 0, false, nil) au lieu
			// d'une erreur — comportement equivalent à "film absent".
			if isNotFoundErr(err) {
				return nil, 0, false, nil
			}
			return nil, 0, false, fmt.Errorf("GetHighlightEventsChunk(%s): %w", matchID, err)
		}
		return data, manifest.CustomData.FilmMajorVersion, true, nil
	}
	return nil, 0, false, nil
}

// FilmChunkData encapsule les données binaires d'un chunk film.
type FilmChunkData struct {
	Data       []byte
	StartMS    int
	DurationMS int
}

// isNotFoundErr vérifie si l'erreur est un 404 ou 410 (film absent).
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "HTTP 404") || contains(s, "HTTP 410") || contains(s, "ressource absente")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// downloadBlob télécharge un blob Halo sans header d'auth (pre-signed URL)
// et le décompresse zlib (le CDN Azure des films Halo Infinite renvoie du zlib brut).
// Portage de download_film_chunk() (Python api_client.py:485-498).
