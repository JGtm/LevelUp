// Package handlers — admin_auto_sync.go : endpoints diagnostic du scheduler
// d'auto-sync. Permet de voir le résultat détaillé par joueur du dernier cycle
// (raison du skip/failure, compteurs, erreurs) sans avoir accès aux logs
// serveur, et de forcer un cycle on-demand pour reproduire un bug.
//
// Routes (montées sous /api/v1/_diag/auto-sync/, loopback only) :
//   - GET  /snapshot         : retourne le dernier état mémorisé du scheduler
//   - POST /run              : force un cycle synchrone et retourne le snapshot
//   - GET  /probe?gamertag=X : teste pour un joueur la chaîne Discovery→Resolver
//
// Le probe passe par les abstractions pool.Discovery + pool.Resolver, donc
// ne lit jamais directement os.Getenv ni sync_meta. Si une source produit un
// access_token valide avec rotation, le RT rotaté est persisté via le
// callback onRotated (même mécanisme que le scheduler en production).
//
// Le POST /run est synchrone et peut prendre plusieurs minutes (1 cycle =
// N joueurs × appel API Halo + DB writes). Augmenter le timeout client
// (curl -m 600 recommandé).
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// AdminAutoSyncHandler expose les endpoints diagnostic de l'auto-sync.
type AdminAutoSyncHandler struct {
	scheduler *scheduler.AutoSyncScheduler
	cfg       *config.AppConfig
	provider  auth.TokenProvider
}

// NewAdminAutoSyncHandler crée un handler. scheduler doit être non nil
// (le caller dans server.go garde le wiring conditionnel).
// cfg permet à ProbeTokens de localiser la player DB.
// provider est utilisé par le Resolver instancié dans ProbeTokens.
func NewAdminAutoSyncHandler(s *scheduler.AutoSyncScheduler, cfg *config.AppConfig, provider auth.TokenProvider) *AdminAutoSyncHandler {
	return &AdminAutoSyncHandler{scheduler: s, cfg: cfg, provider: provider}
}

// GetSnapshot retourne le snapshot mémorisé du dernier cycle.
// GET /api/v1/_diag/auto-sync/snapshot
func (h *AdminAutoSyncHandler) GetSnapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.scheduler.Snapshot())
}

// RunOnce force un cycle synchrone et retourne le snapshot mis à jour.
// POST /api/v1/_diag/auto-sync/run
//
// Bloquant : peut prendre plusieurs minutes (N joueurs * appel API Halo +
// DB writes). Le scheduler est thread-safe, ce force-run n'interfère pas
// avec les cycles automatiques (ils ne tournent pas en parallèle car
// `Run()` séquence ses ticks via un ticker).
func (h *AdminAutoSyncHandler) RunOnce(w http.ResponseWriter, r *http.Request) {
	_ = h.scheduler.RunOnce(r.Context())
	writeJSON(w, http.StatusOK, h.scheduler.Snapshot())
}

// TokenProbeResult décrit l'état des sources de refresh_token pour un joueur,
// obtenu via la chaîne Discovery → Resolver (pas d'accès direct à os.Getenv
// ou sync_meta).
type TokenProbeResult struct {
	Gamertag string `json:"gamertag"`

	// Vrai si Discovery a trouvé une CredentialSource pour ce gamertag.
	DiscoveredInPool bool `json:"discovered_in_pool"`
	// Origine de la source : "duckdb_msal", "duckdb_oauth", "env_oauth", etc.
	// Voir pool.CredentialSource.Source.
	Source string `json:"source,omitempty"`

	HasMSALCache       bool   `json:"has_msal_cache"`
	HasRefreshToken    bool   `json:"has_refresh_token"`
	RefreshTokenLen    int    `json:"refresh_token_len,omitempty"`
	RefreshTokenSHA256 string `json:"refresh_token_sha256,omitempty"`
	RefreshTokenHead   string `json:"refresh_token_head,omitempty"`
	RefreshTokenTail   string `json:"refresh_token_tail,omitempty"`

	// Résultat du Resolve (pipeline complet : MSAL/OAuth → Exchange Halo).
	ResolveOK              bool   `json:"resolve_ok"`
	ResolveError           string `json:"resolve_error,omitempty"`
	SpartanTokenLen        int    `json:"spartan_token_len,omitempty"`
	RefreshTokenWasRotated bool   `json:"refresh_token_was_rotated"`
}

// fingerprintToken retourne sha256 + head/tail tronqués pour identifier un
// token sans révéler sa valeur (utile pour comparer plusieurs lectures, ex :
// vérifier que `.env.local` n'a pas été modifié sans redémarrage du serveur).
func fingerprintToken(s string) (sha string, head string, tail string) {
	if s == "" {
		return "", "", ""
	}
	sum := sha256.Sum256([]byte(s))
	sha = hex.EncodeToString(sum[:8])
	if len(s) >= 6 {
		head = s[:6]
		tail = s[len(s)-6:]
	} else {
		head = s
		tail = s
	}
	return
}

// ProbeTokens diagnostic complet pour un joueur via Discovery + Resolver.
// Le RT rotaté par Microsoft (si refresh OAuth réussit) est persisté dans
// sync_meta.oauth_refresh_token de la player DB, comme en production.
//
// GET /api/v1/_diag/auto-sync/probe?gamertag=JGtm
func (h *AdminAutoSyncHandler) ProbeTokens(w http.ResponseWriter, r *http.Request) {
	gamertag := r.URL.Query().Get("gamertag")
	if gamertag == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_gamertag", "query param gamertag requis")
		return
	}

	res := TokenProbeResult{Gamertag: gamertag}

	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	discovery := pool.NewDiscovery(h.cfg, pr, titlePkg.DefaultSlug)
	sources, err := discovery.Scan(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "discovery_scan_failed", err.Error())
		return
	}

	// Chercher la source correspondant à ce gamertag.
	var src *pool.CredentialSource
	for i := range sources {
		if sources[i].Gamertag == gamertag {
			src = &sources[i]
			break
		}
	}
	if src == nil {
		// Joueur non découvert : pas de credential utilisable (ni env var ni sync_meta).
		writeJSON(w, http.StatusOK, res)
		return
	}

	res.DiscoveredInPool = true
	res.Source = src.Source
	res.HasMSALCache = src.MSALCache != ""
	res.HasRefreshToken = src.RefreshToken != ""
	res.RefreshTokenLen = len(src.RefreshToken)
	res.RefreshTokenSHA256, res.RefreshTokenHead, res.RefreshTokenTail = fingerprintToken(src.RefreshToken)

	// Tenter le Resolve complet (pipeline MSAL→OAuth→Exchange) avec le même
	// callback onRotated qu'en production : si Microsoft rotate le RT, il est
	// persisté dans sync_meta pour que le scheduler l'utilise ensuite.
	var rotated bool
	onRotated := func(ctx context.Context, gt, newRT string) error {
		dbPath := pr.PlayerDBPath(titlePkg.DefaultSlug, gt)
		db, derr := duckdb.OpenReadWriteShared(dbPath)
		if derr != nil {
			return derr
		}
		defer db.Close() //nolint:errcheck // ref-count : best-effort
		if werr := duckdb.WriteOAuthRefreshToken(ctx, db, newRT); werr != nil {
			return werr
		}
		rotated = true
		return nil
	}

	resolver := pool.NewResolver(h.provider, 0, onRotated)
	resolved, rerr := resolver.Resolve(r.Context(), *src)
	if rerr != nil {
		res.ResolveError = rerr.Error()
	} else if resolved != nil && resolved.Tokens != nil {
		res.ResolveOK = true
		res.SpartanTokenLen = len(resolved.Tokens.SpartanToken)
	}
	res.RefreshTokenWasRotated = rotated

	writeJSON(w, http.StatusOK, res)
}
