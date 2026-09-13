//go:build cgo

// purge_foreign_lusr_chain — retire d'UNE player DB les lignes match_skill_rank
// portant une chaîne LUSR étrangère au titre de cette base.
//
// Pourquoi cet outil existe. Le 2026-06-26, le pipeline de sync V2 a écrit la chaîne
// Halo 5 `h5_arena` dans les player DB halo_infinite de 4 joueurs (913 / 1 064 / 471 /
// 31 lignes, ×2 rating_type) — double source de titre au câblage, cf.
// .ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md. La réparation d'août a AJOUTÉ les
// lignes correctes (append-only : rien n'est jamais supprimé), elle n'a pas retiré les
// fautives. Elles restent dans la table brute, et deux d'entre elles (matchs non
// rejouables) gagnent encore l'arbitrage de match_skill_rank_latest.
//
// JAMAIS DE DELETE. `DELETE FROM match_skill_rank WHERE playlist_group='h5_arena'` sur
// une table indexée est EXACTEMENT le vecteur du bug DuckDB ART #23046 qui a mis des
// bases en FATAL en prod (règle CLAUDE.md n°1, ADR 0026). La purge se fait par
// RECONSTRUCTION CTAS transactionnelle, modelée sur migration/append_only_rebuild.go
// (rebuildAppendOnlyTx) : BEGIN, CTAS filtré, garde de cardinalité AVANT le DROP,
// DROP+RENAME, PK/séquence/défauts/index/vue reposés, COMMIT, CHECKPOINT.
//
// SERVEUR ARRÊTÉ OBLIGATOIRE (modèle mono-process, ADR 0013) : l'outil ouvre la base en
// RW exclusif. Sauvegarde préalable de la base à la charge de l'opérateur.
//
// Usage (depuis apps/go-api), une base à la fois :
//
//	go run -tags cgo ./cmd/purge_foreign_lusr_chain -db ../../data/titles/halo_infinite/players/JGtm/stats.duckdb
//	go run -tags cgo ./cmd/purge_foreign_lusr_chain -db ../../data/titles/halo_infinite/players/JGtm/stats.duckdb -commit
//
// -dry-run vaut true par défaut : sans -commit, l'outil ne fait que recenser.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := flag.String("db", "", "chemin de la player DB (stats.duckdb) — OBLIGATOIRE, une base à la fois")
	chain := flag.String("chain", "h5_arena", "chaîne LUSR étrangère à retirer (playlist_group)")
	dryRun := flag.Bool("dry-run", true, "recensement seul (défaut) ; -commit pour écrire")
	commit := flag.Bool("commit", false, "exécute réellement la reconstruction (serveur ARRÊTÉ, sauvegarde faite)")
	flag.Parse()

	ctx := context.Background()
	if err := run(ctx, *dbPath, *chain, *dryRun && !*commit, *commit); err != nil {
		slog.ErrorContext(ctx, "purge_foreign_lusr_chain: échec", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dbPath, chain string, dryRun, commit bool) error {
	if dbPath == "" {
		return fmt.Errorf("-db est obligatoire (une player DB à la fois)")
	}
	if chain == "" {
		return fmt.Errorf("-chain vide : refus (purger une chaîne non nommée retirerait des lignes saines)")
	}
	if !commit && !dryRun {
		return fmt.Errorf("mode indéterminé : utiliser -dry-run (défaut) ou -commit")
	}

	// RW exclusif : le serveur DOIT être arrêté (modèle mono-process, ADR 0013).
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("ouverture RW de %s (le serveur est-il arrêté ?): %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()

	before, err := censusForeignChain(ctx, db, chain)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: recensement AVANT",
		"db", dbPath, "chain", chain,
		"rows_total", before.TotalRows, "rows_foreign_raw", before.ForeignRaw,
		"rows_foreign_by_rating_type", before.ForeignByRatingType,
		"rows_foreign_latest", before.ForeignLatest)

	if !commit {
		slog.InfoContext(ctx, "purge_foreign_lusr_chain: DRY-RUN — aucune écriture (relancer avec -commit)",
			"db", dbPath, "chain", chain)
		return nil
	}
	if before.ForeignRaw == 0 {
		slog.InfoContext(ctx, "purge_foreign_lusr_chain: rien à purger, base déjà saine",
			"db", dbPath, "chain", chain)
		return nil
	}

	if err := purgeForeignChain(ctx, db, chain, before); err != nil {
		return err
	}

	after, err := censusForeignChain(ctx, db, chain)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: recensement APRÈS",
		"db", dbPath, "chain", chain,
		"rows_total", after.TotalRows, "rows_foreign_raw", after.ForeignRaw,
		"rows_foreign_latest", after.ForeignLatest)
	if after.ForeignRaw != 0 {
		return fmt.Errorf("purge incomplète : %d ligne(s) %q subsistent", after.ForeignRaw, chain)
	}
	return nil
}
