//go:build dev

// Command get-token — utilitaire DEV : échange le refresh token OAuth du
// MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json, source unique
// ADR 0023) contre un Spartan + Clearance token et les imprime EN CLAIR sur
// stdout.
//
// AVERTISSEMENT SÉCURITÉ (S9, lot S) : la sortie contient des tokens Halo
// valides. Ne JAMAIS la capturer, rediriger vers un fichier, coller dans un
// ticket/log, ni committer. Réservé au debug manuel local. Le tag de build `dev`
// l'exclut de `go build ./...` et des binaires de prod :
//
//	go run -tags dev ./cmd/get-token [gamertag] [watcher-tokens-dir]
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/platform/auth"
)

func main() {
	// .env.local reste nécessaire pour SPNKR_AZURE_CLIENT_ID (refresh OAuth).
	loadEnvLocal()

	gamertag := "JGtm"
	if len(os.Args) > 1 {
		gamertag = os.Args[1]
	}
	storeDir := "data/auth/watcher_tokens"
	if len(os.Args) > 2 {
		storeDir = os.Args[2]
	}

	store := auth.NewMultiUserTokenStore(storeDir)
	user, err := store.LoadByGamertag(gamertag)
	if err != nil || user == nil || user.OAuthRefreshToken == "" {
		fmt.Fprintf(os.Stderr, "aucun refresh token pour %s dans %s (err=%v)\n", gamertag, storeDir, err)
		return
	}

	result, err := auth.RefreshHaloTokensViaStoreFirst(
		context.Background(), store, auth.NewSISUProvider(), user.XUID, gamertag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tokens := auth.HaloTokensFromExchange(result)
	if tokens == nil {
		fmt.Fprintf(os.Stderr, "aucun token Halo obtenu pour %s\n", gamertag)
		return
	}

	// AVERTISSEMENT SÉCURITÉ (audit S9) : ces lignes impriment le SpartanToken /
	// ClearanceToken EN CLAIR sur stdout. C'est la raison d'être de ce CLI dev (usage :
	// capturer la sortie pour un appel API ponctuel). Ne JAMAIS rediriger/capturer cette
	// sortie dans un fichier de log, une CI ou un partage — token exploitable en clair.
	fmt.Println("SPARTAN=" + tokens.SpartanToken)
	fmt.Println("CLEARANCE=" + tokens.ClearanceToken)
}

// loadEnvLocal charge .env.local dans l'environnement du process (sans écraser
// les variables déjà définies). Requis pour SPNKR_AZURE_CLIENT_ID.
func loadEnvLocal() {
	for _, path := range []string{".env.local", "../../.env.local"} {
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
