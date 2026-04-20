// Package domain contient les types métier purs de LevelUp (0 import externe, 0 IO).
package domain

// FeatureFlags représente les flags de fonctionnalités actifs sur ce déploiement.
type FeatureFlags struct {
	V7Enabled         bool `json:"v7_enabled"`
	MediaEnabled      bool `json:"media_enabled"`
	DemoMode          bool `json:"demo_mode"`
	DiscordConfigured bool `json:"discord_configured"`
	TailscaleEnabled  bool `json:"tailscale_enabled"`
}

// SettingsExcerpt représente l'extrait des préférences utilisateur nécessaires au shell.
type SettingsExcerpt struct {
	Lang                string `json:"lang"`
	UserTimezone        string `json:"user_timezone"`
	ShowRecords         bool   `json:"show_records"`
	NormalizeModeLabels bool   `json:"normalize_mode_labels"`
}

// CapabilityMap représente les capacités actives selon la configuration serveur.
type CapabilityMap struct {
	CanReadLocalData    bool `json:"can_read_local_data"`
	CanRunSync          bool `json:"can_run_sync"`
	CanUseLiveHalo      bool `json:"can_use_live_halo"`
	CanManageSettings   bool `json:"can_manage_settings"`
	CanResetMediaIndex  bool `json:"can_reset_media_index"`
	CanViewMedia        bool `json:"can_view_media"`
	CanSelfProvision    bool `json:"can_self_provision"`
	CanStartInitialSync bool `json:"can_start_initial_sync"`
	CanManageInstance   bool `json:"can_manage_instance"`
}

// PlayerSummary représente le résumé d'un profil joueur.
type PlayerSummary struct {
	PlayerSlug     string `json:"player_slug"`
	Gamertag       string `json:"gamertag"`
	XUID           string `json:"xuid"`
	WaypointPlayer string `json:"waypoint_player"`
	IsDemo         bool   `json:"is_demo"`
}

// HaloIdentitySummary représente l'identité Halo résolue côté backend.
type HaloIdentitySummary struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
}

// TitleSummary résume un titre supporté pour le frontend.
type TitleSummary struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	IconURL      string   `json:"icon_url,omitempty"`
	Status       string   `json:"status"` // "active" | "coming_soon" | "archived"
	Capabilities []string `json:"capabilities"`
	IsDefault    bool     `json:"is_default"`
}

// BootstrapResponse est la réponse de GET /api/v1/bootstrap.
// Contient tout ce dont le shell React a besoin pour s'initialiser.
type BootstrapResponse struct {
	SetupRequired       bool                 `json:"setup_required"`
	AuthState           string               `json:"auth_state"`  // "missing" | "partial" | "ready"
	SetupState          string               `json:"setup_state"` // "no_halo_link" | "halo_linked_no_profile" | "profile_ready_no_sync" | "ready"
	CurrentPlayer       *PlayerSummary       `json:"current_player"`
	AvailablePlayers    []PlayerSummary      `json:"available_players"`
	CurrentTitleSlug    string               `json:"current_title_slug"` // Sprint 44
	AvailableTitles     []TitleSummary       `json:"available_titles"`   // Sprint 44
	Locale              string               `json:"locale"`
	HintsVisibleDefault bool                 `json:"hints_visible_default"`
	FeatureFlags        FeatureFlags         `json:"feature_flags"`
	Capabilities        CapabilityMap        `json:"capabilities"`
	SettingsExcerpt     SettingsExcerpt      `json:"settings_excerpt"`
	LinkedHaloIdentity  *HaloIdentitySummary `json:"linked_halo_identity,omitempty"`
	ActiveSyncJobID     *string              `json:"active_sync_job_id,omitempty"`
	// Sprint 54 B : privacy du compte courant (chargée en parallèle).
	Privacy *MatchPrivacyInfo `json:"privacy,omitempty"`
	// Auth locale
	AuthMode         string  `json:"auth_mode"`
	RegistrationMode string  `json:"registration_mode"`
	IsAdmin          bool    `json:"is_admin"`
	CurrentUsername  *string `json:"current_username"`
	FirstLaunch      bool    `json:"first_launch"`
}

// PlayersListResponse est la réponse de GET /api/v1/players.
type PlayersListResponse struct {
	Items             []PlayerSummary `json:"items"`
	DefaultPlayerSlug *string         `json:"default_player_slug"`
}
