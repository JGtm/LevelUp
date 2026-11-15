package archlint

// halo_exchange_failure_code_unique_test.go — garde-rail « la cause d'un échec de l'échange Halo se
// tranche en UN endroit de internal/api ».
//
// POURQUOI. Trois chemins de connexion (code d'appareil, site hébergé, application de bureau)
// choisissent entre « les services Halo ne répondent pas » (à réessayer) et « le compte est refusé ».
// Chacun avec sa propre copie de la distinction a déjà divergé : le code d'appareil séparait les deux
// causes, les deux chemins du navigateur envoyaient un refus ET une panne sur le même code, donc le
// même conseil de réessayer.
//
// RÈGLE, sur le code non-test de internal/api : `HaloServicesUnreachable(` n'est appelée que dans la
// fonction haloExchangeFailureCode (handlers/auth_halo_exchange_code.go). Les tests de comportement des
// trois chemins prouvent le résultat ; ce garde-rail empêche une quatrième copie de la distinction.

import (
	"go/ast"
	"sort"
	"strings"
	"testing"
)

// fonctionChoixCodeEchecHalo : la seule fonction de internal/api qui appelle HaloServicesUnreachable.
const fonctionChoixCodeEchecHalo = "haloExchangeFailureCode"

// appelsHaloUnreachableHorsHelper rend, pour un fichier, les appels à HaloServicesUnreachable (qualifiés
// ou non) situés hors de la fonction haloExchangeFailureCode.
func appelsHaloUnreachableHorsHelper(f goSource) []string {
	var violations []string
	for _, decl := range f.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name.Name == fonctionChoixCodeEchecHalo {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if nomAppele(call.Fun) == "HaloServicesUnreachable" {
				violations = append(violations, f.Rel+" : HaloServicesUnreachable appelée dans "+fn.Name.Name+
					" (hors "+fonctionChoixCodeEchecHalo+")")
			}
			return true
		})
	}
	return violations
}

func TestHaloExchangeFailureCode_UnSeulEndroitDansInternalAPI(t *testing.T) {
	var violations []string
	appelsDuHelper := 0
	files := goSourcesUnder(t, "internal/api")
	for _, f := range files {
		violations = append(violations, appelsHaloUnreachableHorsHelper(f)...)
		if strings.HasSuffix(f.Rel, "handlers/auth_halo_exchange_code.go") {
			appelsDuHelper++
		}
	}
	if appelsDuHelper == 0 {
		t.Fatalf("handlers/auth_halo_exchange_code.go introuvable sous internal/api : le garde-rail ne garde plus le helper %s",
			fonctionChoixCodeEchecHalo)
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("la cause d'un échec de l'échange Halo se tranche dans %s seulement :\n  %s",
			fonctionChoixCodeEchecHalo, strings.Join(violations, "\n  "))
	}
}

// Auto-preuve : le helper passe, une copie ailleurs (qualifiée ou non, dans une méthode ou une
// fonction) est refusée.
func TestHaloExchangeFailureCode_AutoPreuve(t *testing.T) {
	ok := parseGoSnippet(t, "internal/api/handlers/ok.go", `package handlers
func haloExchangeFailureCode(err error, refused string) string {
	if auth.HaloServicesUnreachable(err) { return "x" }
	return refused
}`)
	if v := appelsHaloUnreachableHorsHelper(ok); len(v) != 0 {
		t.Fatalf("le helper ne doit pas être refusé : %v", v)
	}
	for nom, src := range map[string]string{
		"qualifié":     "package handlers\nfunc (h *H) poll(err error) bool { return auth_platform.HaloServicesUnreachable(err) }",
		"non qualifié": "package handlers\nfunc poll(err error) bool { return HaloServicesUnreachable(err) }",
	} {
		if v := appelsHaloUnreachableHorsHelper(parseGoSnippet(t, "internal/api/handlers/ko.go", src)); len(v) != 1 {
			t.Errorf("%s : une copie de la distinction doit être refusée (violations : %v)", nom, v)
		}
	}
}
