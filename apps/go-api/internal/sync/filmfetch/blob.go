package filmfetch

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/observability"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
)

// Compteurs expvar du téléchargement des blobs de film (convention observability :
// "<categorie>_<sous_cle>" en snake_case). La brique les signale par ses crochets ; ce paquet les
// compte.
const (
	// metricBlobRetrySuccess : blobs obtenus grâce à une nouvelle tentative — sans elle, c'était
	// un film déclaré illisible.
	metricBlobRetrySuccess = "halo_api_blob_retry_success"
	// metricBlobRetryExhausted : abandons après épuisement des tentatives.
	metricBlobRetryExhausted = "halo_api_blob_retry_exhausted"
	// compteurChunkIncomplet : blobs reçus d'une autre taille que celle annoncée, refusés.
	compteurChunkIncomplet = "film_fetch_chunk_incomplet"
	// cleJournalBlobReseau : clé anti-rafale du journal d'abandon sur échec réseau (une panne fait
	// échouer tous les téléchargements de blobs en rafale).
	cleJournalBlobReseau = "log_throttle_halo_api_download_blob"
)

// crochets : les compteurs et le journal limité que ce paquet pose sur le client des blobs.
func crochets() infinite.Crochets {
	return infinite.Crochets{
		BlobRepriseReussie: func() { observability.AddInt(metricBlobRetrySuccess, 1) },
		BlobEpuise:         func() { observability.AddInt(metricBlobRetryExhausted, 1) },
		BlobIncomplet:      func() { observability.IncCounter(compteurChunkIncomplet) },
		JournalBlobReseau:  journalBlobReseau,
	}
}

// journalBlobReseau journalise l'abandon d'un blob sur échec RÉSEAU, au plus une fois par fenêtre
// de rafale ; le nombre de journaux étouffés depuis le dernier est porté par le suivant.
func journalBlobReseau(ctx context.Context, blobURL string, essais int, err error) {
	if allow, since := observability.AllowThrottledLog(cleJournalBlobReseau, observability.NetworkFloodWindow); allow {
		slog.WarnContext(ctx, "halo_api: downloadBlob échec réseau",
			"url", blobURL, "attempts", essais, "err", err, "throttled_since_last", since)
	}
}

// downloadBlob télécharge un blob du CDN par la brique (reprise, taille annoncée contrôlée) et le
// décompresse. `tailleAnnoncee` (> 0) est la taille du blob BRUT que le manifeste annonce ; 0 =
// aucun contrôle.
func (f *Films) downloadBlob(ctx context.Context, blobURL string, tailleAnnoncee int) ([]byte, error) {
	brut, err := f.blobs.DownloadBlob(ctx, blobURL, tailleAnnoncee)
	if err != nil {
		return nil, err
	}
	return inflateBlob(brut)
}

// inflateBlob décompresse le corps zlib d'un blob (le CDN des films sert le zlib brut, format de
// stockage des films, sans `Content-Encoding`).
func inflateBlob(raw []byte) ([]byte, error) {
	out, err := decfilm.Decompresser(raw)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob: %w", err)
	}
	return out, nil
}
