// Outil ops : RECONSTRUIT OFFLINE les participants Halo 5 droppés à l'ingest
// (rosters incomplets), depuis le kill-feed local (killer_victim_pairs) — sans
// API ni token. INSERT-only strict (ART-safe, jamais d'UPDATE/DELETE indexé).
//
// CAUSE RACINE : en H5, match_participants (issu de la carnage gamertag-keyée)
// a droppé les joueurs dont le gamertag ne résolvait pas en xuid. Le kill-feed
// (issu des EVENTS) porte xuid ET gamertag de TOUS les joueurs → l'identité des
// droppés est déjà LOCALE. Le drop déséquilibre les rosters (2v4 au lieu de 4v4)
// → le moteur LUSR skip ces matchs (bucket skippedImbalance, |teamA-teamB|>1,
// cf. internal/sync/skill_v2_shadow.go). On rétablit les rosters par inférence
// d'équipe sur le kill-graph (2-coloration), zéro réseau.
//
// Usage :
//
//	LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-roster-topup            # DRY-RUN
//	LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-roster-topup --commit   # écrit
//	... --match <id>     # cible un seul match (test/diagnostic, détail complet)
//	... --title halo_5   # défaut halo_5 (l'outil est H5-only par construction)
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

func main() {
	commit := flag.Bool("commit", false, "écrit réellement (défaut false = DRY-RUN, n'écrit rien)")
	matchID := flag.String("match", "", "cible un seul match (optionnel, détail complet)")
	titleSlug := flag.String("title", "halo_5", "slug du titre (défaut halo_5 ; outil H5-only par construction)")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(*titleSlug)
	fmt.Printf("Titre=%s shared=%s commit=%v\n", *titleSlug, sharedPath, *commit)
	if !*commit {
		fmt.Println("[DRY-RUN] aucune écriture — affiche ce qui SERAIT inséré.")
	}

	shared := openDB(sharedPath)
	defer shared.Close()

	candidates, err := loadCandidateMatches(ctx, shared, *matchID)
	if err != nil {
		fatal("loadCandidateMatches: %v", err)
	}
	fmt.Printf("Matchs candidats (roster < kill-feed, 2-équipes, non-firefight) : %d\n", len(candidates))

	var (
		nReconstructible int
		nResidual        int
		nInsertedPlayers int
		nInsertedMatches int
	)
	residualByReason := map[string]int{}
	detail := *matchID != ""
	for _, mid := range candidates {
		res, err := buildReconstruction(ctx, shared, mid)
		if err != nil {
			fmt.Printf("  %s : ERREUR %v (ignoré)\n", mid, err)
			continue
		}
		if res.residualReason != "" {
			nResidual++
			residualByReason[res.residualReason]++
			if detail || nResidual <= 20 {
				fmt.Printf("  RÉSIDU %s : %s — droppés=%d\n", mid, res.residualReason, len(res.dropped))
			}
			continue
		}
		nReconstructible++
		nInsertedPlayers += len(res.toInsert)
		if detail {
			printDetail(res)
		}
		if *commit {
			if err := commitMatch(ctx, shared, res); err != nil {
				fmt.Printf("  %s : COMMIT échoué %v\n", mid, err)
				continue
			}
			nInsertedMatches++
		}
	}

	fmt.Printf("\n=== BILAN ===\n")
	fmt.Printf("Scannés        : %d\n", len(candidates))
	fmt.Printf("Reconstructibles: %d match(s), %d joueur(s) à insérer\n", nReconstructible, nInsertedPlayers)
	fmt.Printf("Résidus        : %d match(s) (à re-fetch)\n", nResidual)
	if nResidual > 0 {
		// Détail des raisons de résidu, trié pour une sortie déterministe.
		reasons := make([]string, 0, len(residualByReason))
		for r := range residualByReason {
			reasons = append(reasons, r)
		}
		sort.Strings(reasons)
		for _, r := range reasons {
			fmt.Printf("  - %s : %d\n", r, residualByReason[r])
		}
	}
	if *commit {
		fmt.Printf("ÉCRITS         : %d match(s) (INSERT-only, garde NOT EXISTS)\n", nInsertedMatches)
	} else {
		fmt.Printf("[DRY-RUN] rien écrit. Relancer avec --commit pour appliquer.\n")
	}
}

// reconstruction : résultat d'analyse d'UN match (avant écriture).
type reconstruction struct {
	matchID        string
	teamA, teamB   int
	knowns         []knownParticipant
	dropped        []droppedPlayer // tous les droppés détectés (avant inférence)
	toInsert       []droppedPlayer // droppés reconstructibles (équipe+outcome inférés)
	residualReason string          // != "" → match en résidu, rien à insérer
}

// loadCandidateMatches sélectionne les matchs CANDIDATS :
//   - 2-équipes (exactement 2 team_id distincts dans match_participants),
//   - nb de joueurs distincts au kill-feed (killer_xuid ∪ victim_xuid, non vides)
//     STRICTEMENT supérieur au COUNT(match_participants),
//   - hors firefight.
//
// NB : volontairement PAS limité à is_ranked — le LUSR couvre aussi le non-classé.
func loadCandidateMatches(ctx context.Context, shared *sql.DB, only string) ([]string, error) {
	const q = `
		WITH mp AS (
			SELECT match_id,
			       COUNT(*)                  AS n_participants,
			       COUNT(DISTINCT team_id)   AS n_teams
			FROM match_participants
			GROUP BY match_id
		),
		kf AS (
			SELECT match_id, COUNT(DISTINCT x) AS n_feed
			FROM (
				SELECT match_id, killer_xuid AS x FROM killer_victim_pairs
				WHERE killer_xuid IS NOT NULL AND killer_xuid <> ''
				UNION
				SELECT match_id, victim_xuid AS x FROM killer_victim_pairs
				WHERE victim_xuid IS NOT NULL AND victim_xuid <> ''
			)
			GROUP BY match_id
		)
		SELECT mp.match_id
		FROM mp
		JOIN kf ON kf.match_id = mp.match_id
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.n_teams = 2
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND kf.n_feed > mp.n_participants
		  AND (? = '' OR mp.match_id = ?)
		ORDER BY mp.match_id`
	rows, err := shared.QueryContext(ctx, q, only, only)
	if err != nil {
		return nil, err
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

// buildReconstruction charge les participants connus + le kill-feed d'un match,
// détecte les droppés, infère leur équipe et vérifie l'équilibre.
func buildReconstruction(ctx context.Context, shared *sql.DB, matchID string) (reconstruction, error) {
	res := reconstruction{matchID: matchID}

	knowns, knownSet, err := loadKnownParticipants(ctx, shared, matchID)
	if err != nil {
		return res, err
	}
	res.knowns = knowns
	if len(knowns) == 0 {
		res.residualReason = "aucun participant connu (impossible d'ancrer les equipes)"
		return res, nil
	}

	// Les deux team_id présents.
	teamSet := map[int]struct{}{}
	for _, k := range knowns {
		teamSet[k.TeamID] = struct{}{}
	}
	if len(teamSet) != 2 {
		res.residualReason = fmt.Sprintf("%d equipes connues (attendu 2)", len(teamSet))
		return res, nil
	}
	teams := make([]int, 0, 2)
	for t := range teamSet {
		teams = append(teams, t)
	}
	sort.Ints(teams)
	res.teamA, res.teamB = teams[0], teams[1]

	edges, gtByXUID, killsByXUID, deathsByXUID, err := loadKillGraph(ctx, shared, matchID)
	if err != nil {
		return res, err
	}

	// Droppés = xuid au kill-feed mais absents de match_participants.
	for x, gt := range gtByXUID {
		if _, known := knownSet[x]; known {
			continue
		}
		res.dropped = append(res.dropped, droppedPlayer{
			XUID:     x,
			Gamertag: gt,
			Kills:    killsByXUID[x],
			Deaths:   deathsByXUID[x],
		})
	}
	sort.Slice(res.dropped, func(i, j int) bool { return res.dropped[i].XUID < res.dropped[j].XUID })
	if len(res.dropped) == 0 {
		res.residualReason = "aucun droppe (roster deja complet vs kill-feed)"
		return res, nil
	}

	toInsert, reason := inferDroppedRoster(res.knowns, res.dropped, res.teamA, res.teamB, edges)
	if reason != "" {
		res.residualReason = reason
		return res, nil
	}
	res.toInsert = toInsert
	return res, nil
}

// loadKnownParticipants charge les participants PERSISTÉS d'un match.
func loadKnownParticipants(ctx context.Context, shared *sql.DB, matchID string) ([]knownParticipant, map[string]struct{}, error) {
	rows, err := shared.QueryContext(ctx, `
		SELECT xuid, COALESCE(team_id, -1), COALESCE(outcome, 0)
		FROM match_participants
		WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''`, matchID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []knownParticipant
	set := map[string]struct{}{}
	for rows.Next() {
		var k knownParticipant
		if err := rows.Scan(&k.XUID, &k.TeamID, &k.Outcome); err != nil {
			return nil, nil, err
		}
		out = append(out, k)
		set[k.XUID] = struct{}{}
	}
	return out, set, rows.Err()
}

// loadKillGraph agrège le kill-feed d'un match : arêtes (killer→victim, poids =
// SUM kill_count), gamertag par xuid, kills/deaths par xuid.
//
// On SOMME kill_count (et pas COUNT(*)) pour être robuste à une éventuelle forme
// agrégée par-paire : en H5 chaque kill = 1 row Count=1 (cf. ingest/kills.go),
// donc SUM(kill_count) == nb de kills dans les deux cas.
func loadKillGraph(ctx context.Context, shared *sql.DB, matchID string) (
	edges []killEdge, gtByXUID map[string]string, killsByXUID, deathsByXUID map[string]int, err error,
) {
	gtByXUID = map[string]string{}
	killsByXUID = map[string]int{}
	deathsByXUID = map[string]int{}

	rows, qerr := shared.QueryContext(ctx, `
		SELECT
			COALESCE(killer_xuid, '')     AS k_xuid,
			COALESCE(killer_gamertag, '') AS k_gt,
			COALESCE(victim_xuid, '')     AS v_xuid,
			COALESCE(victim_gamertag, '') AS v_gt,
			SUM(COALESCE(kill_count, 1))  AS w
		FROM killer_victim_pairs
		WHERE match_id = ?
		GROUP BY 1, 2, 3, 4`, matchID)
	if qerr != nil {
		return nil, nil, nil, nil, qerr
	}
	defer rows.Close()
	for rows.Next() {
		var kx, kg, vx, vg string
		var w int
		if scanErr := rows.Scan(&kx, &kg, &vx, &vg, &w); scanErr != nil {
			return nil, nil, nil, nil, scanErr
		}
		if w <= 0 {
			w = 1
		}
		if kx != "" {
			if kg != "" {
				gtByXUID[kx] = kg
			} else if _, ok := gtByXUID[kx]; !ok {
				gtByXUID[kx] = ""
			}
			killsByXUID[kx] += w
		}
		if vx != "" {
			if vg != "" {
				gtByXUID[vx] = vg
			} else if _, ok := gtByXUID[vx]; !ok {
				gtByXUID[vx] = ""
			}
			deathsByXUID[vx] += w
		}
		if kx != "" && vx != "" {
			edges = append(edges, killEdge{KillerXUID: kx, VictimXUID: vx, Weight: w})
		}
	}
	return edges, gtByXUID, killsByXUID, deathsByXUID, rows.Err()
}

// commitMatch insère les droppés reconstruits dans match_participants + leurs
// alias, en une TRANSACTION par match (BEGIN→INSERT×N→COMMIT). INSERT-only
// strict, garde NOT EXISTS sur (match_id, xuid) → ART-safe (jamais UPDATE/DELETE
// indexé). Ne touche PAS match_registry (UPDATE player_count = vecteur ART).
func commitMatch(ctx context.Context, shared *sql.DB, res reconstruction) error {
	tx, err := shared.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi
	now := time.Now().UTC()
	for _, d := range res.toInsert {
		if err := insertParticipant(ctx, tx, res.matchID, d, now); err != nil {
			return err
		}
		if err := upsertAlias(ctx, tx, d, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// insertParticipant — INSERT-only avec garde NOT EXISTS sur (match_id, xuid).
// La sous-requête WHERE NOT EXISTS rend l'INSERT idempotent SANS ON CONFLICT
// (donc sans toucher l'index PK en write-path UPDATE — ART-safe). Seules les
// colonnes connues sont renseignées ; tout le reste est laissé à NULL (trous
// assumés ; cf. en-tête de fonction pour la liste exacte).
//
// Colonnes RENSEIGNÉES : match_id, xuid, gamertag, team_id, outcome, kills,
// deaths, created_at=now.
// Colonnes NULL (stats secondaires inconnues hors kill-feed) : rank, score,
// assists, shots_fired, shots_hit, damage_dealt, damage_taken, kda, accuracy,
// personal_score, time_played_seconds, avg_life_seconds, kills_expected,
// deaths_expected, kills_stddev, deaths_stddev, team_mmr, enemy_mmr,
// headshot_kills, max_killing_spree, grenade_kills, melee_kills,
// power_weapon_kills, assassination_kills, ground_pound_kills,
// shoulder_bash_kills, joined_in_progress, left_in_progress, first_joined_time,
// last_leave_time, backfill_bits.
//
// present_at_beginning / present_at_completion : laissés NULL (et NON TRUE). Les
// participants H5 D'ORIGINE ont ces colonnes à NULL. La fonction LUSR
// concurrentTeamSize (internal/sync/skill_v2_shadow.go) : si AU MOINS un membre
// d'une équipe a present_at_beginning non-NULL, elle ne compte QUE les TRUE,
// sinon fallback len(team). Mélanger originaux NULL + reconstruits TRUE casse le
// compte d'effectif (déséquilibre fantôme → LUSR skip). En insérant NULL on reste
// cohérent avec les originaux → la fonction retombe uniformément sur len(team)
// pour les deux équipes → équilibre correct.
//
// backfill_bits : laissé NULL. Le bitmask PBit* (writes.go) marque la PRÉSENCE
// d'une source backfill par colonne ; aucun bit ne signifie « reconstruit
// offline depuis le kill-feed ». Inventer une valeur serait ambigu (les readers
// l'interprètent comme « colonne X fiable »), donc NULL = aucune source =
// correct. La traçabilité de l'origine est portée par xuid_aliases.source.
func insertParticipant(ctx context.Context, tx *sql.Tx, matchID string, d droppedPlayer, now time.Time) error {
	// Liste EXACTE des 41 colonnes de match_participants (cf.
	// persist/shared_persister.go::persistParticipants). Les non renseignées
	// reçoivent NULL explicitement pour rester aligné colonne→valeur.
	const q = `
		INSERT INTO match_participants (
			match_id, xuid, gamertag,
			team_id, outcome, rank, score,
			kills, deaths, assists,
			shots_fired, shots_hit,
			damage_dealt, damage_taken,
			kda, accuracy, personal_score,
			time_played_seconds, avg_life_seconds,
			kills_expected, deaths_expected, kills_stddev, deaths_stddev,
			team_mmr, enemy_mmr,
			headshot_kills,
			max_killing_spree, grenade_kills, melee_kills, power_weapon_kills,
			assassination_kills, ground_pound_kills, shoulder_bash_kills,
			present_at_beginning, present_at_completion, joined_in_progress, left_in_progress,
			first_joined_time, last_leave_time,
			backfill_bits,
			created_at
		)
		SELECT
			?, ?, ?,
			?, ?, NULL, NULL,
			?, ?, NULL,
			NULL, NULL,
			NULL, NULL,
			NULL, NULL, NULL,
			NULL, NULL,
			NULL, NULL, NULL, NULL,
			NULL, NULL,
			NULL,
			NULL, NULL, NULL, NULL,
			NULL, NULL, NULL,
			NULL, NULL, NULL, NULL,
			NULL, NULL,
			NULL,
			?
		WHERE NOT EXISTS (
			SELECT 1 FROM match_participants WHERE match_id = ? AND xuid = ?
		)`
	var gt any
	if d.Gamertag != "" {
		gt = d.Gamertag
	} // sinon NULL
	_, err := tx.ExecContext(ctx, q,
		matchID, d.XUID, gt,
		d.InferredTeam, d.InferredOutcome,
		d.Kills, d.Deaths,
		now,
		matchID, d.XUID,
	)
	if err != nil {
		return fmt.Errorf("INSERT match_participants %s/%s: %w", matchID, d.XUID, err)
	}
	return nil
}

// upsertAlias — INSERT OR IGNORE dans xuid_aliases (xuid PK), source marquée
// 'roster_topup' pour tracer l'origine offline. INSERT OR IGNORE ne touche pas
// l'index en UPDATE (insert pur, ignoré si la PK existe) — ART-safe.
func upsertAlias(ctx context.Context, tx *sql.Tx, d droppedPlayer, now time.Time) error {
	if d.XUID == "" {
		return nil
	}
	var gt any
	if d.Gamertag != "" {
		gt = d.Gamertag
	}
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		VALUES (?, ?, ?, 'roster_topup', ?)`,
		d.XUID, gt, now, now)
	if err != nil {
		return fmt.Errorf("INSERT xuid_aliases %s: %w", d.XUID, err)
	}
	return nil
}

func printDetail(res reconstruction) {
	fmt.Printf("\n--- DÉTAIL %s ---\n", res.matchID)
	fmt.Printf("Équipes présentes : teamA=%d teamB=%d\n", res.teamA, res.teamB)
	fmt.Printf("Connus (%d) :\n", len(res.knowns))
	for _, k := range res.knowns {
		fmt.Printf("  xuid=%s team=%d outcome=%d\n", k.XUID, k.TeamID, k.Outcome)
	}
	fmt.Printf("Droppés détectés (%d) :\n", len(res.dropped))
	for _, d := range res.dropped {
		fmt.Printf("  xuid=%s gt=%q kills=%d deaths=%d\n", d.XUID, d.Gamertag, d.Kills, d.Deaths)
	}
	fmt.Printf("À INSÉRER (%d) :\n", len(res.toInsert))
	for _, d := range res.toInsert {
		fmt.Printf("  xuid=%s gt=%q team_INFÉRÉE=%d outcome=%d kills=%d deaths=%d (autres stats=NULL)\n",
			d.XUID, d.Gamertag, d.InferredTeam, d.InferredOutcome, d.Kills, d.Deaths)
	}
}

func openDB(path string) *sql.DB {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		fatal("open %s: %v", path, err)
	}
	return db
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
