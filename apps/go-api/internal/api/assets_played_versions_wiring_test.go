package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// La porte des assets du serveur juge les versions de carte aussi par les matchs joués : la
// configuration des assets de server_apiv1.go renseigne PlayedMapVersions (sans quoi la
// version d'un vrai match reste refusée à un anonyme).
func TestAssetConfig_WiresPlayedMapVersions(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "server_apiv1.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	configs, wired := 0, 0
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "AssetConfig" {
			return true
		}
		configs++
		for _, elt := range lit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "PlayedMapVersions" {
					wired++
				}
			}
		}
		return true
	})
	if configs != 1 || wired != 1 {
		t.Fatalf("assets.AssetConfig : %d littéral(aux), %d avec PlayedMapVersions ; attendu 1 et 1", configs, wired)
	}
}
