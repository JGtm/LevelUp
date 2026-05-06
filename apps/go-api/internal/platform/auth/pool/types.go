// Package pool — gestion d'un pool de tokens Halo Infinite pour parallélisation du sync.
//
// Architecture en 4 couches :
// A. Discovery — scanne env + DuckDB pour détecter les sources de credentials
// B. Resolver — échange CredentialSource → HaloTokens frais (cache TTL ~3h30)
// C. Pool — maintient N tokens vivants, deux politiques d'acquisition (public + pinned)
// D. Client adapter — PooledHaloClient implémente sync.HaloClient dans sync/pooled_client.go
package pool

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
)

// CredentialSource décrit une source de credentials (MSAL cache ou refresh token).
// Obtenue par scan env + DuckDB, sans aucune validation réseau.
type CredentialSource struct {
	Gamertag     string // "Bob", "Alice", etc.
	XUID         string // "1234567890", numérique sans "xuid()"
	PlayerDBPath string // data/titles/halo_infinite/players/Bob/stats.duckdb (pour logs/debuggage)
	MSALCache    string // JSON sérialisé du cache MSAL (sync_meta.msal_token_cache), "" si absent
	RefreshToken string // refresh token OAuth v2 (sync_meta.oauth_refresh_token ou env), "" si absent
	Source       string // "duckdb_msal" | "duckdb_oauth" | "env_oauth" — pour logs
}

// Discovery scanne les sources de credentials disponibles.
// Aucune validation réseau — juste le scan env + DuckDB pour découvrir quels joueurs ont un token.
type Discovery interface {
	// Scan retourne la liste des CredentialSource découvertes.
	// Exclut automatiquement les joueurs sans MSAL cache ET sans refresh token.
	Scan(ctx context.Context) ([]CredentialSource, error)
}

// ResolvedTokens encapsule les tokens Halo frais échangés + métadonnées.
type ResolvedTokens struct {
	Gamertag  string             // gamertag du compte propriétaire du token
	XUID      string             // xuid numérique
	Tokens    *domain.HaloTokens // Spartan + Clearance
	ExpiresAt time.Time          // expiration estimée du Spartan token (best-effort ~4h)
	Source    string             // "duckdb_msal" | "duckdb_oauth" | "env_oauth"
}

// Resolver échange CredentialSource → ResolvedTokens frais.
// Mémoïse les tokens pendant leur durée de vie (~3h30).
type Resolver interface {
	// Resolve échange une CredentialSource en tokens Halo frais.
	// Cache internalisé — appels répétés pour le même gamertag rendent le cached résultat jusqu'à expiration.
	Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error)

	// Refresh force un re-échange du token pour un gamertag donné (par ex après un 401/403).
	// Retourne ErrNoCredentialSource si le gamertag n'a jamais été scanné ou est introuvable.
	Refresh(ctx context.Context, gamertag string) (*ResolvedTokens, error)
}

// AcquirePolicy détermine comment le pool sélectionne un token.
type AcquirePolicy int

const (
	// PolicyAnyPublic : round-robin parmi tous les tokens actifs.
	// Utilisé pour les endpoints publics (GetMatchHistory, GetMatchStats, etc.)
	// qui acceptent n'importe quel gamertag/matchID en paramètre.
	PolicyAnyPublic AcquirePolicy = iota

	// PolicyPinnedPlayer : token du propriétaire spécifique UNIQUEMENT.
	// Utilisé pour les endpoints privacy-gated (GetCareerRank, BattlePass, Challenges)
	// qui ne retournent des données que si l'XUID du token correspond au cible.
	// Retourne ErrNoTokenForPlayer si le gamertag n'a pas de token.
	PolicyPinnedPlayer
)

// Lease encapsule un token acquis du pool + son slot parent.
// Appellé (sous la forme de Lease) doit invoquer Release() après utilisation.
type Lease struct {
	Tokens   *domain.HaloTokens
	Gamertag string // gamertag du compte propriétaire du token
	Release  func() // remet le slot dispo (sémaphore ou no-op selon la politique)
}

// Pool gère N tokens vivants avec deux modes d'acquisition.
type Pool interface {
	// Acquire retourne un Lease (token + Release callback).
	//
	// Paramètres :
	//   ctx : contexte d'annulation
	//   policy : PolicyAnyPublic (round-robin) ou PolicyPinnedPlayer (lookup par gamertag)
	//   pinnedGamertag : si policy == PolicyPinnedPlayer, le gamertag cible (obligatoire)
	//
	// Retourne une erreur si policy == PolicyPinnedPlayer et le gamertag n'existe pas.
	Acquire(ctx context.Context, policy AcquirePolicy, pinnedGamertag string) (*Lease, error)

	// Size retourne le nombre de tokens actifs dans le pool.
	Size() int

	// MarkUnhealthy invalide un token (sur 401/403) et déclenche un Resolver.Refresh asynchrone.
	// Logs informatif (pas d'erreur — les appels concurrents tolèrent l'exclusion temporaire).
	// Après un delai GlobalCooldown, l'appel Refresh() remettra le token en circulation (si succès).
	MarkUnhealthy(gamertag string, reason error)

	// OnHTTPError signale une erreur HTTP (429/503) et déclenche un cooldown global.
	// Marque tous les tokens comme malsains et suspend le refresher pour GlobalCooldown.
	// Non-bloquant : ignores les autres codes d'erreur.
	OnHTTPError(statusCode int)

	// Close stoppe les goroutines background (refresher, etc.) et libère les ressources.
	Close()
}

// PoolOptions configure le comportement du pool.
type PoolOptions struct {
	// MaxSize limite le nombre de tokens dans le pool (0 = tous les sources découverts).
	MaxSize int

	// PerTokenRPS est le nombre de requêtes par seconde **par token**.
	// RPS effectif total = PerTokenRPS × Size().
	// Défaut : 1 RPS/token.
	PerTokenRPS int

	// RefreshInterval est l'intervalle avant expiration du Spartan token auquel on re-échange.
	// Défaut : 3h30 (Spartan ~4h).
	RefreshInterval time.Duration

	// GlobalCooldown est le délai d'attente après un 429/503 global (tout le pool).
	// Après ce délai, les appels Refresh() redémarrent. Défaut : 30s.
	GlobalCooldown time.Duration
}
