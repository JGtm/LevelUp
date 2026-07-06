//go:build dev

// Command get-token — utilitaire DEV : échange le refresh token OAuth
// (SPNKR_OAUTH_REFRESH_TOKEN_JGTM) contre un Spartan + Clearance token et les
// imprime EN CLAIR sur stdout.
//
// AVERTISSEMENT SÉCURITÉ (S9, lot S) : la sortie contient des tokens Halo
// valides. Ne JAMAIS la capturer, rediriger vers un fichier, coller dans un
// ticket/log, ni committer. Réservé au debug manuel local. Le tag de build `dev`
// l'exclut de `go build ./...` et des binaires de prod :
//
//	go run -tags dev ./cmd/get-token
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/platform/auth"
)

func main() {
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
				os.Setenv(key, val)
			}
		}
		break
	}

	rt := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM")
	if rt == "" {
		fmt.Fprintln(os.Stderr, "no token")
		return
	}

	provider := auth.NewMSALProvider()
	tok, err := provider.TryOAuthRefresh(context.Background(), rt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	result, err := auth.ExchangeAccessToken(context.Background(), tok)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("SPARTAN=" + result.Tokens.SpartanToken)
	fmt.Println("CLEARANCE=" + result.Tokens.ClearanceToken)
}
