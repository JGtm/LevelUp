package main

// cmd_repair_highlight_phantoms.go — sous-commande `levelup repair-highlight-phantoms` : retire de
// `highlight_events` les lignes que l ancien lecteur des temps forts emettait en double sous un
// faux XUID, puis recalcule les couples `killer_victim_pairs` de TOUS les matchs dont les couples
// stockes different de l appariement courant (`killcollector.StalePairs`).
//
// SERVEUR ARRETE (ecrivain exclusif), sauvegarde octet pour octet avant toute ecriture, comme
// `compact-passes`. Jamais de DELETE : `highlight_events` est reconstruite sans les fantomes
// (`migration.PurgeHighlightPhantoms`) et `killer_victim_pairs` par echange de table
// (`migration.ReplaceKillerVictimPairs`), chacune en une transaction (index ART, ADR 0019).
// IDEMPOTENTE et REPRENABLE : la selection des couples est leur difference avec le calcul, donc
// une execution interrompue apres la purge reprend les matchs purges, et une seconde execution
// n ecrit rien. La suite de la sequence est `backfill-killsource` (docs/COMMANDS.md).
//
// Usage :
//
//	levelup repair-highlight-phantoms --dry-run       # lecture seule : fantomes et couples a recalculer
//	levelup repair-highlight-phantoms                 # sauvegarde, purge, recalcul des couples
//	levelup repair-highlight-phantoms --title halo_infinite --backup-dir D:\sauvegardes

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/killcollector"
)

func runRepairHighlightPhantoms(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("repair-highlight-phantoms", flag.ExitOnError)
	title := fs.String("title", titlePkg.DefaultSlug, "slug du titre")
	dryRun := fs.Bool("dry-run", false, "lecture seule : aucune ecriture, aucune sauvegarde")
	backupDir := fs.String("backup-dir", "", "dossier des sauvegardes (defaut : celui de la base)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reg := titlePkg.NewRegistryFromConfig(cfg.RepoRoot, slog.Default())
	if !reg.Exists(*title) {
		return fmt.Errorf("titre inconnu du registre : %q", *title)
	}
	path := titlePkg.NewPathResolver(cfg.RepoRoot, reg).SharedDBPath(*title)
	ctx := context.Background()
	if *dryRun {
		return detecterFantomes(ctx, path)
	}
	release, err := dblease.AcquireLease(path, compactLeaseTimeout)
	if err != nil {
		return fmt.Errorf("refus : bail d'ecrivain indisponible : %w", err)
	}
	defer release()
	return reparerFantomes(ctx, *title, path, *backupDir)
}

// detecterFantomes : le dry-run, en lecture seule (refuse si un autre processus tient la base).
func detecterFantomes(ctx context.Context, path string) error {
	handle, err := duckdb.OpenReadOnly(path)
	if err != nil {
		return fmt.Errorf("refus : %s est tenue par un autre processus (serveur arrete ?) : %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	rep, err := migration.DetectHighlightPhantoms(ctx, handle.SQLDb())
	imprimerRapportFantomes(rep)
	if err != nil {
		return err
	}
	st, err := killcollector.StalePairs(ctx, handle.SQLDb())
	fmt.Printf("couples differents de l appariement courant : %d matchs sur %d examines\n",
		len(st.MatchIDs), st.Examined)
	return err
}

// reparerFantomes : sauvegarde, purge de highlight_events, recalcul des couples, CHECKPOINT.
func reparerFantomes(ctx context.Context, slug, path, backupDir string) error {
	if err := preparerPourSauvegarde(ctx, slug, path); err != nil {
		return err
	}
	sauvegarde, err := copierFichier(ctx, path, backupDir, slug, "avant-fantomes")
	if err != nil {
		return fmt.Errorf("sauvegarde prealable impossible, rien n'est ecrit : %w", err)
	}
	fmt.Printf("sauvegarde : %s\n", sauvegarde)
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("refus : %s a ete prise par un autre processus apres la sauvegarde : %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	db := handle.SQLDb()
	rep, err := migration.PurgeHighlightPhantoms(ctx, db, false)
	imprimerRapportFantomes(rep)
	if err != nil {
		return err
	}
	if err := recalculerLesCouples(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint final: %w", err)
	}
	fmt.Println("suite : levelup backfill-killsource --credit-only, puis levelup backfill-killsource")
	return nil
}

// recalculerLesCouples : les couples qui different de l appariement courant, reecrits en un echange.
func recalculerLesCouples(ctx context.Context, db *sql.DB) error {
	st, err := killcollector.StalePairs(ctx, db)
	if err != nil {
		return err
	}
	parMatch := make(map[string][]migration.KillerVictimPair, len(st.Pairs))
	for id, ps := range st.Pairs {
		out := make([]migration.KillerVictimPair, 0, len(ps))
		for _, p := range ps {
			out = append(out, migration.KillerVictimPair{KillerXUID: p.KillerXUID, KillerGamertag: p.KillerGamertag,
				VictimXUID: p.VictimXUID, VictimGamertag: p.VictimGamertag, TimeMS: p.TimeMS})
		}
		parMatch[id] = out
	}
	if _, err := migration.ReplaceKillerVictimPairs(ctx, db, parMatch); err != nil {
		return err
	}
	fmt.Printf("couples recalcules : %d matchs sur %d examines\n", len(st.MatchIDs), st.Examined)
	return nil
}

func imprimerRapportFantomes(rep migration.HighlightPhantomReport) {
	statut := "detecte"
	if rep.Purged {
		statut = "retire"
	}
	fmt.Printf("lignes fantomes %s : %d, sur %d matchs\n", statut, rep.Rows, len(rep.MatchIDs))
}
