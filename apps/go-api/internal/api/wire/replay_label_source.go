// Package wire — replay_label_source.go : la source des catalogues de libellés du rejeu, câblée
// pour les services qui les lisent à la requête.
package wire

import (
	"fmt"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// replayLabelSource implémente port.ReplayLabelSource : le catalogue d'un titre se lit de ses
// mappings versionnés sous `repoRoot`, et seulement quand le titre déclare la capacité `replay`.
type replayLabelSource struct {
	repoRoot string
	titles   *title.Registry
}

// NewReplayLabelSource rend la source des catalogues de libellés du rejeu, lus sous la racine du
// dépôt. Un titre sans la capacité `replay` n'a pas de catalogue : `games.ErrCapabilityNotSupported`.
func NewReplayLabelSource(repoRoot string) port.ReplayLabelSource {
	return replayLabelSource{repoRoot: repoRoot, titles: title.DefaultRegistry()}
}

// LabelCatalogue rend le catalogue du titre (`replaylabels.Catalogue` : lu une fois par processus,
// partagé en lecture seule).
func (s replayLabelSource) LabelCatalogue(titleSlug string) (replay.LabelCatalog, error) {
	desc := s.titles.Get(titleSlug)
	if desc == nil || !desc.HasCapability(title.CapReplay) {
		return replay.LabelCatalog{}, fmt.Errorf("catalogue de libellés du rejeu de %q : %w",
			titleSlug, games.ErrCapabilityNotSupported)
	}
	return replaylabels.Catalogue(s.repoRoot, titleSlug)
}
