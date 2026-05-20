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
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// defaultUserTimezone est le timezone IANA utilisé en l'absence de configuration.
const defaultUserTimezone = "Europe/Paris"

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
	// SharedProvider (commit 8g, retypé 8i) — injecté au boot par main.go en
	// mode B-swap (LEVELUP_USE_SHARED_PROVIDER=1). Passé aux PlayerPoolConfig
	// (satisfait duckdb.SharedReader structurellement) et au SyncEngine
	// via WithSharedProvider. nil en mode legacy.
	SharedProvider sharedprovider.Provider
	// Sprint 40 T2 : Discord webhook URL pour alerting 500 + taux d'erreur.
	// Lit LEVELUP_DISCORD_WEBHOOK_URL ; fallback sur discord_webhook_url dans app_settings.json.
	DiscordWebhookURL string
	// UserTimezone : timezone IANA lue depuis user_timezone dans app_settings.json.
	// Utilisée pour configurer les sessions DuckDB (SET TimeZone).
	UserTimezone string
	// Auth locale : répertoire contenant users.json et invites.json.
	AuthDir string
	// AuthMode : "none" (défaut), "password" ou "xbox" (SSO).
	AuthMode string
	// OAuthRedirectURI : URI publique du callback OAuth Authorization Code Flow (PR 4).
	// Doit être strictement identique à celui configuré côté Azure portail.
	// Lit LEVELUP_OAUTH_REDIRECT_URI. Si vide, /auth/xbox/login retourne 500.
	OAuthRedirectURI string
	// RegistrationMode : "invite" (défaut), "open" ou "closed".
	RegistrationMode string
	// CurrentCSRSeasonID est l'identifiant de saison CSR courant (ex: "CsrSeason8").
	// Lit LEVELUP_CSR_SEASON_ID ou le champ csr_season_id dans app_settings.json.
	// Vide → le sync CSR est skippé silencieusement.
	CurrentCSRSeasonID string
	// MediaCapturesBaseDir : dossier externe configuré par l'utilisateur où sont
	// stockées les captures (snapshot du champ media_captures_base_dir de
	// app_settings.json au démarrage). Vide → fallback sur le chemin interne
	// PlayerCapturesDir. Utilisé par la CLI ; les handlers HTTP relisent le
	// settings store directement pour rester réactifs aux PATCH /settings.
	MediaCapturesBaseDir string
}

// loadDiscordWebhookURL lit le webhook Discord depuis LEVELUP_DISCORD_WEBHOOK_URL,
// DISCORD_WEBHOOK_URL (legacy Python) ou le champ discord_webhook_url de app_settings.json.
func loadDiscordWebhookURL(settingsPath string) string {
	if url := os.Getenv("LEVELUP_DISCORD_WEBHOOK_URL"); url != "" {
		return url
	}
	if url := os.Getenv("DISCORD_WEBHOOK_URL"); url != "" {
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
	// Charger .env.local avant toute lecture de variable d'environnement,
	// pour que SPNKR_OAUTH_REFRESH_TOKEN_* et autres vars locales soient disponibles.
	loadEnvLocal(filepath.Join(repoRoot, ".env.local"))
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
		DiscordWebhookURL: loadDiscordWebhookURL(getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json"))),
		AuthDir:           getEnvOrDefault("LEVELUP_AUTH_DIR", filepath.Join(repoRoot, "data", "auth")),
		AuthMode:          getEnvOrDefault("LEVELUP_AUTH_MODE", "none"),
		OAuthRedirectURI:  getEnvOrDefault("LEVELUP_OAUTH_REDIRECT_URI", ""),
		RegistrationMode:  getEnvOrDefault("LEVELUP_REGISTRATION", "invite"),
	}
	appSettingsPath := getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json"))
	cfg.FeatureFlags = LoadFeatureFlags(appSettingsPath)
	cfg.UserTimezone = loadUserTimezone(appSettingsPath)
	cfg.CurrentCSRSeasonID = loadCSRSeasonID(appSettingsPath)
	cfg.MediaCapturesBaseDir = loadMediaCapturesBaseDir(appSettingsPath)
	return cfg, nil
}

// loadMediaCapturesBaseDir lit media_captures_base_dir depuis app_settings.json.
// Retourne "" si le fichier est absent ou le champ manquant — dans ce cas le
// fallback PlayerCapturesDir s'applique (cf. PathResolver.ResolveCapturesDir).
func loadMediaCapturesBaseDir(settingsPath string) string {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if s, ok := m["media_captures_base_dir"].(string); ok {
		return s
	}
	return ""
}

// loadCSRSeasonID lit l'identifiant de saison CSR depuis LEVELUP_CSR_SEASON_ID
// ou le champ csr_season_id dans app_settings.json.
func loadCSRSeasonID(settingsPath string) string {
	if id := os.Getenv("LEVELUP_CSR_SEASON_ID"); id != "" {
		return id
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if s, ok := m["csr_season_id"].(string); ok {
		return s
	}
	return ""
}

// loadUserTimezone lit user_timezone depuis app_settings.json.
// Retourne "Europe/Paris" par défaut si le fichier est absent ou le champ manquant.
func loadUserTimezone(settingsPath string) string {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return defaultUserTimezone
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return defaultUserTimezone
	}
	if s, ok := m["user_timezone"].(string); ok && s != "" {
		return s
	}
	return defaultUserTimezone
}

// dbProfilesFile représente le format du fichier db_profiles.json (v2.1).
// Structure : { "version": "2.1", "profiles": { "<gamertag>": {...} } }
type dbProfilesFile struct {
	Version  string                    `json:"version"`
	Profiles map[string]dbProfileEntry `json:"profiles"`
}

// dbProfilesFileV3 représente le format v3 title-aware de db_profiles.json.
// Structure : { "version": "3.0", "admin": "<gamertag>", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }
type dbProfilesFileV3 struct {
	Version  string                               `json:"version"`
	Admin    string                               `json:"admin"`
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
		titleSlug := title.DefaultSlug
		if len(titleFilter) > 0 && titleFilter[0] != "" {
			titleSlug = titleFilter[0]
		}
		return []domain.PlayerSummary{{
			PlayerSlug:     "demo-player",
			Gamertag:       "DemoPlayer",
			XUID:           "0",
			WaypointPlayer: "DemoPlayer",
			IsDemo:         true,
			TitleSlug:      titleSlug,
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

// AdminPlayer retourne le gamertag désigné comme admin dans db_profiles.json (champ "admin").
// Retourne "" si le fichier est absent, illisible ou si le champ n'est pas défini (format v2).
func (c *AppConfig) AdminPlayer() string {
	data, err := os.ReadFile(c.DBProfilesPath)
	if err != nil {
		return ""
	}
	var f dbProfilesFileV3
	if err := json.Unmarshal(data, &f); err != nil {
		return ""
	}
	return f.Admin
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
	if filter != "" && filter != title.DefaultSlug {
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
			TitleSlug:      title.DefaultSlug,
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
				TitleSlug:      titleSlug,
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

// loadEnvLocal lit un fichier .env.local et injecte les variables dans l'environnement
// du processus, sans écraser les variables déjà définies (env process a priorité).
// Format : KEY=VALUE, lignes # ignorées, lignes vides ignorées.
// Si le fichier est absent, la fonction retourne silencieusement.
func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // .env.local absent — ok en CI/prod
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Supprimer les guillemets simples ou doubles éventuels.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		// Ne pas écraser une variable déjà définie dans l'environnement réel.
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

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
		return []string{
			"http://localhost:5173", "http://127.0.0.1:5173",
			"http://localhost:5174", "http://127.0.0.1:5174",
		}
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

// autoDetectRepoRoot remonte depuis le binaire (ou le cwd en fallback) pour
// trouver la racine du repo. Cherche la présence de db_profiles.example.json.
func autoDetectRepoRoot() string {
	// Tente d'abord depuis l'exécutable (binaire compilé), puis depuis le cwd
	// (nécessaire avec `go run` qui place l'exe dans un répertoire temp).
	candidates := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	for _, start := range candidates {
		dir := start
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
	}
	return "."
}
