//go:build ignore

// cmd/seed-medal — pose un badge de domination (dominance_flag=1) sur le dernier match de JGtm.
// Usage (depuis apps/go-api/) :
//
//	go run ./cmd/seed-medal/main.go
//
// Pour supprimer après les tests :
//
//	go run ./cmd/seed-medal/main.go --delete
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	// dominanceFlagDomination = 1 (cf. internal/analysis/home.go)
	dominanceFlagDomination = 1
	playerDB                = "../../data/titles/halo_infinite/players/JGtm/stats.duckdb"
	sharedDB                = "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	gamertag                = "JGtm"
)

func main() {
	deleteMode := flag.Bool("delete", false, "Retire le badge de domination (remet dominance_flag à 0)")
	flag.Parse()

	// 1. Trouver le dernier match_id de JGtm via shared
	shared, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("shared open error:", err)
		os.Exit(1)
	}

	var xuid string
	if err := shared.QueryRow(
		`SELECT xuid FROM xuid_aliases WHERE lower(gamertag) = lower(?) LIMIT 1`,
		gamertag,
	).Scan(&xuid); err != nil {
		shared.Close()
		fmt.Println("xuid introuvable pour", gamertag, ":", err)
		os.Exit(1)
	}
	fmt.Printf("xuid de %s : %s\n", gamertag, xuid)

	var matchID string
	if err := shared.QueryRow(`
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		ORDER BY mr.start_time DESC
		LIMIT 1
	`, xuid).Scan(&matchID); err != nil {
		shared.Close()
		fmt.Println("Aucun match trouvé pour", gamertag, ":", err)
		os.Exit(1)
	}
	shared.Close()
	fmt.Printf("Dernier match_id de %s : %s\n", gamertag, matchID)

	// 2. Mettre à jour dominance_flag dans player_match_enrichment
	player, err := sql.Open("duckdb", playerDB)
	if err != nil {
		fmt.Println("player db open error:", err)
		os.Exit(1)
	}
	defer player.Close()

	newFlag := dominanceFlagDomination
	if *deleteMode {
		newFlag = 0
	}

	res, err := player.Exec(
		`UPDATE player_match_enrichment SET dominance_flag = ? WHERE match_id = ?`,
		newFlag, matchID,
	)
	if err != nil {
		fmt.Println("Erreur UPDATE:", err)
		os.Exit(1)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		// La ligne n'existe pas encore dans player_match_enrichment → INSERT
		_, err = player.Exec(
			`INSERT INTO player_match_enrichment (match_id, dominance_flag) VALUES (?, ?)
			 ON CONFLICT (match_id) DO UPDATE SET dominance_flag = excluded.dominance_flag`,
			matchID, newFlag,
		)
		if err != nil {
			fmt.Println("Erreur INSERT:", err)
			os.Exit(1)
		}
		fmt.Println("(ligne créée par INSERT)")
	}

	if *deleteMode {
		fmt.Printf("✓ dominance_flag remis à 0 sur match %s\n", matchID)
	} else {
		fmt.Printf("\n✓ Badge domination posé :\n")
		fmt.Printf("  match_id       = %s\n", matchID)
		fmt.Printf("  dominance_flag = %d (Domination)\n", dominanceFlagDomination)
		fmt.Printf("\nPour retirer : go run ./cmd/seed-medal/main.go --delete\n")
	}
}
