package tactical

// placement.go — LE PLACEMENT D'UNE MORT : seul, ou près d'un coéquipier, à l'instant de la mort.
//
// Le FAIT est établi au sync (`match_death_context`) : la distance au coéquipier VISIBLE le plus
// proche, ou son absence. Ce fichier retrouve la ligne de contexte d'une mort donnée et la
// compare à la portée du radar du match, par la comparaison unique `coordination.APortee`.
//
// Pur : aucune base, aucun fichier.

import (
	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// TolerancePlacementMs est l'écart d'instant, en millisecondes et bornes comprises, en deçà
// duquel une ligne de contexte est celle d'une mort : le journal des morts et le contexte sont
// deux lectures du même film sur l'horloge du match, et une mort se retrouve à la même
// milliseconde au cas nominal ; la tolérance absorbe un arrondi d'un producteur à l'autre.
const TolerancePlacementMs int64 = 1500

// ContexteLePlusProche rend la ligne de contexte de la mort de `victime` dans `matchID` à
// l'instant `t` : la plus proche dans le temps à ± TolerancePlacementMs (à égalité d'écart, la
// plus ancienne), ou nil.
func ContexteLePlusProche(contextes []domain.ContexteDeMort, matchID, victime string, t int64) *domain.ContexteDeMort {
	var meilleur *domain.ContexteDeMort
	var ecartMin int64
	for i := range contextes {
		c := &contextes[i]
		if c.MatchID != matchID || c.VictimXUID != victime {
			continue
		}
		ecart := c.TimeMs - t
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > TolerancePlacementMs {
			continue
		}
		if meilleur == nil || ecart < ecartMin || (ecart == ecartMin && c.TimeMs < meilleur.TimeMs) {
			meilleur, ecartMin = c, ecart
		}
	}
	return meilleur
}

// PlacementDeLaMort rend le placement d'une mort, ou nil quand il ne se dit pas :
//
//	sans contexte                         nil (rien n'a été mesuré) ;
//	aucun coéquipier visible               seul, sans distance (vrai quelle que soit la portée) ;
//	distance mesurée, portée inconnue      nil (la règle du match n'est pas établie) ;
//	distance à la portée ou en deçà        près, avec la distance ;
//	distance au-delà                       seul, avec la distance.
func PlacementDeLaMort(c *domain.ContexteDeMort, rayon float64, aUnRayon bool) *domain.TacticalPlacement {
	if c == nil {
		return nil
	}
	if c.PlusProcheM == nil {
		return &domain.TacticalPlacement{Seul: true}
	}
	if !aUnRayon {
		return nil
	}
	d := *c.PlusProcheM
	return &domain.TacticalPlacement{Seul: !coordination.APortee(&d, rayon), DistanceM: &d}
}
