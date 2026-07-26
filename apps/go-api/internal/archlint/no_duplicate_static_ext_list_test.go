// Package archlint — no_duplicate_static_ext_list_test.go : garde-rail « une
// seule liste d'extensions statiques » (CLAUDE.md règle 6).
//
// La source unique est `middleware.IsStaticAssetPath` (map staticAssetExts dans
// internal/api/middleware/static_assets.go). Elle est consommée par le rate
// limiter (exemption du bucket httprate), par serveStaticFile (Cache-Control) et
// par mountSPA (404 franc). Une 2e liste littérale ailleurs re-diverge
// silencieusement : c'est exactement ce qui s'était produit entre le fix 404 des
// assets et l'exemption rate-limit (l'un connaissait les extensions, l'autre non
// → les fichiers de public/ consommaient le bucket, 429 muets en prod).
//
// Règle : aucun fichier .go (hors la définition canonique) ne doit contenir 4
// extensions statiques littérales distinctes ou plus DONT au moins une extension
// propre au web (`.css`, `.js`, `.mjs`, `.map`, `.woff`, `.woff2`, `.ttf`) — la
// signature d'un catalogue « fichiers servis par le front ». Les listes purement
// images (`.png/.jpg/.jpeg/.gif` des vignettes média, `.webp` de discovery)
// répondent à un autre besoin (formats acceptés d'un pipeline média), ne
// divergent pas de la liste du front et restent donc autorisées.
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

// staticExtCanonicalFile : LA définition (seule autorisée à porter la liste).
const staticExtCanonicalFile = "internal/api/middleware/static_assets.go"

// staticExtLiteralRE matche une extension statique littérale entre guillemets
// (`".png"`, `".woff2"`…). Les extensions médias non servies par le front
// (.mp4, .mkv) sont hors périmètre.
var staticExtLiteralRE = regexp.MustCompile(
	`"\.(png|jpe?g|webp|gif|svg|ico|css|js|mjs|map|woff2?|ttf|txt|xml|json)"`)

// staticExtListThreshold : nombre d'extensions distinctes au-delà duquel un
// fichier est considéré comme portant un catalogue dupliqué.
const staticExtListThreshold = 4

// webOnlyExts : extensions qui n'apparaissent QUE dans un catalogue de fichiers
// servis par le front (jamais dans une liste de formats média). Leur présence
// distingue une vraie duplication d'une liste d'images légitime.
var webOnlyExts = map[string]bool{
	`".css"`: true, `".js"`: true, `".mjs"`: true, `".map"`: true,
	`".woff"`: true, `".woff2"`: true, `".ttf"`: true,
}

func TestNoDuplicateStaticAssetExtList(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == staticExtCanonicalFile {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			found := distinctStaticExts(string(data))
			if len(found) >= staticExtListThreshold && hasWebOnlyExt(found) {
				violations = append(violations,
					rel+"  ("+strconv.Itoa(len(found))+" extensions : "+strings.Join(found, " ")+")")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("liste d'extensions statiques dupliquée (règle 6) — consommer "+
			"middleware.IsStaticAssetPath au lieu de redéclarer la liste :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestStaticExtDetector_FlagsCatalogue vérifie que le ratchet n'est pas vide de
// sens : un catalogue front recopié DOIT être détecté, une liste d'images ne
// doit PAS l'être.
func TestStaticExtDetector_FlagsCatalogue(t *testing.T) {
	catalogue := "var exts = map[string]struct{}{\n\t\".png\": {}, \".svg\": {}, \".css\": {}, \".js\": {},\n}\n"
	found := distinctStaticExts(catalogue)
	if len(found) < staticExtListThreshold || !hasWebOnlyExt(found) {
		t.Errorf("catalogue front non détecté : %v", found)
	}

	imagesOnly := "if ext == \".png\" || ext == \".jpg\" || ext == \".jpeg\" || ext == \".gif\" {}\n"
	found = distinctStaticExts(imagesOnly)
	if hasWebOnlyExt(found) {
		t.Errorf("liste d'images signalée à tort comme catalogue front : %v", found)
	}

	comment := "// \".png\" \".css\" \".js\" \".svg\" dans un commentaire\n"
	if got := distinctStaticExts(comment); len(got) != 0 {
		t.Errorf("les commentaires doivent être ignorés, trouvé %v", got)
	}
}

// hasWebOnlyExt indique si la liste contient au moins une extension propre au
// web (signature d'un catalogue front, cf. webOnlyExts).
func hasWebOnlyExt(exts []string) bool {
	for _, e := range exts {
		if webOnlyExts[e] {
			return true
		}
	}
	return false
}

// distinctStaticExts retourne les extensions statiques littérales distinctes
// trouvées dans le source, hors lignes de commentaire.
func distinctStaticExts(src string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		for _, m := range staticExtLiteralRE.FindAllString(line, -1) {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out
}
