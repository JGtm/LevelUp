//go:build cgo

// cmd/backfill_participation_info — backfille les 4 colonnes ParticipationInfo
// (present_at_beginning, present_at_completion, joined_in_progress, left_in_progress)
// pour les lignes match_participants où ces colonnes sont NULL.
//
// Stratégie :
//  1. Applique la migration ADD COLUMN IF NOT EXISTS (idempotent).
//  2. Sélectionne les match_ids distincts où present_at_beginning IS NULL.
//  3. Pour chaque match : GetMatchStats API → ExtractParticipants → UPDATE ciblé.
//
// L'UPDATE est limité aux 4 colonnes NULL : aucun risque de régression sur les
// autres colonnes. La clause WHERE ... IS NULL rend l'opération idempotente.
//
// Ne pas lancer pendant que le serveur est en cours d'écriture sur shared.
//
// Usage (depuis apps/go-api/) :
//
//	go run -tags cgo ./cmd/backfill_participation_info/
//	go run -tags cgo ./cmd/backfill_participation_info/ -dry-run -limit 10
//	go run -tags cgo ./cmd/backfill_participation_info/ -gamertag Chocoboflor
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

func main() {
	dataDir := flag.String("data", "data", "Racine du dossier data/")
	titleSlug := flag.String("title", "halo_infinite", "Title slug")
	envFile := flag.String("env-file", ".env.local", "Chemin .env.local")
	authFile := flag.String("auth-file", "data/auth/watcher_tokens.json", "Chemin watcher_tokens.json")
	gamertag := flag.String("gamertag", "Chocoboflor", "Gamertag pour charger les tokens")
	limit := flag.Int("limit", 0, "Nombre max de matchs à traiter (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "Afficher les matchs sans modifier la DB")
	rps := flag.Int("rps", 3, "Requêtes API par seconde")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	loadEnvLocal(*envFile)

	sharedDBPath := filepath.Join(*dataDir, "titles", *titleSlug, "warehouse", "shared_matches_v2.duckdb")

	ctx := context.Background()

	dbDSN := sharedDBPath
	if *dryRun {
		dbDSN = sharedDBPath + "?access_mode=read_only"
	}
	db, err := sql.Open("duckdb", dbDSN)
	if err != nil {
		fatalf("open shared DB: %v", err)
	}
	defer db.Close()

	if !*dryRun {
		if err := applyMigration(ctx, db); err != nil {
			fatalf("migration: %v", err)
		}
	}

	matchIDs, err := queryNullMatchIDs(ctx, db, *limit)
	if err != nil {
		fatalf("query nulls: %v", err)
	}
	fmt.Printf("Matchs à backfiller (present_at_beginning IS NULL) : %d\n", len(matchIDs))

	if *dryRun || len(matchIDs) == 0 {
		if *dryRun && len(matchIDs) > 0 {
			n := 20
			if n > len(matchIDs) {
				n = len(matchIDs)
			}
			fmt.Println("--- dry-run : premiers matchs ---")
			for _, id := range matchIDs[:n] {
				fmt.Printf("  %s\n", id)
			}
		}
		return
	}

	tokens, err := loadTokens(ctx, *authFile, *gamertag)
	if err != nil {
		fatalf("tokens: %v", err)
	}
	client := gosync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, *rps)

	var (
		updated int
		skipped int
		apiGone int
		apiErr  int
	)
	start := time.Now()

	for i, matchID := range matchIDs {
		matchJSON, err := client.GetMatchStats(ctx, matchID)
		if err != nil {
			if isGone(err) {
				apiGone++
				if (i+1)%50 == 0 || apiGone <= 5 {
					fmt.Printf("  [%d/%d] EXPIRÉ %s\n", i+1, len(matchIDs), matchID)
				}
				continue
			}
			slog.WarnContext(ctx, "GetMatchStats échoué", "match_id", matchID, "err", err)
			apiErr++
			continue
		}

		rows := gosync.ExtractParticipants(matchJSON)
		if len(rows) == 0 {
			skipped++
			continue
		}

		n, err := updateParticipationInfo(ctx, db, rows)
		if err != nil {
			slog.WarnContext(ctx, "UPDATE échoué", "match_id", matchID, "err", err)
			apiErr++
			continue
		}
		updated += n

		if (i+1)%100 == 0 || i == len(matchIDs)-1 {
			fmt.Printf("  [%d/%d] +%d lignes (total %d) | expirés=%d erreurs=%d\n",
				i+1, len(matchIDs), n, updated, apiGone, apiErr)
		}
	}

	fmt.Printf("\n=== Résultat ===\n")
	fmt.Printf("Lignes mises à jour : %d\n", updated)
	fmt.Printf("Matchs expirés API  : %d\n", apiGone)
	fmt.Printf("Matchs sans joueurs : %d\n", skipped)
	fmt.Printf("Erreurs             : %d\n", apiErr)
	fmt.Printf("Durée               : %s\n", time.Since(start).Round(time.Second))
}

// applyMigration ajoute les 4 colonnes si elles n'existent pas encore.
func applyMigration(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_beginning  BOOLEAN;
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_completion BOOLEAN;
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS joined_in_progress    BOOLEAN;
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS left_in_progress      BOOLEAN;
	`)
	return err
}

// queryNullMatchIDs retourne les match_ids distincts où present_at_beginning IS NULL,
// ordonnés du plus récent au plus ancien (les récents ont plus de chances d'être
// encore accessibles via l'API). Si la colonne n'existe pas encore (migration
// non appliquée), retourne tous les match_ids distincts.
func queryNullMatchIDs(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := `
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.present_at_beginning IS NULL
		ORDER BY mr.start_time DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil && strings.Contains(err.Error(), "present_at_beginning") {
		// Colonne absente : migration pas encore appliquée → tous les matchs.
		q2 := `SELECT DISTINCT mp.match_id FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			ORDER BY mr.start_time DESC`
		if limit > 0 {
			q2 += fmt.Sprintf(" LIMIT %d", limit)
		}
		rows, err = db.QueryContext(ctx, q2)
	}
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

// updateParticipationInfo écrit les 4 booleans pour les lignes dont
// present_at_beginning est encore NULL. Retourne le nombre de lignes modifiées.
func updateParticipationInfo(ctx context.Context, db *sql.DB, rows []gosync.ParticipantRow) (int, error) {
	total := 0
	for _, row := range rows {
		if row.PresentAtBeginning == nil && row.PresentAtCompletion == nil &&
			row.JoinedInProgress == nil && row.LeftInProgress == nil {
			continue // ParticipationInfo absent dans le JSON de ce match
		}
		res, err := db.ExecContext(ctx, `
			UPDATE match_participants
			SET
				present_at_beginning  = ?,
				present_at_completion = ?,
				joined_in_progress    = ?,
				left_in_progress      = ?
			WHERE match_id = ?
			  AND xuid = ?
			  AND present_at_beginning IS NULL`,
			row.PresentAtBeginning, row.PresentAtCompletion,
			row.JoinedInProgress, row.LeftInProgress,
			row.MatchID, row.XUID,
		)
		if err != nil {
			return total, fmt.Errorf("UPDATE %s/%s: %w", row.MatchID, row.XUID, err)
		}
		n, _ := res.RowsAffected()
		total += int(n)
	}
	return total, nil
}

func isGone(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "HTTP 404") || strings.Contains(s, "HTTP 410") ||
		strings.Contains(s, "ressource absente")
}

func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func loadTokens(ctx context.Context, authFile, gamertag string) (*struct {
	SpartanToken   string
	ClearanceToken string
}, error) {
	store := auth.NewTokenStore(authFile)
	stored, _ := store.Load()
	if stored != nil && stored.IsXSTSValid(0) {
		result, err := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken)
		if err == nil {
			return &struct {
				SpartanToken   string
				ClearanceToken string
			}{result.SpartanToken, result.ClearanceToken}, nil
		}
	}

	provider := auth.NewSISUProvider()
	_ = provider

	envKey := "SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)
	if rt := os.Getenv(envKey); rt != "" {
		tok, err := auth.NewSISUProvider().TryOAuthRefresh(ctx, rt)
		if err == nil && tok != "" {
			result, err := auth.ExchangeAccessToken(ctx, tok)
			if err == nil {
				return &struct {
					SpartanToken   string
					ClearanceToken string
				}{result.Tokens.SpartanToken, result.Tokens.ClearanceToken}, nil
			}
		}
	}

	return nil, fmt.Errorf("tokens introuvables pour %s (vérifier %s et .env.local)", gamertag, authFile)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
