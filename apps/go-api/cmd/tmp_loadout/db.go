package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

// dbCrossCheck looks up the match whose id starts with 000d5950 and pulls, per
// participant xuid, the kills-by-weapon distribution (killer_victim_pairs) to
// cross-reference the keyframe loadout table with who actually wielded each weapon.
func dbCrossCheck() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	// 1. Resolve the match_id.
	var matchID string
	err = db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&matchID)
	if err != nil {
		fmt.Println("match lookup:", err)
		// list a few to see id format
		rows, _ := db.Query(`SELECT match_id FROM match_registry LIMIT 5`)
		if rows != nil {
			fmt.Println("exemples match_id:")
			for rows.Next() {
				var m string
				rows.Scan(&m)
				fmt.Println("  ", m)
			}
			rows.Close()
		}
		return
	}
	fmt.Printf("match_id = %s\n", matchID)

	// 2. Roster : xuid -> gamertag (xuid_aliases) + team + kills.
	fmt.Printf("\n--- Roster du match (team, xuid, gamertag, kills) ---\n")
	rows, err := db.Query(`
		SELECT mp.team_id, mp.xuid, COALESCE(xa.gamertag,'?') AS gt, mp.kills, mp.deaths, mp.power_weapon_kills, mp.melee_kills
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON CAST(xa.xuid AS VARCHAR) = CAST(mp.xuid AS VARCHAR)
		WHERE mp.match_id = ?
		ORDER BY mp.team_id, mp.kills DESC`, matchID)
	if err != nil {
		fmt.Println("roster:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var team, kills, deaths, pwk, melee int
		var xuid, gt string
		rows.Scan(&team, &xuid, &gt, &kills, &deaths, &pwk, &melee)
		pi := piLabel(xuid)
		fmt.Printf("  team=%d  xuid=%-18s %-16s %-8s K=%-2d D=%-2d powerWpnK=%d meleeK=%d\n",
			team, xuid, gt, pi, kills, deaths, pwk, melee)
	}
}

// dbRosterOrder prints participants in physical/insertion order (no ORDER BY) to
// compare the player-slot order against the keyframe record order 0..7.
func dbRosterOrder() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()
	const matchID = "000d5950-83d9-423f-ab55-d068a7237b9f"
	fmt.Printf("\n--- Ordre physique des participants (rowid / created_at) ---\n")
	rows, err := db.Query(`
		SELECT mp.team_id, mp.xuid, COALESCE(xa.gamertag,'?'), mp.created_at
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON CAST(xa.xuid AS VARCHAR)=CAST(mp.xuid AS VARCHAR)
		WHERE mp.match_id = ?
		ORDER BY mp.created_at, mp.rowid`, matchID)
	if err != nil {
		fmt.Println("order:", err)
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var team int
		var xuid, gt, ts string
		rows.Scan(&team, &xuid, &gt, &ts)
		fmt.Printf("  slot#%d team=%d %-18s %-16s %s  ts=%s\n", i, team, xuid, gt, piLabel(xuid), ts)
		i++
	}
}

// piLabel maps a xuid string back to its pi index (from the bit-verified table).
func piLabel(xuid string) string {
	for i, x := range piXUID {
		if fmt.Sprintf("%d", x) == xuid {
			return fmt.Sprintf("pi%d", i)
		}
	}
	return "pi?"
}

// dbMedals lists medals per player for the match — some medals are weapon-specific
// (sniper/rocket/etc.) and can corroborate a loadout->player assignment.
func dbMedals() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()
	const matchID = "000d5950-83d9-423f-ab55-d068a7237b9f"
	schemaCols(db, "medals_earned")
	fmt.Printf("\n--- Médailles par joueur ---\n")
	rows, err := db.Query(`
		SELECT me.player_xuid, COALESCE(xa.gamertag,'?'), me.medal_name_id, me.count
		FROM medals_earned me
		LEFT JOIN xuid_aliases xa ON CAST(xa.xuid AS VARCHAR)=CAST(me.player_xuid AS VARCHAR)
		WHERE me.match_id = ?
		ORDER BY me.player_xuid, me.count DESC`, matchID)
	if err != nil {
		fmt.Println("medals:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var xuid, gt string
		var medal, cnt sql.NullString
		rows.Scan(&xuid, &gt, &medal, &cnt)
		fmt.Printf("  %-16s %-8s medal=%-12s x%s\n", gt, piLabel(xuid), medal.String, cnt.String)
	}
}

func schemaCols(db *sql.DB, table string) {
	fmt.Printf("\n[schema %s]\n", table)
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		fmt.Println(" schema err:", err)
		return
	}
	for rows.Next() {
		var c, t string
		rows.Scan(&c, &t)
		fmt.Printf("  %-28s %s\n", c, t)
	}
	rows.Close()
}
