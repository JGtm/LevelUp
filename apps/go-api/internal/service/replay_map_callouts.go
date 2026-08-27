package service

// replay_map_callouts.go — LES ZONES NOMMÉES (« callouts ») DU REJEU 2D.
//
// MÊME MODÈLE QUE LE FOND DE CARTE (replay_map_background.go) : la résolution se fait AU
// SERVICE, par map-module, sans re-cuisson des artefacts de rejeu — l'artefact ne nomme
// pas sa carte, la base la nomme, et le catalogue versionné porte les zones.
//
//	match -> nom(s) de carte (registre partagé)      ReplayMapNameRepo
//	nom de carte -> module (map_quant_bounds.json)   filmdec.LoadMapQuantCatalog
//	module -> zones nommées (map_callouts.json)      replay.LoadMapCallouts
//
// PAS D'ESSAI map_id ICI — et ce n'est PLUS un choix, c'est un manque daté. L'en-tête
// affirmait jusqu'au 2026-08-27 qu'une carte Forge n'a aucune zone nommée ; c'est faux, seul
// son CANEVAS n'en porte pas. Ses zones vivent dans son map.mvar (18 sur Isolation), et
// port.MatchMapKeys porte déjà le MapID dont l'essai aurait besoin —
// replay_map_objectives.go s'en sert. Il manque le catalogue côté données : tant qu'il
// n'existe pas, la résolution s'arrête proprement (champ absent, jamais d'erreur) et une
// carte Forge n'affiche pas ses callouts alors que le jeu, lui, les affiche.
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
	if s.maps == nil {
		return nil, port.ErrMapCalloutsNotAvailable
	}
	keys, err := s.maps.MapKeysForMatch(ctx, matchID)
	if err != nil || len(keys.Names) == 0 {
		// Journalisé, jamais avalé : une carte qu'on ne sait pas nommer est une donnée
		// manquante — la même règle que le fond de carte.
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — pas de zones nommées",
			"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	res := title.NewPathResolver(s.repoRoot)
	quant, err := filmdec.LoadMapQuantCatalog(res.MapQuantBoundsPath(s.titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue de bornes illisible — pas de zones nommées",
			"err", err, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	module := ""
	for _, name := range keys.Names {
		if entry, lookErr := quant.Lookup(name); lookErr == nil && entry.Module != "" {
			module = entry.Module
			break
		}
	}
	if module == "" {
		slog.DebugContext(ctx, "rejeu 2D : aucun module pour les identités de carte candidates",
			"match_id", matchID, "candidats", keys.Names, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(s.titleSlug))
	if err != nil {
		// Le catalogue est VERSIONNÉ : son absence ou son illisibilité n'est pas le cas
		// nominal d'une carte sans zones — on le dit, puis on dégrade.
		slog.WarnContext(ctx, "rejeu 2D : catalogue de callouts illisible — pas de zones nommées",
			"err", err, "titleSlug", s.titleSlug)
		return nil, port.ErrMapCalloutsNotAvailable
	}
	entry, err := cat.Lookup(module)
	if err != nil {
		if !errors.Is(err, replay.ErrCalloutsUnknownMap) {
			slog.WarnContext(ctx, "rejeu 2D : lookup callouts en échec",
				"err", err, "module", module, "titleSlug", s.titleSlug)
		}
		// Module hors catalogue = canevas Forge ou carte non couverte : absence propre.
		return nil, port.ErrMapCalloutsNotAvailable
	}
	return &entry, nil
}
