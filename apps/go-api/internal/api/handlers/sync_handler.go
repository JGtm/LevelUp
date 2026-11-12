// Package handlers — sync_handler.go : lancement des syncs depuis l'API (sync initiale, sync delta d'un
// joueur ou de tous).
//
// POST /sync/initial, POST /sync/all et POST /players/{player_slug}/sync créent un job et répondent 202
// aussitôt. Le handler décode la requête, appelle port.SyncService (garde-fous, résolution du profil,
// moteur et lancement du job y vivent) et traduit ses refus typés (port.SyncJobError) en réponses HTTP.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// maxPlayerSlugLen borne la longueur d'un player_slug de corps de requête.
const maxPlayerSlugLen = 50

// SyncHandler expose les routes de synchronisation des données Halo.
type SyncHandler struct {
	svc port.SyncService
}

// NewSyncHandler crée un SyncHandler.
func NewSyncHandler(svc port.SyncService) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// ─── Montage Huma + types Input/Output ───────────────────────────────────────

// syncJobOutput : 202 Accepted, corps = snapshot du job de sync créé (les 3 routes partagent ce
// contrat). Le 202 est posé UNE fois par route via humacore.DefaultStatus au montage (source unique :
// runtime ET document).
type syncJobOutput struct {
	Body *domain.AsyncJobStatus
}

// syncInitialInput : corps brut décodé maison. RawBody (pas Body typé) → préserve le contrat 400
// {invalid_body} sur JSON invalide (un Body typé renverrait le 422 de validation Huma).
type syncInitialInput struct {
	RawBody []byte
}

// syncDeltaInput : path param {player_slug}.
type syncDeltaInput struct {
	PlayerSlug string `path:"player_slug"`
}

// MountInitialAndAll enregistre les routes admin /sync/initial et /sync/all via Huma sur le routeur
// chi fourni (middleware RequireAuth/RequireAdmin hérités du groupe admin où le routeur les monte).
func (h *SyncHandler) MountInitialAndAll(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/sync/initial", h.StartInitialSync, humacore.Op("postSyncInitial", "Lancer la synchronisation initiale", "sync"),
		humacore.DefaultStatus(http.StatusAccepted))
	huma.Post(api, "/sync/all", h.StartSyncAll, humacore.Op("postSyncAll", "Sync delta global (tous les joueurs configurés)", "sync"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// MountDelta enregistre la route /sync via Huma sur le sous-routeur chi (préfixe /players/{player_slug}
// + middleware hérités — lit {player_slug} parent).
func (h *SyncHandler) MountDelta(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/sync", h.StartDeltaSync, humacore.Op("postPlayerSync", "Sync delta par joueur", "sync"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// StartInitialSync lance la sync initiale d'un joueur.
// POST /sync/initial -> 202 AsyncJobStatus. Un refus d'instance (sync initiale désactivée) prime sur un
// corps invalide.
func (h *SyncHandler) StartInitialSync(ctx context.Context, in *syncInitialInput) (*syncJobOutput, error) {
	if err := h.svc.CheckInitialSyncEnabled(ctx); err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	var req domain.InitialSyncStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requete JSON invalide.")
	}
	if err := checkPlayerSlug(req.PlayerSlug); err != nil {
		return nil, err
	}
	job, err := h.svc.StartInitialSync(ctx, syncCaller(ctx), req)
	if err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	return &syncJobOutput{Body: job}, nil
}

// StartDeltaSync lance une synchronisation delta pour un joueur donné.
// POST /api/v1/players/{player_slug}/sync → 202 AsyncJobStatus.
// Contrairement à StartInitialSync, cette route n'est pas protégée par can_start_initial_sync.
func (h *SyncHandler) StartDeltaSync(ctx context.Context, in *syncDeltaInput) (*syncJobOutput, error) {
	if in.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug manquant.")
	}
	job, err := h.svc.StartDeltaSync(ctx, syncCaller(ctx), in.PlayerSlug)
	if err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	return &syncJobOutput{Body: job}, nil
}

// StartSyncAll lance une synchronisation delta pour tous les joueurs configurés.
// POST /api/v1/sync/all → 202 AsyncJobStatus.
func (h *SyncHandler) StartSyncAll(ctx context.Context, _ *struct{}) (*syncJobOutput, error) {
	job, err := h.svc.StartSyncAll(ctx, syncCaller(ctx))
	if err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	return &syncJobOutput{Body: job}, nil
}

// checkPlayerSlug refuse un player_slug de corps de requête vide ou trop long.
func checkPlayerSlug(slug string) error {
	if slug == "" || len(slug) > maxPlayerSlugLen {
		return humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
	}
	return nil
}

// syncCaller décrit l'appelant aux jobs de sync : la session de la requête et ses jetons Halo.
func syncCaller(ctx context.Context) port.SyncCaller {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return port.SyncCaller{}
	}
	return port.SyncCaller{Authenticated: true, HaloTokens: sess.HaloTokens}
}
