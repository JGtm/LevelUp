// Package handlers — handler HTTP du module Prestige (Phase 3).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// fourni et enregistre les 28 routes via huma.*. Logique métier inchangée
// (prestige.Service), seul le wrapping HTTP change. Les routes sont relatives au
// point de montage : server_apiv1.go appelle ph.Mount(r) DANS le groupe
// r.Route("/players/{player_slug}", …), gardé par ownershipMW (ADR 0029). Le
// préfixe /players/{player_slug} n'apparaît donc pas dans les littéraux ci-dessous
// (chemins relatifs), mais TOUTE route est effectivement servie sous ce préfixe et
// protégée par la propriété joueur — le segment {player_slug} porte l'ownership et
// le client DOIT le fournir (côté web : chokepoint scopedToPlayer de
// apps/web/src/lib/prestige.ts, figé par prestige.paths.test.ts). Les handlers qui
// ont besoin de l'acteur pour leur logique métier le relisent depuis le body/la
// query (user_id / created_by / requested_by) et le RÉCONCILIENT avec la session
// via authorizeActor (l'acteur doit être un profil possédé — CanAccessPlayer,
// multi-profil famille), sinon 403 player_forbidden : ownershipMW garde le segment
// d'URL, pas le payload, donc un acteur body pointant un tiers serait un BOLA
// horizontal sans cette garde (garde-rail AST : prestige_actor_guard_test.go).
// Les routes unitaires par {id} ne lisent pas d'acteur — le slug ne sert alors qu'à ownershipMW.
//
// Couvre les endpoints REST (chemins relatifs, tous sous /players/{player_slug}) :
//
//	POST   /prestige/challenges                 — créer un défi (avec quotas mode pilote)
//	GET    /prestige/challenges/{id}            — détail d'un défi
//	GET    /prestige/challenges                 — liste filtrée
//	PATCH  /prestige/challenges/{id}            — édition (cible recalcule palier en mode libre)
//	DELETE /prestige/challenges/{id}            — abandon
//	POST   /prestige/challenges/{id}/suggest-next — alternatives palier supérieur
//	GET    /prestige/me                          — total PP + niveau (par titre ou cross-titre)
//	GET    /templates/suggest                    — propositions catalogue
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/prestige"
)

// PrestigeHandler regroupe les routes du module Prestige.
//
// Une seule struct car le module a un service unique et toutes les routes
// sont closely related. Préférable à 5 handlers fragmentés pour la lisibilité.
type PrestigeHandler struct {
	svc        prestige.Service
	appPlayers AppPlayersFunc
	actorGuard ActorGuard
}

// AppPlayersFunc retourne les joueurs de l'app (db_profiles). Sert à résoudre
// xuid ↔ player_slug lors de la gestion du roster d'escouade (taguer user_id
// des membres qui sont utilisateurs de l'app, résoudre le xuid du créateur).
// Peut être nil (les membres ne seront alors pas tagués user_id).
type AppPlayersFunc func(ctx context.Context) ([]domain.PlayerSummary, error)

// ActorGuard valide que l'appelant (session dans ctx) a le droit d'agir au nom
// de `actorSlug` (tout created_by/requested_by/user_id lu depuis body ou query).
// Renvoie false → 403. Réutilise les primitives d'ownership (ADR 0029) ; câblé
// par le routeur. Nil = non câblé (tests / enforcement off) → passant.
type ActorGuard func(ctx context.Context, actorSlug string) bool

// NewPrestigeHandler construit le handler Prestige.
func NewPrestigeHandler(svc prestige.Service, appPlayers AppPlayersFunc) *PrestigeHandler {
	return &PrestigeHandler{svc: svc, appPlayers: appPlayers}
}

// WithActorGuard injecte la garde d'autorisation acteur, appliquée par
// authorizeActor sur TOUTE route lisant un acteur depuis le body/la query
// (ADR 0029 étendu : réconciliation user_id/created_by/requested_by ↔ session,
// clôt le BOLA horizontal résiduel des routes prestige non-squad).
func (h *PrestigeHandler) WithActorGuard(g ActorGuard) *PrestigeHandler {
	h.actorGuard = g
	return h
}

// Mount enregistre les 26 routes via Huma sur le routeur chi fourni.
//
// Les routes à corps REQUIS utilisent RawBody (décodage maison) pour préserver
// le contrat 400 {invalid_body} sur JSON malformé (un Body typé renverrait le 422
// de validation Huma). suggest-next ne lit pas de corps → input path seul.
func (h *PrestigeHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)

	huma.Post(api, "/prestige/challenges", h.CreateChallenge)
	huma.Get(api, "/prestige/challenges", h.ListActiveChallenges)
	huma.Get(api, "/prestige/challenges/{id}", h.GetChallenge)
	huma.Patch(api, "/prestige/challenges/{id}", h.UpdateChallenge)
	huma.Delete(api, "/prestige/challenges/{id}", h.AbandonChallenge)
	huma.Post(api, "/prestige/challenges/{id}/suggest-next", h.SuggestNext)

	huma.Post(api, "/arcs", h.CreateArc)
	huma.Get(api, "/arcs", h.ListArcs)
	huma.Get(api, "/arcs/presets", h.ListArcPresets)
	huma.Post(api, "/arcs/presets/{id}/adopt", h.AdoptPresetArc)
	huma.Get(api, "/arcs/{id}", h.GetArc)
	huma.Delete(api, "/arcs/{id}", h.DeleteArc)

	huma.Get(api, "/prestige/me", h.GetMyPrestige)
	huma.Get(api, "/templates/suggest", h.SuggestTemplates)

	huma.Post(api, "/squads/{squad_id}/challenges", h.CreateSquadChallenge)
	huma.Get(api, "/squads/{squad_id}/challenges", h.ListSquadChallenges)
	huma.Post(api, "/squads/{squad_id}/challenges/pool/refresh", h.RefreshSquadPool)
	huma.Post(api, "/squad-challenges/{id}/join", h.JoinSquadChallenge)

	huma.Post(api, "/squads", h.CreateSquad)
	huma.Get(api, "/squads", h.ListMySquads)
	huma.Patch(api, "/squads/{squad_id}", h.RenameSquad)
	huma.Delete(api, "/squads/{squad_id}", h.DeleteSquad)
	huma.Post(api, "/squads/{squad_id}/members", h.AddSquadMember)
	huma.Delete(api, "/squads/{squad_id}/members/{xuid}", h.RemoveSquadMember)
	huma.Post(api, "/squad-challenges/{id}/evaluate", h.EvaluateSquadChallenge)
	huma.Get(api, "/squads/{squad_id}/orientation", h.SquadOrientation)

	huma.Post(api, "/pilot-mode/enable", h.EnablePilotMode)
	huma.Post(api, "/pilot-mode/disable", h.DisablePilotMode)
}

// authorizeActor renvoie nil si l'appelant peut agir au nom de actorSlug (ou si
// la garde n'est pas câblée). Sinon renvoie une erreur Huma 403 player_forbidden.
func (h *PrestigeHandler) authorizeActor(ctx context.Context, actorSlug string) error {
	if h.actorGuard == nil || h.actorGuard(ctx, actorSlug) {
		return nil
	}
	return humacore.NewError(http.StatusForbidden, "player_forbidden",
		"Accès non autorisé à ce joueur.")
}

// ─────────── DTOs requête/réponse ───────────

type createChallengeBody struct {
	UserID          string  `json:"user_id"`
	TitleSlug       string  `json:"title_slug"`
	ArcID           string  `json:"arc_id,omitempty"`
	TemplateID      string  `json:"template_id,omitempty"`
	Metric          string  `json:"metric"`
	Target          float64 `json:"target"`
	WindowType      string  `json:"window_type"`
	WindowValue     string  `json:"window_value,omitempty"`
	Cadence         string  `json:"cadence"`
	EvalType        string  `json:"eval_type"`
	Mode            string  `json:"mode"`
	Label           string  `json:"label,omitempty"`
	IsPrivate       bool    `json:"is_private,omitempty"`
	TargetPerMember float64 `json:"target_per_member,omitempty"`
	Position        int     `json:"position,omitempty"`
	// Source : origine du défi (ADR 0020). Optionnel — défaut "user" (création
	// manuelle). Une valeur non reconnue est ignorée (retombe sur "user").
	Source string `json:"source,omitempty"`
}

type updateChallengeBody struct {
	Target *float64 `json:"target,omitempty"`
	Label  *string  `json:"label,omitempty"`
}

// ─────────── Inputs/Outputs Huma génériques ───────────

// idInput : un seul path param {id} (challenges, arcs, squad-challenges).
type idInput struct {
	ID string `path:"id"`
}

// noContentOutput : réponse 204 sans corps.
type noContentOutput struct {
	Status int
}

// challengeOutput : 200 — objet Challenge brut (contrat writeJSON inchangé).
type challengeOutput struct{ Body prestige.Challenge }

// challengeCreatedOutput : 201 — objet Challenge créé.
type challengeCreatedOutput struct {
	Status int
	Body   prestige.Challenge
}

// mapOutput : 200 — corps map[string]any (préserve les enveloppes {challenges,count} etc.).
type mapOutput struct{ Body map[string]any }

// ─── Inputs à corps brut (RawBody → contrat 400 invalid_body sur JSON malformé) ──

// rawBodyInput : corps REQUIS décodé maison, sans path param.
type rawBodyInput struct {
	RawBody []byte
}

// idBodyInput : path {id} + corps REQUIS décodé maison.
type idBodyInput struct {
	ID      string `path:"id"`
	RawBody []byte
}

// squadIDBodyInput : path {squad_id} + corps REQUIS décodé maison.
type squadIDBodyInput struct {
	SquadID string `path:"squad_id"`
	RawBody []byte
}

// ─────────── CreateChallenge ───────────

// CreateChallenge gère POST /prestige/challenges.
func (h *PrestigeHandler) CreateChallenge(ctx context.Context, in *rawBodyInput) (*challengeCreatedOutput, error) {
	var body createChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
	}
	req := prestige.CreateChallengeRequest{
		UserID:          body.UserID,
		TitleSlug:       body.TitleSlug,
		ArcID:           body.ArcID,
		TemplateID:      body.TemplateID,
		Metric:          body.Metric,
		Target:          body.Target,
		WindowType:      prestige.WindowType(body.WindowType),
		WindowValue:     body.WindowValue,
		Cadence:         prestige.Cadence(body.Cadence),
		EvalType:        prestige.EvalType(body.EvalType),
		Mode:            prestige.ChallengeMode(body.Mode),
		Label:           body.Label,
		IsPrivate:       body.IsPrivate,
		TargetPerMember: body.TargetPerMember,
		Position:        body.Position,
		// Création HTTP = manuelle par défaut. Une origine explicite valide
		// (ex. front pilote) reste prioritaire ; toute valeur inconnue -> "user".
		Source: prestige.ChallengeSourceUser,
	}
	if prestige.IsValidChallengeSource(body.Source) {
		req.Source = body.Source
	}
	c, err := h.svc.CreateChallenge(ctx, req)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &challengeCreatedOutput{Status: http.StatusCreated, Body: c}, nil
}

// ─────────── GetChallenge ───────────

func (h *PrestigeHandler) GetChallenge(ctx context.Context, in *idInput) (*challengeOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	c, err := h.svc.GetChallenge(ctx, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &challengeOutput{Body: c}, nil
}

// ─────────── ListActiveChallenges ───────────

type listActiveChallengesInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
}

func (h *PrestigeHandler) ListActiveChallenges(ctx context.Context, in *listActiveChallengesInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	list, err := h.svc.ListActiveChallenges(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"challenges": list, jsonKeyCount: len(list)}}, nil
}

// ─────────── UpdateChallenge ───────────

func (h *PrestigeHandler) UpdateChallenge(ctx context.Context, in *idBodyInput) (*challengeOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body updateChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	c, err := h.svc.UpdateChallenge(ctx, in.ID, prestige.UpdateChallengePatch{
		Target: body.Target,
		Label:  body.Label,
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &challengeOutput{Body: c}, nil
}

// ─────────── AbandonChallenge ───────────

func (h *PrestigeHandler) AbandonChallenge(ctx context.Context, in *idInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	if err := h.svc.AbandonChallenge(ctx, in.ID); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

// ─────────── SuggestNext ───────────

// SuggestNext gère POST /prestige/challenges/{id}/suggest-next. Ne lit pas de corps
// (input path seul) → pas de RawBody (sinon Huma rendrait un corps requis).
func (h *PrestigeHandler) SuggestNext(ctx context.Context, in *idInput) (*mapOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	templates, err := h.svc.SuggestNext(ctx, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"suggestions": templates}}, nil
}

// ─────────── GetMyPrestige ───────────

type getMyPrestigeInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
}

// userPrestigeOutput : 200 — objet UserPrestige brut.
type userPrestigeOutput struct{ Body prestige.UserPrestige }

// GetMyPrestige gère GET /prestige/me.
//
// Si title_slug est fourni → vue par titre.
// Sinon → cross-titre (somme tous titres).
func (h *PrestigeHandler) GetMyPrestige(ctx context.Context, in *getMyPrestigeInput) (*userPrestigeOutput, error) {
	if in.UserID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_user_id", "user_id requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	up, err := h.svc.GetUserPrestige(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &userPrestigeOutput{Body: up}, nil
}

// ─────────── SuggestTemplates ───────────

type suggestTemplatesInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
	Count     string `query:"count"`
}

func (h *PrestigeHandler) SuggestTemplates(ctx context.Context, in *suggestTemplatesInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	count := 3
	if in.Count != "" {
		if n, err := strconv.Atoi(in.Count); err == nil && n > 0 && n <= 10 {
			count = n
		}
	}
	templates, err := h.svc.SuggestTemplates(ctx, in.UserID, in.TitleSlug, count)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"templates": templates}}, nil
}
