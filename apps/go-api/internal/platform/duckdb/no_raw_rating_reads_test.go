package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestNoRawAppendOnlyReads — garde-rail Lot B (ADR 0026 / règle ART n°2).
//
// Les tables append-only (match_skill_rank, match_csrs, player_csr_snapshots,
// pve_match_stats) portent N versions par clé (id + written_at). Une lecture
// APPLICATIVE de la table BRUTE sert des lignes périmées de façon non
// déterministe. Toute lecture passe donc par la vue `<table>_latest`.
//
// Périmètre : couches de LECTURE uniquement (platform/duckdb, api, service,
// analysis). Les writers (internal/sync, internal/persist), les migrations
// (internal/games/*/migrations, internal/migration, schema.go) et les CLI de
// diagnostic (cmd/) lisent légitimement le brut → hors scan.
//
// Ajout d'une lecture brute = migrer vers `_latest`, ou (rare) l'allowlister
// ci-dessous avec une justification datée. L'allowlist est décroissante.
func TestNoRawAppendOnlyReads(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// thisFile = <goapi>/internal/platform/duckdb/no_raw_rating_reads_test.go
	goAPIRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	scanDirs := []string{
		filepath.Join(goAPIRoot, "internal", "platform", "duckdb"),
		filepath.Join(goAPIRoot, "internal", "api"),
		filepath.Join(goAPIRoot, "internal", "service"),
		filepath.Join(goAPIRoot, "internal", "analysis"),
	}

	// `FROM|JOIN <table>` : le groupe 2 capture un éventuel suffixe `_latest`.
	// Une occurrence dont le groupe 2 est vide = lecture BRUTE.
	rawRe := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(match_skill_rank|match_csrs|player_csr_snapshots|pve_match_stats)(_latest)?\b`)

	// Allowlist datée (2026-07-02) — lectures brutes VOLONTAIRES et documentées.
	allow := map[string]string{
		"queries_home_citations.go":    "Q26g : filtre H5 placeholder CSR=0 appliqué AVANT le choix de ligne (la vue a déjà tranché CSR>LUSR) + tie-break written_at déterministe ; Q26f effective_type (classification CSR/LUSR — Découverte §7)",
		"queries_career.go":            "Q8LUSRHistoryPlayer : graphe d'évolution veut TOUS les checkpoints (raw volontaire, documenté)",
		"queries_career_encounters.go": "Q24LUSRHistory (enemy strength) : valeur LUSR-spécifique ; _latest (CSR>LUSR) la changerait — Découverte §7",
		"queries_squad.go":             "MAX(expected_win_prob) WHERE IS NOT NULL : stale-safe (colonne écrite seulement sur LUSR) ; _latest perdrait le winProb sur ranked",
		"career_repo_csr_seasons.go":   "SELECT DISTINCT season_id : stale-safe (season_id invariant entre versions)",
	}

	var offenders []string
	for _, dir := range scanDirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			hasRaw := false
			for _, m := range rawRe.FindAllSubmatch(src, -1) {
				if len(m[2]) == 0 { // pas de suffixe `_latest` → lecture brute
					hasRaw = true
					break
				}
			}
			if !hasRaw {
				return nil
			}
			if _, allowed := allow[filepath.Base(path)]; allowed {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			offenders = append(offenders, rel)
			return nil
		})
	}

	if len(offenders) > 0 {
		t.Errorf("lecture(s) brute(s) d'une table append-only hors vue _latest (ADR 0026) — "+
			"migrer vers `<table>_latest` ou allowlister avec justification datée :\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
