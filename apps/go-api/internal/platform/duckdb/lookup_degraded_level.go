package duckdb

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// degradedLookupLevel : niveau de journal d'une résolution best-effort (libellé, image, mode)
// qui rend un résultat vide après une erreur. Une table absente est un schéma non migré ou une
// base de test : DEBUG, comme tous les chemins qui tolèrent cette absence (isTableNotFoundErr).
// Toute autre panne est un WARN, rétrogradé en DEBUG quand la requête a été abandonnée
// (observability.LevelUnlessCanceled). Le journal PRÉCÈDE la dégradation : une résolution qui
// rend vide sans trace est indiscernable d'un nom inconnu.
func degradedLookupLevel(ctx context.Context, err error) slog.Level {
	if isTableNotFoundErr(err) {
		return slog.LevelDebug
	}
	return observability.LevelUnlessCanceled(ctx, err, slog.LevelWarn)
}
