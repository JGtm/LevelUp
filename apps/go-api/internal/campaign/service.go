package campaign

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"levelup/go-api/internal/analysis/temporal"
)

// service.go — orchestrateur ImprovementCampaign (V1 §4.5).
//
// Une instance Service prend une Repo (interface pour mockabilité) +
// SampleProvider (lecture des valeurs d'axe pour calcul snapshot/courant).
//
// Les méthodes Start/Pause/Close/Abandon sont des transitions d'état pures
// (1 UPDATE SQL). Evaluate recompute LOWESS + MWU + heuristique R5.

// ErrNotFound — campagne inconnue.
var ErrNotFound = errors.New("campaign: not found")

// ErrAlreadyActive — tentative de StartCampaign alors qu'une autre est active
// pour le même (user, titleSlug). Cf. plan §4.5 : 1 max active à la fois.
var ErrAlreadyActive = errors.New("campaign: another campaign is already active")

// ErrInvalidStatus — transition d'état non autorisée (ex: Pause sur Completed).
var ErrInvalidStatus = errors.New("campaign: invalid status transition")

// ErrInvalidAxis — axis vide ou axis_kind hors enum.
var ErrInvalidAxis = errors.New("campaign: invalid axis or axis_kind")

// Repo abstrait l'accès stats.duckdb (testabilité).
type Repo interface {
	Insert(ctx context.Context, c ImprovementCampaign) error
	GetByID(ctx context.Context, id string) (ImprovementCampaign, error)
	GetActive(ctx context.Context, userID, titleSlug string) (ImprovementCampaign, error)
	// ListEnded liste les campagnes closes (completed/abandoned) d'un joueur sur
	// un titre, les plus récentes d'abord (tri par ended_at desc). Sert la
	// surface « Historique » de l'onglet Réalisations (Lot C).
	ListEnded(ctx context.Context, userID, titleSlug string) ([]ImprovementCampaign, error)
	UpdateStatus(ctx context.Context, id string, status CampaignStatus, endedAt *time.Time) error
	UpdateEvaluation(ctx context.Context, id string, eval Evaluation) error
	LinkedChallengeIDs(ctx context.Context, campaignID string) ([]string, error)
	LinkChallenge(ctx context.Context, challengeID, campaignID string) error
}

// SampleProvider charge les valeurs brutes d'un axe pour un intervalle.
// Implémenté côté platform/duckdb pour s'interfacer avec match_participants
// et personal_score_awards.
type SampleProvider interface {
	// LoadAxisSamples retourne les valeurs (chronologiquement asc) de l'axe
	// pour les matchs du joueur dans la fenêtre + sur le playlist_group cible.
	// playlistGroup="all" → pas de filtre.
	LoadAxisSamples(
		ctx context.Context,
		userID, titleSlug, axis string, axisKind AxisKind,
		playlistGroup string,
		since, until time.Time,
	) ([]float64, error)
}

// LeverageProvider retourne les leviers actuels du profil joueur (V2 §4 — R5
// "axe sort du bottom-3 du radar"). Optionnel : si non câblé, R5 ne déclenche
// que sur la condition plateau_60d.
type LeverageProvider interface {
	// CurrentLeverageComponents retourne les composantes / axes identifiés
	// comme leviers d'amélioration actuels du joueur. Si l'axe de la
	// campagne n'est plus dans cette liste, on peut suggérer la clôture.
	CurrentLeverageComponents(ctx context.Context, userID, titleSlug string) ([]string, error)
}

// Evaluation porte le résultat pur du calcul (sans I/O).
type Evaluation struct {
	CurrentRaw           sql.NullFloat64
	CurrentLOWESS        sql.NullFloat64
	MatchesSinceStart    int
	EvaluatedAt          time.Time
	MannWhitneyP         sql.NullFloat64
	ProgressionConfirmed bool
	AutoClosureSuggested bool
	AutoClosureReason    string
}

// Service orchestre les opérations sur les campagnes.
type Service struct {
	repo      Repo
	samples   SampleProvider
	leverages LeverageProvider // optionnel — active R5 "axe sort bottom-3"
}

// NewService construit le service.
func NewService(repo Repo, samples SampleProvider) *Service {
	return &Service{repo: repo, samples: samples}
}

// WithLeverageProvider injecte la source des leviers courants pour R5
// (V2 §4). Chainable. Si non câblé, R5 ne déclenche que sur plateau_60d.
func (s *Service) WithLeverageProvider(p LeverageProvider) *Service {
	s.leverages = p
	return s
}

// StartParams sont les paramètres de StartCampaign.
type StartParams struct {
	UserID        string
	TitleSlug     string
	Axis          string
	AxisKind      AxisKind
	PlaylistGroup string // "" → "all"
}

// StartCampaign crée une nouvelle campagne après snapshot des 100 derniers
// matchs du joueur sur l'axe ciblé. Une seule campagne active à la fois.
func (s *Service) StartCampaign(ctx context.Context, p StartParams, now time.Time) (ImprovementCampaign, error) {
	if p.Axis == "" {
		return ImprovementCampaign{}, ErrInvalidAxis
	}
	if p.AxisKind != AxisKindRadar && p.AxisKind != AxisKindLUSRComponent {
		return ImprovementCampaign{}, ErrInvalidAxis
	}
	if p.PlaylistGroup == "" {
		p.PlaylistGroup = "all"
	}
	// Vérif unicité campagne active.
	if existing, err := s.repo.GetActive(ctx, p.UserID, p.TitleSlug); err == nil && existing.IsActive() {
		return ImprovementCampaign{}, ErrAlreadyActive
	}
	// Snapshot : moyenne des 100 derniers matchs avant 'now' sur la playlist.
	since := now.AddDate(0, 0, -180) // fenêtre élargie ; samples capera à 100
	samples, err := s.samples.LoadAxisSamples(ctx, p.UserID, p.TitleSlug, p.Axis, p.AxisKind, p.PlaylistGroup, since, now)
	if err != nil {
		return ImprovementCampaign{}, fmt.Errorf("load snapshot samples: %w", err)
	}
	if len(samples) > 100 {
		samples = samples[len(samples)-100:]
	}
	snapshot, sample := mean(samples), len(samples)

	c := ImprovementCampaign{
		ID:                 uuid.NewString(),
		UserID:             p.UserID,
		TitleSlug:          p.TitleSlug,
		Axis:               p.Axis,
		AxisKind:           p.AxisKind,
		StartedAt:          now,
		Status:             StatusActive,
		PlaylistGroup:      p.PlaylistGroup,
		SnapshotValue:      snapshot,
		SnapshotSample:     sample,
		LinkedChallengeIDs: nil,
	}
	if err := s.repo.Insert(ctx, c); err != nil {
		return ImprovementCampaign{}, fmt.Errorf("insert: %w", err)
	}
	slog.InfoContext(ctx, "campaign: started",
		"campaign_id", c.ID, "user_id", c.UserID, "title_slug", c.TitleSlug,
		"axis", c.Axis, "axis_kind", c.AxisKind, "playlist_group", c.PlaylistGroup,
		"snapshot_value", snapshot, "snapshot_sample", sample,
	)
	return c, nil
}

// GetActive retourne la campagne active du joueur (hydratée avec les défis liés).
func (s *Service) GetActive(ctx context.Context, userID, titleSlug string) (ImprovementCampaign, error) {
	c, err := s.repo.GetActive(ctx, userID, titleSlug)
	if err != nil {
		return c, err
	}
	ids, _ := s.repo.LinkedChallengeIDs(ctx, c.ID)
	c.LinkedChallengeIDs = ids
	return c, nil
}

// ListEnded retourne les campagnes closes (completed/abandoned) du joueur, les
// plus récentes d'abord. Lecture seule (pas d'hydratation des défis liés : la
// surface historique n'en a pas besoin).
func (s *Service) ListEnded(ctx context.Context, userID, titleSlug string) ([]ImprovementCampaign, error) {
	return s.repo.ListEnded(ctx, userID, titleSlug)
}

// GetByID retourne une campagne par ID (hydratée avec les défis liés).
func (s *Service) GetByID(ctx context.Context, id string) (ImprovementCampaign, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return c, err
	}
	ids, _ := s.repo.LinkedChallengeIDs(ctx, id)
	c.LinkedChallengeIDs = ids
	return c, nil
}

// PauseCampaign passe une campagne active en paused.
func (s *Service) PauseCampaign(ctx context.Context, id string) error {
	return s.transition(ctx, id, StatusActive, StatusPaused, nil)
}

// ResumeCampaign passe une campagne paused en active.
func (s *Service) ResumeCampaign(ctx context.Context, id string) error {
	return s.transition(ctx, id, StatusPaused, StatusActive, nil)
}

// CloseCampaign clôt une campagne (active ou paused) avec un endedAt.
func (s *Service) CloseCampaign(ctx context.Context, id string, now time.Time) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.IsEnded() {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, StatusCompleted, &now)
}

// AbandonCampaign abandonne une campagne (idem Close mais avec status=abandoned).
func (s *Service) AbandonCampaign(ctx context.Context, id string, now time.Time) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.IsEnded() {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, StatusAbandoned, &now)
}

// LinkChallenge tague un défi avec une campagne (ou unlink si campaignID="").
func (s *Service) LinkChallenge(ctx context.Context, challengeID, campaignID string) error {
	return s.repo.LinkChallenge(ctx, challengeID, campaignID)
}

// EvaluateActive recompute LOWESS + MWU pour la campagne active du joueur.
// No-op (et nil error) si pas de campagne active. Idempotent : peut être
// appelé depuis le hook post-sync à chaque match ingéré.
func (s *Service) EvaluateActive(ctx context.Context, userID, titleSlug string, now time.Time) error {
	c, err := s.repo.GetActive(ctx, userID, titleSlug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return s.Evaluate(ctx, c, now)
}

// Evaluate recompute current+LOWESS+MWU+R5 pour une campagne donnée et persiste.
func (s *Service) Evaluate(ctx context.Context, c ImprovementCampaign, now time.Time) error {
	if !c.IsActive() {
		return nil
	}
	pre, post := s.loadCampaignSnapshots(ctx, c, now)

	eval := Evaluation{
		MatchesSinceStart: len(post),
		EvaluatedAt:       now,
	}
	applyCampaignAggregateStats(&eval, post)
	applyCampaignMannWhitney(ctx, &eval, c, pre, post)
	s.applyCampaignAutoClosureRules(ctx, &eval, c, now)
	if eval.AutoClosureSuggested {
		slog.InfoContext(ctx, "campaign: auto-closure suggested",
			"campaign_id", c.ID, "axis", c.Axis, "reason", eval.AutoClosureReason,
		)
	}
	return s.repo.UpdateEvaluation(ctx, c.ID, eval)
}

// loadCampaignSnapshots charge pre (180j cappé 100) + post (depuis StartedAt).
func (s *Service) loadCampaignSnapshots(ctx context.Context, c ImprovementCampaign, now time.Time) ([]float64, []float64) {
	preStart := c.StartedAt.AddDate(0, 0, -180)
	pre, _ := s.samples.LoadAxisSamples(ctx, c.UserID, c.TitleSlug, c.Axis, c.AxisKind, c.PlaylistGroup, preStart, c.StartedAt)
	if len(pre) > 100 {
		pre = pre[len(pre)-100:]
	}
	post, _ := s.samples.LoadAxisSamples(ctx, c.UserID, c.TitleSlug, c.Axis, c.AxisKind, c.PlaylistGroup, c.StartedAt, now)
	return pre, post
}

// applyCampaignAggregateStats renseigne CurrentRaw + CurrentLOWESS depuis post.
func applyCampaignAggregateStats(eval *Evaluation, post []float64) {
	if len(post) > 0 {
		raw := mean(post)
		eval.CurrentRaw = sql.NullFloat64{Float64: roundTo(raw, 2), Valid: true}
	}
	if len(post) >= 3 {
		sm := temporal.LowessSmooth(post, LOWESSAlpha)
		if last := lastValid(sm); !math.IsNaN(last) {
			eval.CurrentLOWESS = sql.NullFloat64{Float64: roundTo(last, 2), Valid: true}
		}
	}
}

// applyCampaignMannWhitney calcule le test MWU + flag ProgressionConfirmed.
func applyCampaignMannWhitney(ctx context.Context, eval *Evaluation, c ImprovementCampaign, pre, post []float64) {
	if len(post) < MinMatchesForMannWhitney || len(pre) < 3 {
		return
	}
	_, p := MannWhitneyU(pre, post)
	eval.MannWhitneyP = sql.NullFloat64{Float64: p, Valid: true}
	eval.ProgressionConfirmed = p < MannWhitneyThreshold
	if eval.ProgressionConfirmed {
		slog.InfoContext(ctx, "campaign: progression confirmed",
			"campaign_id", c.ID, "axis", c.Axis, "p_value", p,
			"matches_post", len(post), "matches_pre", len(pre),
		)
	}
}

// applyCampaignAutoClosureRules applique les 2 règles R5 (plateau, leverage).
//
// R5.1 : plateau 60j sans variation > 0.02 sur LOWESS.
// R5.2 (V2 §4) : axe sorti des leviers prioritaires (priorité plus haute, overwrite).
func (s *Service) applyCampaignAutoClosureRules(
	ctx context.Context, eval *Evaluation, c ImprovementCampaign, now time.Time,
) {
	if eval.ProgressionConfirmed {
		return
	}
	if now.Sub(c.StartedAt) > PlateauWindowDays*24*time.Hour && eval.CurrentLOWESS.Valid {
		delta := math.Abs(eval.CurrentLOWESS.Float64 - c.SnapshotValue)
		if delta < 0.02 {
			eval.AutoClosureSuggested = true
			eval.AutoClosureReason = "plateau_60d"
		}
	}
	if s.leverages == nil {
		return
	}
	current, err := s.leverages.CurrentLeverageComponents(ctx, c.UserID, c.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "campaign: leverage provider failed (R5 cond 2 skipped)",
			"campaign_id", c.ID, "err", err)
		return
	}
	if !containsString(current, c.Axis) && len(current) > 0 {
		eval.AutoClosureSuggested = true
		eval.AutoClosureReason = "axis_no_longer_priority"
	}
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// transition applique un changement de status si le current matche `from`.
func (s *Service) transition(ctx context.Context, id string, from, to CampaignStatus, endedAt *time.Time) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.Status != from {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, to, endedAt)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, v := range xs {
		s += v
	}
	return s / float64(len(xs))
}

func roundTo(v float64, decimals int) float64 {
	m := math.Pow10(decimals)
	return math.Round(v*m) / m
}

func lastValid(xs []float64) float64 {
	for i := len(xs) - 1; i >= 0; i-- {
		if !math.IsNaN(xs[i]) {
			return xs[i]
		}
	}
	return math.NaN()
}
