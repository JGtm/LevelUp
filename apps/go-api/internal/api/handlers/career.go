// Package handlers — CareerHandler : GET .../pages/career[/top-matches|/encounters|/highlight-matches|/top-encounters|/rivals].
package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

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

// GetCareer retourne la réponse complète de la page Carrière.
func (h *CareerHandler) GetCareer(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetCareerPage(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "career_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetTopMatches retourne les top/pires matchs du joueur (format léger legacy).
func (h *CareerHandler) GetTopMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetTopMatches(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "top_matches_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetEncounters retourne les joueurs les plus fréquemment croisés (format léger).
func (h *CareerHandler) GetEncounters(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetEncounters(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "encounters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetHighlightMatches retourne les 15 meilleurs et 15 pires matchs au format
// ExplorerMatchesRow (mêmes 21 colonnes que la page Explorer) avec les cascade
// counts pour les filtres Expérience / Saisons.
//
// GET /api/v1/players/{player_slug}/pages/career/highlight-matches
//
// Query params optionnels :
//   - experience : "all" | "ranked" | "unranked" (défaut "all")
//   - season_ids : CSV des IDs de saison à filtrer (ex. "season6,season7")
func (h *CareerHandler) GetHighlightMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")

	if h.newMHSvc == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "match_history_unavailable",
			"highlight-matches requires MatchHistoryService factory")
		return
	}

	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	mhSvc, _, _, err := h.newMHSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	input := parseHighlightFilterInput(r.URL.Query())
	data, err := svc.GetHighlightMatchIDs(r.Context(), input)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "highlight_matches_error", err.Error())
		return
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

	bestMatches, err := enrichHighlightMatches(r.Context(), mhSvc, bestRows)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "highlight_best_enrich_error", err.Error())
		return
	}
	worstMatches, err := enrichHighlightMatches(r.Context(), mhSvc, worstRows)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "highlight_worst_enrich_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, domain.CareerHighlightMatchesResponse{
		BestMatches:         bestMatches,
		WorstMatches:        worstMatches,
		AvailableExperience: data.AvailableExperience,
		AvailableSeasons:    data.AvailableSeasons,
		AvailableModes:      data.AvailableModes,
		AvailablePlaylists:  data.AvailablePlaylists,
	})
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

// GetTopEncountersRich retourne les 10 joueurs les plus croisés au niveau
// carrière (hors amis) au format MatchEncounterRow (mêmes 8 colonnes que le
// tableau "Historique de rencontre" de Match View).
//
// GET /api/v1/players/{player_slug}/pages/career/top-encounters
func (h *CareerHandler) GetTopEncountersRich(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetTopEncounters(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "top_encounters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetRivals retourne les top némésis (deaths DESC) et top souffre-douleur
// (frags DESC), 10 chacun.
//
// GET /api/v1/players/{player_slug}/pages/career/rivals
func (h *CareerHandler) GetRivals(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetRivals(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "rivals_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetCareerCSRs retourne les classements CSR par playlist.
// GET /players/{player_slug}/pages/career/csrs
func (h *CareerHandler) GetCareerCSRs(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	// ?season= optionnel : saison CSR à afficher (vide → saison courante).
	season := strings.TrimSpace(r.URL.Query().Get("season"))
	resp, err := svc.GetCareerCSRs(r.Context(), season)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "csrs_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// enrichHighlightMatches enrichit une liste de rows highlight via
// MatchHistoryService.GetPage(MatchIDs=...) puis projette en ExplorerMatchesRow.
// Préserve l'ordre d'entrée (Q9b trie par dominance prio + perf, ordre que le
// MatchHistoryService.sortItems casserait sinon) et propage HadBotTeammate
// depuis la row source (le service de match history ne lit pas ce flag).
func enrichHighlightMatches(ctx context.Context, mhSvc port.MatchHistoryService, rows []domain.HighlightMatchIDRow) ([]domain.ExplorerMatchesRow, error) {
	if len(rows) == 0 {
		return []domain.ExplorerMatchesRow{}, nil
	}
	matchIDs := make([]string, len(rows))
	for i, r := range rows {
		matchIDs[i] = r.MatchID
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
		return nil, err
	}
	// Index par match_id pour réordonner selon la liste d'entrée (préserve
	// le tri Q9b dominance+perf que sortItems aurait écrasé).
	byID := make(map[string]domain.MatchHistoryRow, len(resp.Table.Items))
	for _, item := range resp.Table.Items {
		byID[item.MatchID] = item
	}
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
	return out, nil
}
