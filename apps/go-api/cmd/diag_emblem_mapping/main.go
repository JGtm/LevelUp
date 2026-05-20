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

	"levelup/go-api/internal/platform/auth"
)

func main() {
	want := "olympus_campaign_windfall"
	if len(os.Args) > 1 {
		want = os.Args[1]
	}
	gamertag := "JGtm"
	if len(os.Args) > 2 {
		gamertag = os.Args[2]
	}
	rt := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p := auth.NewMSALProvider()
	at, _ := p.TryOAuthRefresh(ctx, rt)
	res, _ := p.Exchange(ctx, at)

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
