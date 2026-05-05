//go:build cgo

// diag_squad_weapons — diagnostic one-shot pour reproduire le path
// LoadWeaponKills(MatchIDs, XUIDs) utilise par buildSquadWeaponKills.
//
// Usage : go run -tags cgo ./cmd/diag_squad_weapons <main_gt> <teammate_gt>
//
// Ouvre la DB du main, ATTACH shared, liste les matchs ou le main + teammate
// ont joue ensemble (via shared.match_participants), puis execute la requete
// weapon_kills exacte que LoadWeaponKills lance. Affiche les rows et la
// presence des tables/views.
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
	if len(os.Args) < 3 {
		log.Fatalf("usage: diag_squad_weapons <main_gt> <teammate_gt>")
	}
	mainGT := os.Args[1]
	teammateGT := os.Args[2]

	mainStats := filepath.Join("..", "..", "data", "titles", "halo_infinite", "players", mainGT, "stats.duckdb")
	sharedPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	globalAliases := filepath.Join("..", "..", "data", "global", "xuid_aliases.duckdb")
	if _, err := os.Stat(mainStats); err != nil {
		log.Fatalf("DB main introuvable %s: %v", mainStats, err)
	}
	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable %s: %v", sharedPath, err)
	}

	db := openDuckDB(mainStats)
	defer db.Close()

	mustExec(db, fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath))
	if _, err := os.Stat(globalAliases); err == nil {
		mustExec(db, fmt.Sprintf("ATTACH '%s' AS global (READ_ONLY)", globalAliases))
	}

	fmt.Printf("=== Tables/Views in shared catalog ===\n")
	listShared(db)

	mainXUID := lookupXUID(db, mainGT)
	tmXUID := lookupXUID(db, teammateGT)
	fmt.Printf("\n%s xuid=%s\n%s xuid=%s\n", mainGT, mainXUID, teammateGT, tmXUID)

	matches := loadSharedMatches(db, mainXUID, tmXUID)
	fmt.Printf("\n=== Matchs partages %s+%s : %d ===\n", mainGT, teammateGT, len(matches))
	for i, m := range matches {
		if i >= 5 {
			fmt.Printf("  ... (%d more)\n", len(matches)-i)
			break
		}
		fmt.Printf("  %s\n", m)
	}

	if len(matches) == 0 {
		fmt.Println("Pas de matchs partages -> stop ici")
		return
	}

	fmt.Println("\n=== Test LoadWeaponKills (memes filtres que buildSquadWeaponKills) ===")
	rows := queryWeaponKills(db, matches, []string{mainXUID, tmXUID})
	fmt.Printf("Total rows : %d\n", len(rows))
	byPlayer := map[string]int{}
	for _, r := range rows {
		byPlayer[r.xuid] += r.kills
	}
	for xuid, total := range byPlayer {
		gt := mainGT
		if xuid == tmXUID {
			gt = teammateGT
		}
		fmt.Printf("  %s (%s) -> %d kills agreges sur %d matchs\n", gt, xuid, total, len(matches))
	}
	if len(rows) > 0 {
		fmt.Println("\n  Premiers 10 rows :")
		for i, r := range rows {
			if i >= 10 {
				break
			}
			fmt.Printf("    xuid=%s weapon_id=%d kills=%d gm=%v\n", r.xuid, r.weaponID, r.kills, r.isGM)
		}
	}

	// Resolution labels — ATTACH metadata sur la meme connexion.
	fmt.Println("\n=== Resolution weapon_labels ===")
	metaPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")
	if _, err := db.ExecContext(context.Background(),
		fmt.Sprintf("ATTACH '%s' AS meta (READ_ONLY)", metaPath)); err != nil {
		log.Printf("ATTACH meta failed: %v", err)
	}
	metaDB := db

	// Distinct weapon_ids dans les rows non-grenade/melee
	uniq := map[uint64]struct{}{}
	for _, r := range rows {
		if r.isGM {
			continue
		}
		uniq[r.weaponID] = struct{}{}
	}
	fmt.Printf("Distinct weapon_ids dans le sample : %d\n", len(uniq))

	// Lookup — utilise sql.NullString pour distinguer not-found vs erreur SQL
	resolved := 0
	unresolved := []uint64{}
	var firstLookupErr error
	for id := range uniq {
		var name sql.NullString
		err := metaDB.QueryRowContext(context.Background(),
			fmt.Sprintf("SELECT name_fr FROM meta.weapon_labels WHERE weapon_id = %d", id)).Scan(&name)
		if err == sql.ErrNoRows {
			unresolved = append(unresolved, id)
			continue
		}
		if err != nil {
			if firstLookupErr == nil {
				firstLookupErr = err
				fmt.Printf("  ERREUR lookup weapon_id=%d : %v\n", id, err)
			}
			unresolved = append(unresolved, id)
			continue
		}
		if name.Valid && name.String != "" {
			resolved++
		} else {
			unresolved = append(unresolved, id)
		}
	}
	fmt.Printf("Resolus : %d / %d\n", resolved, len(uniq))
	fmt.Printf("Non resolus : %d\n", len(unresolved))
	for i, id := range unresolved {
		if i >= 10 {
			fmt.Printf("  ... (%d more)\n", len(unresolved)-i)
			break
		}
		// Inspecter reconciled_as et weapon_id raw pour ce id
		fmt.Printf("  weapon_id=%d (hex=0x%016x)\n", id, id)
	}

	// Coverage table : combien de weapons distincts dans weapon_labels ?
	var totalLabels int
	if err := metaDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM meta.weapon_labels").Scan(&totalLabels); err != nil {
		fmt.Printf("\nERREUR lecture meta.weapon_labels : %v\n", err)
	} else {
		fmt.Printf("\nTotal entries dans metadata.weapon_labels : %d\n", totalLabels)
	}

	// Etat migration add_weapon_labels — tracking peut etre en player DB
	fmt.Println("\n=== schema_migrations status (player DB et metadata) ===")
	for _, schema := range []string{"main", "meta"} {
		fmt.Printf("  -- catalog=%s\n", schema)
		q := fmt.Sprintf(`SELECT name, schema_done, backfill_done, applied_at FROM %s.schema_migrations
			WHERE name LIKE '%%weapon%%' ORDER BY name`, schema)
		migRows, err := metaDB.QueryContext(context.Background(), q)
		if err != nil {
			fmt.Printf("    err: %v\n", err)
			continue
		}
		count := 0
		for migRows.Next() {
			var name string
			var schemaDone, backfillDone bool
			var appliedAt string
			_ = migRows.Scan(&name, &schemaDone, &backfillDone, &appliedAt)
			fmt.Printf("    %s : schema_done=%v backfill_done=%v applied=%s\n", name, schemaDone, backfillDone, appliedAt)
			count++
		}
		migRows.Close()
		if count == 0 {
			fmt.Printf("    (aucune migration weapon dans %s.schema_migrations)\n", schema)
		}
	}

	// Top weapon_ids avec kill count, joints aux labels — vue globale
	fmt.Println("\n=== Top 30 weapons globaux (squad+main) avec kills + label ===")
	topRows, _ := metaDB.QueryContext(context.Background(), `
		SELECT effective_weapon_id, SUM(1) AS kills,
		       (SELECT name_fr FROM meta.weapon_labels wl WHERE wl.weapon_id = effective_weapon_id) AS label
		FROM shared.v_weapon_kills
		WHERE effective_weapon_id NOT IN (0,1,2)
		GROUP BY effective_weapon_id
		ORDER BY kills DESC
		LIMIT 30
	`)
	if topRows != nil {
		for topRows.Next() {
			var wid uint64
			var kills int
			var label sql.NullString
			_ = topRows.Scan(&wid, &kills, &label)
			tag := "[OK]"
			lbl := ""
			if label.Valid && label.String != "" {
				lbl = label.String
			} else {
				tag = "[??]"
			}
			fmt.Printf("  %s wid=%-22d 0x%016x kills=%-8d %s\n", tag, wid, wid, kills, lbl)
		}
		topRows.Close()
	}

	// Liste des autres tables dans meta pour confirmer la migration
	fmt.Println("\n=== Tables dans meta catalog ===")
	tRows, _ := metaDB.QueryContext(context.Background(), `
		SELECT table_name FROM information_schema.tables
		WHERE table_catalog = 'meta' ORDER BY table_name`)
	if tRows != nil {
		for tRows.Next() {
			var name string
			_ = tRows.Scan(&name)
			fmt.Printf("  %s\n", name)
		}
		tRows.Close()
	}

	// Verifier reconciled_as : combien de rows weapon_kills ont reconciled_as NULL pour les ids non resolus
	if len(unresolved) > 0 {
		fmt.Println("\n=== reconciled_as status pour les ids non resolus (sample) ===")
		for i, id := range unresolved {
			if i >= 5 {
				break
			}
			var totalRows, withRecon, withoutRecon int
			_ = db.QueryRowContext(context.Background(),
				fmt.Sprintf(`SELECT COUNT(*), COUNT(reconciled_as), COUNT(*) - COUNT(reconciled_as)
					FROM shared.weapon_kills WHERE COALESCE(reconciled_as, weapon_id) = %d`, id)).
				Scan(&totalRows, &withRecon, &withoutRecon)
			fmt.Printf("  effective_id=%d : %d rows total, %d ont reconciled_as, %d ne l'ont pas\n",
				id, totalRows, withRecon, withoutRecon)
		}
	}
}

type wkRow struct {
	xuid     string
	weaponID uint64
	kills    int
	isGM     bool
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
		// L'ATTACH peut deja etre en place dans un autre worker; ignore "already exists"
		if !strings.Contains(err.Error(), "already") {
			log.Printf("warn exec(%s): %v", q, err)
		}
	}
}

func listShared(db *sql.DB) {
	rs, err := db.QueryContext(context.Background(), `
		SELECT table_type, table_name FROM information_schema.tables
		WHERE table_catalog = 'shared'
		  AND table_name LIKE '%weapon%'
		ORDER BY table_name
	`)
	if err != nil {
		log.Printf("listShared: %v", err)
		return
	}
	defer rs.Close()
	for rs.Next() {
		var ty, nm string
		_ = rs.Scan(&ty, &nm)
		fmt.Printf("  %-15s %s\n", ty, nm)
	}
}

func lookupXUID(db *sql.DB, gt string) string {
	var xuid string
	err := db.QueryRowContext(context.Background(),
		`SELECT xuid FROM shared.xuid_aliases WHERE gamertag = ? LIMIT 1`, gt).Scan(&xuid)
	if err == nil {
		return xuid
	}
	// fallback global
	err = db.QueryRowContext(context.Background(),
		`SELECT xuid FROM global.xuid_aliases WHERE gamertag = ? LIMIT 1`, gt).Scan(&xuid)
	if err != nil {
		log.Printf("lookupXUID(%s): %v", gt, err)
	}
	return xuid
}

func loadSharedMatches(db *sql.DB, mainXUID, tmXUID string) []string {
	rs, err := db.QueryContext(context.Background(), `
		SELECT mp1.match_id
		FROM shared.match_participants mp1
		JOIN shared.match_participants mp2 USING (match_id)
		WHERE mp1.xuid = ? AND mp2.xuid = ?
		GROUP BY mp1.match_id
	`, mainXUID, tmXUID)
	if err != nil {
		log.Printf("loadSharedMatches: %v", err)
		return nil
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var id string
		_ = rs.Scan(&id)
		out = append(out, id)
	}
	return out
}

// Reproduit exactement buildWeaponKillsQuery (weapon_kills_repo.go) avec
// IncludeGrenadeMelee=true.
func queryWeaponKills(db *sql.DB, matchIDs, xuids []string) []wkRow {
	mph := strings.Repeat(",?", len(matchIDs))[1:]
	xph := strings.Repeat(",?", len(xuids))[1:]

	// Branche 1 : weapon_id reste UBIGINT (pas de cast ::BIGINT — overflow sur
	// les hash filmshell bit63=1). Scan en uint64 cote Go.
	q := `
SELECT wk.xuid, wk.effective_weapon_id, COUNT(*), FALSE
FROM shared.v_weapon_kills wk
WHERE wk.match_id IN (` + mph + `)
  AND wk.effective_weapon_id NOT IN (0,1,2)
  AND wk.xuid IN (` + xph + `)
GROUP BY wk.xuid, wk.effective_weapon_id
UNION ALL
SELECT mp.xuid, 0::UBIGINT, SUM(COALESCE(mp.grenade_kills,0))::INTEGER, TRUE
FROM shared.match_participants mp
WHERE mp.match_id IN (` + mph + `)
  AND mp.xuid IN (` + xph + `)
GROUP BY mp.xuid
HAVING SUM(COALESCE(mp.grenade_kills,0)) > 0
UNION ALL
SELECT mp.xuid, 1::UBIGINT, SUM(COALESCE(mp.melee_kills,0))::INTEGER, TRUE
FROM shared.match_participants mp
WHERE mp.match_id IN (` + mph + `)
  AND mp.xuid IN (` + xph + `)
GROUP BY mp.xuid
HAVING SUM(COALESCE(mp.melee_kills,0)) > 0
`

	args := make([]any, 0, 3*(len(matchIDs)+len(xuids)))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	for _, x := range xuids {
		args = append(args, x)
	}
	for _, id := range matchIDs {
		args = append(args, id)
	}
	for _, x := range xuids {
		args = append(args, x)
	}
	for _, id := range matchIDs {
		args = append(args, id)
	}
	for _, x := range xuids {
		args = append(args, x)
	}

	rs, err := db.QueryContext(context.Background(), q, args...)
	if err != nil {
		log.Printf("queryWeaponKills: %v", err)
		return nil
	}
	defer rs.Close()
	var out []wkRow
	for rs.Next() {
		var r wkRow
		if err := rs.Scan(&r.xuid, &r.weaponID, &r.kills, &r.isGM); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		out = append(out, r)
	}
	return out
}
