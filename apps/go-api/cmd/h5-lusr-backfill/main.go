// Outil ops : backfill LUSR v2 CANONICAL pour Halo 5 (un joueur). Provisionne la
// player DB h5 (migrations TargetPlayer), calcule l'état LUSR v2 + écrit
// match_skill_rank (rating_type='LUSR', lu par l'UI) sur tout l'historique synchronisé.
//
// RÉUTILISE le câblage exact : classifier title-aware (h5 → h5_arena), registre
// runtime avec h5 (CapLUSR), ctx porteur du titre. Écrit dans les VRAIES données
// (le clone qui sert l'app), append-only (match_skill_rank gagne par written_at).
//
// Usage : LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-lusr-backfill [Gamertag]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

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
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	// Résolution xuid robuste : le couple (halo_5, joueur) n'est souvent PAS déclaré
	// (seul JGtm l'est) → chercher la liste du titre, puis la liste globale (les 4
	// joueurs y sont sous halo_infinite, même xuid Xbox).
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

	// Registre runtime AVEC h5 (CapLUSR) → la garde slugHasLUSR passe pour h5.
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: halo5.TitleSlug, Name: "Halo 5", Status: titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{titlePkg.CapLUSR},
	})
	titlePkg.SetDefaultRegistry(reg)

	// Classifier LUSR title-aware (défaut Infinite + h5).
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	lusync.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)

	// Mode canonical (écrit match_skill_rank).
	if err := os.Setenv("LEVELUP_LUSR_V2_ENABLED", "1"); err != nil {
		fatal("setenv ENABLED: %v", err)
	}
	if err := os.Setenv("LEVELUP_LUSR_CANONICAL", "LUSR_V2"); err != nil {
		fatal("setenv CANONICAL: %v", err)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(halo5.TitleSlug)
	playerPath := pr.PlayerDBPath(halo5.TitleSlug, gt)

	// Provisionne la player DB h5 (TargetPlayer) — elle n'existe pas (sync shared-only).
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	halo5migrations.Register() // metadata h5 isolée ; player = fallback HINF (OwnsTarget).
	playerDB := openDB(playerPath)
	defer playerDB.Close()
	if err := migration.RunForTitleDB(playerDB, halo5.TitleSlug, migration.TargetPlayer); err != nil {
		fatal("provision player DB h5: %v", err)
	}

	shared := openDB(sharedPath)
	defer shared.Close()

	// ctx porteur du titre h5 → seam classifier h5 + garde capability.
	runCtx := ctxkeys.WithTitleSlug(ctx, halo5.TitleSlug)

	// Reset watermark + replay via le helper canonique : INSERT sentinelle
	// is_reset=TRUE (append-only #23046), JAMAIS le DELETE WHERE xuid qui est le
	// vecteur ART sur idx_pssv2 (cf. ADR 0026 + RecomputeLUSRCanonicalForPlayer).
	// Owner-only : ne touche que l'état de ce joueur.
	processed, err := lusync.RecomputeLUSRCanonicalForPlayer(runCtx, playerDB, shared, xuid)
	if err != nil {
		fatal("RecomputeLUSRCanonicalForPlayer: %v", err)
	}
	fmt.Printf("BACKFILL LUSR h5 : %d matchs traités (canonical) pour %s (xuid=%s)\n", processed, gt, xuid)

	verify(runCtx, shared, playerDB, xuid)
}

func verify(ctx context.Context, shared, playerDB *sql.DB, xuid string) {
	// État (shared).
	var group string
	var mu, sigma float64
	var exp int
	if err := shared.QueryRowContext(ctx,
		`SELECT playlist_group, mu, sigma, experience FROM player_skill_state_v2_latest WHERE xuid = ?`, xuid,
	).Scan(&group, &mu, &sigma, &exp); err != nil {
		fmt.Printf("état LUSR : (aucun) %v\n", err)
	} else {
		fmt.Printf("ÉTAT LUSR : chaîne=%s μ=%.1f σ=%.1f exp=%d\n", group, mu, sigma, exp)
	}
	// Lignes canoniques (player DB).
	var rows int
	_ = playerDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`).Scan(&rows)
	fmt.Printf("match_skill_rank (LUSR, lu par l'UI) : %d lignes écrites\n", rows)
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
