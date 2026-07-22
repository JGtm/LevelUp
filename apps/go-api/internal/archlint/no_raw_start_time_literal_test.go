// Package archlint — no_raw_start_time_literal_test.go : ratchet H1 (règle
// CLAUDE.md n°8, timezone canonique).
//
// Interdit toute NOUVELLE occurrence du littéral SQL brut du timestamp de début
// de match canonique — `COALESCE(<alias>.start_time_utc, <alias>.start_time AT
// TIME ZONE 'UTC')`. La source unique est analysis.SQLStartTimeCanonical(alias)
// (délégué local platform/duckdb.StartTimeCanonicalSQL). Toute divergence de
// cette expression a causé des décalages de fuseau (DETTE first_joined_time).
//
// Le motif est PRÉCIS : il n'attrape ni `real_start_time AT TIME ZONE 'UTC'`
// (colonne distincte, calcul d'epoch/durée), ni la forme diagnostique d'offset
// `epoch_ms(r.start_time AT TIME ZONE 'UTC') - epoch_ms(r.start_time_utc)`
// (cmd/backfill_first_joined_tz, mesure volontairement le décalage), ni la
// définition elle-même (`"COALESCE(" + prefix + "start_time_utc, "...` — coupée
// par la concaténation). L'allowlist est donc VIDE.
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

// rawStartTimeAllowlist : VIDE. Les 97 sites du pattern canonique ont été migrés
// vers le helper (H1, 2026-07-04). migrations/ est sauté (historique gelé : la
// forme y est un CASE `THEN start_time AT TIME ZONE 'UTC'`, non-COALESCE, donc
// non-matchée de toute façon). Toute nouvelle occurrence fait échouer le test.
var rawStartTimeAllowlist = map[string]bool{}

// rawStartTimeRE matche la forme canonique COALESCE avec start_time_utc ET
// start_time. `[a-z_]*\.?` couvre un préfixe d'alias optionnel (r., mr., reg., nu).
var rawStartTimeRE = regexp.MustCompile(`COALESCE\([a-z_]*\.?start_time_utc,\s*[a-z_]*\.?start_time AT TIME ZONE 'UTC'\)`)

// rawOrderByStartTimeRE (V7b, VF-13) : interdit un ORDER BY sur `start_time`
// BRUT table-qualifié (`ORDER BY mr.start_time`, `ORDER BY r.start_time` …). Une
// fenêtre triée sur le start_time brut mal-classe les imports OpenSpartan à
// start_time NULL/TZ-décalé (bug VF-13 sur Q29HistoryForAvg). Le tri doit passer
// par le fragment canonique StartTimeCanonicalSQL(alias).
//
// FORME TABLE-QUALIFIÉE UNIQUEMENT (`\w+\.start_time`) : un ORDER BY sur une
// COLONNE NUE `start_time` est LÉGITIME quand c'est l'ALIAS d'une projection issue
// du fragment canonique (ex. queries_match.go:400 `ORDER BY start_time DESC` après
// `StartTimeCanonicalSQL("r") AS start_time`). Distinguer les deux par regex seul
// est fragile ; on restreint donc le ratchet à la forme préfixée par une table
// (toujours brute), au prix de laisser passer un éventuel ORDER BY sur alias nu
// vraiment brut — cas rare, couvert par la revue. `[^_]` en fin évite start_time_utc.
var rawOrderByStartTimeRE = regexp.MustCompile(`ORDER BY\s+\w+\.start_time([^_]|$)`)

// rawCastStartTimeDateRE (F2, 2026-07-22) : interdit un bucket jour `CAST(<alias>.
// start_time AS DATE)` sur le start_time BRUT. Un tel CAST découpe les jours dans
// le fuseau naïf de la colonne (décalage aux frontières de jour) au lieu du fragment
// canonique UTC — bug consigné puis corrigé sur loadPlayerStats.accuracy_threshold_days.
// Le bucket jour doit s'écrire CAST(SQLStartTimeCanonical(alias) AS DATE).
//
// Précis : `([a-z]+\.)?start_time AS DATE` exige que `start_time` soit suivi
// IMMÉDIATEMENT de ` AS DATE`. La forme canonique `CAST(COALESCE(mr.start_time_utc,
// mr.start_time AT TIME ZONE 'UTC') AS DATE)` n'est PAS matchée (start_time y est
// suivi de `_utc` ou de ` AT TIME ZONE`). `([a-z]+\.)?` (préfixe alias `word.`, pas
// d'underscore) évite d'attraper la colonne distincte `real_start_time`. Allowlist VIDE.
var rawCastStartTimeDateRE = regexp.MustCompile(`CAST\(([a-z]+\.)?start_time AS DATE\)`)

// rawCastStartTimeDateAllowlist : VIDE. Unique site (accuracy_threshold_days) migré
// vers le fragment canonique le 2026-07-22 (F2). Toute nouvelle occurrence échoue.
var rawCastStartTimeDateAllowlist = map[string]bool{}

// rawOrderByStartTimeAllowlist : dette PRÉEXISTANTE gelée (règle CLAUDE.md n°5 —
// baseline), keyée PAR FICHIER (comme H1 ci-dessus : robuste au décalage de
// lignes). 20 fichiers au 2026-07-07 : outils cmd/ diag/backfill/seed + requêtes
// internes non-lecture-chaude déjà sur start_time brut avant V7b. Le ratchet
// interdit toute NOUVELLE occurrence dans un fichier HORS liste (dont
// queries_match.go — la lecture chaude Q29 corrigée par V7b, volontairement
// ABSENTE : une régression y refait échouer le test). Ne jamais ajouter un fichier
// sans avoir d'abord migré ses sites vers StartTimeCanonicalSQL.
var rawOrderByStartTimeAllowlist = map[string]bool{
	"internal/api/wire/post_sync_progression_queries.go":              true,
	"internal/ops/seed_demo.go":                                       true,
	"internal/ops/seed_demo_corpus.go":                                true,
	"internal/platform/duckdb/campaign_repo.go":                       true,
	"internal/platform/duckdb/engagement_score_repo_queries.go":       true,
	"internal/platform/duckdb/prestige/prestige_baseline_provider.go": true,
	"internal/sync/backfill.go":                                       true,
	"internal/sync/csr_shared_backfill.go":                            true,
	"internal/sync/engagement.go":                                     true,
	"internal/sync/performance_helpers.go":                            true,
	"internal/sync/session_recalc.go":                                 true,
	"cmd/analyze_media_tz/main.go":                                    true,
	"cmd/audit_coverage/main.go":                                      true,
	"cmd/backfill_all/main.go":                                        true,
	"cmd/backfill_participation_info/main.go":                         true,
	"cmd/backfill_quit_timestamps/main.go":                            true,
	"cmd/diag_citation_counters/main.go":                              true,
	"cmd/diag_lusr_player/loaders.go":                                 true,
	"cmd/diag_orphan_session/main.go":                                 true,
	"cmd/levelup/cmd_backfill.go":                                     true,
	"cmd/lusr_v2_phase0/replay.go":                                    true,
	"cmd/seed-medal/main.go":                                          true,
}

func TestNoNewRawStartTimeLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// thisFile = .../internal/archlint/x_test.go → goAPIRoot = .../ (deux niveaux au-dessus d'internal).
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	var orderByViolations []string
	var castDateViolations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// migrations/ = DDL gelée (ADR 0026) : ne pas soumettre au ratchet.
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			// La définition canonique du fragment (source unique) est exemptée du
			// ratchet COALESCE ; le ratchet ORDER BY a son propre allowlist fichier.
			coalesceExempt := rel == "internal/analysis/sql_fragments.go" || rawStartTimeAllowlist[rel]
			orderByExempt := rawOrderByStartTimeAllowlist[rel]
			castDateExempt := rawCastStartTimeDateAllowlist[rel]
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if !coalesceExempt && rawStartTimeRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
				if !orderByExempt && rawOrderByStartTimeRE.MatchString(line) {
					orderByViolations = append(orderByViolations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
				if !castDateExempt && rawCastStartTimeDateRE.MatchString(line) {
					castDateViolations = append(castDateViolations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral SQL brut du start_time canonique interdit (H1, règle CLAUDE.md n°8) — "+
			"utiliser analysis.SQLStartTimeCanonical(alias) (ou le délégué local "+
			"duckdb.StartTimeCanonicalSQL) :\n  %s",
			strings.Join(violations, "\n  "))
	}
	if len(orderByViolations) > 0 {
		t.Errorf("ORDER BY sur start_time BRUT table-qualifié interdit (V7b/VF-13, règle "+
			"CLAUDE.md n°8) — trier via StartTimeCanonicalSQL(alias) (fragment canonique) "+
			"sinon les imports OpenSpartan NULL/TZ-décalés sortent mal triés de la fenêtre :\n  %s",
			strings.Join(orderByViolations, "\n  "))
	}
	if len(castDateViolations) > 0 {
		t.Errorf("CAST(start_time AS DATE) sur start_time BRUT interdit (F2, règle CLAUDE.md "+
			"n°8) — bucketer le jour via CAST(SQLStartTimeCanonical(alias) AS DATE) sinon les "+
			"jours se décalent aux frontières de fuseau :\n  %s",
			strings.Join(castDateViolations, "\n  "))
	}
}
