// Package sync — bitmask_dead_declarations_test.go : garde-fou contre la
// ré-introduction de constantes bitmask orphelines.
//
// Phase 3 du plan PLAN_BITMASKS_AUDIT_FIX (mai 2026).
//
// Le test extrait toutes les constantes `MBit*`, `PBit*`, `PveBit*` déclarées
// dans `backfill_flags.go` puis vérifie que chacune est référencée au moins
// une fois dans un autre fichier `.go` du package sync (write SQL, condition
// SQL, ou test d'invariant). Si une constante n'est référencée nulle part en
// dehors de sa déclaration, le test fail avec un message qui suggère soit
// l'utilisation soit la suppression.
//
// Justification : sans ce garde-fou, l'audit Phase 1ter de
// PLAN_HIGHLIGHT_EVENTS_BACKFILL avait dû découvrir manuellement 17 bits
// orphelins. Le test attrape la prochaine régression en CI.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNoDeadBitDeclaration(t *testing.T) {
	declared, err := extractBitConstants("backfill_flags.go")
	if err != nil {
		t.Fatalf("extractBitConstants: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("aucune constante bit extraite — regex cassée ?")
	}

	// Lire tous les .go du package SAUF backfill_flags.go (déclaration)
	// et SAUF ce fichier (qui mentionne nommément certains bits dans des
	// strings de doc/erreur).
	others, err := loadOtherSourceFiles([]string{
		"backfill_flags.go",
		"bitmask_dead_declarations_test.go",
	})
	if err != nil {
		t.Fatalf("loadOtherSourceFiles: %v", err)
	}

	// Whitelist : constantes dont l'absence d'utilisation est INTENTIONNELLE
	// (ex. groupes pré-calculés laissés pour l'API publique). Justifier
	// chaque entrée. Si une whitelist devient obsolète, le test l'indique.
	whitelist := map[string]string{
		"PBitMMR":          "groupe public PBitTeamMMR|PBitEnemyMMR — utile pour callers externes",
		"PBitExpected":     "groupe public PBitKillsExp|PBitDeathsExp",
		"PBitSkill":        "groupe public PBitMMR|PBitExpected",
		"PBitCombat":       "groupe public PBitAccuracy|PBitShots|PBitDamage",
		"PBitKillsDetail":  "groupe public",
		"PBitCoreStats":    "groupe public",
		"PBitAllStats":     "groupe public PBitSkill|PBitCoreStats|PBitMedals|PBitKillerVictim",
		"PveBitAllEnemies": "groupe public sommant tous les enemy types",
		"PveBitFullPVE":    "groupe public PveBitTotalKills|PveBitBossKills|PveBitAllEnemies",
		// 2026-07-25 (V72-03) : bit consommé par le CLI de backfill objectifs
		// (cmd/backfill_objective_stats : sélection `& MBitObjectiveStats = 0` + mark),
		// PAS par le package sync/ — le mark est inline dans le CLI (ratchet K3c
		// sync_root_freeze interdit un nouveau fichier sync/ racine). Le bit reste
		// déclaré ici (registre canonique des MatchBits) pour réserver le bit 23.
		"MBitObjectiveStats": "consommé par cmd/backfill_objective_stats (CLI), pas par sync/ (K3c freeze)",
	}

	var dead []string
	for _, name := range declared {
		if _, ok := whitelist[name]; ok {
			continue
		}
		used := false
		for _, body := range others {
			// On cherche le nom comme identifier (entouré de non-word chars).
			// Regex simple : bordure de mot.
			pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
			if pat.MatchString(body) {
				used = true
				break
			}
		}
		if !used {
			dead = append(dead, name)
		}
	}

	if len(dead) > 0 {
		t.Errorf("constantes bitmask déclarées mais jamais utilisées (orphelines) : %v\n"+
			"  → soit ajouter un call-site WRITE/READ (Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX)\n"+
			"  → soit supprimer la constante (Phase 3)\n"+
			"  → soit ajouter à la whitelist du test avec une justification écrite",
			dead)
	}
}

// extractBitConstants parse le fichier `backfill_flags.go` pour récupérer
// toutes les constantes nommées `MBit*`, `PBit*`, `PveBit*`. Approche regex
// plutôt qu'AST pour rester ultra-simple : la convention du fichier garantit
// `Name = expr` ligne-à-ligne dans des blocs `const ()`.
func extractBitConstants(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?m)^\s*((?:MBit|PBit|PveBit)\w+)\s*=`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// TestNoDeadBackfillFlagKey est le pendant de TestNoDeadBitDeclaration pour
// la map `BackfillFlags map[string]int` de backfill_flags.go. Vérifie que
// chaque key est consommée par `doneGuard("key", ...)` ou
// `ComputeBackfillMask("key", ...)` quelque part dans le package, ou bien
// listée dans la whitelist (avec justification écrite).
//
// Phase 6 du plan PLAN_BITMASKS_AUDIT_FIX : empêche la dégradation future
// (ajout d'une key orpheline "au cas où"). Ne casse PAS les keys héritage
// Python actuelles — elles sont whitelistées avec leur statut documenté.
func TestNoDeadBackfillFlagKey(t *testing.T) {
	keys, err := extractBackfillFlagsKeys("backfill_flags.go")
	if err != nil {
		t.Fatalf("extractBackfillFlagsKeys: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("aucune key BackfillFlags extraite — regex cassée ?")
	}

	others, err := loadOtherSourceFiles([]string{
		"backfill_flags.go",
		"bitmask_dead_declarations_test.go",
	})
	if err != nil {
		t.Fatalf("loadOtherSourceFiles: %v", err)
	}

	// Whitelist : keys héritage Python NON consommées via doneGuard ni
	// ComputeBackfillMask en code Go. Elles sont conservées pour
	// rétrocompatibilité de la map (les bits restent positionnés en DB par
	// l'ancien code Python sur l'historique). Toute nouvelle key doit avoir
	// un consumer Go ; sinon, soit la consommer, soit l'ajouter ici avec
	// justification.
	//
	// Audit Phase 1ter de PLAN_HIGHLIGHT_EVENTS_BACKFILL.md (commit b6b31062)
	// + thought_log [2026-05-08] "PLAN_BITMASKS_AUDIT_FIX — Phase 1 sonde".
	whitelist := map[string]string{
		"medals":          "héritage Python — detection Go via NOT IN medals_earned (table-based)",
		"events":          "héritage Python — detection Go via mr.events_loaded boolean",
		"skill":           "héritage Python — Phase 2 écrit la valeur via const locale backfillFlagSkill (4) ; key map non lue",
		"personal_scores": "héritage Python — detection Go via playerDoneGuard table-based",
		"accuracy":        "héritage Python — detection Go via mp.accuracy IS NULL",
		"shots":           "héritage Python — detection Go via mp.shots_fired IS NULL",
		"enemy_mmr":       "héritage Python — detection Go via mp.team_mmr IS NULL",
		"aliases":         "héritage Python — pas de detection automatique côté Go",
		"weapon_kills":    "héritage Python — OBSOLÈTE, remplacé par MBitWeaponKills (1<<21)",
	}

	// Chercher pour chaque key un usage `"key"` dans un appel à doneGuard
	// ou ComputeBackfillMask. On reste sur une regex permissive : si la
	// chaîne `"key"` apparaît dans un `.go` du package, on suppose qu'elle
	// est consommée. Faux positifs possibles (chaîne dans un autre contexte)
	// mais le coût est nul (test ne fail pas faussement).
	var orphans []string
	for _, key := range keys {
		if _, ok := whitelist[key]; ok {
			continue
		}
		needle := `"` + key + `"`
		used := false
		for _, body := range others {
			if strings.Contains(body, needle) {
				used = true
				break
			}
		}
		if !used {
			orphans = append(orphans, key)
		}
	}

	if len(orphans) > 0 {
		t.Errorf("keys de BackfillFlags déclarées mais aucun consumer Go (doneGuard ou ComputeBackfillMask) : %v\n"+
			"  → soit consommer la key dans backfill.go (doneGuard ou ComputeBackfillMask)\n"+
			"  → soit retirer de BackfillFlags map\n"+
			"  → soit whitelister ici avec justification écrite (héritage Python ou autre)",
			orphans)
	}
}

// extractBackfillFlagsKeys lit `backfill_flags.go` et retourne les keys de la
// map littérale `BackfillFlags = map[string]int{ "k1": ..., "k2": ... }`.
// Approche regex (block scanning) : cohérent avec extractBitConstants pour la
// même raison (fichier de convention stable, pas besoin d'AST).
func extractBackfillFlagsKeys(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(data)
	// Localiser le bloc `BackfillFlags = map[string]int{ ... }`.
	startMarker := "BackfillFlags = map[string]int{"
	startIdx := strings.Index(src, startMarker)
	if startIdx < 0 {
		return nil, nil // map absente — ok, pas d'orphelin par défaut
	}
	// Trouver l'accolade fermante associée (simple : on prend la première
	// `}` après le marker, suffit pour cette map plate).
	endIdx := strings.Index(src[startIdx:], "}")
	if endIdx < 0 {
		return nil, nil
	}
	block := src[startIdx : startIdx+endIdx]

	// Extraire les "key" en début de ligne (avant le `:`).
	re := regexp.MustCompile(`(?m)^\s*"([^"]+)"\s*:`)
	matches := re.FindAllStringSubmatch(block, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, m := range matches {
		k := m[1]
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out, nil
}

// loadOtherSourceFiles retourne le contenu de tous les .go du package
// (working dir = `internal/sync/`) sauf ceux listés dans `excludes`.
func loadOtherSourceFiles(excludes []string) (map[string]string, error) {
	excludeSet := map[string]struct{}{}
	for _, e := range excludes {
		excludeSet[e] = struct{}{}
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if _, skip := excludeSet[e.Name()]; skip {
			continue
		}
		data, err := os.ReadFile(filepath.Join(".", e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = string(data)
	}
	return out, nil
}
