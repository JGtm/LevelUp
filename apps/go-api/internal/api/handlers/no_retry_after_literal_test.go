package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// retryAfterLiteralDefFile : SEUL fichier autorisé à porter le littéral
// "Retry-After" dans le code de production du paquet — celui qui DÉFINIT la
// constante headerRetryAfter.
const retryAfterLiteralDefFile = "helpers.go"

// reRetryAfterLiteral : le littéral Go `"Retry-After"`, guillemets compris. Ancrer
// sur les guillemets évite de matcher les mentions en commentaire ou les
// identifiants (headerRetryAfter, parseRetryAfter…) : seule la chaîne littérale
// re-diverge, pas le mot.
var reRetryAfterLiteral = regexp.MustCompile(`"Retry-After"`)

// TestNoRetryAfterLiteralInHandlers — garde-rail CLAUDE.md règle n°6. Le nom de
// l'en-tête HTTP 503 était copié dans 10 handlers (home ×2, engagement,
// match_favorite, match_exclusion, notifications, prestige_squads,
// sync_handler_align, media_delete, media_likes) — le ratchet goconst a mordu au
// lot 2 v7.3. Tous les sites passent désormais par la constante de paquet
// headerRetryAfter (helpers.go) ; ce test interdit la réapparition du littéral.
//
// Périmètre volontairement restreint au paquet handlers, pour deux raisons :
//   - les fichiers `_test.go` sont EXCLUS : ils lisent l'en-tête par son NOM HTTP
//     réel (`w.Header().Get("Retry-After")`), ce qui est exactement le contrat
//     client à vérifier — leur imposer la constante testerait la constante
//     contre elle-même au lieu de tester le protocole ;
//   - les clients HTTP d'autres paquets (sync/haloclient, games/halo_5) LISENT
//     l'en-tête d'une réponse tierce : ce n'est pas la même valeur métier et la
//     constante, privée au paquet, ne leur est pas accessible.
func TestNoRetryAfterLiteralInHandlers(t *testing.T) {
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
			name == retryAfterLiteralDefFile {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		if reRetryAfterLiteral.Match(src) {
			offenders = append(offenders, name)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("littéral \"Retry-After\" dupliqué hors %s — "+
			"utiliser la constante de paquet headerRetryAfter :\n  %s",
			retryAfterLiteralDefFile, strings.Join(offenders, "\n  "))
	}
}

// TestRetryAfterGuardIsDiscriminant prouve que le garde-rail ci-dessus MORD, et
// qu'il ne mord QUE sur le littéral :
//   - un `http.Header{"Retry-After": …}` réintroduit est détecté ;
//   - l'usage correct (constante) et les identifiants qui contiennent le mot ne
//     produisent aucun faux positif.
//
// Vérifie AUSSI que le fichier de définition contient réellement la constante :
// si headerRetryAfter déménageait, l'exemption de helpers.go deviendrait un trou
// silencieux (le littéral serait alors toléré dans un fichier qui ne définit plus rien).
func TestRetryAfterGuardIsDiscriminant(t *testing.T) {
	mustMatch := []string{
		`http.Header{"Retry-After": []string{"5"}}`,
		`w.Header().Set("Retry-After", "30")`,
	}
	for _, src := range mustMatch {
		if !reRetryAfterLiteral.MatchString(src) {
			t.Errorf("littéral NON détecté (garde-rail aveugle) : %q", src)
		}
	}

	mustNotMatch := []string{
		`http.Header{headerRetryAfter: []string{"5"}}`,
		`retryAfter := parseRetryAfter(resp.Header.Get(headerRetryAfter), time.Now())`,
		`// l'en-tête Retry-After vaut 5 secondes`,
	}
	for _, src := range mustNotMatch {
		if reRetryAfterLiteral.MatchString(src) {
			t.Errorf("FAUX POSITIF sur usage légitime : %q", src)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	defPath := filepath.Join(filepath.Dir(thisFile), retryAfterLiteralDefFile)
	src, err := os.ReadFile(defPath)
	if err != nil {
		t.Fatalf("lecture du fichier de définition %s : %v", retryAfterLiteralDefFile, err)
	}
	if !strings.Contains(string(src), "headerRetryAfter = "+`"Retry-After"`) {
		t.Errorf("%s ne définit plus headerRetryAfter : son exemption dans "+
			"TestNoRetryAfterLiteralInHandlers est devenue un trou — déplacer l'exemption "+
			"vers le nouveau fichier de définition", retryAfterLiteralDefFile)
	}
}
