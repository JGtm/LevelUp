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
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
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

	xuid, canonicalGT, err := resolveXUID(cfg, gamertag)
	if err != nil {
		fatalf("%v\n", err)
	}

	refreshToken, err := readRefreshTokenFromStdin()
	if err != nil {
		fatalf("lecture stdin: %v\n", err)
	}
	if refreshToken == "" {
		fatalf("refresh_token vide en stdin\n")
	}

	storeDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.UpdateOAuthRefreshToken(xuid, refreshToken); err != nil {
		fatalf("écriture store: %v\n", err)
	}

	if existing, _ := store.Load(xuid); existing != nil && existing.Gamertag == "" {
		existing.Gamertag = canonicalGT
		_ = store.Upsert(existing)
	}

	halo.InvalidateCachedPlayerTokens(xuid)

	fmt.Printf("OK — refresh_token persisté pour %s (xuid=%s)\n", canonicalGT, xuid)
	fmt.Printf("Fichier : %s\\%s.json\n", storeDir, xuid)
}

// readRefreshTokenFromStdin lit stdin entier (jusqu'à EOF) puis trim espaces +
// retours-ligne. Accepte les formats :
//   - token brut sur une ligne
//   - ligne `SPNKR_OAUTH_REFRESH_TOKEN_GAMERTAG=value` (parse côté droit)
func readRefreshTokenFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format env-var-line : extraire la partie droite.
		if idx := strings.Index(line, "="); idx > 0 && strings.HasPrefix(line, "SPNKR_OAUTH_REFRESH_TOKEN_") {
			line = line[idx+1:]
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.TrimSpace(lines[0]), nil
}

// resolveXUID identique à token-capture (DRY évité — projet préfère duplication
// minime entre 2 binaires distincts plutôt qu'un package partagé pour 30L).
func resolveXUID(cfg *config.AppConfig, gamertag string) (xuid, canonicalGT string, err error) {
	players, lerr := cfg.LoadPlayers()
	if lerr != nil {
		return "", "", fmt.Errorf("LoadPlayers: %w", lerr)
	}
	target := strings.ToLower(strings.TrimSpace(gamertag))
	for _, p := range players {
		if strings.ToLower(p.Gamertag) == target {
			if p.XUID == "" {
				return "", "", fmt.Errorf("joueur %q présent mais xuid manquant dans db_profiles.json", p.Gamertag)
			}
			return p.XUID, p.Gamertag, nil
		}
	}
	return "", "", fmt.Errorf("joueur %q absent de db_profiles.json — ajouter une entrée avec xuid avant token-import", gamertag)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERREUR: "+format, args...)
	os.Exit(1)
}
