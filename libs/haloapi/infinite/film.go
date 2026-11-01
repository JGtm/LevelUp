package infinite

// film.go — manifeste d'un film, références de fragments et téléchargement parallèle (`errgroup`)
// de ses fragments. Les octets rendus sont ceux du CDN, COMPRESSÉS : la décompression, la
// règle « seul un film finalisé se télécharge », la version de film et le cache local sont à
// `sync/filmfetch`.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"
)

// FilmChunkParallelism : nombre maximal de téléchargements CDN simultanés d'un film (un film a
// typiquement 10 à 30 fragments de 200 à 500 ms chacun ; le limiteur du client borne le flux).
const FilmChunkParallelism = 8

// FilmManifestDTO : la réponse brute de `GET /{jeu}/films/matches/{id}/spectate`.
type FilmManifestDTO struct {
	BlobStoragePathPrefix string `json:"BlobStoragePathPrefix"`
	CustomData            struct {
		FilmMajorVersion int                `json:"FilmMajorVersion"`
		Chunks           []FilmChunkMetaDTO `json:"Chunks"`
	} `json:"CustomData"`
}

// FilmChunkMetaDTO : un fragment décrit par le manifeste (pas ses octets).
type FilmChunkMetaDTO struct {
	Index                            int    `json:"Index"`
	ChunkType                        int    `json:"ChunkType"`
	ChunkSize                        int    `json:"ChunkSize"`
	ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
	DurationMilliseconds             int    `json:"DurationMilliseconds"`
	FileRelativePath                 string `json:"FileRelativePath"`
}

// GetFilmManifest récupère le manifeste du film d'un match ([CleUGCFilm]). Un film absent
// (404/410, cas normal des vieux matchs) est rendu typé ([IsFilmGoneErr]).
func (c *Client) GetFilmManifest(ctx context.Context, auth Auth, matchID string) (*FilmManifestDTO, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, fmt.Errorf("fetchFilmManifest: matchID invalide %q", matchID)
	}
	endpoint := fmt.Sprintf("%s/films/matches/%s/spectate", c.endpoints.base(CleUGCFilm), url.PathEscape(matchID))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("fetchFilmManifest(%s): %w", matchID, err)
	}
	var m FilmManifestDTO
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("fetchFilmManifest decode(%s): %w", matchID, err)
	}
	return &m, nil
}

// BuildChunkURL construit l'adresse d'un fragment depuis le préfixe de blob et le chemin relatif.
func BuildChunkURL(blobPrefix, fileRelativePath string) string {
	name := strings.TrimLeft(fileRelativePath, "/")
	if name == "" {
		return blobPrefix
	}
	if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
		return blobPrefix + "/" + name
	}
	return blobPrefix + name
}

// FilmChunkRef : un fragment du film et son adresse CDN pré-signée, SANS ses octets (le CDN est
// public : un tiers télécharge sans jeton Halo).
type FilmChunkRef struct {
	Index      int
	ChunkType  int
	StartMS    int
	DurationMS int
	URL        string
}

// Refs rend les références des fragments du manifeste, triées par index ; un fragment sans adresse
// (manifeste sans préfixe de blob) est omis.
func (m *FilmManifestDTO) Refs() []FilmChunkRef {
	out := make([]FilmChunkRef, 0, len(m.CustomData.Chunks))
	for _, ch := range m.CustomData.Chunks {
		u := BuildChunkURL(m.BlobStoragePathPrefix, ch.FileRelativePath)
		if u == "" {
			continue
		}
		out = append(out, FilmChunkRef{Index: ch.Index, ChunkType: ch.ChunkType,
			StartMS: ch.ChunkStartTimeOffsetMilliseconds, DurationMS: ch.DurationMilliseconds, URL: u})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// FragmentBrut : un fragment téléchargé, octets COMPRESSÉS tels que servis par le CDN.
type FragmentBrut struct {
	Index      int
	ChunkType  int
	Brut       []byte
	StartMS    int
	DurationMS int
	// Essais : le nombre d'essais qu'a coûté le blob (> 1 : obtenu grâce à une reprise).
	Essais int
}

// DownloadChunks télécharge en parallèle ([FilmChunkParallelism]) les fragments `chunks` du film
// dont les blobs sont sous `blobPrefix`, et rend leurs octets compressés DANS L'ORDRE de `chunks`.
// La première erreur annule les téléchargements en cours et est rendue, préfixée par `appelant`,
// l'index du fragment et `matchID`.
func (c *Client) DownloadChunks(ctx context.Context, blobPrefix string, chunks []FilmChunkMetaDTO, appelant, matchID string) ([]FragmentBrut, error) {
	out := make([]FragmentBrut, len(chunks))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(FilmChunkParallelism)
	for i, ch := range chunks {
		eg.Go(func() error {
			brut, essais, err := c.DownloadBlob(egCtx, BuildChunkURL(blobPrefix, ch.FileRelativePath), ch.ChunkSize)
			if err != nil {
				return fmt.Errorf("%s chunk %d(%s): %w", appelant, ch.Index, matchID, err)
			}
			out[i] = FragmentBrut{Index: ch.Index, ChunkType: ch.ChunkType, Brut: brut,
				StartMS: ch.ChunkStartTimeOffsetMilliseconds, DurationMS: ch.DurationMilliseconds, Essais: essais}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
