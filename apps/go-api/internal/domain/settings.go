// Package domain — types métier pour les settings et le setup joueur.
// Sprint 16 : GET /settings, PATCH /settings, POST /settings/media/reset-index,
//
//	POST /setup/players, POST /setup/smoke-test.
package domain

// SettingsResponse est le payload retourné par GET /settings.
// discord_webhook_url n'est JAMAIS inclus — seulement discord_webhook_url_present.
type SettingsResponse struct {
	Lang                               string   `json:"lang"`
	DiscordLang                        string   `json:"discord_lang"`
	UserTimezone                       string   `json:"user_timezone"`
	NormalizeModeLabels                bool     `json:"normalize_mode_labels"`
	ShowRecords                        bool     `json:"show_records"`
	RefreshClearsCaches                bool     `json:"refresh_clears_caches"`
	CareerTopExcludeBTB                bool     `json:"career_top_exclude_btb"`
	MediaCapturesBaseDir               string   `json:"media_captures_base_dir"`
	MediaBufferMinutes                 int      `json:"media_buffer_minutes"`
	DiscordNotificationsEnabled        bool     `json:"discord_notifications_enabled"`
	DiscordWebhookURLPresent           bool     `json:"discord_webhook_url_present"`
	DiscordNotifySync                  bool     `json:"discord_notify_sync"`
	DiscordNotifyBackfill              bool     `json:"discord_notify_backfill"`
	DiscordNotifyNewVersion            bool     `json:"discord_notify_new_version"`
	DiscordNotifyNewMedia              bool     `json:"discord_notify_new_media"`
	SpnkrAutoSyncEnabled               bool     `json:"spnkr_auto_sync_enabled"`
	SpnkrAutoSyncIntervalHours         int      `json:"spnkr_auto_sync_interval_hours"`
	SpnkrAutoSyncIntervalMinutes       int      `json:"spnkr_auto_sync_interval_minutes"`
	WatcherPresenceEnabled             bool     `json:"watcher_presence_enabled"`
	WatcherSubscribedPlayers           []string `json:"watcher_subscribed_players"`
	SpnkrRefreshWithBackfill           bool     `json:"spnkr_refresh_with_backfill"`
	SpnkrRefreshBackfillMedals         bool     `json:"spnkr_refresh_backfill_medals"`
	SpnkrRefreshBackfillSkill          bool     `json:"spnkr_refresh_backfill_skill"`
	SpnkrRefreshBackfillAliases        bool     `json:"spnkr_refresh_backfill_aliases"`
	SpnkrRefreshBackfillPersonalScores bool     `json:"spnkr_refresh_backfill_personal_scores"`
	SpnkrRefreshBackfillPerfScores     bool     `json:"spnkr_refresh_backfill_performance_scores"`
	SpnkrRefreshBackfillLUSR           bool     `json:"spnkr_refresh_backfill_lusr"`
	SpnkrRefreshBackfillEvents         bool     `json:"spnkr_refresh_backfill_events"`
	SpnkrRefreshBackfillWeapons        bool     `json:"spnkr_refresh_backfill_weapons"`
	FriendGamertags                    []string `json:"friend_gamertags"`
}

// UpdateSettingsRequest contient les champs modifiables (tous optionnels).
// discord_webhook_url peut être envoyé ici (écriture) mais n'est jamais retourné.
type UpdateSettingsRequest struct {
	Lang                               *string  `json:"lang,omitempty"`
	DiscordLang                        *string  `json:"discord_lang,omitempty"`
	UserTimezone                       *string  `json:"user_timezone,omitempty"`
	NormalizeModeLabels                *bool    `json:"normalize_mode_labels,omitempty"`
	ShowRecords                        *bool    `json:"show_records,omitempty"`
	RefreshClearsCaches                *bool    `json:"refresh_clears_caches,omitempty"`
	CareerTopExcludeBTB                *bool    `json:"career_top_exclude_btb,omitempty"`
	MediaCapturesBaseDir               *string  `json:"media_captures_base_dir,omitempty"`
	MediaBufferMinutes                 *int     `json:"media_buffer_minutes,omitempty"`
	DiscordNotificationsEnabled        *bool    `json:"discord_notifications_enabled,omitempty"`
	DiscordWebhookURL                  *string  `json:"discord_webhook_url,omitempty"` // écriture seule
	DiscordNotifySync                  *bool    `json:"discord_notify_sync,omitempty"`
	DiscordNotifyBackfill              *bool    `json:"discord_notify_backfill,omitempty"`
	DiscordNotifyNewVersion            *bool    `json:"discord_notify_new_version,omitempty"`
	DiscordNotifyNewMedia              *bool    `json:"discord_notify_new_media,omitempty"`
	SpnkrAutoSyncEnabled               *bool    `json:"spnkr_auto_sync_enabled,omitempty"`
	SpnkrAutoSyncIntervalHours         *int     `json:"spnkr_auto_sync_interval_hours,omitempty"`
	SpnkrAutoSyncIntervalMinutes       *int     `json:"spnkr_auto_sync_interval_minutes,omitempty"`
	WatcherPresenceEnabled             *bool    `json:"watcher_presence_enabled,omitempty"`
	WatcherSubscribedPlayers           []string `json:"watcher_subscribed_players,omitempty"`
	SpnkrRefreshWithBackfill           *bool    `json:"spnkr_refresh_with_backfill,omitempty"`
	SpnkrRefreshBackfillMedals         *bool    `json:"spnkr_refresh_backfill_medals,omitempty"`
	SpnkrRefreshBackfillSkill          *bool    `json:"spnkr_refresh_backfill_skill,omitempty"`
	SpnkrRefreshBackfillAliases        *bool    `json:"spnkr_refresh_backfill_aliases,omitempty"`
	SpnkrRefreshBackfillPersonalScores *bool    `json:"spnkr_refresh_backfill_personal_scores,omitempty"`
	SpnkrRefreshBackfillPerfScores     *bool    `json:"spnkr_refresh_backfill_performance_scores,omitempty"`
	SpnkrRefreshBackfillLUSR           *bool    `json:"spnkr_refresh_backfill_lusr,omitempty"`
	SpnkrRefreshBackfillEvents         *bool    `json:"spnkr_refresh_backfill_events,omitempty"`
	SpnkrRefreshBackfillWeapons        *bool    `json:"spnkr_refresh_backfill_weapons,omitempty"`
	FriendGamertags                    []string `json:"friend_gamertags,omitempty"`
}

// MediaResetRequest est le corps de POST /settings/media/reset-index.
type MediaResetRequest struct {
	ConfirmDestructive bool `json:"confirm_destructive"`
	ReindexAfterReset  bool `json:"reindex_after_reset"`
}

// CreatePlayerProfileRequest est le corps de POST /setup/players.
type CreatePlayerProfileRequest struct {
	Gamertag    string `json:"gamertag"`
	XUID        string `json:"xuid,omitempty"`
	ProfileMode string `json:"profile_mode"`         // "xbox" | "azure_manual"
	TitleSlug   string `json:"title_slug,omitempty"` // Sprint 44 : titre cible (défaut: "halo_infinite")
}

// CreatePlayerProfileResponse est la réponse de POST /setup/players (201).
type CreatePlayerProfileResponse struct {
	Player    PlayerSummary `json:"player"`
	DBCreated bool          `json:"db_created"`
	Warnings  []string      `json:"warnings,omitempty"`
}

// InitialSyncStartRequest est le corps de POST /sync/initial.
type InitialSyncStartRequest struct {
	PlayerSlug string `json:"player_slug"`
	MaxMatches int    `json:"max_matches"` // 1-2000, default 200
}
