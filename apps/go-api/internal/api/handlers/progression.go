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
// Réf : ADR 0014.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// ProgressionHandler regroupe les endpoints de progression V2.
//
// Pas de branche « démo » ici : depuis le 2026-07-26, `seed-demo` écrit de vraies
// séries (streak_history), records et jalons dans les bases démo — cf.
// internal/ops/seed_demo_prestige.go. L'ancien fixture read-time de streaks
// masquait ces lignes et affichait des compteurs sans rapport avec le corpus.
type ProgressionHandler struct {
	svc       port.ProgressionService
	titleSlug string // typ. "halo_infinite", utilisé pour filtrer catalog/queries
}

// NewProgressionHandler construit le handler.
func NewProgressionHandler(svc port.ProgressionService, titleSlug string) *ProgressionHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &ProgressionHandler{svc: svc, titleSlug: titleSlug}
}

// Mount enregistre les 3 routes via Huma (Phase 3b) sur le sous-routeur chi
// (qui inclut déjà le préfixe /players/{player_slug} + son middleware
// ownership/title). humacore.NewAPI lit le path param parent {player_slug} et
// hérite du middleware du sous-groupe (cf. TestHumaNestedSubrouterProbe).
func (h *ProgressionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/streaks", h.handleStreaks, humacore.Op("listProgressionStreaks", "Liste des streaks du joueur (active + historique)", "progression"))
	huma.Get(api, "/records", h.handleRecords, humacore.Op("listProgressionRecords", "PB courants + timeline des records battus", "progression"))
	huma.Get(api, "/milestones", h.handleMilestones, humacore.Op("listProgressionMilestones", "Catalogue de milestones + statut Earned par milestone", "progression"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// progPlayerInput : path param parent {player_slug} (streaks + milestones).
type progPlayerInput struct {
	PlayerSlug string `path:"player_slug"`
}

// progRecordsInput : {player_slug} + ?history_limit= (records).
type progRecordsInput struct {
	PlayerSlug   string `path:"player_slug"`
	HistoryLimit int    `query:"history_limit" maximum:"200"`
}

type streaksHumaOutput struct{ Body streaksResponse }
type recordsHumaOutput struct{ Body recordsResponse }
type milestonesHumaOutput struct{ Body milestonesResponse }

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
	Title     string     `json:"title"` // dans la langue de la requête (repli `en`)
	Icon      string     `json:"icon,omitempty"`
	Condition string     `json:"condition,omitempty"` // description lisible dans la langue de la requête ; jamais la formule technique
	Earned    bool       `json:"earned"`
	EarnedAt  *time.Time `json:"earned_at,omitempty"`
}

type milestonesResponse struct {
	Items []milestoneDTO `json:"items"`
}

// ─── Endpoints ─────────────────────────────────────────────────────────────

// handleStreaks : GET /streaks → toutes les streaks du joueur (active + historique).
func (h *ProgressionHandler) handleStreaks(ctx context.Context, in *progPlayerInput) (*streaksHumaOutput, error) {
	list, err := h.svc.Streaks(ctx, in.PlayerSlug, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		return nil, progressionHTTPError(ctx, err)
	}
	items := make([]streakDTO, 0, len(list))
	for _, s := range list {
		items = append(items, toStreakDTO(s))
	}
	return &streaksHumaOutput{Body: streaksResponse{Items: items}}, nil
}

// handleRecords : GET /records → PB courants + timeline.
// Query param optionnel history_limit (défaut 50 ; le schéma refuse au-delà de 200).
func (h *ProgressionHandler) handleRecords(ctx context.Context, in *progRecordsInput) (*recordsHumaOutput, error) {
	limit := in.HistoryLimit
	if limit <= 0 {
		limit = 50
	}
	rec, err := h.svc.Records(ctx, in.PlayerSlug, requestTitleSlug(ctx, h.titleSlug), limit)
	if err != nil {
		return nil, progressionHTTPError(ctx, err)
	}
	resp := recordsResponse{
		PersonalBests: make([]personalBestDTO, 0, len(rec.PersonalBests)),
		History:       make([]recordHistoryDTO, 0, len(rec.History)),
	}
	for _, pb := range rec.PersonalBests {
		resp.PersonalBests = append(resp.PersonalBests, toPBDTO(pb))
	}
	for _, hh := range rec.History {
		resp.History = append(resp.History, toHistoryDTO(hh))
	}
	return &recordsHumaOutput{Body: resp}, nil
}

// handleMilestones : GET /milestones → catalogue du titre + statut Earned.
func (h *ProgressionHandler) handleMilestones(ctx context.Context, in *progPlayerInput) (*milestonesHumaOutput, error) {
	list, err := h.svc.Milestones(ctx, in.PlayerSlug, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		return nil, progressionHTTPError(ctx, err)
	}
	loc := i18n.FromContext(ctx)
	items := make([]milestoneDTO, 0, len(list))
	for _, m := range list {
		dto := toMilestoneDTO(m.Entry, loc)
		dto.Earned = m.Earned
		dto.EarnedAt = m.EarnedAt
		items = append(items, dto)
	}
	return &milestonesHumaOutput{Body: milestonesResponse{Items: items}}, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// progressionErrorCodes : le code 500 de chaque échec de lecture de la progression.
var progressionErrorCodes = map[port.ProgressionErrorKind]string{
	port.ProgErrListStreaks:       "list_streaks_error",
	port.ProgErrListPersonalBests: "list_pbs_error",
	port.ProgErrListRecordHistory: "list_history_error",
	port.ProgErrListCatalog:       "list_catalog_error",
	port.ProgErrListEarned:        "list_earned_error",
}

// progressionHTTPError traduit un échec d'un service de progression, de campagne ou de profil :
// joueur introuvable = 404 player_not_found (contrat préservé), échec de lecture = 500 au code de sa
// nature. Une erreur d'un autre type est une panne : 500.
func progressionHTTPError(ctx context.Context, err error) error {
	var pe *port.ProgressionError
	if !errors.As(err, &pe) {
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "progression_error", err)
	}
	if pe.Kind == port.ProgErrPlayerNotFound {
		return humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", pe.Err)
	}
	if code, ok := progressionErrorCodes[pe.Kind]; ok {
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, code, pe.Err)
	}
	return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "progression_error", pe.Err)
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

func toMilestoneDTO(c milestones.CatalogEntry, loc i18n.Locale) milestoneDTO {
	// A9 : on sert la description lisible dans la langue de la requête, jamais la formule
	// technique c.Condition. Vide si le jalon n'a pas de condition → le front n'affiche rien.
	dto := milestoneDTO{
		ID:        c.ID,
		Metric:    c.Metric,
		Threshold: c.Threshold,
		Title:     c.Title.Pick(loc),
		Icon:      c.Icon,
	}
	dto.Condition = c.ConditionText.PickOptional(loc)
	return dto
}
