//go:build cgo

// diag_emblem_mapping — fetche /hi/Waypoint/file/images/emblems/mapping.json
// (endpoint Grunt) qui mappe (emblem.Id, ConfigurationId) → NameplateCmsPath +
// TextColor. Permet de servir le NameplateCmsPath EXACT pour la palette
// équipée par le joueur (cas JGtm cfg=-809699482).
//
// Usage : diag_emblem_mapping <emblem_substring> [gamertag]
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// acquireDiagTokens : pipeline ADR 0023 (MultiUserTokenStore, source unique).
// Persiste la rotation au store. Même helper que diag_emblem_colors.
func acquireDiagTokens(ctx context.Context, gamertag string) (*auth.ExchangeResult, error) {
	provider := auth.NewSISUProvider()
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())

	var xuid string
	if user, err := store.LoadByGamertag(gamertag); err == nil && user != nil {
		xuid = user.XUID
	}
	return auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, gamertag)
}

func main() {
	want := "olympus_campaign_windfall"
	if len(os.Args) > 1 {
		want = os.Args[1]
	}
	gamertag := "JGtm"
	if len(os.Args) > 2 {
		gamertag = os.Args[2]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := acquireDiagTokens(ctx, gamertag)
	if err != nil || res == nil || res.Tokens == nil {
		fmt.Fprintf(os.Stderr, "FATAL: tokens %s: %v\n", gamertag, err)
		os.Exit(1)
	}

	// Mode probe : arg1 = URL complète → GET authentifié, print status/type/size.
	if strings.HasPrefix(want, "http") {
		req, _ := http.NewRequestWithContext(ctx, "GET", want, nil)
		req.Header.Set("x-343-authorization-spartan", res.Tokens.SpartanToken)
		req.Header.Set("343-clearance", res.Tokens.ClearanceToken)
		resp, perr := http.DefaultClient.Do(req)
		if perr != nil {
			fmt.Fprintln(os.Stderr, "probe error:", perr)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Probe %s\n  Status=%d Content-Type=%s Size=%d\n", want, resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
		return
	}

	url := "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/emblems/mapping.json"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("x-343-authorization-spartan", res.Tokens.SpartanToken)
	req.Header.Set("343-clearance", res.Tokens.ClearanceToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL: HTTP error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d, Size: %d bytes\n\n", resp.StatusCode, len(body))

	var data map[string]map[string]map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("parse err:", err)
		fmt.Println("Preview:", string(body)[:min(500, len(body))])
		return
	}

	found := 0
	for emblemKey, cfgs := range data {
		if !strings.Contains(strings.ToLower(emblemKey), strings.ToLower(want)) {
			continue
		}
		fmt.Printf("=== Emblem key: %q ===\n", emblemKey)
		fmt.Printf("=== %d cfgs mapped ===\n\n", len(cfgs))
		for cfg, mapping := range cfgs {
			fmt.Printf("ConfigurationId=%s\n", cfg)
			pretty, _ := json.MarshalIndent(mapping, "  ", "  ")
			fmt.Printf("  %s\n\n", pretty)
		}
		found++
	}
	if found == 0 {
		fmt.Printf("Emblem %q not found in mapping.json (total %d entries)\n", want, len(data))
		// Sample first 3 keys
		fmt.Println("\nSample keys:")
		i := 0
		for k := range data {
			fmt.Printf("  - %s\n", k)
			i++
			if i >= 5 {
				break
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
