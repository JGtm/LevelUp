//go:build cgo

// cmd/probe-world-stats — Phase A probe (diagnostic, AUCUN INSERT) pour le
// leaderboard mondial enrichi. Valide le process complet sur un échantillon
// avant d'écrire le moindre code de production (cf. .ai/PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Ce que ce probe valide :
//  1. Résolution xuid des gamertags du top-100 mondial via PeopleHub
//     (réutilise le header XBL3.0 RTA d'auth.RefreshUserXSTS — audience
//     http://xboxlive.com, exactement le relying party de PeopleHub).
//  2. Le pipeline GetMatchHistory -> GetMatchStats (les N derniers matchs).
//  3. La STRUCTURE brute de GetMatchStats (--dump-raw) -> confirme les chemins
//     d'extraction (Players[], PlayerId, CoreStats, Outcome, ParticipationInfo)
//     pour coder l'agrégation en Phase B. v1 ne fait PAS l'extraction de stats.
//  4. Le timing réel par match -> calibrage du backfill.
//
// Lecture seule sur la shared DB. Utilise les tokens d'UN joueur (--token-xuid).
// Si la shared DB est verrouillée (serveur API en cours), stopper le serveur ou
// pointer --shared-db sur une copie.
//
// Usage (depuis apps/go-api/, le worktree partage le data/ du repo principal) :
//
//	go run ./cmd/probe-world-stats \
//	  --shared-db   ../../LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
//	  --tokens-dir  ../../LevelUp-go-migration/data/auth/watcher_tokens \
//	  --env-file    ../../LevelUp-go-migration/.env.local \
//	  --token-xuid 2533274823110022 --token-gamertag JGtm \
//	  --playlists-per-season 2 --matches 10 --max-candidates 5 --dump-raw
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	sharedDB := flag.String("shared-db", "", "chemin shared_matches_v2.duckdb (RO ; world_csr_leaderboard_latest) — requis")
	tokensDir := flag.String("tokens-dir", "data/auth/watcher_tokens", "répertoire MultiUserTokenStore")
	envFile := flag.String("env-file", ".env.local", "chemin .env.local (SPNKR_AZURE_CLIENT_ID requis par MSAL)")
	tokenXUID := flag.String("token-xuid", "", "XUID du joueur dont les tokens servent aux appels API — requis")
	tokenGamertag := flag.String("token-gamertag", "", "gamertag du porteur de tokens (logs)")
	playlistsPerSeason := flag.Int("playlists-per-season", 2, "max playlists testées par saison")
	matches := flag.Int("matches", 10, "matchs à fetcher par joueur résolu")
	maxCandidates := flag.Int("max-candidates", 5, "max joueurs essayés par (saison, playlist) avant SKIP")
	maxSeasons := flag.Int("max-seasons", 2, "max saisons testées (récentes d'abord)")
	rateLimit := flag.Int("rate-limit", 60, "requêtes max/minute vers l'API match")
	dumpRaw := flag.Bool("dump-raw", false, "dump JSON brut du 1er GetMatchStats (révèle la structure)")
	metadataDB := flag.String("metadata-db", "", "chemin metadata.duckdb (csr_season_calendars — requis pour --depth-pages)")
	depthPages := flag.Int("depth-pages", 0, "mode depth-scan saison : pagine N pages GetMatchHistory (sans stats) pour le top joueur, bucket par saison")
	flag.Parse()

	loadEnvLocal(*envFile)
	ctx := context.Background()

	if strings.TrimSpace(*sharedDB) == "" || strings.TrimSpace(*tokenXUID) == "" {
		fatal("--shared-db et --token-xuid sont requis")
	}

	// 1. UN SEUL access_token Microsoft (1 refresh RT via MSAL) → on en dérive
	// BOTH le header PeopleHub (XSTS RTA, audience http://xboxlive.com) ET les
	// tokens Halo. Évite la DOUBLE rotation de RT (à usage unique) qui churnait
	// les tokens du porteur — incident probe 2026-06-10.
	store := auth.NewMultiUserTokenStore(*tokensDir)
	bearer, err := store.Load(*tokenXUID)
	if err != nil {
		fatal("chargement tokens %s : %v", *tokenXUID, err)
	}
	// access_token FRAIS via le RT brut (post-retrait MSAL 2026-07-15 : la voie
	// cache MSAL a disparu, ExchangeRefreshTokenWithRotation couvre RTs Azure et
	// MSA natifs). UN SEUL refresh (pas de double rotation de RT), puis
	// persistance de l'état tourné.
	var accessToken string
	if bearer.OAuthRefreshToken != "" {
		at, rotatedRT, _, rerr := auth.ExchangeRefreshTokenWithRotation(ctx, bearer.OAuthRefreshToken)
		if rerr != nil {
			fatal("refresh RT brut (%s) : %v", *tokenXUID, rerr)
		}
		accessToken = at
		if rotatedRT != "" && rotatedRT != bearer.OAuthRefreshToken {
			bearer.OAuthRefreshToken = rotatedRT
		}
	}
	if strings.TrimSpace(accessToken) == "" {
		fatal("aucun access_token frais pour %s (pas de refresh_token exploitable)\n→ re-login SSO Xbox ou `go run ./cmd/token-capture/ %s` (ADR 0023).", *tokenXUID, *tokenGamertag)
	}
	bearer.AccessToken = accessToken
	bearer.OAuthExpiresAt = time.Now().Add(50 * time.Minute)
	_ = store.Upsert(bearer)
	rta, err := auth.AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		fatal("AcquireXSTSForRTA : %v", err)
	}
	xblHeader := fmt.Sprintf("XBL3.0 x=%s;%s", rta.UserHash, rta.Token)
	res, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		fatal("ExchangeAccessToken : %v", err)
	}
	if res.Tokens == nil || strings.TrimSpace(res.Tokens.SpartanToken) == "" {
		fatal("aucun Spartan token obtenu pour xuid %s", *tokenXUID)
	}
	fmt.Printf("[auth] 1 access_token → XSTS RTA + tokens Halo OK pour %s (xuid %s)\n", *tokenGamertag, *tokenXUID)
	client := syncpkg.NewHaloAPIClient(res.Tokens.SpartanToken, res.Tokens.ClearanceToken, ratePerSecond(*rateLimit))
	httpClient := &http.Client{Timeout: 20 * time.Second}

	// 3. Matrice (saison, playlist) -> gamertags depuis le snapshot scrapé.
	matrix, err := loadMatrix(ctx, *sharedDB, *maxSeasons)
	if err != nil {
		fatal("lecture world_csr_leaderboard_latest : %v\n→ shared DB verrouillée ? stopper le serveur ou copier le fichier.", err)
	}
	if len(matrix) == 0 {
		fmt.Println("Aucune ligne dans world_csr_leaderboard_latest — rien à sonder.")
		return
	}

	// 3bis. Mode depth-scan saison : valide la pagination jusqu'aux saisons
	// passées + l'attribution par fenêtre de dates (csr_season_calendars).
	if *depthPages > 0 {
		if strings.TrimSpace(*metadataDB) == "" {
			fatal("--metadata-db requis pour --depth-pages")
		}
		windows, werr := loadSeasonWindows(ctx, *metadataDB)
		if werr != nil {
			fatal("lecture csr_season_calendars : %v", werr)
		}
		fmt.Printf("[calendrier] %d saisons chargées\n", len(windows))
		gt := matrix[0].gamertags[0]
		xuid := peopleHubResolve(ctx, httpClient, xblHeader, gt)
		if xuid == "" {
			fatal("résolution xuid échouée pour %s (depth-scan)", gt)
		}
		fmt.Printf("[depth-scan] joueur %s -> %s (%d pages max)\n", gt, xuid, *depthPages)
		seasonDepthScan(ctx, client, windows, xuid, *depthPages)
		return
	}

	// 4. Sonde.
	rep := &report{seenPlaylistIDs: map[string]int{}, byPlaylist: map[string]*bucket{}}
	runStart := time.Now()
	dumped := false
	perSeason := map[string]int{}
	for _, sp := range matrix {
		if perSeason[sp.season] >= *playlistsPerSeason {
			continue // assez de playlists testées pour cette saison
		}
		perSeason[sp.season]++
		probeSeasonPlaylist(ctx, client, httpClient, xblHeader, sp, *maxCandidates, *matches, *dumpRaw, &dumped, rep)
	}
	rep.totalDuration = time.Since(runStart)
	rep.print()
}

// seasonPlaylist regroupe les gamertags (ordre rank ASC) d'une (saison, playlist).
type seasonPlaylist struct {
	season    string
	playlist  string
	gamertags []string
}

// loadMatrix lit world_csr_leaderboard_latest et groupe par (saison, playlist),
// gamertags triés rank ASC, saisons récentes d'abord (limité à maxSeasons).
func loadMatrix(ctx context.Context, dbPath string, maxSeasons int) ([]seasonPlaylist, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open RO: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.QueryContext(ctx, `
		SELECT season_id, playlist_id, gamertag
		FROM world_csr_leaderboard_latest
		WHERE season_id <> '' AND playlist_id <> '' AND gamertag <> ''
		ORDER BY season_id DESC, playlist_id, rank ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []seasonPlaylist
	idx := map[string]int{} // "season|playlist" -> position dans out
	seasonsSeen := map[string]bool{}
	for rows.Next() {
		var season, playlist, gamertag string
		if err := rows.Scan(&season, &playlist, &gamertag); err != nil {
			return nil, err
		}
		if !seasonsSeen[season] {
			if len(seasonsSeen) >= maxSeasons {
				continue // saison trop ancienne (au-delà de maxSeasons)
			}
			seasonsSeen[season] = true
		}
		key := season + "|" + playlist
		if p, ok := idx[key]; ok {
			out[p].gamertags = append(out[p].gamertags, gamertag)
			continue
		}
		idx[key] = len(out)
		out = append(out, seasonPlaylist{season: season, playlist: playlist, gamertags: []string{gamertag}})
	}
	return out, rows.Err()
}

// probeSeasonPlaylist résout les xuids (PeopleHub) puis fetch les matchs d'un
// candidat qui a des données, pour une (saison, playlist).
func probeSeasonPlaylist(
	ctx context.Context, client *syncpkg.HaloAPIClient, httpClient *http.Client,
	xblHeader string, sp seasonPlaylist, maxCandidates, matches int,
	dumpRaw bool, dumped *bool, rep *report,
) {
	rep.playlistsProbed++
	fmt.Printf("\n=== %s / %s (%d gamertags) ===\n", sp.season, sp.playlist, len(sp.gamertags))

	tried := 0
	for _, gt := range sp.gamertags {
		if tried >= maxCandidates {
			break
		}
		tried++
		xuid := peopleHubResolve(ctx, httpClient, xblHeader, gt)
		if xuid == "" {
			rep.xuidMiss++
			fmt.Printf("  [xuid_miss] %s\n", gt)
			continue
		}
		rep.xuidHit++
		fmt.Printf("  [xuid_ok]   %s -> %s\n", gt, xuid)

		entries, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", xuid), "matchmaking", 0, clampCount(matches))
		if err != nil {
			fmt.Printf("    GetMatchHistory erreur : %v\n", err)
			continue
		}
		fmt.Printf("    %d matchs renvoyés\n", len(entries))
		for _, e := range entries {
			t0 := time.Now()
			stats, serr := client.GetMatchStats(ctx, e.MatchID)
			dt := time.Since(t0)
			rep.matchStatsCalls++
			rep.matchStatsDur += dt
			if serr != nil {
				fmt.Printf("    GetMatchStats(%s) erreur : %v\n", e.MatchID, serr)
				continue
			}
			if dumpRaw && !*dumped {
				*dumped = true
				dumpStats(stats)
			}
			pid := playlistAssetID(stats)
			if pid != "" {
				rep.seenPlaylistIDs[pid]++
			}
			pl := findPlayerByXUID(stats, xuid)
			if pl == nil {
				fmt.Printf("    match %s  %-16s  [joueur %s absent des Players[]]\n", short(e.MatchID), plName(pid), xuid)
				continue
			}
			core := coreStats(pl)
			k, d, a := intOf(core, "Kills"), intOf(core, "Deaths"), intOf(core, "Assists")
			b := rep.bucketFor(pid)
			b.matches++
			switch outcomeOf(pl) {
			case 2:
				b.wins++
			case 3:
				b.losses++
			case 1:
				b.ties++
			case 4:
				b.dnf++
			}
			b.kills += k
			b.deaths += d
			b.assists += a
			b.score += intOf(core, "PersonalScore")
			b.playtimeSec += iso8601Seconds(timePlayedOf(pl))
			fmt.Printf("    match %s  %-16s  out=%d  %d/%d/%d  (%s)\n",
				short(e.MatchID), plName(pid), outcomeOf(pl), k, d, a, dt.Round(time.Millisecond))
		}
		rep.candidatesWithData++
		return // un candidat avec données suffit pour cette (saison, playlist)
	}
	rep.coverageGaps = append(rep.coverageGaps, sp.season+"/"+sp.playlist)
	fmt.Printf("  [coverage_gap] aucun candidat résolu/avec données (sur %d essais)\n", tried)
}

// peopleHubResolve résout gamertag -> xuid via PeopleHub (recherche fuzzy,
// filtrée sur correspondance exacte case-insensitive). "" si non résolu.
func peopleHubResolve(ctx context.Context, hc *http.Client, xblHeader, gamertag string) string {
	endpoint := "https://peoplehub.xboxlive.com/users/me/people/search/decoration/detail,preferredColor?q=" +
		strings.ReplaceAll(gamertag, " ", "%20") + "&maxItems=25"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("x-xbl-contract-version", "3")
	req.Header.Set("Authorization", xblHeader)
	req.Header.Set("Accept-Language", "en-us")
	resp, err := hc.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		fmt.Printf("    [peoplehub HTTP %d] %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return ""
	}
	var data struct {
		People []struct {
			Gamertag string `json:"gamertag"`
			XUID     string `json:"xuid"`
		} `json:"people"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	for _, p := range data.People {
		if strings.EqualFold(strings.TrimSpace(p.Gamertag), strings.TrimSpace(gamertag)) {
			return p.XUID
		}
	}
	return ""
}

// playlistAssetID extrait MatchInfo.Playlist.AssetId (chemin connu via transforms.go).
func playlistAssetID(stats map[string]any) string {
	mi, ok := stats["MatchInfo"].(map[string]any)
	if !ok {
		return ""
	}
	pl, ok := mi["Playlist"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := pl["AssetId"].(string)
	return id
}

// dumpStats imprime le JSON brut indenté (révèle Players[]/CoreStats/Outcome/ParticipationInfo).
func dumpStats(stats map[string]any) {
	b, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		fmt.Printf("    [dump-raw] marshal erreur : %v\n", err)
		return
	}
	fmt.Printf("\n----- DUMP RAW GetMatchStats (1er match) -----\n%s\n----- FIN DUMP -----\n\n", string(b))
}

// bucket agrège les stats d'un joueur sur une playlist (clé = Playlist.AssetId).
type bucket struct {
	matches, wins, losses, ties, dnf int
	kills, deaths, assists, score    int64
	playtimeSec                      float64
}

// report agrège les résultats de la sonde.
type report struct {
	playlistsProbed    int
	candidatesWithData int
	xuidHit            int
	xuidMiss           int
	matchStatsCalls    int
	matchStatsDur      time.Duration
	seenPlaylistIDs    map[string]int
	byPlaylist         map[string]*bucket
	coverageGaps       []string
	totalDuration      time.Duration
}

// bucketFor retourne (en le créant au besoin) le bucket d'une playlist.
func (r *report) bucketFor(playlistAssetID string) *bucket {
	b, ok := r.byPlaylist[playlistAssetID]
	if !ok {
		b = &bucket{}
		r.byPlaylist[playlistAssetID] = b
	}
	return b
}

func (r *report) print() {
	fmt.Printf("\n========== RAPPORT PROBE ==========\n")
	fmt.Printf("(saison,playlist) sondées : %d\n", r.playlistsProbed)
	fmt.Printf("  avec données           : %d\n", r.candidatesWithData)
	fmt.Printf("  coverage gaps          : %d %v\n", len(r.coverageGaps), r.coverageGaps)
	total := r.xuidHit + r.xuidMiss
	rate := 0.0
	if total > 0 {
		rate = 100 * float64(r.xuidHit) / float64(total)
	}
	fmt.Printf("résolution xuid PeopleHub : %d/%d (%.0f%%)\n", r.xuidHit, total, rate)
	fmt.Printf("GetMatchStats             : %d appels\n", r.matchStatsCalls)
	if r.matchStatsCalls > 0 {
		fmt.Printf("  durée moyenne/match     : %s\n", (r.matchStatsDur / time.Duration(r.matchStatsCalls)).Round(time.Millisecond))
	}
	fmt.Printf("agrégats par playlist (bucketing + extraction validés) :\n")
	for id, b := range r.byPlaylist {
		kda := 0.0
		if b.deaths > 0 {
			kda = (float64(b.kills) + float64(b.assists)/3) / float64(b.deaths)
		}
		wr := 0.0
		if b.matches > 0 {
			wr = 100 * float64(b.wins) / float64(b.matches)
		}
		fmt.Printf("    %-18s : %d matchs  %dW-%dL-%dT  KDA=%.2f  V%%=%.0f  %dk/%dd/%da  %.0fs\n",
			plName(id), b.matches, b.wins, b.losses, b.ties, kda, wr, b.kills, b.deaths, b.assists, b.playtimeSec)
	}
	fmt.Printf("durée totale              : %s\n", r.totalDuration.Round(time.Millisecond))
	fmt.Printf("===================================\n")
}

// ratePerSecond convertit un débit/minute en requêtes/seconde (min 1).
func ratePerSecond(perMinute int) int {
	rps := perMinute / 60
	if rps < 1 {
		rps = 1
	}
	return rps
}

func clampCount(n int) int {
	if n < 1 {
		return 1
	}
	if n > 25 {
		return 25
	}
	return n
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// loadEnvLocal injecte les variables de .env.local (sans écraser l'existant).
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
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

// ─── Extraction stats joueur (chemins confirmés Phase A via --dump-raw) ───

// findPlayerByXUID retourne l'entrée Players[] dont PlayerId == "xuid(N)".
func findPlayerByXUID(stats map[string]any, xuid string) map[string]any {
	players, _ := stats["Players"].([]any)
	want := "xuid(" + xuid + ")"
	for _, p := range players {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := pm["PlayerId"].(string); id == want {
			return pm
		}
	}
	return nil
}

// outcomeOf lit Outcome (numérique : 2=win, 3=loss, 1=tie, 4=dnf).
func outcomeOf(pl map[string]any) int {
	v, _ := pl["Outcome"].(float64)
	return int(v)
}

// coreStats retourne PlayerTeamStats[0].Stats.CoreStats (fallback [0].CoreStats).
func coreStats(pl map[string]any) map[string]any {
	pts, _ := pl["PlayerTeamStats"].([]any)
	if len(pts) == 0 {
		return nil
	}
	first, _ := pts[0].(map[string]any)
	if first == nil {
		return nil
	}
	if st, ok := first["Stats"].(map[string]any); ok {
		if core, ok := st["CoreStats"].(map[string]any); ok {
			return core
		}
	}
	core, _ := first["CoreStats"].(map[string]any)
	return core
}

// intOf lit une clé numérique d'une map JSON (0 si absente).
func intOf(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	v, _ := m[key].(float64)
	return int64(v)
}

// timePlayedOf lit ParticipationInfo.TimePlayed (durée ISO-8601).
func timePlayedOf(pl map[string]any) string {
	pi, _ := pl["ParticipationInfo"].(map[string]any)
	if pi == nil {
		return ""
	}
	tp, _ := pi["TimePlayed"].(string)
	return tp
}

// iso8601Seconds parse une durée ISO-8601 ("PT10M39.203S") en secondes.
func iso8601Seconds(s string) float64 {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "PT")
	var total float64
	num := ""
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			num += string(r)
			continue
		}
		v, _ := strconv.ParseFloat(num, 64)
		num = ""
		switch r {
		case 'H':
			total += v * 3600
		case 'M':
			total += v * 60
		case 'S':
			total += v
		}
	}
	return total
}

// plName mappe un Playlist.AssetId vers son nom catalogue (sinon id court).
func plName(assetID string) string {
	if pl, ok := rankedplaylists.Lookup(assetID); ok {
		return pl.NameEN
	}
	if assetID == "" {
		return "(inconnu)"
	}
	return short(assetID)
}

// ─── Depth-scan saison (validation dimension saison via dates calendrier) ───

// seasonWindow : fenêtre temporelle d'une saison (csr_season_calendars).
type seasonWindow struct {
	id    string
	start time.Time
	end   time.Time
}

// loadSeasonWindows lit csr_season_calendars (metadata.duckdb RO). end_date NULL
// (saison courante) → now().
func loadSeasonWindows(ctx context.Context, dbPath string) ([]seasonWindow, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open RO: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	rows, err := db.QueryContext(ctx, `
		SELECT season_id, start_date, COALESCE(end_date, now())
		FROM csr_season_calendars
		WHERE title_id = 'halo_infinite' AND season_id IS NOT NULL AND season_id <> ''
		ORDER BY start_date`)
	if err != nil {
		return nil, fmt.Errorf("query csr_season_calendars: %w", err)
	}
	defer rows.Close()
	var out []seasonWindow
	for rows.Next() {
		var w seasonWindow
		if err := rows.Scan(&w.id, &w.start, &w.end); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// seasonOf retourne le season_id dont la fenêtre [start, end) contient t.
func seasonOf(windows []seasonWindow, t time.Time) string {
	for _, w := range windows {
		if !t.Before(w.start) && t.Before(w.end) {
			return w.id
		}
	}
	return "(hors calendrier)"
}

// seasonDepthScan pagine GetMatchHistory (SANS GetMatchStats — cheap) sur N pages
// et bucket chaque match dans sa fenêtre de saison. Valide : (1) profondeur de
// pagination (atteint-on les saisons passées ?), (2) fenêtrage par dates réelles.
func seasonDepthScan(ctx context.Context, client *syncpkg.HaloAPIClient, windows []seasonWindow, xuid string, pages int) {
	perSeason := map[string]int{}
	var oldest, newest time.Time
	var oldestID string
	total := 0
	for p := 0; p < pages; p++ {
		entries, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", xuid), "matchmaking", p*25, 25)
		if err != nil {
			fmt.Printf("  GetMatchHistory(page %d) erreur : %v\n", p, err)
			break
		}
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			t, perr := time.Parse(time.RFC3339, e.StartTime)
			if perr != nil {
				continue
			}
			total++
			if oldest.IsZero() || t.Before(oldest) {
				oldest = t
				oldestID = e.MatchID
			}
			if t.After(newest) {
				newest = t
			}
			perSeason[seasonOf(windows, t)]++
		}
		if len(entries) < 25 {
			break // fin de l'historique disponible
		}
	}
	fmt.Printf("\n========== DEPTH-SCAN SAISON ==========\n")
	fmt.Printf("matchs paginés : %d\n", total)
	if !oldest.IsZero() {
		fmt.Printf("fenêtre temporelle : %s → %s\n", oldest.Format("2006-01-02"), newest.Format("2006-01-02"))
	}
	fmt.Printf("répartition par saison (fenêtrage dates calendrier) :\n")
	reached := 0
	for _, w := range windows {
		if n := perSeason[w.id]; n > 0 {
			reached++
			fmt.Printf("    %-16s [%s → %s] : %d matchs\n",
				w.id, w.start.Format("2006-01-02"), w.end.Format("2006-01-02"), n)
		}
	}
	if n := perSeason["(hors calendrier)"]; n > 0 {
		fmt.Printf("    (hors calendrier) : %d\n", n)
	}
	verdict := "⚠️ une seule saison atteinte — augmenter --depth-pages pour confirmer"
	if reached >= 2 {
		verdict = "✅ pagination multi-saison OK — saisons passées atteignables"
	}
	fmt.Printf("→ saisons distinctes atteintes : %d  %s\n", reached, verdict)
	// Confirmation airtight : SeasonId du plus ancien match (via 1 GetMatchStats).
	if oldestID != "" {
		if stats, serr := client.GetMatchStats(ctx, oldestID); serr == nil {
			fmt.Printf("plus ancien match %s : MatchInfo.SeasonId=%q  playlist=%s\n",
				short(oldestID), seasonIDOf(stats), plName(playlistAssetID(stats)))
		} else {
			fmt.Printf("plus ancien match %s : GetMatchStats erreur : %v\n", short(oldestID), serr)
		}
	}
	fmt.Printf("=======================================\n")
}

// seasonIDOf lit MatchInfo.SeasonId (ex : "Csr/Seasons/CsrSeason12-1.json").
func seasonIDOf(stats map[string]any) string {
	mi, _ := stats["MatchInfo"].(map[string]any)
	if mi == nil {
		return ""
	}
	sid, _ := mi["SeasonId"].(string)
	return sid
}
