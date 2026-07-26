// cmd/backfill_objective_stats — peuple shared.match_objective_stats pour
// l'historique des matchs à objectif (CTF / Strongholds / KOTH / Oddball /
// Stockpile / Extraction / VIP) en re-fetchant GetMatchStats et en extrayant les
// blocs objectif du payload.
//
// Contexte : les stats objectifs par joueur/match sont DÉJÀ dans le payload
// GetMatchStats (Players[].PlayerTeamStats[0].Stats.<BlocMode>) mais n'étaient
// pas extraites avant le chantier V72-03. Le sync natif les persiste désormais ;
// ce backfill couvre les matchs ANTÉRIEURS. Écriture APPEND-ONLY INSERT-only
// (ART-safe #23046) via persist.InsertObjectiveStats ; lecture par la vue
// match_objective_stats_latest.
//
// Reprise : un match est candidat s'il n'a PAS le bit MBitObjectiveStats posé
// dans match_registry.backfill_completed ET pas encore de ligne _latest. Après
// traitement, le bit est TOUJOURS posé (même pour un match sans mode à objectif,
// qui ne produit aucune ligne — Slayer) → pas de re-fetch inutile au run suivant.
//
// CONSÉQUENCE lors de l'AJOUT d'un bloc de mode (V721-02 : Stockpile, Extraction, VIP) :
// les matchs de ce mode déjà parcourus par un run antérieur portent le bit alors
// qu'ils n'ont produit AUCUNE ligne (le bloc n'était pas encore extrait) — ils ne
// sont donc plus candidats. Il faut RETIRER le bit sur ces seuls matchs avant de
// relancer (reset ciblé par mode via match_registry.pair_name / game_variant_name,
// cf. PLAN_V721_NOTION_BATCH.md § V721-02, item 02.6). Le sync natif, lui, ne pose
// JAMAIS ce bit : les matchs synchronisés depuis restent candidats sans reset.
//
// Usage (serveur LevelUp ARRÊTÉ — accès RW exclusif à la shared DB) :
//
//	go run ./cmd/backfill_objective_stats --gamertag JGtm [--dry-run] [--limit N] [--rps 5]
//
// Auth : MultiUserTokenStore (ADR 0023) en priorité, via le xuid du --gamertag.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	persist "levelup/go-api/internal/persist"
	auth_platform "levelup/go-api/internal/platform/auth"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/objective"
)

func main() {
	gamertag := flag.String("gamertag", "JGtm", "Gamertag dont les tokens servent à l'auth API")
	dryRun := flag.Bool("dry-run", false, "Lister/extraire sans écrire ni marquer")
	limit := flag.Int("limit", 0, "Limiter au N matchs candidats les plus récents (0 = tous)")
	force := flag.Bool("force", false, "Ignorer le bit « déjà tenté ». INDISPENSABLE après un élargissement du périmètre d'extraction : sans lui les matchs du nouveau mode sont sautés en silence. Les matchs ayant déjà des lignes restent exclus.")
	rps := flag.Int("rps", 5, "Requêtes API par seconde")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)

	// 1. Ouvrir la shared DB en RW (échoue si le serveur tient le lock).
	handle, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open shared RW (serveur LevelUp arrêté ?): %v\n", err)
		os.Exit(1)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// 2. Résoudre le xuid du gamertag pour l'auth store-first.
	xuid, err := resolveXUID(cfg, *gamertag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve xuid %s: %v\n", *gamertag, err)
		os.Exit(1)
	}

	// 3. Auth via MultiUserTokenStore (ADR 0023).
	store := auth_platform.NewMultiUserTokenStore(pr.WatcherTokensDir())
	provider := auth_platform.NewSISUProvider()
	exch, err := auth_platform.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, *gamertag, auth_platform.LegacyAuthInputs{})
	if err != nil || exch == nil {
		fmt.Fprintf(os.Stderr, "auth %s: %v\n", *gamertag, err)
		os.Exit(1)
	}
	client := go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, *rps)

	// 4. Lister les matchs candidats (bit absent + pas de ligne _latest).
	matchIDs, err := listCandidateMatchIDs(ctx, db, *limit, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list matchs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Matchs candidats : %d (dry-run=%v, force=%v)\n", len(matchIDs), *dryRun, *force)

	// 5. Re-fetch + extract + persist append-only + mark.
	var fetched, withRows, rowsInserted, marked int
	for i, mid := range matchIDs {
		mj, ferr := client.GetMatchStats(ctx, mid)
		if ferr != nil {
			slog.WarnContext(ctx, "backfill_objective: GetMatchStats échoué", "match_id", mid, "err", ferr)
			continue
		}
		fetched++
		rows := objective.ExtractObjectiveStats(mid, mj)
		if len(rows) > 0 {
			withRows++
		}
		if *dryRun {
			rowsInserted += len(rows)
			continue
		}
		if len(rows) > 0 {
			if err := persist.InsertObjectiveStats(ctx, db, rows); err != nil {
				slog.WarnContext(ctx, "backfill_objective: INSERT échoué", "match_id", mid, "err", err)
				continue
			}
			rowsInserted += len(rows)
		}
		// Marquer le match traité (même 0 ligne) pour le sortir du set de reprise.
		if err := markObjectiveDone(ctx, db, mid); err != nil {
			slog.WarnContext(ctx, "backfill_objective: mark échoué", "match_id", mid, "err", err)
			continue
		}
		marked++
		if (i+1)%25 == 0 {
			fmt.Printf("  [%d/%d] fetched=%d with_rows=%d rows=%d marked=%d\n",
				i+1, len(matchIDs), fetched, withRows, rowsInserted, marked)
		}
	}
	fmt.Printf("\n=== Terminé ===\n  matchs fetchés : %d\n  matchs avec objectifs : %d\n  lignes insérées : %d\n  matchs marqués : %d\n",
		fetched, withRows, rowsInserted, marked)
}

// markObjectiveDone pose le bit MBitObjectiveStats sur match_registry.backfill_completed
// (idempotent, OR bit-à-bit) pour sortir le match du set de reprise — y compris un match
// sans mode à objectif (0 ligne produite). Écriture inline dans le CLI (un seul writer,
// serveur arrêté) plutôt qu'un helper du god-package sync/ (ratchet K3c sync_root_freeze) ;
// match_registry.backfill_completed est une colonne mutable non-ART, cf. MarkPveStatsDone.
func markObjectiveDone(ctx context.Context, db *sql.DB, matchID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?
	`, go_sync.MBitObjectiveStats, matchID)
	return err
}

// resolveXUID lit db_profiles.json pour mapper gamertag → xuid.
func resolveXUID(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return "", err
	}
	for i := range players {
		if players[i].Gamertag == gamertag {
			return players[i].XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent de db_profiles.json", gamertag)
}

// listCandidateMatchIDs retourne les match_id à traiter, triés du plus récent au
// plus ancien.
//
// Deux garde-fous en temps normal : le bit MBitObjectiveStats absent (match jamais
// tenté) ET aucune ligne dans match_objective_stats_latest (match pas déjà couvert).
//
// `force` LÈVE le premier, et c'est indispensable après tout ÉLARGISSEMENT du
// périmètre d'extraction. Raison : `markObjectiveDone` pose le bit pour TOUT match
// fetché, y compris ceux qui n'ont produit aucune ligne — donc après l'ajout d'un
// bloc de stats, les matchs de ce mode sont déjà marqués « tentés » et le backfill
// les saute EN SILENCE, laissant les nouvelles colonnes vides. C'est exactement ce
// qui s'est produit en v7.2.1 (Stockpile/Extraction/VIP : 0 candidat sur 44 matchs
// concernés, constaté en production le 2026-07-26).
//
// Le second garde-fou reste actif même sous `force` : un match qui a DÉJÀ des lignes
// n'est jamais re-fetché, donc aucun doublon possible. Le coût de `force` est donc
// borné aux matchs sans objectifs (Slayer et consorts), re-fetchés pour rien —
// acceptable pour une opération ponctuelle après extension.
func listCandidateMatchIDs(ctx context.Context, db *sql.DB, limit int, force bool) ([]string, error) {
	bitClause := fmt.Sprintf("(COALESCE(mr.backfill_completed, 0) & %d) = 0", go_sync.MBitObjectiveStats)
	if force {
		bitClause = "TRUE"
	}
	q := fmt.Sprintf(`SELECT mr.match_id
	      FROM match_registry mr
	      WHERE %s
	        AND NOT EXISTS (
	            SELECT 1 FROM match_objective_stats_latest o WHERE o.match_id = mr.match_id
	        )
	      ORDER BY `+analysis.SQLStartTimeCanonical("mr")+` DESC`, bitClause)
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
