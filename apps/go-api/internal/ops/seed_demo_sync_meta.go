// Package ops — seed_demo_sync_meta.go : politique d'extraction de la table
// `sync_meta` vers la player DB de DÉMO (jeu de données publié dans le conteneur
// public levelup-demo).
//
// POLITIQUE : liste d'INCLUSION, défaut-refus.
//
// Jusqu'au 2026-08-26 l'extraction utilisait une liste d'EXCLUSION
// (`key NOT IN ('msal_token_cache')`). Elle laissait donc traverser
// `oauth_refresh_token` — le refresh token OAuth du joueur source — vers le jeu de
// données démo. Une liste d'exclusion est structurellement perdante à cet endroit :
// toute clé future traverse PAR DÉFAUT, credential compris, et rien n'oblige celui
// qui l'introduit à venir l'exclure ici. L'inclusion inverse la charge : une clé
// inconnue ne sort pas.
//
// Rappel ADR 0023 Phase 5 (2026-08-25) : les clés auth de `sync_meta` ne sont plus
// ni lues ni écrites par l'application, mais elles CONTIENNENT encore leurs
// dernières valeurs jusqu'au drop physique (recette ADR 0026, prochain rebuild).
// Un RT résiduel y est donc réel, pas théorique.
//
// Ajouter une clé ci-dessous = affirmer, avec justification datée, qu'elle n'est
// PAS un credential ET qu'elle est nécessaire à la démo.
// Garde-rails : seed_demo_sync_meta_guard_test.go.
package ops

import "strings"

// syncMetaKeyXUID : la clé `sync_meta` qui porte le xuid du joueur. Constante
// dédiée parce que le littéral "xuid" désigne AUSSI, ailleurs dans le package, une
// colonne SQL et un champ JSON de db_profiles — trois choses distinctes qu'il ne
// faut pas confondre au moment de toucher l'une d'elles.
const syncMetaKeyXUID = "xuid"

// demoSyncMetaAllowedKeys : les SEULES clés `sync_meta` recopiées dans la player DB
// démo. Tout le reste est laissé côté source.
var demoSyncMetaAllowedKeys = []string{
	// NÉCESSAIRE : duckdb.ResolveXUID (Q3ResolveXUID, platform/duckdb/pool.go) résout
	// le xuid du joueur depuis cette clé quand la config ne le porte pas.
	// extractPlayerTables la réécrit ensuite en DemoXUID : la valeur publiée est
	// l'identité démo, jamais celle du joueur source.
	syncMetaKeyXUID,

	// Sentinelles de migration de la player DB (booléens techniques, aucune donnée
	// joueur). NÉCESSAIRES : leur absence ferait REJOUER la migration correspondante
	// sur la base démo lors du applyMigrationsOnPath du seed. Or
	// `rebuild_career_progression` est un CLICHÉ FIGÉ du schéma — toute colonne qu'il
	// n'énumère pas est silencieusement perdue au rejeu (avertissement daté du
	// 2026-08-05 en tête de rebuildCareerProgression). On les fait donc voyager pour
	// que la démo reste dans l'état « déjà migrée » de la source, exactement comme
	// avant ce changement de politique.
	"career_progression_rebuilt_v1",    // migrations/steps_player_repairs.go : careerProgressionRebuildMetaKey
	"career_xp_total_default_fixed_v1", // migrations/steps_player.go : careerXPTotalFixMetaKey
}

// demoSyncMetaKeyAllowed dit si une clé `sync_meta` traverse vers la démo.
// Défaut : NON.
func demoSyncMetaKeyAllowed(key string) bool {
	for _, k := range demoSyncMetaAllowedKeys {
		if k == key {
			return true
		}
	}
	return false
}

// demoSyncMetaWhere construit la clause WHERE d'extraction de `sync_meta` À PARTIR
// de demoSyncMetaAllowedKeys — source unique, la clause n'est jamais écrite à la
// main (garde-rail TestDemoSyncMetaWhereIsInclusionBuiltFromAllowlist).
//
// Les clés sont des littéraux de ce fichier (jamais une entrée utilisateur), donc
// l'interpolation est sûre ; le garde-rail refuse de toute façon toute clé porteuse
// d'une apostrophe.
func demoSyncMetaWhere() string {
	quoted := make([]string, 0, len(demoSyncMetaAllowedKeys))
	for _, k := range demoSyncMetaAllowedKeys {
		quoted = append(quoted, "'"+k+"'")
	}
	return "key IN (" + strings.Join(quoted, ", ") + ")"
}
