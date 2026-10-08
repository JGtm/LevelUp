package service

// replay_map_callouts.go — LES ZONES NOMMÉES (« callouts ») DU REJEU 2D.
//
// MÊME MODÈLE QUE LE FOND DE CARTE (replay_map_background.go) : la résolution se fait AU
// SERVICE, sans re-cuisson des artefacts de rejeu — l'artefact ne nomme pas sa carte, la
// base la nomme, et les catalogues portent les zones.
//
//	match -> map_id + nom(s) de carte (registre partagé)   ReplayMapNameRepo
//	identités -> zones                                     mapcatalog.SourcesDeZones.Resoudre
//
// LA CASCADE N'EST PAS ÉCRITE ICI : elle vit dans `mapcatalog` (callouts_resolution.go), parce
// que le rattrapage des zones au fetch de film l'interroge aussi pour savoir si une carte a déjà
// ses zones. Ce fichier ne fait que lui fournir ses sources, par des caches de processus :
//
//	catalogue VERSIONNÉ (map_callouts.json)        décodé une fois par processus
//	catalogue GÉNÉRÉ (generated/map_callouts.json) relu quand le fichier change : le rattrapage
//	                                                l'enrichit au fil de l'eau, une carte ajoutée
//	                                                s'affiche sans redémarrage
//	catalogue des bornes (map_quant_bounds.json)   décodé une fois par processus
//	index des identités des fonds publiés          replay.MapBackgroundIndexFor (son propre cache)
//
// OFFLINE PUR : des fichiers et une table. Rien n'ouvre le jeu, rien ne va sur le réseau.

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/replayview"
)

// MapCallouts retourne les zones nommées de la carte du match, dans sa forme SERVIE : le
// catalogue versionné est lu dans sa forme de fichier puis projeté sur le contrat public.
func (s *replayService) MapCallouts(ctx context.Context, matchID string) (*replaydoc.MapCalloutsEntry, error) {
	keys := s.matchMapKeys(ctx, matchID)
	if keys.MapID == "" && len(keys.Names) == 0 {
		// Journalisé, jamais avalé : une carte qu'on ne sait pas nommer est une donnée
		// manquante — la même règle que le fond de carte.
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — pas de zones nommées",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	entry, ok := zonesPourIdentites(ctx, s.repoRoot, s.titleSlug, keys)
	if !ok {
		slog.DebugContext(ctx, "rejeu 2D : carte hors catalogue de callouts",
			"match_id", matchID, "map_id", keys.MapID, "candidats", keys.Names,
			"titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	return replayview.MapCalloutsOf(entry), nil
}

// zonesPourIdentites resout les zones nommees d'une carte a partir de ses identites, par la
// cascade de `mapcatalog.SourcesDeZones` (module, map_id versionne, map_id genere, identite
// declaree).
//
// ELLE EST PARTAGEE, ET C'EST LA REGLE DU DEPOT : deux surfaces la consomment — le rejeu 2D
// (par MATCH, ci-dessus) et l'onglet Tactique (par CARTE, cf. tactical_callouts.go), qui
// nomme ses grappes de reapparition par le callout le plus proche. En deux exemplaires, une
// carte Forge aurait fini par avoir des zones d'un cote et pas de l'autre.
func zonesPourIdentites(ctx context.Context, repoRoot, titleSlug string,
	keys port.MatchMapKeys) (*replay.MapCalloutsEntry, bool) {
	sources, ok := sourcesDeZones(ctx, title.NewPathResolver(repoRoot), titleSlug)
	if !ok {
		return nil, false
	}
	entry, _, ok := sources.Resoudre(mapcatalog.IdentitesDeCarte{MapID: keys.MapID, Noms: keys.Names})
	if !ok {
		return nil, false
	}
	return &entry, true
}

// sourcesDeZones assemble les sources de la cascade depuis les caches du processus.
//
// Le catalogue VERSIONNÉ est le seul indispensable : son absence ou son illisibilité n'est pas
// le cas nominal d'une carte sans zones (installation incomplète, titre sans catalogue) — on le
// dit, puis on dégrade. Les autres sources manquantes ne font que fermer leur essai.
func sourcesDeZones(ctx context.Context, res *title.PathResolver, titleSlug string) (mapcatalog.SourcesDeZones, bool) {
	var s mapcatalog.SourcesDeZones
	cat, err := catalogueDeCallouts(res.MapCalloutsPath(titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "callouts : catalogue illisible — pas de zones nommées",
			"err", err, "titleSlug", titleSlug)
		return s, false
	}
	s.Versionne = cat
	genere := res.MapCalloutsOverlayPath(titleSlug)
	switch g, gerr := catalogueGenere(genere); {
	case gerr == nil:
		s.Genere = g
	case !errors.Is(gerr, fs.ErrNotExist):
		// Absent = aucune carte rattrapée (instance neuve), et ça ne se journalise pas.
		slog.WarnContext(ctx, "callouts : catalogue généré illisible — seules les cartes versionnées sont servies",
			"err", gerr, "path", genere, "titleSlug", titleSlug)
	}
	if quant, qerr := catalogueDeBornes(res.MapQuantBoundsPath(titleSlug)); qerr != nil {
		slog.WarnContext(ctx, "callouts : catalogue de bornes illisible — essai par module abandonné",
			"err", qerr, "titleSlug", titleSlug)
	} else {
		s.Bornes = quant
	}
	fonds := res.MapBackgroundDir(titleSlug)
	if idx, ierr := replay.MapBackgroundIndexFor(ctx, fonds); ierr != nil {
		slog.DebugContext(ctx, "callouts : index des fonds indisponible — identités déclarées ignorées",
			"err", ierr, "path", fonds, "titleSlug", titleSlug)
	} else {
		s.Identites = idx
	}
	return s, true
}

// chargerCallouts et chargerBornes lisent et décodent un catalogue ; variables de paquet pour que les
// tests comptent les lectures du fichier.
var (
	chargerCallouts = replay.LoadMapCallouts
	chargerBornes   = decfilm.LoadMapQuantCatalog
)

// Les catalogues de callouts et de bornes des cartes, décodés une fois par chemin et par processus
// (catalogue_cache.go) ; le catalogue GÉNÉRÉ, relu quand son fichier change.
var (
	cataloguesDeCallouts = nouveauCacheParChemin(func(chemin string) (*replay.MapCalloutsCatalog, error) {
		return chargerCallouts(chemin)
	})
	cataloguesDeBornes = nouveauCacheParChemin(func(chemin string) (*decfilm.MapQuantCatalog, error) {
		return chargerBornes(chemin)
	})
	cataloguesGeneres = nouveauCacheAEmpreinte(func(chemin string) (*replay.MapCalloutsCatalog, error) {
		return chargerCallouts(chemin)
	})
)

// catalogueDeCallouts rend le catalogue de callouts décodé d'un chemin (plusieurs Mo de JSON).
func catalogueDeCallouts(chemin string) (*replay.MapCalloutsCatalog, error) {
	return cataloguesDeCallouts.lire(chemin)
}

// catalogueGenere rend le catalogue généré des zones, relu quand le fichier a changé.
// `fs.ErrNotExist` quand il n'existe pas.
func catalogueGenere(chemin string) (*replay.MapCalloutsCatalog, error) {
	return cataloguesGeneres.lire(chemin)
}

// catalogueDeBornes rend le catalogue des bornes des cartes (`map_quant_bounds.json`) décodé d'un
// chemin.
func catalogueDeBornes(chemin string) (*decfilm.MapQuantCatalog, error) {
	return cataloguesDeBornes.lire(chemin)
}
