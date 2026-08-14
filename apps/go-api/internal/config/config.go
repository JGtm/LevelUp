// Package config lit la configuration de l'application LevelUp.
// Sources : db_profiles.json, app_settings.json, variables d'environnement.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/prestige"
)

// defaultUserTimezone est le timezone IANA utilisé en l'absence de configuration.
const defaultUserTimezone = "Europe/Paris"

// DefaultSessionSecret est la valeur placeholder du secret de signature des
// cookies de session. C'est une valeur PUBLIQUE (committée) : un déploiement qui
// la conserve a une clé HMAC connue de tous. Le garde-fou Validate() refuse de
// démarrer en production tant que LEVELUP_SESSION_SECRET n'est pas surchargé.
const DefaultSessionSecret = "CHANGE_ME_IN_PRODUCTION" // pragma: allowlist secret

// minSessionSecretLen est la longueur minimale (en octets) exigée pour le secret
// de session en production. 32 octets = 256 bits, aligné sur la sortie HMAC-SHA256.
const minSessionSecretLen = 32

// DefaultRateLimitRPM est le plafond par défaut du rate-limiter applicatif
// (requêtes/minute/IP). 300 donne de la marge à un SPA data-heavy qui émet
// plusieurs dizaines d'appels /api/v1/* par page, une fois le bucket keyé sur la
// vraie IP client (LEVELUP_TRUST_PROXY_HEADERS=true derrière un proxy).
const DefaultRateLimitRPM = 300

// AppConfig centralise la configuration de l'application.
type AppConfig struct {
	RepoRoot        string
	DBProfilesPath  string
	AppSettingsPath string
	SessionDir      string
	DemoMode        bool
	DemoFixturesDir string
	// DemoLocale : locale UI forcée en mode démo (vitrine publique). Défaut "en"
	// (audience internationale). Surchargeable via LEVELUP_DEMO_LOCALE — les tests
	// E2E la pinnent à "fr" pour exercer l'UI française (specs FR). Le visiteur peut
	// toujours basculer la langue en session côté client.
	DemoLocale    string
	APIHost       string
	APIPort       int
	SessionSecret string
	CORSOrigins   []string
	Lang          string
	AppVersion    string
	// SharedProvider (commit 8g, retypé 8i) — injecté au boot par main.go en
	// mode B-swap (LEVELUP_USE_SHARED_PROVIDER=1). Provider du shared Halo
	// Infinite (DefaultSlug), source unique des writes Infinite + recherche
	// gamertag. Passé aux PlayerPoolConfig (satisfait duckdb.SharedReader
	// structurellement) et au SyncEngine via WithSharedProvider. nil en mode legacy.
	SharedProvider sharedprovider.Provider
	// SharedManager (multi-titre) — le Manager qui déduplique les Provider par
	// chemin. Injecté au boot par main.go en même temps que SharedProvider (mode
	// B-swap). Permet de résoudre le provider du shared d'un AUTRE titre (Halo 5+)
	// par son path — la lecture per-titre passe par For(SharedDBPath(slug)). Pour
	// DefaultSlug, For() retourne le provider boot (caché par path) → byte-identique
	// à SharedProvider. nil en mode legacy/kill-switch → fallback SharedProvider.
	SharedManager *sharedprovider.Manager
	// TitleReadyNotifier (multi-titre, MT-19 / axe E) — injecté au boot par main.go
	// APRÈS construction du ServiceRegistry (api.BuildTitleReadyNotifier). Émet une
	// notification « titre prêt » (catégorie title_ready) lorsqu'un titre live (Halo
	// 5+, servi par un Runner dédié) a des matchs, dans le flux de notifications du
	// titre PAR DÉFAUT (là où l'utilisateur, invité à « retourner sur Halo Infinite »
	// le temps du backfill, la verra). Idempotence durable côté impl (watermark
	// sync_meta). nil en CLI/tests → le Runner saute l'émission. Signature stdlib-only
	// pour éviter tout cycle d'import api↔config↔livesync.
	TitleReadyNotifier func(ctx context.Context, titleSlug, gamertag, xuid string, inserted int)
	// ProgressionAfterSync (multi-titre) — injecté au boot par main.go APRÈS le
	// ServiceRegistry (api.BuildProgressionAfterSyncHook). Fait tourner le pipeline
	// Progression V2 (streaks/records/milestones/coach) pour le TITRE d'un joueur
	// après un cycle de sync live qui a inséré des matchs — équivalent title-agnostic
	// du post-sync HINF (buildPostSyncDeltaHook), MAIS avec les deps de base (SANS le
	// PrestigeBundle/CoachAdvisor mono-titre). Best-effort. nil en CLI/tests → le
	// Runner saute l'étape. Signature stdlib-only (zéro cycle api↔config↔livesync).
	ProgressionAfterSync func(ctx context.Context, titleSlug, playerSlug string)
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
	// Environment : "production" active le garde-fou de démarrage Validate() qui
	// refuse de booter avec une configuration non sûre (secret par défaut, auth
	// désactivée, CORS localhost). Lit LEVELUP_ENV. Vide / "development" (défaut) →
	// pas de fail-fast, mais les avertissements restent émis au boot.
	Environment string
	// TrustProxyHeaders : n'active la confiance dans les en-têtes d'IP client
	// (X-Real-IP / X-Forwarded-For / True-Client-IP, via chi RealIP) QUE si le
	// serveur tourne derrière un reverse proxy de confiance qui assainit ces
	// en-têtes. Lit LEVELUP_TRUST_PROXY_HEADERS (1/true/yes). Défaut : false —
	// RemoteAddr reste l'adresse TCP réelle du peer (rate-limit, audit et le
	// garde LoopbackOnly des endpoints /_diag ne sont alors pas falsifiables).
	TrustProxyHeaders bool
	// InstanceLocked : verrou « instance fermée ». Quand true, bloque la création
	// de NOUVELLES identités/BDD (register, SSO sur xuid inconnu, /setup/players)
	// sans casser les utilisateurs/joueurs existants. Lit LEVELUP_INSTANCE_LOCKED
	// (1/true/yes) = verrou FORCÉ au boot (override). Le verrou est aussi activable
	// à chaud via app_settings.json:instance_locked (cf. settings store). Le verrou
	// effectif = env OU app_settings.
	InstanceLocked bool
	// CookieSecure pilote le flag Secure du cookie de session : "auto" (défaut,
	// décision par schéma de requête — TLS natif ou X-Forwarded-Proto derrière un
	// proxy de confiance), "true" (force Secure) ou "false" (force non-Secure,
	// filet de secours ops). Lit LEVELUP_COOKIE_SECURE. NE PAS coupler au secret
	// de session : un secret custom en dev local ne doit pas forcer Secure (sinon
	// le navigateur jette le cookie sur http://localhost → session perdue).
	CookieSecure string
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
	// Backup : configuration du scheduler de backup DuckDB via restic.
	// Activé par LEVELUP_BACKUP_ENABLED=true.
	Backup BackupConfig
	// PrestigeEnabled gate les 16 routes Prestige ET le hook post-sync (gate UNIQUE,
	// C7/DEC-4). Source : prestige.IsEnabled → prestige_enabled dans app_settings.json,
	// override d'urgence PRESTIGE_ENABLED. Défaut : true (activation actée, ADR 0005).
	PrestigeEnabled bool
	// WebDistDir : répertoire du build React (Vite) servi en SPA par le routeur.
	// Lit LEVELUP_WEB_DIST (posé par le Dockerfile/compose → /app/apps/web/dist).
	// Vide en dev (Vite sert le front sur :5173) → la SPA n'est pas montée.
	WebDistDir string
	// RateLimitRPM : plafond de requêtes par minute et par IP du rate-limiter
	// applicatif (httprate). Lit LEVELUP_RATE_LIMIT_RPM. Défaut : DefaultRateLimitRPM.
	// ATTENTION : le limiter clé sur RemoteAddr — derrière un reverse proxy, il faut
	// LEVELUP_TRUST_PROXY_HEADERS=true, sinon toutes les requêtes partagent le bucket
	// de l'IP du proxy (127.0.0.1) et le site sature en 429 sous trafic public.
	RateLimitRPM int
	// PersistBatchAsync active le drainage asynchrone du persister batch (queue WAL
	// + worker). Kill-switch : LEVELUP_PERSIST_BATCH_ASYNC=0 → chemin synchrone.
	// Défaut : true. Cycle de vie du kill-switch documenté au câblage boot
	// (cmd/server/main.go). Source UNIQUE lue par main.go, sync_v2_wiring.go et le
	// scheduler (élimine la triple lecture os.Getenv — CR A6).
	PersistBatchAsync bool
	// EventsConvergence active la passe de convergence des highlight_events
	// (scheduler + trigger immédiat). Kill-switch : LEVELUP_EVENTS_CONVERGENCE=0.
	// Défaut : true. Cycle de vie : internal/scheduler/auto_sync.go.
	EventsConvergence bool
	// EventsConvergenceMax borne le nombre de matchs traités par tick de convergence.
	// Lit LEVELUP_EVENTS_CONVERGENCE_MAX (valeur <= 0 ignorée). Défaut :
	// DefaultEventsConvergenceMax.
	EventsConvergenceMax int
	// BuildWorkerToken authentifie les OUVRIERS de la file de construction sur les
	// routes /internal/build-queue/* (piste F §1/§2). Lit
	// LEVELUP_BUILD_WORKER_TOKEN. VIDE PAR DÉFAUT, et c'est le comportement voulu :
	// sans jeton, le protocole ouvrier répond 503 et la feature n'existe pas — le
	// dépôt est PUBLIC, personne ne doit hériter d'une porte ouverte en installant
	// LevelUp. Ce jeton n'ouvre AUCUN accès Halo ni base : il ne donne que le droit
	// de prendre un travail déjà résolu, d'y DÉPOSER l'artefact construit, et d'en
	// rendre le résultat.
	//
	// IL COMMANDE AUSSI LE FIL DE L'EAU : sans jeton, le placement « ouvrier »
	// (replay_build_location) dégrade en « aucune construction » — enfiler quand
	// personne ne viendra vider la file résoudrait un manifeste Halo par match, à
	// chaque cycle, pour rien (cf. replaybuild.DecidePlacement).
	BuildWorkerToken string
}

// DefaultEventsConvergenceMax est le plafond par défaut de matchs traités par la
// passe de convergence events à chaque tick scheduler.
const DefaultEventsConvergenceMax = 50

// BackupConfig centralise la configuration du backup périodique.
// Comportement (enabled, interval, retention) : app_settings.json.
// Chemins machine (BackupDir, ResticRepo) : variables d'environnement.
type BackupConfig struct {
	Enabled     bool          // app_settings: backup_enabled
	BackupDir   string        // env: LEVELUP_BACKUP_DIR
	Interval    time.Duration // app_settings: backup_interval ("6h", "24h"…)
	KeepDaily   int           // app_settings: backup_keep_daily
	KeepWeekly  int           // app_settings: backup_keep_weekly
	KeepMonthly int           // app_settings: backup_keep_monthly
	ResticRepo  string        // env: RESTIC_REPOSITORY
}

// loadBackupConfig lit le comportement depuis app_settings.json et les chemins
// depuis les variables d'environnement. Suit le même pattern que loadUserTimezone.
func BootstrapEnvLocal() {
	repoRoot := getEnvOrDefault("LEVELUP_REPO_ROOT", autoDetectRepoRoot())
	if repoRoot == "" {
		return
	}
	loadEnvLocal(filepath.Join(repoRoot, ".env.local"))
}

// Load charge la configuration depuis les variables d'environnement.
// Les valeurs par défaut correspondent au développement local.
func Load() (*AppConfig, error) {
	repoRoot := getEnvOrDefault("LEVELUP_REPO_ROOT", autoDetectRepoRoot())
	// Charger .env.local avant toute lecture de variable d'environnement,
	// pour que SPNKR_OAUTH_REFRESH_TOKEN_* et autres vars locales soient disponibles.
	// (No-op si main() a déjà appelé BootstrapEnvLocal — loadEnvLocal n'écrase
	// jamais une var déjà définie.)
	loadEnvLocal(filepath.Join(repoRoot, ".env.local"))
	demoMode := strings.ToLower(getEnvOrDefault("LEVELUP_DEMO_MODE", "false")) == "true"

	cfg := &AppConfig{
		RepoRoot:          repoRoot,
		DBProfilesPath:    getEnvOrDefault("LEVELUP_DB_PROFILES", filepath.Join(repoRoot, "db_profiles.json")),
		AppSettingsPath:   getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json")),
		SessionDir:        getEnvOrDefault("LEVELUP_SESSION_DIR", filepath.Join(repoRoot, "data", "sessions")),
		DemoMode:          demoMode,
		DemoFixturesDir:   getEnvOrDefault("LEVELUP_DEMO_FIXTURES_DIR", filepath.Join(repoRoot, "data", "demo")),
		DemoLocale:        getEnvOrDefault("LEVELUP_DEMO_LOCALE", "en"),
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
		Environment:       getEnvOrDefault("LEVELUP_ENV", ""),
		TrustProxyHeaders: parseBoolEnv(getEnvOrDefault("LEVELUP_TRUST_PROXY_HEADERS", "")),
		InstanceLocked:    parseBoolEnv(getEnvOrDefault("LEVELUP_INSTANCE_LOCKED", "")),
		CookieSecure:      parseCookieSecureMode(getEnvOrDefault("LEVELUP_COOKIE_SECURE", "")),
		WebDistDir:        getEnvOrDefault("LEVELUP_WEB_DIST", ""),
		RateLimitRPM:      getEnvInt("LEVELUP_RATE_LIMIT_RPM", DefaultRateLimitRPM),
	}
	appSettingsPath := getEnvOrDefault("LEVELUP_APP_SETTINGS", filepath.Join(repoRoot, "app_settings.json"))
	cfg.UserTimezone = loadUserTimezone(appSettingsPath)
	cfg.CurrentCSRSeasonID = loadCSRSeasonID(appSettingsPath)
	cfg.MediaCapturesBaseDir = loadMediaCapturesBaseDir(appSettingsPath)
	cfg.Backup = loadBackupConfig(repoRoot, appSettingsPath)
	cfg.PrestigeEnabled = prestige.IsEnabled(appSettingsPath)
	cfg.PersistBatchAsync = getEnvOrDefault("LEVELUP_PERSIST_BATCH_ASYNC", "") != "0"
	cfg.EventsConvergence = getEnvOrDefault("LEVELUP_EVENTS_CONVERGENCE", "") != "0"
	cfg.EventsConvergenceMax = getEnvInt("LEVELUP_EVENTS_CONVERGENCE_MAX", DefaultEventsConvergenceMax)
	if cfg.EventsConvergenceMax <= 0 {
		cfg.EventsConvergenceMax = DefaultEventsConvergenceMax
	}
	cfg.BuildWorkerToken = strings.TrimSpace(getEnvOrDefault("LEVELUP_BUILD_WORKER_TOKEN", ""))
	return cfg, nil
}

// IsProduction indique si le serveur tourne en mode production (LEVELUP_ENV=production),
// ce qui active le garde-fou fail-fast de Validate().
func (c *AppConfig) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.Environment), "production")
}

// SecurityWarnings retourne la liste des réglages non sûrs pour un déploiement
// multi-user exposé. Liste vide = configuration sûre. Sert de source unique à la
// fois au garde-fou fail-fast en production (Validate) et au logging
// d'avertissement au démarrage en dev.
func (c *AppConfig) SecurityWarnings() []string {
	var w []string
	switch {
	case c.SessionSecret == DefaultSessionSecret:
		w = append(w, "LEVELUP_SESSION_SECRET non défini : la clé de signature des cookies est la valeur publique par défaut (cookies forgeables)")
	case len(c.SessionSecret) < minSessionSecretLen:
		w = append(w, fmt.Sprintf("LEVELUP_SESSION_SECRET trop court (%d octets, minimum %d)", len(c.SessionSecret), minSessionSecretLen))
	}
	if strings.EqualFold(c.AuthMode, "none") {
		w = append(w, "LEVELUP_AUTH_MODE=none : le contrôle d'accès par joueur (ownership xuid) est entièrement désactivé")
	}
	if c.corsAllLocalhost() {
		w = append(w, "LEVELUP_CORS_ORIGINS non défini : les origines CORS/CSRF autorisées sont limitées à localhost")
	}
	if strings.EqualFold(c.CookieSecure, "false") {
		w = append(w, "LEVELUP_COOKIE_SECURE=false : le cookie de session est posé sans flag Secure (à réserver au dev local / debug ops ; jamais en prod exposée)")
	}
	return w
}

// corsAllLocalhost retourne true si aucune origine CORS « réelle » (non-localhost)
// n'est configurée — typiquement le défaut localhost:5173/5174.
func (c *AppConfig) corsAllLocalhost() bool {
	if len(c.CORSOrigins) == 0 {
		return true
	}
	for _, o := range c.CORSOrigins {
		if !strings.Contains(o, "localhost") && !strings.Contains(o, "127.0.0.1") {
			return false
		}
	}
	return true
}

// Validate applique le garde-fou de démarrage. En production (LEVELUP_ENV=production)
// et hors DemoMode, refuse de démarrer si la configuration est non sûre. Hors
// production, ne renvoie jamais d'erreur — les avertissements restent consultables
// via SecurityWarnings() pour un log au boot. À appeler explicitement depuis
// cmd/server ; Load() ne valide pas (les CLI et tests réutilisent Load avec des
// défauts de dev).
func (c *AppConfig) Validate() error {
	if c.DemoMode || !c.IsProduction() {
		return nil
	}
	if w := c.SecurityWarnings(); len(w) > 0 {
		return fmt.Errorf("configuration non sûre pour la production (LEVELUP_ENV=production) :\n  - %s", strings.Join(w, "\n  - "))
	}
	return nil
}

// EnvMediaDeleteSource est le nom de la variable d'environnement qui force la
// politique de suppression du fichier source après transcodage HLS (override
// process-global, priorité maximale). Valeur permissive (1/true/yes/on → supprime).
const EnvMediaDeleteSource = "LEVELUP_MEDIA_DELETE_SOURCE"

// ResolveMediaDeleteSource résout la politique effective de suppression du fichier
// source (.mkv/.avi…) après un transcodage HLS réussi. Fonction PURE (aucune IO)
// pour rester testable. Précédence :
//  1. envRaw (LEVELUP_MEDIA_DELETE_SOURCE) : s'il est renseigné, il gagne
//     (parseBoolEnv permissif) ;
//  2. sinon storeVal (app_settings.json:media_delete_source_after_transcode) si
//     non-nil : la valeur explicite de l'opérateur ;
//  3. sinon défaut par environnement = isProd (supprime en prod — disque rare, HLS
//     = forme canonique servie ; conserve en local — le source est la copie maître).
//
// envRaw vide N'EST PAS traité comme « false » : une var non définie doit laisser
// la main au store puis au défaut env (sinon toute instance sans la var forcerait
// la conservation, y compris en prod). La distinction « défini/non défini » se fait
// donc sur la chaîne brute (vide = non défini), pas sur le booléen parsé.
func ResolveMediaDeleteSource(envRaw string, storeVal *bool, isProd bool) bool {
	if strings.TrimSpace(envRaw) != "" {
		return parseBoolEnv(envRaw)
	}
	if storeVal != nil {
		return *storeVal
	}
	return isProd
}

// parseBoolEnv interprète une valeur d'environnement comme un booléen permissif
// (1/true/yes/on, insensible à la casse). Vide ou non reconnu → false.
func parseBoolEnv(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// parseCookieSecureMode normalise LEVELUP_COOKIE_SECURE en l'un des modes connus :
// "true", "false" ou "auto" (défaut). Toute valeur vide ou non reconnue retombe
// sur "auto" (décision par schéma de requête) — choix sûr car il pose Secure dès
// qu'on détecte du HTTPS, sans le forcer à tort sur du HTTP local.
func parseCookieSecureMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return "true"
	case "0", "false", "no", "off":
		return "false"
	default:
		return "auto"
	}
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
	return readCSRSeasonIDFromFile(settingsPath)
}

// readCSRSeasonIDFromFile lit csr_season_id dans un fichier JSON (global ou
// overlay titre). "" si fichier absent / illisible / champ absent. PMT-4 : réutilisé
// par CSRSeasonIDForTitle pour lire l'overlay du titre.
func readCSRSeasonIDFromFile(path string) string {
	data, err := os.ReadFile(path)
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

// CSRSeasonIDForTitle résout la saison CSR pour un titre (PMT-4). Précédence :
// env LEVELUP_CSR_SEASON_ID (override process) > overlay titre (csr_season_id dans
// data/titles/<slug>/settings.json) > global (CurrentCSRSeasonID). **Dégradation** :
// si le titre n'a pas la capability `Ranked` (ou est inconnu du registre), retourne
// "" — le sync CSR est skippé proprement (jamais de saison Halo héritée par erreur).
// Routage par capability, jamais par comparaison de slug (no_slug_comparison vert).
func (c *AppConfig) CSRSeasonIDForTitle(ctx context.Context, slug string, reg *titlePkg.Registry) string {
	if id := os.Getenv("LEVELUP_CSR_SEASON_ID"); id != "" {
		return id // override process-global (parité loadCSRSeasonID)
	}
	if reg == nil {
		reg = titlePkg.DefaultRegistry() // call-sites sans registre sous la main
	}
	desc := reg.Get(slug)
	if desc == nil || !desc.HasCapability(titlePkg.CapRanked) {
		slog.DebugContext(ctx, "csr_season.degraded", "title", slug, "reason", "cap_ranked_absent")
		return ""
	}
	pr := titlePkg.NewPathResolver(c.RepoRoot, reg)
	if id := readCSRSeasonIDFromFile(pr.TitleSettingsPath(slug)); id != "" {
		return id // overlay fichier du titre (data/titles/<slug>/settings.json)
	}
	if id := strings.TrimSpace(desc.CSRSeasonID); id != "" {
		return id // saison CSR FIXE déclarée dans le descripteur (title.toml).
		// Halo 5 = "h5-lifetime" (service record arena = agrégat sans saison).
	}
	return c.CurrentCSRSeasonID // fallback global
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
