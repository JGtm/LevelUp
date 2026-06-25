// cmd/token-import — importe un refresh_token OAuth Microsoft existant dans le MultiUserTokenStore.
//
// Mode "advanced" : pour les utilisateurs qui ont déjà obtenu un refresh_token
// par un autre moyen (export d'un autre outil, partage manuel, etc.) et veulent
// l'injecter sans passer par le Device Code Flow interactif.
//
// Le refresh_token est lu sur STDIN (pas en argv) pour éviter qu'il apparaisse
// dans `ps`, l'historique shell, ou les logs du shell.
//
// Usage :
//
//	echo "M.C544_SN1.0.U.-..." | go run ./cmd/token-import/ Madina97294
//	cat token_madina.txt | go run ./cmd/token-import/ Madina97294
//
// Le joueur doit être déclaré dans db_profiles.json (avec xuid) avant l'import.
//
// L'essentiel de la logique est dans internal/platform/auth/capturecli/ pour
// permettre les tests unitaires sans device flow.
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	"levelup/go-api/internal/platform/halo"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: cat <token-file> | token-import <gamertag>\n")
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
	xuid, canonicalGT, err := capturecli.ResolveXUIDByGamertag(players, gamertag)
	if err != nil {
		fatalf("%v\n", err)
	}

	refreshToken, err := capturecli.ParseRefreshTokenStdin(os.Stdin)
	if err != nil {
		fatalf("lecture stdin: %v\n", err)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	storeDir := pr.WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := capturecli.PersistRefreshToken(store, xuid, canonicalGT, refreshToken, halo.InvalidateCachedPlayerTokens); err != nil {
		fatalf("persistance RT: %v\n", err)
	}

	fmt.Printf("OK — refresh_token persisté pour %s (xuid=%s)\n", canonicalGT, xuid)
	fmt.Printf("Fichier : %s/%s.json\n", storeDir, xuid)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERREUR: "+format, args...)
	os.Exit(1)
}
