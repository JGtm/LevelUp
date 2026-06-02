// diag_bot_resolution — inspecte la résolution xuid → gamertag pour les bots,
// et permet de réappliquer la vue v_gamertag_lookup en place (--repair).
//
// Usage :
//
//	go run ./cmd/diag_bot_resolution [path]            # inspection lecture seule
//	go run ./cmd/diag_bot_resolution [path] --repair   # re-applique la vue
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func main() {
	// Phase 1.5.1 B : steps Halo title-owned fournis au runner via le provider.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	path := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	repair := false
	for _, a := range os.Args[1:] {
		if a == "--repair" {
			repair = true
			continue
		}
		path = a
	}
	fmt.Printf("=== Diag bot resolution sur %s (repair=%v) ===\n\n", path, repair)

	if repair {
		// Mode write : applique la vue mise à jour.
		dbw, err := sql.Open("duckdb", path)
		if err != nil {
			fmt.Printf("open write: %v\n", err)
			os.Exit(1)
		}
		if err := migration.RunForDB(dbw, migration.TargetShared); err != nil {
			fmt.Printf("RunForDB: %v\n", err)
			dbw.Close()
			os.Exit(1)
		}
		dbw.Close()
		fmt.Println(">> Migrations rejouées (la nouvelle 'repair_v_gamertag_lookup_bots_2026_05_16' force le CREATE OR REPLACE).")
		fmt.Println()
	}

	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Printf("open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 1. Existence de la vue + définition.
	fmt.Println("--- 1. Vues présentes dans la DB ---")
	rows, err := db.Query(`SELECT view_name FROM duckdb_views() WHERE view_name LIKE 'v_%' ORDER BY view_name`)
	if err != nil {
		fmt.Printf("query views: %v\n", err)
	} else {
		for rows.Next() {
			var n string
			_ = rows.Scan(&n)
			fmt.Printf("  - %s\n", n)
		}
		rows.Close()
	}
	fmt.Println()

	// 2. Définition complète de v_gamertag_lookup (pour voir si la CASE bot est dedans).
	fmt.Println("--- 2. Définition v_gamertag_lookup (extrait) ---")
	var sqlText sql.NullString
	if err := db.QueryRow(`SELECT sql FROM duckdb_views() WHERE view_name = 'v_gamertag_lookup'`).Scan(&sqlText); err != nil {
		fmt.Printf("  query: %v\n", err)
	} else if sqlText.Valid {
		s := sqlText.String
		if len(s) > 800 {
			fmt.Println(s[:800] + "...")
		} else {
			fmt.Println(s)
		}
	}
	fmt.Println()

	// 3. Bots distincts en match_participants.
	fmt.Println("--- 3. xuid distincts qui ressemblent à des bots (LIKE 'bid%') ---")
	rows, err = db.Query(`
		SELECT xuid, COUNT(*) AS occurrences
		FROM match_participants
		WHERE xuid LIKE '%id(%' OR xuid LIKE '%bid%' OR xuid LIKE 'Bid%' OR xuid LIKE 'BID%'
		GROUP BY xuid
		ORDER BY occurrences DESC
		LIMIT 20
	`)
	if err != nil {
		fmt.Printf("  query: %v\n", err)
	} else {
		for rows.Next() {
			var x string
			var n int
			_ = rows.Scan(&x, &n)
			fmt.Printf("  xuid=%q  occurrences=%d\n", x, n)
		}
		rows.Close()
	}
	fmt.Println()

	// 4. Résolution via v_gamertag_lookup pour ces bots.
	fmt.Println("--- 4. Résolution via v_gamertag_lookup ---")
	rows, err = db.Query(`
		SELECT xuid, gamertag
		FROM v_gamertag_lookup
		WHERE xuid LIKE '%id(%' OR xuid LIKE '%bid%' OR xuid LIKE 'Bid%' OR xuid LIKE 'BID%'
		ORDER BY xuid
		LIMIT 20
	`)
	if err != nil {
		fmt.Printf("  query: %v\n", err)
	} else {
		for rows.Next() {
			var x, g string
			_ = rows.Scan(&x, &g)
			marker := " "
			if x == g {
				marker = "❌"
			} else {
				marker = "✅"
			}
			fmt.Printf("  %s xuid=%q  gamertag=%q\n", marker, x, g)
		}
		rows.Close()
	}
	fmt.Println()

	// 5. schema_migrations : a-t-on appliqué l'upgrade bot ?
	fmt.Println("--- 5. schema_migrations (cherche upgrade_v_gamertag_lookup) ---")
	rows, err = db.Query(`SELECT name, schema_done, applied_at FROM schema_migrations WHERE name LIKE '%gamertag%' OR name LIKE '%bot%' OR name LIKE '%resolution%' ORDER BY applied_at`)
	if err != nil {
		fmt.Printf("  query: %v\n", err)
	} else {
		hasAny := false
		for rows.Next() {
			hasAny = true
			var name string
			var done bool
			var appliedAt sql.NullString
			_ = rows.Scan(&name, &done, &appliedAt)
			fmt.Printf("  name=%q  done=%v  applied_at=%q\n", name, done, appliedAt.String)
		}
		rows.Close()
		if !hasAny {
			fmt.Println("  (aucune migration matchant le pattern)")
		}
	}
	fmt.Println()

	// 6. Longueur des xuid bots (détection caractères invisibles).
	fmt.Println("--- 6. Longueur des xuid bots (détection caractères invisibles) ---")
	rows, err = db.Query(`
		SELECT xuid, LENGTH(xuid) AS len
		FROM match_participants
		WHERE xuid LIKE '%id(%'
		GROUP BY xuid
		ORDER BY len DESC
		LIMIT 10
	`)
	if err != nil {
		fmt.Printf("  query: %v\n", err)
	} else {
		for rows.Next() {
			var x string
			var n int
			_ = rows.Scan(&x, &n)
			expected := len("bid(N.0)") // 8 pour 1 chiffre, 9 pour 2 chiffres
			fmt.Printf("  xuid=%q  len=%d  (attendu ~%d)\n", x, n, expected)
		}
		rows.Close()
	}
}
