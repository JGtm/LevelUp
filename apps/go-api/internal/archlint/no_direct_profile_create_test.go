// Package archlint — no_direct_profile_create_test.go : ratchet « création de
// profil hors de l'annuaire » (ADR 0035 D4, 2026-09-16).
//
// POURQUOI. Le 2026-07-23, un compte Xbox inconnu a été créé, ses credentials
// persistés et le joueur ajouté au watcher — sans qu'aucun profil de suivi
// n'existe. Deux heures plus tard, un sync écrivait 25 matchs dans l'entrepôt
// partagé pour un joueur que plus rien en aval ne savait résoudre. La cause de
// fond n'est pas un oubli : c'est qu'il y avait DEUX endroits où l'on décidait
// qu'un joueur existe (le wizard créait le profil, le SSO ouvrait le suivi), et
// aucun ordre garanti entre les deux.
//
// Depuis l'ADR 0035 D4, `PlayerDirectory.Onboard` est le SEUL chemin : il crée le
// profil PUIS notifie le watcher, dans cet ordre — que la porte « profil suivi »
// de `watcher.Daemon.AddPlayer` (D3) rend obligatoire. Sans ce garde-rail, un
// second appelant de `CreatePlayer(` re-poserait un chemin qui court-circuite la
// notification (et l'ordre), et le trou se rouvrirait en silence.
//
// PORTÉE. Tout le module. Est interdit l'APPEL d'une méthode `CreatePlayer(` sur
// un récepteur quelconque (`h.profileSvc.CreatePlayer(`, `svc.CreatePlayer(`…)
// hors des paquets autorisés ci-dessous. La DÉFINITION de la méthode
// (`func (s *ProfileService) CreatePlayer(`) n'est jamais une violation : le motif
// exige un sélecteur, donc un point devant. Les commentaires sont ignorés, les
// `_test.go` exclus (ils construisent des doubles et les appellent directement).
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

// profileCreateCallRE — appel de CreatePlayer sur un récepteur (sélecteur).
// `func (s *ProfileService) CreatePlayer(` ne matche pas (pas de point devant le
// nom), pas plus que la déclaration d'interface `CreatePlayer(req …)`.
var profileCreateCallRE = regexp.MustCompile(`\.CreatePlayer\(`)

// profileCreateAllowlist — chemins autorisés à appeler CreatePlayer, relatifs à
// la racine du module, en séparateurs `/`. Un préfixe terminé par `/` couvre le
// paquet et ses sous-paquets. Datée du 2026-09-16 : toute entrée ajoutée ici doit
// porter sa date ET sa justification (ADR 0035 D4). Une seule entrée est
// légitime — l'annuaire lui-même ; en ajouter une seconde, c'est rouvrir le trou
// du 2026-07-23.
var profileCreateAllowlist = []string{
	"internal/service/playerdirectory/",
}

// profileCreateSkippedDirs — arborescences hors code source du module.
var profileCreateSkippedDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true,
	"data": true, "dist": true, "logs": true,
}

func profileCreateAllowed(rel string) bool {
	for _, allowed := range profileCreateAllowlist {
		if strings.HasSuffix(allowed, "/") {
			if strings.HasPrefix(rel, allowed) {
				return true
			}
			continue
		}
		if rel == allowed {
			return true
		}
	}
	return false
}

func TestNoDirectProfileCreate(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// internal/archlint/<fichier> → racine du module.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	err := filepath.WalkDir(moduleRoot, func(path string, dirEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if dirEntry.IsDir() {
			if profileCreateSkippedDirs[dirEntry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(moduleRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if profileCreateAllowed(rel) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue // un commentaire cite le motif, il ne le contourne pas
			}
			if !profileCreateCallRE.MatchString(line) {
				continue
			}
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", moduleRoot, err)
	}

	if len(violations) > 0 {
		t.Errorf("création de profil HORS de l'annuaire — un profil se crée par "+
			"PlayerDirectory.Onboard, qui écrit le profil PUIS notifie le watcher "+
			"(ADR 0035 D4 ; l'ordre inverse est refusé par la porte profil du daemon) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
