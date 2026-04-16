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

// NewRegistry crée un registre avec les titres par défaut.
func NewRegistry() *Registry {
	r := &Registry{
		titles: make(map[string]*TitleDescriptor),
	}
	// Halo Infinite est toujours enregistré.
	r.Register(&TitleDescriptor{
		Slug:     DefaultSlug,
		Name:     "Halo Infinite",
		Provider: "halo_infinite",
		Status:   StatusActive,
		Capabilities: []Capability{
			CapMatchmaking, CapFirefight, CapForge,
			CapMedia, CapRanked, CapCareer,
		},
		IsDefault: true,
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

// DBProfilesPath retourne le chemin de db_profiles.json (global).
func (p *PathResolver) DBProfilesPath() string {
	return filepath.Join(p.repoRoot, "db_profiles.json")
}

// AppSettingsPath retourne le chemin de app_settings.json (global).
func (p *PathResolver) AppSettingsPath() string {
	return filepath.Join(p.repoRoot, "app_settings.json")
}

// --- Legacy paths (rétrocompatibilité pré-migration) ---

// LegacyWarehouseDir retourne le chemin legacy warehouse (avant namespace).
// Ex: data/warehouse/
func (p *PathResolver) LegacyWarehouseDir() string {
	return filepath.Join(p.repoRoot, "data", "warehouse")
}

// LegacySharedDBPath retourne le chemin legacy de la base partagée.
func (p *PathResolver) LegacySharedDBPath() string {
	return filepath.Join(p.LegacyWarehouseDir(), "shared_matches_v2.duckdb")
}

// LegacyMetadataDBPath retourne le chemin legacy de la base metadata.
func (p *PathResolver) LegacyMetadataDBPath() string {
	return filepath.Join(p.LegacyWarehouseDir(), "metadata.duckdb")
}

// LegacyPlayerDir retourne le chemin legacy d'un joueur.
func (p *PathResolver) LegacyPlayerDir(gamertag string) string {
	return filepath.Join(p.repoRoot, "data", "players", gamertag)
}

// LegacyDemoFixturesDir retourne le chemin legacy des fixtures démo.
func (p *PathResolver) LegacyDemoFixturesDir() string {
	return filepath.Join(p.repoRoot, "tests", "fixtures", "ref_player")
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
