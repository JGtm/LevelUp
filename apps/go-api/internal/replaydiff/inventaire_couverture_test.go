package replaydiff

// inventaire_couverture_test.go — L'INVENTAIRE DES MESURES DE COUVERTURE, TIRE DE LA FORME DU
// DOCUMENT ET NON D'UNE LISTE RECOPIEE.
//
// La source est `film/replay/testdata/document_shape.golden` : l'empreinte de forme du document
// STOCKE (noms, balises JSON, types, a toute profondeur), regeneree par le test du producteur a
// chaque changement de forme — et refusee si `SchemaVersion` ne suit pas. La lire ici, sans
// importer `film/replay` (la surface de la facade du decodeur est comptee par
// `archlint/film_facade_surface_test.go`), donne l'ensemble EXACT des feuilles que `aplatir`
// posera sous `coverage.*` et `bombStats.coverage.*` : un champ neuf dans la couverture entre
// dans cet inventaire au commit qui l'ajoute, et le ratchet (`polarite_ratchet_test.go`) exige
// alors sa polarite.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cheminFormeDocument : le golden de forme du producteur, relatif a ce paquet.
var cheminFormeDocument = filepath.Join("..", "games", "halo_infinite", "film", "replay",
	"testdata", "document_shape.golden")

// champForme est UNE ligne de champ du golden : `  Nom json:"tag,omitempty" Type`.
type champForme struct {
	nom, tag, typ string
}

// lireForme lit le golden en une table type -> champs.
func lireForme(path string) (map[string][]champForme, error) {
	f, err := os.Open(path) //nolint:gosec // chemin de testdata du depot
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	types := map[string][]champForme{}
	courant := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ligne := sc.Text()
		switch {
		case strings.HasPrefix(ligne, "#") || strings.TrimSpace(ligne) == "":
			continue
		case strings.HasPrefix(ligne, "  "):
			if courant == "" {
				continue
			}
			c, ok := lireChamp(strings.TrimSpace(ligne))
			if !ok {
				return nil, fmt.Errorf("ligne de champ illisible : %q", ligne)
			}
			types[courant] = append(types[courant], c)
		case !strings.Contains(ligne, " "):
			courant = ligne
			types[courant] = nil
		default:
			courant = "" // lignes d'en-tete (`schema 76`, `empreinte-...`)
		}
	}
	return types, sc.Err()
}

// lireChamp decoupe `Nom json:"tag,opts" Type`.
func lireChamp(s string) (champForme, bool) {
	i := strings.Index(s, ` json:"`)
	if i < 0 {
		return champForme{}, false
	}
	reste := s[i+len(` json:"`):]
	j := strings.Index(reste, `" `)
	if j < 0 {
		return champForme{}, false
	}
	tag := reste[:j]
	if k := strings.IndexByte(tag, ','); k >= 0 {
		tag = tag[:k]
	}
	return champForme{nom: s[:i], tag: tag, typ: strings.TrimSpace(reste[j+2:])}, true
}

// estScalaireNumerique : les types Go que `encoding/json` ecrit en nombre.
func estScalaireNumerique(t string) bool {
	for _, p := range []string{"int", "uint", "float"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

// inventorierNumeriques rend les chemins NUMERIQUES (au sens de `aplatir`) poses sous `racine`
// pour le type `typ` : feuilles numeriques, longueurs de tableaux (`/n`), cles de map en `*`.
func inventorierNumeriques(types map[string][]champForme, racine, typ string) []string {
	var out []string
	var marcher func(prefixe, t string, profondeur int)
	marcher = func(prefixe, t string, profondeur int) {
		t = strings.TrimPrefix(t, "*")
		switch {
		case profondeur > 8:
			return
		case strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "["):
			out = append(out, prefixe+"/n")
		case strings.HasPrefix(t, "map[string]"):
			marcher(prefixe+"."+jokerCle, strings.TrimPrefix(t, "map[string]"), profondeur+1)
		case estScalaireNumerique(t):
			out = append(out, prefixe)
		default:
			champs, connu := types[t]
			if !connu {
				return // string, bool, ou type sans champ publie
			}
			for _, c := range champs {
				if c.tag == "-" {
					continue
				}
				if c.tag == "" && c.nom == strings.TrimPrefix(c.typ, "*") {
					marcher(prefixe, c.typ, profondeur) // champ embarque : promu tel quel
					continue
				}
				nom := c.tag
				if nom == "" {
					nom = c.nom
				}
				marcher(prefixe+"."+nom, c.typ, profondeur+1)
			}
		}
	}
	marcher(racine, typ, 0)
	sort.Strings(out)
	return out
}

// inventaireCouverture : toutes les mesures numeriques de couverture que le document stocke
// peut porter, `coverage.*` et `bombStats.coverage.*`.
func inventaireCouverture(path string) ([]string, error) {
	types, err := lireForme(path)
	if err != nil {
		return nil, err
	}
	if _, ok := types["Coverage"]; !ok {
		return nil, fmt.Errorf("%s : type Coverage absent", path)
	}
	if _, ok := types["BombStatsCoverage"]; !ok {
		return nil, fmt.Errorf("%s : type BombStatsCoverage absent", path)
	}
	out := inventorierNumeriques(types, "coverage", "Coverage")
	return append(out, inventorierNumeriques(types, "bombStats.coverage", "BombStatsCoverage")...), nil
}
