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
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /_diag/auto-sync (middleware LoopbackOnly hérité) et enregistre les 3 routes
// via huma.*. Logique métier inchangée (scheduler + pool Discovery/Resolver),
// seul le wrapping HTTP change. Les chemins relatifs sont identiques aux routes
// chi d'origine (montées sous /_diag/auto-sync par server.go).
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
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/sync"
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

// Mount enregistre les 3 routes via Huma sur le sous-routeur chi (préfixe
// /_diag/auto-sync + middleware LoopbackOnly hérités de server.go).
func (h *AdminAutoSyncHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/snapshot", h.handleGetSnapshot)
	huma.Post(api, "/run", h.handleRunOnce)
	huma.Get(api, "/probe", h.handleProbeTokens)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// autoSyncProbeInput : ?gamertag= (toléré vide → 400 missing_gamertag, parse maison).
type autoSyncProbeInput struct {
	Gamertag string `query:"gamertag"`
}

type autoSyncSnapshotOutput struct {
	Body scheduler.SchedulerSnapshot
}
type autoSyncProbeOutput struct{ Body TokenProbeResult }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetSnapshot retourne le snapshot mémorisé du dernier cycle.
// GET /api/v1/_diag/auto-sync/snapshot
func (h *AdminAutoSyncHandler) handleGetSnapshot(_ context.Context, _ *struct{}) (*autoSyncSnapshotOutput, error) {
	return &autoSyncSnapshotOutput{Body: h.scheduler.Snapshot()}, nil
}

// handleRunOnce force un cycle synchrone et retourne le snapshot mis à jour.
// POST /api/v1/_diag/auto-sync/run
//
// Bloquant : peut prendre plusieurs minutes (N joueurs * appel API Halo +
// DB writes). Le scheduler est thread-safe, ce force-run n'interfère pas
// avec les cycles automatiques (ils ne tournent pas en parallèle car
// `Run()` séquence ses ticks via un ticker).
func (h *AdminAutoSyncHandler) handleRunOnce(ctx context.Context, _ *struct{}) (*autoSyncSnapshotOutput, error) {
	_ = h.scheduler.RunOnce(ctx)
	return &autoSyncSnapshotOutput{Body: h.scheduler.Snapshot()}, nil
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

	HasMSALCache    bool `json:"has_msal_cache"`
	HasRefreshToken bool `json:"has_refresh_token"`
	RefreshTokenLen int  `json:"refresh_token_len,omitempty"`
	// S5 (sécurité, lot S) : SEUL le sha256 tronqué identifie le token. head/tail
	// (préfixe/suffixe en clair) supprimés — un diagnostic ne doit pas exposer de
	// fragment de secret, même sur une route loopback+admin.
	RefreshTokenSHA256 string `json:"refresh_token_sha256,omitempty"`

	// Résultat du Resolve (pipeline complet : MSAL/OAuth → Exchange Halo).
	ResolveOK              bool   `json:"resolve_ok"`
	ResolveError           string `json:"resolve_error,omitempty"`
	SpartanTokenLen        int    `json:"spartan_token_len,omitempty"`
	RefreshTokenWasRotated bool   `json:"refresh_token_was_rotated"`
}

// fingerprintToken retourne un sha256 tronqué identifiant un token sans révéler
// sa valeur (utile pour comparer plusieurs lectures, ex : vérifier que `.env.local`
// n'a pas été modifié sans redémarrage du serveur). S5 / audit M3 (lot S) :
// head/tail retirés — même tronqués, ils exposaient un fragment du secret.
func fingerprintToken(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// handleProbeTokens diagnostic complet pour un joueur via Discovery + Resolver.
// Le RT rotaté par Microsoft (si refresh OAuth réussit) est persisté dans
// sync_meta.oauth_refresh_token de la player DB, comme en production.
//
// GET /api/v1/_diag/auto-sync/probe?gamertag=JGtm
func (h *AdminAutoSyncHandler) handleProbeTokens(ctx context.Context, in *autoSyncProbeInput) (*autoSyncProbeOutput, error) {
	gamertag := in.Gamertag
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_gamertag", "query param gamertag requis")
	}

	res := TokenProbeResult{Gamertag: gamertag}

	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	discovery := pool.NewDiscovery(h.cfg, pr, titlePkg.DefaultSlug)
	sources, err := discovery.Scan(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "discovery_scan_failed", err.Error())
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
		return &autoSyncProbeOutput{Body: res}, nil
	}

	res.DiscoveredInPool = true
	res.Source = src.Source
	res.HasMSALCache = src.MSALCache != ""
	res.HasRefreshToken = src.RefreshToken != ""
	res.RefreshTokenLen = len(src.RefreshToken)
	res.RefreshTokenSHA256 = fingerprintToken(src.RefreshToken)

	// Tenter le Resolve complet (pipeline MSAL→OAuth→Exchange) avec le même
	// callback onRotated qu'en production (ADR 0023) :
	//  1. PRIORITÉ — écriture MultiUserTokenStore (source canonique)
	//  2. Compat — écriture aussi sync_meta DuckDB (legacy, retiré Phase 5)
	authStoreForProbe := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	var rotated bool
	onRotated := func(ctx context.Context, gt, newRT string) error {
		if user, lerr := authStoreForProbe.LoadByGamertag(gt); lerr == nil && user != nil && user.XUID != "" {
			if werr := authStoreForProbe.UpdateOAuthRefreshToken(user.XUID, newRT); werr != nil {
				slog.WarnContext(ctx, "probe: store write échoué", "gamertag", gt, "err", werr)
			}
		}
		dbPath := pr.PlayerDBPath(titlePkg.DefaultSlug, gt)
		db, release, derr := sync.AcquirePlayerWriterStandalone(ctx, dbPath)
		if derr != nil {
			return derr
		}
		defer release()
		if werr := duckdb.WriteOAuthRefreshToken(ctx, db, newRT); werr != nil {
			return werr
		}
		rotated = true
		return nil
	}

	resolver := pool.NewResolver(h.provider, 0, onRotated)
	resolved, rerr := resolver.Resolve(ctx, *src)
	if rerr != nil {
		res.ResolveError = rerr.Error()
	} else if resolved != nil && resolved.Tokens != nil {
		res.ResolveOK = true
		res.SpartanTokenLen = len(resolved.Tokens.SpartanToken)
	}
	res.RefreshTokenWasRotated = rotated

	return &autoSyncProbeOutput{Body: res}, nil
}
