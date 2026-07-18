package handlers

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestPrestigeHandler_ActorGuard_DeniesForeignActor_AllActorRoutes vérifie la
// réconciliation acteur ↔ session sur TOUTES les routes prestige qui lisent un
// acteur (user_id / created_by / requested_by) depuis le body ou la query — pas
// seulement les routes squad top-level historiques. ownershipMW garde le segment
// {player_slug} de l'URL, pas le payload : sans cette garde, un utilisateur A
// passe la garde d'URL avec SON slug et cible B via le body (BOLA horizontal).
//
// La garde n'autorise ici que « alice » ; chaque requête prétend agir « bob ».
func TestPrestigeHandler_ActorGuard_DeniesForeignActor_AllActorRoutes(t *testing.T) {
	onlyAlice := func(_ context.Context, slug string) bool { return slug == "alice" }
	cases := []struct {
		name, method, path, body string
	}{
		// prestige.go
		{"create_challenge", http.MethodPost, "/prestige/challenges", `{"user_id":"bob","title_slug":"halo_infinite","metric":"FieldKDA","target":1.5,"window_type":"session","cadence":"weekly","eval_type":"threshold","mode":"libre"}`},
		{"list_active_challenges", http.MethodGet, "/prestige/challenges?user_id=bob&title_slug=halo_infinite", ""},
		{"get_my_prestige", http.MethodGet, "/prestige/me?user_id=bob&title_slug=halo_infinite", ""},
		{"suggest_templates", http.MethodGet, "/templates/suggest?user_id=bob&title_slug=halo_infinite", ""},
		// prestige_arcs.go
		{"create_arc", http.MethodPost, "/arcs", `{"user_id":"bob","title_slug":"halo_infinite","title":"Ascension"}`},
		{"delete_arc", http.MethodDelete, "/arcs/arc1?user_id=bob&objectives=delete", ""},
		{"list_arcs", http.MethodGet, "/arcs?user_id=bob&title_slug=halo_infinite", ""},
		{"list_arc_presets", http.MethodGet, "/arcs/presets?user_id=bob&title_slug=halo_infinite", ""},
		{"adopt_preset_arc", http.MethodPost, "/arcs/presets/p1/adopt", `{"user_id":"bob","title_slug":"halo_infinite"}`},
		// prestige_squad_challenges.go
		{"create_squad_challenge", http.MethodPost, "/squads/sq1/challenges", `{"title_slug":"halo_infinite","mode":"collective","eval_type":"threshold","window_type":"session","target_per_member":5,"created_by":"bob"}`},
		{"join_squad_challenge", http.MethodPost, "/squad-challenges/sc1/join", `{"user_id":"bob","chosen_tier":"heroic"}`},
		// prestige_squads.go
		{"enable_pilot_mode", http.MethodPost, "/pilot-mode/enable", `{"user_id":"bob","title_slug":"halo_infinite"}`},
		{"disable_pilot_mode", http.MethodPost, "/pilot-mode/disable", `{"user_id":"bob","title_slug":"halo_infinite"}`},
		{"refresh_squad_pool", http.MethodPost, "/squads/sq1/challenges/pool/refresh", `{"title_slug":"halo_infinite","requested_by":"bob"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRouterGuarded(&mockPrestigeService{}, onlyAlice)
			var req *http.Request
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("status=%d, want 403 (acteur étranger refusé); body=%s", w.Code, w.Body.String())
			}
		})
	}
}

// TestPrestigeHandler_ActorGuard_AllowsOwnActor_AllActorRoutes : l'acteur légitime
// (« alice », profil possédé) passe la garde et la route s'exécute normalement.
// Un test vert par famille de routes gardées (challenges, arcs, me/templates,
// squad-challenges, pilot-mode, pool).
func TestPrestigeHandler_ActorGuard_AllowsOwnActor_AllActorRoutes(t *testing.T) {
	onlyAlice := func(_ context.Context, slug string) bool { return slug == "alice" }
	cases := []struct {
		name, method, path, body string
		wantCode                 int
	}{
		{"create_challenge", http.MethodPost, "/prestige/challenges", `{"user_id":"alice","title_slug":"halo_infinite","metric":"FieldKDA","target":1.5,"window_type":"session","cadence":"weekly","eval_type":"threshold","mode":"libre"}`, http.StatusCreated},
		{"list_active_challenges", http.MethodGet, "/prestige/challenges?user_id=alice&title_slug=halo_infinite", "", http.StatusOK},
		{"get_my_prestige", http.MethodGet, "/prestige/me?user_id=alice&title_slug=halo_infinite", "", http.StatusOK},
		{"suggest_templates", http.MethodGet, "/templates/suggest?user_id=alice&title_slug=halo_infinite", "", http.StatusOK},
		{"create_arc", http.MethodPost, "/arcs", `{"user_id":"alice","title_slug":"halo_infinite","title":"Ascension"}`, http.StatusCreated},
		{"delete_arc", http.MethodDelete, "/arcs/arc1?user_id=alice&objectives=delete", "", http.StatusNoContent},
		{"list_arcs", http.MethodGet, "/arcs?user_id=alice&title_slug=halo_infinite", "", http.StatusOK},
		{"list_arc_presets", http.MethodGet, "/arcs/presets?user_id=alice&title_slug=halo_infinite", "", http.StatusOK},
		{"adopt_preset_arc", http.MethodPost, "/arcs/presets/p1/adopt", `{"user_id":"alice","title_slug":"halo_infinite"}`, http.StatusCreated},
		{"create_squad_challenge", http.MethodPost, "/squads/sq1/challenges", `{"title_slug":"halo_infinite","mode":"collective","eval_type":"threshold","window_type":"session","target_per_member":5,"created_by":"alice"}`, http.StatusCreated},
		{"join_squad_challenge", http.MethodPost, "/squad-challenges/sc1/join", `{"user_id":"alice","chosen_tier":"heroic"}`, http.StatusNoContent},
		{"enable_pilot_mode", http.MethodPost, "/pilot-mode/enable", `{"user_id":"alice","title_slug":"halo_infinite"}`, http.StatusOK},
		{"disable_pilot_mode", http.MethodPost, "/pilot-mode/disable", `{"user_id":"alice","title_slug":"halo_infinite"}`, http.StatusNoContent},
		{"refresh_squad_pool", http.MethodPost, "/squads/sq1/challenges/pool/refresh", `{"title_slug":"halo_infinite","requested_by":"alice"}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRouterGuarded(&mockPrestigeService{}, onlyAlice)
			var req *http.Request
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("status=%d, want %d (acteur légitime autorisé); body=%s", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

// actorFieldNames : champs du payload (body/query) qui désignent l'acteur au nom
// duquel la requête agit. Toute lecture d'un de ces champs sur `body`/`in` doit
// s'accompagner d'un appel à authorizeActor dans le même handler.
var actorFieldNames = map[string]bool{"UserID": true, "CreatedBy": true, "RequestedBy": true}

// TestPrestigeHandler_ActorGuard_StructuralCoverage est le garde-rail structurel
// (CLAUDE.md règle 6) : par analyse AST, toute méthode de *PrestigeHandler qui lit
// un champ acteur (body.UserID / in.CreatedBy / ...) DOIT appeler h.authorizeActor.
// Un futur handler qui relit l'acteur sans réconciliation fait échouer ce test —
// impossible de réintroduire le BOLA horizontal par construction.
func TestPrestigeHandler_ActorGuard_StructuralCoverage(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "prestige") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		file, perr := parser.ParseFile(fset, name, nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue // fonctions libres et méthodes sans corps ignorées
			}
			if !isPrestigeHandlerMethod(fn) {
				continue
			}
			if !funcReadsActorField(fn) {
				continue
			}
			if !funcCallsAuthorizeActor(fn) {
				t.Errorf("%s: la méthode %s lit un champ acteur (body/in.{user_id,created_by,requested_by}) "+
					"sans appeler h.authorizeActor — BOLA horizontal potentiel (réconcilier l'acteur avec la session)",
					name, fn.Name.Name)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier prestige*.go scanné — le garde-rail ne couvre rien")
	}
}

// isPrestigeHandlerMethod : la méthode a un receiver *PrestigeHandler.
func isPrestigeHandlerMethod(fn *ast.FuncDecl) bool {
	if len(fn.Recv.List) != 1 {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "PrestigeHandler"
}

// funcReadsActorField : le corps lit body.<Actor> ou in.<Actor> (les deux seuls
// porteurs de payload par convention des handlers — évite les faux positifs sur
// les champs d'objets métier comme sq.CreatedBy).
func funcReadsActorField(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		base, ok := sel.X.(*ast.Ident)
		if !ok || (base.Name != "body" && base.Name != "in") {
			return true
		}
		if actorFieldNames[sel.Sel.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// funcCallsAuthorizeActor : le corps appelle h.authorizeActor(...).
func funcCallsAuthorizeActor(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "authorizeActor" {
			found = true
			return false
		}
		return true
	})
	return found
}
