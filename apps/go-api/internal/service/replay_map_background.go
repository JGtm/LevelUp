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
//	match -> map_id + nom(s) de carte (registre partagé)   ReplayMapNameRepo
//	map_id -> image + calage (map_backgrounds/{map_id})    la clé des cartes FORGE
//	nom de carte -> module   (map_quant_bounds.json)       filmdec.NormalizeMapName + Lookup
//	module -> image + calage (map_backgrounds/{module})    le repli des cartes NATIVES
//
// POURQUOI DEUX CLÉS. Une carte Forge communautaire vit sur un CANEVAS (8 installés)
// partagé par des dizaines de cartes : le module ne peut pas keyer son fond, seul son
// map_id (asset UGC, présent sur chaque match) la désigne. Une carte native, elle, reste
// keyée par son dossier installé. L'essai map_id se fait D'ABORD : un fond sous cette clé
// n'existe que pour la carte exacte du match.
//
// Le maillon nom -> module est celui de `cmd/replay-build` : ce lien n'a qu'une source, le
// catalogue de bornes. En écrire une seconde ici, c'est se garantir deux vérités qui
// divergeront.
//
// OFFLINE PUR : ces étapes lisent des fichiers versionnés et une table. Rien n'ouvre le
// jeu, rien ne va sur le réseau.

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
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return s.loadMapBackground(ctx, key)
}

// MapBackgroundImage retourne les octets PNG du fond de carte du match.
//
// L'image n'est servie QUE si son sidecar est lisible : une image sans calage ne se superpose
// à rien, et la publier laisserait croire à un fond posé au bon endroit.
func (s *replayService) MapBackgroundImage(ctx context.Context, matchID string) ([]byte, error) {
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if _, err := s.loadMapBackground(ctx, key); err != nil {
		return nil, err
	}
	path := title.NewPathResolver(s.repoRoot).MapBackgroundPath(s.titleSlug, key)
	blob, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	if err != nil {
		return nil, fmt.Errorf("lecture image de fond %s: %w", key, err)
	}
	return blob, nil
}

// resolveBackgroundKey traduit le match en clé de fond : le map_id du match quand un fond
// existe sous cette clé (cartes Forge), sinon le module installé résolu par les noms
// candidats (cartes natives).
func (s *replayService) resolveBackgroundKey(ctx context.Context, matchID string) (string, error) {
	if s.maps == nil {
		return "", port.ErrMapBackgroundNotAvailable
	}
	keys, err := s.maps.MapKeysForMatch(ctx, matchID)
	if err != nil || (keys.MapID == "" && len(keys.Names) == 0) {
		// Journalisé, jamais avalé : une carte qu'on ne sait pas nommer est une donnée
		// manquante, et c'est elle qu'on ira chercher si un fond manque à l'écran.
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — pas de fond",
			"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	// 1. La clé map_id — celle des cartes Forge. La PRÉSENCE du sidecar décide : un fond
	// sous cette clé désigne la carte exacte du match, jamais son canevas.
	if keys.MapID != "" {
		p := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, keys.MapID)
		if _, statErr := os.Stat(p); statErr == nil {
			return keys.MapID, nil
		}
	}
	// 2. Le repli module — celui des cartes natives, via le catalogue de bornes.
	catPath := title.NewPathResolver(s.repoRoot).MapQuantBoundsPath(s.titleSlug)
	cat, err := filmdec.LoadMapQuantCatalog(catPath)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue de bornes illisible — pas de fond",
			"err", err, "path", catPath, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	for _, name := range keys.Names {
		entry, lookErr := cat.Lookup(name)
		if lookErr == nil && entry.Module != "" {
			return entry.Module, nil
		}
	}
	slog.DebugContext(ctx, "rejeu 2D : aucun fond pour les identités de carte candidates",
		"match_id", matchID, "map_id", keys.MapID, "candidats", keys.Names, "titleSlug", s.titleSlug)
	return "", port.ErrMapBackgroundNotAvailable
}

// loadMapBackground lit le sidecar de calage d'une clé de fond (module installé ou map_id).
func (s *replayService) loadMapBackground(ctx context.Context, key string) (*replay.MapBackground, error) {
	path := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, key)
	bg, err := replay.LoadMapBackground(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Cas NORMAL : toutes les cartes n'ont pas un fond cuit. Le client dégrade sur
			// le sol structurel — ce n'est pas une panne, ce n'est pas un warn.
			slog.DebugContext(ctx, "rejeu 2D : pas de fond figé pour cette clé",
				"cle", key, "titleSlug", s.titleSlug)
		} else {
			slog.WarnContext(ctx, "rejeu 2D : fond de carte illisible",
				"err", err, "cle", key, "path", path, "titleSlug", s.titleSlug)
		}
		return nil, port.ErrMapBackgroundNotAvailable
	}
	return bg, nil
}
