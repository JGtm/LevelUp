// Outil ops : smoke READ-ONLY de la surface de lecture Halo 5 sur les VRAIES
// données synchronisées (JGtm). Ouvre une player DB TEMP attachée à une COPIE
// isolée du shared h5, puis exécute les repos de lecture title-aware (match
// history, compare, stats) et imprime les preuves clés :
//   - N matchs réellement servis pour JGtm en h5 (chemin legacy title-aware) ;
//   - KDA JAMAIS fabriqué (compare/stats : 0/nil pour h5, règle absolue) ;
//   - baseline dégâts résolue à 115 (pas 225) via games.EffectiveHpToKill.
//
// ZÉRO écriture sur le clone de déploiement, ZÉRO token (surfaces token-free).
// Usage : go run ./cmd/h5-read-smoke [chemin shared h5 source] [chemin config worktree]
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"levelup/go-api/internal/games"
	halo5 "levelup/go-api/internal/games/halo_5"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const jgtmXUID = "2533274823110022"

func main() {
	src := "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_5/warehouse/shared_matches_v2.duckdb"
	if len(os.Args) > 1 {
		src = os.Args[1]
	}
	// Racine du clone qui porte config/titles/halo_5 (LoadFromConfigDir y ajoute
	// "config/titles" lui-même) — le worktree feat/multititre-peripherie.
	configRoot := "c:/Users/Guillaume/Downloads/Scripts/levelup-multititre"
	if len(os.Args) > 2 {
		configRoot = os.Args[2]
	}
	ctx := context.Background()

	// Resolver d'endpoints depuis la config worktree (h5 : 115 + no_native_kda=true).
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(configRoot, []string{halo5.TitleSlug, "halo_infinite"}, nil); len(errs) != 0 {
		fatal("load mappings: %v", errs)
	}
	games.SetDefaultEndpointResolver(games.NewMappingsEndpointResolver(reg, "halo_infinite"))
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

	fmt.Printf("RESOLVER h5 : ProvidesNativeKDA=%v (attendu false)  EffectiveHpToKill=%.0f (attendu 115)\n",
		games.ProvidesNativeKDA(halo5.TitleSlug), games.EffectiveHpToKill(halo5.TitleSlug))

	// Copie isolée du shared h5 (lecture seule, zéro écriture sur le clone deploy).
	tmp, err := os.MkdirTemp("", "h5read")
	if err != nil {
		fatal("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmp)
	sharedCopy := filepath.Join(tmp, "shared_matches_v2.duckdb")
	if err := copyFile(src, sharedCopy); err != nil {
		fatal("copy shared: %v", err)
	}
	_ = copyFile(src+".wal", sharedCopy+".wal") // .wal éventuel (best-effort)

	pcfg := duckdb.PlayerPoolConfig{
		Gamertag:     "JGtm",
		XUID:         jgtmXUID,
		TitleSlug:    halo5.TitleSlug,
		PlayerDBPath: filepath.Join(tmp, "stats.duckdb"),
		SharedDBPath: sharedCopy,
		MetaDBPath:   "", // pas de metadata h5 → labels dégradés (sans incidence sur ce smoke)
	}
	pdb, err := duckdb.GetOrOpen(ctx, pcfg)
	if err != nil {
		fatal("GetOrOpen player h5: %v", err)
	}

	// 1) Match history — chemin legacy title-aware (MatchHistoryRepo.LoadAll → SharedReadDB).
	if hist, err := duckdb.NewMatchHistoryRepo(pdb).LoadAll(ctx); err != nil {
		fmt.Printf("MATCH HISTORY : ERR %v\n", err)
	} else {
		fmt.Printf("MATCH HISTORY : %d matchs servis pour JGtm (h5, lu via le chemin produit title-aware)\n", len(hist))
	}

	// 2) Compare — KDA gaté (NULL pour h5), KDR conservé.
	if cmp, err := duckdb.NewCompareRepo(pdb).GetLocalStats(ctx, jgtmXUID, halo5.TitleSlug); err != nil {
		fmt.Printf("COMPARE : ERR %v\n", err)
	} else {
		fmt.Printf("COMPARE : matches=%d  KDA=%.3f (0 = non fabriqué, règle OK)  KDR=%.3f  winrate=%.3f\n",
			cmp.Matches, cmp.KDA, cmp.KDR, cmp.WinRate)
	}

	// 3) Stats — KDA per-match nil (h5 ne le fournit pas), baseline OC/DR = 115.
	if stats, err := duckdb.NewStatsRepo(pdb).LoadStatsMatches(ctx); err != nil {
		fmt.Printf("STATS : ERR %v\n", err)
	} else {
		nilKDA := 0
		for i := range stats {
			if stats[i].KDA == nil {
				nilKDA++
			}
		}
		fmt.Printf("STATS : %d matchs ; KDA nil sur %d/%d (attendu %d/%d — jamais fabriqué)\n",
			len(stats), nilKDA, len(stats), len(stats), len(stats))
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
