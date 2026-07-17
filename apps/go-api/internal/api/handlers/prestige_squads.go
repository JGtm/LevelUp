// Package handlers — prestige_squads.go : handlers Huma du roster d escouade (CRUD).
// Extrait de prestige.go (K3f god-file split, 2026-07-06), même package.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/prestige"
)

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
	_, _, gamertagByXUID, dirErr := h.playerDirectory(ctx)
	if dirErr != nil {
		slog.DebugContext(ctx, "prestige: squad list gamertag backfill skipped (annuaire indisponible)",
			"err", dirErr, "user_id", in.UserID)
	}
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
		} else {
			slog.DebugContext(ctx, "prestige: squad usual contexts unavailable (indice omis)",
				"err", uErr, "squad_id", sq.ID)
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
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
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
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
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
	if err := h.authorizeActor(ctx, body.RequestedBy); err != nil {
		return nil, err
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
