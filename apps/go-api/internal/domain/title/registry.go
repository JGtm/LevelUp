// Package title — registre des titres supportés et résolution de chemins.
//
// Sprint 44 WP0 : colonne vertébrale du runtime multi-titres.
// Le TitleRegistry est la source de vérité centrale pour les titres supportés.
// Le PathResolver est le point unique de résolution des chemins title-aware.
package title

import (
	"fmt"
	"path/filepath"
	"sync"
)

// ---------------------------------------------------------------------------
// TitleDescriptor — description d'un titre supporté
// ---------------------------------------------------------------------------

// Status représente l'état d'un titre dans le registre.
type Status string

const (
	StatusActive     Status = "active"
	StatusComingSoon Status = "coming_soon"
	StatusArchived   Status = "archived"
)

// Capability décrit une fonctionnalité supportée par un titre.
type Capability string

const (
	CapMatchmaking Capability = "matchmaking"
	CapFirefight   Capability = "firefight"
	CapForge       Capability = "forge"
	CapMedia       Capability = "media"
	CapRanked      Capability = "ranked"
	CapCareer      Capability = "career"
	// CapSeasonPass — progression de pass saisonnier / Battlepass (Halo Infinite).
	// Absente ⇒ la page career/season-pass dégrade en FeatureUnavailable + l'onglet
	// « Pass saisonnier » est masqué. Halo 5 ne l'expose PAS (pas de Battlepass ;
	// l'inventaire REQ personnel n'est pas servi — sonde 404 — donc aucune surface
	// de remplacement câblée).
	CapSeasonPass   Capability = "season_pass"
	CapAssetImages  Capability = "asset.images" // Asset Drawer — thumbnails maps & armes
	CapAchievements Capability = "achievements" // Xbox achievements bilingues (page Achievements)
	CapEngagement   Capability = "engagement"   // Score d'engagement intra-match + coefficients personnels
	CapLUSR         Capability = "lusr"         // Rating interne LevelUp (LUSR v2 TrueSkill2) — calcul post-sync + Stratégie C

	// CapWorldLeaderboard — classements mondiaux (CSR scrapé Waypoint + stats
	// agrégées par titre). Absente ⇒ page leaderboard dégrade en vide+200 (PMT-7).
	CapWorldLeaderboard Capability = "world.leaderboard"

	// CapNativeKillMechanics — le titre fournit NATIVEMENT le détail des kills par
	// mécanique (assassinats + compétences spartiate : ground pound, shoulder bash),
	// agrégés par-joueur dans le carnage. Halo 5 only (Infinite ne les expose pas).
	// Absente ⇒ le front masque les sections « assassinats / compétences spartiate ».
	CapNativeKillMechanics Capability = "native_kill_mechanics"

	// CapWeaponKills — le titre produit le détail des kills PAR ARME (pipeline
	// film-decoder côté ingestion). Halo Infinite : oui (films) ; Halo 5 : non
	// (pas de film-decoder weapon-per-kill). Sert le prédicat de complétude du
	// snapshot (Phase 2) : un titre SANS cette cap n'EXIGE pas les weapon-kills
	// pour qu'un match soit « complet ». À NE PAS confondre avec
	// CapNativeKillMechanics (assassinats/ground-pound natifs, Halo-5-only).
	CapWeaponKills Capability = "weapon_kills"
)

// TitleDescriptor décrit un titre supporté avec ses métadonnées.
type TitleDescriptor struct {
	Slug         string       `json:"slug"`
	Name         string       `json:"name"`
	Provider     string       `json:"provider"` // ex: "halo_infinite", "halo_mcc"
	IconURL      string       `json:"icon_url"`
	Status       Status       `json:"status"`
	Capabilities []Capability `json:"capabilities"`
	IsDefault    bool         `json:"is_default"` // halo_infinite = true

	// IsInternal marque un titre INTERNE/TEST (fixture multi-titre comme
	// synthetic_title_b) : découvert + enregistré + inspectable côté admin, mais
	// JAMAIS exposé dans le switcher utilisateur (cf. PublicTitles). Évite qu'une
	// fixture de test « fuite » dans l'UI prod. Défaut false (vrai titre).
	IsInternal bool `json:"is_internal"`

	// Identifiants plateforme pour le matching de présence (watcher)
	XboxTitleID string `json:"xbox_title_id"` // ex: "1144039928" pour Halo Infinite
	SteamAppID  string `json:"steam_app_id"`  // ex: "1336960" pour Halo Infinite (Steam)

	// PlacementMatches est le nombre de matchs de placement des modes classés du
	// titre (Halo Infinite = 5 depuis Season 3 ; Halo 5 = 10). Externalisé du code
	// (le « 5 »/« 10 » était hardcodé). 0 = non déclaré → le consommateur applique
	// son défaut. NB : Halo Infinite a aussi une table de seuils PAR SAISON (10→5)
	// qui prime ; cette valeur est le défaut du titre.
	PlacementMatches int `json:"placement_matches"`

	// CSRSeasonID est l'identifiant de saison CSR FIXE du titre (overlay descripteur,
	// PMT-4). Pour Halo 5, le service record arena est un agrégat LIFETIME sans
	// saison → "h5-lifetime" (déclaré dans title.toml). Consommé par
	// CSRSeasonIDForTitle (précédence : env > overlay fichier > CE champ > global) :
	// aligne la LECTURE (GetCSRSnapshots WHERE season_id) sur l'ÉCRITURE (sync h5).
	// "" = non déclaré → fallback global (CurrentCSRSeasonID), comportement Infinite.
	CSRSeasonID string `json:"csr_season_id"`
}

// HasCapability vérifie si le titre supporte une fonctionnalité.
func (t *TitleDescriptor) HasCapability(cap Capability) bool {
	for _, c := range t.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// IsActive indique si le titre est au statut actif. Prédicat pur (zéro IO) —
// MT-22 (PMT-8). Utilisé par le middleware RequireActiveTitle et tout
// consommateur qui n'a pas le registre sous la main.
func (t *TitleDescriptor) IsActive() bool {
	return t.Status == StatusActive
}

// ---------------------------------------------------------------------------
// TitleRegistry — registre centralisé des titres
// ---------------------------------------------------------------------------

// Registry est le registre centralisé des titres supportés.
type Registry struct {
	mu     sync.RWMutex
	titles map[string]*TitleDescriptor
}

// DefaultSlug est le slug du titre par défaut (Halo Infinite).
const DefaultSlug = "halo_infinite"

// defaultRegistry : registre PARTAGÉ par les helpers package-level (XboxTitleIDFor)
// et les call-sites de service qui n'ont pas d'instance sous la main. Lazy par
// défaut (built-in halo_infinite), mais REMPLAÇABLE au boot par un registre piloté
// par config via SetDefaultRegistry (MT-16 / day-one 2e titre). Les descripteurs
// sont la source UNIQUE de vérité.
var (
	defaultRegistryMu   sync.RWMutex
	defaultRegistryInst *Registry
)

func defaultRegistry() *Registry {
	defaultRegistryMu.RLock()
	r := defaultRegistryInst
	defaultRegistryMu.RUnlock()
	if r != nil {
		return r
	}
	defaultRegistryMu.Lock()
	defer defaultRegistryMu.Unlock()
	if defaultRegistryInst == nil {
		defaultRegistryInst = NewRegistry()
	}
	return defaultRegistryInst
}

// SetDefaultRegistry remplace le registre partagé par défaut (appelé UNE fois au
// boot avec le registre piloté par config : built-in halo_infinite + titres
// additionnels découverts sous config/titles/*). Thread-safe. nil est ignoré
// (garde défensive — on ne vide jamais le registre partagé).
//
// Après cet appel, tous les call-sites passant par DefaultRegistry() (front
// switcher BuildAvailableTitles, résolution de titre, XboxTitleIDFor, etc.) voient
// les titres additionnels. Doit être posé AVANT que le serveur n'accepte du trafic.
func SetDefaultRegistry(reg *Registry) {
	if reg == nil {
		return
	}
	defaultRegistryMu.Lock()
	defaultRegistryInst = reg
	defaultRegistryMu.Unlock()
}

// DefaultRegistry expose le registre partagé. Avant SetDefaultRegistry, retombe
// sur le built-in lazy (halo_infinite seul) — donc mono-titre sûr en test/CLI.
func DefaultRegistry() *Registry { return defaultRegistry() }

// XboxTitleIDFor retourne l'identifiant Xbox numérique (en string) pour un slug
// LevelUp, lu depuis le descripteur du titre (registry-driven, PMT-6 — supprime
// le switch qui dupliquait TitleDescriptor.XboxTitleID). Utilisé lors du sync
// achievements pour filtrer l'API Xbox par titre. "" si le slug est inconnu —
// l'appelant doit gérer ce cas. (Halo Infinite standalone = 2043073184, porté par
// son descripteur ; distinct de 1144039928 = MCC.)
func XboxTitleIDFor(slug string) string {
	return defaultRegistry().XboxTitleIDFor(slug)
}

// XboxTitleIDFor lit l'XboxTitleID du descripteur du titre. "" si inconnu.
func (r *Registry) XboxTitleIDFor(slug string) string {
	if d := r.Get(slug); d != nil {
		return d.XboxTitleID
	}
	return ""
}

// ServiceConfigIDFor retourne le ServiceConfigID (SCID) Xbox pour un slug LevelUp.
// Retourne "" si le slug n'est pas reconnu ou si le SCID n'est pas encore connu.
// Le SCID HI sera confirmé lors du premier sync avec le bon title ID.
func ServiceConfigIDFor(slug string) string {
	return "" // SCID HI non encore confirmé — le filtre titleId est suffisant
}

// NewRegistry crée un registre avec les titres par défaut.
func NewRegistry() *Registry {
	r := &Registry{
		titles: make(map[string]*TitleDescriptor),
	}
	// Halo Infinite est toujours enregistré.
	r.Register(&TitleDescriptor{
		Slug:     DefaultSlug,
		Name:     "Halo Infinite",
		Provider: DefaultSlug,
		Status:   StatusActive,
		Capabilities: []Capability{
			CapMatchmaking, CapFirefight, CapForge,
			CapMedia, CapRanked, CapCareer, CapSeasonPass, CapAssetImages,
			CapAchievements, CapEngagement, CapLUSR, CapWorldLeaderboard,
			CapWeaponKills,
		},
		IsDefault:        true,
		XboxTitleID:      "2043073184",
		SteamAppID:       "1336960",
		PlacementMatches: 5, // défaut Halo Infinite depuis Season 3 (table par-saison prime)
	})
	return r
}

// Register ajoute un titre au registre.
func (r *Registry) Register(desc *TitleDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.titles[desc.Slug] = desc
}

// Get retourne le descripteur d'un titre ou nil si inconnu.
func (r *Registry) Get(slug string) *TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.titles[slug]
}

// Exists vérifie si un titre est enregistré.
func (r *Registry) Exists(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.titles[slug]
	return ok
}

// IsActive vérifie qu'un titre est enregistré ET actif (Status == StatusActive).
// MT-22 (PMT-8) : le Status est enfin consulté. Le gating runtime des routes
// title-scoped passe par le middleware RequireActiveTitle (503 title_unavailable),
// pas par le seam TitleExtractor (qui résout n'importe quel titre connu).
func (r *Registry) IsActive(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	td, ok := r.titles[slug]
	return ok && td.IsActive()
}

// Active retourne uniquement les titres au statut actif.
func (r *Registry) Active() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.IsActive() {
			out = append(out, t)
		}
	}
	return out
}

// NonArchived retourne les titres actifs ET coming_soon (exclut uniquement
// archived). MT-22 (PMT-8) : le switcher front liste les titres jouables +
// ceux « bientôt disponibles » (annotés via leur Status), mais jamais les
// titres retirés (archived). Cf. BuildAvailableTitles.
func (r *Registry) NonArchived() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.Status != StatusArchived {
			out = append(out, t)
		}
	}
	return out
}

// PublicTitles retourne les titres exposables dans le SWITCHER UTILISATEUR :
// exclut les archived (titres retirés) ET les internal (fixtures de test comme
// synthetic_title_b). C'est la vue user-facing — par opposition à NonArchived
// (qui inclut les internes) et All (admin). Conserve les coming_soon non-internes
// (badge « bientôt ») et leur Status. Cf. buildAvailableTitlesFrom.
func (r *Registry) PublicTitles() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.Status == StatusArchived || t.IsInternal {
			continue
		}
		out = append(out, t)
	}
	return out
}

// All retourne tous les titres enregistrés.
func (r *Registry) All() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		out = append(out, t)
	}
	return out
}

// Default retourne le titre par défaut (halo_infinite).
func (r *Registry) Default() *TitleDescriptor {
	return r.Get(DefaultSlug)
}

// ---------------------------------------------------------------------------
// PathResolver — résolution centralisée des chemins title-aware
// ---------------------------------------------------------------------------

// PathResolver résout les chemins physiques en fonction du titre.
// Tous les chemins passent par ce résolveur — aucun filepath.Join(repoRoot, "data", ...)
// n'est autorisé dans les handlers, services, ops ou tests.
type PathResolver struct {
	repoRoot string
	registry *Registry
}

// NewPathResolver crée un PathResolver.
// Si registry n'est pas fourni, un registre par défaut est utilisé.
func NewPathResolver(repoRoot string, registry ...*Registry) *PathResolver {
	var reg *Registry
	if len(registry) > 0 && registry[0] != nil {
		reg = registry[0]
	} else {
		reg = NewRegistry()
	}
	return &PathResolver{repoRoot: repoRoot, registry: reg}
}

// RepoRoot retourne la racine du repo.
func (p *PathResolver) RepoRoot() string {
	return p.repoRoot
}

// --- Chemins title-aware ---

// TitleDataDir retourne le répertoire racine des données d'un titre.
// Ex: data/titles/halo_infinite/
func (p *PathResolver) TitleDataDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "data", "titles", titleSlug)
}

// WarehouseDir retourne le répertoire warehouse d'un titre.
// Ex: data/titles/halo_infinite/warehouse/
func (p *PathResolver) WarehouseDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "warehouse")
}

// SharedDBPath retourne le chemin de la base partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
func (p *PathResolver) SharedDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_matches_v2.duckdb")
}

// SnapshotsDir retourne le répertoire racine des snapshots Parquet immuables
// d'un titre (durabilité / lecture découplée des écritures, Phase 2).
// Ex: data/titles/halo_infinite/warehouse/snapshots/
func (p *PathResolver) SnapshotsDir(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "snapshots")
}

// SnapshotVersionDir retourne le répertoire d'une version de snapshot. Le nom
// est zéro-paddé pour que l'ordre lexicographique du système de fichiers
// coïncide avec l'ordre chronologique des versions.
// Ex: data/titles/halo_infinite/warehouse/snapshots/v00000000000000000042/
func (p *PathResolver) SnapshotVersionDir(titleSlug string, version int64) string {
	return filepath.Join(p.SnapshotsDir(titleSlug), fmt.Sprintf("v%020d", version))
}

// SnapshotCurrentManifestPath retourne le chemin du pointeur atomique CURRENT
// désignant la version de snapshot active (flip via os.Rename).
// Ex: data/titles/halo_infinite/warehouse/snapshots/CURRENT.json
func (p *PathResolver) SnapshotCurrentManifestPath(titleSlug string) string {
	return filepath.Join(p.SnapshotsDir(titleSlug), "CURRENT.json")
}

// GlobalXuidAliasesDBPath retourne le chemin de la base globale xbox aliases
// (P5.1, ADR 0008). Le mapping xuid → gamertag est un identifiant Microsoft
// (Xbox Services) qui ne dépend pas du titre — donc DB globale partagée
// par tous les titres.
//
// Ex: data/global/xbox_aliases.duckdb
func (p *PathResolver) GlobalXuidAliasesDBPath() string {
	return filepath.Join(p.repoRoot, "data", "global", "xbox_aliases.duckdb")
}

// MetadataDBPath retourne le chemin de la base metadata d'un titre.
// Ex: data/titles/halo_infinite/warehouse/metadata.duckdb
func (p *PathResolver) MetadataDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "metadata.duckdb")
}

// SharedPVEDBPath retourne le chemin de la base PvE partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_pve.duckdb
func (p *PathResolver) SharedPVEDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_pve.duckdb")
}

// SharedSocialDBPath retourne le chemin de la base sociale partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_social.duckdb
func (p *PathResolver) SharedSocialDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_social.duckdb")
}

// PlayerDir retourne le répertoire d'un joueur pour un titre.
// Ex: data/titles/halo_infinite/players/Chocoboflor/
func (p *PathResolver) PlayerDir(titleSlug, gamertag string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "players", gamertag)
}

// PlayerDBPath retourne le chemin de la DB stats d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/stats.duckdb
func (p *PathResolver) PlayerDBPath(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "stats.duckdb")
}

// PlayerArchiveDir retourne le répertoire d'archive d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/archive/
func (p *PathResolver) PlayerArchiveDir(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "archive")
}

// PlayerCapturesDir retourne le répertoire captures d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/captures/
func (p *PathResolver) PlayerCapturesDir(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "captures")
}

// ResolveCapturesDir centralise la résolution du dossier captures d'un joueur.
//
// Tous les call-sites manipulant un dossier de médias (upload, scan, reindex,
// URL serving, CLI) DOIVENT passer par cette fonction pour garantir un
// comportement uniforme et éviter qu'un fallback hardcodé crée un dossier
// fantôme jamais relu.
//
// Règle :
//   - baseDir non vide (settings.MediaCapturesBaseDir) → {baseDir}/{gamertag}
//   - baseDir vide → fallback interne PlayerCapturesDir = .../{gamertag}/captures
func (p *PathResolver) ResolveCapturesDir(titleSlug, gamertag, baseDir string) string {
	if baseDir != "" {
		return filepath.Join(baseDir, gamertag)
	}
	return p.PlayerCapturesDir(titleSlug, gamertag)
}

// MediaDataDir retourne la base média interne canonique {repoRoot}/data/media.
//
// Layout title-agnostic et PLAT : data/media/{gamertag}/{thumbs,hls,...} — distinct
// du PlayerCapturesDir per-titre (data/titles/.../{gamertag}/captures). C'est la
// localisation réelle des médias en production (montée sur /app/data/media dans le
// conteneur). Sert de fallback de résolution quand le media_captures_base_dir
// configuré est absent ou invalide (garde-fou incident prod 2026-06-13 : un chemin
// Windows déployé sur le VPS Linux rendait TOUS les médias illisibles).
func (p *PathResolver) MediaDataDir() string {
	return filepath.Join(p.repoRoot, "data", "media")
}

// BackupDir retourne le répertoire de backup d'un joueur pour un titre.
// Ex: data/titles/halo_infinite/backups/Chocoboflor/
func (p *PathResolver) BackupDir(titleSlug, gamertag string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "backups", gamertag)
}

// DemoFixturesDir retourne le répertoire de fixtures démo pour un titre.
// Ex: tests/fixtures/titles/halo_infinite/ref_player/
func (p *PathResolver) DemoFixturesDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "tests", "fixtures", "titles", titleSlug, "ref_player")
}

// --- Chemins globaux (non namespacés par titre) ---

// SessionDir retourne le répertoire des sessions.
// Ex: data/sessions/
func (p *PathResolver) SessionDir() string {
	return filepath.Join(p.repoRoot, "data", "sessions")
}

// JobsCachePath retourne le chemin du cache des jobs.
// Ex: data/cache/jobs.json
func (p *PathResolver) JobsCachePath() string {
	return filepath.Join(p.repoRoot, "data", "cache", "jobs.json")
}

// SyncCacheDir retourne le répertoire racine du cache fetch intermédiaire
// (Phase 2 refactor Collect→Persist). Chaque cycle de sync crée un
// sous-dossier `{cycle_id}/` à l'intérieur. Ex: data/sync_cache/
func (p *PathResolver) SyncCacheDir() string {
	return filepath.Join(p.repoRoot, "data", "sync_cache")
}

// WALDir retourne le répertoire WAL de la BatchQueue persist (Phase 4.7
// closure). Contient les batches JSON pending entre Submit et ACK Persister.
// Recovery au boot via persist.BatchQueue.RecoverPending. Ex: data/wal/
func (p *PathResolver) WALDir() string {
	return filepath.Join(p.repoRoot, "data", "wal")
}

// DBProfilesPath retourne le chemin de db_profiles.json (global).
func (p *PathResolver) DBProfilesPath() string {
	return filepath.Join(p.repoRoot, "db_profiles.json")
}

// AppSettingsPath retourne le chemin de app_settings.json (global).
func (p *PathResolver) AppSettingsPath() string {
	return filepath.Join(p.repoRoot, "app_settings.json")
}

// TitleSettingsPath retourne le chemin de l'overlay de settings d'un titre
// (data/titles/<slug>/settings.json) — miroir per-titre d'AppSettingsPath
// (global), point d'extension du seam PMT-4 « base globale + overlay par titre ».
// Le fichier est OPTIONNEL : absent ⇒ le titre hérite intégralement du global.
// slug vide → titre par défaut.
func (p *PathResolver) TitleSettingsPath(slug string) string {
	if slug == "" {
		slug = DefaultSlug
	}
	return filepath.Join(p.repoRoot, "data", "titles", slug, "settings.json")
}

// WatcherTokensPath retourne le chemin du fichier de tokens watcher.
// Ex: data/auth/watcher_tokens.json
//
// LEGACY (mono-user) : conservé pour le watcher daemon historique.
// Pour le SSO Xbox multi-user, utiliser WatcherTokensDir() + 1 fichier par XUID.
func (p *PathResolver) WatcherTokensPath() string {
	return filepath.Join(p.repoRoot, "data", "auth", "watcher_tokens.json")
}

// WatcherTokensDir retourne le répertoire des tokens watcher multi-user (SSO Xbox).
// Store GLOBAL title-agnostic : les tokens d'auth (refresh-token / XSTS) sont attachés
// au compte Xbox (xuid), PAS à un jeu — un même compte donne accès à tous les titres.
// Layout : data/auth/watcher_tokens/{xuid}.json (1 fichier par utilisateur).
// Décision D4 (cf. SPRINT_XBOX_SSO §0bis) : source unique, write-to-temp + rename atomique,
// perms 0600 sur fichiers / 0700 sur répertoire.
func (p *PathResolver) WatcherTokensDir() string {
	return filepath.Join(p.repoRoot, "data", "auth", "watcher_tokens")
}

// --- Validation ---

// ValidateTitle vérifie qu'un slug est valide dans le registre.
func (p *PathResolver) ValidateTitle(slug string) error {
	if slug == "" {
		return fmt.Errorf("title_slug vide")
	}
	if !p.registry.Exists(slug) {
		return fmt.Errorf("titre inconnu : %q", slug)
	}
	return nil
}
