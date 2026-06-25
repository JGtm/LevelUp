// cmd/token-capture — obtient un refresh_token Microsoft pour un joueur via Device Code Flow
// et l'écrit directement dans le MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
//
// Flow :
//  1. Résout xuid depuis db_profiles.json via gamertag (config.LoadPlayers).
//  2. Initie un Device Code Flow sur login.microsoftonline.com (même Azure app que le serveur).
//  3. Affiche le lien + code à envoyer au joueur — il s'authentifie dans son navigateur.
//  4. Poll jusqu'à confirmation.
//  5. Écrit le refresh_token directement dans le store (UpdateOAuthRefreshToken) + complète
//     l'entrée avec gamertag/xuid si nouvelle.
//  6. Invalide le cache process des HaloTokens pour ce xuid (force re-acquire au prochain refresh).
//
// Aucune manipulation manuelle de .env.local nécessaire — au prochain redémarrage du serveur,
// le Pool trouve le token dans le store et fonctionne immédiatement.
//
// Usage :
//
//	go run ./cmd/token-capture/ <gamertag>
//	go run ./cmd/token-capture/ Madina97294
//
// Si le joueur n'est pas dans db_profiles.json, le tool affiche une erreur explicite.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	"levelup/go-api/internal/platform/halo"
)

const (
	deviceCodeURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	tokenURL      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	xboxScopes    = "Xboxlive.signin Xboxlive.offline_access"

	authTimeout           = 15 * time.Minute
	pollSlowDownIncrement = 5
	pollMaxInterval       = 60
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// resolveClientID délègue à la source unique des credentials Azure (seam).
// Doit correspondre au client_id du refresh serveur — cf. auth.TokenCaptureClientID
// (couplage : un refresh_token est lié à son client émetteur).
func resolveClientID() string {
	return auth.TokenCaptureClientID()
}

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: token-capture <gamertag>\n  ex: token-capture Madina97294\n")
	}
	gamertag := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fatalf("config.Load: %v\n", err)
	}

	players, err := cfg.LoadPlayers()
	if err != nil {
		fatalf("LoadPlayers: %v\n", err)
	}
	xuid, resolvedGT, err := capturecli.ResolveXUIDByGamertag(players, gamertag)
	if err != nil {
		fatalf("%v\n", err)
	}

	clientID := resolveClientID()
	fmt.Printf("Azure client_id utilisé : %s\n", clientID)
	fmt.Printf("Joueur résolu : %s (xuid=%s)\n", resolvedGT, xuid)

	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	dc, err := startDeviceFlow(ctx, clientID)
	if err != nil {
		fatalf("démarrage du Device Code Flow: %v\n", err)
	}

	printInstructions(resolvedGT, dc)

	_, refreshToken, err := pollToken(ctx, clientID, dc.DeviceCode, dc.Interval)
	if err != nil {
		fatalf("polling: %v\n", err)
	}
	if refreshToken == "" {
		fatalf("refresh_token absent de la réponse Microsoft\n")
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	storeDir := pr.WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := capturecli.PersistRefreshToken(store, xuid, resolvedGT, refreshToken, halo.InvalidateCachedPlayerTokens); err != nil {
		fatalf("persistance RT: %v\n", err)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  OK — Token persisté dans le store canonique\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  Fichier : %s\\%s.json\n", storeDir, xuid)
	fmt.Println()
	fmt.Println("  Redémarrer le serveur (ou laisser Air relancer) — le Pool")
	fmt.Println("  trouvera le token immédiatement, aucune édition .env.local.")
	fmt.Println()
}

func startDeviceFlow(ctx context.Context, clientID string) (*deviceCodeResponse, error) {
	body := url.Values{
		"client_id": {clientID},
		"scope":     {xboxScopes},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var dc deviceCodeResponse
	if err := json.Unmarshal(raw, &dc); err != nil {
		return nil, fmt.Errorf("décodage JSON: %w (body: %s)", err, raw)
	}
	if dc.Error != "" {
		return nil, fmt.Errorf("%s: %s", dc.Error, dc.ErrorDesc)
	}
	if dc.DeviceCode == "" {
		return nil, fmt.Errorf("device_code absent (body: %s)", raw)
	}
	if dc.Interval <= 0 {
		dc.Interval = 5
	}
	return &dc, nil
}

func printInstructions(gamertag string, dc *deviceCodeResponse) {
	sep := strings.Repeat("=", 60)
	fmt.Println()
	fmt.Println(sep)
	fmt.Printf("  INSTRUCTIONS POUR %s\n", strings.ToUpper(gamertag))
	fmt.Println(sep)
	fmt.Println()
	fmt.Printf("  1. Ouvre ce lien dans ton navigateur :\n")
	fmt.Printf("     %s\n", dc.VerificationURI)
	fmt.Println()
	fmt.Printf("  2. Entre ce code :\n")
	fmt.Printf("     %s\n", dc.UserCode)
	fmt.Println()
	fmt.Printf("  3. Connecte-toi avec ton compte Microsoft Xbox.\n")
	fmt.Printf("     C'est tout ! Dis-moi quand c'est fait.\n")
	fmt.Println()
	fmt.Printf("  (expire dans %d minutes)\n", dc.ExpiresIn/60)
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("En attente de l'authentification...")
}

func pollToken(ctx context.Context, clientID, deviceCode string, interval int) (accessToken, refreshToken string, err error) {
	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		body := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {clientID},
			"device_code": {deviceCode},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
			strings.NewReader(body.Encode()))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tok tokenResponse
		if err := json.Unmarshal(raw, &tok); err != nil {
			return "", "", fmt.Errorf("décodage JSON poll: %w", err)
		}

		if tok.AccessToken != "" {
			fmt.Println("Authentification reussie !")
			return tok.AccessToken, tok.RefreshToken, nil
		}

		switch tok.Error {
		case "authorization_pending":
			// Normal — on continue
		case "slow_down":
			interval += pollSlowDownIncrement
			if interval > pollMaxInterval {
				interval = pollMaxInterval
			}
		case "":
			return "", "", fmt.Errorf("réponse inattendue (HTTP %d): %s", resp.StatusCode, raw)
		default:
			return "", "", fmt.Errorf("erreur Microsoft %q: %s", tok.Error, tok.ErrorDesc)
		}
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERREUR: "+format, args...)
	os.Exit(1)
}
