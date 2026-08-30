// Outil ops : backfill ENRICHISSEMENT PAR JOUEUR pour Halo 5 (un joueur), à partir
// du shared déjà rempli. Provisionne la player DB h5 (migrations TargetPlayer) puis
// recalcule l'enrichment title-agnostic (baseline rows + had_bot + sessions +
// performance + engagement + coefs + assists + dominance + is_with_friends + mv_*).
// HORS LUSR — le LUSR est recalculé séparément par cmd/h5-lusr-backfill, APRÈS la
// classification ranked.
//
// Le sync live h5 est shared-only → la player DB n'existe pas avant ce backfill.
// Idempotent (append-only par stage, ensurePlayerEnrichmentRows en delta).
//
// Usage : LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-enrich [Gamertag] [ami1,ami2,...]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	lusync "levelup/go-api/internal/sync"
)

func main() {
	gt := "JGtm"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	var friends []string
	if len(os.Args) > 2 && strings.TrimSpace(os.Args[2]) != "" {
		for _, f := range strings.Split(os.Args[2], ",") {
			if f = strings.TrimSpace(f); f != "" {
				friends = append(friends, f)
			}
		}
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	// Résolution xuid robuste : le couple (halo_5, joueur) n'est souvent PAS déclaré
	// (seul JGtm l'est) → chercher d'abord la liste du titre, puis la liste globale
	// (les 4 joueurs y sont sous halo_infinite, même xuid Xbox).
	findXUID := func(slug string) string {
		ps, e := cfg.LoadPlayers(slug)
		if e != nil {
			return ""
		}
		for i := range ps {
			if ps[i].Gamertag == gt {
				return ps[i].XUID
			}
		}
		return ""
	}
	xuid := findXUID(halo5.TitleSlug)
	if xuid == "" {
		xuid = findXUID("")
	}
	if xuid == "" {
		fatal("xuid introuvable pour %q dans db_profiles", gt)
	}

	// Classifier de chaîne (title-aware) : la segmentation d'historique du
	// performance_score délègue au seam LUSR (GetPerformanceChain → GetLUSRChain),
	// donc requis MÊME pour l'enrichment hors-LUSR (sinon panic au 1er perf).
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	lusync.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)
	// Famille de la chaîne de perf classée (ranked_slayer / ranked_objectif) : h5
	// n'a pas de sous-mode → classifier dédié qui répond false.
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	lusync.SetObjectiveFamilyClassifierForTitle(halo5.TitleSlug, halo5.IsObjectiveSubMode)

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(halo5.TitleSlug)
	playerPath := pr.PlayerDBPath(halo5.TitleSlug, gt)

	// Provisionne la player DB h5 (TargetPlayer) — schéma player = fallback HINF
	// (OwnsTarget=metadata seul côté h5) → player_match_enrichment + colonnes
	// engagement présentes. Crée le fichier stats.duckdb s'il n'existe pas.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	halo5migrations.Register()
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		fatal("mkdir player dir: %v", err)
	}
	playerDB := openDB(playerPath)
	defer playerDB.Close()
	if err := migration.RunForTitleDB(playerDB, halo5.TitleSlug, migration.TargetPlayer); err != nil {
		fatal("provision player DB h5: %v", err)
	}

	shared := openDB(sharedPath)
	defer shared.Close()

	runCtx := ctxkeys.WithTitleSlug(ctx, halo5.TitleSlug)
	fmt.Printf("enrich h5 : gamertag=%s xuid=%s amis=%v\n", gt, xuid, friends)

	report, err := lusync.BackfillEnrichmentFromShared(runCtx, playerDB, shared, xuid, friends, true)
	if err != nil {
		fatal("BackfillEnrichmentFromShared: %v", err)
	}
	fmt.Printf("ENRICH h5 %s : baseline=%d had_bot=%d sessions=%d perf=%d engagement=%d coefs=%d assists=%d dominance=%d friends_promoted=%d mv=%d errors=%d (%dms)\n",
		gt, report.BaselineRowsCreated, report.HadBotUpdated, report.SessionsAssigned,
		report.PerfComputed, report.EngagementComputed, report.CoefsUpdated, report.AssistsModes,
		report.DominanceMatches, report.FriendsResult.MatchesPromoted, report.AggregatesCreated,
		len(report.Errors), report.Duration.Milliseconds())
	for _, e := range report.Errors {
		fmt.Printf("  - erreur: %v\n", e)
	}

	verify(runCtx, playerDB, xuid)
}

func verify(ctx context.Context, playerDB *sql.DB, xuid string) {
	count := func(q string) int {
		var n int
		_ = playerDB.QueryRowContext(ctx, q).Scan(&n)
		return n
	}
	fmt.Printf("player_match_enrichment_latest : %d lignes ; perf non-null=%d ; session_id non-null=%d ; is_with_friends=%d\n",
		count(`SELECT COUNT(*) FROM player_match_enrichment_latest`),
		count(`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`),
		count(`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE session_id IS NOT NULL`),
		count(`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE is_with_friends`))
}

func openDB(path string) *sql.DB {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		fatal("open %s: %v", path, err)
	}
	return db
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
