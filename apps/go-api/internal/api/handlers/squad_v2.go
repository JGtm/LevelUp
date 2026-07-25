// Package handlers — squad_v2.go : handler HTTP de la nouvelle page Squad V2
// (multi-coéquipiers, fondations Phase 0).
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/squad/v2?teammates=gt1,gt2,gt3&period=1y
//
// Vit en parallèle de l'endpoint legacy /pages/squad jusqu'à migration complète
// du frontend (cf. PLAN_SQUAD_GO_PORTAGE).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le GET via huma.Get. Logique métier
// inchangée (SquadV2Service), seul le wrapping HTTP change.
//
// Filtres cascade supportés : experience_types, playlists, maps, modes
// (label FR COALESCE(name_fr, name) via PairMode/AssetReference).
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// squadV2MaxTeammates est la borne haute du nombre de coéquipiers acceptés
// par l'endpoint (cohérent avec service.MaxTeammates et la version Python).
const squadV2MaxTeammates = 3

// SquadV2Handler gère l'endpoint Squad V2.
type SquadV2Handler struct {
	newSvc ContextFactory[port.SquadV2Service]
}

// NewSquadV2Handler construit un SquadV2Handler avec sa factory de service.
func NewSquadV2Handler(newSvc ContextFactory[port.SquadV2Service]) *SquadV2Handler {
	return &SquadV2Handler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SquadV2Handler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/squad/v2", h.GetSquadPage, humacore.Op("getSquadV2Page", "Page Squad V2 (multi-coéquipiers, fondations Phase 0)", "squad"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// squadV2PageInput : {player_slug} + query params (tous tolérés vides — le
// parsing maison reproduit les codes d'erreur d'origine).
type squadV2PageInput struct {
	PlayerSlug      string `path:"player_slug"`
	Teammates       string `query:"teammates"`
	Period          string `query:"period"`
	ExperienceTypes string `query:"experience_types"`
	Playlists       string `query:"playlists"`
	Maps            string `query:"maps"`
	Modes           string `query:"modes"`
}

type squadV2PageOutput struct{ Body *domain.SquadPageV2Response }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// GetSquadPage traite GET /api/v1/players/{player_slug}/pages/squad/v2.
//
// Query params :
//   - teammates        : CSV de gamertags (max 3, optionnel)
//   - period           : "all" | "2y" | "1y" | "1m" | "1w" (défaut "all")
//   - experience_types : CSV de types d'expérience (ex: "PVP classé,PVE")
//   - playlists        : CSV de noms de playlist FR (ex: "Partie rapide")
//   - maps             : CSV de noms de carte FR (ex: "Décharge")
//
// Statuts :
//   - 200 : payload valide
//   - 400 : params invalides (period inconnu, trop de coéquipiers)
//   - 404 : joueur principal introuvable (factory)
//   - 503 : capability match.history absente pour le titre courant
func (h *SquadV2Handler) GetSquadPage(ctx context.Context, in *squadV2PageInput) (*squadV2PageOutput, error) {
	svc, _, gamertag, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	teammates, err := parseSquadV2Teammates(in.Teammates)
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_teammates", err.Error())
	}
	period, err := parseSquadV2Period(in.Period)
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_period", err.Error())
	}

	experienceTypes := parseCSVFilter(in.ExperienceTypes)
	playlists := parseCSVFilter(in.Playlists)
	maps := parseCSVFilter(in.Maps)
	modes := parseCSVFilter(in.Modes)

	titleSlug := ctxkeys.TitleSlug(ctx)

	slog.InfoContext(ctx, "squad_v2: lecture page",
		"player", gamertag,
		"title_slug", titleSlug,
		"teammates_count", len(teammates),
		"period", string(period),
		"experience_types", experienceTypes,
		"playlists", playlists,
		"maps", maps,
		"modes", modes,
	)

	resp, err := svc.GetSquadPage(ctx, titleSlug, gamertag, teammates, period, experienceTypes, playlists, maps, modes)
	if err != nil {
		if mapped, ok := MapCapabilityError(ctx, err, "squad.page"); ok {
			return nil, mapped
		}
		slog.ErrorContext(ctx, "squad_v2: erreur service",
			"player", gamertag, "title_slug", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "squad_v2_error", err.Error())
	}

	return &squadV2PageOutput{Body: resp}, nil
}

// parseSquadV2Teammates valide et découpe la liste CSV de coéquipiers.
// Retourne nil (et pas d'erreur) si le paramètre est absent ou vide.
func parseSquadV2Teammates(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		gt := strings.TrimSpace(p)
		if gt == "" {
			continue
		}
		out = append(out, gt)
	}
	if len(out) > squadV2MaxTeammates {
		return nil, fmt.Errorf("max %d coéquipiers, %d fournis", squadV2MaxTeammates, len(out))
	}
	return out, nil
}

// parseSquadV2Period valide la période en s'appuyant sur temporal.Period.
// Vide → temporal.PeriodAll (pas de filtrage temporel).
func parseSquadV2Period(raw string) (temporal.Period, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return temporal.PeriodAll, nil
	}
	p := temporal.Period(raw)
	if !p.IsValid() {
		return "", fmt.Errorf("période %q inconnue (admises : all, 2y, 1y, 1m, 1w)", raw)
	}
	return p, nil
}

// parseCSVFilter découpe un paramètre CSV en slice de strings non-vides.
// Retourne nil si le paramètre est absent ou entièrement vide.
func parseCSVFilter(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
