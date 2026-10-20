// Package api — prestige_player_slug_mw.go : middleware de câblage du player_slug
// du chemin dans le contexte de résolution Prestige.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/wire"
)

// prestigePlayerSlugCtx stampe le {player_slug} du chemin (déjà réconcilié avec
// la session par ownershipMW) dans le contexte : le LazyPrestigeService y ancre
// le player DB via wire.PlayerSlugFromContext. Ce que WithPlayerSlug attendait
// « à terme » (cf. prestige_lazy_service.go). Deux effets, tous deux nécessaires :
//
//  1. RÉPARE les routes unitaires par {id} (GetChallenge, UpdateChallenge,
//     AbandonChallenge, SuggestNext, GetArc) et les lectures roster
//     (ListSquadMembers via ListMySquads) qui, sans slug en contexte, n'ont
//     aucun body/query acteur à résoudre → échouaient en ErrPlayerNotResolved.
//  2. CLÔT le BOLA objet-level pour les objets ISOLÉS par player DB (défis/arcs
//     perso vivent dans stats.duckdb du joueur du chemin) : un {id} appartenant
//     à un autre joueur n'est pas dans ce DB → 404. Le slug du chemin étant
//     ownership-gardé, l'utilisateur ne peut cibler que son propre player DB.
//
// Les escouades et leurs défis (DB sociale partagée tous joueurs, non isolés par player
// DB) ne sont PAS couverts par cette isolation : chaque méthode du service qui reçoit un
// identifiant d'escouade vérifie que l'acteur en est membre (assertMemberUser ou
// membersIfMember) — garde-rail internal/prestige/squad_member_guard_test.go, qui date
// ses exceptions.
func prestigePlayerSlugCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slug := chi.URLParam(req, "player_slug")
		next.ServeHTTP(w, req.WithContext(wire.WithPlayerSlug(req.Context(), slug)))
	})
}
