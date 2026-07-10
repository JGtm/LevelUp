package notifications

import (
	"encoding/json"
	"fmt"
	"time"
)

// Category identifie le type d'événement à l'origine de la notification.
// Doit rester en sync avec :
//   - migration steps_player_notifications.go (seed des préférences)
//   - frontend features/notifications/i18n.ts (clés notif.<category>.title)
type Category string

const (
	CategoryAppRelease  Category = "app_release"
	CategoryMatchSynced Category = "match_synced"
	CategoryMediaAdded  Category = "media_added"
	CategoryMediaLiked  Category = "media_liked"
	// CategoryObjectiveAssigned : conservée pour rétro-compat des notifs déjà en DB
	// + seed des préférences — n'est PLUS émise depuis 2026-07 (DP2 : doublon exact
	// de objective_completed, les deux étaient branchés sur PersonalAwardCount).
	// Modèle : CategorySeasonPassLevel.
	CategoryObjectiveAssigned  Category = "objective_assigned"
	CategoryObjectiveCompleted Category = "objective_completed"
	CategoryChallengeAdded     Category = "challenge_added"
	CategoryChallengeCompleted Category = "challenge_completed"
	// CategorySeasonPassLevel : conservée pour rétro-compat des notifs déjà en DB
	// — n'est plus émise depuis 2026-05-16 (remplacée par CategoryCareerRank pour
	// le rang carrière Halo et CategoryBattlepassCompleted pour les tracks BP).
	CategorySeasonPassLevel  Category = "season_pass_level"
	CategorySyncError        Category = "sync_error"
	CategoryPersonalRecord   Category = "personal_record"
	CategoryThresholdCrossed Category = "threshold_crossed"
	// §6 Squad/Sessions overhaul : flow ami.
	CategoryFriendAdded         Category = "friend_added"
	CategoryFriendSyncCompleted Category = "friend_sync_completed"
	// 2026-05-08 : audit santé DB périodique (warnings remontés à l'admin).
	CategoryDataHealthWarning Category = "data_health_warning"
	// 2026-05-16 : extension notifications — rang Halo, skill CSR/LUSR, BP,
	// citations/commendations à paliers.
	CategoryCareerRank          Category = "career_rank"
	CategorySkillTier           Category = "skill_tier"
	CategoryBattlepassCompleted Category = "battlepass_completed"
	CategoryCitationTier        Category = "citation_tier"
	CategoryCitationMastery     Category = "citation_mastery"
	// 2026-05-18 : couche 3 du plan PROGRESSION_TRACKING (Ascension V2) —
	// alertes du coach proactif. Les catégories `personal_record` et
	// `threshold_crossed` existaient déjà ; ces 6 sont nouvelles.
	CategoryRecordNearMiss    Category = "record_near_miss"
	CategoryMilestoneUnlocked Category = "milestone_unlocked"
	CategoryMilestoneNearMiss Category = "milestone_near_miss"
	CategoryLUSRTierApproach  Category = "lusr_tier_approach"
	CategoryStreakMilestone   Category = "streak_milestone"
	CategoryComebackWelcome   Category = "comeback_welcome"
	// 2026-06-09 : coach soft-négatif (Coach V3 Phase A) — axe en baisse soutenue
	// « à consolider ». Catégorie NEUTRE distincte de threshold_crossed (positif) :
	// une tendance descendante ne doit pas s'afficher « Palier franchi ».
	CategoryTrendConsolidate Category = "trend_consolidate"
	// 2026-05-24 : Pattern Engine v3 (PLAN_PATTERN_ENGINE.md phases 1-3).
	CategoryPatternStrength Category = "pattern_strength"
	CategoryPatternWeakness Category = "pattern_weakness"
	CategoryPatternBehavior Category = "pattern_behavior"
	CategoryPatternLever    Category = "pattern_lever"
	// 2026-05-24 : Combat Profile Wiring Phase 5 — alertes proactives OC/DR/activité.
	CategoryCombatPattern Category = "combat_pattern"
	// 2026-06-23 : MT-19 / axe E — « titre prêt ». Émise à la première synchro d'un
	// titre fraîchement activé (Halo 5+) ayant produit des matchs, dans le flux du
	// titre par défaut (away-case : l'utilisateur a été invité à retourner sur Halo
	// Infinite le temps du backfill). title-agnostic : le titre voyage en Params.
	CategoryTitleReady Category = "title_ready"
)

// AllCategories retourne toutes les catégories MVP (utile pour les tests et le seed).
func AllCategories() []Category {
	return []Category{
		CategoryAppRelease, CategoryMatchSynced, CategoryMediaAdded,
		CategoryMediaLiked,
		CategoryObjectiveAssigned, CategoryObjectiveCompleted,
		CategoryChallengeAdded, CategoryChallengeCompleted,
		CategorySeasonPassLevel, CategorySyncError,
		CategoryPersonalRecord, CategoryThresholdCrossed,
		CategoryFriendAdded, CategoryFriendSyncCompleted,
		CategoryCareerRank, CategorySkillTier,
		CategoryBattlepassCompleted,
		CategoryCitationTier, CategoryCitationMastery,
		// Progression V2 — coach proactif.
		CategoryRecordNearMiss, CategoryMilestoneUnlocked,
		CategoryMilestoneNearMiss, CategoryLUSRTierApproach,
		CategoryStreakMilestone, CategoryComebackWelcome,
		CategoryTrendConsolidate,
		// Pattern Engine v3.
		CategoryPatternStrength, CategoryPatternWeakness,
		CategoryPatternBehavior, CategoryPatternLever,
		// Combat Profile Phase 5.
		CategoryCombatPattern,
		// MT-19 / axe E.
		CategoryTitleReady,
	}
}

// Severity indique le niveau visuel de la notification côté UI.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarn    Severity = "warn"
	SeverityError   Severity = "error"
)

// Delivery indique le canal de livraison côté client (toast, in-app, les deux, off).
type Delivery string

const (
	DeliveryBoth  Delivery = "both"
	DeliveryInApp Delivery = "inapp"
	DeliveryToast Delivery = "toast"
	DeliveryOff   Delivery = "off"
)

// Actor identifie l'auteur d'un événement (ex: l'uploader pour media_added).
type Actor struct {
	XUID string `json:"xuid"`
	Name string `json:"name"`
}

// Notification est le modèle exposé par l'API HTTP.
//
// title_key et body_key sont des clés i18n résolues côté client à partir
// de params (zéro FR/EN en dur dans la DB ou les payloads — cf. audit V7).
type Notification struct {
	ID           int64           `json:"id"`
	Category     Category        `json:"category"`
	Severity     Severity        `json:"severity"`
	TitleKey     string          `json:"title_key"`
	BodyKey      string          `json:"body_key,omitempty"`
	Params       json.RawMessage `json:"params,omitempty"`
	TargetRoute  string          `json:"target_route,omitempty"`
	TargetSearch json.RawMessage `json:"target_search,omitempty"`
	Actor        *Actor          `json:"actor,omitempty"`
	Source       string          `json:"source"`
	CreatedAt    time.Time       `json:"created_at"`
	ReadAt       *time.Time      `json:"read_at,omitempty"`
}

// Preference représente l'état stocké d'une catégorie dans notification_preferences.
type Preference struct {
	Category Category `json:"category"`
	Enabled  bool     `json:"enabled"`
	Delivery Delivery `json:"delivery"`
}

// EmitInput est le payload passé à Emitter.Emit. Tous les champs sont
// optionnels sauf Category, TitleKey et Source.
type EmitInput struct {
	Category     Category
	Severity     Severity
	TitleKey     string
	BodyKey      string
	Params       map[string]any
	TargetRoute  string
	TargetSearch map[string]any
	Actor        *Actor
	Source       string
}

// ListFilter contrôle le résultat de Service.List.
type ListFilter struct {
	UnreadOnly bool
	Category   Category // vide = pas de filtre
	Limit      int      // <=0 ou >maxLimit → fallback aux defauts
	BeforeID   int64    // 0 = pas de cursor
}

// ListResult est la réponse paginée de Service.List.
type ListResult struct {
	Items      []Notification `json:"items"`
	NextCursor *int64         `json:"next_cursor,omitempty"`
}

// UnreadCount expose le total et la répartition par catégorie.
type UnreadCount struct {
	Count      int            `json:"count"`
	ByCategory map[string]int `json:"by_category"`
}

// MarkResult est le retour des opérations bulk MarkRead / MarkAllRead.
type MarkResult struct {
	Updated int `json:"updated"`
}

// Validate vérifie qu'un EmitInput est suffisamment renseigné pour être inséré.
// Renvoie une erreur descriptive sur premier champ manquant.
func (e EmitInput) Validate() error {
	if e.Category == "" {
		return fmt.Errorf("emit: category required")
	}
	if e.TitleKey == "" {
		return fmt.Errorf("emit: title_key required")
	}
	if e.Source == "" {
		return fmt.Errorf("emit: source required")
	}
	return nil
}

// EncodedParams sérialise Params en JSON, retournant nil si la map est vide.
// Plafonne à maxParamsBytes pour éviter qu'un bug d'émission ne bloate la DB.
func (e EmitInput) EncodedParams() ([]byte, error) {
	return encodeMap(e.Params)
}

// EncodedTargetSearch sérialise TargetSearch en JSON (même règles que Params).
func (e EmitInput) EncodedTargetSearch() ([]byte, error) {
	return encodeMap(e.TargetSearch)
}

const maxParamsBytes = 4 * 1024 // 4 KB

func encodeMap(m map[string]any) ([]byte, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	if len(b) > maxParamsBytes {
		return nil, fmt.Errorf("encode payload: size %d exceeds cap %d", len(b), maxParamsBytes)
	}
	return b, nil
}
