// Package archlint — killsource_recherche_sans_carte_test.go : LE DECODAGE SANS CARTE RESTE A LA
// RECHERCHE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. » Depuis
// ce jour, `killsource.Decode` refuse un film sans carte (`ErrCarteAbsente`) : le decoder aux
// largeurs d une AUTRE carte desynchronise la marche des morts et fait publier le scan a sa place.
// Une seule porte reste ouverte vers les largeurs par defaut, `Options.RechercheSansCarte`, pour
// les instruments de recherche qui mesurent un film quelconque du cache sans base pour en
// resoudre la carte.
//
// CE RATCHET TIENT LA PORTE FERMEE EN PRODUCTION : aucun fichier `.go` hors tests ne la cite, sauf
// le paquet `killsource` lui-meme (qui la declare et la lit). Un appelant de production qui
// l ouvrirait reintroduirait exactement le repli retire du registre
// (`repli_carte_absente_largeurs_par_defaut`). Allowlist VIDE hors du paquet : elle ne se remplit
// pas.
//
// MUTATION JOUEE : poser `opts.RechercheSansCarte = true` dans `replaybuild/kills.go` le fait
// rougir.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// rechercheSansCarteRE : le champ, cite hors commentaire.
var rechercheSansCarteRE = regexp.MustCompile(`\bRechercheSansCarte\b`)

// paquetKillsource : le seul paquet de production autorise a citer le champ (il le declare).
const paquetKillsource = "internal/games/halo_infinite/film/internal/facts/killsource/"

func TestRechercheSansCarteAbsenteDeLaProduction(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici)))

	var violations []string
	fichiers := 0
	for _, sous := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sous), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, paquetKillsource) && !strings.Contains(rel[len(paquetKillsource):], "/") {
				return nil
			}
			fichiers++
			data, errLecture := os.ReadFile(path) //nolint:gosec // chemin issu du parcours du module
			if errLecture != nil {
				return errLecture
			}
			for i, ligne := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(strings.TrimSpace(ligne), "//") {
					continue
				}
				if rechercheSansCarteRE.MatchString(ligne) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(ligne))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sous, err)
		}
	}
	if fichiers == 0 {
		t.Fatal("aucun fichier de production parcouru : le ratchet ne garde plus rien")
	}
	if len(violations) > 0 {
		t.Errorf("`RechercheSansCarte` cite en PRODUCTION — c est le repli « largeurs d une autre "+
			"carte » que la regle « pas de repli » interdit ; resoudre la carte ou mettre le film de "+
			"cote (`ErrCarteAbsente`) :\n  %s", strings.Join(violations, "\n  "))
	}
}
