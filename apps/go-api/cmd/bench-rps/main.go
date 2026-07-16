// Package main — cmd/bench-rps : benchmark dry-run du rate-limiter HaloAPIClient.
//
// OBJECTIF
// --------
//
// Mesurer le wall-clock d'un cycle "sync delta" pour différentes valeurs de RPS
// (1/3/5/...) afin de valider la valeur RequestsPerSecond utilisée en prod
// (cmd/server/main.go:809).
//
// CONTRAINTE : aucune écriture DB. Le bench s'arrête à l'équivalent de la
// Phase 2 du SyncEngine (fetchMatchData) — pas d'insertion shared/player.
//
// MODES
// -----
//
//   - sim  : httptest server local qui simule la latence réseau Halo (150ms
//     par appel par défaut). Aucun token, aucune connexion externe.
//   - real : vraie API Halo. Nécessite SPARTAN_TOKEN + CLEARANCE_TOKEN en env.
//     Consomme du quota — utiliser un compte de test.
//
// USAGE
// -----
//
//	# Simulation (par défaut)
//	go run ./cmd/bench-rps -rps 1,3,5
//
//	# Avec latence custom
//	go run ./cmd/bench-rps -rps 1,3,5 -latency-ms 200
//
//	# Réel (consomme quota API)
//	SPARTAN_TOKEN=... CLEARANCE_TOKEN=... go run ./cmd/bench-rps \
//	    -mode real -xuid 2535469190789936 -rps 1,3,5
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

// redirectTransport réécrit toutes les URLs sortantes vers `host`. Permet à un
// HaloAPIClient (qui parle aux hosts Halo en dur) de taper un httptest.Server.
type redirectTransport struct {
	host string
	base http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = rt.host
	return rt.base.RoundTrip(clone)
}

func main() {
	mode := flag.String("mode", "sim", "Mode : 'sim' | 'real' (1 token) | 'real-multi' (N tokens concurrents).")
	xuid := flag.String("xuid", "", "XUID joueur (mode real uniquement). Sans 'xuid()'.")
	gamertagsRaw := flag.String("gamertags", "",
		"Mode real-multi : liste 'NAME:XUID,NAME:XUID' (ex: 'JGTM:2533274823110022,CHOCOBOFLOR:2535469190789936'). "+
			"NAME sert à lire SPNKR_OAUTH_REFRESH_TOKEN_<NAME> depuis .env.local.")
	count := flag.Int("count", 25, "Nombre de matchs à fetcher (max 25).")
	rpsListRaw := flag.String("rps", "1,3,5", "Liste RPS à bencher, séparés par virgule.")
	latencyMs := flag.Int("latency-ms", 150, "Latence simulée par requête (mode sim uniquement).")
	flag.Parse()

	rpsList, err := parseRPSList(*rpsListRaw)
	if err != nil {
		fatalf("flag -rps invalide : %v", err)
	}
	if *count < 1 || *count > 25 {
		fatalf("flag -count doit être entre 1 et 25 (reçu %d)", *count)
	}

	ctx := context.Background()

	switch *mode {
	case "sim":
		runSimMode(ctx, *count, *latencyMs, rpsList)
	case "real":
		if *xuid == "" {
			fatalf("flag -xuid requis en mode real")
		}
		runRealMode(ctx, *xuid, *count, rpsList)
	case "real-multi":
		if *gamertagsRaw == "" {
			fatalf("flag -gamertags requis en mode real-multi (ex: 'JGTM:2533274823110022,CHOCOBOFLOR:2535469190789936')")
		}
		runRealMultiMode(ctx, *gamertagsRaw, *count, rpsList)
	default:
		fatalf("flag -mode invalide : %q (attendu : sim|real|real-multi)", *mode)
	}
}

// runSimMode benche contre un httptest server qui retourne du JSON minimal après
// `latencyMs` de sleep. Reproduit la séquence d'appels d'un sync delta réel :
// 1× GetMatchHistory + N× GetMatchStats + N× GetMatchSkill.
func runSimMode(ctx context.Context, count, latencyMs int, rpsList []int) {
	latency := time.Duration(latencyMs) * time.Millisecond
	srv := newSimulatorServer(latency, count)
	defer srv.Close()

	srvHost := mustHostFromURL(srv.URL)

	fmt.Printf("=== BENCH-RPS — MODE SIM ===\n")
	fmt.Printf("Latence simulée : %v par requête\n", latency)
	fmt.Printf("Matchs par cycle : %d (1 history + %d stats + %d skill = %d appels)\n\n",
		count, count, count, 2*count+1)

	results := make([]benchResult, 0, len(rpsList))
	for _, rps := range rpsList {
		httpClient := &http.Client{
			Transport: &redirectTransport{host: srvHost, base: srv.Client().Transport},
			Timeout:   20 * time.Second,
		}
		client := syncpkg.NewHaloAPIClient("spartan", "clearance", rps).WithHTTPClient(httpClient)
		r := runCycleAgainstClient(ctx, client, "xuid(simulated)", rps, count)
		results = append(results, r)
	}
	printSummary(results)
}

func mustHostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		fatalf("URL httptest invalide : %v", err)
	}
	return u.Host
}

// runRealMode utilise la vraie API Halo. NE FAIT AUCUNE écriture DB.
func runRealMode(ctx context.Context, xuid string, count int, rpsList []int) {
	spartan := os.Getenv("SPARTAN_TOKEN")
	clearance := os.Getenv("CLEARANCE_TOKEN")
	if spartan == "" || clearance == "" {
		fatalf("SPARTAN_TOKEN et CLEARANCE_TOKEN requis en mode real")
	}

	fmt.Printf("=== BENCH-RPS — MODE REAL ===\n")
	fmt.Printf("XUID : %s\n", xuid)
	fmt.Printf("Matchs par cycle : %d (1 history + %d stats + %d skill = %d appels)\n",
		count, count, count, 2*count+1)
	fmt.Printf("ATTENTION : consomme du quota API Halo. Aucune écriture DB.\n\n")

	playerID := fmt.Sprintf("xuid(%s)", xuid)

	results := make([]benchResult, 0, len(rpsList))
	for _, rps := range rpsList {
		client := syncpkg.NewHaloAPIClient(spartan, clearance, rps)
		r := runCycleAgainstClient(ctx, client, playerID, rps, count)
		results = append(results, r)
	}
	printSummary(results)
}

// benchResult agrège les métriques d'un cycle pour un RPS donné.
type benchResult struct {
	RPS              int
	TotalCalls       int
	WallClock        time.Duration
	EffectiveRPS     float64
	Errors           int
	RateLimitHits429 int
}

// runCycleAgainstClient est le cœur du bench : il appelle GetMatchHistory puis
// fetche en parallèle stats+skill pour chaque match.
func runCycleAgainstClient(ctx context.Context, client *syncpkg.HaloAPIClient, playerID string, rps, count int) benchResult {
	start := time.Now()
	var errCount, rate429 atomic.Int64

	// 1. Récupère l'historique (1 appel).
	entries, err := client.GetMatchHistory(ctx, playerID, "matchmaking", 0, count)
	if err != nil {
		classifyErr(err, &errCount, &rate429)
		fmt.Printf("RPS=%d : GetMatchHistory échoué : %v\n", rps, err)
		return benchResult{RPS: rps, WallClock: time.Since(start), Errors: int(errCount.Load())}
	}

	// 2. Fetch parallèle (errgroup, pas de SetLimit — c'est le rate-limiter
	//    qui sérialise). Reproduit engine.go:312-331.
	eg, egCtx := errgroup.WithContext(ctx)
	for _, e := range entries {
		matchID := e.MatchID
		eg.Go(func() error {
			if _, err := client.GetMatchStats(egCtx, matchID); err != nil {
				classifyErr(err, &errCount, &rate429)
			}
			// GetMatchSkill nécessite la liste des XUIDs — on en passe un seul
			// (le mode sim ignore le contenu, le mode real retournera une
			// réponse limitée mais le coût RPS est mesuré pareil).
			if _, err := client.GetMatchSkill(egCtx, matchID, []string{"0"}); err != nil {
				classifyErr(err, &errCount, &rate429)
			}
			return nil
		})
	}
	_ = eg.Wait()

	wallClock := time.Since(start)
	totalCalls := 1 + 2*len(entries)
	effRPS := float64(totalCalls) / wallClock.Seconds()

	return benchResult{
		RPS:              rps,
		TotalCalls:       totalCalls,
		WallClock:        wallClock,
		EffectiveRPS:     effRPS,
		Errors:           int(errCount.Load()),
		RateLimitHits429: int(rate429.Load()),
	}
}

func classifyErr(err error, errCount, rate429 *atomic.Int64) {
	if err == nil {
		return
	}
	errCount.Add(1)
	var herr *syncpkg.HTTPError
	if errors.As(err, &herr) && herr.StatusCode == http.StatusTooManyRequests {
		rate429.Add(1)
	}
}

func printSummary(results []benchResult) {
	fmt.Println()
	fmt.Println("┌─────────┬───────────┬──────────────┬──────────────┬────────┬───────┐")
	fmt.Println("│ RPS cfg │ Appels    │ Wall-clock   │ RPS effectif │ Errors │ 429   │")
	fmt.Println("├─────────┼───────────┼──────────────┼──────────────┼────────┼───────┤")
	for _, r := range results {
		fmt.Printf("│ %-7d │ %-9d │ %-12v │ %-12.2f │ %-6d │ %-5d │\n",
			r.RPS, r.TotalCalls, r.WallClock.Round(10*time.Millisecond),
			r.EffectiveRPS, r.Errors, r.RateLimitHits429)
	}
	fmt.Println("└─────────┴───────────┴──────────────┴──────────────┴────────┴───────┘")
}

// newSimulatorServer crée un httptest.Server qui :
//   - GET /hi/players/{gt}/matches : retourne `count` entries (latence latencyMs).
//   - GET /hi/matches/{id}/stats  : retourne stats JSON minimal (latence latencyMs).
//   - POST .../skill              : retourne map vide (latence latencyMs).
//
// Toutes les routes sleepent `latency` avant de répondre pour simuler le RTT
// réseau Halo (typiquement 100-200ms).
func newSimulatorServer(latency time.Duration, count int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(latency)
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/matches") && strings.Contains(path, "/players/"):
			// Historique : génère `count` entries factices.
			results := make([]map[string]any, count)
			for i := 0; i < count; i++ {
				results[i] = map[string]any{
					"MatchId": fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
					"MatchInfo": map[string]any{
						"StartTime": "2025-01-01T00:00:00Z",
					},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Start":       0,
				"Count":       count,
				"ResultCount": count,
				"Results":     results,
			})
		case strings.HasSuffix(path, "/stats"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"MatchId": "00000000-0000-0000-0000-000000000001",
				"MatchInfo": map[string]any{
					"StartTime": "2025-01-01T00:00:00Z",
					"EndTime":   "2025-01-01T00:10:00Z",
				},
				"Players": []any{},
			})
		default:
			// /skill et autres endpoints non spécifiés
			_ = json.NewEncoder(w).Encode(map[string]any{"Value": []any{}})
		}
	}))
}

// gamertagSpec parse un item "NAME:XUID" du flag -gamertags.
type gamertagSpec struct {
	Name string // upper-case suffix de SPNKR_OAUTH_REFRESH_TOKEN_*
	XUID string
}

func parseGamertagsFlag(raw string) ([]gamertagSpec, error) {
	parts := strings.Split(raw, ",")
	out := make([]gamertagSpec, 0, len(parts))
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("entrée invalide %q (attendu NAME:XUID)", p)
		}
		out = append(out, gamertagSpec{Name: strings.ToUpper(kv[0]), XUID: kv[1]})
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("mode real-multi nécessite ≥ 2 gamertags (reçu %d)", len(out))
	}
	return out, nil
}

// loadEnvLocal lit .env.local (racine repo) et exporte les vars dans os.Setenv.
// Copie du pattern utilisé par cmd/get-token. Silencieux si fichier absent.
func loadEnvLocal() {
	for _, path := range []string{".env.local", "../../.env.local", "../../../.env.local"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
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
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		return
	}
}

// exchangeTokens convertit un refresh_token OAuth en (Spartan, Clearance) tokens
// frais via le pipeline MSAL + Halo Waypoint exchange. Réplique cmd/get-token.
func exchangeTokens(ctx context.Context, refreshToken string) (spartan, clearance string, err error) {
	provider := auth.NewSISUProvider()
	tok, err := provider.TryOAuthRefresh(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("TryOAuthRefresh: %w", err)
	}
	result, err := auth.ExchangeAccessToken(ctx, tok)
	if err != nil {
		return "", "", fmt.Errorf("ExchangeAccessToken: %w", err)
	}
	return result.Tokens.SpartanToken, result.Tokens.ClearanceToken, nil
}

// runRealMultiMode : pour chaque RPS configuré, lance N cycles concurrents
// (1 par gamertag), chacun avec son propre HaloAPIClient à RPS dédié. Mesure
// le wall-clock global + 429 par token. Reproduit l'option 2 (5 RPS par-token).
func runRealMultiMode(ctx context.Context, gamertagsRaw string, count int, rpsList []int) {
	loadEnvLocal()
	specs, err := parseGamertagsFlag(gamertagsRaw)
	if err != nil {
		fatalf("flag -gamertags : %v", err)
	}

	fmt.Printf("=== BENCH-RPS — MODE REAL-MULTI ===\n")
	fmt.Printf("Tokens concurrents : %d\n", len(specs))
	fmt.Printf("Matchs par cycle : %d (1 history + %d stats + %d skill = %d appels par token)\n",
		count, count, count, 2*count+1)
	fmt.Printf("Total par RPS : %d cycles × %d appels = %d appels API\n",
		len(specs), 2*count+1, len(specs)*(2*count+1))
	fmt.Printf("ATTENTION : consomme du quota API Halo. Aucune écriture DB.\n\n")

	// Exchange tokens en parallèle.
	type tokenSet struct {
		spec      gamertagSpec
		spartan   string
		clearance string
	}
	tokensets := make([]tokenSet, len(specs))
	for i, spec := range specs {
		rt := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + spec.Name)
		if rt == "" {
			fatalf("SPNKR_OAUTH_REFRESH_TOKEN_%s absent de .env.local", spec.Name)
		}
		spartan, clearance, err := exchangeTokens(ctx, rt)
		if err != nil {
			fatalf("exchange tokens pour %s : %v", spec.Name, err)
		}
		tokensets[i] = tokenSet{spec: spec, spartan: spartan, clearance: clearance}
		fmt.Printf("Token %s prêt (spartan %d chars)\n", spec.Name, len(spartan))
	}
	fmt.Println()

	for _, rps := range rpsList {
		start := time.Now()
		var wg sync.WaitGroup
		results := make([]benchResult, len(tokensets))

		for i, ts := range tokensets {
			i, ts := i, ts
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := syncpkg.NewHaloAPIClient(ts.spartan, ts.clearance, rps)
				playerID := fmt.Sprintf("xuid(%s)", ts.spec.XUID)
				results[i] = runCycleAgainstClient(ctx, client, playerID, rps, count)
			}()
		}
		wg.Wait()
		wallClock := time.Since(start)

		// Aggregate.
		totalCalls, totalErrs, total429 := 0, 0, 0
		for _, r := range results {
			totalCalls += r.TotalCalls
			totalErrs += r.Errors
			total429 += r.RateLimitHits429
		}
		fmt.Printf("RPS=%d par-token (%d tokens) :\n", rps, len(tokensets))
		fmt.Printf("  Wall-clock global : %v\n", wallClock.Round(10*time.Millisecond))
		fmt.Printf("  RPS effectif global : %.2f (cible théorique : %d)\n",
			float64(totalCalls)/wallClock.Seconds(), rps*len(tokensets))
		fmt.Printf("  Total appels : %d | erreurs : %d | 429 : %d\n", totalCalls, totalErrs, total429)
		for i, r := range results {
			fmt.Printf("    [%s] %v, %d appels, %d errs, %d × 429\n",
				tokensets[i].spec.Name, r.WallClock.Round(10*time.Millisecond),
				r.TotalCalls, r.Errors, r.RateLimitHits429)
		}
		fmt.Println()
	}
}

func parseRPSList(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("valeur RPS invalide %q : %w", p, err)
		}
		if v < 1 {
			return nil, fmt.Errorf("RPS doit être ≥ 1 (reçu %d)", v)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("liste RPS vide")
	}
	return out, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bench-rps: "+format+"\n", args...)
	os.Exit(1)
}
