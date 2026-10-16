// Package archlint — player_read_cache_invalidation_test.go : ratchet « points
// d'invalidation du cache des lectures joueur » (plan perf 2026-09-23, lot L5b).
//
// POURQUOI. Les lignes de filtres et l'historique canonique d'un joueur sont gardés
// en mémoire (platform/duckdb/player_read_cache.go, TTL 60 s). Leur fraîcheur repose
// sur des appels à duckdb.InvalidatePlayerReadCaches posés là où les lignes changent :
// retirer l'un d'eux ne casse aucun test fonctionnel — la page sert simplement des
// données périmées jusqu'au TTL. Ce test verrouille leur présence :
//
//   - sync.runPostSyncPipeline : fin du post-sync du joueur (chemins V1 et V2) ;
//   - sync.flushReadCaches : vidage à mi-pipeline, une fois les agrégats rafraîchis ;
//     runPostSyncPipeline l'appelle AVANT la source du kill (étape film lente), ce que
//     TestFlushReadCachesPrecedesKillSource tient ;
//   - sync.RecomputeIsWithFriends : recalcul « avec amis » hors sync (liste d'amis) ;
//   - duckdb.MatchExclusionRepo.SetExclusion : exclusion d'un match.
//
// Un point retiré à dessein se retire d'ici dans le même commit, avec sa raison.
// Invariant I3 de l'ADR 0036 (docs/adr/0036-page-reads-are-scoped.md).
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

// playerReadCacheInvalidationPoints — (fichier relatif à internal/, fonction ou
// méthode) qui doit appeler InvalidatePlayerReadCaches.
var playerReadCacheInvalidationPoints = []struct {
	file string
	fn   string
}{
	{"sync/engine_postsync.go", "runPostSyncPipeline"},
	{"sync/engine_postsync_scoring.go", "flushReadCaches"},
	{"sync/friends_recompute.go", "RecomputeIsWithFriends"},
	{"platform/duckdb/match_exclusion_repo.go", "SetExclusion"},
}

func TestPlayerReadCacheInvalidationPoints(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	for _, p := range playerReadCacheInvalidationPoints {
		path := filepath.Join(internalRoot, filepath.FromSlash(p.file))
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", p.file, err)
		}
		body := funcBody(file, p.fn)
		if body == nil {
			t.Errorf("%s : fonction %s introuvable (renommée ? mettre ce ratchet à jour)", p.file, p.fn)
			continue
		}
		if !callsFunc(body, "InvalidatePlayerReadCaches") {
			t.Errorf("%s : %s n'appelle plus InvalidatePlayerReadCaches — les lectures joueur "+
				"mises en cache resteraient périmées jusqu'au TTL", p.file, p.fn)
		}
	}
}

// funcBody rend le corps de la fonction ou méthode `name` du fichier, nil sinon.
func funcBody(file *ast.File, name string) *ast.BlockStmt {
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == name {
			return fd.Body
		}
	}
	return nil
}

// callsFunc dit si le bloc contient un appel (direct, qualifié ou différé) à `name`.
func callsFunc(body *ast.BlockStmt, name string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return !found
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			found = found || fun.Name == name
		case *ast.SelectorExpr:
			found = found || fun.Sel.Name == name
		}
		return !found
	})
	return found
}

// TestFlushReadCachesPrecedesKillSource : runPostSyncPipeline vide les caches de lecture du
// joueur (flushReadCaches) AVANT la source du kill (runKillSource), la première étape qui
// télécharge des films pendant des minutes. Sans cet ordre, une page lue pendant les films
// sert un « Solo » ou un match sans LUSR jusqu'au TTL, ce que le réordonnancement du post-sync
// doit éviter.
func TestFlushReadCachesPrecedesKillSource(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	path := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "sync", "engine_postsync.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("engine_postsync.go : %v", err)
	}
	body := funcBody(file, "runPostSyncPipeline")
	if body == nil {
		t.Fatal("runPostSyncPipeline introuvable (renommée ? mettre ce ratchet à jour)")
	}
	flush, kill := firstCallPos(body, "flushReadCaches"), firstCallPos(body, "runKillSource")
	if flush == token.NoPos || kill == token.NoPos {
		t.Fatalf("appels introuvables dans runPostSyncPipeline : flushReadCaches=%v runKillSource=%v", flush != token.NoPos, kill != token.NoPos)
	}
	if flush > kill {
		t.Error("runPostSyncPipeline appelle runKillSource AVANT flushReadCaches : les pages lues pendant " +
			"la lecture des films serviraient des lignes périmées")
	}
}

// firstCallPos rend la position du premier appel (direct ou qualifié) à `name`, NoPos sinon.
func firstCallPos(body *ast.BlockStmt, name string) token.Pos {
	pos := token.NoPos
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || pos != token.NoPos {
			return pos == token.NoPos
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if fun.Name == name {
				pos = call.Pos()
			}
		case *ast.SelectorExpr:
			if fun.Sel.Name == name {
				pos = call.Pos()
			}
		}
		return pos == token.NoPos
	})
	return pos
}
