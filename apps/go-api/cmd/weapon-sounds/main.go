// Command weapon-sounds — extraction des sons de tir par arme depuis les tags `sbnk`.
//
// POURQUOI CET OUTIL. Les `.pck` du jeu livrent 90 170 `.wem` anonymes : aucune bank
// Wwise sur disque (0 bank sur 1645 `.pck`) et aucun nom de tag dans les modules
// (`stringsSize` = 0 sur les 132 modules). Un pack par arme identifie l'ARME de facon
// certaine, mais rien n'y designe le TIR parmi 80 a 360 sons. La mesure qui debloque :
// les banks ont ete converties en tags `sbnk` (1305 dans `pc/globals/globals-rtx-new`),
// et les IDs `.wem` d'une arme s'y retrouvent.
//
// Plan de rattachement : `.ai/V7.5/PLAN_EXTRACTION_SONS_ARMES.md`.
//
// Mode `probe` (etape 1) : decompresse des `sbnk` et statue sur le format reel de leur
// charge utile — bank Wwise verbatim (`BKHD`/`HIRC`) ou variante maison.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// moduleParDefaut : le module qui porte les `sbnk` (mesure : les 40 IDs `.wem` du fusil
// d'assaut y sont tous localises).
const moduleParDefaut = "pc/globals/globals-rtx-new.module"

// wemTemoins : IDs `.wem` reellement presents dans `sb_010_wea_un_assaultrifle.pck`.
// Ils servent de temoins pour reconnaitre le `sbnk` du fusil d'assaut sans nom de tag.
var wemTemoins = []uint32{14649067, 1002108249, 1004646855, 1009888121, 665681453, 253891388}

func main() {
	deploy := flag.String("deploy", "", "racine `deploy` des archives du jeu (auto-detectee si vide)")
	module := flag.String("module", moduleParDefaut, "module a sonder, relatif a la racine deploy")
	mode := flag.String("mode", "probe", "probe | map")
	pck := flag.String("pck", "", "chemin d'un .pck d'arme (mode map)")
	sortie := flag.String("json", "", "fichier JSON de sortie (mode map, facultatif)")
	// Defaut = tous : l'heuristique « une bank d'arme est petite » est FAUSSE (mesure :
	// la bank du fusil d'assaut fait 1,5 Mo, absente des 60 plus petites).
	limite := flag.Int("limite", 0, "nombre de tags sbnk decompresses par la sonde (0 = tous)")
	wem := flag.String("wem", "", "IDs .wem temoins, separes par des virgules (defaut : fusil d'assaut)")
	flag.Parse()

	racine, err := resoudreDeploy(*deploy)
	if err != nil {
		fmt.Fprintln(os.Stderr, "racine deploy introuvable:", err)
		os.Exit(1)
	}
	chemin := filepath.Join(racine, filepath.FromSlash(*module))

	temoins, err := parserWem(*wem)
	if err != nil {
		fmt.Fprintln(os.Stderr, "option -wem invalide:", err)
		os.Exit(1)
	}

	switch *mode {
	case "probe":
		err = sonder(chemin, temoins, *limite)
	case "map":
		if *pck == "" {
			err = fmt.Errorf("le mode map exige -pck")
			break
		}
		err = cartographier(chemin, *pck, *sortie)
	default:
		err = fmt.Errorf("mode inconnu %q", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "echec:", err)
		os.Exit(1)
	}
}

// resoudreDeploy rend la racine `deploy`, explicite ou auto-detectee.
func resoudreDeploy(explicite string) (string, error) {
	if explicite != "" {
		return explicite, nil
	}
	return himap.DeployRoot()
}

// parserWem lit la liste d'IDs temoins ; vide rend les temoins du fusil d'assaut.
func parserWem(s string) ([]uint32, error) {
	if strings.TrimSpace(s) == "" {
		return wemTemoins, nil
	}
	var out []uint32
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		out = append(out, uint32(v))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun ID exploitable")
	}
	return out, nil
}
