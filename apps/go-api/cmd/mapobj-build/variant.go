// cmd/mapobj-build — variant.go : choix de la variante `.mvar` d'une carte.
//
// POURQUOI CE FICHIER EXISTE. Un asset de carte expose plusieurs `.mvar`. Le choix se
// faisait au nombre d'objectifs, et ce critère se retourne contre lui-même sur les cartes
// bâties dans Forge : le canevas de base livré avec le jeu contient le RACK des objets de
// mode — un exemplaire de chaque, rangé hors du terrain — pendant que la carte réellement
// jouée vit dans l'autre fichier.
//
// Mesuré le 2026-08-08 sur Vagabond (asset 105f5d84) :
//
//	fo08_wetland.mvar :  100 objets,  20 objectifs, tous à z = 50,50 exactement,
//	                     étalés sur 8,2 m alors que ses objets couvrent 356 m  ->  2,3 %
//	map.mvar          : 4709 objets,   4 objectifs (3 zones de Bastion + 1 crâne),
//	                     étalés sur 22,3 m                                     -> 12,7 %
//
// L'ancien critère retenait le rack : trois « zones de Bastion » de 1 m de rayon, à 2 m
// l'une de l'autre, au même millimètre d'altitude. Publier ça, c'est publier un objectif
// au mauvais endroit — ce que l'en-tête de fetch.go dit vouloir éviter.
package main

import "levelup/go-api/internal/analysis/replay/mapvar"

// parkedSpreadRatio : sous cette fraction de l'emprise des objets de la variante,
// l'emprise des objectifs n'est plus un placement mais un rangement.
//
// Calibration sur les 37 cartes du catalogue (2026-08-08) : le rack de Vagabond est à
// 2,3 %, la carte la plus basse ensuite est `corpo_map` à 15,8 %, puis 44,4 %. Le seuil
// est posé entre les deux, avec un facteur ~2 de marge de part et d'autre.
const parkedSpreadRatio = 0.05

// minObjectivesForSpread : à un seul objectif l'emprise vaut 0 et ne dit rien.
const minObjectivesForSpread = 2

// isParkedPalette dit si les objectifs d'une variante sont RANGÉS (rack du canevas) au
// lieu d'être POSÉS sur le terrain.
func isParkedPalette(v *mapvar.Variant) bool {
	objectives := v.Objectives()
	if len(objectives) < minObjectivesForSpread || len(v.Objects) == 0 {
		return false
	}
	objMinX, objMinY, objMaxX, objMaxY := inf, inf, -inf, -inf
	for _, o := range v.Objects {
		objMinX, objMaxX = min(objMinX, o.Pos.X), max(objMaxX, o.Pos.X)
		objMinY, objMaxY = min(objMinY, o.Pos.Y), max(objMaxY, o.Pos.Y)
	}
	objSpread := max(objMaxX-objMinX, objMaxY-objMinY)
	if objSpread <= 0 {
		return false
	}
	gMinX, gMinY, gMaxX, gMaxY := inf, inf, -inf, -inf
	for _, o := range objectives {
		gMinX, gMaxX = min(gMinX, o.Pos.X), max(gMaxX, o.Pos.X)
		gMinY, gMaxY = min(gMinY, o.Pos.Y), max(gMaxY, o.Pos.Y)
	}
	goalSpread := max(gMaxX-gMinX, gMaxY-gMinY)
	return goalSpread < parkedSpreadRatio*objSpread
}

// inf sert de borne initiale aux min/max (pas de math.Inf pour rester en float64 littéral
// lisible dans les deux sens).
const inf = 1e18
