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
// État réel des clés auth de `sync_meta` — NOTE HOTFIX (2026-08-26) : ce fichier est
// porté sur main AVANT la Phase 5 ADR 0023 (qui arrive avec v7.5.0). Sur ce main-ci,
// `oauth_refresh_token` est donc encore ÉCRIT à chaque rotation (double-write de
// compat) et lu par les fallbacks legacy : un RT FRAIS y est présent en permanence —
// raison d'être immédiate de ce verrou (la démo publique en recopiait un à chaque
// deploy). Après v7.5.0 : plus aucun lecteur de runtime sauf la migration one-shot du
// boot `migrateLegacyAuthTokensAtBoot` (kill-switch daté, retrait cible 2026-10-01),
// et les valeurs restent présentes jusqu'au drop physique des colonnes (recette
// ADR 0026, prochain rebuild). Dans les deux mondes : un RT résiduel y est réel, pas
// théorique.
//
// Ajouter une clé ci-dessous = affirmer, avec justification datée ET vérifiée sur
// pièces, qu'elle n'est PAS un credential ET qu'elle a un lecteur réel côté démo.
// Garde-rails : seed_demo_sync_meta_guard_test.go.
//
// PORTÉE : ce verrou couvre le chemin d'EXTRACTION (une player DB de prod → la démo),
// seul chemin par lequel une donnée réelle peut fuiter. Le seeder SYNTHÉTIQUE
// (seed_demo_synthetic_player.go) fabrique une base ex nihilo : aucune donnée de prod
// ne le traverse, donc aucun credential n'y est possible et il n'a pas besoin du même
// verrou. Il insère encore une ligne `sync_meta.xuid` que personne ne lit (cf. ci-
// dessous) — résidu de parité de schéma, signalé au journal, non traité ici.
package ops

import "strings"

// demoSyncMetaAllowedKeys : les SEULES clés `sync_meta` recopiées dans la player DB
// démo. Tout le reste est laissé côté source.
//
// La clé `xuid` N'Y EST PAS, et c'est délibéré (revue R1, 2026-08-26) : elle n'a
// aucun lecteur de production. `duckdb.ResolveXUID` (platform/duckdb/pool.go) et sa
// requête `Q3ResolveXUID` (platform/duckdb/queries.go) sont du code mort — vérifié
// sur pièces, leurs seuls appelants sont dans player_repos_test.go. Le xuid du joueur
// démo vient de la CONFIG écrite par writeDemoConfigsMulti (db_profiles.json), pas de
// la base. La faire voyager n'apportait rien et coûtait un risque : sa réécriture en
// DemoXUID était conditionnée à une égalité de valeur (`WHERE key='xuid' AND value=?`)
// dont personne ne vérifiait le nombre de lignes touchées — deux sources divergentes
// et le xuid RÉEL du joueur source était publié en silence. Clé retirée, réécriture
// devenue morte supprimée avec elle.
var demoSyncMetaAllowedKeys = []string{
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
