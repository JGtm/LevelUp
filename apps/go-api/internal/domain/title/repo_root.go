package title

import (
	"fmt"
	"os"
	"path/filepath"
)

// repoRootMarker est le fichier qui marque la racine du dépôt. Il est GITIGNORÉ (config des
// joueurs suivis) : présent sur un poste de travail configuré, absent d'un checkout neuf et
// de tout worktree git — d'où le repli versionné ci-dessous.
const repoRootMarker = "db_profiles.json"

// repoRootMarkersVersionnes : le repli, mesuré par la découverte D4 (3.1.2) du 2026-09-16 —
// `go run ./cmd/film-profiles-build` sortait en 1 depuis un worktree (`LEVELUP_REPO_ROOT non
// défini et db_profiles.json introuvable en remontant depuis le cwd`), et le défaut valait pour
// TOUS les outils de catalogue.
//
// LES DEUX MARQUEURS SONT EXIGÉS ENSEMBLE, et c'est le point : `apps/go-api/go.mod` seul
// désignerait le module Go de n'importe quelle copie décompressée, `.git` seul désignerait
// n'importe quel dépôt. Leur conjonction ne décrit que la racine de CE dépôt, et elle est
// versionnée — donc vraie sur un checkout neuf, sur un worktree et en CI.
//
// `.git` est cherché en FICHIER OU EN DOSSIER : un worktree git n'a pas de dossier `.git`, il a
// un fichier `.git` de 91 octets qui pointe vers le dépôt principal. C'est exactement le cas que
// ce repli existe pour servir.
var repoRootMarkersVersionnes = []string{
	filepath.Join("apps", "go-api", "go.mod"),
	".git",
}

// FindRepoRoot renvoie la racine du dépôt, dans cet ORDRE DE PRIORITÉ :
//
//  1. LEVELUP_REPO_ROOT si la variable est définie (même variable que le serveur, garantit une
//     résolution de chemins identique) ;
//  2. la remontée depuis le répertoire courant jusqu'au marqueur `db_profiles.json` — l'ordre
//     historique, inchangé : un poste configuré rend exactement ce qu'il rendait ;
//  3. à défaut, la remontée jusqu'aux marqueurs VERSIONNÉS (`apps/go-api/go.mod` + `.git`).
//
// Destiné aux OUTILS HORS LIGNE (cmd/*-build) qui n'ouvrent pas de DB : il évite
// d'importer internal/config, dont la chaîne DuckDB impose CGO.
func FindRepoRoot() (string, error) {
	if r := os.Getenv("LEVELUP_REPO_ROOT"); r != "" {
		return r, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, ok := remonteVers(dir, func(d string) bool { return existe(filepath.Join(d, repoRootMarker)) }); ok {
		return root, nil
	}
	if root, ok := remonteVers(dir, porteLesMarqueursVersionnes); ok {
		return root, nil
	}
	return "", fmt.Errorf("LEVELUP_REPO_ROOT non défini, et ni %s ni les marqueurs versionnés "+
		"(%s) trouvés en remontant depuis le cwd", repoRootMarker, filepath.Join("apps", "go-api", "go.mod")+" + .git")
}

// porteLesMarqueursVersionnes : le répertoire porte TOUS les marqueurs versionnés.
func porteLesMarqueursVersionnes(dir string) bool {
	for _, m := range repoRootMarkersVersionnes {
		if !existe(filepath.Join(dir, m)) {
			return false
		}
	}
	return true
}

// remonteVers remonte de `dir` vers la racine du volume et rend le premier répertoire que
// `reconnait` accepte.
func remonteVers(dir string, reconnait func(string) bool) (string, bool) {
	for {
		if reconnait(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// existe : le chemin existe, fichier ou dossier (un worktree git porte un `.git` FICHIER).
func existe(chemin string) bool {
	_, err := os.Stat(chemin)
	return err == nil
}
