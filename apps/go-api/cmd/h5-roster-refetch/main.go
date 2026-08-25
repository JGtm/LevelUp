// Outil ops : RE-FETCH la carnage des matchs Halo 5 au roster INCOMPLET qui ne sont
// PAS reconstructibles offline (résidus de cmd/h5-roster-topup : BTB 8v8, matchs à
// substitutions → roster reconstruit > 8 / déséquilibré, ou équipe indéterminable),
// puis TOP-UP (INSERT-only) les participants MANQUANTS avec STATS COMPLÈTES depuis la
// carnage. INSERT-only strict, garde NOT EXISTS sur (match_id, xuid) → ART-safe (jamais
// d'UPDATE/DELETE indexé). Transaction par match. Ne touche PAS match_registry.
//
// IDENTITÉ — kill-feed LOCAL d'abord (pas le résolveur Xbox externe). La carnage h5 est
// gamertag-keyée (Player.Xuid toujours null) ; le résolveur PeopleHub/profil Xbox échoue
// sur les vieux gamertags H5. MAIS le kill-feed (killer_victim_pairs) porte gamertag ET
// xuid de TOUS les joueurs ayant tué/été tué dans le match → la map gamertag→xuid est
// déjà LOCALE et complète. Ordre de résolution par joueur de la carnage :
//
//	(1) map kill-feed locale du match (autoritative pour les résidus) ;
//	(2) sinon le résolveur universel worldenrich SI câblé (--resolver) ;
//	(3) sinon SKIP ce joueur (trou assumé — resolve-or-skip de mapCarnageParticipants :
//	    PK (match_id, xuid="") collisionnerait). Le kill-feed garde son identité gamertag.
//
// AUTH : RÉUTILISE EXACTEMENT le câblage de cmd/h5-backfill (token store-first →
// ctxkeys.WithHaloAuth → halo5.NewCaptureSource). LEVELUP_H5_AUTH_AS=JGtm = le SEUL
// token v4 vivant : la carnage h5 (/h5/{mode}/matches/{id}, header Spartan v4, SANS
// clearance ni xuid) sert l'historique de N'IMPORTE quel gamertag avec N'IMPORTE quel
// token v4 valide.
//
// Usage :
//
//	LEVELUP_REPO_ROOT=<repo principal> LEVELUP_H5_AUTH_AS=JGtm go run ./cmd/h5-roster-refetch            # DRY-RUN
//	... --commit          # écrit réellement (INSERT-only, garde NOT EXISTS)
//	... --match <id>      # cible un seul match (test/diagnostic)
//	... --limit N         # ne traite que N matchs candidats (test progressif)
//	... --title halo_5    # défaut halo_5 (outil H5-only par construction)
//	... --resolver        # active le fallback résolveur universel worldenrich (défaut OFF)
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/platform/auth"
)

// h5RefetchMode = segment d'URL de la carnage pour les candidats. Les candidats sont
// non-firefight 2-équipes → Arena (le seul mode H5 ingéré à 2 camps ; Warzone exclu à
// la collecte, cf. capture.go::isExcludedH5GameMode). Le backfill CSR par match
// (livesync/csr_match.go) hardcode aussi "arena" pour re-fetcher un match_id arbitraire.
const h5RefetchMode = "arena"

func main() {
	commit := flag.Bool("commit", false, "écrit réellement (défaut false = DRY-RUN, n'écrit rien)")
	matchID := flag.String("match", "", "cible un seul match (optionnel, détail complet)")
	limit := flag.Int("limit", 0, "ne traite que N matchs candidats (0 = tous ; test progressif)")
	titleSlug := flag.String("title", halo5.TitleSlug, "slug du titre (défaut halo_5 ; outil H5-only par construction)")
	useResolver := flag.Bool("resolver", false, "active le fallback résolveur universel worldenrich (défaut OFF : kill-feed local seul)")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}

	// Auth = câblage EXACT de cmd/h5-backfill. Le compte d'auth (token v4 emprunté)
	// vient de LEVELUP_H5_AUTH_AS ; à défaut le 1er joueur déclaré (mais en pratique
	// JGtm est le seul RT vivant → toujours fourni en env). On ne cible aucun joueur
	// précis : la carnage sert l'historique de n'importe quel match.
	authGT := os.Getenv("LEVELUP_H5_AUTH_AS")
	players, err := cfg.LoadPlayers()
	if err != nil {
		fatal("LoadPlayers: %v", err)
	}
	if authGT == "" {
		if len(players) == 0 {
			fatal("aucun joueur déclaré et LEVELUP_H5_AUTH_AS vide (token d'auth introuvable)")
		}
		authGT = players[0].Gamertag
	}
	var authXUID string
	for i := range players {
		if players[i].Gamertag == authGT {
			authXUID = players[i].XUID
		}
	}
	if authXUID == "" {
		fatal("xuid auth introuvable pour %q dans db_profiles (LEVELUP_H5_AUTH_AS)", authGT)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(*titleSlug)
	fmt.Printf("Titre=%s shared=%s auth_as=%s commit=%v resolver=%v\n",
		*titleSlug, sharedPath, authGT, *commit, *useResolver)
	if !*commit {
		fmt.Println("[DRY-RUN] aucune écriture — affiche ce qui SERAIT inséré.")
	}

	// Token store-first → ctx (l'adapter h5 lit le token du ctx). Identique à h5-backfill.
	store := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), authXUID, authGT)
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens auth_as=%s: err=%v", authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	fmt.Printf("auth_xuid=%s spartan_len=%d\n", authXUID, len(res.Tokens.SpartanToken))

	// Source live (réseau différé au-delà de l'auth ; token déjà dans le ctx). MÊME
	// constructeur que h5-backfill (GetMatchCarnage exposé par *Client).
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	// Fallback résolveur universel (worldenrich) optionnel — OFF par défaut : pour les
	// résidus, le kill-feed local est autoritatif et le résolveur Xbox échoue sur les
	// vieux gamertags H5. On le câble seulement si --resolver (parité avec le backfill).
	var universal func(gamertag string) string
	if *useResolver {
		universal = livesync.BuildBackfillDeps(ctx, cfg, src, authGT, authXUID).ResolveXUID
	}

	shared := openDB(sharedPath)
	defer shared.Close()

	candidates, err := loadResidualCandidates(ctx, shared, *matchID, *limit)
	if err != nil {
		fatal("loadResidualCandidates: %v", err)
	}
	fmt.Printf("Matchs candidats (roster < kill-feed, 2-équipes, non-firefight) : %d\n", len(candidates))
	if len(candidates) == 0 {
		fmt.Println("Rien à faire.")
		return
	}

	runRefetch(ctx, shared, src, universal, candidates, *commit, *matchID != "")
}

// runRefetch traite chaque candidat (best-effort) et imprime le bilan. Un match qui
// échoue au fetch ou au commit ne bloque pas les autres.
func runRefetch(ctx context.Context, shared *sql.DB, src halo5.CaptureSource,
	universal func(string) string, candidates []string, commit, detail bool) {
	var (
		processed   int // candidats effectivement fetchés + mappés (carnage OK)
		apiErrors   int // matchs sautés sur erreur API/decode
		insertedRow int // participants réellement insérés
		unresolved  int // joueurs de la carnage non résolus (trous assumés)
		matchesWith int // matchs ayant inséré ≥ 1 participant
	)
	for i, mid := range candidates {
		if i > 0 && i%25 == 0 {
			fmt.Printf("  ... progression : %d/%d candidats traités (insérés=%d, erreurs_api=%d)\n",
				i, len(candidates), insertedRow, apiErrors)
		}

		// (a) map identité LOCALE depuis le kill-feed du match (gamertag→xuid).
		idMap, err := loadKillFeedIdentity(ctx, shared, mid)
		if err != nil {
			fmt.Printf("  %s : ERREUR kill-feed %v (sauté)\n", mid, err)
			apiErrors++ // comptabilisé comme skip best-effort
			continue
		}
		resolve := makeResolver(idMap, universal)

		// (b) fetch carnage (best-effort : erreur API → skip ce match, continue).
		carnage, cerr := src.GetMatchCarnage(ctx, mid, h5RefetchMode)
		if cerr != nil {
			fmt.Printf("  %s : carnage indisponible %v (sauté)\n", mid, cerr)
			apiErrors++
			continue
		}
		processed++

		// (c) map carnage → participants COMPLETS (réutilise le mapping live h5, dropped
		// = joueurs non résolus). present_at_* restent NULL (cf. mapCarnageParticipants).
		var dropped int
		rows := halo5.MapCarnageParticipants(ctx, mid, carnage, resolve, &dropped)
		unresolved += dropped

		// (d) INSÉRER uniquement les (match_id, xuid) ABSENTS, INSERT-only NOT EXISTS.
		ins, mErr := topUpMatch(ctx, shared, mid, rows, commit, detail)
		if mErr != nil {
			fmt.Printf("  %s : COMMIT échoué %v (sauté)\n", mid, mErr)
			apiErrors++
			continue
		}
		insertedRow += ins
		if ins > 0 {
			matchesWith++
		}
		if detail {
			fmt.Printf("  %s : carnage=%d résolus mappés / %d non résolus → %d à insérer (absents)\n",
				mid, len(rows), dropped, ins)
		}
	}

	fmt.Printf("\n=== BILAN ===\n")
	fmt.Printf("Candidats         : %d\n", len(candidates))
	fmt.Printf("Carnage fetchée   : %d\n", processed)
	fmt.Printf("Erreurs API/skip  : %d\n", apiErrors)
	fmt.Printf("Joueurs non résolus (trous) : %d\n", unresolved)
	if commit {
		fmt.Printf("INSÉRÉS           : %d participant(s) sur %d match(s) (INSERT-only, garde NOT EXISTS)\n",
			insertedRow, matchesWith)
	} else {
		fmt.Printf("[DRY-RUN] %d participant(s) SERAIENT insérés sur %d match(s). Relancer avec --commit.\n",
			insertedRow, matchesWith)
	}
}

// loadResidualCandidates sélectionne les matchs RÉSIDUS (roster incomplet vs kill-feed),
// MÊME prédicat que cmd/h5-roster-topup::loadCandidateMatches : 2-équipes, non-firefight,
// nb de joueurs distincts au kill-feed STRICTEMENT > COUNT(match_participants). L'offline
// (topup) a déjà complété les Arena propres (roster ≤ 8) ; ce qui RESTE candidat ici sont
// les BTB 8v8 et matchs à substitutions (total > 8) qu'il a laissés en résidu. limit > 0
// borne le nombre traité (test progressif) ; only != "" cible un match unique.
func loadResidualCandidates(ctx context.Context, shared *sql.DB, only string, limit int) ([]string, error) {
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
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, rows.Err()
}

// loadKillFeedIdentity construit la map gamertag→xuid LOCALE d'un match depuis
// killer_victim_pairs (killer_gamertag↔killer_xuid ET victim_gamertag↔victim_xuid).
// Complète pour tous les joueurs ayant tué/été tué — c'est l'identité autoritative des
// résidus (le résolveur Xbox externe échoue sur les vieux gamertags H5).
func loadKillFeedIdentity(ctx context.Context, shared *sql.DB, matchID string) (map[string]string, error) {
	rows, err := shared.QueryContext(ctx, `
		SELECT gamertag, xuid FROM (
			SELECT killer_gamertag AS gamertag, killer_xuid AS xuid FROM killer_victim_pairs WHERE match_id = ?
			UNION ALL
			SELECT victim_gamertag AS gamertag, victim_xuid AS xuid FROM killer_victim_pairs WHERE match_id = ?
		)
		WHERE gamertag IS NOT NULL AND gamertag <> '' AND xuid IS NOT NULL AND xuid <> ''`,
		matchID, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var gt, xuid string
		if err := rows.Scan(&gt, &xuid); err != nil {
			return nil, err
		}
		out[gt] = xuid // dernier vu gagne (xuid stable par gamertag dans un match)
	}
	return out, rows.Err()
}

// topUpMatch insère les participants ABSENTS d'un match en UNE transaction (INSERT-only,
// garde NOT EXISTS sur (match_id, xuid)). Retourne le nombre de participants insérés. En
// DRY-RUN : ne touche pas la DB mais compte ce qui SERAIT inséré (via le set déjà présent).
func topUpMatch(ctx context.Context, shared *sql.DB, matchID string, rows []domain.MatchParticipantRow,
	commit, detail bool) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	present, err := loadPresentXUIDs(ctx, shared, matchID)
	if err != nil {
		return 0, err
	}

	// DRY-RUN : on ne fait que compter les rows dont le xuid est absent.
	if !commit {
		n := 0
		for i := range rows {
			if _, ok := present[rows[i].XUID]; !ok {
				n++
			}
		}
		return n, nil
	}

	tx, err := shared.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi
	now := time.Now().UTC()
	inserted := 0
	for i := range rows {
		r := rows[i]
		if _, ok := present[r.XUID]; ok {
			continue // déjà présent → ne JAMAIS toucher (INSERT-only strict)
		}
		n, err := insertParticipant(ctx, tx, r, now)
		if err != nil {
			return 0, err
		}
		inserted += n
		if err := insertAlias(ctx, tx, r, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

// loadPresentXUIDs charge le set des xuid DÉJÀ persistés pour un match (garde anti-doublon
// + filet DRY-RUN). La garde NOT EXISTS dans l'INSERT reste l'autorité ART-safe ; ce set
// évite juste un INSERT no-op par row déjà présente.
func loadPresentXUIDs(ctx context.Context, shared *sql.DB, matchID string) (map[string]struct{}, error) {
	rows, err := shared.QueryContext(ctx,
		`SELECT xuid FROM match_participants WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		out[x] = struct{}{}
	}
	return out, rows.Err()
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
