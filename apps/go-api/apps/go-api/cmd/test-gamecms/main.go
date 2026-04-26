package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	authpkg "levelup/go-api/internal/platform/auth"
)

func main() {
	ctx := context.Background()
	rt := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM")
	provider := authpkg.NewMSALProvider()
	at, _ := provider.TryOAuthRefresh(ctx, rt)
	res, _ := provider.Exchange(ctx, at)
	if res == nil {
		fmt.Println("Exchange failed")
		return
	}
	fmt.Println("Spartan len:", len(res.Tokens.SpartanToken), "Clearance len:", len(res.Tokens.ClearanceToken))

	url := "https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/RewardTracks/CareerRanks/careerRank1.json"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("x-343-authorization-spartan", res.Tokens.SpartanToken)
	if res.Tokens.ClearanceToken != "" {
		req.Header.Set("343-clearance", res.Tokens.ClearanceToken)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.StatusCode, "len:", len(body))
	// Find first rank with translations
	var raw map[string]interface{}
	json.Unmarshal(body, &raw)
	if ranks, ok := raw["Ranks"].([]interface{}); ok && len(ranks) > 1 {
		r := ranks[1].(map[string]interface{})
		title := r["RankTitle"]
		fmt.Println("Sample RankTitle:")
		jsonOut, _ := json.MarshalIndent(title, "  ", "  ")
		fmt.Println(string(jsonOut))
	}
	_ = strings.Contains
}
