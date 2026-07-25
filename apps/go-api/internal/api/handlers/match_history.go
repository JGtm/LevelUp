// Package handlers — MatchHistoryHandler : POST .../pages/match-history/query
//
//	GET  .../pages/match-history/export
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST query via huma.Post. Logique métier
// inchangée (MatchHistoryService), seul le wrapping HTTP change. La route GET
// export reste servie en chi inline (CSV — non migrée).
//
// Le corps de query est lu via RawBody (pas de Body typé) pour reproduire
// EXACTEMENT le contrat de décodage d'origine : un JSON invalide renvoie
// 400 {invalid_body} (parse maison) et non le 422 de validation Huma qu'un Body
// typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MatchHistoryHandler gère POST .../pages/match-history/query
// et GET .../pages/match-history/export.
type MatchHistoryHandler struct {
	newSvc ContextFactory[port.MatchHistoryService]
}

// NewMatchHistoryHandler crée un MatchHistoryHandler.
func NewMatchHistoryHandler(newSvc ContextFactory[port.MatchHistoryService]) *MatchHistoryHandler {
	return &MatchHistoryHandler{newSvc: newSvc}
}

// Mount enregistre la route POST query via Huma sur le sous-routeur chi
// (préfixe /players/{player_slug} + middleware ownership/title hérités).
// La route GET export reste enregistrée en chi inline (server.go) — non migrée.
func (h *MatchHistoryHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/match-history/query", h.handleQuery)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// matchHistoryQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type matchHistoryQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type matchHistoryQueryOutput struct {
	Body domain.MatchHistoryPageResponse
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleQuery retourne la page d'historique paginée et filtrée.
// POST /api/v1/players/{player_slug}/pages/match-history/query
func (h *MatchHistoryHandler) handleQuery(ctx context.Context, in *matchHistoryQueryInput) (*matchHistoryQueryOutput, error) {
	svc, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.MatchHistoryQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}

	if err := req.Validate(); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_request", err.Error())
	}

	resp, err := svc.GetPage(ctx, req)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "match_history_error", err.Error())
	}

	// Générer le token d'export si demandé.
	if req.IncludeExportHint && resp.ExportHint != nil {
		token, err := encodeExportToken(req)
		if err == nil {
			resp.ExportHint.Token = &token
		}
	}

	return &matchHistoryQueryOutput{Body: resp}, nil
}

// Export retourne un CSV de l'historique des matchs filtrés.
// GET /api/v1/players/{player_slug}/pages/match-history/export?token=<base64url>
// Le token est généré par Query avec include_export_hint=true.
func (h *MatchHistoryHandler) Export(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_token", "paramètre token requis")
		return
	}

	req, err := decodeExportToken(token)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_token", "token invalide ou expiré")
		return
	}

	rows, err := svc.ExportCSV(r.Context(), req)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "export_error", err.Error())
		return
	}

	fileName := fmt.Sprintf("levelup_matches_%s.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))

	cw := csv.NewWriter(w)
	// En-tête CSV.
	_ = cw.Write([]string{
		"match_id", "start_time", "outcome", "score",
		"map", "mode", "playlist",
		"team_mmr", "enemy_mmr", "delta_mmr",
		"win_rate_hist", "performance_score",
		"avg_life", "match_url",
	})
	for _, row := range rows {
		teamMMR := formatOptFloat(row.TeamMMR)
		enemyMMR := formatOptFloat(row.EnemyMMR)
		deltaMMR := formatOptFloat(row.DeltaMMR)
		winRate := formatOptFloat(row.WinRateHist)
		perfScore := ""
		if row.PerformanceScoreRelative != nil {
			perfScore = fmt.Sprintf("%d", *row.PerformanceScoreRelative)
		}
		_ = cw.Write([]string{
			row.MatchID,
			row.StartTime.Format(time.RFC3339),
			row.OutcomeLabel,
			row.ScoreLabel,
			optStr(row.MapUI),
			optStr(row.ModeUI),
			optStr(row.PlaylistLabel),
			teamMMR, enemyMMR, deltaMMR,
			winRate, perfScore,
			row.AverageLifeMMSS,
			row.MatchURL,
		})
	}
	cw.Flush()
}

// ---------------------------------------------------------------------------
// Helpers token export (stateless — base64url JSON)
// ---------------------------------------------------------------------------

func encodeExportToken(req domain.MatchHistoryQueryRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func decodeExportToken(token string) (domain.MatchHistoryQueryRequest, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return domain.MatchHistoryQueryRequest{}, err
	}
	var req domain.MatchHistoryQueryRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return domain.MatchHistoryQueryRequest{}, err
	}
	return req, nil
}

func formatOptFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *v)
}

func optStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
