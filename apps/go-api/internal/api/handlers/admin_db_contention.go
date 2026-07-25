// Package handlers — admin_db_contention.go : dashboard admin « Contention DB
// (sync) ».
//
// GET /admin/db-contention → compteurs du sharedprovider B-swap (nombre de
// swaps RO↔RW, durées, lectures rejetées en 503). Lecture seule des métriques
// expvar — ne déclenche aucune écriture, ne peut pas échouer.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin + NoStore hérités) et enregistre
// la route via huma.Get. Logique métier inchangée (DBContentionProvider), seul
// le wrapping HTTP change. Le chemin relatif est identique à la route chi
// d'origine (montée sous /admin par server.go).
package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// DBContentionProvider retourne la capture de contention DB (implémenté par
// ServiceRegistry.DBContention — injecté pour éviter le cycle d'import).
type DBContentionProvider func(ctx context.Context) domain.DBContentionResponse

// AdminDBContentionHandler sert le dashboard admin « Contention DB (sync) ».
type AdminDBContentionHandler struct {
	get DBContentionProvider
}

// NewAdminDBContentionHandler construit le handler.
func NewAdminDBContentionHandler(get DBContentionProvider) *AdminDBContentionHandler {
	return &AdminDBContentionHandler{get: get}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /admin +
// middleware RequireAuth/RequireAdmin + NoStore hérités).
func (h *AdminDBContentionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/db-contention", h.handleGet, humacore.Op("getAdminDBContention", "Contention DB — compteurs du shared provider B-swap (auth admin requis)", "admin"))
}

type adminDBContentionOutput struct {
	Body domain.DBContentionResponse
}

// handleGet retourne les compteurs de contention du shared provider (B-swap).
// GET /admin/db-contention.
func (h *AdminDBContentionHandler) handleGet(ctx context.Context, _ *struct{}) (*adminDBContentionOutput, error) {
	return &adminDBContentionOutput{Body: h.get(ctx)}, nil
}
