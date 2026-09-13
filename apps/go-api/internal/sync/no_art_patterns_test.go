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
	// player_match_enrichment (append-only #23645, 2026-06-21) : la table la PLUS
	// écrite, migrée append-only (id PK + stage + vue _latest). Tous les writers
	// sont des INSERT purs taggés ; zéro ON CONFLICT/DELETE/UPDATE. Le durcissement
	// complémentaire (interdire UPDATE + FROM brut) vit dans append_only_state_guard_test.go.
	"player_match_enrichment",
	// match_objective_stats (V72-03, 2026-07-25) : stats objectifs par joueur/match
	// (CTF/Zones/Oddball), CREEE directement append-only (id PK seq + written_at + vue
	// match_objective_stats_latest). Writer unique = persist.persistObjectiveStats
	// (INSERT pur dans la transaction shared). Lecture via la vue _latest UNIQUEMENT.
	"match_objective_stats",
	// match_kill_events (J4, 2026-08-01) : table append-only NET-NEUVE (1 ligne par
	// mort, crédit du kill-feed + source du dégât). Son persister
	// (internal/persist/kill_events_persister.go) n'émet QU'UN statement, un INSERT —
	// il n'a RIEN à faire dans allowlistArtPatterns, et c'est précisément ce qui permet
	// de protéger la table ici sans aucune exception. Le remplacement d'une passe de
	// décodage se fait en écrivant une NOUVELLE passe, jamais en effaçant l'ancienne :
	// la vue match_kill_events_latest ne rend que la dernière. Le DELETE nu, lui, est
	// couvert par TestNoRawDeleteOnAppendOnlyTables ci-dessous (scopé au nom exact de
	// la table) et par append_only_state_guard_test.go.
	"match_kill_events",
	// match_weapon_shots (J4, 2026-08-01) : table append-only NET-NEUVE (grain match x
	// joueur x arme, une seule mesure : le nombre de tirs décodés). Son persister
	// (internal/persist/weapon_shots_persister.go) n'émet qu'un INSERT — aucune entrée
	// d'allowlist à prévoir. Même mécanique de passe et même couverture DELETE que
	// match_kill_events ci-dessus.
	"match_weapon_shots",
	// match_usage_players / match_usage_films (session-usage, 2026-09-04) : tables
	// append-only NET-NEUVES (résumé d'usage équipement/socles dérivé de l'artefact de
	// rejeu, une passe = un artefact projeté). Leur persister
	// (internal/persist/usage_summary_persister.go) n'émet que des INSERT dans une
	// transaction unique — aucune entrée d'allowlist à prévoir. Même mécanique de passe
	// (vues _latest) et même couverture DELETE que match_kill_events ci-dessus.
	"match_usage_players",
	"match_usage_films",
	// match_bomb_stats (E3 Assaut, 2026-09-04) : table append-only NET-NEUVE (5 statistiques
	// d'objectif du mode Assaut par joueur/match, reconstruites du film — l'API 343 n'en
	// publie aucune). Son persister (internal/persist/bomb_stats_persister.go) n'émet que des
	// INSERT — aucune entrée d'allowlist à prévoir, ni ici ni dans allowlistRawDelete.
	// Remplacer une passe = en écrire une nouvelle ; la vue match_bomb_stats_latest ne rend
	// que la dernière ligne par (match_id, xuid).
	"match_bomb_stats",
	// match_flag_grabs_net (prises nettes de drapeau, 2026-09-13) : table append-only
	// NET-NEUVE (prises brutes et nettes par joueur/match, lues du calque de drapeau de
	// l'artefact, plus la fenêtre de jonglage appliquée). Son persister
	// (internal/persist/flag_grabs_net_persister.go) n'émet que des INSERT dans une
	// transaction unique — aucune entrée d'allowlist à prévoir, ni ici ni dans
	// allowlistRawDelete. Remplacer une passe = en écrire une nouvelle ; la vue
	// match_flag_grabs_net_latest ne rend que la dernière ligne par (match_id, xuid).
	"match_flag_grabs_net",
	// kill_positions / match_weapon_hit_distance (G4 du registre v2, enrôlement 2026-09-05) :
	// les deux dernières tables du film restées HORS des deux listes anti-ART alors qu'elles
	// sont append-only avec vue _latest depuis leur migration. Vérifié sur pièces avant
	// enrôlement :
	//   - kill_positions : rebuild append-only G.2 (2026-08-30) par
	//     games/halo_infinite/migrations/steps_appendonly_misc.go (id PK kill_positions_seq +
	//     written_at), puis bascule sur un arbitrage PAR PASSE au lot 1.7 (2026-09-09,
	//     steps_shared_kill_positions_pass.go : + decode_pass NOT NULL, vue
	//     kill_positions_latest = DERNIÈRE PASSE ENTIÈRE par match) ; deux écrivains, tous deux
	//     en INSERT pur : persist/kill_position_persister.go (passe film Infinite) et
	//     persist/shared_persister.go persistKillPositionsPass (chemin builder Halo 5).
	//     Pas de decoder_rev sur cette table, et c'est voulu (decode_pass discrimine déjà).
	//   - match_weapon_hit_distance : CRÉÉE append-only (migration/steps_shared_weapon_hit_distance.go,
	//     id PK seq + decode_pass + decoder_rev + written_at + vue _latest par PASSE) ; écrivain
	//     unique persist/weapon_hit_distance_persister.go, un seul statement INSERT.
	// Enrôlement GRATUIT : aucune violation existante, aucune entrée d'allowlist créée. Le `\b`
	// final ne déborde pas sur les vues `_latest` (`_` est un caractère de mot).
	"kill_positions",
	"match_weapon_hit_distance",
	// match_player_positions (décision utilisateur 1 du plan v2, 2026-09-06) : la table de la
	// CARTE DE CHALEUR devient une PROJECTION DE L'ARTEFACT de rejeu, écrite dans le cycle de
	// sync. Elle était jusque-là remplie par un outil de diagnostic en DELETE-then-INSERT sur
	// le handle de LECTURE du pool — hors pression concurrente, donc tolérable ; sous le
	// nouveau régime, ce DELETE indexé serait le déclencheur ART direct. Elle est convertie
	// append-only (migration/steps_shared_player_positions_appendonly.go : id PK +
	// positions_pass + vue match_player_positions_latest PAR PASSE) et son unique écrivain,
	// persist/player_positions_persister.go, n'émet que des INSERT — aucune entrée d'allowlist.
	// `PlayerPositionsRepo.WriteMatch` a été SUPPRIMÉE avec ses tests.
	"match_player_positions",
	// match_lives / match_death_context (7C isolement au sync, 2026-09-07) : tables
	// append-only NET-NEUVES (vies nommées du film ; voisinage à l'instant de chaque mort du
	// journal). Leur persister (internal/persist/lives_persister.go) n'émet que des INSERT
	// dans une transaction unique — aucune entrée d'allowlist à prévoir, ni ici ni dans
	// allowlistRawDelete. Même mécanique de passe que match_kill_events : remplacer une passe
	// consiste à en écrire une nouvelle sous un `decode_pass` neuf, et les vues _latest ne
	// rendent que la DERNIÈRE PASSE PAR MATCH — jamais la dernière ligne par clé, sans quoi
	// une passe plus courte laisserait survivre les lignes de la précédente.
	"match_lives",
	"match_death_context",
	// NB (2026-08-03) : `media_likes_history` et `media_match_associations_history` sont
	// append-only elles aussi mais N'ONT PAS leur place ICI — même raison que
	// `player_records_history` ci-dessus : elles co-résident dans
	// internal/persist/shared_social_persister.go avec le fallback legacy ON CONFLICT de
	// `player_records`, et ce scan est FILE-level → faux positif immédiat. Elles sont
	// gardées STATEMENT-level par TestNoMutationOnMediaAppendOnlyTables
	// (append_only_state_guard_test.go), qui couvre en plus le volet UPDATE et inclut
	// internal/ops/ (où vivent leurs writers).
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
	// (Entrée `internal/sync/writes.go` retirée en V4b/2026-07-07 : le trio
	// InsertRegistryIfNotExists/InsertParticipants/InsertMedals — seuls porteurs
	// des ON CONFLICT DO UPDATE dans ce fichier — a été supprimé (0 caller prod
	// depuis que l'import OpenSpartan est routé via persist.SharedPersister en E1).
	// writes.go n'a plus aucun pattern à risque → entrée morte.)
	//
	// L'allowlist est désormais vide : aucun pattern ART toléré en prod.
}

// allowlistRawDelete : DELETE bruts sur table append-only TOLÉRÉS, avec
// justification (cf. TestNoRawDeleteOnAppendOnlyTables). Un raw DELETE reste à
// risque ART — n'ajouter ici QU'AVEC une raison documentée prouvant l'absence de
// déclencheur (player DB single-writer + zéro concurrence + PK BIGINT, pas VARCHAR).
var allowlistRawDelete = map[string]string{
	// (Entrée `internal/sync/skill_rating_postsync_persist.go` retirée en
	// V4c/2026-07-07 : la fonction compactMatchSkillRankSuperseded (DELETE de
	// compaction) a été SUPPRIMÉE — elle déclenchait le bug ART #23645 malgré
	// mono-writer + PK BIGINT (crash JGtm 2026-06-20). La table match_skill_rank
	// reste append-only pur, la vue _latest reste correcte. Le fichier a par
	// ailleurs migré vers internal/sync/skill/ et ne contient plus aucun DELETE.
	// Entrée doublement morte → retirée.)
	//
	// L'allowlist est désormais vide : aucun DELETE brut toléré sur table append-only.
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
			if !dansLePerimetreART(path) {
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
			if !dansLePerimetreART(path) {
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

	// V4d (VF-6) : le même contrôle anti-pourrissement s'applique à
	// allowlistRawDelete. Une entrée survivante doit encore contenir un DELETE brut
	// sur une table protégée (sinon elle protège du code disparu — c'est exactement
	// le trou qu'a révélé VF-6 : skill_rating_postsync_persist.go n'existait plus).
	for fileRel, reason := range allowlistRawDelete {
		absPath := filepath.Join(repoRoot, filepath.FromSlash(fileRel))
		content, err := os.ReadFile(absPath)
		if err != nil {
			t.Errorf("allowlistRawDelete : fichier introuvable %q (raison: %q) — entrée à retirer ?", fileRel, reason)
			continue
		}
		text := stripGoComments(string(content))
		hasRawDelete := false
		for _, table := range tablesProtegees {
			delRegex := regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b`)
			if delRegex.MatchString(text) {
				hasRawDelete = true
				break
			}
		}
		if !hasRawDelete {
			t.Errorf("allowlistRawDelete obsolète : %q n'a plus de DELETE brut sur table protégée "+
				"dans son CODE (commentaires strippés) → retirer l'entrée (raison historique: %q)",
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
			if !dansLePerimetreART(path) {
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

// ─── D-B1 : les écritures à NOM DE TABLE INTERPOLÉ (G.2, 2026-09-13) ────────────────
//
// TROU COMBLÉ ICI. Les trois scans ci-dessus cherchent tous un nom de table
// LITTÉRAL dans la source. `internal/ops/seed_demo_corpus.go` construisait ses
// écritures par `fmt.Sprintf("UPDATE %s …", t.table)` : aucun littéral
// `UPDATE weapon_kills` n'existait dans le fichier, si bien que ni le scan
// principal ni le tripwire bulk ne pouvaient voir un UPDATE set-based nu sur
// quatre tables qu'ils couvrent pourtant (weapon_kills, medals_earned,
// killer_victim_pairs, match_participants). Le cas précis a été corrigé au lot
// B.3.6 ; le GARDE-RAIL, lui, restait aveugle à la forme (Découverte D-B1).
//
// CE QUE CE SCAN DÉCIDE. Une écriture à nom de table interpolé est SUSPECTE
// quand elle n'est pas ligne à ligne À VALEURS LIÉES — c'est-à-dire quand le
// littéral ne porte AUCUN placeholder `?`. Sans valeur liée, un `UPDATE %s SET …
// WHERE <prédicat>` est set-based : un seul statement qui touche N entrées
// d'index, exactement le déclencheur ART. Avec `?`, la boucle est un UPDATE par
// ligne (forme que le projet prescrit) — c'est la forme livrée par
// `seed_demo_corpus.go` et elle doit rester VERTE. Un `ON CONFLICT … DO UPDATE`
// dans un littéral interpolé est refusé même avec des valeurs liées : l'UPSERT
// est un déclencheur en soi.
//
// PÉRIMÈTRE. Le nom de table étant inconnu à la lecture, la corrélation est
// FILE-level, comme TestNoARTPatternsOnProtectedTables : seuls les fichiers qui
// nomment par ailleurs au moins une table protégée ou critique sont jugés. Un
// outil générique qui ne nomme aucune de ces tables (ex. `internal/ops/restore.go`,
// qui vide une table par `DELETE FROM %q` avec un nom venu du jeu de parquets)
// n'est donc PAS jugé ici — limite ASSUMÉE et consignée, pas un oubli.

// reInterpolatedUpdate / reInterpolatedDelete / reInterpolatedInsert — formes
// d'écriture dont le NOM DE TABLE est un verbe d'interpolation. `UPDATE` exige un
// `SET` dans la fenêtre : sans lui, le `"… UPDATE %q: %w"` d'un fmt.Errorf
// matcherait (faux positif mesuré sur mode_playlist_fr.go).
var (
	reInterpolatedUpdate = regexp.MustCompile(`(?is)\bUPDATE\s+%[sqv]\b.{0,400}?\bSET\b`)
	reInterpolatedDelete = regexp.MustCompile(`(?is)\bDELETE\s+FROM\s+%[sqv]`)
	reInterpolatedInsert = regexp.MustCompile(`(?is)\bINSERT\s+INTO\s+%[sqv]`)
	// Concaténation : le verbe est la FIN du littéral, immédiatement suivi de `+`.
	reConcatWrite = regexp.MustCompile("(?is)\\b(?:UPDATE|DELETE\\s+FROM|INSERT\\s+INTO)\\s*[\"`]\\s*\\+")
	// ON CONFLICT … DO UPDATE, fenêtre bornée au statement courant.
	reUpsertInWindow = regexp.MustCompile(`(?is)\bON\s+CONFLICT\b.{0,200}?\bDO\s+UPDATE\b`)
	// Littéraux Go : raw string entre backticks, ou string interprétée.
	reGoStringLiteral = regexp.MustCompile("(?s)`[^`]*`|\"(?:[^\"\\\\\n]|\\\\.)*\"")
)

// interpolatedWriteViolations rend les motifs suspects d'UN contenu source déjà
// déscommenté. Fonction pure : c'est elle que le témoin rouge exerce.
func interpolatedWriteViolations(text string) []string {
	var out []string
	for _, lit := range reGoStringLiteral.FindAllString(text, -1) {
		var forme string
		switch {
		case reInterpolatedUpdate.MatchString(lit):
			forme = "UPDATE <table interpolee>"
		case reInterpolatedDelete.MatchString(lit):
			forme = "DELETE FROM <table interpolee>"
		case reInterpolatedInsert.MatchString(lit):
			forme = "INSERT INTO <table interpolee>"
		default:
			continue
		}
		if reUpsertInWindow.MatchString(lit) {
			out = append(out, forme+" + ON CONFLICT DO UPDATE")
			continue
		}
		if !strings.Contains(lit, "?") {
			out = append(out, forme+" sans valeur liee (set-based)")
		}
	}
	for _, idx := range reConcatWrite.FindAllStringIndex(text, -1) {
		fin := idx[0] + 400
		if fin > len(text) {
			fin = len(text)
		}
		fenetre := text[idx[0]:fin]
		if reUpsertInWindow.MatchString(fenetre) {
			out = append(out, "ecriture concatenee + ON CONFLICT DO UPDATE")
			continue
		}
		if !strings.Contains(fenetre, "?") {
			out = append(out, "ecriture concatenee sans valeur liee (set-based)")
		}
	}
	return out
}

// dansLePerimetreART — périmètre commun des scans anti-ART : code de production
// uniquement (hors tests, migrations one-shot, CLI et scripts mono-processus).
// ops/ EST inclus (plomberie in-process, cf. E3 2026-07-03).
func dansLePerimetreART(path string) bool {
	if strings.HasSuffix(path, "_test.go") {
		return false
	}
	for _, seg := range []string{"migration", "migrations", "cmd", "scripts"} {
		if strings.Contains(path, "/"+seg+"/") || strings.Contains(path, "\\"+seg+"\\") {
			return false
		}
	}
	return true
}

// TestNoInterpolatedWriteOnProtectedTables — le scan de production de D-B1.
func TestNoInterpolatedWriteOnProtectedTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	tablesSurveillees := append(append([]string{}, tablesProtegees...), criticalMatchTables...)
	regexTables := make([]*regexp.Regexp, 0, len(tablesSurveillees))
	for _, table := range tablesSurveillees {
		regexTables = append(regexTables, regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(table)+`\b`))
	}

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
		if !strings.HasSuffix(path, ".go") || !dansLePerimetreART(path) {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := stripGoComments(string(content))
		nommeUneTable := false
		for _, re := range regexTables {
			if re.MatchString(text) {
				nommeUneTable = true
				break
			}
		}
		if !nommeUneTable {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		for _, v := range interpolatedWriteViolations(text) {
			violations = append(violations, v+" file="+filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("ecriture a NOM DE TABLE INTERPOLE sans valeur liee (declencheur ART, "+
			"invisible aux scans a nom litteral — cf. D-B1) : %d :\n  - %s\n"+
			"Remede : N statements ligne a ligne `WHERE <cle> = ?` (modele "+
			"internal/ops/seed_demo_corpus.go) ou INSERT-only via internal/persist.",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestInterpolatedWriteDetection_Sanity — LE TÉMOIN. Sans lui, un scan qui ne
// détecte plus rien passerait pour vert. Les chaînes ci-dessous vivent dans CE
// fichier de test : aucun fichier de production n'est fabriqué pour l'occasion.
func TestInterpolatedWriteDetection_Sanity(t *testing.T) {
	cas := []struct {
		nom    string
		source string
		rouge  bool
	}{
		{
			// VERBATIM : la forme que portait seed_demo_corpus.go avant B.3.6
			// (`git show 044751026^:apps/go-api/internal/ops/seed_demo_corpus.go`,
			// lignes 372-374). C'est elle que les trois autres scans ne voyaient pas.
			nom:    "UPDATE interpole set-based (la forme reelle de D-B1)",
			source: "stmt := fmt.Sprintf(`UPDATE %s SET %s FROM _xuid_map m WHERE %s.%s = m.old_xuid`, t.table, set, t.table, xuidCol)",
			rouge:  true,
		},
		{
			nom:    "UPDATE interpole ligne a ligne a valeurs liees (seed_demo_corpus)",
			source: "stmt := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = ?`, t.table, set, xuidCol)",
			rouge:  false,
		},
		{
			nom:    "DELETE FROM interpole",
			source: "q := fmt.Sprintf(`DELETE FROM %s WHERE match_id IN (SELECT match_id FROM tmp)`, table)",
			rouge:  true,
		},
		{
			nom:    "INSERT INTO interpole avec UPSERT malgre les valeurs liees",
			source: "q := fmt.Sprintf(`INSERT INTO %s (match_id, v) VALUES (?, ?) ON CONFLICT (match_id) DO UPDATE SET v = excluded.v`, table)",
			rouge:  true,
		},
		{
			// NB : pas de `written_at = now()` dans ce temoin — le ratchet
			// TestWrittenAtEcrituresEnUTC (internal/migration) scanne TOUT le module et
			// prendrait la chaine de test pour une horloge nue dans un ordre SQL reel.
			nom:    "concatenation set-based",
			source: `q := "UPDATE " + table + " SET playlist_group = 'h5_arena' WHERE playlist_group IS NULL"`,
			rouge:  true,
		},
		{
			nom:    "concatenation ligne a ligne a valeurs liees",
			source: `q := "UPDATE " + table + " SET v = ? WHERE match_id = ?"`,
			rouge:  false,
		},
		{
			nom:    "message d erreur portant UPDATE %q (faux positif a ne pas rendre)",
			source: "return fmt.Errorf(\"applyPlaylistFRSeeds UPDATE %q: %w\", seed.en, err)",
			rouge:  false,
		},
		{
			nom:    "INSERT INTO litteral (deja couvert par les autres scans)",
			source: "q := `INSERT INTO match_skill_rank (match_id) VALUES (?)`",
			rouge:  false,
		},
	}
	for _, c := range cas {
		got := interpolatedWriteViolations(c.source)
		if c.rouge && len(got) == 0 {
			t.Errorf("%s : le garde-rail NE MORD PAS (aucune violation rendue)", c.nom)
		}
		if !c.rouge && len(got) > 0 {
			t.Errorf("%s : faux positif — violations rendues : %v", c.nom, got)
		}
	}
}
