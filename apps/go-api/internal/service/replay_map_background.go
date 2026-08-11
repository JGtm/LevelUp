package service

// replay_map_background.go — LE FOND DE CARTE DU REJEU 2D.
//
// CE QUE CE FICHIER RÉSOUT. L'artefact de rejeu porte des trajectoires en mètres monde ; les
// 21 fonds de carte figés portent, chacun, le calage qui pose leur image dans CE MÊME repère
// (`replay.MapBackground`). Il ne manquait que le maillon du milieu : quelle carte a été
// jouée. Le film ne la nomme pas, donc le document non plus — la base la nomme.
//
// LA CHAÎNE, ET ELLE N'EST DÉCLARÉE QU'UNE FOIS :
//
//	match -> nom(s) de carte (registre partagé)          ReplayMapNameRepo
//	nom de carte -> module   (map_quant_bounds.json)     filmdec.NormalizeMapName + Lookup
//	module -> image + calage (map_backgrounds/{module})  PathResolver + replay.LoadMapBackground
//
// Le second maillon est celui de `cmd/replay-build` : le lien « nom affiché -> module » n'a
// qu'une source, le catalogue de bornes. En écrire une seconde ici, c'est se garantir deux
// vérités qui divergeront.
//
// OFFLINE PUR : ces trois étapes lisent des fichiers versionnés et une table. Rien n'ouvre
// le jeu, rien ne va sur le réseau.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// MapBackground retourne le calage du fond de carte du match.
func (s *replayService) MapBackground(ctx context.Context, matchID string) (*replay.MapBackground, error) {
	module, err := s.resolveMapModule(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return s.loadMapBackground(ctx, module)
}

// MapBackgroundImage retourne les octets PNG du fond de carte du match.
//
// L'image n'est servie QUE si son sidecar est lisible : une image sans calage ne se superpose
// à rien, et la publier laisserait croire à un fond posé au bon endroit.
func (s *replayService) MapBackgroundImage(ctx context.Context, matchID string) ([]byte, error) {
	module, err := s.resolveMapModule(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if _, err := s.loadMapBackground(ctx, module); err != nil {
		return nil, err
	}
	path := title.NewPathResolver(s.repoRoot).MapBackgroundPath(s.titleSlug, module)
	blob, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	if err != nil {
		return nil, fmt.Errorf("lecture image de fond %s: %w", module, err)
	}
	return blob, nil
}

// resolveMapModule traduit le match en module de carte, en essayant les noms candidats dans
// l'ordre de fiabilité rendu par le repo.
func (s *replayService) resolveMapModule(ctx context.Context, matchID string) (string, error) {
	if s.maps == nil {
		return "", port.ErrMapBackgroundNotAvailable
	}
	names, err := s.maps.MapNamesForMatch(ctx, matchID)
	if err != nil || len(names) == 0 {
		// Journalisé, jamais avalé : une carte qu'on ne sait pas nommer est une donnée
		// manquante, et c'est elle qu'on ira chercher si un fond manque à l'écran.
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — pas de fond",
			"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	catPath := title.NewPathResolver(s.repoRoot).MapQuantBoundsPath(s.titleSlug)
	cat, err := filmdec.LoadMapQuantCatalog(catPath)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue de bornes illisible — pas de fond",
			"err", err, "path", catPath, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	for _, name := range names {
		entry, lookErr := cat.Lookup(name)
		if lookErr == nil && entry.Module != "" {
			return entry.Module, nil
		}
	}
	slog.DebugContext(ctx, "rejeu 2D : aucun module pour les noms de carte candidats",
		"match_id", matchID, "candidats", names, "titleSlug", s.titleSlug)
	return "", port.ErrMapBackgroundNotAvailable
}

// loadMapBackground lit le sidecar de calage d'un module.
func (s *replayService) loadMapBackground(ctx context.Context, module string) (*replay.MapBackground, error) {
	path := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, module)
	bg, err := replay.LoadMapBackground(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Cas NORMAL : seules 21 cartes ont un fond cuit. Le client dégrade sur le
			// sol structurel — ce n'est pas une panne, ce n'est pas un warn.
			slog.DebugContext(ctx, "rejeu 2D : pas de fond figé pour ce module",
				"module", module, "titleSlug", s.titleSlug)
		} else {
			slog.WarnContext(ctx, "rejeu 2D : fond de carte illisible",
				"err", err, "module", module, "path", path, "titleSlug", s.titleSlug)
		}
		return nil, port.ErrMapBackgroundNotAvailable
	}
	return bg, nil
}
