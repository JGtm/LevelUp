// Package handlers — handler HTTP du module Prestige (Phase 3).
//
// Couvre les endpoints REST :
//
//	POST   /api/v1/challenges           — créer un défi (avec quotas mode pilote)
//	GET    /api/v1/challenges/{id}      — détail d'un défi
//	GET    /api/v1/challenges           — liste filtrée
//	PATCH  /api/v1/challenges/{id}      — édition (cible recalcule palier en mode libre)
//	DELETE /api/v1/challenges/{id}      — abandon
//	POST   /api/v1/challenges/{id}/suggest-next — alternatives palier supérieur
//	GET    /api/v1/prestige/me          — total PP + niveau (par titre ou cross-titre)
//	GET    /api/v1/prestige/leaderboard — classement amis
//	GET    /api/v1/templates/suggest    — propositions catalogue
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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
// Renvoie false → 403. Réutilise les primitives d'ownership (ADR 0024) ; câblé
// par le routeur. Nil = non câblé (tests / enforcement off) → passant.
type ActorGuard func(ctx context.Context, actorSlug string) bool

// NewPrestigeHandler construit le handler Prestige.
func NewPrestigeHandler(svc prestige.Service, appPlayers AppPlayersFunc) *PrestigeHandler {
	return &PrestigeHandler{svc: svc, appPlayers: appPlayers}
}

// WithActorGuard injecte la garde d'autorisation acteur des routes squad
// (ADR 0024 étendu aux routes top-level /squads, hors groupe /players/{slug}).
func (h *PrestigeHandler) WithActorGuard(g ActorGuard) *PrestigeHandler {
	h.actorGuard = g
	return h
}

// authorizeActor renvoie true si l'appelant peut agir au nom de actorSlug (ou si
// la garde n'est pas câblée). Sinon écrit 403 player_forbidden et renvoie false.
func (h *PrestigeHandler) authorizeActor(w http.ResponseWriter, r *http.Request, actorSlug string) bool {
	if h.actorGuard == nil || h.actorGuard(r.Context(), actorSlug) {
		return true
	}
	writeError(r.Context(), w, http.StatusForbidden, "player_forbidden",
		"Accès non autorisé à ce joueur.")
	return false
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

// ─────────── CreateChallenge ───────────

// CreateChallenge gère POST /challenges.
func (h *PrestigeHandler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	var body createChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
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
	c, err := h.svc.CreateChallenge(r.Context(), req)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// ─────────── GetChallenge ───────────

func (h *PrestigeHandler) GetChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	c, err := h.svc.GetChallenge(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── ListActiveChallenges ───────────

func (h *PrestigeHandler) ListActiveChallenges(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	list, err := h.svc.ListActiveChallenges(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenges": list, jsonKeyCount: len(list)})
}

// ─────────── UpdateChallenge ───────────

func (h *PrestigeHandler) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body updateChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	c, err := h.svc.UpdateChallenge(r.Context(), id, prestige.UpdateChallengePatch{
		Target: body.Target,
		Label:  body.Label,
	})
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── AbandonChallenge ───────────

func (h *PrestigeHandler) AbandonChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	if err := h.svc.AbandonChallenge(r.Context(), id); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────── SuggestNext ───────────

func (h *PrestigeHandler) SuggestNext(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	templates, err := h.svc.SuggestNext(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": templates})
}

// ─────────── GetMyPrestige ───────────

// GetMyPrestige gère GET /prestige/me.
//
// Si title_slug est fourni → vue par titre.
// Sinon → cross-titre (somme tous titres).
func (h *PrestigeHandler) GetMyPrestige(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_user_id", "user_id requis")
		return
	}
	titleSlug := r.URL.Query().Get("title_slug")
	up, err := h.svc.GetUserPrestige(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, up)
}

// ─────────── SuggestTemplates ───────────

func (h *PrestigeHandler) SuggestTemplates(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	count := 3
	if c := r.URL.Query().Get("count"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 && n <= 10 {
			count = n
		}
	}
	templates, err := h.svc.SuggestTemplates(r.Context(), userID, titleSlug, count)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

// ─────────── Arcs ───────────

type createArcBody struct {
	UserID      string `json:"user_id"`
	TitleSlug   string `json:"title_slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// CreateArc gère POST /arcs.
func (h *PrestigeHandler) CreateArc(w http.ResponseWriter, r *http.Request) {
	var body createArcBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	a, err := h.svc.CreateArc(r.Context(), prestige.CreateArcRequest{
		UserID:      body.UserID,
		TitleSlug:   body.TitleSlug,
		Title:       body.Title,
		Description: body.Description,
	})
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// GetArc gère GET /arcs/{id}.
func (h *PrestigeHandler) GetArc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	a, err := h.svc.GetArc(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// DeleteArc gère DELETE /arcs/{id}?user_id=&objectives=delete|detach.
//
//   - objectives=delete : supprime aussi les objectifs (abandon, ou hard delete
//     si l'arc a moins d'1h → zéro cooldown).
//   - objectives=detach : détache les objectifs (gardés, redeviennent libres).
func (h *PrestigeHandler) DeleteArc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id requis")
		return
	}
	objectives := r.URL.Query().Get("objectives")
	if objectives != "delete" && objectives != "detach" {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_input",
			"objectives doit valoir 'delete' ou 'detach'")
		return
	}
	opts := prestige.DeleteArcOptions{CascadeObjectives: objectives == "delete"}
	if err := h.svc.DeleteArc(r.Context(), userID, id, opts); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListArcs gère GET /arcs?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	arcs, err := h.svc.ListArcs(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"arcs": arcs, jsonKeyCount: len(arcs)})
}

// ListArcPresets gère GET /arcs/presets?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcPresets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	presets, err := h.svc.ListArcPresets(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"presets": presets, jsonKeyCount: len(presets)})
}

type adoptPresetArcBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// AdoptPresetArc gère POST /arcs/presets/{id}/adopt.
func (h *PrestigeHandler) AdoptPresetArc(w http.ResponseWriter, r *http.Request) {
	presetID := chi.URLParam(r, "id")
	if presetID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body adoptPresetArcBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.UserID == "" || body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	arc, err := h.svc.AdoptPresetArc(r.Context(), body.UserID, body.TitleSlug, presetID)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, arc)
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

// CreateSquadChallenge gère POST /squads/{squad_id}/challenges.
func (h *PrestigeHandler) CreateSquadChallenge(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body createSquadChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	body.SquadID = squadID
	sc, err := h.svc.CreateSquadChallenge(r.Context(), prestige.CreateSquadChallengeRequest{
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
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

// ListSquadChallenges gère GET /squads/{squad_id}/challenges.
func (h *PrestigeHandler) ListSquadChallenges(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	list, err := h.svc.ListSquadChallenges(r.Context(), squadID)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"squad_challenges": list, jsonKeyCount: len(list)})
}

type joinSquadChallengeBody struct {
	UserID     string `json:"user_id"`
	ChosenTier string `json:"chosen_tier,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// JoinSquadChallenge gère POST /squad-challenges/{id}/join.
func (h *PrestigeHandler) JoinSquadChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body joinSquadChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if err := h.svc.JoinSquadChallenge(r.Context(), id, body.UserID, prestige.Tier(body.ChosenTier), body.IsPrivate); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
			xuidBySlug[p.PlayerSlug] = p.XUID
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
func (h *PrestigeHandler) CreateSquad(w http.ResponseWriter, r *http.Request) {
	var body createSquadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.Name == "" || body.CreatedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_fields", "name et created_by requis")
		return
	}
	if !h.authorizeActor(w, r, body.CreatedBy) {
		return
	}
	slugByXUID, xuidBySlug, gamertagByXUID, err := h.playerDirectory(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "directory_error", err.Error())
		return
	}
	creatorXUID := xuidBySlug[body.CreatedBy]
	if creatorXUID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "unknown_creator",
			"created_by introuvable parmi les joueurs de l'app")
		return
	}
	sc, err := h.svc.CreateSquad(r.Context(), prestige.CreateSquadRequest{
		Name:      body.Name,
		CreatedBy: body.CreatedBy,
		Members:   buildSquadMembers(body, creatorXUID, slugByXUID, gamertagByXUID),
	})
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

// ListMySquads gère GET /squads?user_id=slug — escouades dont user_id est
// membre-user, roster embarqué.
func (h *PrestigeHandler) ListMySquads(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_user_id", "user_id requis")
		return
	}
	if !h.authorizeActor(w, r, userID) {
		return
	}
	squads, err := h.svc.ListSquadsForUser(r.Context(), userID)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	titleSlug := r.URL.Query().Get("title_slug")
	// Annuaire (best-effort) pour combler les gamertags absents des snapshots
	// membres (legacy/pré-colonne) — app-players uniquement, via lookup canonique.
	_, _, gamertagByXUID, _ := h.playerDirectory(r.Context())
	out := make([]squadWithMembers, 0, len(squads))
	for _, sq := range squads {
		members, mErr := h.svc.ListSquadMembers(r.Context(), sq.ID)
		if mErr != nil {
			writeServiceError(r.Context(), w, mErr)
			return
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
		if pls, mds, uErr := h.svc.SquadUsualContexts(r.Context(), roster, titleSlug); uErr == nil {
			entry.UsualPlaylists = pls
			entry.UsualModes = mds
		}
		out = append(out, entry)
	}
	writeJSON(w, http.StatusOK, map[string]any{"squads": out, jsonKeyCount: len(out)})
}

// AddSquadMember gère POST /squads/{squad_id}/members.
func (h *PrestigeHandler) AddSquadMember(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body addSquadMemberBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.XUID == "" || body.RequestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_fields", "xuid et requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, body.RequestedBy) {
		return
	}
	slugByXUID, _, gamertagByXUID, err := h.playerDirectory(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "directory_error", err.Error())
		return
	}
	gamertag := body.Gamertag
	if gamertag == "" {
		gamertag = gamertagByXUID[body.XUID]
	}
	member := prestige.SquadMember{Xuid: body.XUID, UserID: slugByXUID[body.XUID], Gamertag: gamertag}
	if err := h.svc.AddSquadMember(r.Context(), squadID, member, body.RequestedBy); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveSquadMember gère DELETE /squads/{squad_id}/members/{xuid}?requested_by=slug.
func (h *PrestigeHandler) RemoveSquadMember(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	xuid := chi.URLParam(r, "xuid")
	if squadID == "" || xuid == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_fields", "squad_id et xuid requis")
		return
	}
	requestedBy := r.URL.Query().Get("requested_by")
	if requestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_requested_by", "requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, requestedBy) {
		return
	}
	if err := h.svc.RemoveSquadMember(r.Context(), squadID, xuid, requestedBy); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type renameSquadBody struct {
	Name        string `json:"name"`
	RequestedBy string `json:"requested_by"` // player_slug de l'acteur (membre-user)
}

// RenameSquad gère PATCH /squads/{squad_id} — renomme l'escouade.
func (h *PrestigeHandler) RenameSquad(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body renameSquadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.Name == "" || body.RequestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_fields", "name et requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, body.RequestedBy) {
		return
	}
	if err := h.svc.RenameSquad(r.Context(), squadID, body.Name, body.RequestedBy); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSquad gère DELETE /squads/{squad_id}?requested_by=slug — supprime
// l'escouade (retrait append-only de tous ses membres).
func (h *PrestigeHandler) DeleteSquad(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	requestedBy := r.URL.Query().Get("requested_by")
	if requestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_requested_by", "requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, requestedBy) {
		return
	}
	if err := h.svc.DeleteSquad(r.Context(), squadID, requestedBy); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type evaluateSquadChallengeBody struct {
	RequestedBy string `json:"requested_by"` // player_slug de l'acteur (membre-user)
}

// EvaluateSquadChallenge gère POST /squad-challenges/{id}/evaluate : recalcule
// et persiste la progression du défi, et retourne la progression par membre.
func (h *PrestigeHandler) EvaluateSquadChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body evaluateSquadChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.RequestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_requested_by", "requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, body.RequestedBy) {
		return
	}
	progress, err := h.svc.EvaluateSquadChallenge(r.Context(), id, body.RequestedBy)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"progress": progress})
}

// SquadOrientation gère GET /squads/{squad_id}/orientation?requested_by=slug :
// renvoie l'axe focal de l'escouade (orientation à renforcer), "" si indisponible.
func (h *PrestigeHandler) SquadOrientation(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	requestedBy := r.URL.Query().Get("requested_by")
	if requestedBy == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_requested_by", "requested_by requis")
		return
	}
	if !h.authorizeActor(w, r, requestedBy) {
		return
	}
	axis, err := h.svc.SquadOrientation(r.Context(), squadID, requestedBy)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"axis": axis})
}

// ─────────── Mode pilote ───────────

type pilotModeBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// EnablePilotMode gère POST /pilot-mode/enable.
//
// Active le mode pilote pour un joueur : auto-attribue 1 quotidien + 1 hebdo
// forcé + propose 3 hebdo au choix. Idempotent : si des défis pilote actifs
// existent déjà sur ces cadences, ils sont conservés.
func (h *PrestigeHandler) EnablePilotMode(w http.ResponseWriter, r *http.Request) {
	var body pilotModeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.UserID == "" || body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	out, err := h.svc.EnablePilotMode(r.Context(), body.UserID, body.TitleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DisablePilotMode gère POST /pilot-mode/disable.
//
// Désactive le mode pilote. Les défis pilote en cours sont conservés (le
// joueur peut les terminer), aucune nouvelle auto-attribution ne se fera.
func (h *PrestigeHandler) DisablePilotMode(w http.ResponseWriter, r *http.Request) {
	var body pilotModeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.UserID == "" || body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	if err := h.svc.DisablePilotMode(r.Context(), body.UserID, body.TitleSlug); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
func (h *PrestigeHandler) RefreshSquadPool(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body refreshSquadPoolBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_title_slug", "title_slug requis")
		return
	}
	pool, err := h.svc.RefreshSquadPool(r.Context(), squadID, body.TitleSlug, body.RequestedBy)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pool": pool, jsonKeyCount: len(pool)})
}

// ─────────── Helper d'erreurs ───────────

// writeServiceError mappe les erreurs du service vers des codes HTTP.
//
// Centralise la traduction pour éviter la duplication dans chaque handler.
func writeServiceError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dblease.ErrDBLocked):
		// Le sync engine ou un autre handler tient le lease — on demande au
		// client de retry sous 5 s. Cf. plan db-concurrency commit 2.
		w.Header().Set("Retry-After", "5")
		writeError(ctx, w, http.StatusServiceUnavailable, "db_busy",
			"database is currently busy, please retry")
	case errors.Is(err, prestige.ErrChallengeNotFound),
		errors.Is(err, prestige.ErrArcNotFound),
		errors.Is(err, prestige.ErrUserNotFound):
		writeError(ctx, w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, prestige.ErrInvalidInput):
		writeError(ctx, w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, prestige.ErrForbidden):
		writeError(ctx, w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, prestige.ErrNotEditable):
		writeError(ctx, w, http.StatusForbidden, "not_editable", err.Error())
	case errors.Is(err, prestige.ErrAlreadyTerminal):
		writeError(ctx, w, http.StatusConflict, "already_terminal", err.Error())
	case errors.Is(err, prestige.ErrCooldownActive):
		writeError(ctx, w, http.StatusTooManyRequests, "cooldown_active", err.Error())
	default:
		// Erreur masquée à l'extérieur — ne pas exposer les internals
		// si la cause n'est pas explicitement une de nos sentinelles.
		msg := err.Error()
		if strings.Contains(msg, "stretch") {
			// Cas particulier RejectTooEasy formaté avec stretch dans le message
			writeError(ctx, w, http.StatusBadRequest, "challenge_too_easy", msg)
			return
		}
		writeError(ctx, w, http.StatusInternalServerError, "internal_error", msg)
	}
}
