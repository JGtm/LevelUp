package main

// cmd_backfill_is_ranked.go — sous-commande `levelup backfill-is-ranked`.
//
// ELLE RÉPARE match_registry.is_ranked DEPUIS LE CSR : tout match qui porte une ligne CSR par match
// (shared.match_csrs_latest) est classé (règle unique de classement, décision D-10 : le CSR n'est
// écrit que pour un RankRecap à InitialMeasurementMatches > 0). Même mécanique que
// backfill-registry-names : sync.BackfillRegistryIsRanked → persist.RegistryRankedPersister, un match
// à la fois, FAUX/NULL vers VRAI uniquement, jamais de déclassement.
//
// Idempotente : une seconde passe ne trouve plus de candidat.
//
// ORDRE D'EMPLOI : après `levelup backfill --shared-csr`, qui écrit le CSR selon la règle unique.
//
// SERVEUR ARRÊTÉ, Y COMPRIS POUR --dry-run (la base partagée est tenue en écriture par le serveur).
//
// ENCHAÎNEMENT COMPLET (l'engagement saute les matchs déjà calculés et dérive son mode de is_ranked ; le LUSR
// exclut les matchs classés) — printIsRankedSequence l'imprime à la fin d'une écriture :
//
//	backfill --all --shared-csr  ->  backfill --all --csr  ->  backfill-is-ranked
//	  ->  backfill --all --lusr  ->  backfill --all --engagement-scores --force
//
// Usage :
//
//	levelup backfill-is-ranked --dry-run   # comptes, aucune écriture
//	levelup backfill-is-ranked             # écriture

import (
	"context"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

func runBackfillIsRanked(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-is-ranked", flag.ExitOnError)
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "slug du titre")
	dryRun := fs.Bool("dry-run", false, "compter sans rien écrire (serveur arrêté, comme l'écriture)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(*titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("base introuvable (%s): %w", sharedPath, err)
	}
	ctx := context.Background()
	var (
		stats go_sync.RegistryRankedStats
		err   error
	)
	if *dryRun {
		sharedDB, release, oerr := duckdb.OpenReadForQuery(sharedPath)
		if oerr != nil {
			return erreurOuverture("open shared lecture", sharedPath, oerr)
		}
		defer release()
		stats, err = go_sync.BackfillRegistryIsRanked(ctx, sharedDB, go_sync.RegistryRankedOptions{DryRun: true})
	} else {
		handle, oerr := duckdb.OpenReadWrite(sharedPath)
		if oerr != nil {
			return erreurOuverture("open shared RW", sharedPath, oerr)
		}
		defer handle.Close()
		stats, err = go_sync.BackfillRegistryIsRanked(ctx, handle.SQLDb(), go_sync.RegistryRankedOptions{})
	}
	if err != nil {
		return err
	}
	printRegistryRankedStats(stats)
	if !stats.DryRun {
		printIsRankedSequence()
	}
	if stats.Errors > 0 {
		return fmt.Errorf("%w : %d matchs", errConvergenceIncomplete, stats.Errors)
	}
	return nil
}

// printRegistryRankedStats imprime les comptes du rattrapage.
func printRegistryRankedStats(s go_sync.RegistryRankedStats) {
	verbe, mode := "corrigés", "écriture"
	if s.DryRun {
		verbe, mode = "corrigeables", "simulation"
	}
	fmt.Printf("is_ranked depuis le CSR (%s) :\n", mode)
	fmt.Printf("  matchs à CSR non classés, %s : %d (candidats %d)\n", verbe, s.Fixed, s.Candidates)
	fmt.Printf("  matchs à CSR déjà classés : %d\n", s.AlreadyRanked)
	fmt.Printf("  matchs classés sans aucun CSR (jamais déclassés, à rattraper par backfill --shared-csr) : %d\n", s.RankedWithoutCSR)
	fmt.Printf("  échecs d'écriture : %d\n", s.Errors)
}

// printIsRankedSequence rappelle les recalculs qui suivent une réparation d'is_ranked : sans eux, le LUSR
// garde les matchs classés dans son calcul et l'engagement (qui saute les matchs déjà calculés) reste
// dans l'ancien mode.
func printIsRankedSequence() {
	fmt.Println("suite obligatoire (serveur arrêté) :")
	fmt.Println("  1. levelup backfill --all --shared-csr   puis   levelup backfill --all --csr   (avant cette commande)")
	fmt.Println("  2. levelup backfill --all --lusr                      (recalcul LUSR v2 canonique, exclut les matchs classés)")
	fmt.Println("  3. levelup backfill --all --engagement-scores --force (mode d'engagement dérivé de is_ranked)")
}
