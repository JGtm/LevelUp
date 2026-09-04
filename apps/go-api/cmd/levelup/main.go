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
//	levelup seed           career-ranks | citation-mappings | medals | rank-translations
//	levelup notify-version --version v1.2.3
//	levelup notify-sync    --gamertag X --op sync_delta --duration 120s [--matches N]
//	levelup compare-db     --go-db PATH --python-db PATH [--json]
//	levelup gate-check     [--gamertag X] [--json]
//	levelup sync-delta     (--gamertag X | --all) [--max-matches N] [--match-type T] [--rps N]
//	levelup sync-full      (--gamertag X | --all) [--max-matches N] [--match-type T] [--rps N]
//	levelup sync-achievements (--gamertag X | --all) [--dry-run]
//	levelup add-title      --name "Nom du jeu" [--slug s] [--capabilities c1,c2] [--xbox-id X] [--steam-id S]
//	levelup populate-assets [--types map,playlist] [--langs fr-FR] [--dry-run] [--force] [--title-id slug]
//
// Variables d'environnement : LEVELUP_REPO_ROOT (auto-detecte si absent).
//
// Implementation des sous-commandes :
//   - cmd_data.go    - backup, restore, archive, index-media, seed
//   - cmd_ops.go     - healthcheck, diagnose, check-env, compare-db, gate-check
//   - cmd_sync.go    - sync-delta, sync-full
//   - cmd_notify.go  - notify-version, notify-sync
//   - cmd_title.go   - add-title
//   - cmd_populate_assets.go - populate-assets (traductions d'assets Discovery UGC)
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
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

	// MT-07 : source title-owned des libellés de rangs (le sous-commande
	// `seed rank-translations` via ops.SeedRankTranslations en dépend).
	migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)

	// Steps de migration title-owned (parité cmd/server). SANS ça, les RACINES
	// shared_social (create_base_shared_social_schema → table media_files / associations)
	// ne sont PAS exécutées par RunForDB/RunForTitleDB dans la CLI → seed-demo média
	// échoue (media_files absente). index-media et seed-demo en dépendent.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

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
	case "seed-demo":
		exitErr = runSeedDemo(cfg, args)
	case "notify-version":
		exitErr = runNotifyVersion(cfg, args)
	case "notify-sync":
		exitErr = runNotifySync(cfg, args)
	case "compare-db":
		exitErr = runCompareDB(cfg, args)
	case "gate-check":
		exitErr = runGateCheck(cfg, args)
	case "sync-delta":
		exitErr = runSyncDelta(cfg, args)
	case "sync-full":
		exitErr = runSyncFull(cfg, args)
	case "sync-achievements":
		exitErr = runSyncAchievements(cfg, args)
	case "backfill":
		exitErr = runBackfill(cfg, args)
	case "replay-events":
		exitErr = runReplayEvents(cfg, args)
	case "reset-bitmasks":
		exitErr = runResetBitmasks(cfg, args)
	case "engagement-coefs":
		exitErr = runEngagementCoefs(cfg, args)
	case "rebuild-pme-art":
		exitErr = runRebuildPME(cfg, args)
	case "consolidate-aliases":
		exitErr = runConsolidateAliases(cfg, args)
	case "recompute-friends":
		exitErr = runRecomputeFriends(cfg, args)
	case "backfill-squad-creators":
		exitErr = runBackfillSquadCreators(cfg, args)
	case "backfill-h5-kill-mechanics":
		exitErr = runBackfillH5KillMechanics(cfg, args)
	case "backfill-killsource":
		exitErr = runBackfillKillSource(cfg, args)
	case "backfill-medailles-feed":
		exitErr = runBackfillMedaillesFeed(cfg, args)
	case "archive-films":
		exitErr = runArchiveFilms(cfg, args)
	case "backfill-replay":
		exitErr = runBackfillReplay(cfg, args)
	case "backfill-usage-summary":
		exitErr = runBackfillUsageSummary(cfg, args)
	case "migrate":
		exitErr = runMigrate(cfg, args)
	case "restore-csr":
		exitErr = runRestoreCSR(cfg, args)
	case "add-title":
		exitErr = runAddTitle(cfg, args)
	case "populate-assets":
		exitErr = runPopulateAssets(cfg, args)
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
  seed-demo       Generer les donnees demo (data/demo/) depuis un joueur source anonymise
  notify-version  Envoyer une notification Discord de nouvelle version
  notify-sync     Envoyer une notification Discord de fin de sync (test/debug)
  compare-db      Comparer la parite Go vs Python (DB joueur)
  gate-check      Verifier la checklist Gate Phase 4
  sync-delta      Lancer une sync delta pour un joueur ou pour tous les joueurs configures
  sync-full       Parcourir les N derniers matchs API et insérer les manquants (comble les trous)
  sync-achievements Lancer le backfill des achievements Xbox (admin one-shot, --dry-run dispo)
  backfill        Lancer un backfill local (Go-only, pas d'API) — voir --engagement-scores
  replay-events   Re-parse highlight events sur les matchs cassés (parser bit-aligné fix mai 2026)
  reset-bitmasks  Reset rétroactif skill/participants/PVE bits (Phase 4 PLAN_BITMASKS_AUDIT_FIX)
  engagement-coefs Recompute des coefficients d'engagement (--with-scores pour rejouer aussi les scores) — bypasse les migrations
  rebuild-pme-art  Reconstruit l'index ART de player_match_enrichment (--all|--gamertag) — anti-corruption DuckDB 1.5.x, serveur arrêté
  consolidate-aliases  Merge la DB globale xbox_aliases dans shared.xuid_aliases (dédup par xuid) — serveur arrêté
  recompute-friends Recompute is_with_friends sur toutes les player DBs (idempotent, --dry-run dispo)
  backfill-squad-creators Réinscrit le créateur manquant dans les escouades legacy (append-only, idempotent, --dry-run dispo, serveur arrêté)
  backfill-h5-kill-mechanics Corrige les mécaniques de kill H5 (assassination/ground_pound/shoulder_bash) écrites à 0 avant l'activation du mapper (re-fetch carnage, UPDATE ciblé, --dry-run dispo, serveur arrêté)
  backfill-killsource Remplit match_kill_events + match_weapon_shots : décodage HORS LIGNE des films en cache (gros films en dernier, reprenable par decoder_rev) puis producteur credit-seul depuis highlight_events (--dry-run, --limit, --films-only, --credit-only, serveur arrêté). --online --gamertag <GT> va CHERCHER les films absents du cache et les y archive : c'est ce qui rattrape l'attribution des assistances, sans film il n'y en a aucune
  backfill-medailles-feed Rend leur nom aux médailles déjà en base : relit HORS LIGNE le chunk highlight des films en cache, apparie par (xuid, time_ms) et remplit highlight_events.raw_json + type_hint, restés vides depuis avril 2026 (415 matchs / 22 031 events sans identité, donc aucune médaille au fil des éliminations). Film absent du cache = match consigné et sauté (--dry-run, --limit, --cache, serveur arrêté)
  archive-films   Télécharge et CONSERVE les films manquants du cache local (manifeste + chunks complets, sans aucun décodage). Les films EXPIRENT côté 343 et ne se retéléchargent jamais : cette passe est la seule qui les sauve. Lecture seule sur la base (elle ne peut rien corrompre) mais SERVEUR ARRÊTÉ quand même : DuckDB n'autorise qu'un processus par fichier (--dry-run, --limit, --gamertag, --sauter-marques)
  backfill-replay Construit les artefacts de rejeu 2D de tous les films en cache : décodage HORS LIGNE via la librairie replaybuild, UN PROCESSUS PAR FILM (un film-bombe n'emporte plus la passe ni la machine ; gros films en dernier, reprenable par SchemaVersion, échecs ventilés : carte hors catalogue, mémoire, mort subite) (--dry-run, --limit, --force, --only-existing, --mem-limit-gib)
  backfill-usage-summary Résume les artefacts de rejeu déjà cuits en usages d'équipement et de socles (match_usage_players + match_usage_films, append-only) : AUCUN décodage de film, un artefact lu à la fois, reprenable par (summary_rev, artifact_schema) via la vue _latest. --dry-run imprime les compteurs par match (prises nommées/anonymes, bonus par famille) pour les contrôles croisés (--force, --match, --limit, serveur arrêté)
  migrate         Migrer les donnees vers le namespace multi-titres
  restore-csr     Restaurer les CSR historiques depuis un backup DuckDB legacy (--gamertag X --backup PATH [--dry-run] [--mode preserve|overwrite])
  add-title       Initialiser l'arborescence d'un nouveau titre de jeu
  populate-assets Peupler asset_translations (noms localises des assets via Discovery UGC)

Options globales:
  LEVELUP_REPO_ROOT        Racine du repo (auto-detecte si absent)
  LEVELUP_NOTIFY_VERSIONS  Mettre a '1' pour activer les notifs de version en prod
  DISCORD_WEBHOOK_URL      URL webhook Discord (prevaut sur app_settings.json)

Aide par commande:
  levelup <commande> --help
`)
}
