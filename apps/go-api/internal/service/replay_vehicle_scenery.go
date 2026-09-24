package service

// replay_vehicle_scenery.go — LES VEHICULES DE DECOR D UNE CARTE, decides A LA REQUETE (retours du
// rejeu, lot M7, 2026-09-24). La regle et sa preuve vivent a cote
// (replay_vehicle_scenery_rule.go) ; ce fichier ne fait que CHARGER la zone
// jouable de la carte du match.
//
// A LA REQUETE ET NON A LA CUISSON : la zone jouable est une REFERENCE DE CARTE du titre (le fond
// publie et son calage), que l artefact ne connait pas — le film ne nomme pas sa carte. Meme regle
// et meme raison que `mapObjectives` : une reference qui s ameliore (un fond cuit pour une carte
// qui n en avait pas) profite aux artefacts deja publies sans recuisson.
//
// LA CLE DE FOND EST CELLE DU FOND SERVI (`resolveBackgroundKeyDepuis`, la seule cascade) : la
// zone jouable et l image affichee sous le rejeu sont, par construction, la meme carte.
//
// COUT BORNE : le masque n est charge que si le document porte une vie candidate (pose seule) —
// 3 documents sur 107 au parc du 2026-09-24. Chargement et fermeture mesures de 40 ms (Starboard)
// a 290 ms (Behemoth, 2 451 x 2 452 cellules), aucun cache : rien ne reste en memoire.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapdecoupe"
	"levelup/go-api/internal/port"
)

// resolveVehicleScenery pose `doc.VehicleScenery` : absent quand aucune vie n est candidate ; zone
// `unknown` (rien de masque) quand la carte n a pas de zone jouable publiee.
func (s *replayService) resolveVehicleScenery(ctx context.Context, doc *replay.ReplayDocument,
	matchID string, keys port.MatchMapKeys) {
	if !hasPosedOnlyVehicle(doc) {
		return
	}
	zone := s.playAreaFor(ctx, matchID, keys)
	var area playArea
	if zone != nil {
		area = zone
	}
	doc.VehicleScenery = decideVehicleScenery(doc, area)
	v := doc.VehicleScenery
	slog.DebugContext(ctx, "rejeu 2D : decor de carte",
		"match_id", matchID, "zone", v.Zone, "sol", v.Floor, "candidates", v.Candidates,
		"masquees", len(v.Hidden), "dans_la_zone", v.InPlayArea, "zone_inconnue", v.ZoneUnknown,
		"titleSlug", s.titleSlug)
}

// playAreaFor charge la zone jouable EN PLAN de la carte du match : le masque de son fond publie,
// ferme au rayon canonique du depot. nil = carte sans zone connue (repli `unknown`, compte).
func (s *replayService) playAreaFor(ctx context.Context, matchID string,
	keys port.MatchMapKeys) *mapdecoupe.Masque {
	key, err := s.resolveBackgroundKeyDepuis(ctx, keys, nil, "match_id", matchID)
	if err != nil {
		return nil
	}
	image, _, err := s.backgroundImageFile(ctx, key)
	if err != nil {
		return nil
	}
	meta := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, key)
	m, err := mapdecoupe.ChargeMasque(image, meta)
	if err != nil {
		// Un fond PRESENT mais illisible est une donnee abimee, pas le cas nominal : on le dit,
		// puis on degrade — rien n est masque.
		slog.WarnContext(ctx, "rejeu 2D : zone jouable illisible — aucun decor masque",
			"err", err, "cle", key, "match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	return m.Comble(mapdecoupe.ToleranceParDefaut)
}
