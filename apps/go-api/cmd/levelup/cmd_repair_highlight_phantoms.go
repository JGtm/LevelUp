package main

// cmd_repair_highlight_phantoms.go — sous-commande `levelup repair-highlight-phantoms` : retire de
// `highlight_events` les lignes que l ancien lecteur des temps forts emettait en double sous un
// faux XUID, puis recalcule les couples `killer_victim_pairs` des matchs touches.
//
// SERVEUR ARRETE (ecrivain exclusif), sauvegarde octet pour octet avant toute ecriture, comme
// `compact-passes`. Jamais de DELETE sur `highlight_events` (index ART, ADR 0019) : la table est
// reconstruite sans les fantomes (migration.PurgeHighlightPhantoms). IDEMPOTENTE : sans fantome,
// rien n est ecrit. La suite de la sequence est `backfill-killsource` (docs/COMMANDS.md).
//
// Usage :
//
//	levelup repair-highlight-phantoms --dry-run       # lecture seule : lignes et matchs touches
//	levelup repair-highlight-phantoms                 # sauvegarde, purge, recalcul des couples
//	levelup repair-highlight-phantoms --title halo_infinite --backup-dir D:\sauvegardes

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"

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
	n, err := killcollector.RebuildPairsAfterPhantomPurge(ctx, db, rep.MatchIDs)
	fmt.Printf("couples recalcules : %d matchs\n", n)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint final: %w", err)
	}
	if len(rep.MatchIDs) > 0 {
		fmt.Printf("suite : levelup backfill-killsource --match %s --credit-only, puis sans --credit-only\n",
			strings.Join(prefixes8(rep.MatchIDs), ","))
	}
	return nil
}

func prefixes8(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id[:min(8, len(id))])
	}
	return out
}

func imprimerRapportFantomes(rep migration.HighlightPhantomReport) {
	statut := "detecte"
	if rep.Purged {
		statut = "retire"
	}
	fmt.Printf("lignes fantomes %s : %d, sur %d matchs\n", statut, rep.Rows, len(rep.MatchIDs))
}
