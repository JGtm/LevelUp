// Package auth — migration.go : import boot-time des tokens legacy vers MultiUserTokenStore.
//
// Refactor token storage (ADR 0023) : MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json)
// devient la source autoritaire des tokens auth (refresh_token OAuth + MSAL cache). Cette
// migration s'exécute au boot du serveur, AVANT pool.Discovery, pour seeder le store depuis
// les sources legacy si elles ne sont pas encore migrées.
//
// Sources legacy migrées :
//   - env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> → store.OAuthRefreshToken
//   - sync_meta.oauth_refresh_token (DuckDB)       → store.OAuthRefreshToken
//
// Priorité quand plusieurs sources ont une valeur pour le même xuid :
//  1. DuckDB sync_meta (probablement le RT rotaté maintenu par le Pool)
//  2. env var (seed bootstrap initial, peut être stale)
//
// Idempotence : si le store contient déjà un OAuthRefreshToken pour un xuid, on ne
// touche PAS — le store est autoritaire (peut contenir un RT plus récent qu'aucune
// source legacy ne connaît).
//
// Le package auth ne dépend pas de DuckDB : le caller (cmd/server/main.go) fournit
// les valeurs déjà lues via le callback `LegacySourcesReader`.
//
// KILL-SWITCH DATÉ (CLAUDE.md règle 11) — EXCEPTION UNIQUE de l'ADR 0023 Phase 5 :
// cette migration one-shot est le SEUL chemin de code encore autorisé à lire une
// source de credentials legacy (env var + sync_meta). Tous les autres lecteurs ont
// été supprimés le 2026-08-25 (bascule du défaut : plus aucun fallback runtime).
//   - Date cible de retrait : 2026-10-01.
//   - Critère mesurable : 0 token migré au boot sur 30 jours de logs prod, soit
//     `auth_migration: scan terminé` avec rt_migrated=0 à chaque boot (et aucun
//     `auth_migration: RT migré vers store`). Vérifier avant de retirer :
//     `grep 'auth_migration: RT migré' sync.log`.
//
// Au retrait : supprimer ce fichier, son test, le wiring boot
// (cmd/server/main.go migrateLegacyAuthTokensAtBoot + legacyAuthSourcesReader) et
// les derniers helpers DuckDB de queries_auth.go.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// LegacyPlayer est une vue minimale d'un joueur pour la migration.
// Fournit uniquement les champs nécessaires à l'identification et à la
// résolution du chemin DuckDB legacy. Évite une dépendance circulaire vers
// le package config / domain.
type LegacyPlayer struct {
	XUID     string
	Gamertag string
	// PlayerDBPath : chemin absolu vers stats.duckdb du joueur (pour lecture
	// legacy sync_meta). Vide si la DB n'existe pas — la migration skip alors
	// la source DuckDB et tente uniquement l'env var.
	PlayerDBPath string
}

// LegacySources regroupe les valeurs legacy lues pour un joueur donné.
// Toutes les valeurs sont des strings vides quand absentes.
type LegacySources struct {
	EnvRT    string // SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_UPPER>
	DuckDBRT string // sync_meta WHERE key='oauth_refresh_token'
}

// LegacySourcesReader lit les sources legacy pour un joueur. Le caller production
// branche cette fonction sur env var + DuckDB. Les tests fournissent un mock.
//
// Retourne LegacySources{} (vide) si rien n'est lu, error seulement pour les
// échecs structurels (DB illisible) — l'absence de valeur n'est pas une erreur.
type LegacySourcesReader func(ctx context.Context, player LegacyPlayer) (LegacySources, error)

// MigrationStats résume ce qui a été migré pendant un appel à
// MigrateLegacyTokens. Utilisé pour le logging et les tests.
type MigrationStats struct {
	PlayersScanned  int // total players inspectés
	PlayersSkipped  int // store déjà rempli, rien à faire
	OAuthRTMigrated int // RT copié depuis legacy (env ou DuckDB)
	Errors          int // erreurs individuelles non bloquantes
}

// EnvRefreshTokenForGamertag retourne SPNKR_OAUTH_REFRESH_TOKEN_<KEY> où KEY est
// le gamertag normalisé : majuscules, ' '/'-'/'.' → '_'. Compatible avec la
// convention historique partagée entre registry.go, watcher_refresh.go,
// auto_sync.go et pool/discovery.go.
//
// Exposée pour permettre au caller production de fournir un LegacySourcesReader
// qui lit l'env var sans dupliquer la logique de normalisation.
func EnvRefreshTokenForGamertag(gamertag string) string {
	if gamertag == "" {
		return ""
	}
	key := strings.ToUpper(strings.TrimSpace(gamertag))
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}

// MigrateLegacyTokens copie les refresh tokens legacy (env var + sync_meta
// DuckDB) dans le MultiUserTokenStore pour les joueurs qui n'y ont pas encore
// d'entrée.
//
// Idempotent : peut être appelé à chaque boot sans corrompre le store. Best-effort
// par joueur (une erreur sur un joueur n'interrompt pas les autres).
//
// Le store est autoritaire : si un xuid a déjà OAuthRefreshToken défini dans le
// store, la migration ne l'écrase pas, même si une source legacy a une valeur
// différente.
func MigrateLegacyTokens(
	ctx context.Context,
	store *MultiUserTokenStore,
	players []LegacyPlayer,
	reader LegacySourcesReader,
) (MigrationStats, error) {
	if store == nil {
		return MigrationStats{}, fmt.Errorf("auth_migration: store nil")
	}
	if reader == nil {
		return MigrationStats{}, fmt.Errorf("auth_migration: reader nil")
	}

	stats := MigrationStats{}

	for _, p := range players {
		stats.PlayersScanned++

		if !xuidIsSafe(p.XUID) {
			slog.WarnContext(ctx, "auth_migration: xuid invalide, joueur ignoré",
				"gamertag", p.Gamertag, "xuid", p.XUID)
			stats.Errors++
			continue
		}

		existing, err := store.Load(p.XUID)
		if err != nil && !errors.Is(err, ErrUserTokensNotFound) {
			slog.WarnContext(ctx, "auth_migration: lecture store échouée, joueur ignoré",
				"xuid", p.XUID, "gamertag", p.Gamertag, "err", err)
			stats.Errors++
			continue
		}

		if existing != nil && existing.OAuthRefreshToken != "" {
			stats.PlayersSkipped++
			continue
		}

		sources, err := reader(ctx, p)
		if err != nil {
			slog.WarnContext(ctx, "auth_migration: lecture sources legacy échouée",
				"xuid", p.XUID, "gamertag", p.Gamertag, "err", err)
			stats.Errors++
			continue
		}

		if rt := pickRT(sources); rt != "" {
			if err := store.UpdateOAuthRefreshToken(p.XUID, rt); err != nil {
				slog.WarnContext(ctx, "auth_migration: écriture RT échouée",
					"xuid", p.XUID, "gamertag", p.Gamertag, "err", err)
				stats.Errors++
			} else {
				stats.OAuthRTMigrated++
				slog.InfoContext(ctx, "auth_migration: RT migré vers store",
					"xuid", p.XUID, "gamertag", p.Gamertag, "source", rtSource(sources))
			}
		}

		// Compléter le gamertag dans le store si la nouvelle entrée vient d'être créée
		// par UpdateXxx (XUID seul). Permet à LoadByGamertag de fonctionner.
		if existing == nil && p.Gamertag != "" {
			if fresh, err := store.Load(p.XUID); err == nil && fresh.Gamertag == "" {
				fresh.Gamertag = p.Gamertag
				if err := store.Upsert(fresh); err != nil {
					slog.WarnContext(ctx, "auth_migration: complétion gamertag échouée",
						"xuid", p.XUID, "err", err)
				}
			}
		}
	}

	slog.InfoContext(ctx, "auth_migration: scan terminé",
		"players_scanned", stats.PlayersScanned,
		"players_skipped", stats.PlayersSkipped,
		"rt_migrated", stats.OAuthRTMigrated,
		"errors", stats.Errors)

	return stats, nil
}

// pickRT choisit le RT à migrer entre DuckDB et env var.
// DuckDB est prioritaire : c'est le RT rotaté maintenu par le Pool, le plus
// récent. L'env var est un seed bootstrap qui peut être stale après rotation.
func pickRT(sources LegacySources) string {
	if sources.DuckDBRT != "" {
		return sources.DuckDBRT
	}
	return sources.EnvRT
}

// rtSource retourne le label de la source effectivement utilisée par pickRT.
// Pour le logging structuré.
func rtSource(sources LegacySources) string {
	if sources.DuckDBRT != "" {
		return "duckdb"
	}
	if sources.EnvRT != "" {
		return "env_var"
	}
	return "none"
}
