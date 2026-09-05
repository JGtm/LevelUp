package replay

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// world_object_precision.go — LES LARGEURS D'AXE DU CHEMIN WORLD-OBJECT, POSÉES DEPUIS LA
// CARTE DU MATCH.
//
// LE DÉFAUT DE PAQUET EST CELUI DE CLIFFHANGER, et ce n'est pas un repli neutre :
// `filmdec.WorldObjectPrecision = {13,13,14}` est EXACTEMENT l'entrée `cliffhanger` du
// catalogue `map_quant_bounds.json` (module `ridgeline`). Jusqu'au 2026-08-15, aucun chemin de
// production ne l'écrasait — toutes les autres cartes déquantifiaient leurs objets du monde
// (projectiles ti=41, équipement ti=37, armes au sol ti=42, corps rigides ti=38) aux largeurs
// d'une carte qui n'était pas la leur, et rien ne le signalait.
//
// CE QUE ÇA COÛTAIT, MESURÉ AVANT DE CORRIGER (7 films, 7 cartes, critère : part
// d'échantillons tombant dans l'emprise du nuage des BIPÈDES du même film, en coordonnées
// normalisées de l'AABB — sans base, non circulaire) :
//
//	Bazaar     [17 17 16]    0,09 %  ->  99,41 %     (12 694 échantillons ti=41)
//	Illusion   [18 18 17]    0,51 %  ->  99,61 %     (16 956)
//	Catalyst   [15 15 15]   28,46 %  ->  99,60 %     (14 124)
//	Oasis      [15 15 14]   31,31 %  ->  98,96 %     ( 9 128)
//	Smallhalla [15 15 17]   65,21 %  ->  99,79 %     (11 725)
//	Cliffhanger[13 13 14]   92,11 %  ->  92,11 %     (13 544 — TÉMOIN, +0,00 point)
//
// POURQUOI LE CATALOGUE ET NON `DetectI0Layout`. Les largeurs sont `AxisWidths`, déduit des
// bornes par la loi du moteur, porté par la MÊME entrée qui fournit déjà les bornes. Le
// découpage lu dans le bitstream (`filmdec.DetectI0Layout`) est le CONTRÔLE que le commentaire
// d'`AxisWidths` réclame — accord 7 films sur 7 le 2026-08-15 — jamais l'entrée : s'il
// contredisait le catalogue, ce seraient les BORNES qui seraient fausses.

// installWorldObjectPrecision installe, pour la durée du décodage, les largeurs d'axe de la
// CARTE DU MATCH sur le chemin world-object, et rend la fonction de restauration.
//
// PRÉ-REQUIS : l'appelant détient `filmdec.LockProcessDecode` — le descripteur est un global
// de paquet, et deux films décodés en parallèle se voleraient leurs largeurs.
//
// Largeurs absentes de l'entrée (catalogue antérieur au champ, entrée fabriquée à la main) :
// le défaut est CONSERVÉ et l'écart est LOGGÉ. Jamais de dégradation silencieuse.
//
// `slog.Warn` et non `WarnContext` : `BuildFromFilm` — le seul appelant — ne prend pas de
// `ctx`, et tout le fichier `build.go` journalise ainsi.
func installWorldObjectPrecision(e filmdec.MapQuantEntry, matchID string) (restore func()) {
	if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {
		slog.Warn("largeurs d'axe absentes de l'entrée de catalogue — objets du monde déquantifiés aux largeurs par défaut",
			"module", e.Module, "match_id", matchID,
			"defaut", filmdec.WorldObjectPrecision.AxisW)
		return func() {}
	}
	prev := filmdec.WorldObjectPrecision
	// e.Layout() porte les largeurs d'axe ET la largeur de l'index de région (2 bits sur
	// Live Fire — lot C catalogues, 2026-08-27) : les deux sont des constantes par carte.
	filmdec.SetWorldObjectPrecisionFromLayout(e.Layout())
	return func() { filmdec.WorldObjectPrecision = prev }
}
