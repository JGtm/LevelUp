package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// dbBusyLiteralDefFile : SEUL fichier autorisé à porter le littéral "db_busy"
// dans le code de production du paquet — celui qui DÉFINIT le helper errDBBusy.
const dbBusyLiteralDefFile = "helpers.go"

// reDBBusyLiteral : le littéral Go `"db_busy"`, guillemets compris. Le guillemet
// ouvrant fait partie du motif → "home_page_db_busy" (code distinct de home.go,
// dégradation de page et non refus d'écriture) n'est PAS capturé, pas plus que
// les mentions en commentaire sans guillemets.
var reDBBusyLiteral = regexp.MustCompile(`"db_busy"`)

// TestNoDBBusyLiteralInHandlers — garde-rail CLAUDE.md règle n°6. L'enveloppe
// 503 « base occupée » (huma.ErrorWithHeaders + humacore.NewError(503,
// "db_busy", "database is currently busy, please retry") + Retry-After:5) était
// recopiée à l'identique dans 7 handlers write : notifications, media_likes,
// media_delete, match_favorite, match_exclusion, engagement, prestige_squads.
// Sept copies d'un contrat client (code + message + header) qu'un seul oubli de
// mise à jour suffit à désaligner. Tous passent désormais par errDBBusy()
// (helpers.go) ; ce test interdit la réapparition du littéral.
//
// Périmètre restreint au paquet handlers, `_test.go` EXCLUS : les tests
// vérifient le code d'erreur tel que le client HTTP le lit — c'est précisément
// le contrat à assurer, et l'exprimer via le helper testerait le helper contre
// lui-même.
func TestNoDBBusyLiteralInHandlers(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile) // internal/api/handlers

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") ||
			name == dbBusyLiteralDefFile {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		if reDBBusyLiteral.Match(src) {
			offenders = append(offenders, name)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("littéral \"db_busy\" dupliqué hors %s — "+
			"utiliser le helper de paquet errDBBusy() :\n  %s",
			dbBusyLiteralDefFile, strings.Join(offenders, "\n  "))
	}
}

// TestDBBusyGuardIsDiscriminant prouve que le garde-rail MORD sur une enveloppe
// réintroduite et ne produit pas de faux positif sur les usages légitimes.
//
// Vérifie AUSSI que le fichier de définition contient réellement errDBBusy : si
// le helper déménageait, l'exemption de helpers.go deviendrait un trou silencieux.
func TestDBBusyGuardIsDiscriminant(t *testing.T) {
	mustMatch := []string{
		`humacore.NewError(http.StatusServiceUnavailable, "db_busy", "database is currently busy, please retry")`,
		`writeError(ctx, w, 503, "db_busy", msg)`,
	}
	for _, src := range mustMatch {
		if !reDBBusyLiteral.MatchString(src) {
			t.Errorf("littéral NON détecté (garde-rail aveugle) : %q", src)
		}
	}

	mustNotMatch := []string{
		`return errDBBusy()`,
		`humacore.NewError(http.StatusServiceUnavailable, "home_page_db_busy", msg)`,
		`// ErrDBLocked est mappé sur le code db_busy`,
	}
	for _, src := range mustNotMatch {
		if reDBBusyLiteral.MatchString(src) {
			t.Errorf("FAUX POSITIF sur usage légitime : %q", src)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	defPath := filepath.Join(filepath.Dir(thisFile), dbBusyLiteralDefFile)
	src, err := os.ReadFile(defPath)
	if err != nil {
		t.Fatalf("lecture %s: %v", dbBusyLiteralDefFile, err)
	}
	if !strings.Contains(string(src), "func errDBBusy()") {
		t.Errorf("%s ne définit plus errDBBusy : son exemption dans "+
			"TestNoDBBusyLiteralInHandlers est devenue un trou — déplacer "+
			"l'exemption vers le nouveau fichier de définition", dbBusyLiteralDefFile)
	}
}
