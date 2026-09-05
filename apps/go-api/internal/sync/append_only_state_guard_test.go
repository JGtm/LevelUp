// Package sync — append_only_state_guard_test.go : garde-fou anti-régression pour
// les tables d'ÉTAT migrées en APPEND-ONLY (doctrine zéro DELETE / zéro UPDATE
// indexé / zéro ON CONFLICT DO UPDATE).
//
// no_art_patterns_test.go ne couvre QUE ON CONFLICT / INSERT OR REPLACE et reste
// aveugle au DELETE nu (faux positifs file-level). Ce test comble le trou pour les
// tables converties : tout `DELETE FROM <table>` ou `INSERT … ON CONFLICT … DO
// UPDATE` / `INSERT OR REPLACE` sur l'une d'elles, dans le hot path serveur
// (hors _test / migration / ops / cmd / scripts), fait échouer la CI. L'état se lit
// via `<table>_latest` ; toute mutation est un INSERT pur.
//
// Ajouter une table ici à CHAQUE conversion append-only (campagne en cours).

package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// appendOnlyStateTables : tables d'état converties en append-only (event-log).
// Aucun DELETE / ON CONFLICT DO UPDATE / INSERT OR REPLACE toléré dessus.
var appendOnlyStateTables = []string{
	"match_favorites",
	"match_favorites_history",
	"media_likes",
	"media_likes_history",
	"squad_member",
	"squad_member_history",
	"notification_preferences",
	"notification_preferences_history",
	"media_match_associations",
	"media_match_associations_history",
	"player_notifications",
	"player_notifications_history",
	"squad_challenge_participant",
	"squad_challenge_participant_history",
	"user_prestige",
	"user_prestige_history",
	// Pattern in-place (id PK + vue _latest, comme match_skill_rank) : la table
	// garde son nom, pas de table _history séparée. Lecture via
	// lusr_component_history_latest. Writers (loaders + persister) = INSERT pur.
	"lusr_component_history",
	// player_match_enrichment : la table la PLUS écrite, migrée append-only
	// (#23046, 2026-06-21) — id PK + colonne stage + written_at, lecture via
	// player_match_enrichment_latest (merge-on-read par-groupe). INSERT pur taggé.
	"player_match_enrichment",
	// pve_match_stats : append-only in-place (id PK + vue pve_match_stats_latest).
	// L'écriture passe par un guard SELECT-then-INSERT idempotent (pve_persister.go) ;
	// l'ancien INSERT OR IGNORE est interdit (audit adversarial 2026-06-21).
	"pve_match_stats",
	// personal_score_awards : append-only GÉNÉRATION (Phase 2, 2026-06-21). L'ancien
	// DELETE+INSERT (replace-semantics, vecteur ART sur 4 index idx_psa_*) est remplacé
	// par INSERT pur taggé generation_id (séquence psa_generation_seq, partagée par
	// appel) + tombstone pour le clear. Lecture via personal_score_awards_latest
	// (DENSE_RANK, génération MAX, tombstones exclus). Zéro DELETE/ON CONFLICT.
	"personal_score_awards",
	// player_skill_state_v2 : append-only (written_at + vue _latest). Le reset
	// watermark (RecomputeLUSRCanonicalForPlayer) n'utilise plus DELETE WHERE xuid
	// (vecteur ART sur PK + idx_pssv2) mais une row sentinelle is_reset=TRUE
	// (Phase 2, 2026-06-21). Lecture via player_skill_state_v2_latest.
	"player_skill_state_v2",
	// match_citations : append-only GÉNÉRATION (Phase 2, 2026-06-21). L'ancien
	// DELETE+réécriture (recompute SOUSTRACTIF, PK composite) est remplacé par
	// INSERT pur taggé generation_id (par match) ; la sentinelle '_processed' gère
	// le cas 0 citation. Lecture via match_citations_latest (DENSE_RANK par match_id).
	"match_citations",
	// weapon_kills : append-only GÉNÉRATION (Phase 2, 2026-06-21). L'ancien
	// DELETE+INSERT (idx_wk_match_xuid, DB shared multi-writer) est remplacé par
	// INSERT pur taggé generation_id (par (match,xuid)). Lecture via la vue
	// v_weapon_kills (DENSE_RANK génération MAX). Pas de PK → simple ALTER ADD COLUMN.
	"weapon_kills",
	// match_objective_stats (V72-03, 2026-07-25) : stats objectifs par joueur/match
	// (CTF/Zones/Oddball), append-only créée directement (id PK seq + written_at + vue
	// match_objective_stats_latest). Écriture = INSERT pur (persistObjectiveStats) ;
	// aucun DELETE / ON CONFLICT / INSERT OR REPLACE|IGNORE toléré. Lecture via _latest.
	"match_objective_stats",
	// match_kill_events / match_weapon_shots (J4, 2026-08-01) : créées directement
	// append-only (id PK seq + decode_pass + written_at + vue _latest). L'unité de
	// génération est la PASSE DE DÉCODAGE, pas la ligne : la vue retient la DERNIÈRE
	// PASSE par match. Écriture = INSERT pur (kill_events_persister.go /
	// weapon_shots_persister.go, un seul statement chacun) ; aucun DELETE /
	// ON CONFLICT / INSERT OR REPLACE|IGNORE toléré. Lecture via _latest UNIQUEMENT —
	// une lecture brute servirait les lignes des passes précédentes.
	"match_kill_events",
	"match_weapon_shots",
	// match_usage_players / match_usage_films (session-usage, 2026-09-04) : créées
	// directement append-only (id PK seq + summary_pass + summary_rev + written_at +
	// vues _latest). L'unité de génération est LA PROJECTION D'UN ARTEFACT ENTIER, pas
	// la ligne : la vue films_latest retient la dernière PASSE par match, et
	// players_latest se JOINT à elle (un joueur d'une passe précédente disparaît avec
	// elle). Écriture = INSERT pur (persist/usage_summary_persister.go, deux statements
	// dans une transaction unique) ; aucun DELETE / ON CONFLICT / INSERT OR
	// REPLACE|IGNORE toléré. Lecture via _latest UNIQUEMENT.
	// Recette ADR 0026 étape 5 — l'inscription manquait à l'arrivée de la branche.
	"match_usage_players",
	"match_usage_films",
	// match_bomb_stats (E3 Assaut, 2026-09-04) : créée directement append-only (id PK seq +
	// written_at + vue match_bomb_stats_latest). Unité de génération = la PASSE DE DÉCODAGE,
	// arbitrée par written_at puis id. Écriture = INSERT pur (bomb_stats_persister.go) ;
	// aucun DELETE / ON CONFLICT / INSERT OR REPLACE|IGNORE toléré. Lecture via _latest
	// UNIQUEMENT — une lecture brute servirait les lignes des passes précédentes.
	"match_bomb_stats",
	// kill_positions / match_weapon_hit_distance (G4 du registre v2, enrôlement 2026-09-05) :
	// les deux dernières tables du film qui n'étaient enrôlées dans AUCUNE des deux listes
	// anti-ART, alors que les deux sont append-only avec vue _latest depuis leur migration.
	//   - kill_positions : rebuild append-only G.2 (2026-08-30,
	//     games/halo_infinite/migrations/steps_appendonly_misc.go) — id PK + written_at + vue
	//     kill_positions_latest par (match_id, killer_xuid, time_ms). Unité de génération = LA
	//     LIGNE (pas la passe) : la table n'a volontairement pas de decoder_rev, written_at
	//     arbitre. Écrivains INSERT purs : persist/kill_position_persister.go (film Infinite),
	//     persist/shared_persister.go persistKillPositions (builder Halo 5).
	//   - match_weapon_hit_distance : créée append-only
	//     (migration/steps_shared_weapon_hit_distance.go) — id PK seq + decode_pass +
	//     decoder_rev + written_at + vue _latest qui retient LA DERNIÈRE PASSE PAR MATCH
	//     (l'unité de production est le film entier). Écrivain unique INSERT pur :
	//     persist/weapon_hit_distance_persister.go.
	// Aucun DELETE / ON CONFLICT / INSERT OR REPLACE|IGNORE sur l'une ou l'autre aujourd'hui :
	// l'enrôlement ne crée aucune entrée d'allowlist.
	"kill_positions",
	"match_weapon_hit_distance",
	// match_player_positions (décision utilisateur 1 du plan v2, 2026-09-06) : convertie
	// append-only (id PK + positions_pass + written_at + vue _latest PAR PASSE) le jour où elle
	// est devenue une projection de l'artefact de rejeu, écrite dans le cycle de sync.
	// L'unité de génération est LA PROJECTION D'UN MATCH ENTIER : toutes les lignes d'une passe
	// partagent positions_pass et written_at, et la vue retient la dernière passe par match.
	// Écriture = INSERT pur (persist/player_positions_persister.go) ; lecture via _latest
	// UNIQUEMENT — une lecture brute empilerait toutes les projections d'un match.
	"match_player_positions",
}

// rawPMEReadAllowlist : accès BRUTS intentionnels à player_match_enrichment (hors
// vue _latest) tolérés dans le hot path — retirés du texte avant le scan raw-read.
var rawPMEReadAllowlist = []string{
	// player_persister.go : ancre d'idempotence du sous-batch live (vérifie la
	// présence d'une row stage='live' sur la table physique). Pas un reader UI.
	"FROM player_match_enrichment WHERE match_id = ? AND stage = 'live'",
	// engagement.go loadExistingEngagementScores : marqueur d'idempotence writer-side
	// (engagement déjà TENTÉ — borne la croissance des matchs insufficient_history).
	// La vue n'expose pas `stage` → lecture physique nécessaire.
	"FROM player_match_enrichment WHERE stage = 'engagement'",
}

// TestPlayerMatchEnrichmentAppendOnlyAccess durcit la doctrine append-only sur
// player_match_enrichment au-delà du test ON CONFLICT/DELETE ci-dessus :
//   - AUCUN `UPDATE player_match_enrichment` (append-only = INSERT pur),
//   - AUCUNE lecture brute `FROM/JOIN player_match_enrichment` (hors _latest) dans
//     le hot path serveur → doit passer par player_match_enrichment_latest (sinon
//     fan-out N rows/match : agrégats faussés, JOIN dupliqués, delta-filters en boucle).
//
// Exclut migration (table physique), ops/cmd/scripts (outils), validation (parité
// Go-vs-Python legacy : la py DB n'a pas la vue). Allowlist = ancre EXISTS du persister.
func TestPlayerMatchEnrichmentAppendOnlyAccess(t *testing.T) {
	repoRoot := findRepoRoot(t)
	reUpdate := regexp.MustCompile(`(?i)\bUPDATE\s+player_match_enrichment\b`)
	reRawRead := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+player_match_enrichment(\w*)`)

	var violations []string
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
			strings.Contains(path, "/ops/") || strings.Contains(path, "\\ops\\") ||
			strings.Contains(path, "/cmd/") || strings.Contains(path, "\\cmd\\") ||
			strings.Contains(path, "/validation/") || strings.Contains(path, "\\validation\\") ||
			strings.Contains(path, "/scripts/") || strings.Contains(path, "\\scripts\\") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := stripGoComments(string(content))
		for _, allowed := range rawPMEReadAllowlist {
			text = strings.ReplaceAll(text, allowed, "")
		}
		rel, _ := filepath.Rel(repoRoot, path)
		rel = filepath.ToSlash(rel)
		if reUpdate.MatchString(text) {
			violations = append(violations, "UPDATE player_match_enrichment dans "+rel)
		}
		for _, m := range reRawRead.FindAllStringSubmatch(text, -1) {
			if m[1] == "" { // suffixe vide = table brute (ni _latest ni __appendonly/__rebuilt)
				violations = append(violations, "FROM/JOIN player_match_enrichment brut (hors _latest) dans "+rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("RÉGRESSION append-only PME : %d accès interdit(s) "+
			"(lecture via player_match_enrichment_latest ; écriture = INSERT pur taggé stage) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

func TestNoMutationOnAppendOnlyStateTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	reOnConflictDoUpdate := regexp.MustCompile(`(?is)\bON\s+CONFLICT\b[^;]*\bDO\s+UPDATE\b`)
	reInsertOrReplace := regexp.MustCompile(`(?i)\bINSERT\s+OR\s+REPLACE\b`)
	reInsertOrIgnore := regexp.MustCompile(`(?i)\bINSERT\s+OR\s+IGNORE\b`)

	var violations []string
	for _, table := range appendOnlyStateTables {
		reDelete := regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b`)
		// INSERT ... ON CONFLICT/REPLACE/IGNORE visant cette table : on borne au statement
		// commençant par INSERT [OR REPLACE|IGNORE] INTO <table> (statement-level → pas de
		// faux positif file-level vs un INSERT OR IGNORE sur une AUTRE table du même fichier).
		// Borne = `;` OU backtick (fin de la string SQL Go) : sans le backtick, [^;]* ferait
		// le pont jusqu'au ON CONFLICT d'une AUTRE table plus loin dans le MÊME fichier (les
		// strings SQL Go n'ont pas de `;` final) — faux positif observé sur shared_persister.go.
		reInsertTable := regexp.MustCompile(`(?is)\bINSERT\s+(?:OR\s+(?:REPLACE|IGNORE)\s+)?INTO\s+` + regexp.QuoteMeta(table) + "\\b[^;`]*")

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
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/ops/") || strings.Contains(path, "\\ops\\") ||
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
			rel = filepath.ToSlash(rel)
			if reDelete.MatchString(text) {
				violations = append(violations, "DELETE FROM "+table+" dans "+rel)
			}
			for _, stmt := range reInsertTable.FindAllString(text, -1) {
				if reOnConflictDoUpdate.MatchString(stmt) || reInsertOrReplace.MatchString(stmt) || reInsertOrIgnore.MatchString(stmt) {
					violations = append(violations, "INSERT ON CONFLICT/REPLACE/IGNORE sur "+table+" dans "+rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("RÉGRESSION append-only : %d mutation(s) interdite(s) sur table d'état "+
			"(append-only = INSERT pur + vue _latest ; zéro DELETE/ON CONFLICT) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// mediaAppendOnlyTables : les deux tables append-only du domaine MÉDIA
// (`media_likes_history`, `media_match_associations_history` — id PK + written_at +
// vues `_latest`). Elles sont VOLONTAIREMENT absentes de `tablesProtegees`
// (no_art_patterns_test.go) : ce scan-là est FILE-level, et ces tables co-résident
// dans `internal/persist/shared_social_persister.go` avec le fallback legacy
// `ON CONFLICT` de `player_records` — les y ajouter produirait un faux positif
// immédiat (même cas que `player_records_history`, déjà documenté là-bas).
//
// Le garde-rail ci-dessous comble ce trou en restant STATEMENT-level : chaque motif
// est ancré sur le NOM EXACT de la table, donc un `ON CONFLICT` / `DELETE` visant une
// AUTRE table du même fichier ne peut pas le déclencher.
var mediaAppendOnlyTables = []string{
	"media_likes_history",
	"media_match_associations_history",
}

// allowlistMediaMutation : sites de prod où un DELETE/UPDATE sur une table média
// append-only serait toléré. Format : "fichier — raison (date)". Toute entrée doit
// prouver l'absence de déclencheur ART (le bug DuckDB #23046 FATAL-invalide le handle
// partagé pour tout le process — cf. incident catalog_fetch_queue 2026-06-19).
//
// Vide au 2026-08-03 : aucune mutation tolérée. Les seuls DELETE existants vivent
// dans `cmd/cleanup_media_index` et `cmd/purge_player_media` — outils one-shot
// mono-process, hors serveur, exclus du périmètre comme partout ailleurs.
var allowlistMediaMutation = map[string]string{}

// reMediaDelete / reMediaUpdate : motifs STATEMENT-level pour une table donnée.
// `\b` en fin de nom ne déborde pas sur un suffixe (`_latest`, `_v2` : `_` est un
// caractère de mot, donc pas de frontière) → zéro faux positif de voisinage.
func reMediaDelete(table string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b`)
}

func reMediaUpdate(table string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\bUPDATE\s+` + regexp.QuoteMeta(table) + `\b`)
}

// TestNoMutationOnMediaAppendOnlyTables — garde-rail DELETE/UPDATE dédié aux tables
// média append-only. Il complète TestNoMutationOnAppendOnlyStateTables sur DEUX axes :
//
//  1. Le volet UPDATE : le test générique ci-dessus ne couvre que DELETE et
//     INSERT…ON CONFLICT/REPLACE/IGNORE. Un `UPDATE media_likes_history SET is_active…`
//     (la mutation « naturelle » qu'écrirait quelqu'un qui ignore le modèle event-log)
//     n'était interdit NULLE PART.
//  2. Le périmètre `internal/ops/` : le test générique l'exclut (outillage), or c'est
//     précisément là que vivent les writers de ces deux tables (`ops/media_associate.go`
//     INSERT dans media_match_associations_history, `ops/media_store.go` en crée le
//     substrat). Exclure ops/ vidait le garde-rail de sa substance sur ce domaine.
//     ops/ tourne IN-PROCESS (même raisonnement que l'inclusion d'ops/ dans
//     no_art_patterns_test.go, E3 2026-07-03) → soumis au tripwire.
//
// Restent exclus : _test.go, /migration/ (rebuild one-shot sur la table physique),
// /cmd/ et /scripts/ (outils autonomes mono-process).
func TestNoMutationOnMediaAppendOnlyTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string
	for _, table := range mediaAppendOnlyTables {
		reDelete := reMediaDelete(table)
		reUpdate := reMediaUpdate(table)

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
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
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
			rel = filepath.ToSlash(rel)
			if _, allowed := allowlistMediaMutation[rel]; allowed {
				return nil
			}
			if reDelete.MatchString(text) {
				violations = append(violations, "DELETE FROM "+table+" dans "+rel)
			}
			if reUpdate.MatchString(text) {
				violations = append(violations, "UPDATE "+table+" dans "+rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("RÉGRESSION append-only MÉDIA : %d mutation(s) interdite(s) "+
			"(append-only = INSERT pur + vue _latest ; un DELETE/UPDATE indexé déclenche "+
			"le bug ART DuckDB #23046 qui FATAL-invalide le handle shared_social) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestMediaMutationDetection_Sanity prouve que le garde-rail ci-dessus MORD : un
// garde-fou qui ne détecte jamais rien est inutile (même doctrine que
// TestBareBulkUpdateDetection_Sanity). Vérifie les deux sens :
//   - il attrape le DELETE et l'UPDATE sur la table exacte ;
//   - il LAISSE PASSER l'INSERT pur, la lecture via la vue _latest, et une mutation
//     visant une AUTRE table (preuve du caractère statement-level : pas de faux
//     positif file-level, qui est la raison même de l'existence de ce test).
func TestMediaMutationDetection_Sanity(t *testing.T) {
	for _, table := range mediaAppendOnlyTables {
		reDelete := reMediaDelete(table)
		reUpdate := reMediaUpdate(table)

		mustMatch := []string{
			"q := `DELETE FROM " + table + " WHERE media_file_id = ?`",
			"q := `UPDATE " + table + " SET is_active = FALSE WHERE media_file_id = ?`",
		}
		for _, src := range mustMatch {
			if !reDelete.MatchString(src) && !reUpdate.MatchString(src) {
				t.Errorf("table=%s : mutation NON détectée (garde-rail aveugle) : %q", table, src)
			}
		}

		mustNotMatch := []string{
			"q := `INSERT INTO " + table + " (media_file_id, is_active) VALUES (?, ?)`",
			"q := `SELECT media_file_id FROM " + table + "_latest WHERE is_active`",
			"q := `DELETE FROM media_files WHERE id = ?`",
			// Fixture synthétique, pas un exemple de code réel : prouve l'absence
			// de faux positif du regex sur une table hors périmètre (media_files
			// n'est pas dans mediaAppendOnlyTables). La colonne liked ayant été
			// DROPPÉE le 2026-08-04, la chaîne emploie une colonne existante.
			"q := `UPDATE media_files SET status = 'deleted' WHERE id = ?`",
		}
		for _, src := range mustNotMatch {
			if reDelete.MatchString(src) || reUpdate.MatchString(src) {
				t.Errorf("table=%s : FAUX POSITIF sur écriture légitime : %q", table, src)
			}
		}
	}
}
