package wire

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// signalerLigneIllisible journalise et compte une ligne dont le Scan a échoué dans une lecture
// best-effort du câblage ; la ligne est ignorée, la lecture continue. Une lecture qui perd des
// lignes sans le dire sert un résultat partiel pour un résultat complet.
func signalerLigneIllisible(ctx context.Context, titleSlug, lecture string, err error) {
	observability.IncCounterT(titleSlug, "wire_rows_scan_failed_total")
	slog.WarnContext(ctx, "wire: ligne illisible ignorée", "lecture", lecture, "titleSlug", titleSlug, "err", err)
}
