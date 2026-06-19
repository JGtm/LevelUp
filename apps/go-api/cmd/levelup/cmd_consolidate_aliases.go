package main

// cmd_consolidate_aliases.go — sous-commande one-shot `levelup consolidate-aliases`.
//
// Merge la DB globale `xbox_aliases.duckdb` (mapping xuid→gamertag, store
// historique P5.3) dans `shared_matches_v2.xuid_aliases` — la table que lit
// réellement le résolveur (v_gamertag_lookup pour l'affichage,
// LookupXUIDByGamertag pour les coéquipiers, invariant I13). Élimine le 2e
// store redondant.
//
// Dédup STRICTE par xuid : `ON CONFLICT (xuid) DO NOTHING` (xuid est PK de
// shared.xuid_aliases) — aucun doublon possible, les xuids déjà présents côté
// shared sont conservés tels quels (shared = source vivante prioritaire).
//
// Idempotent. Pré-requis : serveur arrêté (lock DuckDB mono-writer sur shared).
// Après merge, la DB globale peut être supprimée (plus aucun lecteur).
//
// Usage :
//
//	levelup consolidate-aliases [--global-db <chemin>]

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

func runConsolidateAliases(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("consolidate-aliases", flag.ExitOnError)
	globalDB := fs.String("global-db", "", "Chemin de la DB globale xbox_aliases (défaut: data/global/xbox_aliases.duckdb)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	globalPath := *globalDB
	if globalPath == "" {
		globalPath = resolver.GlobalXuidAliasesDBPath()
	}
	if _, err := os.Stat(globalPath); err != nil {
		fmt.Printf("consolidate-aliases: DB globale absente (%s) — rien à merger.\n", globalPath)
		return nil
	}
	sharedPath := resolver.SharedDBPath(titlePkg.DefaultSlug)

	db, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrêté ?)", sharedPath, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx,
		fmt.Sprintf("ATTACH '%s' AS glb (READ_ONLY)", filepath.ToSlash(globalPath))); err != nil {
		return fmt.Errorf("attach global (%s): %w", globalPath, err)
	}

	var before int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM xuid_aliases").Scan(&before); err != nil {
		return fmt.Errorf("count before: %w", err)
	}

	res, err := db.ExecContext(ctx, `
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		SELECT g.xuid, g.gamertag, g.last_seen, 'global_xbox', now()
		FROM glb.xuid_aliases g
		WHERE g.gamertag IS NOT NULL AND g.gamertag != ''
		ON CONFLICT (xuid) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("merge insert: %w", err)
	}
	inserted, _ := res.RowsAffected()

	var after int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM xuid_aliases").Scan(&after); err != nil {
		return fmt.Errorf("count after: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}

	fmt.Printf("consolidate-aliases OK: shared.xuid_aliases %d -> %d (+%d depuis global, dédup par xuid)\n",
		before, after, inserted)
	fmt.Printf("  La DB globale %s peut maintenant être supprimée (plus aucun lecteur).\n", globalPath)
	return nil
}
