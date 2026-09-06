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
//	nom de carte -> clé de fond publiée                    replay.MapBackgroundIndex
//
// POURQUOI DEUX CLÉS. Une carte Forge communautaire vit sur un CANEVAS (8 installés)
// partagé par des dizaines de cartes : le module ne peut pas keyer son fond, seul son
// map_id (asset UGC, présent sur chaque match) la désigne. Une carte native, elle, reste
// keyée par son dossier installé. L'essai map_id se fait D'ABORD : un fond sous cette clé
// n'existe que pour la carte exacte du match.
//
// POURQUOI UN INDEX DE NOMS ET PLUS LE CATALOGUE DE BORNES. L'essai map_id échoue dès que la
// carte a été REPUBLIÉE sous un nouvel asset depuis la cuisson de son fond — mesuré le
// 2026-08-27 : Salvation, Dynasty, Shogun, Houseki, Starboard et Shiro, jouées sous un map_id
// mort, s'affichaient sans fond alors que leur image existe. Le repli historique passait par
// `map_quant_bounds.json`, qui envoie un nom de carte Forge vers son CANEVAS — un module qui
// n'a jamais de fond publié, donc un cul-de-sac par construction. L'index de fonds
// (`replay.MapBackgroundIndex`) lit à la place les identités que la cuisson a DÉJÀ écrites dans
// chaque sidecar. Mesuré sur les 123 map_id du registre : l'index résout tout ce que le
// catalogue de bornes résolvait (zéro divergence, zéro carte résolue par lui seul) et 9 map_id
// de plus. Le repli par bornes est donc retiré : deux chemins pour le même lien, dont un mort,
// sont deux vérités qui divergeront.
//
// OFFLINE PUR : ces étapes lisent des fichiers versionnés et une table. Rien n'ouvre le
// jeu, rien ne va sur le réseau.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/replayview"
)

// MapBackground retourne le calage du fond de carte du match, dans sa forme SERVIE : le
// sidecar est lu tel qu'il est ecrit sur disque (`analysis/replay`) puis projete sur le
// contrat public (`domain/replaydoc`), comme le document lui-meme.
func (s *replayService) MapBackground(ctx context.Context, matchID string) (*replaydoc.MapBackground, error) {
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, err
	}
	bg, err := s.loadMapBackground(ctx, key)
	if err != nil {
		return nil, err
	}
	return replayview.MapBackgroundOf(bg), nil
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
// existe sous cette clé (cartes Forge encore publiées sous leur asset du jour), sinon la clé
// que l'index des fonds attache à l'une des identités candidates (cartes natives, et cartes
// republiées depuis la cuisson de leur fond).
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
	// 2. Le repli par NOM — l'index des identités que la cuisson a écrites dans les sidecars.
	// Il rattrape les cartes republiées sous un nouvel asset (le map_id de l'étape 1 est mort)
	// et les cartes natives, dont le fond est keyé par module installé.
	dir := title.NewPathResolver(s.repoRoot).MapBackgroundDir(s.titleSlug)
	idx, err := replay.MapBackgroundIndexFor(dir)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : index des fonds indisponible — pas de fond",
			"err", err, "path", dir, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	for _, name := range keys.Names {
		if cle, ok := idx.Lookup(name); ok {
			return cle, nil
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
