package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// modeNameTrCanonicalFile — SEUL fichier de la couche autorisé à lire
// metadata.mode_name_tr en SQL.
const modeNameTrCanonicalFile = "mode_name_tr.go"

// modeNameTrReadRE matche toute LECTURE SQL de la table (`FROM mode_name_tr`,
// `JOIN mode_name_tr`), quelle que soit la mise en forme du littéral (une ligne
// ou SQL indenté multi-lignes : dans les deux formes rencontrées, le `FROM` et le
// nom de table restent sur la même ligne). Les ÉCRITURES (`INSERT INTO
// mode_name_tr`, DDL des migrations) ne sont PAS visées : elles vivent
// légitimement dans internal/persist et internal/games/*/migrations.
var modeNameTrReadRE = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+mode_name_tr\b`)

// TestNoModeNameTrLiteralOutsideCanonicalFile — garde-rail (règle CLAUDE.md n°6 :
// « ≤ 2 copies d'un même pattern ; à la 3e, centraliser ET poser un garde-rail —
// une factorisation sans garde-rail re-diverge »).
//
// Constat du 2026-07-25 : la requête de résolution FR des modes existait en SIX
// exemplaires dans cette couche (career_repo_highlights, explorer_repo,
// match_history_fr_translations, squad_repo_mapstats au littéral près ;
// home_repo_translations et media_repo_translations en variante de mise en
// forme), chacun avec sa propre gestion du vide, de la table absente et du handle
// FATAL-invalidated. Tous convergent désormais vers mode_name_tr.go
// (queryModeNameTrFR / loadModeNamesFRForKeys / loadKnownModesEN).
//
// Périmètre : internal/platform/duckdb (sous-paquets inclus) — la seule couche
// autorisée à parler SQL aux repos de lecture (internal/service ne peut pas
// importer duckdb, cf. no_duckdb_import_test.go). Les lectures de mode_name_tr
// des couches ops/ et cmd/ (inventaire de couverture, diagnostics) sont des
// requêtes DIFFÉRENTES dans une autre couche : hors périmètre, donc pas
// allowlistées non plus.
//
// ALLOWLIST VIDE, et elle doit le rester : ajouter une lecture = appeler le
// helper canonique, ou l'étendre.
func TestNoModeNameTrLiteralOutsideCanonicalFile(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	layerRoot := filepath.Dir(thisFile)

	var offenders []string
	err := filepath.WalkDir(layerRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".go") || strings.HasSuffix(base, "_test.go") {
			return nil
		}
		if base == modeNameTrCanonicalFile {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			// Les commentaires qui RENVOIENT au helper canonique restent permis.
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if modeNameTrReadRE.MatchString(line) {
				rel, _ := filepath.Rel(layerRoot, path)
				offenders = append(offenders,
					filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", layerRoot, err)
	}

	if len(offenders) > 0 {
		t.Errorf("lecture SQL de mode_name_tr hors du fichier canonique %s — "+
			"utiliser queryModeNameTrFR / loadModeNamesFRForKeys / loadKnownModesEN "+
			"(règle CLAUDE.md n°6) :\n  %s",
			modeNameTrCanonicalFile, strings.Join(offenders, "\n  "))
	}
}
