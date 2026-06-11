// Package handlers — admin_db_contention.go : dashboard admin « Contention DB
// (sync) ».
//
// GET /admin/db-contention → compteurs du sharedprovider B-swap (nombre de
// swaps RO↔RW, durées, lectures rejetées en 503). Lecture seule des métriques
// expvar — ne déclenche aucune écriture, ne peut pas échouer.
package handlers

import (
	"context"
	"net/http"

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

// Get retourne les compteurs de contention du shared provider (B-swap).
// GET /admin/db-contention.
func (h *AdminDBContentionHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.get(r.Context()))
}
