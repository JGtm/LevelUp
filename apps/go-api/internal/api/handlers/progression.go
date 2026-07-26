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

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// ProgressionResolver retourne le PlayerDB pour un slug donné.
// Implémenté par la registry (config.ResolvePlayer wrappé).
type ProgressionResolver func(ctx context.Context, slug string) (*duckdb.PlayerDB, error)

// ProgressionHandler regroupe les endpoints de progression V2.
//
// Pas de branche « démo » ici : depuis le 2026-07-26, `seed-demo` écrit de vraies
// séries (streak_history), records et jalons dans les bases démo — cf.
// internal/ops/seed_demo_prestige.go. L'ancien fixture read-time de streaks
// masquait ces lignes et affichait des compteurs sans rapport avec le corpus.
type ProgressionHandler struct {
	resolve   ProgressionResolver
	titleSlug string // typ. "halo_infinite", utilisé pour filtrer catalog/queries
}

// NewProgressionHandler construit le handler.
func NewProgressionHandler(resolve ProgressionResolver, titleSlug string) *ProgressionHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &ProgressionHandler{resolve: resolve, titleSlug: titleSlug}
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
	HistoryLimit int    `query:"history_limit"`
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
	ID          string     `json:"id"`
	Metric      string     `json:"metric"`
	Threshold   float64    `json:"threshold"`
	TitleEN     string     `json:"title_en"`
	TitleFR     string     `json:"title_fr"`
	Icon        string     `json:"icon,omitempty"`
	ConditionFR string     `json:"condition_fr,omitempty"`
	ConditionEN string     `json:"condition_en,omitempty"`
	Earned      bool       `json:"earned"`
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
}

type milestonesResponse struct {
	Items []milestoneDTO `json:"items"`
}

// ─── Endpoints ─────────────────────────────────────────────────────────────

// handleStreaks : GET /streaks → toutes les streaks du joueur (active + historique).
func (h *ProgressionHandler) handleStreaks(ctx context.Context, in *progPlayerInput) (*streaksHumaOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewStreaksRepo(pdb.Player)
	list, err := repo.List(ctx, pdb.XUID, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "progression: list streaks", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_streaks_error", err.Error())
	}
	items := make([]streakDTO, 0, len(list))
	for _, s := range list {
		items = append(items, toStreakDTO(s))
	}
	return &streaksHumaOutput{Body: streaksResponse{Items: items}}, nil
}

// handleRecords : GET /records → PB courants + timeline.
// Query param optionnel history_limit (défaut 50, max 200).
func (h *ProgressionHandler) handleRecords(ctx context.Context, in *progRecordsInput) (*recordsHumaOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	limit := in.HistoryLimit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	pbRepo := duckdb.NewPersonalRecordsRepo(pdb)
	pbList, err := pbRepo.ListByXUID(ctx, pdb.XUID)
	if err != nil {
		slog.WarnContext(ctx, "progression: list PBs", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_pbs_error", err.Error())
	}

	historyRepo := duckdb.NewRecordHistoryRepo(pdb.Player)
	histList, err := historyRepo.ListRecent(ctx, pdb.XUID, requestTitleSlug(ctx, h.titleSlug), limit)
	if err != nil {
		slog.WarnContext(ctx, "progression: list record history", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_history_error", err.Error())
	}

	resp := recordsResponse{
		PersonalBests: make([]personalBestDTO, 0, len(pbList)),
		History:       make([]recordHistoryDTO, 0, len(histList)),
	}
	// A4 — filtre read-side : une métrique hors catalogue (ex « best_kda »
	// legacy) n'est pas servie à l'UI (elle n'a ni libellé ni bornes connues).
	// Trace la dégradation sans avaler l'anomalie (règle logging n°3).
	for _, pb := range pbList {
		if !records.IsKnownMetric(pb.Metric) {
			slog.WarnContext(ctx, "progression: PB métrique hors catalogue non servie",
				"metric", pb.Metric, "period", pb.Period)
			continue
		}
		resp.PersonalBests = append(resp.PersonalBests, toPBDTO(pb))
	}
	for _, hh := range histList {
		if !records.IsKnownMetric(hh.Metric) {
			slog.WarnContext(ctx, "progression: record history métrique hors catalogue non servi",
				"metric", hh.Metric, "period", hh.Period)
			continue
		}
		resp.History = append(resp.History, toHistoryDTO(hh))
	}
	return &recordsHumaOutput{Body: resp}, nil
}

// handleMilestones : GET /milestones → catalogue du titre + statut Earned.
func (h *ProgressionHandler) handleMilestones(ctx context.Context, in *progPlayerInput) (*milestonesHumaOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	items, err := h.milestoneDTOs(ctx, pdb, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		return nil, err
	}
	return &milestonesHumaOutput{Body: milestonesResponse{Items: items}}, nil
}

// milestoneDTOs charge le catalogue du titre + le statut Earned du joueur puis les JOINT
// en DTOs. Extrait du handler (K1h) — la jointure catalog × earned n'est plus dans la
// méthode HTTP. Erreurs mappées en humacore.NewError (5xx).
func (h *ProgressionHandler) milestoneDTOs(ctx context.Context, pdb *duckdb.PlayerDB, titleSlug string) ([]milestoneDTO, error) {
	catalog, err := duckdb.NewMilestoneCatalogRepo(pdb.Metadata).ListByTitle(ctx, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "progression: list catalog", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_catalog_error", err.Error())
	}
	earnedList, err := duckdb.NewMilestoneEarnedRepo(pdb.Player).ListByUser(ctx, pdb.XUID, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "progression: list earned", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_earned_error", err.Error())
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
			// earned_at NULL (backfill A6 non dérivable) → EarnedAt zéro : le
			// jalon reste débloqué mais sans date (le front n'affiche pas de date
			// fausse). Sinon on sert la vraie date de franchissement.
			if !at.IsZero() {
				t := at
				dto.EarnedAt = &t
			}
		}
		items = append(items, dto)
	}
	return items, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// resolvePlayer résout le slug courant ou renvoie une erreur Huma 404
// (contrat préservé : {code:player_not_found}).
func (h *ProgressionHandler) resolvePlayer(ctx context.Context, slug string) (*duckdb.PlayerDB, error) {
	pdb, err := h.resolve(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return pdb, nil
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
	// A9 : on sert les descriptions localisées (le front choisit selon la locale),
	// jamais la formule technique c.Condition. Vides si le jalon n'a pas de
	// condition → le front n'affiche rien.
	return milestoneDTO{
		ID:          c.ID,
		Metric:      c.Metric,
		Threshold:   c.Threshold,
		TitleEN:     c.TitleEN,
		TitleFR:     c.TitleFR,
		Icon:        c.Icon,
		ConditionFR: c.ConditionFR,
		ConditionEN: c.ConditionEN,
	}
}
