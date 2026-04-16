// Package service orchestre la logique métier en combinant config + repositories.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// BootstrapService construit le BootstrapResponse pour l'endpoint /api/v1/bootstrap.
type BootstrapService struct {
	cfg      *config.AppConfig
	bootRepo port.BootstrapRepository
}

// NewBootstrapService crée un BootstrapService.
func NewBootstrapService(cfg *config.AppConfig, bootRepo port.BootstrapRepository) *BootstrapService {
	return &BootstrapService{cfg: cfg, bootRepo: bootRepo}
}

// Build construit la réponse bootstrap complète.
// sess peut être nil si la session n'est pas encore initialisée.
func (s *BootstrapService) Build(ctx context.Context, sess *domain.SessionData) (*domain.BootstrapResponse, error) {
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "bootstrap: erreur chargement joueurs", "err", err)
		players = []domain.PlayerSummary{}
	}

	appSettings, err := s.cfg.LoadAppSettings()
	if err != nil {
		slog.WarnContext(ctx, "bootstrap: erreur chargement app_settings", "err", err)
		appSettings = map[string]interface{}{}
	}

	setupRequired := !s.cfg.DemoMode && len(players) == 0
	capabilities := buildCapabilities(s.cfg, appSettings)
	settingsExcerpt := buildSettingsExcerpt(s.cfg, appSettings)
	flags := buildFeatureFlags(s.cfg, appSettings)

	setupState := resolveSetupState(players)

	var currentPlayer *domain.PlayerSummary
	if len(players) > 0 {
		p := players[0]
		currentPlayer = &p
	}

	// Sprint 44 : titre courant depuis la session, fallback halo_infinite.
	currentTitleSlug := titlePkg.DefaultSlug
	if sess != nil && sess.CurrentTitleSlug != "" {
		currentTitleSlug = sess.CurrentTitleSlug
	}

	return &domain.BootstrapResponse{
		SetupRequired:       setupRequired,
		AuthState:           ResolveAuthState(sess),
		SetupState:          setupState,
		LinkedHaloIdentity:  ResolveLinkedIdentity(sess),
		CurrentPlayer:       currentPlayer,
		AvailablePlayers:    players,
		CurrentTitleSlug:    currentTitleSlug,
		AvailableTitles:     buildAvailableTitles(),
		Locale:              settingsExcerpt.Lang,
		HintsVisibleDefault: true,
		FeatureFlags:        flags,
		Capabilities:        capabilities,
		SettingsExcerpt:     settingsExcerpt,
	}, nil
}

// BuildPlayersList construit la liste des joueurs pour GET /api/v1/players.
func (s *BootstrapService) BuildPlayersList(_ context.Context) (*domain.PlayersListResponse, error) {
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("BuildPlayersList: %w", err)
	}
	var defaultSlug *string
	if len(players) > 0 {
		slug := players[0].PlayerSlug
		defaultSlug = &slug
	}
	return &domain.PlayersListResponse{
		Items:             players,
		DefaultPlayerSlug: defaultSlug,
	}, nil
}

// --- helpers ---

func buildCapabilities(cfg *config.AppConfig, settings map[string]interface{}) domain.CapabilityMap {
	mediaEnabled := getBoolSetting(settings, "media_enabled", true)
	return domain.CapabilityMap{
		CanReadLocalData:    true,
		CanRunSync:          !cfg.DemoMode,
		CanUseLiveHalo:      !cfg.DemoMode,
		CanManageSettings:   true,
		CanResetMediaIndex:  true,
		CanViewMedia:        mediaEnabled,
		CanSelfProvision:    getBoolSetting(settings, "can_self_provision", true),
		CanStartInitialSync: getBoolSetting(settings, "can_start_initial_sync", !cfg.DemoMode),
		CanManageInstance:   true,
	}
}

func buildSettingsExcerpt(cfg *config.AppConfig, settings map[string]interface{}) domain.SettingsExcerpt {
	lang := getStringSetting(settings, "lang", cfg.Lang)
	return domain.SettingsExcerpt{
		Lang:                lang,
		UserTimezone:        getStringSetting(settings, "user_timezone", "Europe/Paris"),
		ShowRecords:         getBoolSetting(settings, "show_records", true),
		NormalizeModeLabels: getBoolSetting(settings, "normalize_mode_labels", true),
	}
}

func buildFeatureFlags(cfg *config.AppConfig, settings map[string]interface{}) domain.FeatureFlags {
	return domain.FeatureFlags{
		V7Enabled:         true,
		MediaEnabled:      getBoolSetting(settings, "media_enabled", true),
		DemoMode:          cfg.DemoMode,
		DiscordConfigured: getStringSetting(settings, "discord_webhook_url", "") != "",
		TailscaleEnabled:  getBoolSetting(settings, "tailscale_enabled", false),
	}
}

func resolveSetupState(players []domain.PlayerSummary) string {
	if len(players) == 0 {
		return "no_halo_link"
	}
	// Sprint 31 : vérifier sync_meta.initial_sync_completed_at à terme.
	return "profile_ready_no_sync"
}

// ResolveAuthState déduit l'état d'authentification depuis la session.
func ResolveAuthState(sess *domain.SessionData) string {
	if sess == nil || !sess.AuthReady {
		return "missing"
	}
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.Gamertag == "" {
		return "partial"
	}
	return "ready"
}

// ResolveLinkedIdentity extrait l'identité Halo liée si présente dans la session.
func ResolveLinkedIdentity(sess *domain.SessionData) *domain.HaloIdentitySummary {
	if sess == nil || sess.LinkedHaloIdentity == nil {
		return nil
	}
	return &domain.HaloIdentitySummary{
		Gamertag: sess.LinkedHaloIdentity.Gamertag,
		XUID:     sess.LinkedHaloIdentity.XUID,
	}
}

func getBoolSetting(settings map[string]interface{}, key string, def bool) bool {
	if v, ok := settings[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func getStringSetting(settings map[string]interface{}, key, def string) string {
	if v, ok := settings[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

// buildAvailableTitles construit la liste des titres depuis le registre.
func buildAvailableTitles() []domain.TitleSummary {
	reg := titlePkg.NewRegistry()
	all := reg.All()
	out := make([]domain.TitleSummary, 0, len(all))
	for _, t := range all {
		caps := make([]string, len(t.Capabilities))
		for i, c := range t.Capabilities {
			caps[i] = string(c)
		}
		out = append(out, domain.TitleSummary{
			Slug:         t.Slug,
			Name:         t.Name,
			IconURL:      t.IconURL,
			Status:       string(t.Status),
			Capabilities: caps,
			IsDefault:    t.IsDefault,
		})
	}
	return out
}
