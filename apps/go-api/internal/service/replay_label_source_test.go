package service

import (
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// labelsSousRacine : la source des catalogues de libellés du rejeu lue des mappings versionnés sous
// `root`, comme le câblage de production (sans sa porte de capacité).
type labelsSousRacine struct{ root string }

func (s labelsSousRacine) LabelCatalogue(titleSlug string) (replay.LabelCatalog, error) {
	return replaylabels.Catalogue(s.root, titleSlug)
}

// labelsDe rend la source des catalogues lus sous `root`.
func labelsDe(root string) port.ReplayLabelSource { return labelsSousRacine{root: root} }
