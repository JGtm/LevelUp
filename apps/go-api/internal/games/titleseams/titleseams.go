// Package titleseams — point de câblage UNIQUE des seams title-owned.
//
// Pourquoi ce paquet (2026-09-16) : les seams title-owned (provider des étapes de
// migration, racine des jalons Halo 5, traductions de rangs, classifiers LUSR et
// famille objectif) étaient posés à la main dans `cmd/server/main.go` et nulle part
// ailleurs. Conséquence mesurée sur un `sync-full` CLI : `post-sync: PANIC récupéré …
// classifier LUSR non câblé` (fail-loud MT-15, `internal/sync/skill/skill_chain_provider.go`)
// → `perf_scores=0 lusr=0 citations=0 dominance=0` sur toute la passe. Tout binaire
// `package main` qui importe `internal/sync` (ou `internal/sync/skill`) a ce trou.
//
// Qui doit l'appeler : TOUT `package main` de `cmd/` qui importe `internal/sync` ou
// `internal/sync/skill`, au démarrage, avant tout appel au moteur de sync ou aux
// migrations. Garde-rail : `internal/archlint/titleseams_wired_test.go` (allowlist vide).
//
// Ce paquet ne contient AUCUNE logique : uniquement les appels `Set*` / `Register()`,
// dans l'ordre du bloc serveur d'origine.
package titleseams

import (
	"path/filepath"

	halo5 "levelup/go-api/internal/games/halo_5"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	syncpkg "levelup/go-api/internal/sync"
)

// RegisterAll pose les huit seams title-owned. Idempotent (chaque appel écrase la
// même variable de paquet par la même valeur) et sans effet de bord DB.
//
// prestigeConfigDir : `config/titles/{slug}` du titre par défaut — utilisé UNIQUEMENT
// pour la racine du seed des jalons Halo 5 (son parent, `config/titles/`). Vide =
// seed des jalons h5 non câblé (même garde que le serveur) ; les autres seams sont
// posés dans tous les cas. Utiliser PrestigeConfigDir pour le calculer depuis le
// repoRoot.
func RegisterAll(prestigeConfigDir string) {
	// Phase 1.5.1 B (ADR 0025) : étapes de migration title-owned (Halo Infinite)
	// auprès du runner, avant tout RunForDB.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// ROOT FIX assets Halo 5 : le set h5 possède SON milestone_catalog (schéma +
	// seed) — la racine config/titles/ doit être injectée AVANT Register pour que le
	// seed h5 trouve config/titles/halo_5/milestones/catalog.toml.
	if prestigeConfigDir != "" {
		halo5migrations.SetMilestonesSeedRoot(filepath.Dir(prestigeConfigDir))
	}
	halo5migrations.Register()
	// MT-07 : source title-owned des libellés de rangs de carrière (seed offline).
	migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)
	// MT-15 : classifier LUSR title-owned (pair_name → chaîne TrueSkill). GetLUSRChain
	// panique si non posé (fail-loud) — protège le chemin de scoring live.
	syncpkg.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	// MT-15+ : classifier LUSR title-aware pour Halo 5 (pas de pair_name → chaîne
	// unique h5_arena). Sans ça, h5 collapserait tous ses modes dans arena_slayer.
	syncpkg.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)
	// Scission ranked par famille (D-A) : classifier title-owned de la famille
	// objectif, consommé par GetPerformanceChain. h5 n'a pas de pair_name → classifier
	// dédié qui répond false (tout son classé va en ranked_slayer).
	syncpkg.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	syncpkg.SetObjectiveFamilyClassifierForTitle(halo5.TitleSlug, halo5.IsObjectiveSubMode)
}

// PrestigeConfigDir reproduit le calcul du serveur (`cmd/server/main.go`) :
// `<repoRoot>/config/titles/{slug par défaut}`. Source unique pour que les CLI
// n'en gardent pas chacune une copie littérale.
func PrestigeConfigDir(repoRoot string) string {
	if repoRoot == "" {
		return ""
	}
	return filepath.Join(repoRoot, "config", "titles", migration.DefaultSlug)
}
