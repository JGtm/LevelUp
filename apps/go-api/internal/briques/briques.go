// Package briques est le résolveur unique des briques Go de LevelUp : tout code qui lit un fichier
// d'une brique sur le disque (fixtures, goldens, sources balayées par un garde-rail, registres versionnés)
// en demande le dossier ici, jamais par un chemin écrit à la main depuis la racine du dépôt.
//
// Contrat. Le dossier d'une brique est celui que l'outil Go désigne pour son module
// (`go list -m -f {{.Dir}} <module>`, lancé depuis la racine du module `apps/go-api`) : le dossier de
// l'espace de travail tant que la brique y est liée par `go.work`, le cache de modules sinon. Le chemin
// d'un fichier d'une brique n'est donc pas relatif au dépôt, et le dossier rendu peut être en lecture
// seule : seul un outil qui écrit passe par [Modifiable].
//
// Le chemin de garde-rail (`libs/<brique>/<suite>`, séparateurs "/") reste la clé logique des tables des
// garde-rails ; [SurDisque] et [DeGardeRail] en font la correspondance avec le disque.
//
// Ce paquet ne sert qu'aux tests et aux outils hors ligne : le serveur ne l'importe pas (il lance
// l'outil Go).
package briques

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"levelup/go-api/internal/platform/childproc"
)

// prefixeGardeRail ouvre le chemin de garde-rail d'un fichier de brique.
const prefixeGardeRail = "libs/"

// modules associe chaque brique au chemin de son module.
var modules = map[string]string{
	"film":      "github.com/JGtm/LevelUp/libs/film",
	"gamefiles": "github.com/JGtm/LevelUp/libs/gamefiles",
	"haloapi":   "github.com/JGtm/LevelUp/libs/haloapi",
	"himap":     "github.com/JGtm/LevelUp/libs/himap",
}

// Noms rend les briques connues, triées.
func Noms() []string {
	out := make([]string, 0, len(modules))
	for nom := range modules {
		out = append(out, nom)
	}
	sort.Strings(out)
	return out
}

// Module rend le chemin du module d'une brique.
func Module(brique string) (string, bool) {
	m, ok := modules[brique]
	return m, ok
}

var (
	onceDossiers sync.Once
	dossiers     map[string]string
	errDossiers  error
)

// racineModuleAPI rend la racine du module `apps/go-api`, d'où l'outil Go est lancé : elle se déduit
// de l'emplacement de ce fichier source (apps/go-api/internal/briques).
func racineModuleAPI() (string, error) {
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("briques : runtime.Caller a échoué, racine de apps/go-api indéductible")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ce))), nil
}

// resoudre interroge l'outil Go une seule fois pour toutes les briques.
func resoudre() (map[string]string, error) {
	onceDossiers.Do(func() {
		racine, err := racineModuleAPI()
		if err != nil {
			errDossiers = err
			return
		}
		noms := Noms()
		args := []string{"list", "-m", "-f", "{{.Path}}\t{{.Dir}}"}
		for _, nom := range noms {
			args = append(args, modules[nom])
		}
		cmd := childproc.Command(context.Background(), "go", args...)
		cmd.Dir = racine
		sortie, err := cmd.Output()
		if err != nil {
			errDossiers = fmt.Errorf("briques : go %s : %w", strings.Join(args, " "), err)
			return
		}
		parModule := map[string]string{}
		for _, ligne := range strings.Split(strings.TrimSpace(string(sortie)), "\n") {
			chemin, dir, ok := strings.Cut(strings.TrimRight(ligne, "\r"), "\t")
			if ok && dir != "" {
				parModule[chemin] = dir
			}
		}
		dossiers = map[string]string{}
		for _, nom := range noms {
			dir, ok := parModule[modules[nom]]
			if !ok {
				errDossiers = fmt.Errorf("briques : l'outil Go ne rend aucun dossier pour le module %s", modules[nom])
				return
			}
			dossiers[nom] = dir
		}
	})
	return dossiers, errDossiers
}

// Dir rend le dossier d'une brique sur le disque.
func Dir(brique string) (string, error) {
	if _, ok := modules[brique]; !ok {
		return "", fmt.Errorf("briques : brique inconnue %q (connues : %s)", brique, strings.Join(Noms(), ", "))
	}
	d, err := resoudre()
	if err != nil {
		return "", err
	}
	return d[brique], nil
}

// Chemin rend le chemin, sur le disque, d'un élément d'une brique.
func Chemin(brique string, elems ...string) (string, error) {
	dir, err := Dir(brique)
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dir}, elems...)...), nil
}

// SurDisque rend le chemin absolu d'un chemin de garde-rail `libs/<brique>[/<suite>]`. ok est faux quand
// rel ne désigne pas une brique (le chemin est alors relatif à apps/go-api) ; l'erreur dit une brique
// inconnue ou un outil Go muet.
func SurDisque(rel string) (abs string, ok bool, err error) {
	if !strings.HasPrefix(rel, prefixeGardeRail) {
		return "", false, nil
	}
	nom, suite, _ := strings.Cut(strings.TrimPrefix(rel, prefixeGardeRail), "/")
	dir, err := Dir(nom)
	if err != nil {
		return "", true, err
	}
	return filepath.Join(dir, filepath.FromSlash(suite)), true, nil
}

// DeGardeRail est l'inverse de [SurDisque] : le chemin de garde-rail d'un fichier ou d'un dossier absolu
// d'une brique ; ok est faux hors de toute brique.
func DeGardeRail(abs string) (rel string, ok bool, err error) {
	d, err := resoudre()
	if err != nil {
		return "", false, err
	}
	for _, nom := range Noms() {
		r, errRel := filepath.Rel(d[nom], abs)
		if errRel != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			continue
		}
		if r == "." {
			return prefixeGardeRail + nom, true, nil
		}
		return prefixeGardeRail + nom + "/" + filepath.ToSlash(r), true, nil
	}
	return "", false, nil
}

// racineDepot rend la racine du dépôt : deux niveaux au-dessus de `apps/go-api`.
func racineDepot() (string, error) {
	api, err := racineModuleAPI()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(api)), nil
}

// Modifiable rend nil quand le dossier de la brique est dans le dépôt (une copie de travail qu'un outil
// peut réécrire) et une erreur sinon : un dossier du cache de modules n'est ni à soi ni inscriptible, et
// un outil qui écrit ne l'y suit jamais.
func Modifiable(brique string) error {
	dir, err := Dir(brique)
	if err != nil {
		return err
	}
	depot, err := racineDepot()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(depot, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("briques : le dossier de la brique %s (%s) est hors du dépôt : refus d'y écrire", brique, dir)
	}
	return nil
}
