// Package archlint — garde-fous d'architecture vérifiés en test (ratchet).
//
// no_bare_resolve_mode_ui_test.go : interdit les NOUVEAUX appels nus à
// analysis.ResolveModeUI (dérivation de mode pair-only). Depuis la
// centralisation de la convention « mode = pair, sinon game_variant »
// (analysis.ResolveModeUIWithVariant, fix filtres H5 2026-07-22), un caller qui
// dérive un libellé de mode UI doit passer par le helper WithVariant — un appel
// pair-only rate les titres sans pair_name (Halo 5) et re-crée exactement le
// bug « catégorie Modes vide » (leçon CLAUDE.md n°6 : une factorisation sans
// garde-rail re-diverge).
//
// Les sites historiques restants sont grandfathered dans l'allowlist ci-dessous
// avec leur justification ; toute NOUVELLE occurrence hors allowlist échoue.
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

// bareResolveModeUIAllowlist : fichiers (chemin relatif depuis internal/) où un
// appel nu à analysis.ResolveModeUI reste TOLÉRÉ. Grandfathered le 2026-07-22
// (fix filtres H5) : ces sites implémentent déjà leur propre fallback variant
// (deux appels successifs) ou sont légitimement pair-only. Retrait par site à
// mesure de leur migration vers ResolveModeUIWithVariant.
var bareResolveModeUIAllowlist = map[string]bool{
	"sync/recent_matches.go":                true, // :153/:157 deux-étapes pair puis variant (candidat migration)
	"service/session_page_service.go":       true, // :431-443 deux-étapes avec logique EN/FR propre (candidat migration)
	"service/match_view_builders_header.go": true, // :111-118 pair-only assumé — ModeNameFR pré-résolu par le repo en amont
	"platform/duckdb/explorer_repo.go":      true, // :497/:524 résolution repo-side des noms de pair (pas un mode UI de filtre)
}

// bareResolveModeUIRE matche un appel qualifié analysis.ResolveModeUI( — la
// forme WithVariant ne matche pas (frontière : parenthèse ouvrante immédiate).
var bareResolveModeUIRE = regexp.MustCompile(`analysis\.ResolveModeUI\(`)

func TestNoNewBareResolveModeUI(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if bareResolveModeUIAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if bareResolveModeUIRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+" → "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("appel nu à analysis.ResolveModeUI hors allowlist (%d) — utiliser "+
			"analysis.ResolveModeUIWithVariant (convention mode = pair sinon game_variant, "+
			"sinon la catégorie Modes redevient vide sur les titres sans pair type Halo 5) :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// ---------------------------------------------------------------------------
// TestNoNewModePlaylistLabelLiteral — garde-rail complémentaire (lot M5 L3,
// 2026-09-08, .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §2.B/§4).
//
// rankedplaylists.go portait NameEN/NameFR comme des CHAMPS de struct littéral
// assignés en dur (`NameEN: "Ranked Arena", NameFR: "Arène classée"`) — exactement
// l'anti-pattern « map Go de libellés de mode/playlist » que le plan interdit
// (CLAUDE.md « Multi-titre » : TOML config/titles/{slug}/mappings/, jamais un
// libellé FR/EN en dur côté Go). Migré vers ranked_playlists_labels.toml (même
// loader/validation que assets.toml) ; NameEN()/NameFR() sont maintenant des
// MÉTHODES qui lisent ce TOML — un futur retour en arrière (nouveau champ
// littéral, ou une nouvelle map[string]string clé→nom FR/EN sur ce même modèle)
// recréerait le problème. Volontairement étroit (un seul motif regex, comme les
// autres ratchets du dossier) : il attrape le motif exact déjà vu deux fois dans
// ce fichier avant migration, pas toute forme possible de libellé en dur (cf.
// no_french_label_literal_test.go pour la mesure large par littéral accentué).
var modePlaylistLabelFieldRE = regexp.MustCompile(`\b(NameEN|NameFR)\s*:\s*"`)

// modePlaylistLabelFieldAllowlist : fichiers où NameEN:/NameFR: reste toléré comme
// champ littéral. Grandfathered au 2026-09-08 — toute NOUVELLE occurrence échoue.
var modePlaylistLabelFieldAllowlist = map[string]bool{
	// Tier CSR (Bronze..Onyx), PAS un mode/playlist — famille distincte, déjà
	// identifiée en double/triple ailleurs (compare_service.go::csrRankLabel,
	// home_canonical_skill.go::csrTierENtoFR, sync/csr_writes.go::tierENtoFR ;
	// cf. .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §11, découverte M5 L5). Pas
	// touché par ce ratchet (portée L3 = modes/playlists), ni par ce lot.
	"analysis/skill_v2/tier.go": true,
}

func TestNoNewModePlaylistLabelLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "/migrations/") || modePlaylistLabelFieldAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if modePlaylistLabelFieldRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+" → "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("nouveau champ NameEN:/NameFR: assigné à un littéral Go (%d) — un nom de "+
			"mode/playlist se déclare dans un TOML par-titre (config/titles/{slug}/mappings/, "+
			"ou si aucun chemin de DI n'existe pour l'appelant, un TOML embarqué documenté au "+
			"même titre que ranked_playlists_labels.toml), jamais un littéral Go (lot M5 L3, "+
			"2026-09-08) :\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}
