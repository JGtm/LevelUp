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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/mapcatalog"
)

// dumpIndex est le dépôt local de `.mvar`, indexé par nom de fichier.
type dumpIndex struct {
	dir   string
	files map[string]bool
	// ambigus : les `mvar_file` que PLUSIEURS cartes du catalogue d'objectifs partagent.
	//
	// CINQUANTE-HUIT CARTES DECLARENT `map.mvar`. Un fichier de ce nom depose a plat dans le
	// depot serait donc reclame par les 58, et chacune recevrait les socles d'une carte
	// etrangere — sans le moindre signe exterieur. Le cas s'est produit pendant ce chantier :
	// 65 cartes sur 72 sont sorties avec des socles qui n'etaient pas les leurs, et la
	// signature etait un nombre de socles uniforme sur des cartes sans rapport.
	//
	// L'AMBIGUITE SE CALCULE, elle ne se devine pas : c'est le catalogue qui dit quels noms
	// sont partages, pas une liste de noms suspects ecrite a la main.
	ambigus map[string]bool
}

// newDumpIndex liste les `.mvar` du dossier. Un dossier vide est une erreur : il n'y a
// alors rien à produire, et écrire un catalogue vide effacerait le précédent.
func newDumpIndex(dir string, objectifs *replay.MapObjectivesCatalog) (*dumpIndex, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	idx := &dumpIndex{dir: dir, files: map[string]bool{}, ambigus: map[string]bool{}}
	if objectifs != nil {
		vus := map[string]int{}
		for _, e := range objectifs.Maps {
			if e.MvarFile != "" {
				vus[e.MvarFile]++
			}
		}
		for nom, n := range vus {
			if n > 1 {
				idx.ambigus[nom] = true
			}
		}
	}
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
		// 1. Le nom enregistré EST celui du dump (ingestion `--from-file`) — SEULEMENT si ce
		//    nom n'est pas partagé par plusieurs cartes. Sur un nom ambigu (`map.mvar` et ses
		//    57 homonymes), la règle est REFUSÉE : mieux vaut une carte absente du catalogue
		//    qu'une carte dotée des socles d'une autre.
		nomSiNonAmbigu(d, e.MvarFile),
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

// nomSiNonAmbigu rend le nom brut, ou la chaîne vide si plusieurs cartes le partagent (la
// chaîne vide se disqualifie d'elle-même dans la boucle des candidats).
func nomSiNonAmbigu(d *dumpIndex, mvarFile string) string {
	if d.ambigus[mvarFile] {
		return ""
	}
	return mvarFile
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
func ingest(mapID string, e replay.MapObjectivesEntry, path, base string,
) (replay.MapWeaponPadsEntry, int, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return replay.MapWeaponPadsEntry{}, 0, err
	}
	out, mixedPads, mixedPts, err := mapcatalog.EntryFromMvar(mapID, e, buf, base)
	if err != nil {
		return replay.MapWeaponPadsEntry{}, 0, err
	}
	// LES FUSIONS HETEROGENES SE DISENT. Elles doivent rester a zero : un point de grenade
	// absorbe dans un point d'equipement publierait une nature fausse, et le regroupement jette
	// le type des absorbes. Un compte non nul est un signal, pas un detail de journal.
	if mixedPts > 0 {
		slog.Warn("mapopads: points d'apparition a natures MELANGEES — la nature publiee est "+
			"celle du representant, celles des objets absorbes sont perdues",
			"carte", base, "map_id", mapID, "points_melanges", mixedPts)
	}
	return out, mixedPads, nil
}
