// Package sync — no_art_patterns_test.go : Phase 6 du plan d'éradication
// ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Guard-rail anti-régression** : ce test scanne les fichiers Go du
// projet pour détecter l'apparition de patterns SQL à risque ART
// (ON CONFLICT DO UPDATE, INSERT OR REPLACE) sur les tables migrées en
// append-only (tablesProtegees). Si une nouvelle occurrence apparaît hors
// allowlist, le test fail — sauf si l'auteur l'ajoute explicitement à
// l'allowlist (avec justification).
//
// **Portée — ce que ce scan NE couvre PAS** (pour ne pas donner de fausse
// assurance, cf. revue D4-2) :
//   - `DELETE FROM` et `UPDATE … SET` nus ne sont PAS dans patternsAtRisk
//     (trop de faux positifs file-level sur les UPDATE bitmask sérialisés
//     légitimes). La forme bulk `UPDATE … FROM (VALUES …)` — le vrai
//     déclencheur ART qui touche N entrées d'index en 1 statement — est,
//     elle, gardée par TestNoBulkMultiRowUpdateOnCriticalTables (ci-dessous).
//   - Les tables de "match-of-record" (match_registry, match_participants,
//     medals_earned, killer_victim_pairs) ne sont PAS dans
//     tablesProtegees : elles ne sont pas append-only et leurs UPDATE bitmask /
//     row-by-row sérialisés par dblease sont sûrs. Elles sont
//     protégées AUTREMENT : INSERT-only par construction via le package persist
//     (chemin batch par défaut) + combat_write_guard_test.go (killer_victim +
//     highlight_events) + le tripwire bulk-UPDATE ci-dessous.
//
// Ces tests tournent en CI normale (pas de build tag) : rapides et
// déterministes (grep sur les fichiers source).

package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// patternsAtRisk : motifs SQL connus comme déclencheurs du bug ART.
// Toute occurrence dans le hot path (non-test, non-migration) est suspect.
var patternsAtRisk = []*regexp.Regexp{
	// LIMITATION CONNUE (revue 2026-05-30) : `[^)]*` s'arrête au premier `)`, donc
	// la forme `ON CONFLICT (cols) DO UPDATE` n'est PAS matchée — seules la forme
	// nue `ON CONFLICT DO UPDATE` et `INSERT OR REPLACE` le sont. Renforcer la
	// regex pour couvrir `(cols)` est tentant MAIS le scan est file-level : un
	// fichier qui écrit une table protégée en append-only ET contient par ailleurs
	// un ON CONFLICT légitime sur une AUTRE table (ex: skill_rating_loaders.go →
	// match_skill_rank append-only + lusr_component_history ON CONFLICT ; career.go
	// → player_csr_snapshots append-only + playlists_catalog ON CONFLICT) produirait
	// des faux positifs. Une détection fiable demanderait une analyse statement-level
	// (AST). En l'état : garde-rail best-effort, à compléter par la revue de code.
	regexp.MustCompile(`(?i)\bON\s+CONFLICT\b[^)]*\bDO\s+UPDATE\b`),
	// Forme parenthésée `ON CONFLICT (cols) DO UPDATE` — non couverte par le motif
	// nu ci-dessus (son `[^)]*` s'arrête au premier `)`). C'est précisément la forme
	// utilisée par les writers metadata catalogue/cache (catalog_fetcher, asset_index,
	// waypoint_assets_raw) qui ont FATAL-invalidé metadata.duckdb (thought_log 2026-05-30).
	regexp.MustCompile(`(?i)\bON\s+CONFLICT\s*\([^)]*\)\s*DO\s+UPDATE\b`),
	regexp.MustCompile(`(?i)\bINSERT\s+OR\s+REPLACE\b`),
	// NB : INSERT OR IGNORE n'est PAS scanné ici (file-level → faux positifs : un fichier
	// écrivant une table protégée + un INSERT OR IGNORE légitime sur medals_earned/xuid_aliases).
	// Il est gardé STATEMENT-LEVEL par append_only_state_guard_test.go (cible la table exacte).
}

// tablesProtegees : tables que ce PR a migrées en append-only. Toute
// écriture mutative (DELETE/UPDATE/UPSERT) sur ces tables hors allowlist
// est interdite. Ajouter une table ici quand sa migration append-only
// est mergée.
var tablesProtegees = []string{
	"match_skill_rank",
	"match_csrs",
	"player_csr_snapshots",
	"pve_match_stats",
	// Progression V2 (fix 2026-05-30) : on protège les tables CIBLES dont les
	// écritures ont été migrées hors ON CONFLICT :
	//  - milestone_earned : SELECT-then-INSERT (insert-only).
	//  - streak / streak_history : append-only (INSERT pur + vue streak_latest).
	// NB : la table legacy `player_records` n'est PAS protégée — un ON CONFLICT y
	// subsiste VOLONTAIREMENT dans 2 fallbacks de transition non-prod
	// (persistPlayerRecordsLegacy + l'ancien fallback test). Idem on ne protège pas
	// `player_records_history` ici car elle co-réside dans shared_social_persister.go
	// avec ce fallback legacy (le scan est file-level → faux positif). Le chemin
	// progression records écrit bien en append-only via AppendPlayerRecord.
	"streak",
	"streak_history",
	"milestone_earned",
	// metadata.duckdb (fix 2026-05-30) : writers catalogue/cache migrés hors
	// ON CONFLICT DO UPDATE vers SELECT-then-write ((*duckdb.DB).UpsertNoConflict).
	// Un ON CONFLICT DO UPDATE sur ces tables FATAL-invalide le handle metadata
	// partagé → modes/playlists en anglais sur toute l'app (incident page Synthesis).
	"playlists_catalog",
	"maps_catalog",
	"game_variants_catalog",
	"map_mode_pair_definitions",
	"pair_mode_label_translations",
	"asset_index",
	"waypoint_assets_raw",
	"season_calendars",
	"csr_season_calendars",
	"waypoint_resource_snapshots",
	// catalog_fetch_queue (fix ART 2026-06-19) : file de drain DiscoveryUGC migrée
	// en append-only (catalog_fetcher_service.go : plus de DELETE/UPDATE, pending =
	// NOT EXISTS catalogue). Un DELETE simple sur sa PK ART FATAL-invalidait
	// metadata.duckdb → cascade (modes/playlists/maps/citations/succès Xbox en échec
	// sur toute l'app — incident sonde live 2026-06-19).
	"catalog_fetch_queue",
	// player_match_enrichment (append-only #23046, 2026-06-21) : la table la PLUS
	// écrite, migrée append-only (id PK + stage + vue _latest). Tous les writers
	// sont des INSERT purs taggés ; zéro ON CONFLICT/DELETE/UPDATE. Le durcissement
	// complémentaire (interdire UPDATE + FROM brut) vit dans append_only_state_guard_test.go.
	"player_match_enrichment",
}

// allowlistArtPatterns : sites de prod où un pattern à risque reste
// présent VOLONTAIREMENT. Chaque entrée doit avoir un commentaire
// expliquant pourquoi. Format : "fichier:ligne_approx — raison".
//
// L'allowlist est volontairement vide pour les tables protégées —
// les patterns ne doivent pas exister. Pour les tables MOYEN/FAIBLE
// non encore migrées (cf. audit_art_writes.md), l'allowlist contient
// les sites tolérés temporairement.
var allowlistArtPatterns = map[string]string{
	// (Entrée `internal/persist/doc.go` retirée en E4 : ses seuls « patterns »
	// vivaient dans un commentaire ; le scan principal strippe les commentaires,
	// donc l'entrée était morte — TestAllowlistJustifiesEverything la refuse
	// désormais, bloquant + strip cohérent avec le scan.)
	//
	// FAUX POSITIF file-level (append-only #23046) : writes.go co-localise
	// UpsertPlayerEnrichment (player_match_enrichment, INSERT pur append-only) avec
	// les ON CONFLICT LÉGITIMES sur match_registry/match_participants (NON append-only,
	// UPSERT sérialisé par dblease — absents de tablesProtegees). La protection PRÉCISE
	// (statement-level) de player_match_enrichment vit dans append_only_state_guard_test.go
	// (TestNoMutationOnAppendOnlyStateTables + TestPlayerMatchEnrichmentAppendOnlyAccess).
	"internal/sync/writes.go": "Faux positif file-level : ON CONFLICT sur match_registry/match_participants (non append-only), pas sur player_match_enrichment (INSERT pur)",
}

// allowlistRawDelete : DELETE bruts sur table append-only TOLÉRÉS, avec
// justification (cf. TestNoRawDeleteOnAppendOnlyTables). Un raw DELETE reste à
// risque ART — n'ajouter ici QU'AVEC une raison documentée prouvant l'absence de
// déclencheur (player DB single-writer + zéro concurrence + PK BIGINT, pas VARCHAR).
var allowlistRawDelete = map[string]string{
	// compactMatchSkillRankSuperseded : DELETE de compaction (garde MAX(id) par
	// match_id+rating_type). Sur PLAYER DB (lease KindPlayer, single-writer, jamais
	// partagée → zéro concurrence) + PK BIGINT id (≠ VARCHAR, le déclencheur ART
	// historique). Justifié in-code. Blast radius = 1 player DB, pas la metadata
	// partagée (≠ incident catalog_fetch_queue).
	"internal/sync/skill_rating_postsync_persist.go": "compactMatchSkillRankSuperseded : DELETE compaction player DB single-writer, PK BIGINT, zéro concurrence — justifié in-code",
}

// TestNoARTPatternsOnProtectedTables — guard-rail principal.
//
// Pour chaque table protégée, scanne tous les fichiers .go non-test du
// projet. Toute occurrence d'un pattern à risque sur cette table est
// reportée. Le test fail si une occurrence non listée dans
// allowlistArtPatterns apparaît.
func TestNoARTPatternsOnProtectedTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string
	for _, table := range tablesProtegees {
		tableRegex := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(table) + `\b`)

		err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				// Skip vendor, .git, node_modules, etc.
				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" ||
					name == "data" || name == "logs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Exclure les fichiers de test, migrations (one-shot boot), outils
			// CLI/scripts one-shot (cmd/, scripts/ — exécutés hors serveur,
			// mono-processus), et le présent guard-rail. NB (E3, 2026-07-03) : ops/
			// N'EST PLUS exclu — sa plomberie (catalog_refresh, lying_bits_reset,
			// data_quality) tourne IN-PROCESS, donc soumise au tripwire.
			if strings.HasSuffix(path, "_test.go") ||
				strings.Contains(path, "/migration/") ||
				strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/migrations/") ||
				strings.Contains(path, "\\migrations\\") ||
				strings.Contains(path, "/cmd/") ||
				strings.Contains(path, "\\cmd\\") ||
				strings.Contains(path, "/scripts/") ||
				strings.Contains(path, "\\scripts\\") {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil // skip silently
			}
			// Retire les commentaires Go avant matching : un commentaire qui mentionne
			// le pattern à risque (ex. « évite ON CONFLICT DO UPDATE ») n'est pas une
			// écriture réelle et ne doit pas déclencher le garde-fou.
			text := stripGoComments(string(content))
			if !tableRegex.MatchString(text) {
				return nil
			}
			for _, pat := range patternsAtRisk {
				if pat.MatchString(text) {
					rel, _ := filepath.Rel(repoRoot, path)
					rel = filepath.ToSlash(rel)
					// Skip si le fichier est dans l'allowlist explicite.
					if _, allowed := allowlistArtPatterns[rel]; allowed {
						continue
					}
					violations = append(violations,
						"table="+table+" pattern_detected file="+rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("REGRESSION ART : %d violations détectées sur tables protégées :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
		t.Logf("Si l'ajout est volontaire, l'auteur doit migrer la table en append-only OU justifier dans le commit.")
	}
}

// TestNoRawDeleteOnAppendOnlyTables — un DELETE brut sur une table append-only
// (tablesProtegees) est INTERDIT. Le bug ART DuckDB « Failed to delete all rows
// from index » FATAL-invalide le handle metadata partagé pour tout le process
// (incident catalog_fetch_queue 2026-06-19 — drain DiscoveryUGC autonome au boot).
// Sur ces tables, seuls INSERT / INSERT OR IGNORE et SELECT-then-UPDATE single-row
// sont permis. Le motif est SCOPÉ au nom exact de la table → zéro faux positif
// (contrairement à patternsAtRisk qui est file-level). migrations/cmd/scripts
// exclus (rebuild one-shot, mono-processus) ; ops/ scanné depuis E3 (in-process).
func TestNoRawDeleteOnAppendOnlyTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string
	for _, table := range tablesProtegees {
		delRegex := regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b`)
		err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" ||
					name == "data" || name == "logs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") ||
				strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/migrations/") || strings.Contains(path, "\\migrations\\") ||
				strings.Contains(path, "/cmd/") || strings.Contains(path, "\\cmd\\") ||
				strings.Contains(path, "/scripts/") || strings.Contains(path, "\\scripts\\") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			if delRegex.MatchString(stripGoComments(string(content))) {
				rel, _ := filepath.Rel(repoRoot, path)
				relSlash := filepath.ToSlash(rel)
				if _, allowed := allowlistRawDelete[relSlash]; allowed {
					return nil
				}
				violations = append(violations, "table="+table+" DELETE brut file="+relSlash)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("DELETE brut sur table append-only (INTERDIT — bug ART, cf. incident catalog_fetch_queue) : %d :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestAllowlistJustifiesEverything — vérifie que chaque entrée de
// l'allowlist correspond bien à un fichier qui contient au moins un
// pattern à risque. Si une entrée est obsolète (fichier nettoyé), le
// test alerte pour la retirer.
func TestAllowlistJustifiesEverything(t *testing.T) {
	repoRoot := findRepoRoot(t)

	for fileRel, reason := range allowlistArtPatterns {
		// Normaliser les séparateurs.
		absPath := filepath.Join(repoRoot, filepath.FromSlash(fileRel))
		content, err := os.ReadFile(absPath)
		if err != nil {
			t.Errorf("allowlist : fichier introuvable %q (raison: %q) — entrée à retirer ?", fileRel, reason)
			continue
		}
		// Détection cohérente avec le scan principal (stripGoComments) : une
		// « justification » qui n'existe que dans un commentaire ne compte PAS —
		// le scan strippe les commentaires, donc une telle entrée est morte.
		text := stripGoComments(string(content))
		hasRiskPattern := false
		for _, pat := range patternsAtRisk {
			if pat.MatchString(text) {
				hasRiskPattern = true
				break
			}
		}
		if !hasRiskPattern {
			t.Errorf("allowlist ART obsolète : %q n'a plus de pattern à risque dans son CODE "+
				"(commentaires strippés) → retirer l'entrée (raison historique: %q)",
				fileRel, reason)
		}
	}
}

// criticalMatchTables : tables de "match-of-record" + enrichment qui ne sont PAS
// append-only (donc absentes de tablesProtegees) mais dont la forme bulk
// `UPDATE … FROM (VALUES …)` multi-row est le déclencheur ART direct (1 statement
// touchant N entrées de l'index → "Failed to delete all rows from index").
// Le row-by-row UPDATE sérialisé reste autorisé (cf. PostSyncEnrichmentPersister).
var criticalMatchTables = []string{
	"match_registry",
	"match_participants",
	"medals_earned",
	"killer_victim_pairs",
	"weapon_kills",
	"match_skill_rank",
	"match_csrs",
	"player_csr_snapshots",
}

// reBulkUpdateFromValues détecte `UPDATE <table> … FROM (VALUES …)` (bulk
// multi-row). Le `.{0,400}?` borne la fenêtre au statement courant pour éviter
// les faux positifs cross-statement (un UPDATE row-by-row et un SELECT FROM
// (VALUES) sans rapport dans le même fichier ne doivent pas matcher).
func reBulkUpdateFromValues(table string) *regexp.Regexp {
	return regexp.MustCompile(`(?is)\bUPDATE\s+` + regexp.QuoteMeta(table) + `\b.{0,400}?\bFROM\s*\(\s*VALUES\b`)
}

// reUpdateRawSQL capture un littéral SQL brut (délimité par des backticks Go)
// contenant `UPDATE <table>`. Le second garde-fou vérifie ensuite la présence d'au
// moins un placeholder `?` DANS ce littéral : son absence = UPDATE set-based multi-row
// NU (prédicat pur, aucune valeur liée), l'autre déclencheur ART direct à côté de
// `FROM (VALUES)`. Un UPDATE row-by-row sérialisé lie toujours match_id à `?`.
// Ancrage sur les backticks (bornes réelles du littéral SQL) → pas de faux positif
// sur les commentaires ni de fenêtre tronquée (E2, 2026-07-03).
func reUpdateRawSQL(table string) *regexp.Regexp {
	return regexp.MustCompile("(?is)`[^`]*\\bUPDATE\\s+" + regexp.QuoteMeta(table) + "\\b[^`]*`")
}

// TestNoBulkMultiRowUpdateOnCriticalTables — garde-fou complémentaire à
// TestNoARTPatternsOnProtectedTables. Les tables match-of-record ne sont pas
// dans tablesProtegees (leurs UPDATE bitmask/row-by-row sérialisés sont sûrs),
// mais la forme bulk `UPDATE … FROM (VALUES …)` multi-row — le vrai déclencheur
// ART — y est INTERDITE. Ce test fail si elle réapparaît sur une table critique.
// Périmètre identique au scan principal (hors _test/migration/cmd/scripts ; ops/ inclus depuis E3).
func TestNoBulkMultiRowUpdateOnCriticalTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string
	for _, table := range criticalMatchTables {
		reValues := reBulkUpdateFromValues(table)
		reStmt := reUpdateRawSQL(table)
		err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" ||
					name == "data" || name == "logs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") ||
				strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/migrations/") || strings.Contains(path, "\\migrations\\") ||
				strings.Contains(path, "/cmd/") || strings.Contains(path, "\\cmd\\") ||
				strings.Contains(path, "/scripts/") || strings.Contains(path, "\\scripts\\") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			text := stripGoComments(string(content))
			rel, _ := filepath.Rel(repoRoot, path)
			if reValues.MatchString(text) {
				violations = append(violations, "bulk-from-values table="+table+" file="+filepath.ToSlash(rel))
			}
			// Bare bulk (set-based, aucun `?` dans le corps du statement) — E2.
			for _, stmt := range reStmt.FindAllString(text, -1) {
				if !strings.Contains(stmt, "?") {
					violations = append(violations, "bare-bulk (aucun placeholder) table="+table+" file="+filepath.ToSlash(rel))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("ART bulk UPDATE multi-row détecté sur table(s) critique(s) — "+
			"utiliser N UPDATE row-by-row `WHERE match_id = ?` (PostSyncEnrichmentPersister) ou "+
			"INSERT-only (persist) :\n  - %s",
			strings.Join(violations, "\n  - "))
	}
}

// TestBareBulkUpdateDetection_Sanity valide que le garde-fou bare-bulk MORD :
// il attrape un UPDATE set-based nu (aucun `?`) et LAISSE PASSER un row-by-row
// `WHERE match_id = ?`. Un garde-fou qui ne détecte jamais rien est inutile.
func TestBareBulkUpdateDetection_Sanity(t *testing.T) {
	re := reUpdateRawSQL("match_registry")
	bare := "q := `UPDATE match_registry SET pair_name = game_variant_name || map_name WHERE pair_id IS NOT NULL`"
	rowByRow := "q := `UPDATE match_registry SET pair_name = ? WHERE match_id = ?`"
	if m := re.FindString(bare); m == "" || strings.Contains(m, "?") {
		t.Errorf("bare bulk devrait matcher SANS placeholder, got %q", m)
	}
	if m := re.FindString(rowByRow); m == "" || !strings.Contains(m, "?") {
		t.Errorf("row-by-row devrait matcher AVEC placeholder, got %q", m)
	}
}

// reBlockComment / reLineComment : pour retirer les commentaires Go avant le
// scan des patterns à risque (un commentaire explicatif n'est pas une écriture).
var (
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLineComment  = regexp.MustCompile(`//[^\n]*`)
)

// stripGoComments retire les commentaires bloc et ligne. Approximation suffisante
// pour le garde-fou : un `//` à l'intérieur d'une string literal (ex. URL) peut
// être tronqué, mais les patterns SQL recherchés (ON CONFLICT…, INSERT OR REPLACE)
// vivent dans des raw strings multi-lignes dont la ligne pertinente ne contient
// pas de `//`.
func stripGoComments(src string) string {
	src = reBlockComment.ReplaceAllString(src, "")
	src = reLineComment.ReplaceAllString(src, "")
	return src
}

// findRepoRoot retourne la racine de `apps/go-api/` (depuis laquelle les
// paths de l'allowlist sont relatifs : ils commencent par `internal/...`).
// Les tests Go tournent depuis le dossier du package, donc on remonte
// jusqu'à trouver le `go.mod` du module.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("module root (go.mod) non trouvé depuis %s", wd)
	return ""
}
