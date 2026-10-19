// Package archlint — pool_no_shared_handle_test.go : ratchet « le pool joueur ne tient ni ne
// rouvre aucun handle sur la base partagée » (D-15 du plan de release v8, 2026-10-19).
//
// POURQUOI. En mode B-swap le provider d'une base partagée en est l'UNIQUE détenteur (ADR 0016) :
// il ferme son handle lecture seule, puis l'ouvre en écriture. OpenReadOnly partage un handle
// ref-compté par chemin, si bien qu'une seule référence tenue ailleurs survit à cette fermeture et
// fait refuser l'ouverture en écriture (« different configuration »), à chaque tentative. Le pool
// rouvrait `pdb.Shared` sur la base de CHAQUE PlayerDB, tous titres confondus, à chaque retour en
// lecture du provider du titre par défaut : la base Halo 5 restait tenue et plus aucune écriture
// n'y passait jusqu'au redémarrage du serveur.
//
// RÈGLE. Dans le code non-test du paquet `internal/platform/duckdb` (le pool ; le provider vit
// dans le sous-paquet sharedprovider, hors de la règle) :
//   - aucune affectation `x.Shared = …` : `PlayerDB.Shared` n'est posé qu'à la construction ;
//   - aucune ouverture (`Open*`) sur le chemin de la base partagée (`SharedDBPath` /
//     `sharedDBPath`) hors de `openPlayerDB`, qui ne l'ouvre qu'en mode legacy (sans provider).
//
// Allowlist vide.
package archlint

import (
	"go/ast"
	"strings"
	"testing"
)

// poolSharedHandleDefects rend les lignes fautives d'un fichier du pool.
func poolSharedHandleDefects(src goSource) []int {
	var lines []int
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Shared" {
					lines = append(lines, src.Fset.Position(sel.Pos()).Line)
				}
			}
		case *ast.FuncDecl:
			if x.Name.Name == "openPlayerDB" {
				return false // seul constructeur autorisé (ouverture legacy)
			}
		case *ast.CallExpr:
			if isOpenCall(x.Fun) && mentionsSharedPath(x.Args) {
				lines = append(lines, src.Fset.Position(x.Pos()).Line)
			}
		}
		return true
	})
	return lines
}

func isOpenCall(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.Ident:
		return strings.HasPrefix(f.Name, "Open")
	case *ast.SelectorExpr:
		return strings.HasPrefix(f.Sel.Name, "Open")
	}
	return false
}

func mentionsSharedPath(args []ast.Expr) bool {
	found := false
	for _, a := range args {
		ast.Inspect(a, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr:
				if v.Sel.Name == "SharedDBPath" || v.Sel.Name == "sharedDBPath" {
					found = true
				}
			case *ast.Ident:
				if v.Name == "sharedDBPath" {
					found = true
				}
			}
			return !found
		})
	}
	return found
}

func TestPoolNeTientAucunHandleSurLaBasePartagee(t *testing.T) {
	for _, s := range goSourcesUnder(t, "internal/platform/duckdb") {
		if strings.Count(strings.TrimPrefix(s.Rel, "internal/platform/duckdb/"), "/") > 0 {
			continue // sous-paquets (sharedprovider : le détenteur légitime)
		}
		for _, line := range poolSharedHandleDefects(s) {
			t.Errorf("%s:%d : le pool tient ou rouvre un handle sur la base partagée — en mode B-swap "+
				"son provider en est l'unique détenteur (ADR 0016) ; lire par PlayerDB.SharedReadDB()", s.Rel, line)
		}
	}
}

func TestPoolSharedHandleGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "x.go", `package duckdb
func (pdb *PlayerDB) RestoreSharedAfterSwap() error {
	newShared, err := OpenReadOnly(pdb.sharedDBPath, pdb.userTimezone)
	pdb.Shared = newShared
	_, _, _ = OpenReadForQuery(cfg.SharedDBPath)
	return err
}`)
	if got := poolSharedHandleDefects(bad); len(got) != 3 {
		t.Fatalf("défauts vus : %v (3 attendus)", got)
	}
	good := parseGoSnippet(t, "y.go", `package duckdb
func openPlayerDB(cfg PlayerPoolConfig) *PlayerDB {
	sharedDB, _ := OpenReadOnly(cfg.SharedDBPath, cfg.UserTimezone)
	return &PlayerDB{Shared: sharedDB, sharedDBPath: cfg.SharedDBPath}
}
func (pdb *PlayerDB) SharedDBPath() string { return pdb.sharedDBPath }
func other() { meta, _ := OpenReadWriteShared(cfg.MetaDBPath); pdb.SharedSocial = meta }`)
	if got := poolSharedHandleDefects(good); len(got) != 0 {
		t.Fatalf("faux positifs : %v", got)
	}
}
