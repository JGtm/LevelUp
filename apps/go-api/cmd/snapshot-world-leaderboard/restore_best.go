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

// restoreOptions rassemble les réglages du mode réparation. `season` est OBLIGATOIRE
// et désigne UNE saison précise : une restauration écrit dans la seule archive du
// classement mondial, elle ne doit jamais s'appliquer à un périmètre qu'on n'a pas
// nommé (cf. validation dans main.go).
type restoreOptions struct {
	sharedDBPath string
	titleSlug    string
	season       string
	execute      bool
}

// runRestoreMode exécute le balayage de restauration sur UNE saison d'un titre.
// Ouvre la shared DB en RW (comme le reste du job : serveur API arrêté requis) même
// en dry-run — les migrations doivent être appliquées pour que la vue _latest existe.
func runRestoreMode(ctx context.Context, log *slog.Logger, opt restoreOptions) {
	db, err := openSharedRW(opt.sharedDBPath)
	if err != nil {
		fatal("open shared DB: %v", err)
	}
	defer db.Close()
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		fatal("migration shared: %v", err)
	}

	fmt.Printf("Restauration du meilleur lot — titre %s, saison %s%s\n",
		opt.titleSlug, opt.season, dryRunSuffix(!opt.execute))
	restored, alreadyBest, failed, err := restoreBestBatches(ctx, log, db, opt)
	if err != nil {
		fatal("restauration: %v", err)
	}

	verb := "à restaurer"
	if opt.execute {
		verb = "restauré(s)"
	}
	log.InfoContext(ctx, "restore-best terminé", "titleSlug", opt.titleSlug, "season", opt.season,
		"execute", opt.execute, "restored", restored, "already_best", alreadyBest, "failed", failed)
	fmt.Printf("\nTerminé : %d couple(s) %s, %d déjà au meilleur, %d en échec.%s\n",
		restored, verb, alreadyBest, failed, executeHint(opt.execute, restored))
}

// restoreBestBatches parcourt les playlists de la saison demandée et restaure celles
// dont le lot SERVI serait refusé face au meilleur lot historique. Retourne les
// compteurs. Le périmètre est strictement borné à opt.season : les autres saisons ne
// sont ni lues ni touchées.
//
// Le verdict réutilise la règle du garde-fou de capture, prise à l'envers :
// DegradedBatchReason(référence = meilleur lot, candidat = lot servi). Si le lot servi
// aurait été refusé face au meilleur, c'est exactement qu'il ne mérite pas d'être
// servi. Une seule définition de « dégradé » pour la prévention et la réparation.
//
// Profondeur : on passe 0 (aucun plafond) — le but de la réparation est justement de
// servir l'archive la plus riche, sans borner la référence à une profondeur de cycle.
func restoreBestBatches(
	ctx context.Context, log *slog.Logger, db *sql.DB, opt restoreOptions,
) (restored, alreadyBest, failed int, err error) {
	all, err := duckdb.WorldCSRSeasonPlaylistPairs(ctx, db, opt.titleSlug)
	if err != nil {
		return 0, 0, 0, err
	}
	keys := make([]duckdb.WorldCSRBatchKey, 0, len(all))
	for _, k := range all {
		if k.SeasonID == opt.season {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		fmt.Printf("  (aucun snapshot pour la saison %s)\n", opt.season)
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
		reason := duckdb.DegradedBatchReason(best.Stats, served, 0)
		if reason == "" {
			alreadyBest++
			fmt.Printf("  %s / %s : déjà au meilleur (%s)\n", key.SeasonID, key.PlaylistID, describe(served))
			continue
		}
		fmt.Printf("  %s / %s : servi %s  <-  meilleur %s du %s — %s\n",
			key.SeasonID, key.PlaylistID, describe(served), describe(best.Stats),
			best.FetchedAt.Format(time.RFC3339), reason)
		if !opt.execute {
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
// depthLimit = le -limit de CE run (0 = échelle complète) : sans lui, un run peu
// profond serait refusé pour toujours face à une archive plus profonde, alors qu'il a
// ramené exactement ce qu'on lui a demandé (cf. duckdb.DegradedBatchReason).
//
// FAIL-OPEN, comme le cron : si la qualité du lot servi est illisible, on laisse
// passer. Un problème de lecture ne doit jamais empêcher une capture — le risque
// inverse (bloquer toute écriture) est le plus coûteux.
func cliBatchRefusalReason(ctx context.Context, db *sql.DB, key duckdb.WorldCSRBatchKey, entries []domain.LeaderboardEntry, depthLimit int) string {
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
	return duckdb.DegradedBatchReason(served, duckdb.WorldCSRStatsOfEntries(entries), depthLimit)
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
