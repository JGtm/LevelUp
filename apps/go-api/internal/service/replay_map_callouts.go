package service

// replay_map_callouts.go — LES ZONES NOMMÉES (« callouts ») DU REJEU 2D.
//
// MÊME MODÈLE QUE LE FOND DE CARTE (replay_map_background.go) : la résolution se fait AU
// SERVICE, sans re-cuisson des artefacts de rejeu — l'artefact ne nomme pas sa carte, la
// base la nomme, et le catalogue versionné porte les zones.
//
// DEUX ESSAIS, DANS CET ORDRE, PARCE QU'IL Y A DEUX FAMILLES DE CARTES :
//
//	match -> map_id + nom(s) de carte (registre partagé)   ReplayMapNameRepo
//	1. nom de carte -> module (map_quant_bounds.json)      filmdec.LoadMapQuantCatalog
//	   module -> zones (map_callouts.json, `maps`)         MapCalloutsCatalog.Lookup
//	2. map_id -> zones (map_callouts.json, `maps_by_id`)   MapCalloutsCatalog.LookupByID
//
// L'ESSAI PAR MODULE PASSE D'ABORD, ET CE N'EST PAS ARBITRAIRE : une carte intégrée a une
// entrée de MEILLEURE qualité (polygones du designer, découpés sur le décor praticable,
// libellés 816/816). Une carte Forge n'a pas de module — l'essai 1 rend une absence propre
// et l'essai 2 la rattrape par son asset UGC.
//
// AVANT LE 2026-09-02 IL N'Y AVAIT QUE L'ESSAI 1, et une carte Forge n'affichait donc
// aucune zone alors que le jeu, lui, les affiche. Ce n'était pas un choix : le catalogue
// n'avait pas d'espace de clés pour elles.
//
// OFFLINE PUR : deux fichiers versionnés et une table. Rien n'ouvre le jeu, rien ne va
// sur le réseau.

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// MapCallouts retourne les zones nommées de la carte du match.
func (s *replayService) MapCallouts(ctx context.Context, matchID string) (*replay.MapCalloutsEntry, error) {
	keys := s.matchMapKeys(ctx, matchID)
	if keys.MapID == "" && len(keys.Names) == 0 {
		// Journalisé, jamais avalé : une carte qu'on ne sait pas nommer est une donnée
		// manquante — la même règle que le fond de carte.
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — pas de zones nommées",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	res := title.NewPathResolver(s.repoRoot)
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(s.titleSlug))
	if err != nil {
		// Le catalogue est VERSIONNÉ : son absence ou son illisibilité n'est pas le cas
		// nominal d'une carte sans zones — on le dit, puis on dégrade.
		slog.WarnContext(ctx, "rejeu 2D : catalogue de callouts illisible — pas de zones nommées",
			"err", err, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	if entry, ok := s.calloutsByModule(ctx, cat, keys); ok {
		return entry, nil
	}
	entry, err := cat.LookupByID(keys.MapID)
	if err != nil {
		if !errors.Is(err, replay.ErrCalloutsUnknownMap) {
			slog.WarnContext(ctx, "rejeu 2D : lookup callouts par map_id en échec",
				"err", err, "map_id", keys.MapID, "titleSlug", s.titleSlug)
		}
		// Ni module ni asset au catalogue : absence propre, le rejeu reste entier.
		slog.DebugContext(ctx, "rejeu 2D : carte hors catalogue de callouts",
			"match_id", matchID, "map_id", keys.MapID, "candidats", keys.Names,
			"titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	return &entry, nil
}

// calloutsByModule tente l'essai 1 : nom de carte -> module -> entrée du catalogue.
//
// Rend `false` pour TOUTE absence (catalogue de bornes illisible, aucun module pour les
// noms candidats, module hors catalogue de callouts) — l'appelant enchaîne sur le map_id.
// Une carte Forge passe ici sans rien trouver, et c'est le cas nominal.
func (s *replayService) calloutsByModule(ctx context.Context, cat *replay.MapCalloutsCatalog,
	keys port.MatchMapKeys) (*replay.MapCalloutsEntry, bool) {
	if len(keys.Names) == 0 {
		return nil, false
	}
	res := title.NewPathResolver(s.repoRoot)
	quant, err := filmdec.LoadMapQuantCatalog(res.MapQuantBoundsPath(s.titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue de bornes illisible — essai par module abandonné",
			"err", err, "titleSlug", s.titleSlug)
		return nil, false
	}
	for _, name := range keys.Names {
		entry, lookErr := quant.Lookup(name)
		if lookErr != nil || entry.Module == "" {
			continue
		}
		zones, cerr := cat.Lookup(entry.Module)
		if cerr == nil {
			return &zones, true
		}
		if !errors.Is(cerr, replay.ErrCalloutsUnknownMap) {
			slog.WarnContext(ctx, "rejeu 2D : lookup callouts par module en échec",
				"err", cerr, "module", entry.Module, "titleSlug", s.titleSlug)
		}
	}
	return nil, false
}
