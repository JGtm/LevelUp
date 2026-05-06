//go:build cgo

// diag_weapon_citations — vérifie le wiring complet des citations weapon_stat :
//  1. Quelles citations sont configurées avec mapping_type='weapon_stat' ?
//  2. Pour un match donné, est-ce que les weapon_kills:* dans stats matchent
//     les stat_name attendus par les mappings ?
//
// Usage : go run -tags cgo ./cmd/diag_weapon_citations <player_gt> <match_id>
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: diag_weapon_citations <player_gt> [match_id]")
	}
	playerGT := os.Args[1]
	matchID := ""
	if len(os.Args) >= 3 {
		matchID = os.Args[2]
	}

	playerStats := filepath.Join("..", "..", "data", "titles", "halo_infinite", "players", playerGT, "stats.duckdb")
	sharedPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	metaPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")

	db := openDuckDB(playerStats)
	defer db.Close()
	mustExec(db, fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath))
	mustExec(db, fmt.Sprintf("ATTACH '%s' AS meta (READ_ONLY)", metaPath))

	fmt.Println("=== 0. Schéma réel de meta.citation_mappings ===")
	schemaRows, err := db.QueryContext(context.Background(), `
		SELECT column_name, data_type FROM information_schema.columns
		WHERE table_catalog = 'meta' AND table_name = 'citation_mappings'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("schema: %v", err)
	}
	hasStatName := false
	for schemaRows.Next() {
		var col, ty string
		_ = schemaRows.Scan(&col, &ty)
		fmt.Printf("  %-25s %s\n", col, ty)
		if col == "stat_name" {
			hasStatName = true
		}
	}
	schemaRows.Close()
	if !hasStatName {
		fmt.Println("\n  /!\\  La colonne stat_name n'existe PAS dans meta.citation_mappings")
		fmt.Println("\n=== Mapping_types présents ===")
		mtRows, _ := db.QueryContext(context.Background(),
			"SELECT mapping_type, COUNT(*) FROM meta.citation_mappings GROUP BY mapping_type ORDER BY 2 DESC")
		for mtRows.Next() {
			var mt string
			var cnt int
			_ = mtRows.Scan(&mt, &cnt)
			fmt.Printf("  %-15s : %d\n", mt, cnt)
		}
		mtRows.Close()

		fmt.Println("\n=== Test SELECT loadFullCitationMappings (du code Go) ===")
		_, qerr := db.QueryContext(context.Background(),
			`SELECT citation_name_norm, medal_ids, stat_name, award_name, custom_function FROM meta.citation_mappings LIMIT 1`)
		if qerr != nil {
			fmt.Printf("  ECHEC : %v\n", qerr)
		}

		fmt.Println("\n=== Etat de match_citations (player DB) ===")
		var totalCit int
		_ = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM match_citations").Scan(&totalCit)
		fmt.Printf("  Total rows match_citations : %d\n", totalCit)
		if totalCit > 0 {
			distRows, _ := db.QueryContext(context.Background(),
				"SELECT citation_name_norm, COUNT(*), SUM(value) FROM match_citations GROUP BY citation_name_norm ORDER BY 3 DESC LIMIT 20")
			for distRows.Next() {
				var n string
				var c, s int
				_ = distRows.Scan(&n, &c, &s)
				fmt.Printf("    %-30s : %d matches, %d total\n", n, c, s)
			}
			distRows.Close()
		}
		return
	}

	fmt.Println("\n=== 1. Citations configurées en weapon_stat ===")
	rows, err := db.QueryContext(context.Background(), `
		SELECT citation_name_norm, citation_name_display, stat_name
		FROM meta.citation_mappings
		WHERE mapping_type = 'weapon_stat'
		ORDER BY citation_name_norm
	`)
	if err != nil {
		log.Fatalf("query weapon_stat mappings: %v", err)
	}
	weaponStatNames := []string{}
	for rows.Next() {
		var norm, display, stat sql.NullString
		_ = rows.Scan(&norm, &display, &stat)
		fmt.Printf("  norm=%-30s display=%-40s stat_name=%s\n", norm.String, display.String, stat.String)
		if stat.Valid {
			weaponStatNames = append(weaponStatNames, stat.String)
		}
	}
	rows.Close()
	fmt.Printf("  Total : %d citations weapon_stat\n", len(weaponStatNames))

	fmt.Println("\n=== 2. Mapping weapon_id → name_en disponibles ===")
	wnRows, _ := db.QueryContext(context.Background(),
		"SELECT weapon_id, name_en FROM meta.weapon_labels ORDER BY name_en")
	weaponNames := map[uint64]string{}
	for wnRows.Next() {
		var id uint64
		var name string
		_ = wnRows.Scan(&id, &name)
		weaponNames[id] = name
	}
	wnRows.Close()
	fmt.Printf("  Total : %d weapons connues\n", len(weaponNames))

	// Verifier que chaque stat_name="weapon_kills:X" a un X qui correspond à un name_en lowercased
	fmt.Println("\n=== 3. Cohérence stat_name <-> weapon_labels.name_en ===")
	misses := 0
	matches := 0
	for _, sn := range weaponStatNames {
		if !strings.HasPrefix(sn, "weapon_kills:") {
			fmt.Printf("  [??] stat_name='%s' n'est pas en format weapon_kills:X\n", sn)
			continue
		}
		expected := strings.TrimPrefix(sn, "weapon_kills:")
		found := false
		for _, n := range weaponNames {
			if strings.ToLower(n) == strings.ToLower(expected) {
				found = true
				break
			}
		}
		if found {
			matches++
		} else {
			fmt.Printf("  [MISS] stat_name='%s' (cherché 'weapon_kills:%s') ne trouve aucun weapon_labels.name_en\n", sn, expected)
			misses++
		}
	}
	fmt.Printf("  Cohérence : %d match, %d miss\n", matches, misses)

	if matchID == "" {
		// Trouver un match récent du joueur
		var xuid string
		_ = db.QueryRowContext(context.Background(),
			"SELECT xuid FROM shared.xuid_aliases WHERE gamertag = ? LIMIT 1", playerGT).Scan(&xuid)
		if xuid == "" {
			fmt.Println("\n=== Pas de match_id fourni, pas de xuid → stop ===")
			return
		}
		_ = db.QueryRowContext(context.Background(), `
			SELECT mp.match_id FROM shared.match_participants mp
			WHERE mp.xuid = ? AND mp.kills > 5
			ORDER BY mp.match_id DESC LIMIT 1`, xuid).Scan(&matchID)
	}
	if matchID == "" {
		fmt.Println("\n=== Pas de match candidat → stop ===")
		return
	}

	fmt.Printf("\n=== 4. Pour match %s, kills par arme du joueur %s ===\n", matchID, playerGT)
	var xuid string
	_ = db.QueryRowContext(context.Background(),
		"SELECT xuid FROM shared.xuid_aliases WHERE gamertag = ? LIMIT 1", playerGT).Scan(&xuid)

	wkRows, err := db.QueryContext(context.Background(), `
		SELECT effective_weapon_id, COUNT(*) AS kills
		FROM shared.v_weapon_kills
		WHERE match_id = ? AND xuid = ? AND effective_weapon_id NOT IN (0,1,2)
		GROUP BY effective_weapon_id ORDER BY kills DESC`, matchID, xuid)
	if err != nil {
		log.Fatalf("weapon_kills: %v", err)
	}
	for wkRows.Next() {
		var wid uint64
		var kills int
		_ = wkRows.Scan(&wid, &kills)
		name, ok := weaponNames[wid]
		// Casse canonique (name_en) — alignée avec citations.go:loadWeaponKills
		// et stat_name="weapon_kills:<NameEN>" du Python source.
		statKey := "weapon_kills:" + name
		marker := "[??]"
		matched := ""
		if ok {
			marker = "[OK]"
		}
		// Cette stat key match-elle un mapping ?
		for _, sn := range weaponStatNames {
			if sn == statKey {
				matched = " => active mapping " + sn
				break
			}
		}
		if !ok {
			fmt.Printf("  %s wid=%-22d kills=%-3d (NON résolu — kill ignoré dans stats[])\n", marker, wid, kills)
		} else {
			fmt.Printf("  %s wid=%-22d kills=%-3d name=%-25s stat_key='%s'%s\n", marker, wid, kills, name, statKey, matched)
		}
	}
	wkRows.Close()

	fmt.Printf("\n=== 5. Citations existantes pour match %s ===\n", matchID)
	citRows, err := db.QueryContext(context.Background(), `
		SELECT citation_name_norm, value FROM match_citations
		WHERE match_id = ? ORDER BY value DESC LIMIT 30`, matchID)
	if err != nil {
		fmt.Printf("  pas de table match_citations OU erreur : %v\n", err)
		return
	}
	count := 0
	for citRows.Next() {
		var norm string
		var val int
		_ = citRows.Scan(&norm, &val)
		fmt.Printf("  %s = %d\n", norm, val)
		count++
	}
	citRows.Close()
	fmt.Printf("  Total : %d entrees\n", count)
}

func openDuckDB(path string) *sql.DB {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(%s): %v", path, err)
	}
	return sql.OpenDB(connector)
}

func mustExec(db *sql.DB, q string) {
	if _, err := db.ExecContext(context.Background(), q); err != nil {
		if !strings.Contains(err.Error(), "already") {
			log.Printf("warn exec(%s): %v", q, err)
		}
	}
}
