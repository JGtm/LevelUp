// Package archlint — nginx_assets_v_test.go : garde-rail « le paramètre v des images
// d'assets est filtré à l'entrée » (plan sécurité du 2026-10-15, F-1a R1-2).
//
// POURQUOI. Le paramètre v de GET /api/v1/assets/maps/…/image entre dans un chemin disque
// côté Go. Le Go le refuse lui-même (forme UUID, confinement du stock), et nginx le refuse
// en amont : défense en profondeur, appliquée sur le VPS le 2026-10-16 avant la livraison
// du correctif Go. Ce test tient la copie versionnée de ce filtre.
//
// CE QUI EST EXIGÉ :
//   - packaging/nginx/snippets/levelup-assets-v.conf contient un `if ($arg_v ~* "…")
//     { return 400; }` dont l'expression reconnaît chaque forme de traversée (`..`, `/`,
//     barre oblique inverse, `%2f`, `%5c`, `%2e`, en toute casse : $arg_v n'est pas décodé
//     par nginx) et laisse passer un UUID ;
//   - dans chaque site versionné, chaque `server` 443 sert /api/v1/assets/ par une location
//     qui inclut ce fichier à son emplacement du VPS.
package archlint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// nginxAssetsVSnippet est le filtre versionné, relatif à packaging/nginx/.
const nginxAssetsVSnippet = "snippets/levelup-assets-v.conf"

// nginxAssetsVInclude est l'emplacement du filtre sur le VPS (directive include des sites).
const nginxAssetsVInclude = "/etc/nginx/snippets/levelup-assets-v.conf"

// nginxAssetsProbeURI est un chemin servi par la location des images d'assets.
const nginxAssetsProbeURI = "/api/v1/assets/maps/halo_infinite/m/image"

// nginxAssetsVTraps : valeurs de v que le filtre doit refuser ; nginxAssetsVLegit : valeur
// légitime qu'il doit laisser passer.
var (
	nginxAssetsVTraps = []string{"..", "a/b", `a\b`, "%2f", "%2F", "%5c", "%5C", "%2e%2e", "%2E", "..%2f"}
	nginxAssetsVLegit = "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
)

// nginxIfArgVRe lit `if ($arg_v ~* "EXPR") { return 400; }` dans le texte brut (le
// tokeniseur du garde-rail retire les barres obliques inverses des chaînes).
var nginxIfArgVRe = regexp.MustCompile(`if\s*\(\s*\$arg_v\s+~\*\s+"([^"]*)"\s*\)\s*\{\s*return\s+400\s*;\s*\}`)

func TestNginxAssetsVSnippetRefusesTraversal(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(nginxPackagingDir(t), filepath.FromSlash(nginxAssetsVSnippet)))
	if err != nil {
		t.Fatalf("lecture de %s : %v", nginxAssetsVSnippet, err)
	}
	for _, v := range nginxAssetsVSnippetViolations(string(data)) {
		t.Errorf("%s : %s", nginxAssetsVSnippet, v)
	}
}

func TestNginxAssetsLocationsIncludeVSnippet(t *testing.T) {
	dir := nginxPackagingDir(t)
	names := make([]string, 0, len(nginxConfFiles))
	for name := range nginxConfFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("lecture de %s : %v", name, err)
			continue
		}
		for _, v := range nginxAssetsIncludeViolations(string(data)) {
			t.Errorf("%s : %s", name, v)
		}
	}
}

// TestNginxAssetsVGuardDetectsDefects — l'analyse sait échouer sur chaque défaut visé.
func TestNginxAssetsVGuardDetectsDefects(t *testing.T) {
	snippet := `if ($arg_v ~* "(\.\.|/|\x5c|%2f|%5c|%2e)") { return 400; }`
	if got := nginxAssetsVSnippetViolations(snippet); len(got) != 0 {
		t.Fatalf("filtre sain signalé : %v", got)
	}
	snippetDefects := map[string]string{
		"filtre absent":         "# rien",
		"encodage %2f oublié":   strings.Replace(snippet, "|%2f", "", 1),
		"antislash oublié":      strings.Replace(snippet, `|\x5c`, "", 1),
		"point encodé oublié":   strings.Replace(snippet, "|%2e", "", 1),
		"sensible à la casse":   strings.Replace(snippet, "~*", "~", 1),
		"réponse 200":           strings.Replace(snippet, "return 400", "return 200", 1),
		"filtre trop large":     strings.Replace(snippet, `(\.\.|`, `(-|\.\.|`, 1),
		"expression non valide": strings.Replace(snippet, `(\.\.|`, `((\.\.|`, 1),
	}
	for name, src := range snippetDefects {
		if src == snippet {
			t.Fatalf("%s : la mutation n'a rien changé (harnais)", name)
		}
		if got := nginxAssetsVSnippetViolations(src); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}

	site := `server {
    listen 443 ssl;
    location /api/v1/assets/ { include ` + nginxAssetsVInclude + `; proxy_pass http://b; }
    location / { proxy_pass http://b; }
}`
	if got := nginxAssetsIncludeViolations(site); len(got) != 0 {
		t.Fatalf("site sain signalé : %v", got)
	}
	siteDefects := map[string]string{
		"include retiré":           strings.Replace(site, "include "+nginxAssetsVInclude+"; ", "", 1),
		"autre fichier inclus":     strings.Replace(site, nginxAssetsVInclude, "/etc/nginx/snippets/autre.conf", 1),
		"location des assets ôtée": strings.Replace(site, "location /api/v1/assets/ { include "+nginxAssetsVInclude+"; proxy_pass http://b; }\n", "", 1),
		"aucun server 443":         strings.Replace(site, "listen 443 ssl;", "listen 80;", 1),
	}
	for name, src := range siteDefects {
		if src == site {
			t.Fatalf("%s : la mutation n'a rien changé (harnais)", name)
		}
		if got := nginxAssetsIncludeViolations(src); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}
}

// nginxAssetsVSnippetViolations juge le filtre : forme, réponse 400, expression qui refuse
// chaque traversée en toute casse et laisse passer un UUID.
func nginxAssetsVSnippetViolations(src string) []string {
	m := nginxIfArgVRe.FindStringSubmatch(src)
	if m == nil {
		return []string{`aucun « if ($arg_v ~* "…") { return 400; } » (filtre insensible à la casse, réponse 400)`}
	}
	re, err := regexp.Compile("(?i)" + m[1])
	if err != nil {
		return []string{fmt.Sprintf("expression %q illisible : %v", m[1], err)}
	}
	var out []string
	for _, v := range nginxAssetsVTraps {
		if !re.MatchString(v) {
			out = append(out, fmt.Sprintf("v=%q n'est pas refusé par %q", v, m[1]))
		}
	}
	if re.MatchString(nginxAssetsVLegit) {
		out = append(out, fmt.Sprintf("v=%q (UUID légitime) refusé par %q", nginxAssetsVLegit, m[1]))
	}
	return out
}

// nginxAssetsIncludeViolations : chaque server 443 sert les images d'assets par une
// location qui inclut le filtre.
func nginxAssetsIncludeViolations(src string) []string {
	root, err := parseNginxConf(src)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	tls := 0
	for _, srv := range root.all("server") {
		if !srv.listensTLS() {
			continue
		}
		tls++
		loc := srv.selectLocation(nginxAssetsProbeURI)
		if loc == nil {
			out = append(out, fmt.Sprintf("server (ligne %d) : aucune location ne sert %s", srv.line, nginxAssetsProbeURI))
			continue
		}
		if !nginxIncludes(loc, nginxAssetsVInclude) {
			out = append(out, fmt.Sprintf("location %q (ligne %d) : `include %s;` absent (filtre du paramètre v)",
				loc.matcher(), loc.line, nginxAssetsVInclude))
		}
	}
	if tls == 0 {
		out = append(out, "aucun bloc server n'écoute en 443")
	}
	return out
}

// nginxIncludes dit si loc inclut path.
func nginxIncludes(loc *ngxNode, path string) bool {
	for _, d := range loc.all("include") {
		if d.arg(0) == path {
			return true
		}
	}
	return false
}
