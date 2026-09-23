// Package handlers — CareerHandler : GET .../pages/career[/top-matches|/encounters|/highlight-matches|/top-encounters|/rivals|/csrs].
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 7 GET via huma.Get. Logique métier
// inchangée (CareerService + MatchHistoryService), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ServiceFactory est un type générique pour les factory de service player-scoped.
// Chaque handler reçoit une factory qui résout le slug → service injecté.
type ServiceFactory[S any] func(ctx context.Context, slug string) (S, error)

// ContextFactory est une factory qui retourne un service + XUID + Gamertag.
// Utilisé par les handlers qui ont besoin du contexte joueur en plus du service.
type ContextFactory[S any] func(ctx context.Context, slug string) (svc S, xuid, gamertag string, err error)

// CareerHandler gère les endpoints de la page Carrière.
//
// Highlight matches utilisent MatchHistoryService pour enrichir les match_ids
// de Q9b en ExplorerMatchesRow complets (réutilise la chaîne enrichRow déjà
// éprouvée par l'Explorer).
type CareerHandler struct {
	newSvc   ServiceFactory[port.CareerService]
	newMHSvc ContextFactory[port.MatchHistoryService] // optionnel — nécessaire pour highlight-matches
}

// NewCareerHandler crée un CareerHandler avec une factory de service injectée.
// newMHSvc peut être nil — dans ce cas l'endpoint highlight-matches retourne
// une 503 propre (la page Carrière reste fonctionnelle pour le reste).
func NewCareerHandler(
	newSvc ServiceFactory[port.CareerService],
	newMHSvc ContextFactory[port.MatchHistoryService],
) *CareerHandler {
	return &CareerHandler{newSvc: newSvc, newMHSvc: newMHSvc}
}

// Mount enregistre les 7 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *CareerHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/career", h.handleGetCareer, humacore.Op("getCareerPage", "Page Carrière complète", "career"))
	huma.Get(api, "/pages/career/top-matches", h.handleGetTopMatches, humacore.Op("getCareerTopMatches", "Top matchs du joueur", "career"))
	huma.Get(api, "/pages/career/encounters", h.handleGetEncounters, humacore.Op("getCareerEncounters", "Encounters du joueur", "career"))
	huma.Get(api, "/pages/career/highlight-matches", h.handleGetHighlightMatches, humacore.Op("getCareerHighlightMatches", "Matchs marquants — 15 best + 15 worst (format Explorer)", "career"))
	huma.Get(api, "/pages/career/top-encounters", h.handleGetTopEncountersRich, humacore.Op("getCareerTopEncounters", "Joueurs les plus croisés (hors amis)", "career"))
	huma.Get(api, "/pages/career/rivals", h.handleGetRivals, humacore.Op("getCareerRivals", "Top némésis + Top souffre-douleur", "career"))
	huma.Get(api, "/pages/career/csrs", h.handleGetCareerCSRs, humacore.Op("getCareerCSRs", "Classements CSR par playlist (snapshots current / season / all-time)", "career"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// careerPlayerInput : path param parent {player_slug} (endpoints sans query).
type careerPlayerInput struct {
	PlayerSlug string `path:"player_slug"`
}

// careerHighlightInput : {player_slug} + query params de filtrage (tous tolérés
// vides — parseHighlightFilterInput applique les normalisations côté service).
type careerHighlightInput struct {
	PlayerSlug    string `path:"player_slug"`
	Experience    string `query:"experience"`
	SeasonIDs     string `query:"season_ids"`
	ModeUIs       string `query:"mode_uis"`
	PlaylistNames string `query:"playlist_names"`
}

// careerCSRsInput : {player_slug} + ?season= optionnel.
type careerCSRsInput struct {
	PlayerSlug string `path:"player_slug"`
	Season     string `query:"season"`
}

type careerPageOutput struct{ Body domain.CareerPageResponse }
type careerTopMatchesOutput struct {
	Body domain.CareerTopMatchesResponse
}
type careerEncountersOutput struct {
	Body domain.CareerEncountersResponse
}
type careerHighlightOutput struct {
	Body domain.CareerHighlightMatchesResponse
}
type careerTopEncountersOutput struct {
	Body domain.CareerTopEncountersResponse
}
type careerRivalsOutput struct{ Body domain.CareerRivalsResponse }
type careerCSRsOutput struct{ Body domain.CareerCSRResponse }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetCareer retourne la réponse complète de la page Carrière.
func (h *CareerHandler) handleGetCareer(ctx context.Context, in *careerPlayerInput) (*careerPageOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetCareerPage(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "career_error", err.Error())
	}
	return &careerPageOutput{Body: resp}, nil
}

// handleGetTopMatches retourne les top/pires matchs du joueur (format léger legacy).
func (h *CareerHandler) handleGetTopMatches(ctx context.Context, in *careerPlayerInput) (*careerTopMatchesOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetTopMatches(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "top_matches_error", err.Error())
	}
	return &careerTopMatchesOutput{Body: resp}, nil
}

// handleGetEncounters retourne les joueurs les plus fréquemment croisés (format léger).
func (h *CareerHandler) handleGetEncounters(ctx context.Context, in *careerPlayerInput) (*careerEncountersOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetEncounters(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "encounters_error", err.Error())
	}
	return &careerEncountersOutput{Body: resp}, nil
}

// handleGetHighlightMatches retourne les 15 meilleurs et 15 pires matchs au format
// ExplorerMatchesRow (mêmes 21 colonnes que la page Explorer) avec les cascade
// counts pour les filtres Expérience / Saisons.
//
// GET /api/v1/players/{player_slug}/pages/career/highlight-matches
//
// Query params optionnels :
//   - experience : "all" | "ranked" | "unranked" (défaut "all")
//   - season_ids : CSV des IDs de saison à filtrer (ex. "season6,season7")
func (h *CareerHandler) handleGetHighlightMatches(ctx context.Context, in *careerHighlightInput) (*careerHighlightOutput, error) {
	if h.newMHSvc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "match_history_unavailable",
			"highlight-matches requires MatchHistoryService factory")
	}

	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	mhSvc, _, _, err := h.newMHSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	input := parseHighlightFilterInput(in.highlightQuery())
	data, err := svc.GetHighlightMatchIDs(ctx, input)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "highlight_matches_error", err.Error())
	}

	var bestRows, worstRows []domain.HighlightMatchIDRow
	for _, row := range data.Rows {
		switch row.Section {
		case 1:
			bestRows = append(bestRows, row)
		case 2:
			worstRows = append(worstRows, row)
		}
	}

	bestMatches, worstMatches, err := enrichHighlightSections(ctx, mhSvc, bestRows, worstRows)
	if err != nil {
		// Même code qu'avant la requête unique : l'échec était celui de la première
		// section non vide.
		code := "highlight_best_enrich_error"
		if len(bestRows) == 0 {
			code = "highlight_worst_enrich_error"
		}
		return nil, humacore.NewError(http.StatusInternalServerError, code, err.Error())
	}

	return &careerHighlightOutput{Body: domain.CareerHighlightMatchesResponse{
		BestMatches:         bestMatches,
		WorstMatches:        worstMatches,
		AvailableExperience: data.AvailableExperience,
		AvailableSeasons:    data.AvailableSeasons,
		AvailableModes:      data.AvailableModes,
		AvailablePlaylists:  data.AvailablePlaylists,
	}}, nil
}

// highlightQuery reconstruit un url.Values depuis les champs query de l'input
// pour réutiliser parseHighlightFilterInput à l'identique (CSV → []string).
func (in *careerHighlightInput) highlightQuery() url.Values {
	q := url.Values{}
	q.Set("experience", in.Experience)
	q.Set("season_ids", in.SeasonIDs)
	q.Set("mode_uis", in.ModeUIs)
	q.Set("playlist_names", in.PlaylistNames)
	return q
}

// parseHighlightFilterInput extrait les query params en domain.HighlightFilterInput.
// Tolère les valeurs absentes ou vides — le service applique les normalisations.
// Params CSV : season_ids, mode_uis, playlist_names.
func parseHighlightFilterInput(q url.Values) domain.HighlightFilterInput {
	in := domain.HighlightFilterInput{
		Experience: q.Get("experience"),
	}
	for _, raw := range []struct {
		param string
		dest  *[]string
	}{
		{"season_ids", &in.SeasonIDs},
		{"mode_uis", &in.ModeUIs},
		{"playlist_names", &in.PlaylistNames},
	} {
		if v := q.Get(raw.param); v != "" {
			for _, s := range strings.Split(v, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					*raw.dest = append(*raw.dest, s)
				}
			}
		}
	}
	return in
}

// handleGetTopEncountersRich retourne les 10 joueurs les plus croisés au niveau
// carrière (hors amis) au format MatchEncounterRow (mêmes 8 colonnes que le
// tableau "Historique de rencontre" de Match View).
//
// GET /api/v1/players/{player_slug}/pages/career/top-encounters
func (h *CareerHandler) handleGetTopEncountersRich(ctx context.Context, in *careerPlayerInput) (*careerTopEncountersOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetTopEncounters(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "top_encounters_error", err.Error())
	}
	return &careerTopEncountersOutput{Body: resp}, nil
}

// handleGetRivals retourne les top némésis (deaths DESC) et top souffre-douleur
// (frags DESC), 10 chacun.
//
// GET /api/v1/players/{player_slug}/pages/career/rivals
func (h *CareerHandler) handleGetRivals(ctx context.Context, in *careerPlayerInput) (*careerRivalsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetRivals(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "rivals_error", err.Error())
	}
	return &careerRivalsOutput{Body: resp}, nil
}

// handleGetCareerCSRs retourne les classements CSR par playlist.
// GET /players/{player_slug}/pages/career/csrs
func (h *CareerHandler) handleGetCareerCSRs(ctx context.Context, in *careerCSRsInput) (*careerCSRsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	// ?season= optionnel : saison CSR à afficher (vide → saison courante).
	season := strings.TrimSpace(in.Season)
	resp, err := svc.GetCareerCSRs(ctx, season)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "csrs_error", err.Error())
	}
	return &careerCSRsOutput{Body: resp}, nil
}

// resolve résout le slug courant en CareerService ou renvoie une erreur Huma 404
// (contrat préservé : {code:player_not_found}).
func (h *CareerHandler) resolve(ctx context.Context, slug string) (port.CareerService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}

// enrichHighlightSections enrichit les deux sections (meilleurs, pires matchs) en
// UNE requête MatchHistoryService.GetPage sur l'union de leurs identifiants, puis
// projette chaque section en ExplorerMatchesRow (plan perf 2026-09-23, D5b.6 :
// une requête par section chargeait deux fois tout l'historique). L'enrichissement
// d'une ligne ne dépend que d'elle et de l'historique complet (taux de victoire par
// carte, placements) : l'union rend les mêmes lignes que deux requêtes séparées.
func enrichHighlightSections(
	ctx context.Context,
	mhSvc port.MatchHistoryService,
	best, worst []domain.HighlightMatchIDRow,
) (bestOut, worstOut []domain.ExplorerMatchesRow, err error) {
	matchIDs := highlightMatchIDs(best, worst)
	if len(matchIDs) == 0 {
		return []domain.ExplorerMatchesRow{}, []domain.ExplorerMatchesRow{}, nil
	}
	req := domain.MatchHistoryQueryRequest{
		MatchIDs: matchIDs,
		Pagination: domain.PaginationRequest{
			Page:     1,
			PageSize: len(matchIDs),
		},
	}
	resp, err := mhSvc.GetPage(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]domain.MatchHistoryRow, len(resp.Table.Items))
	for _, item := range resp.Table.Items {
		byID[item.MatchID] = item
	}
	return projectHighlightRows(best, byID), projectHighlightRows(worst, byID), nil
}

// highlightMatchIDs rend l'union ordonnée (sans doublon) des identifiants des deux
// sections : un match peut figurer dans les deux quand l'historique est court.
func highlightMatchIDs(sections ...[]domain.HighlightMatchIDRow) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, rows := range sections {
		for _, r := range rows {
			if _, dup := seen[r.MatchID]; dup {
				continue
			}
			seen[r.MatchID] = struct{}{}
			ids = append(ids, r.MatchID)
		}
	}
	return ids
}

// projectHighlightRows projette une section en ExplorerMatchesRow. Préserve l'ordre
// d'entrée (Q9b trie par dominance prio + perf, ordre que MatchHistoryService.sortItems
// casserait sinon) et propage HadBotTeammate depuis la row source (le service de
// match history ne lit pas ce flag).
func projectHighlightRows(rows []domain.HighlightMatchIDRow, byID map[string]domain.MatchHistoryRow) []domain.ExplorerMatchesRow {
	out := make([]domain.ExplorerMatchesRow, 0, len(rows))
	for _, src := range rows {
		item, ok := byID[src.MatchID]
		if !ok {
			continue // match_id absent du whitelist (filtre is_excluded ou autre)
		}
		row := BuildExplorerRowFromMatchHistory(item)
		row.HadBotTeammate = src.HadBotTeammate
		out = append(out, row)
	}
	return out
}
