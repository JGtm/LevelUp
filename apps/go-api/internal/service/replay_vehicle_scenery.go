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
// COUT BORNE, MESURE (revue RR-M7-01 du 2026-09-24). La premiere version fermait le masque ENTIER a
// chaque requete : 713 Mo alloues pour Behemoth, 194 Mo pour Goliath, 163 Mo pour Starboard, a
// chaque ouverture. Trois gestes, dans l ordre :
//   - rien n est lu quand le document ne porte aucune candidate (104 documents sur 107 au parc) ;
//   - le CADRE decide seul, sur le calage du sidecar (1 a 2 Kio), quand aucune candidate ne tombe
//     dans l image — c est le cas des decors de Starboard : l image n est pas decodee ;
//   - sinon le masque est decode UNE FOIS par processus et par fond, garde sous sa forme COMPACTE
//     (un bit par cellule, 734 Kio au plus : Behemoth), et la fermeture se calcule EN UN POINT
//     (`mapdecoupe.MasqueCompact.PraticableComble`, fenetre locale, egale a la fermeture globale).
//     Le cache est borne a [sceneryMaskCacheMax] fonds et invalide par la signature des deux
//     fichiers (taille et date), comme l index des fonds (`replay.MapBackgroundIndexFor`) ; le
//     decodage se fait sous le verrou, donc jamais deux a la fois.
//
// Mesures du correctif sont dans le test `TestVehicleScenery_CoutMemoire` et au compte rendu du lot.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapdecoupe"
	"levelup/go-api/internal/port"
)

// sceneryMaskCacheMax : combien de masques compacts le processus garde. 8 x 734 Kio au plus
// (5,9 Mo) ; au parc, trois cartes seulement portent une candidate (Starboard, Goliath, et
// Behemoth quand un Mongoose n est pas touche).
const sceneryMaskCacheMax = 8

var (
	sceneryMasksMu    sync.Mutex
	sceneryMasksCache = map[string]sceneryMaskEntry{}
	sceneryMasksOrdre []string // ordre d insertion : le plus ancien sort le premier
)

type sceneryMaskEntry struct {
	signature string
	mask      *mapdecoupe.MasqueCompact
}

// mapPlayArea est la zone jouable EN PLAN d une carte : le cadre de son fond (le calage), et son
// masque compact quand une candidate tombe dans le cadre.
type mapPlayArea struct {
	cal  replay.MapBackgroundCalibration
	mask *mapdecoupe.MasqueCompact
}

// Praticable : hors du cadre, jamais ; dans le cadre, la matiere fermee au rayon canonique. Un
// masque absent n est jamais interroge dans le cadre ([playAreaFor] le charge des qu une candidate
// y tombe) ; s il l etait, la reponse serait la plus prudente — praticable, donc affiche.
func (a mapPlayArea) Praticable(x, y float64) bool {
	if _, _, ok := a.cal.MondeVersPixel(x, y); !ok {
		return false
	}
	if a.mask == nil {
		return true
	}
	return a.mask.PraticableComble(x, y, mapdecoupe.ToleranceParDefaut)
}

// resolveVehicleScenery pose `doc.VehicleScenery` : absent quand aucune vie n est candidate ; zone
// `unknown` (rien de masque) quand la carte n a pas de zone jouable publiee.
func (s *replayService) resolveVehicleScenery(ctx context.Context, doc *replay.ReplayDocument,
	matchID string, keys port.MatchMapKeys) {
	candidates := posedOnlyVehicles(doc)
	if len(candidates) == 0 {
		return
	}
	var area playArea
	if zone, ok := s.playAreaFor(ctx, matchID, keys, candidates); ok {
		area = zone
	}
	doc.VehicleScenery = decideVehicleScenery(doc, area)
	v := doc.VehicleScenery
	slog.DebugContext(ctx, "rejeu 2D : decor de carte",
		"match_id", matchID, "zone", v.Zone, "sol", v.Floor, "candidates", v.Candidates,
		"masquees", len(v.Hidden), "dans_la_zone", v.InPlayArea, "zone_inconnue", v.ZoneUnknown,
		"titleSlug", s.titleSlug)
}

// playAreaFor charge la zone jouable EN PLAN de la carte du match. Faux = carte sans zone connue
// (repli `repli_decor_carte_sans_zone_affiche`, compte dans `zoneUnknown`).
func (s *replayService) playAreaFor(ctx context.Context, matchID string, keys port.MatchMapKeys,
	candidates []replay.VehicleTrack) (mapPlayArea, bool) {
	key, err := s.resolveBackgroundKeyDepuis(ctx, keys, nil, "match_id", matchID)
	if err != nil {
		return mapPlayArea{}, false
	}
	bg, err := s.loadMapBackground(ctx, key)
	if err != nil {
		return mapPlayArea{}, false
	}
	image, _, err := s.imageFileOf(ctx, key, bg)
	if err != nil {
		return mapPlayArea{}, false
	}
	if _, err := os.Stat(image); err != nil {
		slog.DebugContext(ctx, "rejeu 2D : fond sans image — aucun decor masque",
			"err", err, "cle", key, "match_id", matchID, "titleSlug", s.titleSlug)
		return mapPlayArea{}, false
	}
	area := mapPlayArea{cal: bg.Calibration}
	if !anyInFrame(candidates, bg.Calibration) {
		return area, true // le cadre decide seul : l image n est pas decodee
	}
	meta := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, key)
	mask, err := sceneryMaskFor(image, meta)
	if err != nil {
		// Un fond PRESENT mais illisible est une donnee abimee, pas le cas nominal : on le dit,
		// puis on degrade — rien n est masque.
		slog.WarnContext(ctx, "rejeu 2D : zone jouable illisible — aucun decor masque",
			"err", err, "cle", key, "match_id", matchID, "titleSlug", s.titleSlug)
		return mapPlayArea{}, false
	}
	area.mask = mask
	return area, true
}

// anyInFrame dit si une candidate tombe dans le cadre du fond.
func anyInFrame(candidates []replay.VehicleTrack, cal replay.MapBackgroundCalibration) bool {
	for _, v := range candidates {
		if _, _, ok := cal.MondeVersPixel(float64(v.Samples[0].X), float64(v.Samples[0].Y)); ok {
			return true
		}
	}
	return false
}

// sceneryMaskFor rend le masque compact d un fond, decode une seule fois par signature.
func sceneryMaskFor(image, meta string) (*mapdecoupe.MasqueCompact, error) {
	sig, err := signatureFichiers(image, meta)
	if err != nil {
		return nil, err
	}
	sceneryMasksMu.Lock()
	defer sceneryMasksMu.Unlock()
	if e, ok := sceneryMasksCache[image]; ok && e.signature == sig {
		return e.mask, nil
	}
	m, err := mapdecoupe.ChargeMasque(image, meta)
	if err != nil {
		return nil, err
	}
	mask := m.Compacte()
	if _, deja := sceneryMasksCache[image]; !deja {
		sceneryMasksOrdre = append(sceneryMasksOrdre, image)
	}
	sceneryMasksCache[image] = sceneryMaskEntry{signature: sig, mask: mask}
	for len(sceneryMasksOrdre) > sceneryMaskCacheMax {
		delete(sceneryMasksCache, sceneryMasksOrdre[0])
		sceneryMasksOrdre = sceneryMasksOrdre[1:]
	}
	return mask, nil
}

// signatureFichiers resume l etat des fichiers dont depend un masque : taille et date de chacun.
func signatureFichiers(paths ...string) (string, error) {
	sig := ""
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		sig += fmt.Sprintf("%s:%d:%d;", p, info.Size(), info.ModTime().UnixNano())
	}
	return sig, nil
}
