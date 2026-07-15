//go:build cgo

// diag_emblem_colors — dump complet du JSON emblem pour voir si chaque
// ConfigurationId a une palette de couleurs associée. Permet de choisir
// la cfg la plus proche de la palette équipée par le joueur (cas JGtm où
// la cfg -809699482 n'est pas servable mais on prend la 1ère positive
// qui peut avoir les couleurs inversées).
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
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// acquireDiagTokens fait le pipeline ADR 0023 : MultiUserTokenStore → env var fallback.
// Persiste la rotation au store si applicable. Pour CLI diagnostic one-shot.
func acquireDiagTokens(ctx context.Context, gamertag string) (*domain.HaloTokens, error) {
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

	legacy := auth.LegacyAuthInputs{
		OAuthRT: os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)),
		Source:  "env_var",
	}
	result, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, gamertag, legacy)
	if err != nil {
		return nil, err
	}
	if tokens := auth.HaloTokensFromExchange(result); tokens != nil {
		return tokens, nil
	}
	return nil, fmt.Errorf("aucun token disponible pour %s", gamertag)
}

func dumpCoating(coatingPath string) {
	gamertag := "JGtm"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tokens, err := acquireDiagTokens(ctx, gamertag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	url := "https://gamecms-hacs.svc.halowaypoint.com/hi/progression/file/" + coatingPath
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
	req.Header.Set("343-clearance", tokens.ClearanceToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL: HTTP error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("=== %s ===\nStatus: %d\n\n", url, resp.StatusCode)

	var data map[string]any
	if err := json.Unmarshal(body, &data); err == nil {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Println(string(body))
	}
}

func main() {
	emblemPath := "Inventory/Spartan/Emblems/104-001-olympus-campa-2ddbe23b.json"
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "Inventory/") {
		emblemPath = os.Args[1]
	}
	// Mode 2 : si arg1 commence par "configuration/", on dump ce coating
	if len(os.Args) > 1 && strings.HasPrefix(strings.ToLower(os.Args[1]), "configuration/") {
		dumpCoating(os.Args[1])
		return
	}
	gamertag := "JGtm"
	if len(os.Args) > 2 {
		gamertag = os.Args[2]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tokens, err := acquireDiagTokens(ctx, gamertag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	url := "https://gamecms-hacs.svc.halowaypoint.com/hi/progression/file/" + emblemPath
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
	req.Header.Set("343-clearance", tokens.ClearanceToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL: HTTP error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data map[string]any
	_ = json.Unmarshal(body, &data)

	fmt.Printf("=== Emblem: %s ===\n\n", emblemPath)

	// Top-level keys
	fmt.Println("Top-level keys:")
	for k := range data {
		fmt.Printf("  - %s\n", k)
	}
	fmt.Println()

	// AvailableConfigurations details
	configs, _ := data["AvailableConfigurations"].([]any)
	fmt.Printf("=== %d AvailableConfigurations ===\n", len(configs))
	for i, c := range configs {
		entry, _ := c.(map[string]any)
		pretty, _ := json.MarshalIndent(entry, "  ", "  ")
		fmt.Printf("\n[%d]:\n  %s\n", i, string(pretty))
	}

	// CommonData details (might contain default colors)
	if cd, ok := data["CommonData"].(map[string]any); ok {
		fmt.Println("\n=== CommonData ===")
		pretty, _ := json.MarshalIndent(cd, "  ", "  ")
		fmt.Println("  " + string(pretty))
	}
}
