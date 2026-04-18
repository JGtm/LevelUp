// Package config lit la configuration de l'application LevelUp.
// Sources : db_profiles.json, app_settings.json, variables d'environnement.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
)

// AppConfig centralise la configuration de l'application.
type AppConfig struct {
	RepoRoot        string
	DBProfilesPath  string
	AppSettingsPath string
	SessionDir      string
	DemoMode        bool
	DemoFixturesDir string
	APIHost         string
	APIPort         int
	SessionSecret   string
	CORSOrigins     []string
	Lang            string
	AppVersion      string
	FeatureFlags    FeatureFlags
	// Sprint 35 : shadow mode — compare Go vs Python en parallèle.
	// LEVELUP_SHADOW_MODE=both active le mode shadow (appel parallèle Python).
	ShadowMode string
	// PythonURL : URL de base de l'API Python pour le shadow mode.
	// Défaut : http://127.0.0.1:8001
	PythonURL string
	// Sprint 40 T2 : Discord webhook URL pour alerting 500 + taux d'erreur.
	// Lit LEVELUP_DISCORD_WEBHOOK_URL ; fallback sur discord_webhook_url dans app_settings.json.
	DiscordWebhookURL string
}

// loadDiscordWebhookURL lit le webhook Discord depuis LEVELUP_DISCORD_WEBHOOK_URL
// ou le champ discord_webhook_url de app_settings.json.
func loadDiscordWebhookURL(settingsPath string) string {
	if url := os.Getenv("LEVELUP_DISCORD_WEBHOOK_URL"); url != "" {
		return url
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if s, ok := m["discord_webhook_url"].(string); ok {
		if strings.HasPrefix(s, "https://discord.com/api/webhooks/") {
			return s
		}
	}
	return ""
}

// Load charge la configuration depuis les variables d'environnement.
// Les valeurs par défaut correspondent au développement local.
func Load() (*AppConfig, error) {
	repoRoot := getEnvOrDefault("LEVELUP_REPO_ROOT", autoDetectRepoRoot())
	demoMode := strings.ToLower(getEnvOrDefault("LEVELUP_DEMO_MODE", "false")) == "true"

	cfg := &AppConfig{
		RepoRoot:          repoRoot,
		DBProfilesPath:    getEnvOrDefault("LEVELUP_DB_PROFILES", filepath.Join(repoRoot, "db_profiles.json")),
		AppSettingsPath:   getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json")),
		SessionDir:        getEnvOrDefault("LEVELUP_SESSION_DIR", filepath.Join(repoRoot, "data", "sessions")),
		DemoMode:          demoMode,
		DemoFixturesDir:   getEnvOrDefault("LEVELUP_DEMO_FIXTURES_DIR", filepath.Join(repoRoot, "tests", "fixtures", "ref_player")),
		APIHost:           getEnvOrDefault("LEVELUP_API_HOST", "127.0.0.1"),
		APIPort:           getEnvInt("LEVELUP_API_PORT", 8000),
		SessionSecret:     getEnvOrDefault("LEVELUP_SESSION_SECRET", "CHANGE_ME_IN_PRODUCTION"),
		CORSOrigins:       parseCORSOrigins(getEnvOrDefault("LEVELUP_CORS_ORIGINS", "")),
		Lang:              getEnvOrDefault("LEVELUP_LANG", "fr"),
		AppVersion:        getEnvOrDefault("LEVELUP_APP_VERSION", "dev"),
		ShadowMode:        getEnvOrDefault("LEVELUP_SHADOW_MODE", ""),
		PythonURL:         getEnvOrDefault("LEVELUP_PYTHON_URL", "http://127.0.0.1:8001"),
		DiscordWebhookURL: loadDiscordWebhookURL(getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json"))),
	}
	appSettingsPath := getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json"))
	cfg.FeatureFlags = LoadFeatureFlags(appSettingsPath)
	return cfg, nil
}

// dbProfilesFile représente le format du fichier db_profiles.json (v2.1).
// Structure : { "version": "2.1", "profiles": { "<gamertag>": {...} } }
type dbProfilesFile struct {
	Version  string                    `json:"version"`
	Profiles map[string]dbProfileEntry `json:"profiles"`
}

// dbProfilesFileV3 représente le format v3 title-aware de db_profiles.json.
// Structure : { "version": "3.0", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }
type dbProfilesFileV3 struct {
	Version  string                               `json:"version"`
	Profiles map[string]map[string]dbProfileEntry `json:"profiles"`
}

// dbProfileEntry représente une entrée dans la map "profiles" de db_profiles.json.
type dbProfileEntry struct {
	DBPath         string `json:"db_path"`
	XUID           string `json:"xuid"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
}

// LoadPlayers charge db_profiles.json et retourne la liste des joueurs.
// Supporte les formats v2.1 (flat) et v3.0 (title-scoped).
// Si titleFilter est non vide, ne retourne que les joueurs de ce titre.
func (c *AppConfig) LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error) {
	if c.DemoMode {
		return []domain.PlayerSummary{{
			PlayerSlug:     "demo-player",
			Gamertag:       "DemoPlayer",
			XUID:           "0",
			WaypointPlayer: "DemoPlayer",
			IsDemo:         true,
		}}, nil
	}

	data, err := os.ReadFile(c.DBProfilesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.PlayerSummary{}, nil
		}
		return nil, fmt.Errorf("lecture db_profiles.json : %w", err)
	}

	// Détecter la version pour choisir le parser.
	var versionProbe struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &versionProbe); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json : %w", err)
	}

	if versionProbe.Version == "3.0" {
		return c.loadPlayersV3(data, titleFilter...)
	}
	return c.loadPlayersV2(data, titleFilter...)
}

// loadPlayersV2 parse le format v2.1 (flat map gamertag → entry).
func (c *AppConfig) loadPlayersV2(data []byte, titleFilter ...string) ([]domain.PlayerSummary, error) {
	var file dbProfilesFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json v2 : %w", err)
	}

	// En v2.1, tous les profils sont implicitement halo_infinite.
	filter := ""
	if len(titleFilter) > 0 {
		filter = titleFilter[0]
	}
	if filter != "" && filter != "halo_infinite" {
		return []domain.PlayerSummary{}, nil
	}

	players := make([]domain.PlayerSummary, 0, len(file.Profiles))
	for gamertag, p := range file.Profiles {
		wp := p.WaypointPlayer
		if wp == "" {
			wp = gamertag
		}
		players = append(players, domain.PlayerSummary{
			PlayerSlug:     gamertag,
			Gamertag:       gamertag,
			XUID:           p.XUID,
			WaypointPlayer: wp,
			IsDemo:         false,
		})
	}
	return players, nil
}

// loadPlayersV3 parse le format v3.0 (title_slug → gamertag → entry).
func (c *AppConfig) loadPlayersV3(data []byte, titleFilter ...string) ([]domain.PlayerSummary, error) {
	var file dbProfilesFileV3
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json v3 : %w", err)
	}

	filter := ""
	if len(titleFilter) > 0 {
		filter = titleFilter[0]
	}

	var players []domain.PlayerSummary
	for titleSlug, titleProfiles := range file.Profiles {
		if filter != "" && titleSlug != filter {
			continue
		}
		for gamertag, p := range titleProfiles {
			wp := p.WaypointPlayer
			if wp == "" {
				wp = gamertag
			}
			players = append(players, domain.PlayerSummary{
				PlayerSlug:     gamertag,
				Gamertag:       gamertag,
				XUID:           p.XUID,
				WaypointPlayer: wp,
				IsDemo:         false,
			})
		}
	}
	return players, nil
}

// LoadAppSettings charge app_settings.json. Retourne une map vide si absent.
func (c *AppConfig) LoadAppSettings() (map[string]interface{}, error) {
	data, err := os.ReadFile(c.AppSettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("lecture app_settings.json : %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing app_settings.json : %w", err)
	}
	return settings, nil
}

// ServerAddr retourne l'adresse d'écoute au format "host:port".
func (c *AppConfig) ServerAddr() string {
	return fmt.Sprintf("%s:%d", c.APIHost, c.APIPort)
}

// --- helpers ---

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// autoDetectRepoRoot remonte depuis le binaire pour trouver la racine du repo.
// Cherche la présence de db_profiles.json ou db_profiles.example.json.
func autoDetectRepoRoot() string {
	// Dans le worktree go-migration, le binaire sera dans apps/go-api/cmd/server/
	// La racine est 4 niveaux au-dessus.
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	// Remonte jusqu'à trouver db_profiles.example.json (marqueur de la racine repo)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "db_profiles.example.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
