package prestige

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// service.go — orchestrateur du module Prestige.
//
// Référence : Annexe E du plan conceptuel (surface API du module).
//
// Le Service est l'unique point d'entrée pour les autres packages
// (sync, api/handlers). Les internes (palier, baseline, lifecycle, evaluator)
// ne sont pas exposés directement en dehors du package.

// ---------- Erreurs publiques ----------

var (
	ErrChallengeNotFound = errors.New("prestige: challenge not found")
	ErrArcNotFound       = errors.New("prestige: arc not found")
	ErrUserNotFound      = errors.New("prestige: user not found")
	ErrInvalidInput      = errors.New("prestige: invalid input")
	ErrForbidden         = errors.New("prestige: forbidden")
)

// ---------- Service ----------

// Service est le contrat exposé du module Prestige.
type Service interface {
	CreateChallenge(ctx context.Context, req CreateChallengeRequest) (Challenge, error)
	UpdateChallenge(ctx context.Context, id string, patch UpdateChallengePatch) (Challenge, error)
	AbandonChallenge(ctx context.Context, id string) error
	GetChallenge(ctx context.Context, id string) (Challenge, error)
	ListActiveChallenges(ctx context.Context, userID, titleSlug string) ([]Challenge, error)

	// EvaluateForUser ré-évalue tous les défis actifs d'un joueur sur un titre.
	// Appelé par le sync hook après ingestion des nouveaux matchs.
	EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]EvaluationOutcome, error)

	GetUserPrestige(ctx context.Context, userID, titleSlug string) (UserPrestige, error)
	SuggestTemplates(ctx context.Context, userID, titleSlug string, count int) ([]Template, error)
	SuggestNext(ctx context.Context, completedID string) ([]Template, error)

	// Arcs
	CreateArc(ctx context.Context, req CreateArcRequest) (Arc, error)
	ListArcs(ctx context.Context, userID, titleSlug string) ([]Arc, error)
	GetArc(ctx context.Context, id string) (Arc, error)
	// DeleteArc supprime un arc appartenant à userID. opts.CascadeObjectives
	// décide du sort des objectifs (supprimés/abandonnés vs détachés).
	DeleteArc(ctx context.Context, userID, id string, opts DeleteArcOptions) error
	// ListArcPresets retourne le catalogue d'arcs preset d'un titre, chaque
	// preset hydraté avec ses étapes (pour l'aperçu). userID sert à la
	// résolution du service côté lazy (les presets sont title-scoped).
	ListArcPresets(ctx context.Context, userID, titleSlug string) ([]PresetArc, error)
	// AdoptPresetArc matérialise un arc preset pour le joueur : crée l'arc
	// (IsPreset=true) + un objectif par étape (depuis le template + palier cible).
	AdoptPresetArc(ctx context.Context, userID, titleSlug, presetID string) (Arc, error)

	// Escouade
	CreateSquadChallenge(ctx context.Context, req CreateSquadChallengeRequest) (SquadChallenge, error)
	JoinSquadChallenge(ctx context.Context, challengeID, userID string, chosenTier Tier, isPrivate bool) error
	GetSquadChallenge(ctx context.Context, id string) (SquadChallenge, error)
	ListSquadChallenges(ctx context.Context, squadID string) ([]SquadChallenge, error)
	RefreshSquadPool(ctx context.Context, squadID, titleSlug, requestedBy string) ([]Template, error)

	// Escouade — roster (entité Squad / SquadMember, clé xuid). requestedBy =
	// player_slug de l'acteur, qui doit être membre-user pour muter le roster.
	CreateSquad(ctx context.Context, req CreateSquadRequest) (Squad, error)
	ListSquadsForUser(ctx context.Context, userID string) ([]Squad, error)
	GetSquad(ctx context.Context, id string) (Squad, error)
	ListSquadMembers(ctx context.Context, squadID string) ([]SquadMember, error)
	AddSquadMember(ctx context.Context, squadID string, member SquadMember, requestedBy string) error
	RemoveSquadMember(ctx context.Context, squadID, xuid, requestedBy string) error
	// RenameSquad change le nom d'une escouade. requestedBy (player_slug) doit
	// être membre-user de l'escouade.
	RenameSquad(ctx context.Context, squadID, name, requestedBy string) error
	// DeleteSquad supprime une escouade en retirant tous ses membres
	// (append-only : events is_member=FALSE → l'escouade sort de
	// ListSquadsForUser). requestedBy (player_slug) doit être membre-user.
	DeleteSquad(ctx context.Context, squadID, requestedBy string) error
	// SquadUsualContexts dérive les playlists/modes dominants des matchs communs
	// du roster (indice d'affichage du sélecteur, jamais stocké). Lecture seule.
	SquadUsualContexts(ctx context.Context, rosterXUIDs []string, titleSlug string) (playlists, modes []string, err error)
	// EvaluateSquadChallenge recalcule la progression de chaque membre d'un défi
	// d'escouade (no-overlap + agrégation cumulative) et la persiste. requestedBy
	// (player_slug) doit être membre-user de l'escouade.
	EvaluateSquadChallenge(ctx context.Context, squadChallengeID, requestedBy string) ([]SquadParticipantProgress, error)
	// SquadOrientation retourne l'axe focal de l'escouade (le plus faible du
	// profil de perf agrégé) — l'orientation à renforcer ; "" si pas de profil
	// exploitable. requestedBy (player_slug) doit être membre-user.
	SquadOrientation(ctx context.Context, squadID, requestedBy string) (string, error)

	// Mode pilote (auto-attribution)
	EnablePilotMode(ctx context.Context, userID, titleSlug string) (PilotModeAttribution, error)
	DisablePilotMode(ctx context.Context, userID, titleSlug string) error
}

// PilotModeAttribution est le résultat d'une activation du mode pilote.
//
// Référence : Axe 7 du plan conceptuel.
//   - Daily       : 1 défi auto-attribué pour la journée
//   - WeeklyForced: 1 défi forcé pour la semaine
//   - WeeklyChoices: 3 défis proposés, le joueur en choisit au moins 1
type PilotModeAttribution struct {
	Daily         *Challenge `json:"daily,omitempty"`
	WeeklyForced  *Challenge `json:"weekly_forced,omitempty"`
	WeeklyChoices []Template `json:"weekly_choices"`
}

// CreateArcRequest est l'entrée pour créer un arc libre.
type CreateArcRequest struct {
	UserID      string
	TitleSlug   string
	Title       string
	Description string
	// Source trace l'origine d'un arc pour analytics. Valeurs : "user"
	// (manuel, défaut si vide) | "pilot_mode" | "coach" (issu d'une
	// acceptance coach_advisor proposal, cf. ADR 0020).
	Source string
}

// CreateSquadChallengeRequest est l'entrée pour créer un défi d'escouade.
type CreateSquadChallengeRequest struct {
	SquadID         string
	TemplateID      string
	TitleSlug       string
	Mode            SquadMode
	EvalType        EvalType
	WindowType      WindowType
	WindowValue     string
	TargetPerMember float64
	ExpiresAt       *time.Time
	CreatedBy       string
}

// CreateSquadRequest est l'entrée pour créer une escouade.
//
// CreatedBy = player_slug du créateur (utilisateur de l'app). Members = roster
// initial ; le handler y inclut le créateur et résout xuid + user_id(slug) de
// chaque membre (le package prestige ne connaît pas db_profiles).
type CreateSquadRequest struct {
	Name      string
	CreatedBy string
	Members   []SquadMember
}

// ---------- DTOs ----------

// CreateChallengeRequest est l'entrée pour créer un défi.
type CreateChallengeRequest struct {
	UserID          string
	TitleSlug       string
	ArcID           string // vide si standalone
	TemplateID      string // vide si défi 100 % libre
	Metric          string
	Target          float64
	WindowType      WindowType
	WindowValue     string
	Cadence         Cadence
	EvalType        EvalType
	Mode            ChallengeMode
	Label           string
	IsPrivate       bool
	TargetPerMember float64 // pour défis collectifs (0 sinon)
	Position        int     // ordre dans l'arc (0 si standalone)
	// Source trace l'origine d'un challenge pour analytics. Valeurs : "user"
	// (manuel, défaut si vide) | "pilot_mode" | "coach" (issu d'une
	// acceptance coach_advisor proposal, cf. ADR 0020).
	Source string
}

// UpdateChallengePatch décrit une édition partielle d'un défi.
//
// Tous les champs sont optionnels. Si Target != nil, le palier est recalculé
// sur la baseline courante (mode libre uniquement — sinon ErrNotEditable).
type UpdateChallengePatch struct {
	Target *float64
	Label  *string
}

// EvaluationOutcome est le résultat d'une évaluation pour un défi donné.
type EvaluationOutcome struct {
	ChallengeID string
	OldStatus   ChallengeStatus
	NewStatus   ChallengeStatus
	NewValue    float64
	PPCredited  int
	Reason      EvalReason

	// ArcCompletedID est l'ID de l'arc clôturé par cette transition — renseigné
	// uniquement quand la complétion de ce défi termine la dernière étape de son
	// arc. Vide sinon (défi standalone, ou arc encore en cours).
	ArcCompletedID string
	// ArcPPCredited est le bonus PP crédité pour la complétion de l'arc
	// (cf. PPForArcCompletion). 0 si aucun arc n'a été clôturé.
	ArcPPCredited int
}

// ---------- Implémentation ----------

// Deps regroupe les dépendances du service.
type Deps struct {
	Tuning           Tuning
	Challenges       ChallengeRepo
	Arcs             ArcRepo
	Moments          MomentCardRepo
	Prestige         PrestigeRepo
	Telemetry        TelemetryRepo
	BaselineState    BaselineStateRepo
	Templates        TemplateRepo
	PresetArcs       PresetArcRepo
	SquadChallenges  SquadChallengeRepo
	Squads           SquadRepo
	BaselineProvider BaselineProvider     // fournit les MatchData pour calculer la baseline
	SquadMatches     SquadMatchProvider   // métriques par match pour l'éval des défis d'escouade
	SquadProfile     SquadProfileProvider // profil 6-axes agrégé pour le biais coach du pool (optionnel)
	Now              func() time.Time
}

// BaselineProvider est l'abstraction des sources de matchs pour la baseline.
//
// Implémentée hors du package par les adaptateurs `internal/games/...` —
// le module Prestige reste découplé de la sémantique métier des matchs.
type BaselineProvider interface {
	RecentMatches(ctx context.Context, userID, titleSlug, metric string, window int) ([]MatchData, error)
	PopulationPercentile(ctx context.Context, titleSlug, metric string, target float64) (percentile float64, popSize int, err error)
}

// SquadMatchProvider fournit, pour un roster d'escouade, les `limit` derniers
// matchs où TOUT le roster a joué — chacun avec ses participants (pour la règle
// no-overlap) et la valeur de `metric` (canonique, cf. Template.Metric) par
// membre. Implémenté hors package (platform/duckdb) : le module reste découplé
// de la sémantique des matchs. titleSlug est passé pour parité (la DB partagée
// est déjà title-scopée par chemin).
type SquadMatchProvider interface {
	SquadMatchMetrics(ctx context.Context, rosterXUIDs []string, titleSlug, metric string, limit int) ([]SquadMatchMetric, error)
	// SquadUsualContexts dérive les playlists/modes dominants des matchs communs
	// réels du roster (top par fréquence, labels résolus). Indice d'affichage
	// auto-adaptatif (jamais stocké). Listes vides si aucun match commun.
	SquadUsualContexts(ctx context.Context, rosterXUIDs []string, titleSlug string, limit int) (playlists, modes []string, err error)
}

// SquadProfileProvider fournit le profil 6-axes par membre d'une escouade
// (réutilise la base du Synergy Radar analytics) pour orienter le coach
// d'escouade : moyenne par axe → axe le plus faible → biais du pool de défis.
// Optionnel dans Deps (nil → RefreshSquadPool reste un shuffle non biaisé).
type SquadProfileProvider interface {
	SquadAxes(ctx context.Context, rosterXUIDs []string, titleSlug string) ([]map[string]float64, error)
}

// service est l'implémentation par défaut.
type service struct {
	deps    Deps
	emitter *TelemetryEmitter
}

// NewService construit un service Prestige.
func NewService(d Deps) Service {
	if d.Now == nil {
		d.Now = func() time.Time { return time.Now().UTC() }
	}
	return &service{
		deps:    d,
		emitter: NewTelemetryEmitter(d.Telemetry, d.Now),
	}
}

// ---------- CreateChallenge ----------

// Orchestration de création cohésive (validation + activation Prestige + persistance) ;
// le flux de garde/validation est séquentiel et lié (K3f, exemption).
//
//nolint:funlen
func (s *service) CreateChallenge(ctx context.Context, req CreateChallengeRequest) (Challenge, error) {
	if err := validateCreateRequest(req); err != nil {
		return Challenge{}, err
	}

	// Vérification quotas (mode pilote uniquement). Le mode libre n'a pas de plafond.
	if err := s.checkQuotas(ctx, req); err != nil {
		return Challenge{}, err
	}

	now := s.deps.Now()

	// Cooldown anti-farming (tous modes) : refuse de recréer un défi sur une
	// métrique dont un défi précédent a été abandonné/expiré récemment.
	if err := s.checkCooldown(ctx, req, now); err != nil {
		return Challenge{}, err
	}

	// Calculer la baseline et le palier.
	baseline, err := s.computeBaselineFor(ctx, req.UserID, req.TitleSlug, req.Metric)
	if err != nil {
		return Challenge{}, fmt.Errorf("baseline: %w", err)
	}

	stretch := ComputeStretchRatio(req.Target, baseline.Value, metricCeiling(req.Metric), metricKindFor(req.Metric))
	pop, popSize, err := s.deps.BaselineProvider.PopulationPercentile(ctx, req.TitleSlug, req.Metric, req.Target)
	if err != nil {
		// Pop indispo : on continue avec popSize=0, le cap sera désactivé
		slog.WarnContext(ctx, "prestige: population percentile unavailable", "err", err)
		pop, popSize = 0, 0
	}

	tier, reject := CalculatePalier(s.deps.Tuning, CalculatePalierInput{
		Stretch:              stretch,
		PopulationPercentile: pop,
		PopulationSize:       popSize,
		DataTier:             baseline.DataTier,
	})

	// Pour WindowLastNMatches : dériver WindowValue depuis le tuning si non fourni.
	windowValue := req.WindowValue
	if req.WindowType == WindowLastNMatches && windowValue == "" {
		windowValue = strconv.Itoa(s.deps.Tuning.RequiredMatchCount(tier))
	}

	// Calculer ExpiresAt selon tier et mode (nil pour mode libre ou durée = 0).
	var expiresAt *time.Time
	if d := s.deps.Tuning.ExpirationDurationFor(tier, req.Mode); d > 0 {
		exp := now.Add(d)
		expiresAt = &exp
	}

	c := Challenge{
		ID:              newID("ch"),
		UserID:          req.UserID,
		TitleSlug:       req.TitleSlug,
		ArcID:           req.ArcID,
		Position:        req.Position,
		TemplateID:      req.TemplateID,
		Metric:          req.Metric,
		Target:          req.Target,
		TargetPerMember: req.TargetPerMember,
		WindowType:      req.WindowType,
		WindowValue:     windowValue,
		Cadence:         req.Cadence,
		EvalType:        req.EvalType,
		Mode:            req.Mode,
		Tier:            tier,
		DataTier:        baseline.DataTier,
		Label:           req.Label,
		Status:          StatusActive,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		CommittedAt:     &now,
		IsPrivate:       req.IsPrivate,
		// Origine tracée telle quelle (ADR 0020). Les 3 voies de création la
		// renseignent (user via handler HTTP, pilot_mode via pilot_pool, coach via
		// coach_advisor) ; une valeur vide résiduelle (caller legacy) => agrégée
		// "unknown" par l'endpoint diag. Pas de normalisation ici.
		Source: req.Source,
	}

	// En cas de rejet "too_easy", on retourne une erreur sans persister.
	if reject == RejectTooEasy {
		s.emitter.EmitRejected(ctx, req.UserID, c, stretch, baseline.Value, reject)
		return Challenge{}, fmt.Errorf("%w: stretch=%.2f below min", ErrInvalidInput, stretch)
	}

	// data=tracking : on persiste mais sans bonus PP futur (DataTier déjà = tracking).
	if err := s.deps.Challenges.Create(ctx, c); err != nil {
		return Challenge{}, fmt.Errorf("persist challenge: %w", err)
	}

	s.emitter.EmitCreated(ctx, c, stretch, baseline.Value)
	slog.InfoContext(ctx, "prestige: challenge created",
		"challenge_id", c.ID, "user_id", c.UserID, "tier", c.Tier,
		"mode", c.Mode, "cadence", c.Cadence)

	return c, nil
}

// checkQuotas valide les plafonds simultanés et anti-farming en mode pilote.
//
// Règles (Axe 7) :
//   - Mode libre : pas de plafond, retourne nil immédiatement
//   - Mode pilote : applique daily_max, weekly_max, monthly_max, total_active_max
//   - Anti-farming : max free_new_per_day défis libres créés/jour (s'applique
//     uniquement quand un mode pilote est aussi actif côté joueur — sinon
//     le mode libre reste sans contrainte par design)
//
// Le seul mode contraint à la création est ModePilote. Le mode libre passe
// toujours (anti-smurf via baseline + palier suffit).
func (s *service) checkQuotas(ctx context.Context, req CreateChallengeRequest) error {
	if req.Mode != ModePilote {
		return nil
	}

	q := s.deps.Tuning.QuotasPilote

	totalActive, err := s.deps.Challenges.CountActiveTotal(ctx, req.UserID, req.TitleSlug)
	if err != nil {
		return fmt.Errorf("count active total: %w", err)
	}
	if totalActive >= q.TotalActiveMax {
		return fmt.Errorf("%w: %d défis pilote actifs (max %d)",
			ErrInvalidInput, totalActive, q.TotalActiveMax)
	}

	cadenceMax := cadenceQuotaMax(q, req.Cadence)
	if cadenceMax > 0 {
		actives, err := s.deps.Challenges.CountActiveByCadence(ctx, req.UserID, req.TitleSlug, req.Cadence)
		if err != nil {
			return fmt.Errorf("count by cadence: %w", err)
		}
		if actives >= cadenceMax {
			return fmt.Errorf("%w: %d défis %s actifs (max %d)",
				ErrInvalidInput, actives, req.Cadence, cadenceMax)
		}
	}
	return nil
}

// checkCooldown refuse la création d'un défi si la métrique est en cooldown
// anti-farming, i.e. un défi précédent du joueur sur la MÊME métrique a été
// abandonné (24h) ou a expiré (12h) dans la fenêtre. S'applique à tous les
// modes (cf. CooldownEndsAt). La complétion ne déclenche pas de cooldown.
func (s *service) checkCooldown(ctx context.Context, req CreateChallengeRequest, now time.Time) error {
	metric := req.Metric
	prev, err := s.deps.Challenges.List(ctx, ChallengeFilter{
		UserID:    req.UserID,
		TitleSlug: req.TitleSlug,
		Metric:    &metric,
	})
	if err != nil {
		return fmt.Errorf("list for cooldown: %w", err)
	}
	if IsCooldownActive(s.deps.Tuning, prev, now) {
		return ErrCooldownActive
	}
	return nil
}

// cadenceQuotaMax retourne le plafond de simultanés selon la cadence.
// 0 si non applicable (mode libre ou cadence inconnue).
func cadenceQuotaMax(q QuotasPiloteTuning, c Cadence) int {
	switch c {
	case CadenceDaily:
		return q.DailyMax
	case CadenceWeekly:
		return q.WeeklyMax
	case CadenceMonthly:
		return q.MonthlyMax
	}
	return 0
}

// validateCreateRequest applique les validations basiques d'entrée.
func validateCreateRequest(req CreateChallengeRequest) error {
	if req.UserID == "" || req.TitleSlug == "" || req.Metric == "" {
		return fmt.Errorf("%w: user_id/title_slug/metric requis", ErrInvalidInput)
	}
	if req.Target <= 0 {
		return fmt.Errorf("%w: target must be > 0", ErrInvalidInput)
	}
	if !req.WindowType.Valid() || !req.Cadence.Valid() ||
		!req.EvalType.Valid() || !req.Mode.Valid() {
		return fmt.Errorf("%w: enum invalide", ErrInvalidInput)
	}
	return nil
}

// computeBaselineFor charge les matchs récents et calcule la baseline.
func (s *service) computeBaselineFor(ctx context.Context, userID, titleSlug, metric string) (Baseline, error) {
	matches, err := s.deps.BaselineProvider.RecentMatches(
		ctx, userID, titleSlug, metric, s.deps.Tuning.Baseline.WindowMatches,
	)
	if err != nil {
		return Baseline{}, err
	}
	return ComputeBaseline(s.deps.Tuning, userID, titleSlug, metric, matches), nil
}

// ---------- UpdateChallenge ----------

func (s *service) UpdateChallenge(ctx context.Context, id string, patch UpdateChallengePatch) (Challenge, error) {
	c, err := s.deps.Challenges.Get(ctx, id)
	if err != nil {
		return Challenge{}, ErrChallengeNotFound
	}

	if patch.Label != nil {
		if err := s.deps.Challenges.UpdateLabel(ctx, id, *patch.Label); err != nil {
			return Challenge{}, fmt.Errorf("update label: %w", err)
		}
		c.Label = *patch.Label
	}

	if patch.Target != nil {
		if !CanEditTarget(c) {
			return Challenge{}, ErrNotEditable
		}
		oldTier := c.Tier
		newTier, baseline, stretch, err := s.recomputeTier(ctx, c, *patch.Target)
		if err != nil {
			return Challenge{}, err
		}
		now := s.deps.Now()
		if err := s.deps.Challenges.UpdateTarget(ctx, id, *patch.Target, newTier, baseline.DataTier, now); err != nil {
			return Challenge{}, fmt.Errorf("update target: %w", err)
		}
		c.Target = *patch.Target
		c.Tier = newTier
		c.DataTier = baseline.DataTier
		c.LastPalierRecomputeAt = &now

		s.emitter.EmitPalierRecomputed(ctx, c, oldTier, stretch, baseline.Value)
	}

	return c, nil
}

// recomputeTier recharge la baseline et calcule un nouveau palier pour la cible donnée.
func (s *service) recomputeTier(ctx context.Context, c Challenge, target float64) (Tier, Baseline, float64, error) {
	baseline, err := s.computeBaselineFor(ctx, c.UserID, c.TitleSlug, c.Metric)
	if err != nil {
		return TierNormal, Baseline{}, 0, err
	}
	stretch := ComputeStretchRatio(target, baseline.Value, metricCeiling(c.Metric), metricKindFor(c.Metric))
	pop, popSize, _ := s.deps.BaselineProvider.PopulationPercentile(ctx, c.TitleSlug, c.Metric, target)
	tier, _ := CalculatePalier(s.deps.Tuning, CalculatePalierInput{
		Stretch:              stretch,
		PopulationPercentile: pop,
		PopulationSize:       popSize,
		DataTier:             baseline.DataTier,
	})
	return tier, baseline, stretch, nil
}

// ---------- AbandonChallenge ----------

func (s *service) AbandonChallenge(ctx context.Context, id string) error {
	c, err := s.deps.Challenges.Get(ctx, id)
	if err != nil {
		return ErrChallengeNotFound
	}
	if !CanAbandon(c) {
		return ErrAlreadyTerminal
	}
	now := s.deps.Now()
	updated, err := MarkAbandoned(c, now)
	if err != nil {
		return err
	}
	if err := s.deps.Challenges.UpdateStatus(ctx, id, updated.Status, now); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	s.emitter.EmitTransition(ctx, updated, TelemetryAbandoned)
	slog.InfoContext(ctx, "prestige: challenge abandoned", "challenge_id", id)
	return nil
}

// ---------- GetChallenge / ListActiveChallenges ----------

func (s *service) GetChallenge(ctx context.Context, id string) (Challenge, error) {
	c, err := s.deps.Challenges.Get(ctx, id)
	if err != nil {
		return Challenge{}, ErrChallengeNotFound
	}
	return c, nil
}

func (s *service) ListActiveChallenges(ctx context.Context, userID, titleSlug string) ([]Challenge, error) {
	active := StatusActive
	list, err := s.deps.Challenges.List(ctx, ChallengeFilter{
		UserID:    userID,
		TitleSlug: titleSlug,
		Status:    &active,
	})
	if err != nil {
		return nil, err
	}
	// Enrichit chaque défi avec sa valeur courante mesurée. Pure (pas de
	// persistance) — l'évaluateur threshold est lui-même pur. Permet au front
	// de trier par % de progression dans la home Prestige.
	now := s.deps.Now()
	for i, c := range list {
		list[i].CurrentValue = s.computeCurrentValue(ctx, c, now)
		// Récompense PP affichée = ce qui serait crédité à la complétion (même
		// calcul que creditCompletion, isSquad=false pour les défis perso).
		list[i].PPReward = PPForCompletion(s.deps.Tuning, c.Tier, false, c.DataTier)
	}
	return list, nil
}

// computeCurrentValue calcule la valeur courante mesurée pour un défi sans
// persister. Best-effort : si la source des matchs échoue, retourne 0.
func (s *service) computeCurrentValue(ctx context.Context, c Challenge, now time.Time) float64 {
	matches, err := s.deps.BaselineProvider.RecentMatches(
		ctx, c.UserID, c.TitleSlug, c.Metric, s.deps.Tuning.Baseline.WindowMatches,
	)
	if err != nil {
		return 0
	}
	samples := make([]MatchSample, len(matches))
	for i, m := range matches {
		samples[i] = MatchSample{StartedAt: m.StartedAt, MetricValue: m.MetricValue}
	}
	// Threshold suffit pour Phase 2 (cumulative tombe sur le même fallback).
	return EvaluateThreshold(s.deps.Tuning, c, samples, now).NewValue
}

// ---------- GetUserPrestige ----------

func (s *service) GetUserPrestige(ctx context.Context, userID, titleSlug string) (UserPrestige, error) {
	var (
		up  UserPrestige
		err error
	)
	if titleSlug == "" {
		up, err = s.deps.Prestige.GetUserPrestigeCrossTitle(ctx, userID)
	} else {
		up, err = s.deps.Prestige.GetUserPrestige(ctx, userID, titleSlug)
	}
	if err != nil {
		return up, err
	}
	lvl := LevelFromPP(s.deps.Tuning, up.TotalPP)
	up.Level = &lvl
	return up, nil
}

// ---------- Suggestions ----------

func (s *service) SuggestTemplates(ctx context.Context, userID, titleSlug string, count int) ([]Template, error) {
	if count <= 0 {
		count = s.deps.Tuning.Suggestion.AlternativesCount
	}
	// TODO Phase 3 : exclure les templates déjà tentés (lookup dans challenges).
	tmpls, err := s.deps.Templates.Suggest(ctx, titleSlug, nil, count)
	if err != nil {
		return nil, err
	}
	s.annotateCooldowns(ctx, userID, titleSlug, tmpls)
	return tmpls, nil
}

// annotateCooldowns renseigne Template.CooldownEndsAt pour chaque template dont
// la métrique est en cooldown anti-farming actif pour le joueur. Une seule
// lecture des défis du joueur, puis map métrique→fin de cooldown la plus
// lointaine. Best-effort : en cas d'erreur de lecture, on n'annote pas (le
// front retombe sur le refus réactif 429 à la création).
func (s *service) annotateCooldowns(ctx context.Context, userID, titleSlug string, tmpls []Template) {
	if userID == "" || len(tmpls) == 0 {
		return
	}
	now := s.deps.Now()
	all, err := s.deps.Challenges.List(ctx, ChallengeFilter{UserID: userID, TitleSlug: titleSlug})
	if err != nil {
		slog.WarnContext(ctx, "prestige: cooldown annotation skipped", "err", err)
		return
	}
	endByMetric := make(map[string]time.Time)
	for _, c := range all {
		end := CooldownEndsAt(s.deps.Tuning, c)
		if end.IsZero() || !now.Before(end) {
			continue
		}
		if cur, ok := endByMetric[c.Metric]; !ok || end.After(cur) {
			endByMetric[c.Metric] = end
		}
	}
	for i := range tmpls {
		if end, ok := endByMetric[tmpls[i].Metric]; ok {
			e := end
			tmpls[i].CooldownEndsAt = &e
		}
	}
}

func (s *service) SuggestNext(ctx context.Context, completedID string) ([]Template, error) {
	c, err := s.deps.Challenges.Get(ctx, completedID)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	return s.deps.Templates.Suggest(ctx, c.TitleSlug, []string{c.TemplateID}, s.deps.Tuning.Suggestion.AlternativesCount)
}

// EvaluateForUser, evaluateOne, applyTransition : voir service_evaluate.go

// ---------- Helpers ----------

// metricCeiling retourne le plafond physique d'une métrique (0 = pas de plafond).
//
// Utilisé pour le calcul de stretch normalisé sur les métriques bornées.
// Référence : Annexe A du plan conceptuel.
func metricCeiling(metric string) float64 {
	switch metric {
	case "FieldAccuracy", "accuracy":
		return 1.0
	case MetricWinRatePascal, MetricWinRateSnake:
		return 100.0
	case "performance_score":
		return 100.0
	}
	return 0
}

// metricKindFor classe la métrique pour le calcul de stretch.
func metricKindFor(metric string) MetricKind {
	switch metric {
	case "FieldAccuracy", "accuracy",
		MetricWinRatePascal, MetricWinRateSnake,
		MetricKDAPascal, "FieldKDR", "kda", "kdr",
		"performance_score":
		return MetricRatio
	}
	return MetricCount
}

// newID génère un identifiant aléatoire préfixé.
func newID(prefix string) string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return prefix + "_" + hex.EncodeToString(buf[:])
}
