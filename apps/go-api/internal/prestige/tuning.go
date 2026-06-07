package prestige

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// tuning.go — chargement et exposition des paramètres de calage du système.
//
// Source : config/prestige/tuning.toml (modifiable sans redéploiement).
// Si le fichier est absent ou invalide, le système retombe sur des valeurs
// par défaut hardcodées (DefaultTuning) avec un slog.Warn.

// Tuning regroupe tous les paramètres tunables du système Prestige.
type Tuning struct {
	SchemaVersion int                `toml:"schema_version"`
	AntiSmurf     AntiSmurfTuning    `toml:"anti_smurf"`
	Stretch       StretchThresholds  `toml:"stretch_thresholds"`
	PopulationCap PopulationCaps     `toml:"population_caps"`
	PPAmounts     PPAmountsTuning    `toml:"pp_amounts"`
	DataTierMul   DataTierMultiplier `toml:"data_tier"`
	Levels        LevelsTuning       `toml:"levels"`
	Baseline      BaselineTuning     `toml:"baseline"`
	WinRateMin    WinRateMinMatches  `toml:"win_rate_min_matches"`
	Cooldowns     CooldownsTuning    `toml:"cooldowns"`
	QuotasPilote  QuotasPiloteTuning `toml:"quotas_pilote"`
	SquadPool     SquadPoolTuning    `toml:"squad_pool"`
	Suggestion    SuggestionTuning   `toml:"suggestion"`
	Expiration    ExpirationTuning   `toml:"expiration"`
	MatchCount    MatchCountTuning   `toml:"match_count"`
}

type AntiSmurfTuning struct {
	MinStretch             float64 `toml:"min_stretch"`
	PopulationMinThreshold int     `toml:"population_min_threshold"`
	BaselineStaleDays      int     `toml:"baseline_stale_days"`
	RecoveryMatches        int     `toml:"recovery_matches"`
}

type StretchThresholds struct {
	Normal    float64 `toml:"normal"`
	Heroic    float64 `toml:"heroic"`
	Legendary float64 `toml:"legendary"`
	Mythic    float64 `toml:"mythic"`
}

type PopulationCaps struct {
	BelowMedian string `toml:"below_median"`
	BelowP75    string `toml:"below_p75"`
	BelowP90    string `toml:"below_p90"`
	AboveP90    string `toml:"above_p90"`
}

type PPAmountsTuning struct {
	Normal          int     `toml:"normal"`
	Heroic          int     `toml:"heroic"`
	Legendary       int     `toml:"legendary"`
	Mythic          int     `toml:"mythic"`
	SquadMultiplier float64 `toml:"squad_multiplier"`
	// ArcCompletionBonusRatio est la fraction des PP cumulés des objectifs de
	// l'arc reversée en bonus à sa complétion (ex: 0.5 = +50 %). Remplace
	// l'ancien `arc_completion_bonus` (flat, = 1 objectif Mythic) qui rendait
	// un arc entier équivalent à un seul défi. 0 désactive le bonus.
	ArcCompletionBonusRatio float64 `toml:"arc_completion_bonus_ratio"`
	Streak3Sessions         int     `toml:"streak_3_sessions"`
	MatchPlayed             int     `toml:"match_played"`
	MatchWon                int     `toml:"match_won"`
	MedalMin                int     `toml:"medal_min"`
	MedalMax                int     `toml:"medal_max"`
}

type DataTierMultiplier struct {
	Full      float64 `toml:"full"`
	Estimated float64 `toml:"estimated"`
	Tracking  float64 `toml:"tracking"`
}

type LevelsTuning struct {
	Thresholds []int    `toml:"thresholds"`
	Names      []string `toml:"names"`
}

type BaselineTuning struct {
	WindowMatches    int `toml:"window_matches"`
	MatchesFull      int `toml:"matches_full"`
	MatchesEstimated int `toml:"matches_estimated"`
}

type WinRateMinMatches struct {
	Session    int `toml:"session"`
	Rolling7d  int `toml:"rolling_7d"`
	Rolling14d int `toml:"rolling_14d"`
	Rolling30d int `toml:"rolling_30d"`
}

type CooldownsTuning struct {
	ExpiredHours   int `toml:"expired_hours"`
	AbandonedHours int `toml:"abandoned_hours"`
	CompletedHours int `toml:"completed_hours"`
}

type QuotasPiloteTuning struct {
	DailyMax       int `toml:"daily_max"`
	WeeklyMax      int `toml:"weekly_max"`
	MonthlyMax     int `toml:"monthly_max"`
	TotalActiveMax int `toml:"total_active_max"`
	FreeNewPerDay  int `toml:"free_new_per_day"`
}

type SquadPoolTuning struct {
	RefreshPeriodDays int `toml:"refresh_period_days"`
	SizeMin           int `toml:"size_min"`
	SizeMax           int `toml:"size_max"`
}

type SuggestionTuning struct {
	NextPalierExtraStretch float64 `toml:"next_palier_extra_stretch"`
	AlternativesCount      int     `toml:"alternatives_count"`
}

// ExpirationTuning définit les durées d'expiration par tier pour les défis pilote.
// Les défis mode libre n'ont pas de timer (valeur ignorée pour ModeLibre).
type ExpirationTuning struct {
	NormalHours    int `toml:"normal_hours"`
	HeroicHours    int `toml:"heroic_hours"`
	LegendaryHours int `toml:"legendary_hours"`
	MythicHours    int `toml:"mythic_hours"`
}

// MatchCountTuning définit le nombre de matchs requis pour compléter
// un défi threshold selon son tier (WindowLastNMatches).
type MatchCountTuning struct {
	Normal    int `toml:"normal"`
	Heroic    int `toml:"heroic"`
	Legendary int `toml:"legendary"`
	Mythic    int `toml:"mythic"`
}

// DefaultTuning retourne les valeurs hardcodées de fallback.
//
// Doivent rester cohérentes avec config/prestige/tuning.toml. Si un
// fichier TOML est manquant ou corrompu, le système boot quand même
// avec ces valeurs (avec un slog.Warn).
func DefaultTuning() Tuning {
	return Tuning{
		SchemaVersion: 1,
		AntiSmurf: AntiSmurfTuning{
			MinStretch:             0.08,
			PopulationMinThreshold: 50,
			BaselineStaleDays:      60,
			RecoveryMatches:        10,
		},
		Stretch: StretchThresholds{
			Normal: 1.08, Heroic: 1.25, Legendary: 1.50, Mythic: 1.85,
		},
		PopulationCap: PopulationCaps{
			BelowMedian: "normal",
			BelowP75:    "heroic",
			BelowP90:    "legendary",
			AboveP90:    "mythic",
		},
		PPAmounts: PPAmountsTuning{
			Normal: 50, Heroic: 75, Legendary: 125, Mythic: 200,
			SquadMultiplier:         1.20,
			ArcCompletionBonusRatio: 0.5,
			Streak3Sessions:         30,
			MatchPlayed:             10,
			MatchWon:                15,
			MedalMin:                5,
			MedalMax:                20,
		},
		DataTierMul: DataTierMultiplier{
			Full: 1.0, Estimated: 0.5, Tracking: 0.0,
		},
		Levels: LevelsTuning{
			Thresholds: []int{0, 500, 1500, 3000, 6000, 12000},
			Names:      []string{"Recrue", "Soldat", "Vétéran", "Spécialiste", "Élite", "Légendaire"},
		},
		Baseline: BaselineTuning{
			WindowMatches: 20, MatchesFull: 10, MatchesEstimated: 5,
		},
		WinRateMin: WinRateMinMatches{
			Session: 5, Rolling7d: 8, Rolling14d: 15, Rolling30d: 25,
		},
		Cooldowns: CooldownsTuning{
			ExpiredHours: 12, AbandonedHours: 48, CompletedHours: 0,
		},
		QuotasPilote: QuotasPiloteTuning{
			DailyMax: 3, WeeklyMax: 5, MonthlyMax: 2,
			TotalActiveMax: 12, FreeNewPerDay: 3,
		},
		SquadPool: SquadPoolTuning{
			RefreshPeriodDays: 7, SizeMin: 6, SizeMax: 9,
		},
		Suggestion: SuggestionTuning{
			NextPalierExtraStretch: 0.15, AlternativesCount: 3,
		},
		Expiration: ExpirationTuning{
			NormalHours: 48, HeroicHours: 168, LegendaryHours: 336, MythicHours: 720,
		},
		MatchCount: MatchCountTuning{
			Normal: 5, Heroic: 15, Legendary: 30, Mythic: 50,
		},
	}
}

// LoadTuning charge tuning.toml depuis un chemin et retourne la struct.
//
// En cas d'erreur (fichier absent, parse error), retourne DefaultTuning()
// avec un slog.Warn — le système doit pouvoir booter même sans config.
func LoadTuning(path string) Tuning {
	raw, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("prestige: tuning.toml introuvable, fallback DefaultTuning",
			"path", path, "err", err)
		return DefaultTuning()
	}
	var t Tuning
	if err := toml.Unmarshal(raw, &t); err != nil {
		slog.Error("prestige: tuning.toml parse error, fallback DefaultTuning",
			"path", path, "err", err)
		return DefaultTuning()
	}
	if err := t.Validate(); err != nil {
		slog.Warn("prestige: tuning.toml invalide, fallback DefaultTuning",
			"path", path, "err", err)
		return DefaultTuning()
	}
	slog.Debug("prestige: tuning loaded", "path", path, "schema_version", t.SchemaVersion)
	return t
}

// Validate vérifie la cohérence des valeurs.
func (t Tuning) Validate() error {
	if err := t.validateAntiSmurfAndStretch(); err != nil {
		return err
	}
	if err := t.validatePPAndLevels(); err != nil {
		return err
	}
	if err := t.validateMatchCountAndExpiration(); err != nil {
		return err
	}
	if t.SquadPool.SizeMin > t.SquadPool.SizeMax {
		return fmt.Errorf("squad_pool size_min > size_max")
	}
	return nil
}

// validateAntiSmurfAndStretch valide les sections anti_smurf et stretch_thresholds.
func (t Tuning) validateAntiSmurfAndStretch() error {
	if t.AntiSmurf.MinStretch <= 0 {
		return fmt.Errorf("anti_smurf.min_stretch must be > 0")
	}
	if t.Stretch.Normal >= t.Stretch.Heroic ||
		t.Stretch.Heroic >= t.Stretch.Legendary ||
		t.Stretch.Legendary >= t.Stretch.Mythic {
		return fmt.Errorf("stretch_thresholds non monotones")
	}
	return nil
}

// validatePPAndLevels valide pp_amounts, levels et baseline.
func (t Tuning) validatePPAndLevels() error {
	if t.PPAmounts.Normal >= t.PPAmounts.Heroic ||
		t.PPAmounts.Heroic >= t.PPAmounts.Legendary ||
		t.PPAmounts.Legendary >= t.PPAmounts.Mythic {
		return fmt.Errorf("pp_amounts non monotones")
	}
	if t.PPAmounts.ArcCompletionBonusRatio < 0 {
		return fmt.Errorf("pp_amounts.arc_completion_bonus_ratio doit être >= 0")
	}
	if len(t.Levels.Thresholds) != len(t.Levels.Names) {
		return fmt.Errorf("levels.thresholds et levels.names de tailles différentes")
	}
	if len(t.Levels.Thresholds) == 0 {
		return fmt.Errorf("levels.thresholds vide")
	}
	if t.Levels.Thresholds[0] != 0 {
		return fmt.Errorf("levels.thresholds[0] doit être 0")
	}
	for i := 1; i < len(t.Levels.Thresholds); i++ {
		if t.Levels.Thresholds[i] <= t.Levels.Thresholds[i-1] {
			return fmt.Errorf("levels.thresholds non monotones à l'index %d", i)
		}
	}
	if t.Baseline.WindowMatches <= 0 {
		return fmt.Errorf("baseline.window_matches must be > 0")
	}
	return nil
}

// validateMatchCountAndExpiration valide match_count et expiration.
func (t Tuning) validateMatchCountAndExpiration() error {
	if t.MatchCount.Normal <= 0 || t.MatchCount.Normal >= t.MatchCount.Heroic ||
		t.MatchCount.Heroic >= t.MatchCount.Legendary ||
		t.MatchCount.Legendary >= t.MatchCount.Mythic {
		return fmt.Errorf("match_count non monotones ou invalides")
	}
	if t.Expiration.NormalHours <= 0 || t.Expiration.NormalHours >= t.Expiration.HeroicHours ||
		t.Expiration.HeroicHours >= t.Expiration.LegendaryHours ||
		t.Expiration.LegendaryHours >= t.Expiration.MythicHours {
		return fmt.Errorf("expiration non monotone ou invalide")
	}
	return nil
}

// CooldownDuration retourne la durée de cooldown selon l'issue d'un défi.
func (t Tuning) CooldownDuration(status ChallengeStatus) time.Duration {
	switch status {
	case StatusExpired:
		return time.Duration(t.Cooldowns.ExpiredHours) * time.Hour
	case StatusAbandoned:
		return time.Duration(t.Cooldowns.AbandonedHours) * time.Hour
	case StatusCompleted:
		return time.Duration(t.Cooldowns.CompletedHours) * time.Hour
	}
	return 0
}

// PPForTier retourne le PP de base pour un palier (sans data_tier ni squad_multiplier).
func (t Tuning) PPForTier(tier Tier) int {
	switch tier {
	case TierNormal:
		return t.PPAmounts.Normal
	case TierHeroic:
		return t.PPAmounts.Heroic
	case TierLegendary:
		return t.PPAmounts.Legendary
	case TierMythic:
		return t.PPAmounts.Mythic
	}
	return 0
}

// DataTierMultiplierFor retourne le multiplicateur PP selon DataTier.
func (t Tuning) DataTierMultiplierFor(dt DataTier) float64 {
	switch dt {
	case DataFull:
		return t.DataTierMul.Full
	case DataEstimated:
		return t.DataTierMul.Estimated
	case DataTracking:
		return t.DataTierMul.Tracking
	}
	return 0
}

// PopulationCapTier retourne le cap de palier selon la position dans la population.
//
// percentile attendu dans [0, 1]. Renvoie le palier maximum éligible.
func (t Tuning) PopulationCapTier(percentile float64) Tier {
	switch {
	case percentile < 0.5:
		return Tier(t.PopulationCap.BelowMedian)
	case percentile < 0.75:
		return Tier(t.PopulationCap.BelowP75)
	case percentile < 0.90:
		return Tier(t.PopulationCap.BelowP90)
	default:
		return Tier(t.PopulationCap.AboveP90)
	}
}

// WinRateMinForWindow retourne le minimum de matchs pour évaluer win_rate
// selon le type de fenêtre. 0 si pas de minimum applicable.
func (t Tuning) WinRateMinForWindow(wt WindowType, value string) int {
	switch wt {
	case WindowSession:
		return t.WinRateMin.Session
	case WindowRollingDays:
		switch value {
		case "7":
			return t.WinRateMin.Rolling7d
		case "14":
			return t.WinRateMin.Rolling14d
		case "30":
			return t.WinRateMin.Rolling30d
		}
	}
	return 0
}

// ExpirationDurationFor retourne la durée d'expiration d'un défi selon son tier et son mode.
// Retourne 0 pour le mode libre (pas de timer).
func (t Tuning) ExpirationDurationFor(tier Tier, mode ChallengeMode) time.Duration {
	if mode == ModeLibre {
		return 0
	}
	var hours int
	switch tier {
	case TierNormal:
		hours = t.Expiration.NormalHours
	case TierHeroic:
		hours = t.Expiration.HeroicHours
	case TierLegendary:
		hours = t.Expiration.LegendaryHours
	case TierMythic:
		hours = t.Expiration.MythicHours
	}
	return time.Duration(hours) * time.Hour
}

// RequiredMatchCount retourne le nombre de matchs nécessaires pour compléter
// un défi threshold de type WindowLastNMatches selon son tier.
func (t Tuning) RequiredMatchCount(tier Tier) int {
	switch tier {
	case TierNormal:
		return t.MatchCount.Normal
	case TierHeroic:
		return t.MatchCount.Heroic
	case TierLegendary:
		return t.MatchCount.Legendary
	case TierMythic:
		return t.MatchCount.Mythic
	}
	return t.MatchCount.Normal
}
