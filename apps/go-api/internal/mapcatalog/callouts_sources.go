package mapcatalog

// callouts_sources.go — la lecture des sources de la cascade des zones SANS CACHE : une fois
// par cycle de rattrapage ou par passe de CLI.
//
// Le service lit les MÊMES fichiers par ses propres caches (un décodage par processus pour les
// fichiers versionnés, à empreinte de fichier pour le catalogue généré, qui change au fil de
// l'eau) : `service/replay_map_callouts.go`. Ce sont les deux seuls assembleurs des sources ;
// un troisième se factorise.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// ChargerSourcesDeZones lit les sources de la cascade d'un titre. Une source illisible est
// JOURNALISÉE et laissée vide — l'essai correspondant rend une absence — jamais fatale : sans
// catalogue des bornes, une carte Forge se résout encore par son map_id. Le catalogue généré
// absent est le cas nominal d'une instance neuve et ne se journalise pas.
func ChargerSourcesDeZones(ctx context.Context, res *title.PathResolver, titleSlug string) SourcesDeZones {
	var s SourcesDeZones
	versionne := res.MapCalloutsPath(titleSlug)
	if cat, err := replay.LoadMapCallouts(versionne); err != nil {
		slog.WarnContext(ctx, "zones nommées : catalogue versionné illisible",
			"err", err, "path", versionne, "titleSlug", titleSlug)
	} else {
		s.Versionne = cat
	}
	genere := res.MapCalloutsOverlayPath(titleSlug)
	if cat, err := ChargerCatalogueGenere(genere, titleSlug); err != nil {
		slog.WarnContext(ctx, "zones nommées : catalogue généré illisible — seules les cartes "+
			"versionnées comptent", "err", err, "path", genere, "titleSlug", titleSlug)
	} else {
		s.Genere = cat
	}
	bornes := res.MapQuantBoundsPath(titleSlug)
	if cat, err := decfilm.LoadMapQuantCatalog(bornes); err != nil {
		slog.WarnContext(ctx, "zones nommées : catalogue des bornes illisible — essai par module "+
			"abandonné", "err", err, "path", bornes, "titleSlug", titleSlug)
	} else {
		s.Bornes = cat
	}
	fonds := res.MapBackgroundDir(titleSlug)
	if idx, err := replay.MapBackgroundIndexFor(ctx, fonds); err != nil {
		slog.WarnContext(ctx, "zones nommées : index des fonds illisible — identités déclarées "+
			"ignorées", "err", err, "path", fonds, "titleSlug", titleSlug)
	} else {
		s.Identites = idx
	}
	return s
}
