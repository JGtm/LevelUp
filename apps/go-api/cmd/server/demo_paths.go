// demo_paths.go — résolution des chemins de BOOT du serveur, en démo comme hors démo.
//
// Isolé de main.go pour être TESTABLE : la règle qu'il porte a été enfreinte
// pendant des mois sans que rien ne le signale (cf. demo_paths_test.go).
//
// La disposition de l'arbre démo elle-même vit dans title.DemoLayout (source unique,
// garde-rail internal/archlint/no_demo_layout_translation_test.go) : ce fichier ne fait
// que CHOISIR, au boot, entre elle et le PathResolver du dépôt.
package main

import (
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
)

// bootDBPaths : les quatre bases du warehouse ouvertes (et migrées) au boot.
type bootDBPaths struct {
	shared, metadata, sharedSocial, pve string
}

// resolveBootDBPaths rend les quatre bases du titre PAR DÉFAUT ouvertes au boot (les titres
// additionnels sont provisionnés à part, cf. provisionAdditionalActiveTitles).
//
// INVARIANT : en démo, la traduction vers la fixture est INCONDITIONNELLE. Elle ne dépend
// JAMAIS de l'existence du fichier de production ni de celle du fichier de fixture.
//
// L'implémentation d'avant le 2026-08-05 ne basculait sur la fixture que si la DB de
// production était ABSENTE. Sur toute machine possédant des données réelles — c'est-à-dire
// tout poste de développement — la démo servait donc la PRODUCTION : le joueur
// `demo-player` n'y ayant aucun match, les pages sortaient vides (healthz/home 503,
// `teammates … top_rows_count=0`) et le harnais visuel skippait 6 pages sur 7 en annonçant
// « données absentes ». Les migrations de boot s'appliquaient de surcroît aux bases de
// production, qu'un simple lancement de démo modifiait.
func resolveBootDBPaths(cfg *config.AppConfig, pr *title.PathResolver) bootDBPaths {
	slug := title.DefaultSlug
	if cfg.DemoMode {
		l := cfg.DemoLayout()
		return bootDBPaths{
			shared:       l.SharedDBPath(slug),
			metadata:     l.MetadataDBPath(slug),
			sharedSocial: l.SharedSocialDBPath(slug),
			pve:          l.SharedPVEDBPath(slug),
		}
	}
	return bootDBPaths{
		shared:       pr.SharedDBPath(slug),
		metadata:     pr.MetadataDBPath(slug),
		sharedSocial: pr.SharedSocialDBPath(slug),
		pve:          pr.SharedPVEDBPath(slug),
	}
}

// demoFixturePlayerDBPath rend la player DB du joueur démo principal, telle que `seed-demo`
// la produit (disposition IMBRIQUÉE `<démo>/players/DEMO/stats.duckdb`).
func demoFixturePlayerDBPath(cfg *config.AppConfig) string {
	return cfg.DemoLayout().PlayerDBPath(title.DefaultSlug, config.DemoRoster[0].Dir)
}

// bootLogsConfig rend la configuration des logs par module (et donc du crash log). En démo,
// sans LEVELUP_LOGS_DIR explicite, les fichiers vont sous `<démo>/runtime/logs` (lot B5,
// D-7) : le harnais visuel lance la démo sur le VRAI checkout, dont `logs/` recevait
// jusque-là les logs de la démo.
func bootLogsConfig(preliminaryRepoRoot string) logging.Config {
	logsCfg := logging.LoadConfig(preliminaryRepoRoot)
	if dir, ok := config.DemoLogsDir(); ok && logsCfg.LogsDir != "" {
		logsCfg.LogsDir = dir
	}
	return logsCfg
}
