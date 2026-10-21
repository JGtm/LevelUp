// Package handlers — session_context.go : handler HTTP pour la gestion du contexte session.
//
// Endpoints :
//
//	POST /session/context  → SessionContextResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au même point de
// montage racine (pas de préfixe /players/{player_slug}) et enregistre le POST
// via huma.Post. Logique métier inchangée (session store + registry titres),
// seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// SessionHandler gère les endpoints de session.
type SessionHandler struct {
	store *session.Store
	// canSelectPlayer dit si la session peut LIRE le profil du slug demandé comme
	// joueur courant (même décision que la garde de propriété du groupe
	// /players/{player_slug}). nil : non câblé (mono-utilisateur, tests), tout slug
	// de forme valide est accepté.
	canSelectPlayer PlayerSelectionCheck
	// users : persiste la locale sur le COMPTE d'une session authentifiée qui la
	// change explicitement (SetLocale), pour qu'elle suive à la prochaine connexion
	// sur un autre appareil/cookie. nil : non câblé (mono-utilisateur, tests) — la
	// locale reste alors purement par session, comportement d'origine.
	users *userstore.Store
}

// PlayerSelectionCheck dit si la session de ctx peut choisir le profil `slug`
// comme joueur courant.
type PlayerSelectionCheck func(ctx context.Context, slug string) bool

// WithPlayerAccess câble le contrôle d'accès du joueur courant.
func (h *SessionHandler) WithPlayerAccess(check PlayerSelectionCheck) *SessionHandler {
	h.canSelectPlayer = check
	return h
}

// WithUserStore câble la persistance de la locale sur le compte (cf. champ users).
func (h *SessionHandler) WithUserStore(users *userstore.Store) *SessionHandler {
	h.users = users
	return h
}

// NewSessionHandler crée un SessionHandler.
func NewSessionHandler(store *session.Store) *SessionHandler {
	return &SessionHandler{store: store}
}

// Mount enregistre la route via Huma au même point de montage que le routeur
// chi fourni (racine : POST /session/context, pas de path param parent).
func (h *SessionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/session/context", h.handlePostContext, humacore.Op("updateSessionContext", "Mise à jour du contexte de session", "bootstrap"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// sessionContextInput : corps brut décodé maison. RawBody (pas Body typé) →
// préserve le contrat 400 {invalid_body} sur JSON invalide (un Body typé
// renverrait le 422 de validation Huma).
type sessionContextInput struct {
	RawBody []byte
}

type sessionContextOutput struct{ Body domain.SessionContextResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handlePostContext met à jour le contexte de session (player_slug, title_slug,
// locale).
//
// POST /session/context
func (h *SessionHandler) handlePostContext(ctx context.Context, in *sessionContextInput) (*sessionContextOutput, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session non initialisée")
	}

	var req domain.SessionContextRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps JSON invalide")
	}
	// Validation AVANT toute mutation : WithSession persiste la session de la
	// requête (Touch) même quand le handler répond en erreur.
	if err := validateSessionContextRequest(&req); err != nil {
		return nil, err
	}
	// Contrôle d'accès AVANT toute mutation, pour la même raison : le joueur
	// courant ne désigne qu'un profil que la session peut lire.
	if req.PlayerSlug != nil && h.canSelectPlayer != nil && !h.canSelectPlayer(ctx, *req.PlayerSlug) {
		slog.WarnContext(ctx, "session_context: joueur courant refusé — profil non accessible",
			"slug", *req.PlayerSlug)
		return nil, humacore.NewError(http.StatusForbidden, "player_forbidden",
			"Access to this player is not allowed.")
	}

	if req.PlayerSlug != nil {
		sess.CurrentPlayerSlug = req.PlayerSlug
	}
	if req.TitleSlug != nil && *req.TitleSlug != "" {
		// Sprint 44 : switch titre — valider que le slug est connu. MT-16 : le
		// registre PARTAGÉ (DefaultRegistry) connaît les titres découverts en
		// config → on peut switcher vers un 2e titre déclaré.
		reg := titlePkg.DefaultRegistry()
		if reg.Exists(*req.TitleSlug) {
			sess.CurrentTitleSlug = *req.TitleSlug
			// Invalider le joueur courant lors d'un switch de titre
			// (le frontend devra re-sélectionner).
			sess.CurrentPlayerSlug = nil
		}
	}
	if req.Locale != nil {
		sess.Locale = *req.Locale
		h.persistAccountLocale(ctx, sess, *req.Locale)
	}

	if err := h.store.Save(sess); err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "session_save_error", err)
	}

	return &sessionContextOutput{Body: domain.SessionContextResponse{
		CurrentPlayerSlug: sess.CurrentPlayerSlug,
		CurrentTitleSlug:  sess.CurrentTitleSlug,
		AvailableTitles:   service.BuildAvailableTitles(),
		Locale:            sess.Locale,
		HintsVisible:      sess.HintsVisible,
		AuthReady:         sess.AuthReady,
	}}, nil
}

// persistAccountLocale propage la locale au COMPTE d'une session authentifiée, pour
// qu'elle suive à la prochaine connexion sur un autre appareil/cookie au lieu de
// retomber au défaut "fr". Best-effort : une session anonyme (sess.Username nil) ou
// un store non câblé ne font rien ; une écriture échouée est journalisée et ne fait
// pas échouer la requête (la locale reste posée en session dans tous les cas).
func (h *SessionHandler) persistAccountLocale(ctx context.Context, sess *domain.SessionData, locale string) {
	if h.users == nil || sess.Username == nil {
		return
	}
	if err := h.users.SetLocale(*sess.Username, locale); err != nil {
		slog.ErrorContext(ctx, "session_context: persistance locale compte échouée",
			"username", *sess.Username, "err", err)
	}
}
