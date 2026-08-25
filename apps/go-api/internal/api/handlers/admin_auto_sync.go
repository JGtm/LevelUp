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
// ne lit jamais directement le store. Si la source produit un access_token
// valide avec rotation, le RT rotaté est persisté via le callback onRotated
// (même mécanisme que le scheduler en production).
//
// Le POST /run est synchrone et peut prendre plusieurs minutes (1 cycle =
// N joueurs × appel API Halo + DB writes). Augmenter le timeout client
// (curl -m 600 recommandé).
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/scheduler"
)

// AdminAutoSyncHandler expose les endpoints diagnostic de l'auto-sync.
type AdminAutoSyncHandler struct {
	scheduler *scheduler.AutoSyncScheduler
	cfg       *config.AppConfig
	provider  auth.TokenProvider
}

// NewAdminAutoSyncHandler crée un handler.
// cfg permet à ProbeTokens de localiser la player DB.
// provider est utilisé par le Resolver instancié dans ProbeTokens.
//
// scheduler PEUT être nil : en MODE DÉMO (rendu du contrat / tests de contrat)
// aucun ordonnanceur n'est câblé, mais les routes sont tout de même montées pour
// que le contrat publié décrive la surface complète (V721-04). Les endpoints qui
// en dépendent répondent alors 503 scheduler_unavailable — même contrat d'erreur
// que POST /admin/actions/auto-sync/run. En production le wiring conditionnel de
// server_apiv1.go reste inchangé (scheduler non nil dès que les routes existent).
func NewAdminAutoSyncHandler(s *scheduler.AutoSyncScheduler, cfg *config.AppConfig, provider auth.TokenProvider) *AdminAutoSyncHandler {
	return &AdminAutoSyncHandler{scheduler: s, cfg: cfg, provider: provider}
}

// errSchedulerUnavailable : réponse des endpoints diag quand aucun ordonnanceur
// n'est câblé (mode démo). Aligné sur handleRunSyncCycle (admin_actions.go).
func errSchedulerUnavailable() error {
	return humacore.NewError(http.StatusServiceUnavailable, "scheduler_unavailable",
		"Scheduler auto-sync indisponible.")
}

// Mount enregistre les 3 routes via Huma sur le sous-routeur chi (préfixe
// /_diag/auto-sync + middleware LoopbackOnly hérités de server.go).
func (h *AdminAutoSyncHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/snapshot", h.handleGetSnapshot, humacore.Op("getDiagAutoSyncSnapshot", "Diag snapshot de l'ordonnanceur auto-sync", "diagnostics"))
	huma.Post(api, "/run", h.handleRunOnce, humacore.Op("postDiagAutoSyncRun", "Diag force un cycle auto-sync (loopback + admin)", "diagnostics"))
	huma.Get(api, "/probe", h.handleProbeTokens, humacore.Op("getDiagAutoSyncProbe", "Diag sonde la résolution des tokens auth d'un joueur (loopback + admin)", "diagnostics"))
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
	if h.scheduler == nil {
		return nil, errSchedulerUnavailable()
	}
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
	if h.scheduler == nil {
		return nil, errSchedulerUnavailable()
	}
	_ = h.scheduler.RunOnce(ctx)
	return &autoSyncSnapshotOutput{Body: h.scheduler.Snapshot()}, nil
}

// TokenProbeResult décrit l'état de la source de refresh_token d'un joueur,
// obtenu via la chaîne Discovery → Resolver (pas d'accès direct au store).
type TokenProbeResult struct {
	Gamertag string `json:"gamertag"`

	// Vrai si Discovery a trouvé une CredentialSource pour ce gamertag.
	DiscoveredInPool bool `json:"discovered_in_pool"`
	// Origine de la source : "watcher_oauth" (source unique ADR 0023).
	// Voir pool.CredentialSource.Source.
	Source string `json:"source,omitempty"`

	HasRefreshToken bool `json:"has_refresh_token"`
	RefreshTokenLen int  `json:"refresh_token_len,omitempty"`
	// S5 (sécurité, lot S) : SEUL le sha256 tronqué identifie le token. head/tail
	// (préfixe/suffixe en clair) supprimés — un diagnostic ne doit pas exposer de
	// fragment de secret, même sur une route loopback+admin.
	RefreshTokenSHA256 string `json:"refresh_token_sha256,omitempty"`

	// Résultat du Resolve (pipeline complet : OAuth refresh → Exchange Halo).
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
// Le RT rotaté par Microsoft (si refresh OAuth réussit) est persisté dans le
// MultiUserTokenStore, comme en production (ADR 0023).
//
// GET /api/v1/_diag/auto-sync/probe?gamertag=JGtm
func (h *AdminAutoSyncHandler) handleProbeTokens(ctx context.Context, in *autoSyncProbeInput) (*autoSyncProbeOutput, error) {
	gamertag := in.Gamertag
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_gamertag", "query param gamertag requis")
	}

	res := TokenProbeResult{Gamertag: gamertag}

	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	authStoreForProbe := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	discovery := pool.NewDiscoveryWithStore(h.cfg, pr, titlePkg.DefaultSlug, authStoreForProbe)
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
		// Joueur non découvert : aucun refresh token dans le MultiUserTokenStore.
		return &autoSyncProbeOutput{Body: res}, nil
	}

	res.DiscoveredInPool = true
	res.Source = src.Source
	res.HasRefreshToken = src.RefreshToken != ""
	res.RefreshTokenLen = len(src.RefreshToken)
	res.RefreshTokenSHA256 = fingerprintToken(src.RefreshToken)

	// Tenter le Resolve complet (OAuth refresh → Exchange) avec le même callback
	// onRotated qu'en production (ADR 0023) : écriture MultiUserTokenStore, source
	// unique (le double-write sync_meta a été retiré en Phase 5).
	var rotated bool
	onRotated := func(ctx context.Context, gt, newRT string) error {
		user, lerr := authStoreForProbe.LoadByGamertag(gt)
		if lerr != nil || user == nil || user.XUID == "" {
			return fmt.Errorf("probe: xuid introuvable pour %s: %w", gt, lerr)
		}
		if werr := authStoreForProbe.UpdateOAuthRefreshToken(user.XUID, newRT); werr != nil {
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
