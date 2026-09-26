package haloclient

// halo_client_film_taille.go — « PRÉSENT » N'EST PAS « COMPLET » AU TÉLÉCHARGEMENT (J2.4 du plan
// de suite de l'audit du décodeur de film, 2026-09-26).
//
// Le manifeste de l'API annonce la taille de chaque blob (`ChunkSize`), et c'est la taille du
// blob BRUT tel que le CDN le sert, avant décompression : le témoin
// `internal/sync/testdata/jgtm_full_match/README.md` compare les blobs bruts téléchargés à
// `ChunkSize` (« 0 mismatch »), et le CDN sert du zlib que [inflateBlob] décompresse. Un blob
// d'une autre taille est refusé ([ErrChunkIncomplet]) AVANT décompression : rien n'est rendu,
// donc rien n'est écrit au cache ; le refus est compté (`film_fetch_chunk_incomplet`) et
// journalisé ; ce n'est pas un film perdu ([IsFilmGoneErr] le laisse passer) — le cycle suivant
// le retente. Pas de retentative immédiate : un CDN qui sert un blob tronqué le resservirait.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability"
)

// compteurChunkIncomplet : blobs de film reçus d'une autre taille que celle annoncée.
const compteurChunkIncomplet = "film_fetch_chunk_incomplet"

// ErrChunkIncomplet : le CDN a servi un blob de film d'une autre taille que celle que le
// manifeste annonce (`ChunkSize`).
type ErrChunkIncomplet struct {
	URL     string
	Annonce int // `ChunkSize` du manifeste (octets du blob brut)
	Recu    int // octets du blob brut reçu
}

func (e *ErrChunkIncomplet) Error() string {
	return fmt.Sprintf("blob de film incomplet (%d octets reçus, %d annoncés) : %s", e.Recu, e.Annonce, e.URL)
}

// essaiBlob : l'état de la boucle de [HaloAPIClient.downloadBlob] au moment d'un 200.
type essaiBlob struct {
	attempt   int
	revalider bool
	start     time.Time
}

// blobRecu traite un blob servi en 200 : contrôle de la taille annoncée (0 = inconnue), puis
// décompression et journal de succès.
func blobRecu(ctx context.Context, blobURL string, raw []byte, tailleAnnoncee int, e essaiBlob) ([]byte, error) {
	if tailleAnnoncee > 0 && len(raw) != tailleAnnoncee {
		observability.IncCounter(compteurChunkIncomplet)
		slog.WarnContext(ctx, "halo_api: blob de film d'une autre taille que celle annoncée — refusé",
			"url", blobURL, "annonce", tailleAnnoncee, "recu", len(raw), "attempts", e.attempt+1)
		return nil, &ErrChunkIncomplet{URL: blobURL, Annonce: tailleAnnoncee, Recu: len(raw)}
	}
	out, err := inflateBlob(raw)
	if err != nil {
		return nil, err
	}
	if e.attempt > 0 {
		observability.AddInt(metricBlobRetrySuccess, 1)
		if e.revalider {
			slog.InfoContext(ctx, "halo_api: downloadBlob 304 puis succès",
				"url", blobURL, "attempts", e.attempt+1)
		}
	}
	slog.DebugContext(ctx, "halo_api: downloadBlob succès",
		"url", blobURL, "bytes_compressed", len(raw), "bytes_inflated", len(out),
		"attempts", e.attempt+1, "duration_ms", time.Since(e.start).Milliseconds())
	return out, nil
}
