package replay

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// world_object_precision.go — LES LARGEURS D'AXE DU CHEMIN WORLD-OBJECT, POSÉES DEPUIS LA
// CARTE DU MATCH.
//
// LE DÉFAUT DE PAQUET EST CELUI DE CLIFFHANGER, et ce n'est pas un repli neutre :
// `grammar.WorldObjectPrecision = {13,13,14}` est EXACTEMENT l'entrée `cliffhanger` du
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
// découpage lu dans le bitstream (`grammar.DetectI0Layout`) est le CONTRÔLE que le commentaire
// d'`AxisWidths` réclame — accord 7 films sur 7 le 2026-08-15 — jamais l'entrée : s'il
// contredisait le catalogue, ce seraient les BORNES qui seraient fausses.

// installWorldObjectPrecision installe, pour la durée du décodage, les largeurs d'axe de la
// CARTE DU MATCH sur le chemin world-object — SUR LE PROFIL DE BALAYAGE DU CONTEXTE DU FILM.
//
// IL LIT LE PROFIL DU FILM DEPUIS LE LOT 2.1 (item 2.1.3) : la carte du match n'arrive plus par
// un paramètre à part, elle est un CHAMP du profil résolu une fois par `BuildFromFilm`.
//
// IL ÉCRIT SUR LE CONTEXTE DEPUIS LE LOT 2.3, plus dans le processus. La double écriture datée
// (`doubleEcritureGlobales`, posée au lot 2.1 avec « retrait cible : lot 2.3 ») est RETIRÉE avec
// la variable de paquet qu'elle alimentait : les quarante balayages de `BuildFromFilm`
// construisent leurs lecteurs par le contexte (`FilmContext.NouveauLecteur`, ou en recevant son
// profil), donc le canal existe sans état de processus — et deux films peuvent se décoder en
// parallèle. Aucune restauration n'est nécessaire : le contexte meurt avec la cuisson.
//
// Largeurs absentes de l'entrée (catalogue antérieur au champ, entrée fabriquée à la main) :
// le défaut est CONSERVÉ et l'écart est LOGGÉ. Jamais de dégradation silencieuse.
//
// `slog.Warn` et non `WarnContext` : `BuildFromFilm` — le seul appelant — ne prend pas de
// `ctx`, et tout le fichier `build.go` journalise ainsi.
func installWorldObjectPrecision(fc *grammar.FilmContext, matchID string, fb *fallback.Compteur) {
	e := fc.Profile().Map()
	if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {
		// REPLI NOMME ET COMPTE (D14) : le defaut conserve est celui d'UNE carte, applique a
		// toutes. Le journal le disait deja ; le compte le fait voyager avec l'artefact.
		fb.Declenche(fallback.NomLargeursAxeParDefautConservees)
		slog.Warn("largeurs d'axe absentes de l'entrée de catalogue — objets du monde déquantifiés aux largeurs par défaut",
			"module", e.Module, "match_id", matchID,
			"defaut", fc.ProfilDeBalayage().LargeursObjetDuMonde().AxisW)
		return
	}
	// e.Layout() porte les largeurs d'axe ET la largeur de l'index de région (2 bits sur
	// Live Fire — lot C catalogues, 2026-08-27) : les deux sont des constantes par carte.
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(e.Layout())
	fc.PoserProfilDeBalayage(bal)
}
