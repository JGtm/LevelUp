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
// Allowlist DATÉE de deux entrées (ci-dessous) : deux lectures venues de api/wire, qui vivaient hors du
// périmètre balayé et que le déplacement du 2026-11-12 (4.4) a fait entrer dans le paquet.
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

// poolSharedHandleAllowlist : fichier du paquet -> nombre d'ouvertures de la base partagée tolérées, avec la
// date et le RISQUE. Les deux entrées ne sont pas des fautes du pool (le pool ne tient rien) : ce sont des
// lectures d'administration qui ouvrent la base partagée d'un titre PAR SON CHEMIN, comme elles le faisaient dans
// api/wire avant 4.4, hors du périmètre de ce garde. Une entrée ne s'ajoute pas : elle se supprime avec le code
// qui la justifie, et une entrée sans ouverture fait échouer le test.
var poolSharedHandleAllowlist = map[string]struct {
	n      int
	reason string
}{
	"internal/platform/duckdb/title_bases.go": {1, "2026-11-12 : TitleBases.ReadShared, déplacée à l'identique de api/wire/registry_data_quality.go (4.4) — " +
		"pour un titre dont le provider ne tient pas le fichier, OpenReadOnly ouvre une référence en LECTURE tenue le temps de la " +
		"tâche d'administration ; si ce fichier devenait celui d'un provider B-swap, cette référence ferait refuser sa bascule en " +
		"écriture (« different configuration », ADR 0016). Le titre tenu par le provider passe, lui, par AcquireSharedRead."},
	"internal/platform/duckdb/last_match_reader.go": {1, "2026-11-12 : LastMatchReader.LastMatchByXUID, déplacée à l'identique de " +
		"api/wire/registry_monitoring_freshness.go (4.4) — OpenReadForQuery emprunte le handle en cache (RO ou RW) et ne ferme que " +
		"ce qu'il ouvre ; seul risque : une référence en lecture tenue le temps d'une requête groupée, pendant une bascule du provider."},
}

func TestPoolNeTientAucunHandleSurLaBasePartagee(t *testing.T) {
	vus := map[string]int{}
	for _, s := range goSourcesUnder(t, "internal/platform/duckdb") {
		if strings.Count(strings.TrimPrefix(s.Rel, "internal/platform/duckdb/"), "/") > 0 {
			continue // sous-paquets (sharedprovider : le détenteur légitime)
		}
		lines := poolSharedHandleDefects(s)
		vus[s.Rel] = len(lines)
		allowed := poolSharedHandleAllowlist[s.Rel].n
		if len(lines) <= allowed {
			continue
		}
		for _, line := range lines {
			t.Errorf("%s:%d : le pool tient ou rouvre un handle sur la base partagée — en mode B-swap "+
				"son provider en est l'unique détenteur (ADR 0016) ; lire par PlayerDB.SharedReadDB()", s.Rel, line)
		}
	}
	for rel, e := range poolSharedHandleAllowlist {
		if n, ok := vus[rel]; !ok || n != e.n {
			t.Errorf("%s : allowlist de %d ouverture(s), %d vue(s) — ajuster ou retirer l'entrée de poolSharedHandleAllowlist", rel, e.n, n)
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
