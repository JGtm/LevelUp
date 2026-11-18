package replayregen

// raster.go — le fichier tactique d'un rejeu dont le film a quitté l'archive. Le rejeu rangé ne se refait plus (il n'y a
// plus de film), mais son fichier tactique ne dépend que du rejeu : la file le reprojette sans lire aucun film et sans
// toucher au rejeu ([Deps.RebuildRaster]). Sans cela, un fichier tactique d'un ancien format ne serait jamais refait et
// le match resterait « non mesuré » dans l'onglet Tactique.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// Compteurs expvar des fichiers tactiques refaits sans film.
const (
	MetricRasterDone   = "replay_regen_raster_done_total"
	MetricRasterFailed = "replay_regen_raster_failed_total"
)

// processRaster refait le fichier tactique d'UN rejeu sans film : relit son état (un fichier déjà refait entre-temps se
// saute), reprojette, compte. Un échec se journalise, se compte et reste visible comme tel ; un arrêt du serveur n'est ni
// un succès ni un échec (repris à l'ouverture suivante).
func (s *Service) processRaster(ctx context.Context, c Candidate) {
	defer s.addDone()
	if !s.deps.RasterStale(ctx, c.TitleSlug, c.MatchID) {
		return // idempotence : déjà refait
	}
	err := s.deps.RebuildRaster(ctx, c.TitleSlug, c.MatchID)
	switch {
	case err == nil:
		observability.IncCounter(MetricRasterDone)
		slog.InfoContext(ctx, "replay_regen: fichier tactique refait sans film",
			"titleSlug", c.TitleSlug, "match_id", c.MatchID)
	case ctx.Err() != nil:
		// arrêt du serveur
	default:
		observability.IncCounter(MetricRasterFailed)
		s.mu.Lock()
		s.failed++
		s.mu.Unlock()
		slog.WarnContext(ctx, "replay_regen: fichier tactique non refait",
			"titleSlug", c.TitleSlug, "match_id", c.MatchID, "err", err)
	}
}
