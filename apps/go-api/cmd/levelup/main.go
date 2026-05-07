// cmd/levelup - CLI d'exploitation LevelUp.
//
// Sous-commandes disponibles :
//
//	levelup backup         --gamertag X [--output-dir D] [--compression-level N]
//	levelup restore        --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
//	levelup healthcheck    [--verbose]
//	levelup diagnose       --db PATH [--verbose]
//	levelup check-env
//	levelup archive        --gamertag X --xuid U --cutoff YYYY-MM-DD [--delete-after] [--dry-run]
//	levelup index-media    --gamertag X [--force-rescan] [--tolerance-min N]
//	levelup seed           career-ranks | citation-mappings | medals
//	levelup notify-version --version v1.2.3
//	levelup notify-sync    --gamertag X --op sync_delta --duration 120s [--matches N]
//	levelup compare-db     --go-db PATH --python-db PATH [--json]
//	levelup gate-check     [--gamertag X] [--json]
//	levelup surface-status [--json]
//	levelup sync-delta     (--gamertag X | --all) [--max-matches N] [--match-type T] [--rps N]
//	levelup sync-achievements (--gamertag X | --all) [--dry-run]
//	levelup add-title      --name "Nom du jeu" [--slug s] [--capabilities c1,c2] [--xbox-id X] [--steam-id S]
//
// Variables d'environnement : LEVELUP_REPO_ROOT (auto-detecte si absent).
//
// Implementation des sous-commandes :
//   - cmd_data.go    - backup, restore, archive, index-media, seed
//   - cmd_ops.go     - healthcheck, diagnose, check-env, compare-db, gate-check, surface-status
//   - cmd_sync.go    - sync-delta
//   - cmd_notify.go  - notify-version, notify-sync
//   - cmd_title.go   - add-title
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur config: %v\n", err)
		os.Exit(1)
	}

	subcmd := os.Args[1]
	args := os.Args[2:]

	var exitErr error
	switch subcmd {
	case "backup":
		exitErr = runBackup(cfg, args)
	case "restore":
		exitErr = runRestore(cfg, args)
	case "healthcheck":
		exitErr = runHealthcheck(cfg, args)
	case "diagnose":
		exitErr = runDiagnose(args)
	case "check-env":
		exitErr = runCheckEnv(cfg)
	case "archive":
		exitErr = runArchive(cfg, args)
	case "index-media":
		exitErr = runIndexMedia(cfg, args)
	case "seed":
		exitErr = runSeed(cfg, args)
	case "notify-version":
		exitErr = runNotifyVersion(cfg, args)
	case "notify-sync":
		exitErr = runNotifySync(cfg, args)
	case "compare-db":
		exitErr = runCompareDB(cfg, args)
	case "gate-check":
		exitErr = runGateCheck(cfg, args)
	case "surface-status":
		exitErr = runSurfaceStatus(cfg, args)
	case "sync-delta":
		exitErr = runSyncDelta(cfg, args)
	case "sync-achievements":
		exitErr = runSyncAchievements(cfg, args)
	case "backfill":
		exitErr = runBackfill(cfg, args)
	case "engagement-coefs":
		exitErr = runEngagementCoefs(cfg, args)
	case "recompute-friends":
		exitErr = runRecomputeFriends(cfg, args)
	case "migrate":
		exitErr = runMigrate(cfg, args)
	case "add-title":
		exitErr = runAddTitle(cfg, args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "sous-commande inconnue: %q\n", subcmd)
		printUsage()
		os.Exit(1)
	}

	if exitErr != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", exitErr)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`levelup - Outil d'exploitation LevelUp (backend Go)

Usage:
  levelup <commande> [options]

Commandes:
  backup          Sauvegarder une DB joueur en Parquet Zstd
  restore         Restaurer une DB joueur depuis un backup Parquet
  healthcheck     Diagnostic d'integrite des bases DuckDB
  diagnose        Inspecter le schema d'une DB DuckDB
  check-env       Verifier l'environnement et la configuration
  archive         Archiver les matchs anciens en Parquet
  index-media     Indexer et associer les medias au joueur
  seed            Peupler les referentiels metadata.duckdb
  notify-version  Envoyer une notification Discord de nouvelle version
  notify-sync     Envoyer une notification Discord de fin de sync (test/debug)
  compare-db      Comparer la parite Go vs Python (DB joueur)
  gate-check      Verifier la checklist Gate Phase 4
  surface-status  Afficher le backend actif par surface (feature flags)
	sync-delta      Lancer une sync delta pour un joueur ou pour tous les joueurs configures
  sync-achievements Lancer le backfill des achievements Xbox (admin one-shot, --dry-run dispo)
  backfill        Lancer un backfill local (Go-only, pas d'API) — voir --engagement-scores
  engagement-coefs Recompute des coefficients d'engagement (--with-scores pour rejouer aussi les scores) — bypasse les migrations
  recompute-friends Recompute is_with_friends sur toutes les player DBs (idempotent, --dry-run dispo)
  migrate         Migrer les donnees vers le namespace multi-titres
  add-title       Initialiser l'arborescence d'un nouveau titre de jeu

Options globales:
  LEVELUP_REPO_ROOT        Racine du repo (auto-detecte si absent)
  LEVELUP_NOTIFY_VERSIONS  Mettre a '1' pour activer les notifs de version en prod
  DISCORD_WEBHOOK_URL      URL webhook Discord (prevaut sur app_settings.json)

Aide par commande:
  levelup <commande> --help
`)
}
