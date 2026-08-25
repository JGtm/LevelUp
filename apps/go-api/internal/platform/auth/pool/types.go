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

	"golang.org/x/time/rate"

	"levelup/go-api/internal/domain"
)

// CredentialSource décrit une source de credentials (refresh token OAuth v2).
// Obtenue par scan du MultiUserTokenStore, sans aucune validation réseau.
type CredentialSource struct {
	Gamertag     string // "Bob", "Alice", etc.
	TitleSlug    string // "halo_infinite" — titre propriétaire du token (Phase 1.6 : clé pool (titleSlug,gamertag))
	XUID         string // "1234567890", numérique sans "xuid()"
	PlayerDBPath string // data/titles/halo_infinite/players/Bob/stats.duckdb (pour logs/debuggage)
	RefreshToken string // refresh token OAuth v2 (MultiUserTokenStore), "" si absent
	Source       string // "watcher_oauth" (source unique ADR 0023) — pour logs
}

// Discovery scanne les sources de credentials disponibles.
// Aucune validation réseau — juste le scan du MultiUserTokenStore pour découvrir
// quels joueurs ont un refresh token.
type Discovery interface {
	// Scan retourne la liste des CredentialSource découvertes.
	// Exclut automatiquement les joueurs sans refresh token.
	Scan(ctx context.Context) ([]CredentialSource, error)
}

// ResolvedTokens encapsule les tokens Halo frais échangés + métadonnées.
type ResolvedTokens struct {
	Gamertag  string             // gamertag du compte propriétaire du token
	XUID      string             // xuid numérique
	Tokens    *domain.HaloTokens // Spartan + Clearance
	ExpiresAt time.Time          // expiration estimée du Spartan token (best-effort ~4h)
	Source    string             // "watcher_oauth" (source unique ADR 0023)
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

// TokenRotationCallback est invoqué par le Resolver chaque fois qu'un refresh
// OAuth v2 retourne un refresh_token rotaté par Microsoft. Le caller doit
// persister ce nouveau RT pour qu'il soit utilisé au prochain refresh
// (sinon Microsoft refusera avec invalid_grant).
//
// Le callback est exécuté en best-effort : une erreur n'interrompt pas le
// Resolve (les tokens Halo sont déjà obtenus), mais elle est loguée par le
// Resolver. Le caller doit éviter les opérations bloquantes longues —
// idéalement < 1 s.
type TokenRotationCallback func(ctx context.Context, gamertag, newRefreshToken string) error

// ReauthCallback est invoqué par le Resolver quand l'état « ré-authentification
// requise » d'un joueur change : required=true lorsque le refresh_token est mort
// (credentials présents mais refresh définitivement KO), required=false lors d'un
// refresh réussi. Le caller persiste l'état (MultiUserTokenStore) et notifie
// (bannière in-app + Discord opt-in, PR-B). Best-effort, non bloquant.
type ReauthCallback func(ctx context.Context, gamertag, xuid string, required bool)

// AuthErrorCallback est invoqué par le Resolver quand un échec OAuth PERMANENT
// (classe "config" ou "revoked") est observé pour un joueur — class et msg sont
// alors non vides — puis avec class=="" lors d'un refresh réussi (effacement).
// Le caller persiste l'état (dashboard admin « Santé des tokens »).
// Best-effort, non bloquant. msg ne contient jamais de token/secret.
type AuthErrorCallback func(ctx context.Context, gamertag, xuid, class, msg string)

// ResolverCallbacks regroupe les callbacks optionnels du Resolver (tous nullables).
type ResolverCallbacks struct {
	OnRotated   TokenRotationCallback
	OnReauth    ReauthCallback
	OnAuthError AuthErrorCallback
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
	Gamertag string        // gamertag du compte propriétaire du token
	Release  func()        // remet le slot dispo (sémaphore ou no-op selon la politique)
	Limiter  *rate.Limiter // rate.Limiter du slot (PerTokenRPS) — nil si le pool n'expose pas
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

	// HasPlayer retourne true si un slot est disponible (peu importe son état
	// healthy actuel) pour ce gamertag. Permet aux callers de skip
	// silencieusement les joueurs qui n'ont pas de token dans le pool, sans
	// passer par un Acquire/Release qui retournerait juste une erreur.
	HasPlayer(gamertag string) bool

	// MarkUnhealthy invalide un token (sur 401/403) et déclenche un Resolver.Refresh asynchrone.
	// Logs informatif (pas d'erreur — les appels concurrents tolèrent l'exclusion temporaire).
	// Après un delai GlobalCooldown, l'appel Refresh() remettra le token en circulation (si succès).
	MarkUnhealthy(gamertag string, reason error)

	// OnHTTPError signale une erreur HTTP GLOBALE (503, ou 429 sans token
	// identifiable) et déclenche un cooldown global : marque tous les tokens
	// malsains et suspend le refresher. À réserver aux signaux SERVEUR (503) ou
	// aux 429 non imputables à un token précis — sinon préférer On429ForToken.
	// retryAfter : durée demandée par le header HTTP Retry-After (0 = absent →
	// backoff exponentiel planché à globalCooldown). Non-bloquant.
	OnHTTPError(statusCode int, retryAfter time.Duration)

	// On429ForToken signale un rate-limit (429) imputable à UN token précis
	// (gamertag du lease fautif). Met CE token en cooldown temporel borné et le
	// skippe à l'acquisition, SANS cooldown global ni re-exchange (le token reste
	// valide). Les autres tokens continuent de servir. gamertag vide → filet
	// global (OnHTTPError). C'est la voie normale d'un 429 en round-robin.
	On429ForToken(gamertag string, retryAfter time.Duration)

	// AddOrUpdateSource (E.v2, 2026-05-24) — hot-add ou refresh d'un slot par
	// gamertag. Si le gamertag existe déjà dans le pool : re-Resolve et update
	// le slot (réutilise rate limiter + index round-robin). Sinon : append un
	// nouveau slot.
	//
	// LIMITATION : nouveau slot ajouté APRÈS le boot est seulement reachable
	// via PolicyPinnedPlayer (canal round-robin sized at boot, capacité non
	// extensible sans drain/refill). Pour LevelUp ce n'est pas un problème car
	// auto-sync utilise PolicyPinnedPlayer (1 token par joueur).
	//
	// Cas d'usage : periodic re-scan Discovery.Scan() détecte un nouveau token
	// (env var ou watcher_tokens.json mis à jour) → push dans le pool sans
	// reboot du serveur.
	//
	// Erreurs : resolver.Resolve échoué OU MaxSize atteint si nouveau slot.
	AddOrUpdateSource(ctx context.Context, src CredentialSource) error

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
