package replay

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
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

// doubleEcritureGlobales — KILL-SWITCH DATE DE LA DOUBLE ECRITURE (D7 du PLAN_DECODEUR_FILM,
// regle 11 du depot).
//
//	BASCULE DU DEFAUT : 2026-09-17 (lot 2.1). Ce site LIT desormais le profil du film
//	                    ([filmdec.Profile]) et ECRIT ENCORE la globale de paquet, parce
//	                    qu aucun lecteur de bits ne lit le profil avant le lot 2.2.
//	RETRAIT CIBLE     : lot 2.3 (« plus de globale, plus de verrou »). Ce jour-la cette
//	                    constante et la branche qu elle garde disparaissent avec
//	                    `filmdec.WorldObjectPrecision`.
//	CRITERE MESURABLE : ZERO variable de paquet mutable dans `filmdec`, mesure par
//	                    `archlint/TestFilmdecPackageVarsNeCroitPas` (`filmdecVarsGeles` a 0).
//
// CE QU ELLE N EST PAS : une bascule A/B. La mettre a `false` aujourd hui ne « choisit » rien —
// elle laisserait les lecteurs de bits sur le defaut de paquet, donc sur les largeurs d une
// carte qui n est pas celle du match. Elle existe pour que le retrait du lot 2.3 soit un geste
// NOMME et DATE, et pour que `TestProfilEgaleGlobalesWorldObject` sache ce qu il doit exiger.
const doubleEcritureGlobales = true

// installWorldObjectPrecision installe, pour la durée du décodage, les largeurs d'axe de la
// CARTE DU MATCH sur le chemin world-object, et rend la fonction de restauration.
//
// IL LIT LE PROFIL DU FILM DEPUIS LE LOT 2.1 (item 2.1.3) : la carte du match n arrive plus par
// un parametre a part, elle est un CHAMP du profil resolu une fois par `BuildFromFilm`. Ce qu il
// ECRIT n a pas change — la globale de paquet — et c est la double ecriture que
// [doubleEcritureGlobales] date.
//
// PRÉ-REQUIS : l'appelant détient `filmdec.LockProcessDecode` — les largeurs vivent dans le
// PROFIL HÉRITÉ du processus (lot 2.2.b : ce n'est plus une globale de paquet, mais c'est
// toujours un état de processus), et deux films décodés en parallèle se voleraient les leurs.
//
// POURQUOI L'HÉRITAGE ET PAS LE PROFIL DU FILM (lot 2.2.b). Les largeurs doivent atteindre les
// quarante balayages de `BuildFromFilm`, dont chacun construit ses propres lecteurs de bits
// sans recevoir de profil. Tant que le profil ne descend pas jusqu'à eux — lot 2.5 —, le seul
// canal est celui que cet installateur pose et restaure autour de la cuisson.
//
// Largeurs absentes de l'entrée (catalogue antérieur au champ, entrée fabriquée à la main) :
// le défaut est CONSERVÉ et l'écart est LOGGÉ. Jamais de dégradation silencieuse.
//
// `slog.Warn` et non `WarnContext` : `BuildFromFilm` — le seul appelant — ne prend pas de
// `ctx`, et tout le fichier `build.go` journalise ainsi.
func installWorldObjectPrecision(prof filmdec.Profile, matchID string, fb *fallback.Compteur) (restore func()) {
	e := prof.Map()
	if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {
		// REPLI NOMME ET COMPTE (D14) : le defaut conserve est celui d'UNE carte, applique a
		// toutes. Le journal le disait deja ; le compte le fait voyager avec l'artefact.
		fb.Declenche(fallback.NomLargeursAxeParDefautConservees)
		slog.Warn("largeurs d'axe absentes de l'entrée de catalogue — objets du monde déquantifiés aux largeurs par défaut",
			"module", e.Module, "match_id", matchID,
			"defaut", filmdec.WorldObjectPrecisionActuelle().AxisW)
		return func() {}
	}
	if !doubleEcritureGlobales {
		// Lot 2.3 : les lecteurs prennent le profil, il n y a plus rien a installer.
		return func() {}
	}
	prev := filmdec.WorldObjectPrecisionActuelle()
	// e.Layout() porte les largeurs d'axe ET la largeur de l'index de région (2 bits sur
	// Live Fire — lot C catalogues, 2026-08-27) : les deux sont des constantes par carte.
	filmdec.SetWorldObjectPrecisionFromLayout(e.Layout())
	return func() { filmdec.PoserWorldObjectPrecision(prev) }
}
