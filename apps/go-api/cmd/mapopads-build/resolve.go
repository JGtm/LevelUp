// cmd/mapopads-build — resolve.go : QUEL FICHIER POUR QUELLE CARTE, et ce qu'on en extrait.
//
// LE PROBLÈME, en un exemple. Le dépôt de mesure (`.ai/re_dump/mapvar/`) nomme ses 199
// fichiers `{carte}_{fichier}.mvar` : `catalyst_catalyst.mvar`, `cliffhanger_ridgeline.mvar`,
// `aquarius_-_ranked_ctf_aquarius.mvar`. Le catalogue d'objectifs, lui, enregistre le nom
// que la chaîne de production a vu — TANTÔT le nom relatif de l'asset UGC (`catalyst.mvar`),
// TANTÔT le nom du dump lui-même (`cliffhanger_ridgeline.mvar`), selon que la carte a été
// ingérée par le réseau ou par `--from-file`. Aucune des deux écritures n'est fausse ; il
// faut simplement les deux règles, et une troisième pour les cartes sans nom public.
//
// LES TROIS RÈGLES SONT ESSAYÉES DANS L'ORDRE, ET AUCUNE NE DEVINE : chacune construit un
// nom EXACT et vérifie qu'il existe. Rien n'est cherché par ressemblance, rien n'est résolu
// par suffixe — `map.mvar` est le nom d'une trentaine de cartes, et servir le fichier d'une
// carte à une autre publierait ses socles sous le mauvais map_id, en silence.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// dumpIndex est le dépôt local de `.mvar`, indexé par nom de fichier.
type dumpIndex struct {
	dir   string
	files map[string]bool
}

// newDumpIndex liste les `.mvar` du dossier. Un dossier vide est une erreur : il n'y a
// alors rien à produire, et écrire un catalogue vide effacerait le précédent.
func newDumpIndex(dir string) (*dumpIndex, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	idx := &dumpIndex{dir: dir, files: map[string]bool{}}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".mvar") {
			continue
		}
		idx.files[e.Name()] = true
	}
	if len(idx.files) == 0 {
		return nil, fmt.Errorf("aucun .mvar dans %s", dir)
	}
	return idx, nil
}

func (d *dumpIndex) count() int { return len(d.files) }

// resolve rend le chemin du `.mvar` d'une carte, son nom de base, et s'il a été trouvé.
func (d *dumpIndex) resolve(mapID string, e replay.MapObjectivesEntry) (string, string, bool) {
	if e.MvarFile == "" {
		return "", "", false
	}
	candidats := []string{
		// 1. Le nom enregistré EST celui du dump (ingestion `--from-file`).
		e.MvarFile,
		// 2. Le dump préfixe le nom d'asset par le nom public de la carte, minuscules et
		//    espaces changés en soulignés (`Aquarius - Ranked` + `ctf_aquarius.mvar`).
		prefixe(e.PublicName) + "_" + e.MvarFile,
		// 3. Sans nom public, le dump a pris le map_id comme préfixe.
		mapID + "_" + e.MvarFile,
	}
	for _, c := range candidats {
		if c == "" || !d.files[c] {
			continue
		}
		return filepath.Join(d.dir, c), c, true
	}
	return "", "", false
}

// prefixe met un nom public à la forme du dépôt. Vide quand le nom l'est : la règle 2 se
// disqualifie alors d'elle-même plutôt que de produire `_fichier.mvar`.
func prefixe(publicName string) string {
	if publicName == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(publicName, " ", "_"))
}

// ingest lit un `.mvar` et en tire l'entrée de catalogue d'une carte. Le second retour est
// le nombre d'emplacements où des objets de FAMILLES différentes ont fusionné — zéro sur
// les cartes mesurées, et il se journalise pour que ça reste une mesure.
func ingest(mapID string, e replay.MapObjectivesEntry, path, base string) (replay.MapWeaponPadsEntry, int, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return replay.MapWeaponPadsEntry{}, 0, err
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		return replay.MapWeaponPadsEntry{}, 0, fmt.Errorf("parser %s: %w", base, err)
	}
	spots := mapvar.PadSpots(v)
	out := replay.MapWeaponPadsEntry{
		MapID: mapID, PublicName: e.PublicName, Module: e.Module, MvarFile: base,
		LevelID: v.LevelID, ObjectsN: len(v.Objects),
		Pads: make([]replay.MapWeaponPadSpot, 0, len(spots)),
	}
	mixed := 0
	for _, s := range spots {
		if s.Mixed {
			mixed++
		}
		out.Pads = append(out.Pads, replay.MapWeaponPadSpot{
			Pos:     s.Pos,
			TypeID:  fmt.Sprintf("0x%08X", uint32(s.TypeID)),
			Family:  string(s.Family),
			Objects: s.Objects,
		})
	}
	// LES POINTS D'APPARITION, dans une liste a part — voir MapWeaponPadsEntry.SpawnPoints.
	pts := mapvar.SpawnPoints(v)
	out.SpawnPoints = make([]replay.MapSpawnPointSpot, 0, len(pts))
	for _, s := range pts {
		out.SpawnPoints = append(out.SpawnPoints, replay.MapSpawnPointSpot{
			Pos:     s.Pos,
			TypeID:  fmt.Sprintf("0x%08X", uint32(s.TypeID)),
			Kind:    string(s.Kind),
			Objects: s.Objects,
		})
	}
	return out, mixed, nil
}
