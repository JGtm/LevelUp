// Package domain — types métier pour les settings et le setup joueur.
// Sprint 16 : GET /settings, PATCH /settings, POST /settings/media/reset-index,
//
//	POST /setup/players, POST /setup/smoke-test.
package domain

// SettingsResponse est le payload retourné par GET /settings.
// discord_webhook_url n'est JAMAIS inclus — seulement discord_webhook_url_present.
type SettingsResponse struct {
	Lang                 string `json:"lang"`
	DiscordLang          string `json:"discord_lang"`
	UserTimezone         string `json:"user_timezone"`
	NormalizeModeLabels  bool   `json:"normalize_mode_labels"`
	ShowRecords          bool   `json:"show_records"`
	RefreshClearsCaches  bool   `json:"refresh_clears_caches"`
	CareerTopExcludeBTB  bool   `json:"career_top_exclude_btb"`
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
	// MediaDeleteSourceAfterTranscode : valeur EFFECTIVE résolue (bool, pas un
	// pointeur) de la politique de suppression du source après transcodage HLS.
	// Résolue par le handler via config.ResolveMediaDeleteSource (env > store > isProd).
	MediaDeleteSourceAfterTranscode    bool     `json:"media_delete_source_after_transcode"`
	MediaToleranceMinutes              int      `json:"media_tolerance_minutes"`
	MediaWatcherEnabled                bool     `json:"media_watcher_enabled"`
	MediaWatcherDebounceSeconds        int      `json:"media_watcher_debounce_seconds"`
	DiscordNotificationsEnabled        bool     `json:"discord_notifications_enabled"`
	DiscordWebhookURLPresent           bool     `json:"discord_webhook_url_present"`
	DiscordNotifySync                  bool     `json:"discord_notify_sync"`
	DiscordNotifyBackfill              bool     `json:"discord_notify_backfill"`
	DiscordNotifyNewVersion            bool     `json:"discord_notify_new_version"`
	DiscordNotifyFriends               bool     `json:"discord_notify_friends"` // §6.B
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

	// --- Règles de sessions ---
	SessionGapMinutes          int    `json:"session_gap_minutes"`
	SessionSplitOnRankedChange bool   `json:"session_split_on_ranked_change"`
	SessionTeamChangeMode      string `json:"session_team_change_mode"`

	// --- Règles de badges narratifs ---
	OutcomeExcludeBotMatchesFromBadges  bool   `json:"outcome_exclude_bot_matches_from_badges"`
	OutcomeExcludeBotMatchesFromRecords bool   `json:"outcome_exclude_bot_matches_from_records"`
	OutcomeBadgeSensitivity             string `json:"outcome_badge_sensitivity"`

	// RendementExcludeAssists : si true, le rendement combat (OffensiveConversion)
	// est calculé sans les assistances (= 225×kills/dégâts) sur TOUS les composants
	// rendement. Défaut false (assists comptés à 1/3, convention Halo).
	RendementExcludeAssists bool `json:"rendement_exclude_assists"`

	// ShowProgression contrôle l'affichage du système Objectifs/Prestige
	// (section Accueil + entrée nav L1). Défaut : true.
	ShowProgression bool `json:"show_progression"`

	// CoachProactiveMode active la proposition automatique de challenges/arcs
	// Prestige par le coach (cf. ADR 0020, DEC-2). Défaut : true (bascule
	// 2026-07-22) ; opt-out explicite (false) respecté.
	CoachProactiveMode bool `json:"coach_proactive_mode"`

	// AuthProvider indique le mécanisme d'authentification actif.
	// Valeurs : "msal" (défaut) | "sisu" (Xbox natif).
	AuthProvider string `json:"auth_provider"`

	// InstanceLocked : instance fermée (aucune nouvelle identité/BDD). Reflète
	// app_settings.json:instance_locked (pas le verrou env forcé, exposé séparément
	// au /bootstrap). Défaut : false.
	InstanceLocked bool `json:"instance_locked"`
}

// UpdateSettingsRequest contient les champs modifiables (tous optionnels).
// discord_webhook_url peut être envoyé ici (écriture) mais n'est jamais retourné.
type UpdateSettingsRequest struct {
	Lang                 *string `json:"lang,omitempty"`
	DiscordLang          *string `json:"discord_lang,omitempty"`
	UserTimezone         *string `json:"user_timezone,omitempty"`
	NormalizeModeLabels  *bool   `json:"normalize_mode_labels,omitempty"`
	ShowRecords          *bool   `json:"show_records,omitempty"`
	RefreshClearsCaches  *bool   `json:"refresh_clears_caches,omitempty"`
	CareerTopExcludeBTB  *bool   `json:"career_top_exclude_btb,omitempty"`
	MediaCapturesBaseDir *string `json:"media_captures_base_dir,omitempty"`
	// MediaDeleteSourceAfterTranscode : *bool (nil = auto). PATCH persiste le pointeur
	// dans app_settings.json ; la résolution effective reste au déclenchement.
	MediaDeleteSourceAfterTranscode    *bool    `json:"media_delete_source_after_transcode,omitempty"`
	MediaToleranceMinutes              *int     `json:"media_tolerance_minutes,omitempty"`
	MediaWatcherEnabled                *bool    `json:"media_watcher_enabled,omitempty"`
	MediaWatcherDebounceSeconds        *int     `json:"media_watcher_debounce_seconds,omitempty"`
	DiscordNotificationsEnabled        *bool    `json:"discord_notifications_enabled,omitempty"`
	DiscordWebhookURL                  *string  `json:"discord_webhook_url,omitempty"` // écriture seule
	DiscordNotifySync                  *bool    `json:"discord_notify_sync,omitempty"`
	DiscordNotifyBackfill              *bool    `json:"discord_notify_backfill,omitempty"`
	DiscordNotifyNewVersion            *bool    `json:"discord_notify_new_version,omitempty"`
	DiscordNotifyFriends               *bool    `json:"discord_notify_friends,omitempty"` // §6.B
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

	// --- Règles de sessions ---
	SessionGapMinutes          *int    `json:"session_gap_minutes,omitempty"`
	SessionSplitOnRankedChange *bool   `json:"session_split_on_ranked_change,omitempty"`
	SessionTeamChangeMode      *string `json:"session_team_change_mode,omitempty"`

	// --- Règles de badges narratifs ---
	OutcomeExcludeBotMatchesFromBadges  *bool   `json:"outcome_exclude_bot_matches_from_badges,omitempty"`
	OutcomeExcludeBotMatchesFromRecords *bool   `json:"outcome_exclude_bot_matches_from_records,omitempty"`
	OutcomeBadgeSensitivity             *string `json:"outcome_badge_sensitivity,omitempty"`

	// RendementExcludeAssists : toggle rendement combat sans assistances.
	RendementExcludeAssists *bool `json:"rendement_exclude_assists,omitempty"`

	// ShowProgression : toggle d'affichage Objectifs/Prestige.
	ShowProgression *bool `json:"show_progression,omitempty"`

	// CoachProactiveMode : toggle pont coach → Prestige (cf. ADR 0020).
	CoachProactiveMode *bool `json:"coach_proactive_mode,omitempty"`

	// AuthProvider bascule le mécanisme d'authentification. "msal" | "sisu".
	AuthProvider *string `json:"auth_provider,omitempty"`

	// InstanceLocked : verrou « instance fermée » activable à chaud (admin).
	InstanceLocked *bool `json:"instance_locked,omitempty"`
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
	// InitialMaxMatches : nombre de matchs à synchroniser à l'onboarding pour ce
	// (joueur, titre). 0 = défaut. Persisté dans db_profiles.json (Pass B).
	InitialMaxMatches int `json:"initial_max_matches,omitempty"`
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
	MaxMatches int    `json:"max_matches"` // 1-2000 ; 0 = défaut profil (initial_max_matches) puis 200
	// TitleSlug : titre cible (multi-titre). Vide → titre du contexte (header/
	// session) puis halo_infinite. Lève l'ambiguïté quand un gamertag existe sous
	// plusieurs titres et cible la bonne DB.
	TitleSlug string `json:"title_slug,omitempty"`
}
