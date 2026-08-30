//go:build cgo

// diag_film_avail — MESURE JETABLE : le film d un match est-il ENCORE servi par 343 ?
//
// Question a laquelle il repond : les matchs sans passe de film depuis avril 2026
// peuvent-ils encore etre rattrapes, ou leur film a-t-il expire cote serveur ?
// Il ne telecharge AUCUN chunk : seul le manifeste est demande (GetFilmChunkURLs).
//
// Usage (depuis apps/go-api/) :
//
//	go run -tags cgo ./cmd/diag_film_avail -per-month 5
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// voiesDeFilm : la liste SQL des voies de FILM, construite depuis le vocabulaire de portee
// (`killscope`) et jamais recopiee — une copie qui derive d un caractere rendrait la mesure
// aveugle sans erreur ni compteur (ratchet archlint.TestNoRawKillScopeLiteral).
func voiesDeFilm() string {
	voies := killscope.FilmReadPaths()
	for i, v := range voies {
		voies[i] = "'" + v + "'"
	}
	return strings.Join(voies, ", ")
}

// startTimeCanon : le fragment canonique (regle 8). Jamais `start_time` brut.
var startTimeCanon = analysis.SQLStartTimeCanonical("r")

type candidat struct {
	mois    string
	matchID string
}

func main() {
	dbPath := flag.String("db", "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb", "shared DB")
	envFile := flag.String("env-file", "../../.env.local", "chemin .env.local")
	authFile := flag.String("auth-file", "../../data/auth/watcher_tokens.json", "watcher_tokens.json")
	gamertag := flag.String("gamertag", "Chocoboflor", "gamertag porteur des tokens")
	perMonth := flag.Int("per-month", 5, "echantillon par mois")
	since := flag.String("since", "2026-01", "premier mois inclus")
	flag.Parse()

	loadEnvLocal(*envFile)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	ctx := context.Background()

	cands, err := lireCandidats(ctx, *dbPath, *since, *perMonth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "candidats: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%d candidats\n", len(cands))

	tokens, err := loadTokens(ctx, *authFile, *gamertag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tokens: %v\n", err)
		os.Exit(1)
	}
	client := gosync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 3)

	type stat struct{ trouve, absent, erreur int }
	par := map[string]*stat{}
	fmt.Println("mois\tmatch_id\tetat\tchunks")
	for _, c := range cands {
		s := par[c.mois]
		if s == nil {
			s = &stat{}
			par[c.mois] = s
		}
		refs, found, err := client.GetFilmChunkURLs(ctx, c.matchID)
		switch {
		case err != nil:
			s.erreur++
			fmt.Printf("%s\t%s\tERREUR\t%v\n", c.mois, c.matchID, err)
		case !found:
			s.absent++
			fmt.Printf("%s\t%s\tEXPIRE\t0\n", c.mois, c.matchID)
		default:
			s.trouve++
			fmt.Printf("%s\t%s\tDISPO\t%d\n", c.mois, c.matchID, len(refs))
		}
	}

	fmt.Println("\n=== SYNTHESE ===")
	fmt.Println("mois\tdispo\texpire\terreur")
	mois := make([]string, 0, len(par))
	for m := range par {
		mois = append(mois, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(mois)))
	for _, m := range mois {
		s := par[m]
		fmt.Printf("%s\t%d\t%d\t%d\n", m, s.trouve, s.absent, s.erreur)
	}
}

// lireCandidats rend, par mois, les matchs SANS aucune ligne de voie film dans la passe
// courante — donc ceux dont l assistant reste inconnu.
func lireCandidats(ctx context.Context, dbPath, since string, perMonth int) ([]candidat, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, `
		-- ⚠ LA SELECTION PART DU REGISTRE, PAS DES KILL-EVENTS. Une premiere version partait
		-- de match_kill_events_latest : un match sans AUCUN kill-event lui etait invisible,
		-- et le perimetre mesure s en trouvait sous-estime (999 candidats reels contre 374
		-- annonces le 2026-08-29). Le tri passe par le COALESCE canonique (regle 8).
		WITH c AS (
			SELECT strftime(`+startTimeCanon+`,'%Y-%m') AS mois, r.match_id,
			       ROW_NUMBER() OVER (PARTITION BY strftime(`+startTimeCanon+`,'%Y-%m')
			                          ORDER BY `+startTimeCanon+` DESC) AS rn
			FROM match_registry r
			WHERE NOT EXISTS (
				SELECT 1 FROM match_kill_events_latest e
				WHERE e.match_id = r.match_id AND e.read_path IN (`+voiesDeFilm()+`)
			) AND strftime(`+startTimeCanon+`,'%Y-%m') >= ?
		)
		SELECT mois, match_id FROM c WHERE rn <= ? ORDER BY mois DESC, match_id`, since, perMonth)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []candidat
	for rows.Next() {
		var c candidat
		if err := rows.Scan(&c.mois, &c.matchID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key, val := strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// loadTokens : XSTS du store mono-user d abord, puis le MultiUserTokenStore (source unique
// ADR 0023).
//
// ⚠ CHAQUE ECHEC EST JOURNALISE. Une premiere version jetait les quatre erreurs et sortait
// « tokens introuvables » : la cause reelle (`AADSTS70000`, rotation perdue, mauvais xuid)
// disparaissait, alors que la doctrine du depot impose de DIAGNOSTIQUER avant de conclure — et
// que re-capturer un token « pour reparer » est justement l erreur que cette doctrine interdit.
func loadTokens(ctx context.Context, authFile, gamertag string) (*struct {
	SpartanToken   string
	ClearanceToken string
}, error) {
	store := auth.NewTokenStore(authFile)
	stored, err := store.Load()
	if err != nil {
		slog.WarnContext(ctx, "diag_film_avail: store mono-user illisible", "fichier", authFile, "err", err)
	}
	if stored != nil && stored.IsXSTSValid(0) {
		res, xerr := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken)
		if xerr == nil {
			return &struct {
				SpartanToken   string
				ClearanceToken string
			}{res.SpartanToken, res.ClearanceToken}, nil
		}
		slog.WarnContext(ctx, "diag_film_avail: echange XSTS refuse — repli sur le refresh token",
			"err", xerr)
	}

	tokenStore := auth.NewMultiUserTokenStore(strings.TrimSuffix(authFile, ".json"))
	user, lerr := tokenStore.LoadByGamertag(gamertag)
	if lerr != nil || user == nil {
		return nil, fmt.Errorf("aucun joueur %q dans le magasin de tokens: %w", gamertag, lerr)
	}
	res, rerr := auth.RefreshHaloTokensViaStoreFirst(ctx, tokenStore, auth.NewSISUProvider(), user.XUID, gamertag)
	if rerr != nil {
		return nil, fmt.Errorf("refresh du token de %s (xuid %s): %w", gamertag, user.XUID, rerr)
	}
	tokens := auth.HaloTokensFromExchange(res)
	if tokens == nil {
		return nil, fmt.Errorf("echange sans tokens Halo pour %s", gamertag)
	}
	return &struct {
		SpartanToken   string
		ClearanceToken string
	}{tokens.SpartanToken, tokens.ClearanceToken}, nil
}
