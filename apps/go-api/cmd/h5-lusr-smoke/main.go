// Outil ops : smoke READ-ONLY de la chaîne LUSR v2 pour Halo 5 sur les VRAIES
// données. Copie isolée du shared h5, calcule l'état LUSR v2 (shadow-only) pour
// JGtm via le classifier title-aware (chaîne unique h5_arena), et imprime
// mu/sigma/experience. Prouve que la v2 (données basiques k/d/a/outcome/time)
// tourne pour Halo 5 sans MMR. ZÉRO écriture sur le clone de déploiement.
//
// Usage : go run ./cmd/h5-lusr-smoke [chemin shared h5 source]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	lusync "levelup/go-api/internal/sync"
)

const jgtmXUID = "2533274823110022"

func main() {
	src := "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_5/warehouse/shared_matches_v2.duckdb"
	if len(os.Args) > 1 {
		src = os.Args[1]
	}
	if err := os.Setenv("LEVELUP_LUSR_V2_ENABLED", "1"); err != nil {
		fatal("setenv: %v", err)
	}

	// Classifier LUSR title-aware : défaut Infinite (fail-loud) + dédié Halo 5
	// (chaîne unique h5_arena ; h5 n'a pas de pair_name).
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	lusync.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)

	// Copie isolée du shared h5 (lecture seule côté source).
	tmp, err := os.MkdirTemp("", "h5lusr")
	if err != nil {
		fatal("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmp)
	sharedCopy := filepath.Join(tmp, "shared.duckdb")
	if err := copyFile(src, sharedCopy); err != nil {
		fatal("copy shared: %v", err)
	}
	_ = copyFile(src+".wal", sharedCopy+".wal")

	shared, err := sql.Open("duckdb", sharedCopy)
	if err != nil {
		fatal("open shared: %v", err)
	}
	defer shared.Close()

	// ctx porteur du titre h5 → le seam GetLUSRChainForTitle route vers le classifier h5.
	ctx := ctxkeys.WithTitleSlug(context.Background(), halo5.TitleSlug)

	if _, err := shared.ExecContext(ctx, `DELETE FROM player_skill_state_v2 WHERE xuid = ?`, jgtmXUID); err != nil {
		fmt.Printf("reset state (non bloquant): %v\n", err)
	}

	processed, err := lusync.RunLUSRV2ShadowOwnerOnly(ctx, nil, lusync.NewPinnedSharedAccess(shared), jgtmXUID)
	if err != nil {
		fatal("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	fmt.Printf("LUSR v2 shadow h5 : %d matchs traités pour JGtm (chaîne title-aware h5_arena)\n", processed)

	rows, err := shared.QueryContext(ctx,
		`SELECT playlist_group, mu, sigma, experience, last_match_at
		   FROM player_skill_state_v2_latest WHERE xuid = ? ORDER BY playlist_group`, jgtmXUID)
	if err != nil {
		fatal("read state: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var group string
		var mu, sigma float64
		var exp int
		var lastAt sql.NullTime
		if err := rows.Scan(&group, &mu, &sigma, &exp, &lastAt); err != nil {
			fatal("scan: %v", err)
		}
		when := "—"
		if lastAt.Valid {
			when = lastAt.Time.Format("2006-01-02")
		}
		fmt.Printf("  chaîne=%-10s  μ=%.1f  σ=%.1f  exp=%d  dernier_match=%s\n", group, mu, sigma, exp, when)
		n++
	}
	if n == 0 {
		fmt.Println("  (aucun état LUSR — vérifier données / capability CapLUSR / classifier)")
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
