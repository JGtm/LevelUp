// Package notifications — routes_guard_test.go : garde-rail (ratchet) du format des
// TargetRoute joueur.
//
// Interdit tout littéral de route joueur ("/players/… ou "/t/…) construit à la main
// dans une valeur TargetRoute, hors du helper canonique PlayerTargetRoute (routes.go).
// Sans ce ratchet les ~11 émetteurs title-scopés re-divergent (leçon CLAUDE.md n°6 :
// le prédicat bot est passé de 8 à 36 copies après une centralisation sans garde-rail).
//
// Portée / limites (best-effort assumé, cf. no_art_patterns_test.go) :
//   - Scan LIGNE À LIGNE après strip des lignes de commentaire. Un TargetRoute dont la
//     valeur littérale serait éclatée sur plusieurs lignes échapperait — aucun émetteur
//     ne le fait, la revue de code complète le filet.
//   - Les routes AGNOSTIQUES (/help, /help/changelog, /changelog, /settings) ne portent
//     ni "/players/" ni "/t/" : elles ne sont PAS attrapées, donc PAS allowlistées (une
//     entrée d'allowlist morte est elle-même un anti-pattern — cf. TestAllowlist… des
//     autres ratchets). Elles restent titre-indépendantes par conception.
//   - Le helper routes.go porte les littéraux "/t/" et "/players/" mais dans une ligne
//     `return …` (aucun token TargetRoute) et dans des commentaires (strippés) : il n'est
//     donc jamais attrapé, sans allowlist.
package notifications

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// targetRouteLiteralAllowlist : fichiers (chemin relatif depuis internal/) où un
// littéral de route joueur sur une ligne TargetRoute est TOLÉRÉ, avec justification
// datée. VIDE par conception : après la migration lot A (2026-07-23) tout émetteur
// title-scopé passe par PlayerTargetRoute. N'ajouter une entrée qu'avec une raison
// datée prouvant qu'un littéral direct est inévitable.
var targetRouteLiteralAllowlist = map[string]string{}

// targetRouteLineHasRawPlayerLiteral détecte une valeur TargetRoute construite avec un
// littéral de route joueur brut (ancien format /players/… ou nouveau format /t/… posé
// à la main au lieu de passer par PlayerTargetRoute).
//
// Un commentaire de fin de ligne est d'abord retiré : un champ documenté par un exemple
// (`TargetRoute *string // ex: "/players/…"`) ou une ligne de code commentée n'est pas
// une construction de route. Limite (best-effort) : un commentaire bloc /* … */ sur une
// seule ligne, ou un « // » à l'intérieur d'une string, ne sont pas gérés — aucun
// émetteur n'écrit ainsi ; la revue complète le filet.
func targetRouteLineHasRawPlayerLiteral(line string) bool {
	if idx := strings.Index(line, "//"); idx >= 0 {
		line = line[:idx]
	}
	if !strings.Contains(line, "TargetRoute") {
		return false
	}
	return strings.Contains(line, "/players/") || strings.Contains(line, "/t/")
}

// TestNoRawPlayerRouteLiteralsInTargetRoute — garde-rail principal. Balaye internal/
// (hors _test.go) et échoue si une valeur TargetRoute embarque un littéral de route
// joueur hors du helper canonique.
func TestNoRawPlayerRouteLiteralsInTargetRoute(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// internal/notifications/routes_guard_test.go → internal/
	internalRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if _, allowed := targetRouteLiteralAllowlist[rel]; allowed {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // commentaire : mention tolérée
			}
			if targetRouteLineHasRawPlayerLiteral(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("littéral de route joueur dans un TargetRoute interdit — passer par "+
			"notifications.PlayerTargetRoute(titleSlug, playerSlug, suffix) (zéro hop, "+
			"title-agnostic) ou allowlister avec justification datée :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestTargetRouteGuard_Sanity — le ratchet doit MORDRE sur les formes dangereuses et
// LAISSER PASSER le helper canonique + les routes agnostiques. Un garde-rail qui ne
// détecte jamais rien est inutile.
func TestTargetRouteGuard_Sanity(t *testing.T) {
	mustMatch := []string{
		`TargetRoute: fmt.Sprintf("/players/%s/media", slug),`, // ancien format inline
		`TargetRoute: "/players/" + playerSlug + "/home",`,     // ancien format concaténé
		`TargetRoute: "/t/halo_infinite/players/x/home",`,      // nouveau format posé en dur
	}
	for _, s := range mustMatch {
		if !targetRouteLineHasRawPlayerLiteral(s) {
			t.Errorf("le ratchet devrait matcher (littéral de route brut) : %q", s)
		}
	}
	mustNotMatch := []string{
		`TargetRoute: notifications.PlayerTargetRoute(o.TitleSlug, slug, "home"),`, // helper canonique
		`TargetRoute: "/help",`,                                              // route agnostique
		`TargetRoute: "/help/changelog",`,                                    // route agnostique
		`TargetRoute:  in.TargetRoute,`,                                      // pass-through (aucun littéral)
		`return "/t/" + titleSlug + "/players/" + playerSlug + "/" + suffix`, // corps du helper (pas de token TargetRoute)
		`TargetRoute  *string // ex: "/players/{slug}/match/{id}"`,           // champ documenté (littéral dans le commentaire strippé)
	}
	for _, s := range mustNotMatch {
		if targetRouteLineHasRawPlayerLiteral(s) {
			t.Errorf("le ratchet ne devrait PAS matcher : %q", s)
		}
	}
}
