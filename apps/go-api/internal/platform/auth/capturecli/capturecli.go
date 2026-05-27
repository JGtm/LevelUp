// Package capturecli — helpers partagés par cmd/token-capture, cmd/token-import,
// et le callback onRotated du serveur principal.
//
// Extrait pour permettre les tests unitaires sans dépendance cgo (DuckDB) :
// le caller passe la liste des joueurs déjà chargée via cfg.LoadPlayers() et
// un CacheInvalidator pour l'invalidation cache process (typiquement
// halo.InvalidateCachedPlayerTokens). Ce package n'importe ni config ni duckdb
// ni halo — uniquement domain + auth.
//
// Toutes les fonctions sont thread-safe (le store l'est).
package capturecli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// CacheInvalidator est le type du callback d'invalidation du cache process
// des HaloTokens. En production, le caller passe `halo.InvalidateCachedPlayerTokens`.
// Dans les tests, un mock qui track les xuid invalidés.
//
// Nil-safe : si l'invalidator est nil, l'invalidation est skipped (utile pour
// les tests purs qui ne se soucient pas du cache).
type CacheInvalidator func(xuid string)

// ResolveXUIDByGamertag résout le xuid canonique pour un gamertag dans une
// liste de joueurs (typiquement issue de cfg.LoadPlayers). Match case-insensitive
// avec trim des espaces.
//
// Retourne une erreur explicite si :
//   - players nil ou vide
//   - gamertag vide après trim
//   - le joueur n'existe pas dans la liste
//   - le xuid est vide pour ce joueur (config incomplète)
//
// Le canonicalGT retourné est la forme exacte (avec casse) telle qu'enregistrée
// dans la liste — à utiliser pour les logs / écritures store.
func ResolveXUIDByGamertag(players []domain.PlayerSummary, gamertag string) (xuid, canonicalGT string, err error) {
	target := strings.ToLower(strings.TrimSpace(gamertag))
	if target == "" {
		return "", "", fmt.Errorf("capturecli: gamertag vide")
	}
	if len(players) == 0 {
		return "", "", fmt.Errorf("capturecli: liste de joueurs vide — vérifier db_profiles.json")
	}
	for _, p := range players {
		if strings.ToLower(p.Gamertag) == target {
			if p.XUID == "" {
				return "", "", fmt.Errorf("capturecli: joueur %q présent mais xuid manquant dans db_profiles.json", p.Gamertag)
			}
			return p.XUID, p.Gamertag, nil
		}
	}
	return "", "", fmt.Errorf("capturecli: joueur %q absent de db_profiles.json — ajouter une entrée avec xuid avant token-capture/token-import", gamertag)
}

// ParseRefreshTokenStdin lit un refresh_token depuis un reader (typiquement
// os.Stdin) et retourne la première valeur non-vide trouvée. Supporte :
//   - Format brut : une seule ligne avec le RT
//   - Format env-var : "SPNKR_OAUTH_REFRESH_TOKEN_X=value" → extrait la partie droite
//   - Ignore les lignes vides et les commentaires (#)
//
// Buffer max 4 MiB par ligne (sécurise contre les RT anormalement longs sans
// risque d'overflow). Erreur si :
//   - reader nil
//   - reader vide
//   - uniquement des lignes vides/commentaires
//   - scan échoue (I/O)
func ParseRefreshTokenStdin(r io.Reader) (string, error) {
	if r == nil {
		return "", fmt.Errorf("capturecli: reader nil")
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 && strings.HasPrefix(line, "SPNKR_OAUTH_REFRESH_TOKEN_") {
			line = strings.TrimSpace(line[idx+1:])
		}
		if line != "" {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("capturecli: scan stdin: %w", err)
	}
	return "", fmt.Errorf("capturecli: aucun refresh_token trouvé en stdin (lignes vides ou commentaires uniquement)")
}

// PersistRefreshToken écrit un refresh_token dans le MultiUserTokenStore en
// préservant les autres champs de l'entrée, complète le gamertag si la nouvelle
// entrée n'en avait pas, puis invalide le cache process des HaloTokens pour
// ce xuid (force re-acquire au prochain refresh).
//
// Opération atomique sur le store via UpdateOAuthRefreshToken (RMW protégé par
// mutex + atomic rename sur disque). L'invalidation du cache process est
// nécessaire car la nouvelle RT peut correspondre à une session Microsoft
// distincte — les Spartan tokens cachés (TTL 50 min) seraient stale (cf. ADR
// 0023 Phase 3bis).
//
// Erreur si :
//   - store nil
//   - refreshToken vide
//   - xuid unsafe (propagée depuis le store via xuidIsSafe)
//
// invalidator peut être nil — l'invalidation est alors skipped (cf. CacheInvalidator).
func PersistRefreshToken(store *auth.MultiUserTokenStore, xuid, gamertag, refreshToken string, invalidator CacheInvalidator) error {
	if store == nil {
		return fmt.Errorf("capturecli: store nil")
	}
	if refreshToken == "" {
		return fmt.Errorf("capturecli: refresh_token vide")
	}
	if err := store.UpdateOAuthRefreshToken(xuid, refreshToken); err != nil {
		return fmt.Errorf("capturecli: écriture RT: %w", err)
	}
	if gamertag != "" {
		if existing, err := store.Load(xuid); err == nil && existing != nil && existing.Gamertag == "" {
			existing.Gamertag = gamertag
			if uerr := store.Upsert(existing); uerr != nil {
				slog.Warn("capturecli: complétion gamertag échouée", "xuid", xuid, "err", uerr)
			}
		}
	}
	if invalidator != nil {
		invalidator(xuid)
	}
	return nil
}

// ResolveXUIDForRotation résout le xuid pour un gamertag dans le contexte du
// callback onRotated (Pool/Resolver). Priorité ADR 0023 :
//  1. Store via LoadByGamertag — l'entrée a été créée par Discovery ou la
//     migration boot-time. Plus rapide qu'un scan de la liste (O(log n) FS
//     vs O(n) linear scan).
//  2. players (depuis cfg.LoadPlayers fourni par le caller) — fallback pour
//     les cas exceptionnels (joueur ajouté post-boot sans token-capture, ou
//     store vide).
//
// Retourne "" si introuvable — le caller doit logger et skipper l'écriture
// store (le RT rotaté reste dans tous les cas écrit en DuckDB en compat).
//
// Le logging est best-effort via slog : pas de propagation d'erreur car le
// caller (onRotated) ne peut rien faire d'utile avec, et ne doit pas
// interrompre le pipeline OAuth.
func ResolveXUIDForRotation(ctx context.Context, store *auth.MultiUserTokenStore, players []domain.PlayerSummary, gamertag string) string {
	if store != nil {
		if user, err := store.LoadByGamertag(gamertag); err == nil && user != nil && user.XUID != "" {
			return user.XUID
		}
	}
	if len(players) == 0 {
		slog.DebugContext(ctx, "capturecli: liste joueurs vide (rotation xuid lookup)")
		return ""
	}
	target := strings.ToLower(strings.TrimSpace(gamertag))
	for _, p := range players {
		if strings.ToLower(p.Gamertag) == target {
			return p.XUID
		}
	}
	return ""
}
