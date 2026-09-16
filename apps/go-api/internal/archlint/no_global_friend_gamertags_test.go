// Package archlint — no_global_friend_gamertags_test.go : ratchet anti-résurrection
// du réglage GLOBAL d'amis (`friend_gamertags` dans app_settings.json).
//
// Les amis sont PAR PROFIL JOUEUR depuis le 2026-09-15 (clé xuid,
// data/global/player_friends.json, platform/friendstore). L'ancienne liste était
// UNE liste pour toute l'instance : elle pilotait `is_with_friends` dans TOUTES
// les player DBs et n'était lisible que par un admin (GET /settings sous
// RequireAdmin) — un utilisateur standard n'avait donc aucune fonctionnalité
// « amis ». Ré-introduire un champ global la ferait revenir en silence.
//
// Deux interdits, ALLOWLIST VIDE (hors le paquet de migration lui-même) :
//
//  1. Le littéral `friend_gamertags` (clé JSON) hors `internal/platform/friendstore/`
//     — seul le code de migration a encore le droit de la lire, et il la lit en
//     `map[string]json.RawMessage`, sans champ typé. Tests inclus : une fixture qui
//     repose la clé rendrait le champ légitime par accident.
//  2. Un champ `FriendGamertags` dans les structures de réglages
//     (`internal/domain/settings.go`, `internal/platform/settings/store.go`) —
//     c'est la forme exacte du défaut d'origine.
//
// Les identifiants `FriendGamertagsResolver` / `squadagg.FriendGamertags` restent
// autorisés : ce sont des listes résolues PAR JOUEUR ou des paramètres de requête,
// pas un réglage d'instance. Le test vise la SOURCE, pas le vocabulaire.
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

// friendStorePkg : SEUL chemin autorisé à nommer la clé legacy (migration one-shot).
const friendStorePkg = "internal/platform/friendstore/"

// settingsStructFiles : les fichiers de structures de réglages, où un champ
// `FriendGamertags` signerait le retour du réglage global.
var settingsStructFiles = []string{
	"internal/domain/settings.go",
	"internal/platform/settings/store.go",
}

var settingsFriendFieldRE = regexp.MustCompile(`\bFriendGamertags\b`)

func TestNoGlobalFriendGamertagsKey(t *testing.T) {
	moduleRoot := archlintModuleRoot(t)

	var violations []string
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(moduleRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, friendStorePkg) || rel == thisGuardRailFile {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "friend_gamertags") {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("clé legacy `friend_gamertags` interdite hors %s (les amis sont par joueur, "+
			"data/global/player_friends.json) :\n  %s", friendStorePkg, strings.Join(violations, "\n  "))
	}
}

func TestSettingsStructsHaveNoFriendGamertagsField(t *testing.T) {
	moduleRoot := archlintModuleRoot(t)

	for _, rel := range settingsStructFiles {
		data, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("lecture %s : %v", rel, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if settingsFriendFieldRE.MatchString(line) {
				t.Errorf("%s:%d — le réglage GLOBAL d'amis est supprimé (2026-09-15) : "+
					"les amis sont par joueur via platform/friendstore. Ligne : %s",
					rel, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// thisGuardRailFile : ce fichier cite la clé legacy dans sa propre documentation.
const thisGuardRailFile = "internal/archlint/no_global_friend_gamertags_test.go"

// archlintModuleRoot retourne la racine du module go-api (parent de internal/).
func archlintModuleRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}
