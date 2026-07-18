// Package settings fournit la lecture/écriture de app_settings.json.
// Sprint 16 : GET /settings, PATCH /settings.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// AppSettings représente la structure complète de app_settings.json.
// Seuls les champs exposés par l'API sont typés — les champs inconnus sont
// préservés dans Extra pour ne pas les effacer lors d'un PATCH.
type AppSettings struct {
	Lang                 string `json:"lang"`
	DiscordLang          string `json:"discord_lang"`
	UserTimezone         string `json:"user_timezone"`
	NormalizeModeLabels  bool   `json:"normalize_mode_labels"`
	ShowRecords          bool   `json:"show_records"`
	RefreshClearsCaches  bool   `json:"refresh_clears_caches"`
	CareerTopExcludeBTB  bool   `json:"career_top_exclude_btb"`
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
	// MediaDeleteSourceAfterTranscode pilote la suppression du fichier source
	// (.mkv/.avi…) après un transcodage HLS réussi. *bool : nil = « auto » (défaut
	// résolu par environnement — supprime en prod, conserve en local). La résolution
	// effective se fait au déclenchement via config.ResolveMediaDeleteSource
	// (env > store > isProd), jamais figée ici.
	MediaDeleteSourceAfterTranscode    *bool    `json:"media_delete_source_after_transcode,omitempty"`
	MediaBufferMinutes                 int      `json:"media_buffer_minutes"`
	MediaWatcherEnabled                bool     `json:"media_watcher_enabled"`
	MediaWatcherDebounceSeconds        int      `json:"media_watcher_debounce_seconds"`
	DiscordNotificationsEnabled        bool     `json:"discord_notifications_enabled"`
	DiscordWebhookURL                  string   `json:"discord_webhook_url"` // jamais exposé côté API
	DiscordNotifySync                  bool     `json:"discord_notify_sync"`
	DiscordNotifyBackfill              bool     `json:"discord_notify_backfill"`
	DiscordNotifyNewVersion            bool     `json:"discord_notify_new_version"`
	DiscordNotifyFriends               bool     `json:"discord_notify_friends"` // §6.B Squad/Sessions overhaul
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

	// RendementExcludeAssists — rendement combat sans assistances (défaut false).
	RendementExcludeAssists bool `json:"rendement_exclude_assists"`

	// ShowProgression — affichage du système Objectifs/Prestige (défaut : true).
	ShowProgression bool `json:"show_progression"`

	// CoachProactiveMode — toggle pont coach → Prestige (cf. ADR 0020).
	// Opt-in explicite : défaut false (zero value).
	CoachProactiveMode bool `json:"coach_proactive_mode"`

	// Capabilities (défaut : true)
	CanSelfProvision    bool `json:"can_self_provision"`
	CanStartInitialSync bool `json:"can_start_initial_sync"`

	// InstanceLocked : verrou « instance fermée » activable à chaud (défaut false).
	// Bloque la création de nouvelles identités/BDD (register, SSO xuid inconnu,
	// setup/players). Cumulé en OU avec LEVELUP_INSTANCE_LOCKED (env, verrou forcé).
	InstanceLocked bool `json:"instance_locked"`

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
	applyAbsentDefaults(cfg, raw)
	return cfg, nil
}

// applyAbsentDefaults réapplique les défauts « clé absente → true » sur un
// AppSettings dérivé de la map raw donnée (rétrocompatibilité fichiers existants).
// Partagé entre Load et ResolveForTitle pour garder une seule source de vérité.
func applyAbsentDefaults(cfg *AppSettings, raw map[string]json.RawMessage) {
	if _, ok := raw["can_self_provision"]; !ok {
		cfg.CanSelfProvision = true
	}
	if _, ok := raw["can_start_initial_sync"]; !ok {
		cfg.CanStartInitialSync = true
	}
	if _, ok := raw["show_progression"]; !ok {
		cfg.ShowProgression = true
	}
}

// ResolveForTitle charge les settings GLOBAUX puis applique l'overlay du titre
// (champ-présent-only) lu depuis overlayPath (PMT-4). Modèle « base globale +
// overlay par titre » : seules les clés présentes dans l'overlay surchargent le
// global ; les autres héritent. overlayPath vide ou fichier absent/vide ⇒ renvoie
// le global INCHANGÉ (cas Halo sans overlay déclaré → byte-identique à Load).
//
// Le caller fournit le chemin (via PathResolver.TitleSettingsPath(slug)) : le
// package settings ne dépend ni du registre de titres ni d'un slug littéral.
func (s *Store) ResolveForTitle(overlayPath string) (*AppSettings, error) {
	base, err := s.Load()
	if err != nil {
		return nil, err
	}
	if overlayPath == "" {
		return base, nil
	}
	overlayData, err := os.ReadFile(overlayPath)
	if err != nil {
		if os.IsNotExist(err) {
			return base, nil // pas d'overlay → global inchangé
		}
		return nil, fmt.Errorf("settings.ResolveForTitle read overlay: %w", err)
	}
	var overlayRaw map[string]json.RawMessage
	if err := json.Unmarshal(overlayData, &overlayRaw); err != nil {
		return nil, fmt.Errorf("settings.ResolveForTitle overlay json: %w", err)
	}
	if len(overlayRaw) == 0 {
		return base, nil
	}

	// Fusion au niveau raw : global ∪ overlay (overlay gagne), puis re-parse typé.
	merged := make(map[string]json.RawMessage, len(base.raw)+len(overlayRaw))
	for k, v := range base.raw {
		merged[k] = v
	}
	for k, v := range overlayRaw {
		merged[k] = v
	}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("settings.ResolveForTitle marshal merged: %w", err)
	}
	out := defaultSettings()
	if err := json.Unmarshal(mergedJSON, out); err != nil {
		return nil, fmt.Errorf("settings.ResolveForTitle unmarshal merged: %w", err)
	}
	out.raw = merged
	applyAbsentDefaults(out, merged)
	return out, nil
}

// SaveTitleOverlay fusionne `fields` (sparse, champ-présent-only — clés = noms JSON
// d'AppSettings, ex. "show_progression") dans l'overlay du titre à overlayPath et
// l'écrit (crée le dossier si besoin). N'affecte JAMAIS le global app_settings.json
// : c'est le pendant écriture de ResolveForTitle (PMT-4 PR-3c). Un overlay vide
// reste vide ; seules les clés fournies sont surchargées.
func (s *Store) SaveTitleOverlay(overlayPath string, fields map[string]json.RawMessage) error {
	if overlayPath == "" {
		return fmt.Errorf("settings.SaveTitleOverlay: overlayPath vide")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := map[string]json.RawMessage{}
	if data, err := os.ReadFile(overlayPath); err == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("settings.SaveTitleOverlay read overlay: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("settings.SaveTitleOverlay stat overlay: %w", err)
	}
	for k, v := range fields {
		existing[k] = v
	}
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("settings.SaveTitleOverlay marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o755); err != nil {
		return fmt.Errorf("settings.SaveTitleOverlay mkdir: %w", err)
	}
	if err := os.WriteFile(overlayPath, out, 0o644); err != nil {
		return fmt.Errorf("settings.SaveTitleOverlay write: %w", err)
	}
	return nil
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
//
//nolint:funlen // série de N if-else parallèle pour chaque champ optionnel — découpage prématuré
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
	if req.MediaDeleteSourceAfterTranscode != nil {
		// Copie du pointeur : champ optionnel *bool (nil = auto conservé si non touché).
		cfg.MediaDeleteSourceAfterTranscode = req.MediaDeleteSourceAfterTranscode
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
	if req.DiscordNotifyFriends != nil {
		cfg.DiscordNotifyFriends = *req.DiscordNotifyFriends
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
	if req.RendementExcludeAssists != nil {
		cfg.RendementExcludeAssists = *req.RendementExcludeAssists
	}
	if req.ShowProgression != nil {
		cfg.ShowProgression = *req.ShowProgression
	}
	if req.CoachProactiveMode != nil {
		cfg.CoachProactiveMode = *req.CoachProactiveMode
	}
	if req.AuthProvider != nil {
		cfg.AuthProvider = *req.AuthProvider
	}
	if req.InstanceLocked != nil {
		cfg.InstanceLocked = *req.InstanceLocked
	}
}

// ToResponse convertit AppSettings en SettingsResponse (sans discord_webhook_url).
func ToResponse(cfg *AppSettings) *domain.SettingsResponse {
	return &domain.SettingsResponse{
		Lang:                 cfg.Lang,
		DiscordLang:          cfg.DiscordLang,
		UserTimezone:         cfg.UserTimezone,
		NormalizeModeLabels:  cfg.NormalizeModeLabels,
		ShowRecords:          cfg.ShowRecords,
		RefreshClearsCaches:  cfg.RefreshClearsCaches,
		CareerTopExcludeBTB:  cfg.CareerTopExcludeBTB,
		MediaCapturesBaseDir: cfg.MediaCapturesBaseDir,
		// Placeholder : nil (*bool auto) → false ici ; le handler GET /settings écrase
		// ce champ par la valeur RÉSOLUE (config.ResolveMediaDeleteSource). Le *bool
		// brut du store n'est JAMAIS exposé tel quel dans la réponse (cf. handleGetSettings).
		MediaDeleteSourceAfterTranscode: cfg.MediaDeleteSourceAfterTranscode != nil && *cfg.MediaDeleteSourceAfterTranscode,
		MediaToleranceMinutes:           cfg.MediaBufferMinutes,
		MediaWatcherEnabled:             cfg.MediaWatcherEnabled,
		MediaWatcherDebounceSeconds:     cfg.MediaWatcherDebounceSeconds,
		DiscordNotificationsEnabled:     cfg.DiscordNotificationsEnabled,
		// Source UNIQUE de la présence webhook : reflète la réalité de l'envoi, qui
		// résout env > store (config.DiscordWebhookURLFromEnv puis discord_webhook_url,
		// cf. notify.notifyConfigFromMap). Un webhook fourni par la seule variable
		// d'env (LEVELUP_DISCORD_WEBHOOK_URL / DISCORD_WEBHOOK_URL) est donc « présent »
		// et l'UI n'affiche plus « aucun webhook » à tort. Ne PAS ré-évaluer ailleurs.
		DiscordWebhookURLPresent:            cfg.DiscordWebhookURL != "" || config.DiscordWebhookURLFromEnv() != "",
		DiscordNotifySync:                   cfg.DiscordNotifySync,
		DiscordNotifyBackfill:               cfg.DiscordNotifyBackfill,
		DiscordNotifyNewVersion:             cfg.DiscordNotifyNewVersion,
		DiscordNotifyFriends:                cfg.DiscordNotifyFriends,
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
		RendementExcludeAssists:             cfg.RendementExcludeAssists,
		ShowProgression:                     cfg.ShowProgression,
		CoachProactiveMode:                  cfg.CoachProactiveMode,
		AuthProvider:                        cfg.AuthProvider,
		InstanceLocked:                      cfg.InstanceLocked,
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
		// Affichage Objectifs/Prestige activé par défaut
		ShowProgression: true,
	}
}
