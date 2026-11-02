package archlint

// token_cipher_single_decision_test.go — garde-rail du chiffreur des fichiers de jetons
// (ADR 0045 D-11, lot L13b).
//
// POURQUOI. Le fichier `watcher_tokens/{xuid}.json` est scellé ou non selon le PROFIL de déploiement
// (capacité TokenVault), jamais selon le système d'exploitation seul, et le serveur comme chaque
// outil en ligne de commande doivent choisir de la même façon : un chargeur de configuration qui
// pose le chiffreur du processus, des magasins qui le lisent à leur construction. Un second endroit
// qui choisit (un outil qui construit son propre DPAPI, un `if runtime.GOOS == "windows"`) ferait
// diverger deux écrivains du même fichier.
//
// DEUX RÈGLES, sur le code non-test d'apps/go-api.
//
//  1. `tokencipher.InstallForProfile` n'est appelée que par internal/config/config_token_vault.go ;
//     `tokencipher.InstallProcess` (chiffreur explicite) n'est jamais appelée en production.
//  2. `tokencipher.NewDPAPI` et `tokencipher.ForVault` ne sont appelées que dans le paquet
//     tokencipher lui-même (les tests le construisent directement, ils sont hors périmètre).

import (
	"go/ast"
	"sort"
	"strings"
	"testing"
)

// fichierChoixChiffreur : le seul appelant de tokencipher.InstallForProfile.
const fichierChoixChiffreur = "internal/config/config_token_vault.go"

func TestTokenCipher_UneSeuleDecision(t *testing.T) {
	var violations []string
	scanned := 0
	for _, f := range goSourcesUnder(t, "internal", "cmd") {
		if strings.HasPrefix(f.Rel, "internal/platform/auth/tokencipher/") {
			continue
		}
		scanned++
		ast.Inspect(f.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "tokencipher" {
				return true
			}
			switch sel.Sel.Name {
			case "InstallForProfile":
				if f.Rel != fichierChoixChiffreur {
					violations = append(violations, f.Rel+" : tokencipher.InstallForProfile hors "+fichierChoixChiffreur)
				}
			case "InstallProcess", "NewDPAPI", "ForVault":
				violations = append(violations, f.Rel+" : tokencipher."+sel.Sel.Name+" en production")
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("aucun fichier scanné : le garde-rail ne garde rien")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("le chiffreur des jetons se choisit en UN endroit (ADR 0045 D-11) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
