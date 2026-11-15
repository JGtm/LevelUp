//go:build cgo

// Package sync — shared_write_guard_test.go : garde-fou d'architecture.
//
// FIGE l'invariant d'orchestration des écritures (cf. revue 2026-06-01) : les
// tables "match-of-record" partagées (match_registry, match_participants,
// medals_earned, weapon_kills) ne doivent être écrites QUE depuis le package
// `internal/persist` (le système Collect→Persist qui orchestre les écritures
// INSERT-only) ou depuis une courte allowlist de sites legacy/post-complétion
// explicitement sanctionnés. Tout NOUVEAU writer direct hors de ce périmètre
// fait échouer le test → force une revue (doit-il passer par persist ?).
//
// Complète :
//   - combat_write_guard_test.go (killer_victim_pairs + highlight_events, pkg sync)
//   - no_art_patterns_test.go (patterns ART sur tables append-only)
//   - TestNoBulkMultiRowUpdateOnCriticalTables (forme bulk UPDATE FROM (VALUES))
//
// Modèle : no_art_patterns_test.go (scan AST-léger, comments strippés).
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// guardedSharedTables : tables partagées dont les écritures sont orchestrées.
// killer_victim_pairs / highlight_events sont gardées par combat_write_guard.
var guardedSharedTables = []string{
	"match_registry",
	"match_participants",
	"medals_earned",
	"weapon_kills",
}

// sharedWritePattern matche une écriture SQL (INSERT/UPDATE/DELETE) ciblant table.
func sharedWritePattern(table string) *regexp.Regexp {
	return regexp.MustCompile(`(?is)(INSERT[^;]*?INTO\s+` + regexp.QuoteMeta(table) +
		`\b|UPDATE\s+` + regexp.QuoteMeta(table) + `\b|DELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b)`)
}

// allowedSharedWriteFiles : sites NON-persist tolérés, par table → chemin relatif complet
// (depuis apps/go-api/) → raison. Le package internal/persist/ est autorisé en bloc (c'est la
// couche d'orchestration). Toute autre écriture doit être listée ici avec justification, sinon le
// test échoue. Chaque raison COMMENCE par la date de l'entrée, puis « (revu AAAA-MM-JJ) » : la
// dernière vérification sur pièces (TestAllowlistsEcritureDatees). Date de l'entrée = celle du premier
// commit où la clé apparaît : git log --reverse --format=%as -S'"<clé>"' -- <fichier de garde>.
//
// Garder cette liste COURTE : une entrée = une dette documentée. Quand un site est migré vers
// persist, retirer son entrée (le test n'exige PAS qu'une entrée corresponde à un write réel,
// mais une liste qui grossit signale une dérive).
var allowedSharedWriteFiles = map[string]map[string]string{
	"match_registry": {
		"internal/sync/killcollector/registry_flags.go": "2026-10-20 (premier commit de l'entrée, squash v8.0.0 ; déménagement daté 2026-09-01 par le texte d'origine ; revu 2026-11-12) : marquerRegistre, UPDATE backfill_completed (bitmask post-complétion), row par row sous le bail d'écriture dblease ; match_registry n'est pas append-only",
		"internal/sync/engagement.go":                   "2026-06-01 (revu 2026-11-12) : persistMatchIntensity, UPDATE match_intensity, row par row dans le burst d'écriture shared de l'engagement",
		"internal/sync/pve.go":                          "2026-06-01 (revu 2026-11-12) : MarkPveStatsDone, UPDATE backfill_completed (bit MBitPVEStats) post-complétion, sous le bail d'écriture",
		"internal/sync/events_replay.go":                "2026-06-01 (revu 2026-11-12) : clearEventsLoaded, UPDATE events_loaded et bit MBitEvents ; outil de rejeu des événements, hors flux primaire",
	},
	// match_participants / medals_earned : plus AUCUN writer direct dans sync/ —
	// le trio legacy InsertParticipants/InsertMedals a été supprimé en V4b
	// (import OpenSpartan routé via persist.SharedPersister depuis E1). Ces tables
	// ne sont donc écrites QUE par internal/persist/. Aucune entrée d'allowlist.
	"weapon_kills": {
		"internal/sync/writes.go": "2026-06-01 (revu 2026-11-12) : InsertWeaponKills, INSERT append-only par génération (weapon_kills_generation_seq, lue par la vue v_weapon_kills), plus aucun DELETE ; backfill film inline sous le bail d'écriture shared",
	},
}

// TestSharedMatchTablesWrittenViaPersistOrAllowlist échoue si une écriture
// directe sur une table match-of-record apparaît hors de internal/persist/ et
// hors allowlist. FIGE l'orchestration Collect→Persist contre les régressions.
func TestSharedMatchTablesWrittenViaPersistOrAllowlist(t *testing.T) {
	repoRoot := findRepoRoot(t)

	patterns := make(map[string]*regexp.Regexp, len(guardedSharedTables))
	for _, table := range guardedSharedTables {
		patterns[table] = sharedWritePattern(table)
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Exclure les one-shot mono-process (migrations, ops, CLI, scripts) — même
		// statut que dans no_art_patterns_test.go. NB : "/migrations/" (pluriel)
		// couvre les migrations title-owned (internal/games/*/migrations/, Phase 1.5 b23).
		if strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
			strings.Contains(path, "/migrations/") || strings.Contains(path, "\\migrations\\") ||
			strings.Contains(path, "/ops/") || strings.Contains(path, "\\ops\\") ||
			strings.Contains(path, "/cmd/") || strings.Contains(path, "\\cmd\\") ||
			strings.Contains(path, "/scripts/") || strings.Contains(path, "\\scripts\\") {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		rel = filepath.ToSlash(rel)
		// Le package d'orchestration est autorisé en bloc.
		if strings.HasPrefix(rel, "internal/persist/") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := stripGoComments(string(content))
		for table, re := range patterns {
			if re.MatchString(text) {
				if _, ok := allowedSharedWriteFiles[table][rel]; !ok {
					violations = append(violations,
						"table="+table+" file="+rel+" (hors internal/persist/ et hors allowlist)")
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("écriture(s) directe(s) sur une table match-of-record hors orchestration persist :\n  - %s\n"+
			"→ router via internal/persist (BatchBuilder/persisters) OU, si legacy sanctionné, ajouter à allowedSharedWriteFiles avec justification.",
			strings.Join(violations, "\n  - "))
	}
}

// regexEntreeDatee : une raison d'allowlist commence par une date AAAA-MM-JJ.
var regexEntreeDatee = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)

// TestAllowlistsEcritureDatees : toute entrée d'allowlist de ce fichier et de
// append_only_state_guard_test.go commence par une date (règle ART 3 : jamais d'allowlist sans
// justification datée).
func TestAllowlistsEcritureDatees(t *testing.T) {
	for table, files := range allowedSharedWriteFiles {
		for rel, raison := range files {
			if !regexEntreeDatee.MatchString(raison) {
				t.Errorf("allowedSharedWriteFiles[%s][%s] : la raison ne commence pas par une date AAAA-MM-JJ : %q", table, rel, raison)
			}
		}
	}
	for rel, raison := range allowlistMediaMutation {
		if !regexEntreeDatee.MatchString(raison) {
			t.Errorf("allowlistMediaMutation[%s] : la raison ne commence pas par une date AAAA-MM-JJ : %q", rel, raison)
		}
	}
	for _, e := range rawPMEReadAllowlist {
		if !regexEntreeDatee.MatchString(e.raison) {
			t.Errorf("rawPMEReadAllowlist[%q] : la raison ne commence pas par une date AAAA-MM-JJ : %q", e.motif, e.raison)
		}
	}
}
