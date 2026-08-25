// cmd/refresh_golden_fixture — Re-capture le fixture canonique des highlight
// events depuis l'API Halo. À ré-exécuter quand l'API Halo change de format
// (le test golden échouera, indiquant qu'il faut refresher).
//
// Usage :
//
//	go run ./cmd/refresh_golden_fixture --gamertag JGtm [--match-id <UUID>]
//
// Si `--match-id` est omis, le tool prend le match le plus récent du joueur.
// Le fixture est écrit dans `internal/analysis/testdata/v41_chunk_he.bin`.
//
// **Pré-requis** : refresh token OAuth du joueur dans le MultiUserTokenStore
// (data/auth/watcher_tokens, source unique ADR 0023), et SPNKR_AZURE_CLIENT_ID
// dans `.env.local`.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	auth_platform "levelup/go-api/internal/platform/auth"
	go_sync "levelup/go-api/internal/sync"
)

const (
	defaultFixturePath  = "internal/analysis/testdata/v41_chunk_he.bin"
	manifestFixturePath = "internal/analysis/testdata/v41_film_manifest.json"
)

func main() {
	gamertag := flag.String("gamertag", "", "Gamertag pour résoudre le refresh token (MultiUserTokenStore)")
	matchID := flag.String("match-id", "", "Match ID UUID à capturer (par défaut : dernier match du joueur)")
	out := flag.String("out", defaultFixturePath, "Chemin de sortie du chunk binaire")
	manifestOut := flag.String("manifest-out", manifestFixturePath, "Chemin de sortie du manifest JSON (optionnel)")
	flag.Parse()

	if strings.TrimSpace(*gamertag) == "" {
		fmt.Fprintln(os.Stderr, "usage: refresh_golden_fixture --gamertag <GT> [--match-id <UUID>] [--out path]")
		os.Exit(2)
	}

	if err := run(*gamertag, *matchID, *out, *manifestOut); err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		os.Exit(1)
	}
}

func run(gamertag, matchID, outPath, manifestPath string) error {
	loadEnvLocal()

	ctx := context.Background()

	// ADR 0023 Phase 5 : refresh token depuis le MultiUserTokenStore, seule source.
	store := auth_platform.NewMultiUserTokenStore("data/auth/watcher_tokens")
	user, err := store.LoadByGamertag(gamertag)
	if err != nil || user == nil || user.OAuthRefreshToken == "" {
		return fmt.Errorf("aucun refresh token pour %s dans data/auth/watcher_tokens: %w", gamertag, err)
	}
	res, err := auth_platform.RefreshHaloTokensViaStoreFirst(
		ctx, store, auth_platform.NewSISUProvider(), user.XUID, gamertag)
	if err != nil {
		return fmt.Errorf("refresh store: %w", err)
	}
	tokens := auth_platform.HaloTokensFromExchange(res)
	if tokens == nil {
		return fmt.Errorf("aucun token Halo obtenu pour %s", gamertag)
	}
	spartan, clearance := tokens.SpartanToken, tokens.ClearanceToken

	client := go_sync.NewHaloAPIClient(spartan, clearance, 2)

	if matchID == "" {
		hist, err := client.GetMatchHistory(ctx, gamertag, "all", 0, 1)
		if err != nil {
			return fmt.Errorf("GetMatchHistory: %w", err)
		}
		if len(hist) == 0 {
			return fmt.Errorf("aucun match trouvé pour %s", gamertag)
		}
		matchID = hist[0].MatchID
		fmt.Printf("dernière match : %s\n", matchID)
	}

	// Manifest brut (utile pour debug, optionnel — best-effort).
	if manifestPath != "" {
		if err := captureManifest(ctx, spartan, clearance, matchID, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] manifest capture: %v\n", err)
		} else {
			fmt.Printf("manifest -> %s\n", manifestPath)
		}
	}

	// Chunk highlight events (la fixture qui pilote le test golden).
	data, version, found, err := client.GetHighlightEventsChunk(ctx, matchID)
	if err != nil {
		return fmt.Errorf("GetHighlightEventsChunk: %w", err)
	}
	if !found {
		return fmt.Errorf("chunk highlight events absent pour ce match (film 404)")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("OK : chunk capturé\n")
	fmt.Printf("  match_id        : %s\n", matchID)
	fmt.Printf("  film_version    : %d\n", version)
	fmt.Printf("  size            : %d octets\n", len(data))
	fmt.Printf("  out             : %s\n", outPath)
	fmt.Printf("\nLance maintenant : go test ./internal/sync/ -run TestGolden\n")
	return nil
}

// captureManifest télécharge le manifest film et le persiste tel quel.
func captureManifest(ctx context.Context, spartan, clearance, matchID, outPath string) error {
	url := fmt.Sprintf("https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/%s/spectate", matchID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-343-authorization-spartan", spartan)
	req.Header.Set("343-clearance", clearance)
	req.Header.Set("Accept", "application/json")

	c := &http.Client{Timeout: 20 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, pretty, 0o644)
}

// loadEnvLocal lit `.env.local` à la racine du repo (pattern repris de
// cmd/get-token).
func loadEnvLocal() {
	for _, path := range []string{".env.local", "../.env.local", "../../.env.local"} {
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
