// Package archlint — no_halowaypoint_literal_test.go : ratchet F (VF-7).
//
// Interdit toute NOUVELLE occurrence du littéral `halowaypoint` (hôte des API
// officielles Halo) dans un fichier .go non-test hors de l'allowlist ci-dessous.
//
// Contexte : le gate F du plan d'audit promettait « grep halowaypoint hors
// games/halo_infinite + platform/halo → 0 (allowlist temporaire datée) » mais
// aucun garde-rail n'a jamais été créé, et K3e a depuis déplacé le client HTTP
// dans internal/sync/haloclient/. Sans ratchet, l'invariant « les URLs Halo en
// dur vivent dans une poignée de fichiers frontière » re-divergera silencieusement
// (leçon CLAUDE.md règle 6 : factorisation/frontière sans garde-rail).
//
// But du test : NON de mettre l'allowlist à 0 aujourd'hui (les URLs Halo sont
// une dépendance réelle de la frontière réseau/adapters), mais de FIGER la liste
// des fichiers frontière autorisés au 2026-07-07 (décroissante) pour que toute
// NOUVELLE URL en dur hors de ces fichiers soit rouge.
//
// Le scan couvre TOUTES les lignes (y compris commentaires) : un commentaire
// documentant une URL halowaypoint est aussi un endroit où une dépendance en
// dur peut se figer/pourrir — on gèle donc l'état complet.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// halowaypointAllowlist — DÉCROISSANTE, figée au 2026-07-07. Chaque entrée est un
// fichier frontière qui a une raison légitime de nommer l'hôte halowaypoint.
// Catégories :
//   - haloclient/       : client HTTP officiel Halo Infinite (déplacé par K3e).
//   - platform/halo/     : providers HTTP (discovery, saisons, privacy, leaderboard).
//   - platform/auth/     : échange de tokens Xbox↔Spartan (audiences XSTS/Spartan).
//   - domain/title/      : defaults documentés des descripteurs d'auth par titre.
//   - assets/            : fetchers gamecms-hacs (URLs d'assets médailles/images).
//   - games/halo_infinite: adapter d'URLs d'assets waypoint (base URL waypoint web).
//   - games/halo_5/       : client H5 (mêmes hôtes officiels).
//   - halotest/          : fake server de test (héberge des URLs factices halowaypoint).
//   - cmd/                : outils diag/probe/fixtures qui tapent l'API officielle.
//   - scripts/            : outil de warm-up d'assets (URLs gamecms).
//
// Toute suppression d'une URL en dur d'un de ces fichiers RETIRE son entrée.
var halowaypointAllowlist = map[string]bool{
	// haloclient/ — client HTTP officiel (endpoints matchs, skill, film, career).
	"internal/sync/haloclient/halo_client.go":                true,
	"internal/sync/haloclient/halo_client_career.go":         true,
	"internal/sync/haloclient/halo_client_film.go":           true,
	"internal/sync/haloclient/halo_skill.go":                 true,
	"internal/sync/haloclient/spartan_nameplate_resolver.go": true,
	// platform/halo/ — providers HTTP.
	"internal/platform/halo/discovery_client.go":    true,
	"internal/platform/halo/leaderboard_scraper.go": true,
	"internal/platform/halo/privacy_provider.go":    true,
	"internal/platform/halo/provider.go":            true,
	"internal/platform/halo/season_provider.go":     true,
	// platform/auth/ — échange tokens Xbox↔Spartan (audiences/hôtes officiels).
	"internal/platform/auth/halo_exchange.go": true,
	"internal/platform/auth/provider.go":      true,
	// domain/title/ — defaults documentés des descripteurs d'auth.
	"internal/domain/title/auth_descriptor.go": true,
	// assets/ — fetchers gamecms-hacs (URLs d'assets).
	"internal/assets/fetcher_gamecms.go": true,
	"internal/assets/kinds.go":           true,
	// games/ — adapters/clients par titre.
	"internal/games/halo_infinite/adapter_asset_urls.go": true,
	"internal/games/halo_5/client.go":                    true,
	// halotest/ — fake server (URLs factices halowaypoint en test-support).
	"internal/sync/halotest/fake_server.go": true,
	// cmd/ — outils diag/probe/fixtures.
	"cmd/diag_appearance/main.go":             true,
	"cmd/diag_emblem_colors/main.go":          true,
	"cmd/diag_emblem_mapping/main.go":         true,
	"cmd/gen_test_fixtures/main.go":           true,
	"cmd/populate-career-rank-images/main.go": true,
	"cmd/probe-h5/main.go":                    true,
	"cmd/probe-mcc/main.go":                   true,
	"cmd/refresh_golden_fixture/main.go":      true,
	"cmd/snapshot-world-leaderboard/main.go":  true,
	// scripts/ — warm-up d'assets (URLs gamecms).
	"scripts/warm_bp_assets/main.go": true,
}

var halowaypointRE = regexp.MustCompile(`halowaypoint`)

func TestNoNewHalowaypointLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	for _, sub := range []string{"internal", "cmd", "scripts"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if halowaypointAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				if halowaypointRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral 'halowaypoint' en dur interdit hors des fichiers frontière (F/VF-7) — "+
			"une URL Halo en dur doit vivre dans un fichier frontière (haloclient/platform/auth/adapters) "+
			"déjà allowlisté, ou l'ajouter à halowaypointAllowlist avec justification datée :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestHalowaypointAllowlistEntriesPointToExistingFiles (self-check, leçon V4d/VF-6) —
// chaque clé de halowaypointAllowlist est un CHEMIN de fichier et doit désigner un
// fichier EXISTANT contenant réellement le littéral. Une entrée dont le fichier a
// disparu (déplacé/supprimé par un refactor) est un trou latent : un fichier recréé
// plus tard à ce chemin pourrait nommer halowaypoint sans déclencher le ratchet.
func TestHalowaypointAllowlistEntriesPointToExistingFiles(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	for rel := range halowaypointAllowlist {
		path := filepath.Join(goAPIRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("halowaypointAllowlist : entrée %q pointe un fichier inexistant (%v) — "+
				"refactor de renommage/suppression ? Retirer l'entrée morte (trou latent : un "+
				"fichier recréé à ce chemin échapperait au ratchet).", rel, err)
			continue
		}
		if !halowaypointRE.Match(data) {
			t.Errorf("halowaypointAllowlist : entrée %q ne contient plus le littéral 'halowaypoint' — "+
				"l'URL en dur a été retirée ? Retirer l'entrée devenue inutile (allowlist décroissante).", rel)
		}
	}
}
