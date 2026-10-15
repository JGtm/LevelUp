package replay

import (
	"context"
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

// installWorldObjectPrecision pose sur le contexte du film le CONTEXTE DE CARTE de la cuisson, par le
// geste unique de la grammaire ([grammar.FilmContext.PoserLaCarteEtLeDecoupage]) : les largeurs d'axe
// de la carte du match (un champ du profil du film, résolu une fois par `BuildFromFilm`) sur le profil
// de balayage du contexte, puis le découpage du bloc MPP que la grammaire résout pour le film. Les
// balayages de la cuisson construisent leurs lecteurs par ce contexte : rien à restaurer, le contexte
// meurt avec la cuisson.
//
// Largeurs absentes de l'entrée (catalogue antérieur au champ, entrée fabriquée à la main) : le défaut
// est CONSERVÉ, compté et journalisé ; un découpage MPP qui ne décide pas est journalisé
// ([signalerLeDecoupageMPP]). Jamais de dégradation silencieuse. Le journal porte le `ctx` de
// l'appelant de la cuisson.
func installWorldObjectPrecision(ctx context.Context, fc *grammar.FilmContext, matchID string, fb *fallback.Compteur) {
	pose := fc.PoserLaCarteEtLeDecoupage()
	if pose.LargeursAbsentes {
		// REPLI NOMME ET COMPTE (D14) : le defaut conserve est celui d'UNE carte, applique a
		// toutes. Le journal le dit ; le compte le fait voyager avec l'artefact.
		fb.Declenche(fallback.NomLargeursAxeParDefautConservees)
		slog.WarnContext(ctx, "largeurs d'axe absentes de l'entrée de catalogue — objets du monde déquantifiés aux largeurs par défaut",
			"module", fc.Profile().Map().Module, "match_id", matchID,
			"defaut", fc.ProfilDeBalayage().LargeursObjetDuMonde().AxisW)
	}
	signalerLeDecoupageMPP(ctx, matchID, pose.MPP)
}
