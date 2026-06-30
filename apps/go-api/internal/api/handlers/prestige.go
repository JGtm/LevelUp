// Package handlers — handler HTTP du module Prestige (Phase 3).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// fourni et enregistre les 26 routes via huma.*. Logique métier inchangée
// (prestige.Service), seul le wrapping HTTP change. Les routes sont relatives au
// point de montage (le routeur racine dans server.go / un sous-routeur de test) —
// aucun préfixe /players/{player_slug} ici, le module Prestige est cross-joueur.
//
// Couvre les endpoints REST :
//
//	POST   /challenges                 — créer un défi (avec quotas mode pilote)
//	GET    /challenges/{id}            — détail d'un défi
//	GET    /challenges                 — liste filtrée
//	PATCH  /challenges/{id}            — édition (cible recalcule palier en mode libre)
//	DELETE /challenges/{id}            — abandon
//	POST   /challenges/{id}/suggest-next — alternatives palier supérieur
//	GET    /prestige/me                — total PP + niveau (par titre ou cross-titre)
//	GET    /templates/suggest          — propositions catalogue
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
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
// de `actorSlug` (created_by/requested_by/user_id des routes squad top-level).
// Renvoie false → 403. Réutilise les primitives d'ownership (ADR 0029) ; câblé
// par le routeur. Nil = non câblé (tests / enforcement off) → passant.
type ActorGuard func(ctx context.Context, actorSlug string) bool

// NewPrestigeHandler construit le handler Prestige.
func NewPrestigeHandler(svc prestige.Service, appPlayers AppPlayersFunc) *PrestigeHandler {
	return &PrestigeHandler{svc: svc, appPlayers: appPlayers}
}

// WithActorGuard injecte la garde d'autorisation acteur des routes squad
// (ADR 0029 étendu aux routes top-level /squads, hors groupe /players/{slug}).
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

	huma.Post(api, "/challenges", h.CreateChallenge)
	huma.Get(api, "/challenges", h.ListActiveChallenges)
	huma.Get(api, "/challenges/{id}", h.GetChallenge)
	huma.Patch(api, "/challenges/{id}", h.UpdateChallenge)
	huma.Delete(api, "/challenges/{id}", h.AbandonChallenge)
	huma.Post(api, "/challenges/{id}/suggest-next", h.SuggestNext)

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

// squadIDInput : un seul path param {squad_id}.
type squadIDInput struct {
	SquadID string `path:"squad_id"`
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

// CreateChallenge gère POST /challenges.
func (h *PrestigeHandler) CreateChallenge(ctx context.Context, in *rawBodyInput) (*challengeCreatedOutput, error) {
	var body createChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
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

// SuggestNext gère POST /challenges/{id}/suggest-next. Ne lit pas de corps
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

// ─────────── Arcs ───────────

type createArcBody struct {
	UserID      string `json:"user_id"`
	TitleSlug   string `json:"title_slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// arcOutput : 200 — objet Arc brut.
type arcOutput struct{ Body prestige.Arc }

// arcCreatedOutput : 201 — objet Arc créé.
type arcCreatedOutput struct {
	Status int
	Body   prestige.Arc
}

// CreateArc gère POST /arcs.
func (h *PrestigeHandler) CreateArc(ctx context.Context, in *rawBodyInput) (*arcCreatedOutput, error) {
	var body createArcBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	a, err := h.svc.CreateArc(ctx, prestige.CreateArcRequest{
		UserID:      body.UserID,
		TitleSlug:   body.TitleSlug,
		Title:       body.Title,
		Description: body.Description,
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcCreatedOutput{Status: http.StatusCreated, Body: a}, nil
}

// GetArc gère GET /arcs/{id}.
func (h *PrestigeHandler) GetArc(ctx context.Context, in *idInput) (*arcOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	a, err := h.svc.GetArc(ctx, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcOutput{Body: a}, nil
}

type deleteArcInput struct {
	ID         string `path:"id"`
	UserID     string `query:"user_id"`
	Objectives string `query:"objectives"`
}

// DeleteArc gère DELETE /arcs/{id}?user_id=&objectives=delete|detach.
//
//   - objectives=delete : supprime aussi les objectifs (abandon, ou hard delete
//     si l'arc a moins d'1h → zéro cooldown).
//   - objectives=detach : détache les objectifs (gardés, redeviennent libres).
func (h *PrestigeHandler) DeleteArc(ctx context.Context, in *deleteArcInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	if in.UserID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id requis")
	}
	if in.Objectives != "delete" && in.Objectives != "detach" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input",
			"objectives doit valoir 'delete' ou 'detach'")
	}
	opts := prestige.DeleteArcOptions{CascadeObjectives: in.Objectives == "delete"}
	if err := h.svc.DeleteArc(ctx, in.UserID, in.ID, opts); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type listArcsInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
}

// ListArcs gère GET /arcs?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcs(ctx context.Context, in *listArcsInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	arcs, err := h.svc.ListArcs(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"arcs": arcs, jsonKeyCount: len(arcs)}}, nil
}

// ListArcPresets gère GET /arcs/presets?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcPresets(ctx context.Context, in *listArcsInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	presets, err := h.svc.ListArcPresets(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"presets": presets, jsonKeyCount: len(presets)}}, nil
}

type adoptPresetArcBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// AdoptPresetArc gère POST /arcs/presets/{id}/adopt.
func (h *PrestigeHandler) AdoptPresetArc(ctx context.Context, in *idBodyInput) (*arcCreatedOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body adoptPresetArcBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.UserID == "" || body.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	arc, err := h.svc.AdoptPresetArc(ctx, body.UserID, body.TitleSlug, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcCreatedOutput{Status: http.StatusCreated, Body: arc}, nil
}

// ─────────── Squad challenges ───────────

type createSquadChallengeBody struct {
	SquadID         string  `json:"squad_id"`
	TemplateID      string  `json:"template_id,omitempty"`
	TitleSlug       string  `json:"title_slug"`
	Mode            string  `json:"mode"`
	EvalType        string  `json:"eval_type"`
	WindowType      string  `json:"window_type"`
	WindowValue     string  `json:"window_value,omitempty"`
	TargetPerMember float64 `json:"target_per_member,omitempty"`
	CreatedBy       string  `json:"created_by"`
}

// squadChallengeCreatedOutput : 201 — objet SquadChallenge créé.
type squadChallengeCreatedOutput struct {
	Status int
	Body   prestige.SquadChallenge
}

// CreateSquadChallenge gère POST /squads/{squad_id}/challenges.
func (h *PrestigeHandler) CreateSquadChallenge(ctx context.Context, in *squadIDBodyInput) (*squadChallengeCreatedOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	var body createSquadChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	body.SquadID = in.SquadID
	sc, err := h.svc.CreateSquadChallenge(ctx, prestige.CreateSquadChallengeRequest{
		SquadID:         body.SquadID,
		TemplateID:      body.TemplateID,
		TitleSlug:       body.TitleSlug,
		Mode:            prestige.SquadMode(body.Mode),
		EvalType:        prestige.EvalType(body.EvalType),
		WindowType:      prestige.WindowType(body.WindowType),
		WindowValue:     body.WindowValue,
		TargetPerMember: body.TargetPerMember,
		CreatedBy:       body.CreatedBy,
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &squadChallengeCreatedOutput{Status: http.StatusCreated, Body: sc}, nil
}

// ListSquadChallenges gère GET /squads/{squad_id}/challenges.
func (h *PrestigeHandler) ListSquadChallenges(ctx context.Context, in *squadIDInput) (*mapOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	list, err := h.svc.ListSquadChallenges(ctx, in.SquadID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"squad_challenges": list, jsonKeyCount: len(list)}}, nil
}

type joinSquadChallengeBody struct {
	UserID     string `json:"user_id"`
	ChosenTier string `json:"chosen_tier,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// JoinSquadChallenge gère POST /squad-challenges/{id}/join.
func (h *PrestigeHandler) JoinSquadChallenge(ctx context.Context, in *idBodyInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body joinSquadChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := h.svc.JoinSquadChallenge(ctx, in.ID, body.UserID, prestige.Tier(body.ChosenTier), body.IsPrivate); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

// ─────────── Escouade — roster (CRUD) ───────────

type squadMemberInput struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag,omitempty"`
}

type createSquadBody struct {
	Name      string             `json:"name"`
	CreatedBy string             `json:"created_by"` // player_slug du créateur
	Members   []squadMemberInput `json:"members,omitempty"`
}

type addSquadMemberBody struct {
	XUID        string `json:"xuid"`
	Gamertag    string `json:"gamertag,omitempty"`
	RequestedBy string `json:"requested_by"` // player_slug de l'acteur (membre-user)
}

// squadWithMembers est la réponse d'une escouade avec son roster + l'indice
// dérivé des playlists/modes habituels (best-effort, jamais stocké).
type squadWithMembers struct {
	Squad          prestige.Squad         `json:"squad"`
	Members        []prestige.SquadMember `json:"members"`
	UsualPlaylists []string               `json:"usual_playlists,omitempty"`
	UsualModes     []string               `json:"usual_modes,omitempty"`
}

// squadCreatedOutput : 201 — objet Squad créé.
type squadCreatedOutput struct {
	Status int
	Body   prestige.Squad
}

// playerDirectory construit les maps xuid→slug, slug→xuid et xuid→gamertag
// depuis l'annuaire db_profiles. Si appPlayers est nil, retourne des maps vides.
func (h *PrestigeHandler) playerDirectory(ctx context.Context) (slugByXUID, xuidBySlug, gamertagByXUID map[string]string, err error) {
	slugByXUID = map[string]string{}
	xuidBySlug = map[string]string{}
	gamertagByXUID = map[string]string{}
	if h.appPlayers == nil {
		return slugByXUID, xuidBySlug, gamertagByXUID, nil
	}
	players, err := h.appPlayers(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, p := range players {
		if p.XUID != "" {
			slugByXUID[p.XUID] = p.PlayerSlug
			if p.Gamertag != "" {
				gamertagByXUID[p.XUID] = p.Gamertag
			}
		}
		if p.PlayerSlug != "" {
			// Clé normalisée minuscule : le slug d'URL (body.CreatedBy) peut différer
			// de casse du slug db_profiles → sinon la résolution du créateur échoue en
			// unknown_creator silencieux (cf. lookup CreateSquad).
			xuidBySlug[strings.ToLower(p.PlayerSlug)] = p.XUID
		}
	}
	return slugByXUID, xuidBySlug, gamertagByXUID, nil
}

// buildSquadMembers assemble le roster : créateur (membre-user) + membres du
// body, dédupliqués par xuid, chacun tagué user_id si joueur de l'app et avec
// son gamertag (snapshot d'affichage : priorité au body, fallback annuaire).
func buildSquadMembers(body createSquadBody, creatorXUID string, slugByXUID, gamertagByXUID map[string]string) []prestige.SquadMember {
	gtFromBody := map[string]string{}
	for _, m := range body.Members {
		if m.XUID != "" && m.Gamertag != "" {
			gtFromBody[m.XUID] = m.Gamertag
		}
	}
	seen := map[string]bool{}
	members := make([]prestige.SquadMember, 0, len(body.Members)+1)
	add := func(xuid string) {
		if xuid == "" || seen[xuid] {
			return
		}
		seen[xuid] = true
		gt := gtFromBody[xuid]
		if gt == "" {
			gt = gamertagByXUID[xuid]
		}
		members = append(members, prestige.SquadMember{Xuid: xuid, UserID: slugByXUID[xuid], Gamertag: gt})
	}
	add(creatorXUID)
	for _, m := range body.Members {
		add(m.XUID)
	}
	return members
}

// CreateSquad gère POST /squads.
func (h *PrestigeHandler) CreateSquad(ctx context.Context, in *rawBodyInput) (*squadCreatedOutput, error) {
	var body createSquadBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.Name == "" || body.CreatedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_fields", "name et created_by requis")
	}
	if err := h.authorizeActor(ctx, body.CreatedBy); err != nil {
		return nil, err
	}
	slugByXUID, xuidBySlug, gamertagByXUID, err := h.playerDirectory(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "directory_error", err.Error())
	}
	creatorXUID := xuidBySlug[strings.ToLower(body.CreatedBy)]
	if creatorXUID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "unknown_creator",
			"created_by introuvable parmi les joueurs de l'app")
	}
	sc, err := h.svc.CreateSquad(ctx, prestige.CreateSquadRequest{
		Name:      body.Name,
		CreatedBy: body.CreatedBy,
		Members:   buildSquadMembers(body, creatorXUID, slugByXUID, gamertagByXUID),
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &squadCreatedOutput{Status: http.StatusCreated, Body: sc}, nil
}

type listMySquadsInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
}

// ListMySquads gère GET /squads?user_id=slug — escouades dont user_id est
// membre-user, roster embarqué.
func (h *PrestigeHandler) ListMySquads(ctx context.Context, in *listMySquadsInput) (*mapOutput, error) {
	if in.UserID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_user_id", "user_id requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	squads, err := h.svc.ListSquadsForUser(ctx, in.UserID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	titleSlug := in.TitleSlug
	// Annuaire (best-effort) pour combler les gamertags absents des snapshots
	// membres (legacy/pré-colonne) — app-players uniquement, via lookup canonique.
	_, _, gamertagByXUID, _ := h.playerDirectory(ctx)
	out := make([]squadWithMembers, 0, len(squads))
	for _, sq := range squads {
		members, mErr := h.svc.ListSquadMembers(ctx, sq.ID)
		if mErr != nil {
			return nil, h.serviceError(ctx, mErr)
		}
		for i := range members {
			if members[i].Gamertag == "" {
				if gt := gamertagByXUID[members[i].Xuid]; gt != "" {
					members[i].Gamertag = gt
				}
			}
		}
		entry := squadWithMembers{Squad: sq, Members: members}
		// Indice dérivé (best-effort) : playlists/modes habituels du roster. Une
		// erreur (provider absent, lecture shared KO) n'empêche pas la liste.
		roster := make([]string, 0, len(members))
		for _, m := range members {
			if m.Xuid != "" {
				roster = append(roster, m.Xuid)
			}
		}
		if pls, mds, uErr := h.svc.SquadUsualContexts(ctx, roster, titleSlug); uErr == nil {
			entry.UsualPlaylists = pls
			entry.UsualModes = mds
		}
		out = append(out, entry)
	}
	return &mapOutput{Body: map[string]any{"squads": out, jsonKeyCount: len(out)}}, nil
}

// AddSquadMember gère POST /squads/{squad_id}/members.
func (h *PrestigeHandler) AddSquadMember(ctx context.Context, in *squadIDBodyInput) (*noContentOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	var body addSquadMemberBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.XUID == "" || body.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_fields", "xuid et requested_by requis")
	}
	if err := h.authorizeActor(ctx, body.RequestedBy); err != nil {
		return nil, err
	}
	slugByXUID, _, gamertagByXUID, err := h.playerDirectory(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "directory_error", err.Error())
	}
	// Gamertag explicite du body, sinon backfill via l'annuaire (delta main porté en Huma).
	gamertag := body.Gamertag
	if gamertag == "" {
		gamertag = gamertagByXUID[body.XUID]
	}
	member := prestige.SquadMember{Xuid: body.XUID, UserID: slugByXUID[body.XUID], Gamertag: gamertag}
	if err := h.svc.AddSquadMember(ctx, in.SquadID, member, body.RequestedBy); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type removeSquadMemberInput struct {
	SquadID     string `path:"squad_id"`
	XUID        string `path:"xuid"`
	RequestedBy string `query:"requested_by"`
}

// RemoveSquadMember gère DELETE /squads/{squad_id}/members/{xuid}?requested_by=slug.
func (h *PrestigeHandler) RemoveSquadMember(ctx context.Context, in *removeSquadMemberInput) (*noContentOutput, error) {
	if in.SquadID == "" || in.XUID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_fields", "squad_id et xuid requis")
	}
	if in.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_requested_by", "requested_by requis")
	}
	if err := h.authorizeActor(ctx, in.RequestedBy); err != nil {
		return nil, err
	}
	if err := h.svc.RemoveSquadMember(ctx, in.SquadID, in.XUID, in.RequestedBy); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type renameSquadBody struct {
	Name        string `json:"name"`
	RequestedBy string `json:"requested_by"` // player_slug de l'acteur (membre-user)
}

// RenameSquad gère PATCH /squads/{squad_id} — renomme l'escouade. (Porté en Huma
// lors du merge : feature ajoutée chi sur main, ré-exprimée en signature Huma.)
func (h *PrestigeHandler) RenameSquad(ctx context.Context, in *squadIDBodyInput) (*noContentOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	var body renameSquadBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.Name == "" || body.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_fields", "name et requested_by requis")
	}
	if err := h.authorizeActor(ctx, body.RequestedBy); err != nil {
		return nil, err
	}
	if err := h.svc.RenameSquad(ctx, in.SquadID, body.Name, body.RequestedBy); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type deleteSquadInput struct {
	SquadID     string `path:"squad_id"`
	RequestedBy string `query:"requested_by"`
}

// DeleteSquad gère DELETE /squads/{squad_id}?requested_by=slug — supprime
// l'escouade (retrait append-only de tous ses membres). Porté en Huma au merge.
func (h *PrestigeHandler) DeleteSquad(ctx context.Context, in *deleteSquadInput) (*noContentOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	if in.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_requested_by", "requested_by requis")
	}
	if err := h.authorizeActor(ctx, in.RequestedBy); err != nil {
		return nil, err
	}
	if err := h.svc.DeleteSquad(ctx, in.SquadID, in.RequestedBy); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type evaluateSquadChallengeBody struct {
	RequestedBy string `json:"requested_by"` // player_slug de l'acteur (membre-user)
}

// EvaluateSquadChallenge gère POST /squad-challenges/{id}/evaluate : recalcule
// et persiste la progression du défi, et retourne la progression par membre.
func (h *PrestigeHandler) EvaluateSquadChallenge(ctx context.Context, in *idBodyInput) (*mapOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body evaluateSquadChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_requested_by", "requested_by requis")
	}
	if err := h.authorizeActor(ctx, body.RequestedBy); err != nil {
		return nil, err
	}
	progress, err := h.svc.EvaluateSquadChallenge(ctx, in.ID, body.RequestedBy)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"progress": progress}}, nil
}

type squadOrientationInput struct {
	SquadID     string `path:"squad_id"`
	RequestedBy string `query:"requested_by"`
}

// SquadOrientation gère GET /squads/{squad_id}/orientation?requested_by=slug :
// renvoie l'axe focal de l'escouade (orientation à renforcer), "" si indisponible.
func (h *PrestigeHandler) SquadOrientation(ctx context.Context, in *squadOrientationInput) (*mapOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	if in.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_requested_by", "requested_by requis")
	}
	if err := h.authorizeActor(ctx, in.RequestedBy); err != nil {
		return nil, err
	}
	axis, err := h.svc.SquadOrientation(ctx, in.SquadID, in.RequestedBy)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"axis": axis}}, nil
}

// ─────────── Mode pilote ───────────

type pilotModeBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// pilotModeOutput : 200 — objet PilotModeAttribution.
type pilotModeOutput struct{ Body prestige.PilotModeAttribution }

// EnablePilotMode gère POST /pilot-mode/enable.
//
// Active le mode pilote pour un joueur : auto-attribue 1 quotidien + 1 hebdo
// forcé + propose 3 hebdo au choix. Idempotent : si des défis pilote actifs
// existent déjà sur ces cadences, ils sont conservés.
func (h *PrestigeHandler) EnablePilotMode(ctx context.Context, in *rawBodyInput) (*pilotModeOutput, error) {
	var body pilotModeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.UserID == "" || body.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	out, err := h.svc.EnablePilotMode(ctx, body.UserID, body.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &pilotModeOutput{Body: out}, nil
}

// DisablePilotMode gère POST /pilot-mode/disable.
//
// Désactive le mode pilote. Les défis pilote en cours sont conservés (le
// joueur peut les terminer), aucune nouvelle auto-attribution ne se fera.
func (h *PrestigeHandler) DisablePilotMode(ctx context.Context, in *rawBodyInput) (*noContentOutput, error) {
	var body pilotModeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.UserID == "" || body.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.svc.DisablePilotMode(ctx, body.UserID, body.TitleSlug); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

// ─────────── Pool collectif squad ───────────

type refreshSquadPoolBody struct {
	TitleSlug   string `json:"title_slug"`
	RequestedBy string `json:"requested_by"`
}

// RefreshSquadPool gère POST /squads/{squad_id}/challenges/pool/refresh.
//
// Génère un pool de 6-9 templates thématiques pour l'escouade. Le membre
// qui requête doit être dans l'escouade. Le pool est ensuite consommé par
// les membres qui peuvent en proposer un défi à l'équipe.
func (h *PrestigeHandler) RefreshSquadPool(ctx context.Context, in *squadIDBodyInput) (*mapOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	var body refreshSquadPoolBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_title_slug", "title_slug requis")
	}
	pool, err := h.svc.RefreshSquadPool(ctx, in.SquadID, body.TitleSlug, body.RequestedBy)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"pool": pool, jsonKeyCount: len(pool)}}, nil
}

// ─────────── Helper d'erreurs ───────────

// serviceError mappe les erreurs du service vers des erreurs Huma (status/code/
// message identiques à l'ancien writeServiceError). ErrDBLocked → 503 + header
// Retry-After:5 (huma.ErrorWithHeaders).
func (h *PrestigeHandler) serviceError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, dblease.ErrDBLocked):
		// Le sync engine ou un autre handler tient le lease — on demande au
		// client de retry sous 5 s. Cf. plan db-concurrency commit 2.
		return huma.ErrorWithHeaders(
			humacore.NewError(http.StatusServiceUnavailable, "db_busy",
				"database is currently busy, please retry"),
			http.Header{"Retry-After": []string{"5"}},
		)
	case errors.Is(err, prestige.ErrChallengeNotFound),
		errors.Is(err, prestige.ErrArcNotFound),
		errors.Is(err, prestige.ErrUserNotFound):
		return humacore.NewError(http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, prestige.ErrInvalidInput):
		return humacore.NewError(http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, prestige.ErrForbidden):
		return humacore.NewError(http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, prestige.ErrNotEditable):
		return humacore.NewError(http.StatusForbidden, "not_editable", err.Error())
	case errors.Is(err, prestige.ErrAlreadyTerminal):
		return humacore.NewError(http.StatusConflict, "already_terminal", err.Error())
	case errors.Is(err, prestige.ErrCooldownActive):
		return humacore.NewError(http.StatusTooManyRequests, "cooldown_active", err.Error())
	default:
		// Erreur masquée à l'extérieur — ne pas exposer les internals
		// si la cause n'est pas explicitement une de nos sentinelles.
		msg := err.Error()
		if strings.Contains(msg, "stretch") {
			// Cas particulier RejectTooEasy formaté avec stretch dans le message
			return humacore.NewError(http.StatusBadRequest, "challenge_too_easy", msg)
		}
		return humacore.NewError(http.StatusInternalServerError, "internal_error", msg)
	}
}
