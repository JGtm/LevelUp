// Package ops — seed_demo_corpus.go : sélection du corpus démo (sessions en
// escouade) + roster multi-joueurs + anonymisation universelle.
//
// La démo n'est plus mono-joueur : on seede un stack de 3 (le joueur source +
// ses 2 coéquipiers les plus fréquents sur les sessions retenues) pour que la
// page Escouade ait de vrais coéquipiers avec leurs stats (perf, LUSR…), tout en
// anonymisant TOUS les participants (xuid + gamertag) — aucune vraie identité ne
// fuite.
package ops

import (
	"context"
	"database/sql"
	"fmt"
)

// DefaultSquadSessions : nombre de sessions en escouade retenues pour le corpus.
const DefaultSquadSessions = 3

// demoRosterEntry décrit le mapping d'un xuid réel vers son identité démo.
type demoRosterEntry struct {
	SourceXUID   string // xuid réel (ou bot/synthétique laissé tel quel)
	DemoXUID     string // xuid anonyme démo (0000…000N)
	DemoGamertag string // "DemoPlayer", "DemoPlayer2", "Player 3", …
	IsRosterMain bool   // true pour le source + les 2 coéquipiers principaux
}

// selectSquadSessionCorpus retourne les match_ids des nSessions sessions en
// escouade (is_with_friends) les plus récentes du joueur source, lues depuis sa
// player DB. Fallback : si aucune session squad, retourne les matchs récents
// classiques (selectRecentMatchIDs côté caller).
func selectSquadSessionCorpus(ctx context.Context, sourcePlayerDBPath string, nSessions int) ([]string, error) {
	db, err := sql.Open("duckdb", sourcePlayerDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open source player DB: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		WITH squad AS (
			SELECT DISTINCT session_id
			FROM player_match_enrichment
			WHERE is_with_friends = TRUE AND session_id IS NOT NULL
		),
		recent AS (
			SELECT s.session_id
			FROM sessions s JOIN squad ON squad.session_id = s.session_id
			ORDER BY s.start_time DESC
			LIMIT ?
		)
		SELECT pme.match_id
		FROM player_match_enrichment pme
		WHERE pme.session_id IN (SELECT session_id FROM recent)`, nSessions)
	if err != nil {
		return nil, fmt.Errorf("query squad corpus: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// buildDemoRoster construit le mapping xuid→identité démo pour un corpus, en
// lisant les participants du shared SOURCE. Ordre :
//   - source → 0000…0000 / "DemoPlayer"
//   - les maxTeammates coéquipiers les plus fréquents (réels, non-bots) →
//     0000…0001 / "DemoPlayer2", 0000…0002 / "DemoPlayer3", …
//   - tous les autres participants réels → "Player N" (anonymes)
//
// Les bots / xuids synthétiques (non 15-16 chiffres) sont exclus du mapping
// (laissés tels quels, pas de vie privée en jeu).
func buildDemoRoster(
	ctx context.Context,
	srcSharedDBPath string,
	matchIDs []string,
	sourceXUID string,
	maxTeammates int,
) ([]demoRosterEntry, error) {
	db, err := sql.Open("duckdb", srcSharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open source shared DB: %w", err)
	}
	defer db.Close()

	idsLit := formatIDsLiteral(matchIDs)
	// Participants réels (xuid 15-16 chiffres) hors source, classés par fréquence.
	q := fmt.Sprintf(`
		SELECT mp.xuid, COUNT(DISTINCT mp.match_id) AS n
		FROM match_participants mp
		WHERE mp.match_id IN (%s)
		  AND mp.xuid <> '%s'
		  AND regexp_matches(mp.xuid, '^[0-9]{15,16}$')
		GROUP BY mp.xuid
		ORDER BY n DESC, mp.xuid`, idsLit, sourceXUID)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows.Close()

	roster := []demoRosterEntry{
		{SourceXUID: sourceXUID, DemoXUID: demoXUIDForIndex(0), DemoGamertag: DefaultDemoGamertag, IsRosterMain: true},
	}
	idx := 1
	for rows.Next() {
		var xuid string
		var n int
		if err := rows.Scan(&xuid, &n); err != nil {
			return nil, err
		}
		e := demoRosterEntry{SourceXUID: xuid, DemoXUID: demoXUIDForIndex(idx)}
		if idx <= maxTeammates {
			e.DemoGamertag = fmt.Sprintf("DemoPlayer%d", idx+1) // DemoPlayer2, DemoPlayer3
			e.IsRosterMain = true
		} else {
			e.DemoGamertag = fmt.Sprintf("Player %d", idx+1)
		}
		roster = append(roster, e)
		idx++
	}
	return roster, rows.Err()
}

// demoXUIDForIndex retourne le xuid démo (16 chiffres) pour un index : 0 →
// "0000000000000000", 1 → "0000000000000001", …
func demoXUIDForIndex(i int) string {
	return fmt.Sprintf("%016d", i)
}

// applyUniversalAnonymization remappe TOUS les xuid + gamertag du shared démo via
// le roster, dans une table temporaire _xuid_map jointe à chaque table cible.
// Couvre : match_participants(xuid,gamertag), medals_earned(xuid),
// weapon_kills(xuid), highlight_events(xuid), killer_victim_pairs(killer/victim
// xuid+gamertag), xuid_aliases(xuid,gamertag).
func applyUniversalAnonymization(ctx context.Context, dst *sql.DB, roster []demoRosterEntry) error {
	if _, err := dst.ExecContext(ctx, `DROP TABLE IF EXISTS _xuid_map`); err != nil {
		return fmt.Errorf("drop map: %w", err)
	}
	if _, err := dst.ExecContext(ctx,
		`CREATE TABLE _xuid_map (old_xuid VARCHAR, new_xuid VARCHAR, new_gamertag VARCHAR)`); err != nil {
		return fmt.Errorf("create map: %w", err)
	}
	for _, e := range roster {
		if _, err := dst.ExecContext(ctx,
			`INSERT INTO _xuid_map VALUES (?, ?, ?)`,
			e.SourceXUID, e.DemoXUID, e.DemoGamertag); err != nil {
			return fmt.Errorf("insert map %s: %w", e.SourceXUID, err)
		}
	}

	// (table, [(xuidCol, gamertagCol)]) — gamertagCol vide = pas de colonne nom.
	type remap struct {
		table string
		pairs [][2]string // {xuidCol, gamertagCol}
	}
	targets := []remap{
		{"match_participants", [][2]string{{"xuid", "gamertag"}}},
		{"xuid_aliases", [][2]string{{"xuid", "gamertag"}}},
		{"medals_earned", [][2]string{{"xuid", ""}}},
		{"weapon_kills", [][2]string{{"xuid", ""}}},
		{"highlight_events", [][2]string{{"xuid", ""}}},
		{"killer_victim_pairs", [][2]string{{"killer_xuid", "killer_gamertag"}, {"victim_xuid", "victim_gamertag"}}},
	}
	for _, t := range targets {
		for _, p := range t.pairs {
			xuidCol, gtCol := p[0], p[1]
			set := fmt.Sprintf("%s = m.new_xuid", xuidCol)
			if gtCol != "" {
				set += fmt.Sprintf(", %s = m.new_gamertag", gtCol)
			}
			stmt := fmt.Sprintf(
				`UPDATE %s SET %s FROM _xuid_map m WHERE %s.%s = m.old_xuid`,
				t.table, set, t.table, xuidCol)
			if _, err := dst.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("anonymize %s.%s: %w", t.table, xuidCol, err)
			}
		}
	}
	_, _ = dst.ExecContext(ctx, `DROP TABLE IF EXISTS _xuid_map`)
	return nil
}

// anonymizeSharedUniversal ouvre la DB shared démo à dbPath et applique le
// remapping universel du roster.
func anonymizeSharedUniversal(ctx context.Context, dbPath string, roster []demoRosterEntry) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open shared %s: %w", dbPath, err)
	}
	defer db.Close()
	return applyUniversalAnonymization(ctx, db, roster)
}
