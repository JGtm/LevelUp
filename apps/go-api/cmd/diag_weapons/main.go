//go:build cgo

// diag_weapons — vérifie end-to-end les armes d'un bon match Chocoboflor.
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	chocoboXUID = "2535469190789936"
	goodMatchID = "81c0fc99-1cfe-4047-8bb7-81ce7d51743f" // Apr 6, 2026 - 6 real weapon_kills
)

func main() {
	shared, err := sql.Open("duckdb", "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb?access_mode=read_only")
	if err != nil {
		log.Fatal(err)
	}
	defer shared.Close()

	meta, err2 := sql.Open("duckdb", "../../data/titles/halo_infinite/warehouse/metadata.duckdb?access_mode=read_only")
	if err2 != nil {
		log.Fatal(err2)
	}
	defer meta.Close()

	fmt.Printf("=== Good match %s... (Apr 6, 2026) ===\n\n", goodMatchID[:8])

	// 1. What weapon_ids does Chocoboflor have in this match?
	fmt.Println("Weapon kills for Chocoboflor:")
	rows, _ := shared.Query(`
		SELECT COALESCE(reconciled_as, weapon_id) AS wid, COUNT(*) AS n
		FROM weapon_kills
		WHERE match_id = ? AND xuid = ?
		  AND COALESCE(reconciled_as, weapon_id) IS NOT NULL
		  AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
		GROUP BY wid ORDER BY n DESC`, goodMatchID, chocoboXUID)
	var weaponIDs []uint64
	for rows.Next() {
		var wid uint64
		var n int
		rows.Scan(&wid, &n)
		weaponIDs = append(weaponIDs, wid)
		fmt.Printf("  weapon_id=%-22d  kills=%d\n", wid, n)
	}
	rows.Close()

	// 2. Are these weapon_ids in metadata.weapon_labels?
	fmt.Println("\nWeapon labels from metadata:")
	for _, wid := range weaponIDs {
		var label sql.NullString
		meta.QueryRow(`SELECT COALESCE(name_fr, name_en) FROM weapon_labels WHERE weapon_id = ?`, wid).Scan(&label)
		lbl := "NOT FOUND"
		if label.Valid {
			lbl = label.String
		}
		fmt.Printf("  weapon_id=%-22d  label=%s\n", wid, lbl)
	}

	// 3. Check if ALL weapon_kills for this match have labels
	fmt.Println("\nWeapon IDs in this match (all players):")
	rows3, _ := shared.Query(`
		SELECT DISTINCT COALESCE(reconciled_as, weapon_id) AS wid
		FROM weapon_kills
		WHERE match_id = ?
		  AND COALESCE(reconciled_as, weapon_id) IS NOT NULL
		  AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)`, goodMatchID)
	var allWIDs []uint64
	for rows3.Next() {
		var wid uint64
		rows3.Scan(&wid)
		allWIDs = append(allWIDs, wid)
	}
	rows3.Close()
	fmt.Printf("Distinct weapon IDs used in match: %d\n", len(allWIDs))

	// Check how many are in weapon_labels
	labeled := 0
	for _, wid := range allWIDs {
		var count int
		meta.QueryRow(`SELECT COUNT(*) FROM weapon_labels WHERE weapon_id = ?`, wid).Scan(&count)
		if count > 0 {
			labeled++
		}
	}
	fmt.Printf("With label in metadata.weapon_labels: %d/%d\n", labeled, len(allWIDs))

	// 4. Summary of what user should see
	fmt.Printf("\n=== What UI should show for match %s... ===\n", goodMatchID[:8])
	fmt.Printf("Chocoboflor should see weapon kills in expander (Q28): %d weapons\n", len(weaponIDs))
	fmt.Printf("Scoreboard 'Outil de destr.' (Q12 top_weapon): should show top weapon\n")

	if len(weaponIDs) > 0 {
		// Get label for top weapon
		var topLabel sql.NullString
		meta.QueryRow(`SELECT COALESCE(name_fr, name_en) FROM weapon_labels WHERE weapon_id = ?`, weaponIDs[0]).Scan(&topLabel)
		lbl := fmt.Sprintf("%d", weaponIDs[0])
		if topLabel.Valid {
			lbl = topLabel.String
		}
		fmt.Printf("Expected top_weapon_label: '%s'\n", lbl)
	}
}
