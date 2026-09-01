package mapcatalog

// entry.go — D'UN `.mvar` A UNE ENTREE DE CATALOGUE, et le verrou qui dit si une entree
// existante decrit encore le meme fichier.
//
// EXTRAIT du `package main` de `cmd/mapopads-build` : le rattrapage au fetch de films a besoin
// des memes gestes A L'EXECUTION. Sans extraction il aurait fallu une troisieme copie.

import (
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// EntryFromMvar tire d'un `.mvar` l'entree de catalogue d'une carte : ses socles d'ARME et ses
// POINTS D'APPARITION d'objet ramassable.
//
// Le second retour est le nombre d'emplacements de socle ou des objets de FAMILLES differentes
// ont fusionne, le troisieme celui des points d'apparition a NATURES melangees. Zero sur les
// cartes mesurees ; ils se journalisent pour que ca reste une mesure et non une garantie.
func EntryFromMvar(mapID string, e replay.MapObjectivesEntry, blob []byte, base string,
) (replay.MapWeaponPadsEntry, int, int, error) {
	v, err := mapvar.Parse(blob)
	if err != nil {
		return replay.MapWeaponPadsEntry{}, 0, 0, fmt.Errorf("parser %s: %w", base, err)
	}
	spots := mapvar.PadSpots(v)
	out := replay.MapWeaponPadsEntry{
		MapID: mapID, PublicName: e.PublicName, Module: e.Module, MvarFile: base,
		LevelID: v.LevelID, ObjectsN: len(v.Objects),
		Pads: make([]replay.MapWeaponPadSpot, 0, len(spots)),
	}
	mixedPads := 0
	for _, s := range spots {
		if s.Mixed {
			mixedPads++
		}
		out.Pads = append(out.Pads, replay.MapWeaponPadSpot{
			Pos:     s.Pos,
			TypeID:  fmt.Sprintf("0x%08X", uint32(s.TypeID)),
			Family:  string(s.Family),
			Objects: s.Objects,
		})
	}
	// LES POINTS D'APPARITION, dans une liste a part — la cle est TOUJOURS ecrite pour une
	// carte ingeree, meme vide : c'est le nil qui signifie « non etablis », et lui seul.
	pts := mapvar.SpawnPoints(v)
	mixedPts := 0
	spawn := make([]replay.MapSpawnPointSpot, 0, len(pts))
	for _, s := range pts {
		if s.Mixed {
			mixedPts++
		}
		spawn = append(spawn, replay.MapSpawnPointSpot{
			Pos:     s.Pos,
			TypeID:  fmt.Sprintf("0x%08X", uint32(s.TypeID)),
			Kind:    string(s.Kind),
			Objects: s.Objects,
		})
	}
	out.SpawnPoints = &spawn
	return out, mixedPads, mixedPts, nil
}

// DriftOf rend la RAISON pour laquelle la source d'une carte ne concorde plus avec son entree
// de catalogue, ou la chaine vide si tout concorde.
//
// LE VERROU EST EN TROIS TERMES ET PAS UN SEUL. La comparaison des socles seule ne verifie RIEN
// sur une carte sans socle : deux listes vides sont egales, donc une carte a zero socle passait
// le verrou sans qu'aucune donnee soit comparee. `objects_n` et `level_id` sont precisement les
// signaux qui ont DETECTE la derive de Deadlock (462 objets au catalogue, 410 au
// telechargement) : ils comparent le FICHIER, pas seulement ce qu'on en a extrait.
func DriftOf(entry, neuf replay.MapWeaponPadsEntry) string {
	switch {
	case entry.ObjectsN != neuf.ObjectsN:
		return "objects_n"
	case entry.LevelID != neuf.LevelID:
		return "level_id"
	case !SamePads(entry.Pads, neuf.Pads):
		return "socles"
	}
	return ""
}

// SamePads compare deux listes de socles par leur serialisation JSON — la MEME comparaison que
// celle du gate de non-regression, pour qu'un ecart invisible a l'oeil ne passe pas.
func SamePads(a, b []replay.MapWeaponPadSpot) bool {
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ja) == string(jb)
}
