// no_player_index_identity_test.go — UN ORDRE N'EST PAS UNE IDENTITÉ.
//
// CE QUE CE GARDE-RAIL EMPÊCHE DE RÉÉCRIRE, et pourquoi il existe. Le rejeu 2D a comparé
// pendant une journée deux numérotations qui n'avaient aucune raison de coïncider : le `pi`
// du roster — un tri alphabétique ASCII que NOUS imposons — et l'index de joueur porté par
// le film. L'écart entre les deux a même été publié comme une découverte sur le format. Ce
// n'en était pas une : il disait seulement qu'on avait comparé notre tri à leur numérotation.
//
// LA RÈGLE, tirée de cet épisode : un index est un ORDRE LOCAL, jamais une IDENTITÉ.
// L'identité d'un joueur est son XUID — stable, global, indépendant de tout tri. Toute
// jointure passe par lui.
//
// CE QUE LE TEST INTERDIT concrètement : réintroduire un champ nommé `PlayerIndex` dans les
// paquets du rejeu et du décodage de film. Le champ existe, mais il s'appelle `FilmIndex`,
// et ce nom porte son statut : il n'est valable qu'à l'intérieur d'un film.
//
// POURQUOI UN TEST SUR LE NOM plutôt que sur l'usage. Interdire « utiliser un index comme
// clé de jointure » demanderait une analyse de flot que ce dépôt n'a pas. Le nom, lui, est
// la seule chose qu'un relecteur voit au moment où il écrit la jointure — c'est là que la
// confusion se produit, et c'est donc là qu'on la bloque. La règle du dépôt le dit
// autrement : à la troisième copie d'un pattern, on centralise ET on pose un garde-rail ;
// une factorisation sans garde-rail re-diverge.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// playerIndexScope : les paquets où l'index de film circule. Ailleurs dans le dépôt,
// `PlayerIndex` désigne autre chose (position dans une équipe pour le calcul de rating) et
// n'a rien à voir avec cette confusion.
var playerIndexScope = []string{
	filepath.Join("analysis", "filmdec"),
	filepath.Join("analysis", "replay"),
}

// playerIndexRE matche une DÉCLARATION ou un accès de champ nommé `PlayerIndex`.
var playerIndexRE = regexp.MustCompile(`\bPlayerIndex\b`)

func TestNoPlayerIndexInFilmScope(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	for _, scope := range playerIndexScope {
		root := filepath.Join(internalRoot, scope)
		if _, err := os.Stat(root); err != nil {
			continue // le paquet peut avoir été déplacé : ce n'est pas l'objet de ce test
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(src), "\n") {
				// Les commentaires ont le droit de NOMMER l'ancien champ : c'est ainsi qu'ils
				// expliquent pourquoi il a changé de nom. Seul le code est interdit.
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				if playerIndexRE.MatchString(line) {
					rel, _ := filepath.Rel(internalRoot, path)
					violations = append(violations,
						filepath.ToSlash(rel)+":"+itoa(i+1)+" : "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("`PlayerIndex` est réapparu dans le périmètre du film (%d occurrences).\n"+
			"Un index de film est un ORDRE, pas une identité : le champ s'appelle `FilmIndex`, "+
			"et une jointure entre joueurs passe par le XUID.\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// itoa évite d'importer strconv pour un seul appel.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
