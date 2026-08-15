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
// MODES :
//
//	probe    (etape 1) statue sur le format des `sbnk` (bank Wwise verbatim ?)
//	map      (etape 2) d'un `.pck` d'arme vers ses evenements Wwise et leurs `.wem`
//	noms     (etape 3) dumpe les chunks `STID`, seule source de noms en clair
//	lien     (etape 3) relie les evenements aux tags `snd!` puis aux tags `weap`
//	banks    (etape 3) histogramme des `sbnk` references par les `snd!`
//	sndscan  (etape 3) cherche des identifiants arbitraires dans les `snd!`
//
// ATTENTION MEMOIRE : `himodule.Open` lit le module ENTIER en memoire. Le module qui porte
// les `sbnk` fait 7,24 Go, celui qui porte les `snd!`/`weap` 0,62 Go. Ne jamais charger les
// deux dans le meme processus : les modes s'echangent leurs resultats par le fichier JSON.
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

// moduleTags : le module qui porte les `snd!` et les `weap` (14 228 et 111).
const moduleTags = "any/globals/globals-rtx-new.module"

// wemTemoins : IDs `.wem` reellement presents dans `sb_010_wea_un_assaultrifle.pck`.
// Ils servent de temoins pour reconnaitre le `sbnk` du fusil d'assaut sans nom de tag.
var wemTemoins = []uint32{14649067, 1002108249, 1004646855, 1009888121, 665681453, 253891388}

func main() {
	deploy := flag.String("deploy", "", "racine `deploy` des archives du jeu (auto-detectee si vide)")
	module := flag.String("module", moduleParDefaut, "module a sonder, relatif a la racine deploy")
	mode := flag.String("mode", "probe", "probe | map | noms | lien | banks | sndscan")
	pck := flag.String("pck", "", "chemin d'un .pck d'arme (mode map)")
	sortie := flag.String("json", "", "fichier JSON (sortie du mode map, entree des modes lien/final)")
	sortieTir := flag.String("out", "", "fichier JSON de sortie du mode final")
	// Defaut = tous : l'heuristique « une bank d'arme est petite » est FAUSSE (mesure :
	// la bank du fusil d'assaut fait 1,5 Mo, absente des 60 plus petites).
	limite := flag.Int("limite", 0, "nombre de tags sbnk decompresses par la sonde (0 = tous)")
	wem := flag.String("wem", "", "IDs recherches, separes par des virgules (defaut : fusil d'assaut)")
	sfx := flag.String("sfx", "", "dossier des .pck (deduit de -deploy si vide) ; construit l'index large")
	emb := flag.String("emb", "", "dossier ou ecrire les .wem embarques des banks (mode lot)")
	etroit := flag.Bool("etroit", false, "valider les sons contre le seul pck de l'arme (comportement d'origine)")
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

	// INDEX LARGE. Sans lui, un `sourceID` n'est accepte que s'il appartient au pack de
	// l'arme — ce qui faisait disparaitre les couches partagees entre armes, precisement
	// celles qui manquaient a l'oreille. Les garde-fous STRUCTURELS (listes d'enfants et
	// d'actions validees contre les objets de la bank) restent inchanges.
	if !*etroit {
		dossier := *sfx
		if dossier == "" {
			dossier = dossierSFXParDefaut(racine)
		}
		idx, errIdx := indexTousPcks(dossier)
		if errIdx != nil {
			fmt.Fprintln(os.Stderr, "index large indisponible, validation etroite:", errIdx)
		} else {
			indexLarge = idx
			fmt.Printf("index large : %d identifiants .wem sur tous les packs\n", len(indexLarge))
		}
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
	case "noms":
		err = listerNoms(chemin)
	case "lien":
		if *sortie == "" {
			err = fmt.Errorf("le mode lien exige -json (le rapport produit par -mode map)")
			break
		}
		err = relier(chemin, *sortie)
	case "banks":
		err = histogrammeBanks(chemin, temoins[0])
	case "sndscan":
		err = sonderSnd(chemin, temoins)
	case "qui":
		err = quiRefere(chemin, temoins[0])
	case "deps":
		err = dependancesDe(chemin, temoins[0])
	case "remonter":
		err = remonter(chemin, temoins[0], *limite)
	case "tir":
		err = sonsDeTir(chemin, temoins[0])
	case "cadence":
		err = sonderCadence(chemin, temoins[0])
	case "arbre":
		if *pck == "" {
			err = fmt.Errorf("le mode arbre exige -pck")
			break
		}
		profondeur := *limite
		if profondeur <= 0 {
			profondeur = 4
		}
		sortieCouches = *sortieTir
		err = arborescence(chemin, *pck, profondeur)
	case "embarques":
		if *pck == "" {
			err = fmt.Errorf("le mode embarques exige -pck")
			break
		}
		err = extraireEmbarques(chemin, *pck, *sortieTir)
	case "lot":
		if *pck == "" || *sortie == "" {
			err = fmt.Errorf("le mode lot exige -pck (dossier SFX) et -json (sortie)")
			break
		}
		dossierEmbarques = *emb
		err = cartographierLot(chemin, *pck, *sortie)
	case "lot-tir":
		if *sortie == "" || *sortieTir == "" {
			err = fmt.Errorf("le mode lot-tir exige -json (sortie du mode lot) et -out")
			break
		}
		err = livrerTirLot(chemin, *sortie, *sortieTir)
	case "final":
		if *sortie == "" {
			err = fmt.Errorf("le mode final exige -json (le rapport produit par -mode map)")
			break
		}
		err = livrerTir(chemin, *sortie, temoins[0], *sortieTir)
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

// parserWem lit la liste d'identifiants recherches ; vide rend les temoins du fusil d'assaut.
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
