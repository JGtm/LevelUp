// Outil de diagnostic (#3) : teste si l'API Halo OFFICIELLE (economy) renvoie
// l'appearance d'un joueur ARBITRAIRE (inconnu) avec les tokens d'un de nos
// joueurs (Spartan + clearance). Tranche : peut-on faire NOTRE propre solution
// sans dotapi.gg ?
//
// Usage : go run ./cmd/diag_appearance [ownerGamertag] [targetXUID]
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

func main() {
	ownerGT := "JGtm"
	targetXUID := "2535427927026623" // Nilton410 (inconnu, non local)
	if len(os.Args) > 1 {
		ownerGT = os.Args[1]
	}
	if len(os.Args) > 2 {
		targetXUID = os.Args[2]
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatalf("config.Load: %v", err)
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		fatalf("LoadPlayers: %v", err)
	}
	var ownerXUID string
	for i := range players {
		if players[i].Gamertag == ownerGT {
			ownerXUID = players[i].XUID
		}
	}
	if ownerXUID == "" {
		fatalf("owner %q introuvable dans db_profiles", ownerGT)
	}

	storeDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	provider := auth.NewSISUProvider()

	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, ownerXUID, ownerGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatalf("refresh tokens %s: err=%v res=%v", ownerGT, err, res)
	}
	spartan := res.Tokens.SpartanToken
	clearance := res.Tokens.ClearanceToken
	fmt.Printf("owner=%s spartan_len=%d clearance_len=%d\n", ownerGT, len(spartan), len(clearance))
	fmt.Printf("target_xuid=%s (inconnu)\n\n", targetXUID)

	endpoints := []string{
		fmt.Sprintf("https://economy.svc.halowaypoint.com/hi/players/xuid(%s)/customization/appearance", targetXUID),
		fmt.Sprintf("https://economy.svc.halowaypoint.com/hi/players/xuid(%s)/customization?view=public", targetXUID),
		// Sanity check : servicerecord (halostats, on sait que ça marche pour inconnus).
		fmt.Sprintf("https://halostats.svc.halowaypoint.com/hi/players/xuid(%s)/Matchmade/servicerecord", targetXUID),
	}
	client := &http.Client{Timeout: 20 * time.Second}
	for _, ep := range endpoints {
		status, body := probe(ctx, client, ep, spartan, clearance)
		preview := body
		if len(preview) > 24000 {
			preview = preview[:24000]
		}
		fmt.Printf("── %s\n   HTTP %d · %d bytes\n   %s\n\n", ep, status, len(body), preview)
	}
}

func probe(ctx context.Context, client *http.Client, url, spartan, clearance string) (int, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return -1, err.Error()
	}
	req.Header.Set("x-343-authorization-spartan", spartan)
	if clearance != "" {
		req.Header.Set("343-clearance", clearance)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return -1, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
