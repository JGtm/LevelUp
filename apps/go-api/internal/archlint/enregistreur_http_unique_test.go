// enregistreur_http_unique_test.go — l'enregistreur de requêtes HTTP des tests est UN seul
// (`internal/haloapi/haloapitest.Enregistreur`).
//
// Contrat : un fichier `_test.go` du module `apps/go-api` qui déclare une méthode `RoundTrip`
// (un `http.RoundTripper` maison) doit être sous `internal/haloapi/haloapitest/` ou figurer dans
// [copiesPreexistantesDEnregistreur]. Un nouveau fichier est rouge : utiliser
// `haloapitest.Enregistreur`. Un fichier de la liste qui ne déclare plus de `RoundTrip` est rouge
// « retirer l'entrée » : la liste ne fait que descendre. Le parcours lit l'AST : un commentaire ou
// une chaîne qui cite `RoundTrip(` ne compte pas.
//
// Mutation qui doit le faire rougir : ajouter `func (x *y) RoundTrip(...)` à un `_test.go` hors liste.
package archlint

import (
	"go/ast"
	"sort"
	"strings"
	"testing"
)

// copiesPreexistantesDEnregistreur : les `_test.go` (relatifs à `apps/go-api`) qui déclaraient déjà
// un `RoundTrip` le 2026-11-01, avant l'enregistreur partagé. Aucune entrée n'est ajoutée.
var copiesPreexistantesDEnregistreur = map[string]bool{
	"internal/games/halo_infinite/haloadapter/aides_test.go":                    true,
	"internal/games/halo_infinite/haloadapter/client_url_dryrun_test.go":        true,
	"internal/games/halo_infinite/haloadapter/privacy_provider_failure_test.go": true,
	"internal/haloapi/xblauth/exchange_descriptor_test.go":                      true,
	"internal/haloapi/xblauth/exchange_test.go":                                 true,
	"internal/platform/auth/sisu_exchange_flow_test.go":                         true,
	"internal/presence/rest_client_test.go":                                     true,
	"internal/sync/filmfetch/aides_test.go":                                     true,
	"internal/sync/filmfetch/blob_reprise_test.go":                              true,
}

// dossierEnregistreurPartage : l'unique endroit où un `RoundTrip` de test est chez lui.
const dossierEnregistreurPartage = "internal/haloapi/haloapitest/"

// declareRoundTrip dit si `src` déclare une méthode nommée RoundTrip. Fonction pure.
func declareRoundTrip(src goSource) bool {
	for _, decl := range src.File.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil && fd.Name.Name == "RoundTrip" {
			return true
		}
	}
	return false
}

// ecartsEnregistreur rend, triés, les écarts entre les fichiers qui déclarent un RoundTrip
// (`declarent`, chemins relatifs au module) et la liste `permis`.
func ecartsEnregistreur(declarent []string, permis map[string]bool) []string {
	var out []string
	vus := map[string]bool{}
	for _, rel := range declarent {
		vus[rel] = true
		if !permis[rel] && !strings.HasPrefix(rel, dossierEnregistreurPartage) {
			out = append(out, rel+" : déclare un RoundTrip hors de l'allowlist — utiliser haloapitest.Enregistreur")
		}
	}
	for rel := range permis {
		if !vus[rel] {
			out = append(out, rel+" : ne déclare plus de RoundTrip — retirer l'entrée de l'allowlist")
		}
	}
	sort.Strings(out)
	return out
}

func TestEnregistreurHTTPUnique(t *testing.T) {
	var declarent []string
	for _, f := range workspaceGoSources(t, "apps/go-api") {
		if !strings.HasSuffix(f.Rel, "_test.go") {
			continue
		}
		src, err := parseFichierGo(f.Chemin)
		if err != nil {
			t.Fatalf("%s : %v", f.Rel, err)
		}
		if declareRoundTrip(src) {
			declarent = append(declarent, strings.TrimPrefix(f.Rel, "apps/go-api/"))
		}
	}
	if ecarts := ecartsEnregistreur(declarent, copiesPreexistantesDEnregistreur); len(ecarts) > 0 {
		t.Errorf("enregistreur de requêtes HTTP de test non unique (règle 6 de CLAUDE.md) :\n  %s", strings.Join(ecarts, "\n  "))
	}
}

func TestEnregistreurHTTPUnique_AutoPreuve(t *testing.T) {
	avec := parseGoSnippet(t, "internal/x/a_test.go", "package x\ntype e struct{}\nfunc (e *e) RoundTrip() {}\n")
	sans := parseGoSnippet(t, "internal/x/b_test.go", "package x\n// func (e *e) RoundTrip(\nfunc RoundTrip() {}\nvar s = \"RoundTrip(\"\n")
	if !declareRoundTrip(avec) || declareRoundTrip(sans) {
		t.Fatal("seule la déclaration d'une méthode RoundTrip compte (ni commentaire, ni chaîne, ni fonction libre)")
	}
	permis := map[string]bool{"internal/x/ancien_test.go": true}
	ecarts := strings.Join(ecartsEnregistreur(
		[]string{"internal/x/nouveau_test.go", "internal/haloapi/haloapitest/enregistreur_test.go"}, permis), "\n")
	if !strings.Contains(ecarts, "internal/x/nouveau_test.go : déclare un RoundTrip hors de l'allowlist — utiliser haloapitest.Enregistreur") ||
		!strings.Contains(ecarts, "internal/x/ancien_test.go : ne déclare plus de RoundTrip — retirer l'entrée") ||
		strings.Contains(ecarts, "haloapitest/enregistreur_test.go") {
		t.Errorf("écarts = %q", ecarts)
	}
	if got := ecartsEnregistreur([]string{"internal/x/ancien_test.go"}, permis); len(got) != 0 {
		t.Errorf("fichier de l'allowlist qui déclare encore un RoundTrip : aucun écart attendu, obtenu %v", got)
	}
}
