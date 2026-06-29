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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// DefaultSquadSessions : nombre de sessions en escouade retenues pour le corpus.
const DefaultSquadSessions = 3

// DefaultDemoMainGamertag : gamertag affiché du joueur démo principal. Distinct du
// répertoire de sa player DB (DefaultDemoGamertag = "DEMO", cf. demoDirForIndex).
// Doit rester aligné sur config.DemoRoster[0].Gamertag.
const DefaultDemoMainGamertag = "DemoPlayer"

// DefaultDemoMainSlug : slug de route du joueur démo principal (player_slug). Utilisé
// pour media_files.player_slug — la page Média filtre par auteur = slug courant
// (route param), pas le gamertag. Doit rester aligné sur config.DemoRoster[0].Slug.
const DefaultDemoMainSlug = "demo-player"

// demoRosterEntry décrit le mapping d'un xuid réel vers son identité démo.
type demoRosterEntry struct {
	SourceXUID   string // xuid réel (ou bot/synthétique laissé tel quel)
	DemoXUID     string // xuid anonyme démo (0000…000N)
	DemoGamertag string // "DemoPlayer", "DemoPlayer2", "Player 3", …
	IsRosterMain bool   // true pour le source + les 2 coéquipiers principaux
}

// seededDemoPlayer décrit une player DB démo effectivement seedée (pour écrire
// db_profiles.json multi-profils).
type seededDemoPlayer struct {
	Dir      string // sous-répertoire : DEMO, DEMO2, DEMO3
	XUID     string // xuid démo : 0000…, 0001, 0002
	Gamertag string // DemoPlayer, DemoPlayer2, DemoPlayer3
}

// selectSquadSessionCorpus retourne les match_ids des nSessions sessions en
// escouade (is_with_friends) les plus récentes du joueur source, lues depuis sa
// player DB. Fallback : si aucune session squad, retourne les matchs récents
// classiques (selectRecentMatchIDs côté caller).
//
// Title-robuste : la requête primaire ordonne les sessions par récence via la
// table `sessions` (peuplée côté Halo Infinite). Certains titres (Halo 5) ont des
// session_id dans player_match_enrichment_latest mais une table `sessions` VIDE
// (gap de sync) → la primaire renverrait 0. On bascule alors sur les nSessions
// sessions escouade les plus FOURNIES (nombre de matchs, proxy de « vraies »
// sessions), sans dépendre de la table `sessions`.
func selectSquadSessionCorpus(ctx context.Context, sourcePlayerDBPath string, nSessions int) ([]string, error) {
	db, err := sql.Open("duckdb", sourcePlayerDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open source player DB: %w", err)
	}
	defer db.Close()

	out, err := querySquadCorpusRecent(ctx, db, nSessions)
	if err == nil && len(out) > 0 {
		return out, nil
	}
	if err != nil {
		// La requête primaire (jointure table `sessions`) échoue sur certains titres
		// (ex. Halo 5 : session_id VARCHAR côté enrichment vs INTEGER côté `sessions`) →
		// on bascule sur le fallback title-robuste au lieu de renvoyer 0 session escouade.
		slog.WarnContext(ctx, "seed-demo: squad corpus primaire échoué, fallback biggest", "err", err)
	}
	// Fallback title-robuste : table `sessions` vide/incompatible (ex. Halo 5).
	return querySquadCorpusBiggest(ctx, db, nSessions)
}

// querySquadCorpusRecent : sessions escouade les plus RÉCENTES (via table sessions).
func querySquadCorpusRecent(ctx context.Context, db *sql.DB, nSessions int) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		WITH squad AS (
			SELECT DISTINCT session_id
			FROM player_match_enrichment_latest
			WHERE is_with_friends = TRUE AND session_id IS NOT NULL
		),
		recent AS (
			SELECT s.session_id
			FROM sessions s JOIN squad ON squad.session_id = s.session_id
			ORDER BY s.start_time DESC
			LIMIT ?
		)
		SELECT pme.match_id
		FROM player_match_enrichment_latest pme
		WHERE pme.session_id IN (SELECT session_id FROM recent)`, nSessions)
	if err != nil {
		return nil, fmt.Errorf("query squad corpus: %w", err)
	}
	return scanMatchIDColumn(rows)
}

// querySquadCorpusBiggest : sessions escouade les plus FOURNIES (sans table sessions).
func querySquadCorpusBiggest(ctx context.Context, db *sql.DB, nSessions int) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		WITH squad AS (
			SELECT session_id, COUNT(*) AS n
			FROM player_match_enrichment_latest
			WHERE is_with_friends = TRUE AND session_id IS NOT NULL
			GROUP BY session_id
			ORDER BY n DESC
			LIMIT ?
		)
		SELECT pme.match_id
		FROM player_match_enrichment_latest pme
		WHERE pme.session_id IN (SELECT session_id FROM squad)`, nSessions)
	if err != nil {
		return nil, fmt.Errorf("query squad corpus (fallback biggest): %w", err)
	}
	return scanMatchIDColumn(rows)
}

// scanMatchIDColumn lit une colonne unique de match_id en []string.
func scanMatchIDColumn(rows *sql.Rows) ([]string, error) {
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

// DefaultRankedMatches : nombre de matchs CLASSÉS récents ajoutés au corpus (pour
// que les playlists classées + CSR apparaissent dans recent_playlist_ranks home).
const DefaultRankedMatches = 15

// selectRecentRankedMatchIDs retourne les N matchs CLASSÉS récents du joueur source
// (lus depuis shared). "classé" dérivé comme Q26gPlaylistPhaseBShared : colonne
// is_ranked OU 'ranked' dans playlist_name/pair_name. Sans ces matchs, le corpus
// (Partie rapide only) ne fait apparaître aucune playlist classée côté home.
func selectRecentRankedMatchIDs(ctx context.Context, sharedDBPath, xuid string, limit int) ([]string, error) {
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		  AND (COALESCE(mr.is_ranked, FALSE)
		       OR STRPOS(LOWER(COALESCE(mr.playlist_name, '')), 'ranked') > 0
		       OR STRPOS(LOWER(COALESCE(mr.pair_name, '')), 'ranked') > 0)
		ORDER BY mr.start_time DESC
		LIMIT ?`, xuid, limit)
	if err != nil {
		return nil, fmt.Errorf("query ranked: %w", err)
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

// buildDemoRoster construit le mapping xuid→identité démo. Deux ensembles de
// match_ids sont fournis :
//   - rankMatchIDs : corpus servant à CLASSER les coéquipiers principaux (les 3
//     sessions en escouade) — garantit que DemoPlayer2/3 sont les vrais partenaires
//     d'escouade, pas des adversaires fréquents des matchs solo ajoutés au corpus.
//   - allMatchIDs : corpus COMPLET (escouade + solo récents) dont TOUS les
//     participants réels doivent être anonymisés (aucune vraie identité ne fuite).
//
// Ordre :
//   - source → 0000…0000 / "DemoPlayer"
//   - les maxTeammates coéquipiers les plus fréquents du corpus escouade →
//     0000…0001 / "DemoPlayer2", 0000…0002 / "DemoPlayer3", …
//   - tous les autres participants réels du corpus complet → "Player N" (anonymes)
//
// Les bots / xuids synthétiques (non 15-16 chiffres) sont exclus du mapping
// (laissés tels quels, pas de vie privée en jeu).
func buildDemoRoster(
	ctx context.Context,
	srcSharedDBPath string,
	rankMatchIDs, allMatchIDs []string,
	sourceXUID string,
	maxTeammates int,
) ([]demoRosterEntry, error) {
	if len(rankMatchIDs) == 0 {
		rankMatchIDs = allMatchIDs // pas de corpus escouade → classer sur tout
	}
	db, err := sql.Open("duckdb", srcSharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open source shared DB: %w", err)
	}
	defer db.Close()

	// Coéquipiers principaux : top-N du corpus escouade.
	ranked, err := queryParticipantsByFreq(ctx, db, rankMatchIDs, sourceXUID)
	if err != nil {
		return nil, err
	}
	priorities := ranked
	if len(priorities) > maxTeammates {
		priorities = priorities[:maxTeammates]
	}
	priorityRank := make(map[string]int, len(priorities))
	for i, x := range priorities {
		priorityRank[x] = i + 1 // 1 → DemoPlayer2, 2 → DemoPlayer3
	}

	// Tous les participants réels du corpus complet (à anonymiser).
	all, err := queryParticipantsByFreq(ctx, db, allMatchIDs, sourceXUID)
	if err != nil {
		return nil, err
	}

	roster := []demoRosterEntry{
		{SourceXUID: sourceXUID, DemoXUID: demoXUIDForIndex(0), DemoGamertag: DefaultDemoMainGamertag, IsRosterMain: true},
	}
	for _, x := range priorities {
		rank := priorityRank[x]
		roster = append(roster, demoRosterEntry{
			SourceXUID:   x,
			DemoXUID:     demoXUIDForIndex(rank),
			DemoGamertag: fmt.Sprintf("DemoPlayer%d", rank+1), // DemoPlayer2, DemoPlayer3
			IsRosterMain: true,
		})
	}
	idx := len(priorities) + 1
	for _, x := range all {
		if _, isPriority := priorityRank[x]; isPriority {
			continue
		}
		roster = append(roster, demoRosterEntry{
			SourceXUID:   x,
			DemoXUID:     demoXUIDForIndex(idx),
			DemoGamertag: fmt.Sprintf("Player %d", idx+1),
		})
		idx++
	}
	return roster, nil
}

// queryParticipantsByFreq retourne les xuid réels (15-16 chiffres, hors source)
// présents dans matchIDs, classés par nombre de matchs décroissant (ordre stable).
func queryParticipantsByFreq(ctx context.Context, db *sql.DB, matchIDs []string, sourceXUID string) ([]string, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	q := fmt.Sprintf(`
		SELECT mp.xuid, COUNT(DISTINCT mp.match_id) AS n
		FROM match_participants mp
		WHERE mp.match_id IN (%s)
		  AND mp.xuid <> '%s'
		  AND regexp_matches(mp.xuid, '^[0-9]{15,16}$')
		GROUP BY mp.xuid
		ORDER BY n DESC, mp.xuid`, formatIDsLiteral(matchIDs), sourceXUID)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var xuid string
		var n int
		if err := rows.Scan(&xuid, &n); err != nil {
			return nil, err
		}
		out = append(out, xuid)
	}
	return out, rows.Err()
}

// unionMatchIDs fusionne plusieurs listes de match_ids en préservant l'ordre de
// première apparition (dédupliqué). La 1re liste passée est prioritaire pour l'ordre.
func unionMatchIDs(lists ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, l := range lists {
		for _, id := range l {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
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
		// Tables Halo 5-spécifiques (absentes côté Infinite → tolérées : table
		// inexistante = skip, cf. errIsMissingTable). Anonymise leur(s) colonne(s)
		// d'identité pour qu'aucun vrai xuid ne fuite.
		{"match_commendations", [][2]string{{"xuid", ""}}},
		{"kill_positions", [][2]string{{"killer_xuid", ""}}},
		{"weapon_accuracy", [][2]string{{"xuid", ""}}},
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
				if errIsMissingTable(err) {
					// Table H5-spécifique absente de cette démo (titre Infinite) → skip.
					continue
				}
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

// rosterMains retourne les entrées principales (source + 2 coéquipiers les plus
// fréquents) — celles qui auront leur propre player DB démo seedée.
func rosterMains(roster []demoRosterEntry) []demoRosterEntry {
	var out []demoRosterEntry
	for _, e := range roster {
		if e.IsRosterMain {
			out = append(out, e)
		}
	}
	return out
}

// demoDirForIndex retourne le sous-répertoire player démo : 0 → "DEMO",
// 1 → "DEMO2", 2 → "DEMO3".
func demoDirForIndex(i int) string {
	if i == 0 {
		return DefaultDemoGamertag
	}
	return fmt.Sprintf("%s%d", DefaultDemoGamertag, i+1)
}

// resolvePlayerDBByXUID lit db_profiles.json (v3.0 nested par titre, ou v2.1 plat)
// et retourne le chemin DB relatif du joueur ayant ce xuid. found=false si absent.
func resolvePlayerDBByXUID(profilesPath, xuid string) (dbRelPath string, found bool, err error) {
	data, rerr := os.ReadFile(profilesPath)
	if rerr != nil {
		return "", false, fmt.Errorf("read profiles: %w", rerr)
	}
	// v3.0 : profiles.{titleSlug}.{gamertag}.{xuid, db_path}
	var v3 struct {
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(data, &v3) == nil {
		for _, byGamertag := range v3.Profiles {
			for _, p := range byGamertag {
				if p.XUID == xuid && p.DBPath != "" {
					return p.DBPath, true, nil
				}
			}
		}
	}
	// v2.1 : profiles.{gamertag}.{xuid, db_path}
	var v2 struct {
		Profiles map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(data, &v2) == nil {
		for _, p := range v2.Profiles {
			if p.XUID == xuid && p.DBPath != "" {
				return p.DBPath, true, nil
			}
		}
	}
	return "", false, nil
}
