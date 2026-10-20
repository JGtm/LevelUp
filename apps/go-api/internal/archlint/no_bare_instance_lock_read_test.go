// Package archlint — no_bare_instance_lock_read_test.go : ratchet « lecture nue
// du verrou d'instance » (ADR 0035 D5, 2026-09-15).
//
// POURQUOI. Le verrou « instance fermée » est le OU de deux sources : l'env
// LEVELUP_INSTANCE_LOCKED (config.AppConfig.InstanceLocked) et la clé
// app_settings.instance_locked. Le 2026-09-15, TROIS copies de ce calcul
// coexistaient — api/handlers/setup.go, api/server_apiv1.go et
// service/bootstrap_service.go — et une seule des trois journalisait son repli sur
// settings illisibles. C'est le seuil de la règle CLAUDE.md n°6 (3e copie →
// helper + garde-rail) : le point de décision unique est désormais
// authz.InstanceLocked, construit UNE fois dans api/server_apiv1.go et injecté
// aux consommateurs sous la forme d'un `func() bool`. Sans ce ratchet, un 4e
// lecteur re-poserait un `cfg.InstanceLocked || s.InstanceLocked` et la divergence
// (un chemin verrouillé, un autre non) reviendrait.
//
// PORTÉE. Tout le module. Est interdite la lecture du CHAMP (`x.InstanceLocked`)
// hors des packages qui le définissent ou le possèdent légitimement :
//
//   - internal/authz/            — le point de décision lui-même ;
//   - internal/config/           — porte la valeur d'env ;
//   - internal/platform/settings — porte la valeur du fichier (+ son PATCH) ;
//   - internal/domain/           — déclare les DTO qui transportent le verrou ;
//   - internal/api/handlers/settings.go — le toggle admin (PATCH /settings) ;
//   - internal/api/server_apiv1.go      — construction du résolveur unique.
//
// L'APPEL `authz.InstanceLocked(...)` n'est jamais une violation (c'est la
// fonction, pas le champ) et les commentaires sont ignorés. Les `_test.go` sont
// exclus : ils construisent des fixtures.
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

// instanceLockFieldRE — lecture du champ InstanceLocked sur un récepteur
// quelconque (`cfg.InstanceLocked`, `s.InstanceLocked`, `appCfg.InstanceLocked`…).
var instanceLockFieldRE = regexp.MustCompile(`\.InstanceLocked\b`)

// authzCallRE — l'appel au point de décision unique, jamais une violation.
var authzCallRE = regexp.MustCompile(`\bauthz\.InstanceLocked\b`)

// instanceLockAllowlist — chemins autorisés à lire le champ, relatifs à la racine
// du module, en séparateurs `/`. Un préfixe terminé par `/` couvre le package et
// ses sous-packages ; sinon c'est un fichier exact. Datée du 2026-09-15 : toute
// entrée ajoutée ici doit porter sa date et sa justification (ADR 0035 D5).
var instanceLockAllowlist = []string{
	"internal/authz/",
	"internal/config/",
	"internal/platform/settings/",
	"internal/domain/",
	"internal/api/handlers/settings.go",
	"internal/api/server_apiv1.go",
}

// skippedDirs — arborescences hors code source du module.
var instanceLockSkippedDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true,
	"data": true, "dist": true, "logs": true,
}

func instanceLockAllowed(rel string) bool {
	for _, allowed := range instanceLockAllowlist {
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

func TestNoBareInstanceLockRead(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// internal/archlint/<fichier> → racine du module.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if instanceLockSkippedDirs[d.Name()] {
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
		if instanceLockAllowed(rel) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue // commentaire : documente le résolveur, ne le contourne pas
			}
			if !instanceLockFieldRE.MatchString(line) || authzCallRE.MatchString(line) {
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
		t.Errorf("lecture NUE du verrou d'instance hors allowlist — le verrou se résout "+
			"en UN point (authz.InstanceLocked, construit dans api/server_apiv1.go) et "+
			"s'injecte en `func() bool` (ADR 0035 D5) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
