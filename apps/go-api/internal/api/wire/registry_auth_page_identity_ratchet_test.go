package wire

// registry_auth_page_identity_ratchet_test.go — ratchet A2 (revue 2026-07).
//
// Invariant : tout constructeur de contexte du package wire qui appelle
// r.enrichWithHaloTokens DOIT aussi appliquer forcePageIdentityXUID dans la MÊME
// fonction — le SUJET des fetches live (identité Spartan, BP/défis) doit venir du
// joueur de la PAGE (pdb.XUID), jamais du compte connecté porté par la session.
//
// Cette classe de bug a récidivé 4 fois (PR #63) : à chaque nouvel appelant qui
// oubliait le forçage, les données du compte connecté étaient persistées sous le
// xuid de la page (season pass = 4e occurrence, finding n°1 de la revue). Le ratchet
// ferme la classe : un futur appelant sans forçage échoue au test, sauf exemption
// DATÉE et justifiée (chemin à xuid explicite, cf. forcePageIdentityExemptions).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// forcePageIdentityExemptions : fonctions appelant enrichWithHaloTokens qui n'ont
// PAS besoin de forcePageIdentityXUID, avec justification datée. Motif d'exemption
// UNIQUE toléré : le service construit cible un xuid EXPLICITE (jamais dérivé de
// ctxkeys.HaloXUID) et ne persiste rien sous un xuid de page → le sujet ne peut pas
// fuiter. Toute autre raison = corriger le code, pas allonger cette liste.
var forcePageIdentityExemptions = map[string]string{
	// Compare : CompareService.FetchRemoteStats cible players/xuid(pdb) explicitement
	// (remoteStats caché, xuid passé au constructeur NewCompareService) ; aucune
	// identité dérivée de ctxkeys.HaloXUID, aucune écriture sous xuid de page.
	// Exempté 2026-07-17 (revue A2).
	"Compare": "FetchRemoteStats cible xuid(pdb) explicite ; pas de sujet ambiant (2026-07-17)",
	// ExplorerCtxWithAuth : RETIRÉ 2026-07-25 (A1). Le player-query Explorer est
	// passé de enrichWithHaloTokens à enrichWithHaloTokensPublicRead (résolution
	// pool-first pour les lectures publiques de tiers) → il n'appelle plus
	// enrichWithHaloTokens, donc n'entre plus dans le périmètre de ce ratchet.
	// Son périmètre est désormais gardé par TestPublicReadPerimeter_Guard
	// (registry_pool_source_test.go). Les providers ciblent toujours un xuid
	// explicite (FetchLiveIdentity(ctx, targetXUID)) — aucune fuite du compte connecté.
}

func TestEnrichCallersForcePageIdentity(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	pkgDir := filepath.Dir(thisFile) // .../internal/api/wire

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("ReadDir wire: %v", err)
	}
	fset := token.NewFileSet()

	// funcName -> fichier, pour les fonctions qui appellent enrichWithHaloTokens
	// SANS forcePageIdentityXUID dans la même fonction.
	enrichNoForce := map[string]string{}
	seenEnrichCaller := false

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(pkgDir, name), nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		for _, decl := range f.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Body == nil {
				continue
			}
			callsEnrich, callsForce := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.SelectorExpr:
					if node.Sel.Name == "enrichWithHaloTokens" {
						callsEnrich = true
					}
				case *ast.Ident:
					if node.Name == "forcePageIdentityXUID" {
						callsForce = true
					}
				}
				return true
			})
			if !callsEnrich {
				continue
			}
			seenEnrichCaller = true
			if !callsForce {
				enrichNoForce[fn.Name.Name] = name
			}
		}
	}

	if !seenEnrichCaller {
		t.Fatal("aucun appelant de enrichWithHaloTokens détecté — le ratchet ne protège plus rien " +
			"(fonction renommée/déplacée ? mettre à jour ce test)")
	}

	// Offenders : enrich sans force ET non exemptés.
	var offenders []string
	for fnName, file := range enrichNoForce {
		if _, exempt := forcePageIdentityExemptions[fnName]; !exempt {
			offenders = append(offenders, fnName+" ("+file+")")
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Errorf("constructeur(s) appelant enrichWithHaloTokens sans forcePageIdentityXUID "+
			"(sujet = joueur de la PAGE, anti-pollution PR #63) — appliquer forcePageIdentityXUID "+
			"dans la fonction, ou ajouter une exemption DATÉE dans forcePageIdentityExemptions :\n  %s",
			strings.Join(offenders, "\n  "))
	}

	// Exemptions périmées : une exemption qui ne correspond plus à un appelant
	// enrich-sans-force (fonction supprimée, ou qui applique désormais le forçage)
	// doit être retirée — sinon l'allowlist ment sur l'état réel du code.
	var stale []string
	for fnName := range forcePageIdentityExemptions {
		if _, ok := enrichNoForce[fnName]; !ok {
			stale = append(stale, fnName)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("exemption(s) périmée(s) dans forcePageIdentityExemptions (plus d'appel "+
			"enrichWithHaloTokens sans forçage sous ce nom) — retirer l'entrée :\n  %s",
			strings.Join(stale, "\n  "))
	}
}
