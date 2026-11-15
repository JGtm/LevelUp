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
	"encoding/json"
	"fmt"
	"os"
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
	"film":      "github.com/JGtm/halo-infinite-film-decoder",
	"gamefiles": "github.com/JGtm/halo-infinite-files-reader",
	"haloapi":   "github.com/JGtm/halo-api",
	"himap":     "github.com/JGtm/halo-infinite-map-images",
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

// ImportDeGardeRail rend le chemin d'import Go d'un chemin de garde-rail `libs/<brique>[/<suite>]` :
// `libs/film/types` -> `<module de film>/types`. ok est faux quand rel ne désigne pas une brique connue.
func ImportDeGardeRail(rel string) (string, bool) {
	reste, ok := strings.CutPrefix(rel, prefixeGardeRail)
	if !ok {
		return "", false
	}
	nom, suite, _ := strings.Cut(reste, "/")
	module, ok := modules[nom]
	if !ok {
		return "", false
	}
	if suite == "" {
		return module, true
	}
	return module + "/" + suite, true
}

// GardeRailDeImport est l'inverse de [ImportDeGardeRail] : le chemin de garde-rail d'un paquet d'une
// brique donné par son chemin d'import ; ok est faux pour un import qui n'est ni un module de brique ni
// un paquet dessous (la frontière d'un chemin compte : `<module>x` n'est pas sous `<module>`).
func GardeRailDeImport(imp string) (string, bool) {
	for _, nom := range Noms() {
		module := modules[nom]
		if imp == module {
			return prefixeGardeRail + nom, true
		}
		if suite, ok := strings.CutPrefix(imp, module+"/"); ok {
			return prefixeGardeRail + nom + "/" + suite, true
		}
	}
	return "", false
}

// EstImportDeBrique dit si le chemin d'import vise un module de brique ou un paquet dessous.
func EstImportDeBrique(imp string) bool {
	_, ok := GardeRailDeImport(imp)
	return ok
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
		dossiers, errDossiers = resoudreAvec(racine, nil)
	})
	return dossiers, errDossiers
}

// resoudreAvec interroge l'outil Go, lancé depuis racine avec les variables d'environnement `env`
// (`CLE=valeur`) en plus de celles du processus, sur le dossier de chaque brique. Sans cache : [resoudre]
// y ajoute le « une seule fois » ; les tests y posent `GOWORK`.
func resoudreAvec(racine string, env []string) (map[string]string, error) {
	noms := Noms()
	args := append(make([]string, 0, 4+len(noms)), "list", "-m", "-f", "{{.Path}}\t{{.Dir}}")
	for _, nom := range noms {
		args = append(args, modules[nom])
	}
	cmd := childproc.Command(context.Background(), "go", args...)
	cmd.Dir = racine
	cmd.Env = append(os.Environ(), env...)
	sortie, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("briques : go %s : %w", strings.Join(args, " "), err)
	}
	parModule := map[string]string{}
	for _, ligne := range strings.Split(strings.TrimSpace(string(sortie)), "\n") {
		chemin, dir, ok := strings.Cut(strings.TrimRight(ligne, "\r"), "\t")
		if ok && dir != "" {
			parModule[chemin] = dir
		}
	}
	out := map[string]string{}
	for _, nom := range noms {
		dir, ok := parModule[modules[nom]]
		if !ok {
			var errDl error
			if dir, errDl = telecharger(racine, env, modules[nom]); errDl != nil {
				return nil, fmt.Errorf("briques : l'outil Go ne rend aucun dossier pour le module %s : %w", modules[nom], errDl)
			}
		}
		out[nom] = dir
	}
	return out, nil
}

// telecharger rend le dossier d'un module requis absent du cache de modules (`go list -m` ne lui donne
// alors aucun dossier : cas d'un cache neuf où seul le build a tiré ses dépendances), après
// `go mod download`.
func telecharger(racine string, env []string, module string) (string, error) {
	cmd := childproc.Command(context.Background(), "go", "mod", "download", "-json", module)
	cmd.Dir = racine
	cmd.Env = append(os.Environ(), env...)
	sortie, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go mod download %s : %w", module, err)
	}
	var info struct{ Dir string }
	if err := json.Unmarshal(sortie, &info); err != nil {
		return "", fmt.Errorf("go mod download %s : sortie illisible : %w", module, err)
	}
	if info.Dir == "" {
		return "", fmt.Errorf("go mod download %s : aucun dossier rendu", module)
	}
	return info.Dir, nil
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

var (
	onceCache    sync.Once
	dossierCache string
	errCache     error
)

// cacheDeModules rend le dossier du cache de modules de l'outil Go (`go env GOMODCACHE`), une seule fois.
func cacheDeModules() (string, error) {
	onceCache.Do(func() {
		racine, err := racineModuleAPI()
		if err != nil {
			errCache = err
			return
		}
		dossierCache, errCache = cacheDeModulesAvec(racine, nil)
	})
	return dossierCache, errCache
}

// cacheDeModulesAvec interroge l'outil Go comme [resoudreAvec], sans cache.
func cacheDeModulesAvec(racine string, env []string) (string, error) {
	cmd := childproc.Command(context.Background(), "go", "env", "GOMODCACHE")
	cmd.Dir = racine
	cmd.Env = append(os.Environ(), env...)
	sortie, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("briques : go env GOMODCACHE : %w", err)
	}
	dir := strings.TrimSpace(string(sortie))
	if dir == "" {
		return "", fmt.Errorf("briques : go env GOMODCACHE ne rend aucun dossier")
	}
	return dir, nil
}

// Modifiable rend nil quand le dossier de la brique est une copie de travail qu'un outil peut réécrire :
// un clone du dépôt de la brique, lié à LevelUp par l'espace de travail Go actif (`GOWORK`, engendré par
// scripts/briques-workspace.sh), où qu'il soit sur le disque. Elle rend une erreur quand le dossier est
// dans le cache de modules : celui-ci est en lecture seule et n'appartient à personne, un outil qui écrit
// ne l'y suit jamais. Sans espace de travail, l'outil Go résout toutes les briques vers le cache : l'erreur
// dit alors d'en désigner un.
func Modifiable(brique string) error {
	dir, err := Dir(brique)
	if err != nil {
		return err
	}
	cache, err := cacheDeModules()
	if err != nil {
		return err
	}
	return controlerModifiable(brique, dir, cache)
}

// controlerModifiable est le jugement de [Modifiable] sur un dossier de brique et le cache de modules,
// sans interroger l'outil Go. Les deux chemins sont comparés après résolution des liens (jonctions et
// liens symboliques) : un clone atteint par une jonction n'est pas pris pour le cache, ni l'inverse.
func controlerModifiable(brique, dir, cache string) error {
	if !estSous(resoudreLiens(cache), resoudreLiens(dir)) {
		return nil
	}
	return fmt.Errorf("briques : le dossier de la brique %s (%s) est dans le cache de modules (%s), en lecture seule : "+
		"refus d'y écrire. Travailler sur un clone du dépôt de la brique relié par l'espace de travail local : "+
		"`bash scripts/briques-workspace.sh <clone>` puis GOWORK=<racine du dépôt>/levelup-briques.work", brique, dir, cache)
}

// resoudreLiens rend le chemin sans lien intermédiaire, ou le chemin nettoyé tel quel quand il n'existe pas.
func resoudreLiens(chemin string) string {
	if r, err := filepath.EvalSymlinks(chemin); err == nil {
		return r
	}
	return filepath.Clean(chemin)
}

// estSous dit si enfant est racine ou un chemin dessous (la frontière d'un nom de dossier compte).
func estSous(racine, enfant string) bool {
	rel, err := filepath.Rel(racine, enfant)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
