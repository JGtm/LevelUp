// Package archlint — no_users_json_literal_test.go : ratchet « chemin du store
// des comptes construit à la main ».
//
// POURQUOI. Le 2026-09-16, `filepath.Join(cfg.AuthDir, "users.json")` existait en
// trois exemplaires (cmd/server, api/server, cmd/levelup identity). Au 3e, la
// règle 6 de CLAUDE.md impose le helper canonique — config.AppConfig.UsersFilePath
// — ET ce garde-rail : sans lui, un 4e appelant re-poserait le littéral et, le
// jour où le nom ou l'emplacement du fichier bouge (surcharge LEVELUP_AUTH_DIR,
// migration), un lecteur ouvrirait un fichier vide pendant que les autres
// écrivent le bon.
//
// PORTÉE. Tout le module Go hors tests et hors le helper lui-même
// (internal/config) et le store qui documente le fichier (platform/userstore).
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

// usersJSONLiteralRE — le littéral du fichier des comptes dans du code (pas un
// commentaire : les lignes commençant par // sont ignorées ci-dessous).
var usersJSONLiteralRE = regexp.MustCompile(`"users\.json"`)

// usersJSONAllowlist — fichiers autorisés à porter le littéral (2026-09-16) :
// le helper canonique et le store des comptes.
var usersJSONAllowlist = []string{
	"internal/config/config_paths.go",
	"internal/platform/userstore/store.go",
}

func TestNoUsersJSONLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(moduleRoot, path)
		rel = filepath.ToSlash(rel)
		for _, allowed := range usersJSONAllowlist {
			if rel == allowed {
				return nil
			}
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if usersJSONLiteralRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("chemin du store des comptes construit à la main — passer par "+
			"config.AppConfig.UsersFilePath() (helper canonique, CLAUDE.md règle 6) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
