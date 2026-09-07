// Package sync - halo_client_film.go : film manifest + chunk download +
// highlight events extraction. Decoupe de halo_client.go (god-file split,
// refactor 2026-05-27).
package haloclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/games"
)

// ─────────────────────────────────────────────────────────────────────────────
// Film API (Sprint 41 T2)
// ─────────────────────────────────────────────────────────────────────────────

const haloUGCHost = "https://discovery-infiniteugc.svc.halowaypoint.com"

// Constantes pour les types de chunks film Halo.
//
// EXPORTÉES parce qu'un consommateur hors de ce paquet doit pouvoir SÉLECTIONNER un type sans
// recopier le nombre : la ventilation des tirs (`sync/killcollector`) ne scanne que la
// REPLICATION_DATA, la population sur laquelle la loi « un fire-event = un tir » a été mesurée.
// Une copie locale du littéral 2 serait un magic number de plus, et il dériverait.
const (
	FilmChunkTypeHeader          = 1
	FilmChunkTypeReplicationData = 2
	FilmChunkTypeHighlightEvents = 3
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
	chunks, found, err := c.fetchFilmChunks(ctx, matchID, "GetMatchFilm",
		func(t int) bool { return t == FilmChunkTypeReplicationData })
	if err != nil || !found {
		return nil, found, err
	}
	result := make(map[int]FilmChunkData, len(chunks))
	for _, ch := range chunks {
		result[ch.Index] = FilmChunkData{
			Data:       ch.Data,
			StartMS:    ch.StartMS,
			DurationMS: ch.DurationMS,
		}
	}
	return result, true, nil
}

// FilmChunk : un chunk du film AVEC son type de manifeste. C'est la forme dont
// a besoin un consommateur qui veut la SÉQUENCE COMPLÈTE (en-tête + réplication
// + kill-feed) et pas seulement un type — cf. GetFilmChunks.
type FilmChunk struct {
	Index      int
	ChunkType  int
	Data       []byte
	StartMS    int
	DurationMS int
}

// GetFilmChunks télécharge TOUS les chunks du film (types 1/2/3) en UNE seule
// lecture de manifeste, triés par index.
//
// POURQUOI CETTE MÉTHODE EXISTE, alors que GetMatchFilm + GetHighlightEventsChunk
// couvraient déjà le besoin du scanner d'armes :
//
//   - `GetMatchFilm` ne rend QUE le type 2 et `GetHighlightEventsChunk` QUE le
//     type 3 : l'EN-TÊTE (chunk 0, type 1) n'était téléchargé par personne.
//     Découverte J0.1 (2026-07-31) : `cmd/fetch_film_chunks` ne le prenait pas
//     non plus, et il a fallu le récupérer à part pour amorcer le World du
//     décodeur de rejeu. Une méthode qui rend la séquence COMPLÈTE ferme ce trou
//     pour tous les consommateurs à la fois.
//   - les deux appels séparés refont CHACUN `fetchFilmManifest`, soit un
//     aller-retour HTTP de plus par film. Ici il n'y en a qu'un. (C'est la
//     « voie propre » que le guide du décodeur nommait : exposer une méthode qui
//     rend les types en une passe, PAS mettre le manifeste en cache global.)
//
// Retourne (nil, false, nil) si le film est absent (404/410) — cas NORMAL, tous
// les matchs n'ont pas de film.
func (c *HaloAPIClient) GetFilmChunks(ctx context.Context, matchID string) ([]FilmChunk, bool, error) {
	return c.fetchFilmChunks(ctx, matchID, "GetFilmChunks", func(int) bool { return true })
}

// fetchFilmChunks : le chemin de téléchargement COMMUN (cache-first puis CDN en
// parallèle), filtré par type. Une seule copie de la mécanique : GetMatchFilm et
// GetFilmChunks n'en sont que deux vues.
//
// Phase 1 (séquentielle) : pré-filtre les chunks. Cache hits → résultat direct.
// Misses → accumulés pour la phase 2 parallèle. Le cache check est rapide
// (fichier local), pas la peine de paralléliser.
//
// Phase 2 (parallèle) : downloads CDN concurrents, chaque goroutine écrivant
// dans un slot pré-alloué (architecture SANS MUTEX). errgroup.WithContext +
// SetLimit borne la concurrence à filmChunkParallelism (8). Si une erreur tombe,
// les autres goroutines abortent via egCtx.Done() (propagé à downloadBlob via
// http.Request). L'assemblage final se fait après eg.Wait() — race-free par
// construction.
func (c *HaloAPIClient) fetchFilmChunks(
	ctx context.Context, matchID, caller string, keep func(chunkType int) bool,
) ([]FilmChunk, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}

	var out []FilmChunk
	var toDownload []filmChunk
	for _, chunk := range manifest.CustomData.Chunks {
		if !keep(chunk.ChunkType) {
			continue
		}
		// Cache disque d'abord. Le cache hérité du projet Python ne porte
		// SYSTÉMATIQUEMENT que les REPLICATION_DATA ; l'en-tête et le kill-feed
		// n'y sont que sur les films téléchargés à la main (ex. J0.1). On tente
		// donc pour tous les types : un miss retombe sur le CDN.
		if cached, cErr := c.localFilmCache.LoadChunk(matchID, chunk.Index); cErr == nil && cached != nil {
			out = append(out, FilmChunk{
				Index:      chunk.Index,
				ChunkType:  chunk.ChunkType,
				Data:       cached,
				StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
				DurationMS: chunk.DurationMilliseconds,
			})
			continue
		}
		toDownload = append(toDownload, chunk)
	}

	if len(toDownload) > 0 {
		downloaded := make([]FilmChunk, len(toDownload))
		eg, egCtx := errgroup.WithContext(ctx)
		eg.SetLimit(filmChunkParallelism)
		for i, ch := range toDownload {
			i, ch := i, ch
			eg.Go(func() error {
				chunkURL := buildChunkURL(manifest.BlobStoragePathPrefix, ch.FileRelativePath)
				data, dErr := c.downloadBlob(egCtx, chunkURL)
				if dErr != nil {
					return fmt.Errorf("%s chunk %d(%s): %w", caller, ch.Index, matchID, dErr)
				}
				downloaded[i] = FilmChunk{
					Index:      ch.Index,
					ChunkType:  ch.ChunkType,
					Data:       data,
					StartMS:    ch.ChunkStartTimeOffsetMilliseconds,
					DurationMS: ch.DurationMilliseconds,
				}
				return nil
			})
		}
		if wErr := eg.Wait(); wErr != nil {
			return nil, false, wErr
		}
		out = append(out, downloaded...)
	}

	if len(out) == 0 {
		return nil, false, nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, true, nil
}

// FilmChunkRef : un chunk du film et son URL CDN PRÉ-SIGNÉE, SANS ses octets.
type FilmChunkRef struct {
	Index      int
	ChunkType  int
	StartMS    int
	DurationMS int
	URL        string
}

// GetFilmChunkURLs résout le manifeste d'un match et rend les RÉFÉRENCES de ses
// chunks — sans télécharger un seul octet.
//
// POURQUOI CETTE MÉTHODE EXISTE. C'est la pièce de sécurité du travail délégué
// (piste F §1) : le manifeste exige les tokens Halo, les blobs non (CDN Azure
// pré-signé, sans authentification). Le VPS web résout donc ICI, met les URL
// dans le job, et l'ouvrier distant décode SANS le moindre secret Halo ni le
// moindre accès à la base. Déléguer la résolution du manifeste à l'ouvrier
// l'obligerait à porter un token — c'est exactement ce qu'on refuse.
//
// Retourne (nil, false, nil) si le film est absent (404/410) — cas NORMAL :
// ~29 % des matchs n'ont plus de film côté serveur.
func (c *HaloAPIClient) GetFilmChunkURLs(ctx context.Context, matchID string) ([]FilmChunkRef, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}
	out := make([]FilmChunkRef, 0, len(manifest.CustomData.Chunks))
	for _, chunk := range manifest.CustomData.Chunks {
		url := buildChunkURL(manifest.BlobStoragePathPrefix, chunk.FileRelativePath)
		if url == "" {
			continue // manifeste sans préfixe de blob (cache écrit localement) : rien à servir
		}
		out = append(out, FilmChunkRef{
			Index:      chunk.Index,
			ChunkType:  chunk.ChunkType,
			StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
			DurationMS: chunk.DurationMilliseconds,
			URL:        url,
		})
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, true, nil
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
		if chunk.ChunkType != FilmChunkTypeHighlightEvents {
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

// isNotFoundErr dit si l'erreur signale une ressource ABSENTE côté Halo (404/410) :
// film, skill, CSR inexistants côté API, ou blob disparu du CDN.
//
// TYPÉ uniquement depuis le 2026-09-05 : le repli textuel
// (« la chaîne contient HTTP 404 ») a disparu — le texte d'une erreur n'est pas
// une API, et il suffisait qu'un message change de forme pour que le prédicat
// devienne muet. Les deux seules sources possibles sont typées : *HTTPError
// (doGet) et *BlobHTTPError (downloadBlob), deux types VOLONTAIREMENT distincts
// (cf. BlobHTTPError). Garde-rail : no_text_predicate_test.go.
func isNotFoundErr(err error) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusGone
	}
	var be *BlobHTTPError
	if errors.As(err, &be) {
		return be.StatusCode == http.StatusNotFound || be.StatusCode == http.StatusGone
	}
	return false
}

// downloadBlob télécharge un blob Halo sans header d'auth (pre-signed URL)
// et le décompresse zlib (le CDN Azure des films Halo Infinite renvoie du zlib brut).
// Portage de download_film_chunk() (Python api_client.py:485-498).
