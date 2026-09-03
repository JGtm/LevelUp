//go:build cgo

// restore_best.go — mode `-restore-best` du job de capture, et garde-fou de qualité
// appliqué au chemin de scrape du CLI.
//
// Le problème que ça règle : la vue world_csr_leaderboard_latest sert le DERNIER lot
// capturé par (titre, saison, playlist). Un cycle dégradé (page Waypoint à moitié
// rendue) suffit donc à masquer un relevé sain — c'est arrivé le 2026-07-07, où
// 86 lignes sans aucun xuid ont recouvert 200 lignes intégralement identifiées. Rien
// n'est perdu (la table est append-only), mais plus rien ne sert le bon lot, et ces
// snapshots sont la SEULE archive : Halo Waypoint retire les saisons passées.
//
// Deux réponses, une seule règle (duckdb.DegradedBatchReason) :
//   - PRÉVENTION : le scrape du CLI refuse d'écrire un lot effondré, comme le cron ;
//   - RÉPARATION : -restore-best ré-INSÈRE le meilleur lot historique (append-only,
//     jamais de DELETE/UPDATE) avec un instant de capture frais, ce qui suffit à ce
//     que la vue le serve à nouveau.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// runRestoreMode exécute le balayage de restauration sur toute la base d'un titre.
// Ouvre la shared DB en RW (comme le reste du job : serveur API arrêté requis) même
// en dry-run — les migrations doivent être appliquées pour que la vue _latest existe.
func runRestoreMode(ctx context.Context, log *slog.Logger, dbPath, titleSlug string, execute bool) {
	db, err := openSharedRW(dbPath)
	if err != nil {
		fatal("open shared DB: %v", err)
	}
	defer db.Close()
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		fatal("migration shared: %v", err)
	}

	fmt.Printf("Restauration du meilleur lot — titre %s%s\n", titleSlug, dryRunSuffix(!execute))
	restored, alreadyBest, failed, err := restoreBestBatches(ctx, log, db, titleSlug, execute)
	if err != nil {
		fatal("restauration: %v", err)
	}

	verb := "à restaurer"
	if execute {
		verb = "restauré(s)"
	}
	log.InfoContext(ctx, "restore-best terminé", "titleSlug", titleSlug, "execute", execute,
		"restored", restored, "already_best", alreadyBest, "failed", failed)
	fmt.Printf("\nTerminé : %d couple(s) %s, %d déjà au meilleur, %d en échec.%s\n",
		restored, verb, alreadyBest, failed, executeHint(execute, restored))
}

// restoreBestBatches parcourt les couples (saison, playlist) et restaure ceux dont le
// lot SERVI serait refusé face au meilleur lot historique. Retourne les compteurs.
//
// Le verdict réutilise la règle du garde-fou de capture, prise à l'envers :
// DegradedBatchReason(référence = meilleur lot, candidat = lot servi). Si le lot servi
// aurait été refusé face au meilleur, c'est exactement qu'il ne mérite pas d'être
// servi. Une seule définition de « dégradé » pour la prévention et la réparation.
func restoreBestBatches(
	ctx context.Context, log *slog.Logger, db *sql.DB, titleSlug string, execute bool,
) (restored, alreadyBest, failed int, err error) {
	keys, err := duckdb.WorldCSRSeasonPlaylistPairs(ctx, db, titleSlug)
	if err != nil {
		return 0, 0, 0, err
	}
	if len(keys) == 0 {
		fmt.Println("  (aucun snapshot en base)")
		return 0, 0, 0, nil
	}
	for _, key := range keys {
		best, okBest, bErr := duckdb.WorldCSRBestBatch(ctx, db, key)
		served, okServed, sErr := duckdb.WorldCSRServedBatchStats(ctx, db, key.TitleSlug, key.SeasonID, key.PlaylistID)
		if bErr != nil || sErr != nil || !okBest || !okServed {
			failed++
			log.ErrorContext(ctx, "restore-best: lecture des lots impossible",
				"season", key.SeasonID, "playlist", key.PlaylistID, "best_err", bErr, "served_err", sErr)
			fmt.Printf("  %s / %s : ERREUR de lecture (couple ignoré)\n", key.SeasonID, key.PlaylistID)
			continue
		}
		reason := duckdb.DegradedBatchReason(best.Stats, served)
		if reason == "" {
			alreadyBest++
			fmt.Printf("  %s / %s : déjà au meilleur (%s)\n", key.SeasonID, key.PlaylistID, describe(served))
			continue
		}
		fmt.Printf("  %s / %s : servi %s  <-  meilleur %s du %s — %s\n",
			key.SeasonID, key.PlaylistID, describe(served), describe(best.Stats),
			best.FetchedAt.Format(time.RFC3339), reason)
		if !execute {
			restored++ // en dry-run, le compteur annonce ce qui SERAIT restauré
			continue
		}
		if rErr := restoreOne(ctx, db, key, best); rErr != nil {
			failed++
			log.ErrorContext(ctx, "restore-best: ré-insertion échouée",
				"season", key.SeasonID, "playlist", key.PlaylistID, "err", rErr)
			fmt.Printf("      -> ECHEC : %v\n", rErr)
			continue
		}
		restored++
		log.InfoContext(ctx, "restore-best: lot restauré", "season", key.SeasonID,
			"playlist", key.PlaylistID, "rows", best.Stats.Rows, "with_xuid", best.Stats.WithXUID,
			"source_fetched_at", best.FetchedAt)
		fmt.Printf("      -> restauré (%d lignes)\n", best.Stats.Rows)
	}
	return restored, alreadyBest, failed, nil
}

// restoreOne recopie le meilleur lot avec un instant de capture frais COMMUN à toutes
// ses lignes : c'est ce qui le rend « dernier » aux yeux de la vue _latest. INSERT
// pur (règle ART) — l'historique reste intact, y compris le lot dégradé.
func restoreOne(ctx context.Context, db *sql.DB, key duckdb.WorldCSRBatchKey, best duckdb.WorldCSRBatch) error {
	entries, err := duckdb.WorldCSRBatchEntries(ctx, db, key, best.FetchedAt)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("meilleur lot vide (fetched_at %s)", best.FetchedAt.Format(time.RFC3339))
	}
	now := time.Now().UTC()
	for i := range entries {
		entries[i].FetchedAt = now
	}
	_, err = duckdb.InsertWorldCSRSnapshot(ctx, db, key.TitleSlug, entries)
	return err
}

// cliBatchRefusalReason applique au scrape du CLI le MÊME garde-fou que le cron :
// un lot effondré face au lot servi n'est pas écrit. Chaîne vide = lot acceptable.
//
// FAIL-OPEN, comme le cron : si la qualité du lot servi est illisible, on laisse
// passer. Un problème de lecture ne doit jamais empêcher une capture — le risque
// inverse (bloquer toute écriture) est le plus coûteux.
func cliBatchRefusalReason(ctx context.Context, db *sql.DB, key duckdb.WorldCSRBatchKey, entries []domain.LeaderboardEntry) string {
	if db == nil || len(entries) == 0 {
		return ""
	}
	served, ok, err := duckdb.WorldCSRServedBatchStats(ctx, db, key.TitleSlug, key.SeasonID, key.PlaylistID)
	if err != nil {
		slog.Default().WarnContext(ctx, "qualité du lot servi illisible — garde-fou non appliqué (lot accepté)",
			"season", key.SeasonID, "playlist", key.PlaylistID, "err", err)
		return ""
	}
	if !ok {
		return "" // première capture de ce couple : rien à protéger.
	}
	return duckdb.DegradedBatchReason(served, duckdb.WorldCSRStatsOfEntries(entries))
}

// describe rend un lot lisible en une ligne de sortie CLI.
func describe(s duckdb.WorldCSRBatchStats) string {
	return fmt.Sprintf("%d lignes / %d xuid", s.Rows, s.WithXUID)
}

// executeHint rappelle comment passer à l'acte après un dry-run concluant.
func executeHint(execute bool, restored int) string {
	if execute || restored == 0 {
		return ""
	}
	return " Relancer avec -execute pour écrire."
}
