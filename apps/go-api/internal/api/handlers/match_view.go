// Package handlers — MatchViewHandler : GET .../matches/{match_id}.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 2 GET via huma.Get. Logique métier
// inchangée (MatchViewService), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// playlistOrSessionPattern : whitelist de chars pour les valeurs de filtres
// passés en query param. Empêche injection même si le repo utilise des
// paramètres préparés. Limite 64 chars (cf. plan Phase 2b §5).
var playlistOrSessionPattern = regexp.MustCompile(`^[A-Za-z0-9 _:.\-]{1,64}$`)

// xuidPattern : XUID Halo = entier décimal (jusqu'à 32 chars). Phase 2c
// (with_player). Mêmes contraintes que côté front (parseFilterSpecFromSearch).
var xuidPattern = regexp.MustCompile(`^\d{1,32}$`)

// parseNeighborsFilterSpec : extrait MatchFilterSpec depuis r.URL.Query().
// Tout filtre invalide est silencieusement ignoré + log warning. Jamais 400.
//
// Les noms de query params utilisés (côté front, voir useNavigateToMatch
// Phase 2b) :
//   - playlist (PlaylistName)
//   - mode (ModeCategory)
//   - from / to (DateFrom / DateTo, ISO 8601 / RFC3339)
//   - session (SessionID)
//   - outcome (Outcome — whitelist win/loss/draw/dnf)
func parseNeighborsFilterSpec(r *http.Request) *domain.MatchFilterSpec {
	q := r.URL.Query()
	spec := &domain.MatchFilterSpec{}
	ctx := r.Context()

	// playlist / mode : multi-valeurs séparées par virgule (Phase 3).
	if vals := parseCsvFilterParam(ctx, q.Get("playlist"), "playlist"); len(vals) > 0 {
		spec.PlaylistNames = vals
	}
	if vals := parseCsvFilterParam(ctx, q.Get("mode"), "mode"); len(vals) > 0 {
		spec.ModeCategories = vals
	}
	if v := strings.TrimSpace(q.Get("session")); v != "" {
		if playlistOrSessionPattern.MatchString(v) {
			spec.SessionID = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "session", "value", v)
		}
	}
	if v := strings.TrimSpace(q.Get("outcome")); v != "" {
		if analysis.IsValidOutcomeLabel(v) {
			spec.Outcome = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "outcome", "value", v)
		}
	}
	if v := strings.TrimSpace(q.Get("from")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			spec.DateFrom = &t
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "from", "value", v, "err", err)
		}
	}
	if v := strings.TrimSpace(q.Get("to")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			spec.DateTo = &t
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "to", "value", v, "err", err)
		}
	}
	if v := strings.TrimSpace(q.Get("with_player")); v != "" {
		if xuidPattern.MatchString(v) {
			spec.WithPlayerXuid = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "with_player", "value", v)
		}
	}
	if spec.IsEmpty() {
		return nil
	}
	return spec
}

// parseCsvFilterParam : découpe une valeur de query param en valeurs multiples
// (séparateur virgule), trim + valide chacune via la whitelist. Les valeurs
// invalides sont ignorées individuellement (log warn). Retourne nil si aucune
// valeur valide — préserve le comportement mono-valeur (1 valeur → slice de 1).
func parseCsvFilterParam(ctx context.Context, raw, param string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if playlistOrSessionPattern.MatchString(v) {
			out = append(out, v)
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", param, "value", v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MatchViewHandler gère GET /players/{player_slug}/matches/{match_id}.
type MatchViewHandler struct {
	newSvc        ServiceFactory[port.MatchViewService]
	settingsStore *settings.Store
	repoRoot      string
}

// NewMatchViewHandler crée un MatchViewHandler.
func NewMatchViewHandler(newSvc ServiceFactory[port.MatchViewService]) *MatchViewHandler {
	return &MatchViewHandler{newSvc: newSvc}
}

// WithMediaURLs câble la transformation des chemins média de l'onglet médias
// (file_path + thumbnail_url) en URLs servables /api/v1/.../media/files/...,
// réutilisant la logique de la galerie (cf. mediaStoredPathToURL dans media.go).
// Sans cet appel, les chemins restent bruts (chemin de test).
func (h *MatchViewHandler) WithMediaURLs(store *settings.Store, repoRoot string) *MatchViewHandler {
	h.settingsStore = store
	h.repoRoot = repoRoot
	return h
}

// Mount enregistre les 4 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *MatchViewHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}", h.handleGetMatchView, humacore.Op("getMatchView", "Détail complet d'un match", "match-view"))
	huma.Get(api, "/matches/{match_id}/neighbors", h.handleGetMatchNeighbors, humacore.Op("getMatchNeighbors", "Match précédent / suivant pour navigation", "match-view"))
	// Deux calques décodés du film : la timeline objectif et les positions keyframe.
	// Titre sans film → 503 capability_not_supported, jamais une 500.
	huma.Get(api, "/matches/{match_id}/objective-events", h.handleGetObjectiveEvents, humacore.Op("getMatchObjectiveEvents", "Timeline objectif mode-agnostique d'un match", "match-view"))
	huma.Get(api, "/matches/{match_id}/positions", h.handleGetMatchPositions, humacore.Op("getMatchPositions", "Positions joueurs aux images-clés d'un match", "match-view"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// matchViewInput : {player_slug} parent + {match_id}. match_id pris en STRING
// (parse maison ci-dessous) pour reproduire le contrat d'origine — un match_id
// vide renvoie 400 missing_match_id.
type matchViewInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
}

// matchNeighborsInput : {player_slug} parent + {match_id}. Les filtres de
// voisinage sont relus depuis la query brute via parseNeighborsFilterSpec, pour
// préserver à l'identique le parsing tolérant (jamais 400) — aucun query param
// n'est déclaré dans l'input (sinon Huma 422 sur valeur invalide).
type matchNeighborsInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
	request    *http.Request
}

// Resolve (interface huma.Resolver) reconstruit une *http.Request portant
// l'URL brute (query) + le contexte, pour réutiliser parseNeighborsFilterSpec
// à l'identique (qui lit r.URL.Query() + r.Context()).
func (in *matchNeighborsInput) Resolve(ctx huma.Context) []error {
	u := ctx.URL()
	in.request = (&http.Request{Method: http.MethodGet, URL: &u}).WithContext(ctx.Context())
	return nil
}

type matchViewOutput struct{ Body domain.MatchViewResponse }
type matchNeighborsOutput struct{ Body domain.MatchNeighbors }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetMatchView retourne la vue détaillée d'un match pour un joueur.
// GET /api/v1/players/{player_slug}/matches/{match_id}
func (h *MatchViewHandler) handleGetMatchView(ctx context.Context, in *matchViewInput) (*matchViewOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	matchID := in.MatchID
	if matchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}

	resp, err := svc.GetMatchView(ctx, matchID)
	if err != nil {
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "match_not_participant" {
			// Couche B (ADR 0029) : match existant mais joueur non-participant.
			return nil, humacore.NewError(http.StatusNotFound, "match_not_participant", apiErr.Message)
		}
		if errors.As(err, &apiErr) && apiErr.Code == "not_found" {
			return nil, humacore.NewError(http.StatusNotFound, "match_not_found", apiErr.Message)
		}
		// PAS DE RENIFLAGE DE CHAÎNE « no rows » ICI (retiré le 2026-08-29) : l'absence
		// est désormais TYPÉE par le service (domain.ErrNotFound, branche ci-dessus).
		// Cette branche textuelle était la dernière échappatoire par laquelle une panne
		// technique dont le message contenait « no rows » se déguisait en 404.
		return nil, humacore.NewError(http.StatusInternalServerError, "match_view_error", err.Error())
	}
	// Onglet médias : réécrire les chemins bruts en URLs servables (même
	// transformation que la galerie /pages/media). Sans ça les vignettes webp
	// et le média dans le lecteur tombent en 404 (chemins relatifs bruts).
	h.transformMatchMediaURLs(in.PlayerSlug, resp.MediaTab.MediaItems)
	return &matchViewOutput{Body: resp}, nil
}

// transformMatchMediaURLs réécrit file_path + thumbnail_url de chaque média en
// URLs servables /api/v1/.../media/files/..., comme transformMediaURLs côté
// galerie. No-op si la slice est vide. La slice est partagée avec resp, donc la
// mutation en place est visible à la sérialisation JSON.
func (h *MatchViewHandler) transformMatchMediaURLs(slug string, items []domain.MatchAssociatedMedia) {
	if len(items) == 0 {
		return
	}
	capturesBase := mediaCapturesBase(h.settingsStore)
	for i := range items {
		items[i].FilePath = mediaStoredPathToURL(slug, items[i].FilePath, capturesBase, h.repoRoot)
		if items[i].ThumbnailURL != nil {
			u := mediaStoredPathToURL(slug, *items[i].ThumbnailURL, capturesBase, h.repoRoot)
			items[i].ThumbnailURL = &u
		}
	}
}

// handleGetMatchNeighbors retourne les matchs adjacents pour la navigation prev/next.
// GET /api/v1/players/{player_slug}/matches/{match_id}/neighbors
func (h *MatchViewHandler) handleGetMatchNeighbors(ctx context.Context, in *matchNeighborsInput) (*matchNeighborsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	matchID := in.MatchID
	if matchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}

	spec := parseNeighborsFilterSpec(in.request)
	var resp domain.MatchNeighbors
	if spec != nil {
		resp, err = svc.GetMatchNeighborsFiltered(ctx, matchID, spec)
	} else {
		resp, err = svc.GetMatchNeighbors(ctx, matchID)
	}
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "neighbors_error", err.Error())
	}
	return &matchNeighborsOutput{Body: resp}, nil
}

// resolve résout le slug courant en MatchViewService ou renvoie une erreur Huma
// 404 (contrat préservé : {code:player_not_found}).
func (h *MatchViewHandler) resolve(ctx context.Context, slug string) (port.MatchViewService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}
