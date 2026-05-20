// Package handlers — progression.go : endpoints HTTP de la couche V2 Ascension.
//
// Routes exposées (sous /api/v1/players/{player_slug}/) :
//   - GET /streaks    : liste des streaks (active + historique) pour ce joueur.
//   - GET /records    : PB courants (player_records) + timeline (record_history).
//   - GET /milestones : catalogue du titre + statut Earned par milestone.
//
// Pas de POST/DELETE/PATCH — la couche V2 émet ses données via le hook
// post-sync (cf. post_sync_progression.go). Les endpoints sont purement
// lecture pour l'UI.
//
// Réf : .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1 (handlers).
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// ProgressionResolver retourne le PlayerDB pour un slug donné.
// Implémenté par la registry (config.ResolvePlayer wrappé).
type ProgressionResolver func(ctx context.Context, slug string) (*duckdb.PlayerDB, error)

// ProgressionHandler regroupe les endpoints de progression V2.
type ProgressionHandler struct {
	resolve   ProgressionResolver
	titleSlug string // typ. "halo_infinite", utilisé pour filtrer catalog/queries
}

// NewProgressionHandler construit le handler.
func NewProgressionHandler(resolve ProgressionResolver, titleSlug string) *ProgressionHandler {
	if titleSlug == "" {
		titleSlug = "halo_infinite"
	}
	return &ProgressionHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre les 3 routes sur un router chi sous-monté (le sub-router
// inclut déjà le préfixe /players/{player_slug}).
func (h *ProgressionHandler) Mount(r chi.Router) {
	r.Get("/streaks", h.ListStreaks)
	r.Get("/records", h.ListRecords)
	r.Get("/milestones", h.ListMilestones)
}

// ─── DTOs réponse ──────────────────────────────────────────────────────────

// streakDTO est la projection JSON d'une streak.
type streakDTO struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	StartedAt        time.Time  `json:"started_at"`
	CurrentLength    int        `json:"current_length"`
	BestLength       int        `json:"best_length"`
	LastIncrementAt  *time.Time `json:"last_increment_at,omitempty"`
	Threshold        *float64   `json:"threshold,omitempty"`
	ShieldsUsed      int        `json:"shields_used"`
	ShieldsAvailable int        `json:"shields_available"`
	Status           string     `json:"status"`
	BrokenAt         *time.Time `json:"broken_at,omitempty"`
	PPMultiplier     float64    `json:"pp_multiplier"`
}

type streaksResponse struct {
	Items []streakDTO `json:"items"`
}

// personalBestDTO est la projection JSON d'un PB.
type personalBestDTO struct {
	Metric             string     `json:"metric"`
	Period             string     `json:"period"`
	Value              float64    `json:"value"`
	AchievedAt         *time.Time `json:"achieved_at,omitempty"`
	AchievedMatchID    string     `json:"achieved_match_id,omitempty"`
	PreviousValue      *float64   `json:"previous_value,omitempty"`
	PreviousAchievedAt *time.Time `json:"previous_achieved_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type recordHistoryDTO struct {
	ID         string    `json:"id"`
	Metric     string    `json:"metric"`
	Period     string    `json:"period"`
	Value      float64   `json:"value"`
	AchievedAt time.Time `json:"achieved_at"`
}

type recordsResponse struct {
	PersonalBests []personalBestDTO  `json:"personal_bests"`
	History       []recordHistoryDTO `json:"history"`
}

// milestoneDTO joint catalogue + earned pour 1 entrée.
type milestoneDTO struct {
	ID        string     `json:"id"`
	Metric    string     `json:"metric"`
	Threshold float64    `json:"threshold"`
	TitleEN   string     `json:"title_en"`
	TitleFR   string     `json:"title_fr"`
	Icon      string     `json:"icon,omitempty"`
	Condition string     `json:"condition,omitempty"`
	Earned    bool       `json:"earned"`
	EarnedAt  *time.Time `json:"earned_at,omitempty"`
}

type milestonesResponse struct {
	Items []milestoneDTO `json:"items"`
}

// ─── Endpoints ─────────────────────────────────────────────────────────────

// ListStreaks : GET /streaks → toutes les streaks du joueur (active + historique).
func (h *ProgressionHandler) ListStreaks(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404(w, r)
	if !ok {
		return
	}
	repo := duckdb.NewStreaksRepo(pdb.Player)
	list, err := repo.List(r.Context(), pdb.XUID, h.titleSlug)
	if err != nil {
		slog.WarnContext(r.Context(), "progression: list streaks", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_streaks_error", err.Error())
		return
	}
	items := make([]streakDTO, 0, len(list))
	for _, s := range list {
		items = append(items, toStreakDTO(s))
	}
	writeJSON(w, http.StatusOK, streaksResponse{Items: items})
}

// ListRecords : GET /records → PB courants + timeline.
//
// Query params optionnels :
//   - history_limit : limite l'historique (défaut 50, max 200)
func (h *ProgressionHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404(w, r)
	if !ok {
		return
	}
	limit := atoi(r.URL.Query().Get("history_limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	pbRepo := duckdb.NewPersonalRecordsRepo(pdb)
	pbList, err := pbRepo.ListByXUID(r.Context(), pdb.XUID)
	if err != nil {
		slog.WarnContext(r.Context(), "progression: list PBs", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_pbs_error", err.Error())
		return
	}

	historyRepo := duckdb.NewRecordHistoryRepo(pdb.Player)
	histList, err := historyRepo.ListRecent(r.Context(), pdb.XUID, h.titleSlug, limit)
	if err != nil {
		slog.WarnContext(r.Context(), "progression: list record history", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_history_error", err.Error())
		return
	}

	resp := recordsResponse{
		PersonalBests: make([]personalBestDTO, 0, len(pbList)),
		History:       make([]recordHistoryDTO, 0, len(histList)),
	}
	for _, pb := range pbList {
		resp.PersonalBests = append(resp.PersonalBests, toPBDTO(pb))
	}
	for _, h := range histList {
		resp.History = append(resp.History, toHistoryDTO(h))
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListMilestones : GET /milestones → catalogue du titre + statut Earned.
func (h *ProgressionHandler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404(w, r)
	if !ok {
		return
	}

	catalogRepo := duckdb.NewMilestoneCatalogRepo(pdb.Metadata)
	catalog, err := catalogRepo.ListByTitle(r.Context(), h.titleSlug)
	if err != nil {
		slog.WarnContext(r.Context(), "progression: list catalog", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_catalog_error", err.Error())
		return
	}

	earnedRepo := duckdb.NewMilestoneEarnedRepo(pdb.Player)
	earnedList, err := earnedRepo.ListByUser(r.Context(), pdb.XUID, h.titleSlug)
	if err != nil {
		slog.WarnContext(r.Context(), "progression: list earned", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_earned_error", err.Error())
		return
	}
	earnedByID := make(map[string]time.Time, len(earnedList))
	for _, e := range earnedList {
		earnedByID[e.MilestoneID] = e.EarnedAt
	}

	items := make([]milestoneDTO, 0, len(catalog))
	for _, c := range catalog {
		dto := toMilestoneDTO(c)
		if at, ok := earnedByID[c.ID]; ok {
			dto.Earned = true
			t := at
			dto.EarnedAt = &t
		}
		items = append(items, dto)
	}
	writeJSON(w, http.StatusOK, milestonesResponse{Items: items})
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// resolveOr404 résout le slug courant ou écrit 404. Pattern aligné sur
// NotificationsHandler.resolve.
func (h *ProgressionHandler) resolveOr404(w http.ResponseWriter, r *http.Request) (*duckdb.PlayerDB, bool) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := h.resolve(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return nil, false
	}
	return pdb, true
}

func toStreakDTO(s streaks.Streak) streakDTO {
	return streakDTO{
		ID:               s.ID,
		Type:             string(s.Type),
		StartedAt:        s.StartedAt,
		CurrentLength:    s.CurrentLength,
		BestLength:       s.BestLength,
		LastIncrementAt:  s.LastIncrementAt,
		Threshold:        s.Threshold,
		ShieldsUsed:      s.ShieldsUsed,
		ShieldsAvailable: s.ShieldsAvailable,
		Status:           string(s.Status),
		BrokenAt:         s.BrokenAt,
		PPMultiplier:     streaks.PPMultiplier(s.CurrentLength),
	}
}

func toPBDTO(pb records.PersonalRecord) personalBestDTO {
	return personalBestDTO{
		Metric:             pb.Metric,
		Period:             string(pb.Period),
		Value:              pb.Value,
		AchievedAt:         pb.AchievedAt,
		AchievedMatchID:    pb.AchievedMatchID,
		PreviousValue:      pb.PreviousValue,
		PreviousAchievedAt: pb.PreviousAchievedAt,
		UpdatedAt:          pb.UpdatedAt,
	}
}

func toHistoryDTO(h records.RecordHistory) recordHistoryDTO {
	return recordHistoryDTO{
		ID:         h.ID,
		Metric:     h.Metric,
		Period:     string(h.Period),
		Value:      h.Value,
		AchievedAt: h.AchievedAt,
	}
}

func toMilestoneDTO(c milestones.CatalogEntry) milestoneDTO {
	return milestoneDTO{
		ID:        c.ID,
		Metric:    c.Metric,
		Threshold: c.Threshold,
		TitleEN:   c.TitleEN,
		TitleFR:   c.TitleFR,
		Icon:      c.Icon,
		Condition: c.Condition,
	}
}
