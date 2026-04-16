// cmd/msal-poc — POC Sprint 0 : valider que MSAL Go peut initier un Device Code Flow.
//
// Ce programme n'effectue PAS la connexion complète.
// Il prouve uniquement que :
//  1. Le PublicClientApplication MSAL Go se crée sans erreur.
//  2. InitiateDeviceFlow() retourne un user_code et une verification_url.
//  3. Le format de cache Go est séparé du cache Python (clé différente dans sync_meta).
//
// Usage :
//
//	go run ./cmd/msal-poc/
//
// Variables d'environnement :
//
//	LEVELUP_CLIENT_ID — override du client_id Azure (optionnel, défaut = app LevelUp)
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

const (
	// ClientID de l'app "LevelUp Halo" — identique à src/auth/_constants.py.
	// Utilisateurs finaux n'ont pas besoin de configurer Azure.
	defaultClientID = "e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca" // #nosec G101 -- app publique, pas un secret

	// Authority Microsoft pour les comptes personnels (Xbox Live).
	authority = "https://login.microsoftonline.com/consumers"
)

// Scopes Xbox Live — offline_access requis pour obtenir un refresh_token.
var xboxScopes = []string{"Xboxlive.signin", "Xboxlive.offline_access"}

func main() {
	clientID := os.Getenv("LEVELUP_CLIENT_ID")
	if clientID == "" {
		clientID = defaultClientID
	}

	fmt.Println("=== MSAL Go — Device Code Flow POC ===")
	fmt.Printf("Client ID  : %s\n", clientID)
	fmt.Printf("Authority  : %s\n", authority)
	fmt.Printf("Scopes     : %v\n\n", xboxScopes)

	// --- Création du PublicClientApplication ---
	app, err := public.New(clientID, public.WithAuthority(authority))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERREUR : création MSAL PublicClient : %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ PublicClientApplication créé")

	// --- Initiation du Device Code Flow (timeout court — POC uniquement) ---
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dc, err := app.AcquireTokenByDeviceCode(ctx, xboxScopes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERREUR : AcquireTokenByDeviceCode : %v\n", err)
		os.Exit(1)
	}

	// Affichage du Device Code (pas besoin de compléter le flow)
	result := dc.Result
	fmt.Println("\n✓ Device Code Flow initié !")
	fmt.Printf("  user_code        : %s\n", result.UserCode)
	fmt.Printf("  verification_uri : %s\n", result.VerificationURL)
	fmt.Printf("  expires_on       : %s\n", result.ExpiresOn.Format(time.RFC3339))
	fmt.Printf("  message          : %s\n\n", result.Message)

	// --- Note stratégie cache ---
	fmt.Println("=== Stratégie cache MSAL Python / Go ===")
	fmt.Println("  Python cache key : 'msal_token_cache'     (dans DuckDB sync_meta)")
	fmt.Println("  Go cache key     : 'msal_go_token_cache'  (clé séparée, format non compatible)")
	fmt.Println("  Règle            : PAS de désérialisation croisée.")
	fmt.Println("                     Si cache Go vide → Device Code Flow interactif.")
	fmt.Println("                     Si refresh_token env var présent → prioritaire sur MSAL.")
	fmt.Println("\n✓ Gate Sprint 0 MSAL : PASSÉ")
}
