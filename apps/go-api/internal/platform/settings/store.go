// Package settings fournit la lecture/écriture de app_settings.json.
// Sprint 16 : GET /settings, PATCH /settings.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"levelup/go-api/internal/domain"
)

// AppSettings représente la structure complète de app_settings.json.
// Seuls les champs exposés par l'API sont typés — les champs inconnus sont
// préservés dans Extra pour ne pas les effacer lors d'un PATCH.
type AppSettings struct {
	Lang                               string   `json:"lang"`
	DiscordLang                        string   `json:"discord_lang"`
	UserTimezone                       string   `json:"user_timezone"`
	NormalizeModeLabels                bool     `json:"normalize_mode_labels"`
	ShowRecords                        bool     `json:"show_records"`
	RefreshClearsCaches                bool     `json:"refresh_clears_caches"`
	CareerTopExcludeBTB                bool     `json:"career_top_exclude_btb"`
	MediaCapturesBaseDir               string   `json:"media_captures_base_dir"`
	MediaBufferMinutes                 int      `json:"media_buffer_minutes"`
	MediaWatcherEnabled                bool     `json:"media_watcher_enabled"`
	MediaWatcherDebounceSeconds        int      `json:"media_watcher_debounce_seconds"`
	DiscordNotificationsEnabled        bool     `json:"discord_notifications_enabled"`
	DiscordWebhookURL                  string   `json:"discord_webhook_url"` // jamais exposé côté API
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
	// Escouade — gamertags des amis par défaut
	FriendGamertags []string `json:"friend_gamertags"`

	// --- Règles de sessions ---
	SessionGapMinutes          int    `json:"session_gap_minutes"`
	SessionSplitOnRankedChange bool   `json:"session_split_on_ranked_change"`
	SessionTeamChangeMode      string `json:"session_team_change_mode"`

	// --- Règles de badges narratifs ---
	OutcomeExcludeBotMatchesFromBadges  bool   `json:"outcome_exclude_bot_matches_from_badges"`
	OutcomeExcludeBotMatchesFromRecords bool   `json:"outcome_exclude_bot_matches_from_records"`
	OutcomeBadgeSensitivity             string `json:"outcome_badge_sensitivity"`

	// Capabilities (défaut : true)
	CanSelfProvision    bool `json:"can_self_provision"`
	CanStartInitialSync bool `json:"can_start_initial_sync"`

	// AuthProvider détermine le mécanisme d'authentification Microsoft/Halo.
	// Valeurs : "msal" (défaut, Azure app) | "sisu" (Xbox natif, sans Azure app).
	AuthProvider string `json:"auth_provider"`

	// raw conserve tous les autres champs pour ne pas les perdre au Save
	raw map[string]json.RawMessage
}

// Store gère la lecture/écriture de app_settings.json.
type Store struct {
	mu   sync.RWMutex
	path string
}

// NewStore crée un Store pour le fichier donné.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load lit app_settings.json et retourne un AppSettings.
// Les champs inconnus sont préservés pour ne pas les perdre lors du Save.
func (s *Store) Load() (*AppSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSettings(), nil
		}
		return nil, fmt.Errorf("settings.Load: %w", err)
	}

	// Parse brut pour conserver les champs inconnus
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("settings.Load json: %w", err)
	}

	// Parse typé pour les champs connus
	cfg := defaultSettings()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("settings.Load typed: %w", err)
	}
	cfg.raw = raw

	// can_self_provision absent → default true
	if _, ok := raw["can_self_provision"]; !ok {
		cfg.CanSelfProvision = true
	}
	// can_start_initial_sync absent → default true
	if _, ok := raw["can_start_initial_sync"]; !ok {
		cfg.CanStartInitialSync = true
	}

	return cfg, nil
}

// Save persiste app_settings.json en préservant les champs inconnus.
func (s *Store) Save(cfg *AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Partir de la copie brute si elle existe
	out := make(map[string]json.RawMessage)
	for k, v := range cfg.raw {
		out[k] = v
	}

	// Marshal des champs connus et écrasement des clés correspondantes
	typed, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("settings.Save marshal: %w", err)
	}
	var typedMap map[string]json.RawMessage
	if err := json.Unmarshal(typed, &typedMap); err != nil {
		return fmt.Errorf("settings.Save unmarshal typed: %w", err)
	}
	for k, v := range typedMap {
		out[k] = v
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("settings.Save indent: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("settings.Save write: %w", err)
	}
	return nil
}

// Apply applique un UpdateSettingsRequest partiel sur un AppSettings existant.
func Apply(cfg *AppSettings, req *domain.UpdateSettingsRequest) {
	if req.Lang != nil {
		cfg.Lang = *req.Lang
	}
	if req.DiscordLang != nil {
		cfg.DiscordLang = *req.DiscordLang
	}
	if req.UserTimezone != nil {
		cfg.UserTimezone = *req.UserTimezone
	}
	if req.NormalizeModeLabels != nil {
		cfg.NormalizeModeLabels = *req.NormalizeModeLabels
	}
	if req.ShowRecords != nil {
		cfg.ShowRecords = *req.ShowRecords
	}
	if req.RefreshClearsCaches != nil {
		cfg.RefreshClearsCaches = *req.RefreshClearsCaches
	}
	if req.CareerTopExcludeBTB != nil {
		cfg.CareerTopExcludeBTB = *req.CareerTopExcludeBTB
	}
	if req.MediaCapturesBaseDir != nil {
		cfg.MediaCapturesBaseDir = *req.MediaCapturesBaseDir
	}
	if req.MediaToleranceMinutes != nil {
		cfg.MediaBufferMinutes = *req.MediaToleranceMinutes
	}
	if req.MediaWatcherEnabled != nil {
		cfg.MediaWatcherEnabled = *req.MediaWatcherEnabled
	}
	if req.MediaWatcherDebounceSeconds != nil {
		cfg.MediaWatcherDebounceSeconds = *req.MediaWatcherDebounceSeconds
	}
	if req.DiscordNotificationsEnabled != nil {
		cfg.DiscordNotificationsEnabled = *req.DiscordNotificationsEnabled
	}
	if req.DiscordWebhookURL != nil {
		cfg.DiscordWebhookURL = *req.DiscordWebhookURL
	}
	if req.DiscordNotifySync != nil {
		cfg.DiscordNotifySync = *req.DiscordNotifySync
	}
	if req.DiscordNotifyBackfill != nil {
		cfg.DiscordNotifyBackfill = *req.DiscordNotifyBackfill
	}
	if req.DiscordNotifyNewVersion != nil {
		cfg.DiscordNotifyNewVersion = *req.DiscordNotifyNewVersion
	}
	if req.DiscordNotifyNewMedia != nil {
		cfg.DiscordNotifyNewMedia = *req.DiscordNotifyNewMedia
	}
	if req.SpnkrAutoSyncEnabled != nil {
		cfg.SpnkrAutoSyncEnabled = *req.SpnkrAutoSyncEnabled
	}
	if req.SpnkrAutoSyncIntervalHours != nil {
		cfg.SpnkrAutoSyncIntervalHours = *req.SpnkrAutoSyncIntervalHours
	}
	if req.SpnkrAutoSyncIntervalMinutes != nil {
		cfg.SpnkrAutoSyncIntervalMinutes = *req.SpnkrAutoSyncIntervalMinutes
	}
	if req.WatcherPresenceEnabled != nil {
		cfg.WatcherPresenceEnabled = *req.WatcherPresenceEnabled
	}
	if req.WatcherSubscribedPlayers != nil {
		cfg.WatcherSubscribedPlayers = req.WatcherSubscribedPlayers
	}
	if req.SpnkrRefreshWithBackfill != nil {
		cfg.SpnkrRefreshWithBackfill = *req.SpnkrRefreshWithBackfill
	}
	if req.SpnkrRefreshBackfillMedals != nil {
		cfg.SpnkrRefreshBackfillMedals = *req.SpnkrRefreshBackfillMedals
	}
	if req.SpnkrRefreshBackfillSkill != nil {
		cfg.SpnkrRefreshBackfillSkill = *req.SpnkrRefreshBackfillSkill
	}
	if req.SpnkrRefreshBackfillAliases != nil {
		cfg.SpnkrRefreshBackfillAliases = *req.SpnkrRefreshBackfillAliases
	}
	if req.SpnkrRefreshBackfillPersonalScores != nil {
		cfg.SpnkrRefreshBackfillPersonalScores = *req.SpnkrRefreshBackfillPersonalScores
	}
	if req.SpnkrRefreshBackfillPerfScores != nil {
		cfg.SpnkrRefreshBackfillPerfScores = *req.SpnkrRefreshBackfillPerfScores
	}
	if req.SpnkrRefreshBackfillLUSR != nil {
		cfg.SpnkrRefreshBackfillLUSR = *req.SpnkrRefreshBackfillLUSR
	}
	if req.SpnkrRefreshBackfillEvents != nil {
		cfg.SpnkrRefreshBackfillEvents = *req.SpnkrRefreshBackfillEvents
	}
	if req.SpnkrRefreshBackfillWeapons != nil {
		cfg.SpnkrRefreshBackfillWeapons = *req.SpnkrRefreshBackfillWeapons
	}
	if req.FriendGamertags != nil {
		cfg.FriendGamertags = req.FriendGamertags
	}
	if req.SessionGapMinutes != nil {
		cfg.SessionGapMinutes = *req.SessionGapMinutes
	}
	if req.SessionSplitOnRankedChange != nil {
		cfg.SessionSplitOnRankedChange = *req.SessionSplitOnRankedChange
	}
	if req.SessionTeamChangeMode != nil {
		cfg.SessionTeamChangeMode = *req.SessionTeamChangeMode
	}
	if req.OutcomeExcludeBotMatchesFromBadges != nil {
		cfg.OutcomeExcludeBotMatchesFromBadges = *req.OutcomeExcludeBotMatchesFromBadges
	}
	if req.OutcomeExcludeBotMatchesFromRecords != nil {
		cfg.OutcomeExcludeBotMatchesFromRecords = *req.OutcomeExcludeBotMatchesFromRecords
	}
	if req.OutcomeBadgeSensitivity != nil {
		cfg.OutcomeBadgeSensitivity = *req.OutcomeBadgeSensitivity
	}
	if req.AuthProvider != nil {
		cfg.AuthProvider = *req.AuthProvider
	}
}

// ToResponse convertit AppSettings en SettingsResponse (sans discord_webhook_url).
func ToResponse(cfg *AppSettings) *domain.SettingsResponse {
	return &domain.SettingsResponse{
		Lang:                                cfg.Lang,
		DiscordLang:                         cfg.DiscordLang,
		UserTimezone:                        cfg.UserTimezone,
		NormalizeModeLabels:                 cfg.NormalizeModeLabels,
		ShowRecords:                         cfg.ShowRecords,
		RefreshClearsCaches:                 cfg.RefreshClearsCaches,
		CareerTopExcludeBTB:                 cfg.CareerTopExcludeBTB,
		MediaCapturesBaseDir:                cfg.MediaCapturesBaseDir,
		MediaToleranceMinutes:               cfg.MediaBufferMinutes,
		MediaWatcherEnabled:                 cfg.MediaWatcherEnabled,
		MediaWatcherDebounceSeconds:         cfg.MediaWatcherDebounceSeconds,
		DiscordNotificationsEnabled:         cfg.DiscordNotificationsEnabled,
		DiscordWebhookURLPresent:            cfg.DiscordWebhookURL != "",
		DiscordNotifySync:                   cfg.DiscordNotifySync,
		DiscordNotifyBackfill:               cfg.DiscordNotifyBackfill,
		DiscordNotifyNewVersion:             cfg.DiscordNotifyNewVersion,
		DiscordNotifyNewMedia:               cfg.DiscordNotifyNewMedia,
		SpnkrAutoSyncEnabled:                cfg.SpnkrAutoSyncEnabled,
		SpnkrAutoSyncIntervalHours:          cfg.SpnkrAutoSyncIntervalHours,
		SpnkrAutoSyncIntervalMinutes:        cfg.SpnkrAutoSyncIntervalMinutes,
		WatcherPresenceEnabled:              cfg.WatcherPresenceEnabled,
		WatcherSubscribedPlayers:            cfg.WatcherSubscribedPlayers,
		SpnkrRefreshWithBackfill:            cfg.SpnkrRefreshWithBackfill,
		SpnkrRefreshBackfillMedals:          cfg.SpnkrRefreshBackfillMedals,
		SpnkrRefreshBackfillSkill:           cfg.SpnkrRefreshBackfillSkill,
		SpnkrRefreshBackfillAliases:         cfg.SpnkrRefreshBackfillAliases,
		SpnkrRefreshBackfillPersonalScores:  cfg.SpnkrRefreshBackfillPersonalScores,
		SpnkrRefreshBackfillPerfScores:      cfg.SpnkrRefreshBackfillPerfScores,
		SpnkrRefreshBackfillLUSR:            cfg.SpnkrRefreshBackfillLUSR,
		SpnkrRefreshBackfillEvents:          cfg.SpnkrRefreshBackfillEvents,
		SpnkrRefreshBackfillWeapons:         cfg.SpnkrRefreshBackfillWeapons,
		FriendGamertags:                     cfg.FriendGamertags,
		SessionGapMinutes:                   cfg.SessionGapMinutes,
		SessionSplitOnRankedChange:          cfg.SessionSplitOnRankedChange,
		SessionTeamChangeMode:               cfg.SessionTeamChangeMode,
		OutcomeExcludeBotMatchesFromBadges:  cfg.OutcomeExcludeBotMatchesFromBadges,
		OutcomeExcludeBotMatchesFromRecords: cfg.OutcomeExcludeBotMatchesFromRecords,
		OutcomeBadgeSensitivity:             cfg.OutcomeBadgeSensitivity,
		AuthProvider:                        cfg.AuthProvider,
	}
}

// Defaults retourne les valeurs par défaut de app_settings.json.
func Defaults() *AppSettings {
	return defaultSettings()
}

// defaultSettings retourne les valeurs par défaut de app_settings.json.
func defaultSettings() *AppSettings {
	return &AppSettings{
		Lang:                "en",
		DiscordLang:         "fr",
		UserTimezone:        "Europe/Paris",
		MediaBufferMinutes:  2,
		CanSelfProvision:    true,
		CanStartInitialSync: true,
		// Règles de sessions
		SessionGapMinutes:     120,       // 2 heures — historique Python
		SessionTeamChangeMode: "friends", // amis seulement — moins sensible aux randoms
		// Règles de badges narratifs
		OutcomeExcludeBotMatchesFromBadges:  true,       // bots faussent les scores adverses
		OutcomeExcludeBotMatchesFromRecords: false,      // pas de changement de comportement par défaut
		OutcomeBadgeSensitivity:             "standard", // seuils historiques Python
	}
}
