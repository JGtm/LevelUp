package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"slices"
	"testing"
)

// TestNewResourcesClientNeSuitQueLesHotesDePublication : une redirection vers un hôte hors liste, vers une adresse
// non https, ou au-delà de la borne est refusée ; une redirection vers un hôte de publication passe.
func TestNewResourcesClientNeSuitQueLesHotesDePublication(t *testing.T) {
	check := newResourcesClient().CheckRedirect
	if check == nil {
		t.Fatal("le client des téléchargements doit contrôler ses redirections")
	}
	redirect := func(raw string, hops int) error {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return check(&http.Request{URL: u}, make([]*http.Request, hops))
	}
	cases := []struct {
		name    string
		url     string
		hops    int
		wantErr bool
	}{
		{"hôte de publication en https", "https://objects.githubusercontent.com/pack.zip", 1, false},
		{"hôte hors liste", "https://evil.example/pack.zip", 1, true},
		{"hôte de publication en http clair", "http://github.com/pack.zip", 1, true},
		{"trop de redirections", "https://github.com/pack.zip", resourcesMaxRedirects, true},
	}
	for _, c := range cases {
		if err := redirect(c.url, c.hops); (err != nil) != c.wantErr {
			t.Errorf("%s : erreur %v, refus attendu %v", c.name, err, c.wantErr)
		}
	}
}

// TestBootConfigureAppelleLEtapeDuPaquetEntreLeVerrouEtLOutillage : l'étape du paquet de ressources s'exécute après
// la prise du verrou d'instance (aucun autre processus ne tient les fichiers du paquet) et avant l'outillage et le
// registre des titres, qui lisent config/ et le paquet. Retirer l'appel ou le déplacer fait échouer ce test : aucun
// autre test n'exerce bootConfigure, qui ouvre la configuration, les journaux et la porte du processus.
func TestBootConfigureAppelleLEtapeDuPaquetEntreLeVerrouEtLOutillage(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "boot_config.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "bootConfigure" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok {
					calls = append(calls, id.Name)
				}
			}
			return true
		})
	}
	lock := slices.Index(calls, "refuseIfInstanceLockFailed")
	pack := slices.Index(calls, "bootResourcePack")
	tooling := slices.Index(calls, "initToolingAndRegistry")
	if lock < 0 || pack < 0 || tooling < 0 {
		t.Fatalf("bootConfigure doit appeler le verrou, l'étape du paquet et l'outillage : %v", calls)
	}
	if !(lock < pack && pack < tooling) {
		t.Fatalf("ordre de bootConfigure : verrou %d, paquet %d, outillage %d (%v)", lock, pack, tooling, calls)
	}
}
