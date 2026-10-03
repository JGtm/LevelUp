package service

// tactical_callouts.go — LES ZONES NOMMEES D'UNE CARTE, pour nommer les grappes de spawn.
//
// L'onglet Tactique raisonne PAR CARTE, le rejeu PAR MATCH — mais la resolution est la
// MEME, et elle reste ecrite une seule fois (`zonesPourIdentites`, replay_map_callouts.go) :
// essai par module d'abord (cartes integrees, polygones du designer), puis par asset UGC
// (cartes Forge). Les identites de carte viennent de `MapKeysForMap`, pose en phase 4.4
// pour exactement ce besoin.
//
// POURQUOI UN PORT ET NON UN APPEL AU SERVICE DE REJEU : un service qui en appelle un autre
// est le couplage horizontal que `arch-rules` interdit. Le contrat est donc etroit — une
// carte, ses zones — et le service Tactique reste testable sans disque ni catalogue.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

// tacticalCalloutsStore lit les zones nommees d'une carte d'un titre.
type tacticalCalloutsStore struct {
	repoRoot  string
	titleSlug string
	maps      port.ReplayMapNameRepo
}

// NewTacticalCalloutsStore construit le lecteur de zones nommees d'un titre.
//
// `maps` nil (titre sans lecteur de cartes) : aucune zone, donc des grappes MUETTES — la
// lecture reste servie, elle ne nomme simplement rien.
func NewTacticalCalloutsStore(repoRoot, titleSlug string, maps port.ReplayMapNameRepo) port.TacticalCalloutsStore {
	return &tacticalCalloutsStore{repoRoot: repoRoot, titleSlug: titleSlug, maps: maps}
}

// ZonesDeLaCarte rend les zones nommees d'une carte, projetees sur le type PUR du
// rasterisage.
//
// UNE ABSENCE N'EST JAMAIS UNE ERREUR : toutes les cartes ne sont pas au catalogue (une
// carte Forge non extraite, une carte hors rotation). Les grappes sortent alors sans nom —
// jamais un nom de repli, que le jeu ne prononcerait pas.
func (s *tacticalCalloutsStore) ZonesDeLaCarte(ctx context.Context, mapID string) []domain.ZoneNommee {
	if s.maps == nil || mapID == "" {
		return nil
	}
	keys, err := s.maps.MapKeysForMap(ctx, mapID)
	if err != nil {
		// JAMAIS AVALEE : sans ce journal, des grappes muettes par PANNE de lecture sont
		// indiscernables de grappes muettes parce que la carte est hors catalogue (revue
		// P1-4). Les deux se degradent pareil a l'ecran, pas dans les logs.
		slog.WarnContext(ctx, "tactique: identites de carte illisibles — grappes sans nom",
			"err", err, "map_id", mapID, "titleSlug", s.titleSlug)
		return nil
	}
	entry, ok := zonesPourIdentites(ctx, s.repoRoot, s.titleSlug, keys)
	if !ok {
		return nil
	}
	return zonesNommees(entry.Zones)
}

// zonesNommees projette les zones du catalogue en gardant LES DEUX LANGUES.
//
// LE NOM D'UN LIEU N'EST PAS UNE CHAINE D'INTERFACE : il vient du catalogue du jeu, pas de
// l'i18n du produit, et le client ne peut pas le traduire. Servir une seule langue aurait
// donc fige la moitie des joueurs sur l'autre. Les deux voyagent, le web choisit (revue P2).
//
// LE REPLI EST PAR LANGUE : chacune retombe sur le nom de CONCEPTION quand son libelle
// manque (le lexique FR ne couvre pas encore tout le vocabulaire Forge). Une zone muette
// des trois cotes est ECARTEE : elle ne peut nommer personne, et la garder ferait d'elle la
// « plus proche » d'une grappe qu'elle laisserait sans nom.
func zonesNommees(zones []replay.CalloutZone) []domain.ZoneNommee {
	out := make([]domain.ZoneNommee, 0, len(zones))
	for _, z := range zones {
		fr, en := repliDeNom(z.FR, z.Name), repliDeNom(z.EN, z.Name)
		if fr == "" && en == "" {
			continue
		}
		out = append(out, domain.ZoneNommee{NomFR: fr, NomEN: en, X: z.X, Y: z.Y})
	}
	return out
}

// repliDeNom rend le libelle, ou le nom de conception a defaut.
func repliDeNom(libelle, conception string) string {
	if libelle != "" {
		return libelle
	}
	return conception
}
