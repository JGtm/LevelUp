// cmd/token-capture — obtient un refresh_token Microsoft pour un joueur via Device Code Flow.
//
// Flow :
//  1. Initie un Device Code Flow sur login.microsoftonline.com (même Azure app que le serveur).
//  2. Affiche le lien + code à envoyer au joueur — il s'authentifie dans son navigateur.
//  3. Poll jusqu'à confirmation, écrit le refresh_token dans un fichier texte.
//
// Usage :
//
//	go run ./cmd/token-capture/ [gamertag]
//	go run ./cmd/token-capture/ Madina97294
//
// Le fichier de sortie (token_<gamertag>.txt) contient la ligne prête à coller dans .env.local :
//
//	SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294=<refresh_token>
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
)

const (
	// defaultClientID : app publique LevelUp (fallback si SPNKR_AZURE_CLIENT_ID absent).
	// Le serveur lit SPNKR_AZURE_CLIENT_ID en priorité (oauth_refresh.go) — ce binaire
	// doit utiliser le même client_id pour que le token capturé soit refreshable par le serveur.
	defaultClientID = "e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca" // #nosec G101 -- app publique
	deviceCodeURL   = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	tokenURL        = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	xboxScopes      = "Xboxlive.signin Xboxlive.offline_access"

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

// resolveClientID retourne SPNKR_AZURE_CLIENT_ID si défini, sinon defaultClientID.
// Doit correspondre au client_id utilisé par le serveur dans oauth_refresh.go.
func resolveClientID() string {
	if v := os.Getenv("SPNKR_AZURE_CLIENT_ID"); v != "" {
		return v
	}
	return defaultClientID
}

func main() {
	gamertag := "Madina97294"
	if len(os.Args) > 1 {
		gamertag = os.Args[1]
	}
	outputFile := fmt.Sprintf("token_%s.txt", gamertag)

	clientID := resolveClientID()
	fmt.Printf("Azure client_id utilisé : %s\n", clientID)

	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	dc, err := startDeviceFlow(ctx, clientID)
	if err != nil {
		fatalf("démarrage du Device Code Flow: %v\n", err)
	}

	printInstructions(gamertag, dc)

	_, refreshToken, err := pollToken(ctx, clientID, dc.DeviceCode, dc.Interval)
	if err != nil {
		fatalf("polling: %v\n", err)
	}
	if refreshToken == "" {
		fatalf("refresh_token absent de la réponse Microsoft\n")
	}

	envKey := fmt.Sprintf("SPNKR_OAUTH_REFRESH_TOKEN_%s", strings.ToUpper(gamertag))
	line := fmt.Sprintf("%s=%s\n", envKey, refreshToken)

	if err := os.WriteFile(outputFile, []byte(line), 0600); err != nil {
		fatalf("écriture %s: %v\n", outputFile, err)
	}

	fmt.Printf("\nToken écrit dans : %s\n", outputFile)
	fmt.Println("Contenu à coller dans .env.local :")
	fmt.Println()
	fmt.Print(line)
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
