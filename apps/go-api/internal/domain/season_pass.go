// Package domain — season_pass.go : types pour la page Season Pass (palmares).
package domain

// SeasonPassStatus représente l'état d'avancement d'un Battle Pass.
type SeasonPassStatus string

const (
	SeasonPassStatusActive     SeasonPassStatus = "active"
	SeasonPassStatusInProgress SeasonPassStatus = "in_progress"
	SeasonPassStatusCompleted  SeasonPassStatus = "completed"
	SeasonPassStatusNotStarted SeasonPassStatus = "not_started"
)

// SeasonPassItemSummary représente un item de récompense (image + titre).
//
// Quality est la rareté brute renvoyée par GameCMS ("Common", "Rare", "Epic",
// "Legendary", "Mythic"). ItemType est la catégorie brute ("ArmorCoating",
// "WeaponCharm", "SpartanEmblem"…). Le frontend assure le mapping vers des
// libellés/couleurs localisés.
type SeasonPassItemSummary struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
	Quality     *string `json:"quality,omitempty"`
	ItemType    *string `json:"item_type,omitempty"`
}

// SeasonPassTierSummary représente un palier individuel avec son visuel principal.
type SeasonPassTierSummary struct {
	Rank        int                     `json:"rank"`
	Title       string                  `json:"title"`
	Description *string                 `json:"description,omitempty"`
	ImageURL    *string                 `json:"image_url,omitempty"`
	Quality     *string                 `json:"quality,omitempty"`
	ItemType    *string                 `json:"item_type,omitempty"`
	IsObtained  bool                    `json:"is_obtained"`
	IsCurrent   bool                    `json:"is_current"`
	IsPremium   bool                    `json:"is_premium"`
	FreeRewards []SeasonPassItemSummary `json:"free_rewards,omitempty"`
}

// SeasonPassTrackSummary résume un Battle Pass / Operation Reward Track.
// Les champs sont un mirror exact des types TypeScript côté frontend.
type SeasonPassTrackSummary struct {
	RewardTrackPath           string                  `json:"reward_track_path"`
	Name                      string                  `json:"name"`
	Description               *string                 `json:"description,omitempty"`
	Status                    SeasonPassStatus        `json:"status"`
	IsActive                  bool                    `json:"is_active"`
	IsOwned                   bool                    `json:"is_owned"`
	HasReachedMaxRank         bool                    `json:"has_reached_max_rank"`
	CurrentRank               int                     `json:"current_rank"`
	PartialProgress           int                     `json:"partial_progress"`
	XPPerRank                 *int                    `json:"xp_per_rank,omitempty"`
	MaxRank                   *int                    `json:"max_rank,omitempty"`
	CompletionPercent         *float64                `json:"completion_percent,omitempty"`
	ActiveTierRank            *int                    `json:"active_tier_rank,omitempty"`
	ActiveTierProgressPercent *float64                `json:"active_tier_progress_percent,omitempty"`
	ImageURL                  *string                 `json:"image_url,omitempty"`
	BackgroundImageURL        *string                 `json:"background_image_url,omitempty"`
	Tiers                     []SeasonPassTierSummary `json:"tiers,omitempty"`
}

// SeasonPassPageResponse est la réponse de l'endpoint /pages/palmares/season-pass.
type SeasonPassPageResponse struct {
	TitleSlug       string                   `json:"title_slug"`
	Available       bool                     `json:"available"`
	ErrorHint       *string                  `json:"error_hint,omitempty"`
	ActiveTrackPath *string                  `json:"active_track_path,omitempty"`
	Challenges      ChallengesResponse       `json:"challenges"`
	Passes          []SeasonPassTrackSummary `json:"passes"`
}
