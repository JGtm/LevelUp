// Package config — feature flags de bascule progressive Go/Python (Sprint 27).
//
// Principe :
//   - Par défaut toutes les surfaces sont sur le backend Go (migration terminée).
//   - En cas d'incident, rollback immédiat via env var LEVELUP_FF_<SURFACE>=python.
//   - Possibilité de persister un rollback dans app_settings.json, clé "feature_flags".
//
// Ordre de priorité (croissant) :
//  1. Défauts hardcodés (tout Go)
//  2. app_settings.json → clé "feature_flags" (objet JSON surface:backend)
//  3. Variables d'environnement LEVELUP_FF_<SURFACE>=go|python
package config

import (
	"encoding/json"
	"os"
	"strings"
)

// Backend identifie quel stack traite une surface fonctionnelle.
type Backend string

const (
	// BackendGo indique que la surface est traitée par le backend Go.
	BackendGo Backend = "go"
	// BackendPython indique un rollback vers le backend Python.
	BackendPython Backend = "python"
)

// Surface identifie une surface fonctionnelle de l'application.
type Surface string

const (
	SurfaceCareer    Surface = "career"
	SurfaceHistory   Surface = "history"
	SurfaceExplorer  Surface = "explorer"
	SurfaceMatchView Surface = "match_view"
	SurfaceStats     Surface = "stats"
	SurfaceSquad     Surface = "squad"
	SurfaceHome      Surface = "home"
	SurfaceAuth      Surface = "auth"
	SurfaceSettings  Surface = "settings"
	SurfaceJobs      Surface = "jobs"
	SurfaceSync      Surface = "sync"
	SurfaceBackfill  Surface = "backfill"
)

// AllSurfaces liste toutes les surfaces dans l'ordre de bascule recommandé.
var AllSurfaces = []Surface{
	SurfaceCareer,
	SurfaceHistory,
	SurfaceExplorer,
	SurfaceMatchView,
	SurfaceStats,
	SurfaceSquad,
	SurfaceHome,
	SurfaceAuth,
	SurfaceSettings,
	SurfaceJobs,
	SurfaceSync,
	SurfaceBackfill,
}

// FeatureFlags contrôle quel backend traite chaque surface.
// Tous les champs sont initialisés à BackendGo par defaultFeatureFlags().
type FeatureFlags struct {
	Career    Backend `json:"career"`
	History   Backend `json:"history"`
	Explorer  Backend `json:"explorer"`
	MatchView Backend `json:"match_view"`
	Stats     Backend `json:"stats"`
	Squad     Backend `json:"squad"`
	Home      Backend `json:"home"`
	Auth      Backend `json:"auth"`
	Settings  Backend `json:"settings"`
	Jobs      Backend `json:"jobs"`
	Sync      Backend `json:"sync"`
	Backfill  Backend `json:"backfill"`
}

// defaultFeatureFlags retourne les flags avec tout sur Go (état post-migration).
func defaultFeatureFlags() FeatureFlags {
	return FeatureFlags{
		Career:    BackendGo,
		History:   BackendGo,
		Explorer:  BackendGo,
		MatchView: BackendGo,
		Stats:     BackendGo,
		Squad:     BackendGo,
		Home:      BackendGo,
		Auth:      BackendGo,
		Settings:  BackendGo,
		Jobs:      BackendGo,
		Sync:      BackendGo,
		Backfill:  BackendGo,
	}
}

// surfaceFields retourne un mapping Surface → pointeur sur le champ Backend correspondant.
// Point de vérité unique : évite le double switch entre BackendFor et applyFlagsMap.
func (ff *FeatureFlags) surfaceFields() map[Surface]*Backend {
	return map[Surface]*Backend{
		SurfaceCareer:    &ff.Career,
		SurfaceHistory:   &ff.History,
		SurfaceExplorer:  &ff.Explorer,
		SurfaceMatchView: &ff.MatchView,
		SurfaceStats:     &ff.Stats,
		SurfaceSquad:     &ff.Squad,
		SurfaceHome:      &ff.Home,
		SurfaceAuth:      &ff.Auth,
		SurfaceSettings:  &ff.Settings,
		SurfaceJobs:      &ff.Jobs,
		SurfaceSync:      &ff.Sync,
		SurfaceBackfill:  &ff.Backfill,
	}
}

// BackendFor retourne le backend associé à une surface.
func (ff *FeatureFlags) BackendFor(s Surface) Backend {
	if ptr, ok := ff.surfaceFields()[s]; ok {
		return *ptr
	}
	return BackendGo
}

// AllOnGo retourne true si toutes les surfaces sont sur le backend Go.
func (ff *FeatureFlags) AllOnGo() bool {
	for _, s := range AllSurfaces {
		if ff.BackendFor(s) != BackendGo {
			return false
		}
	}
	return true
}

// LoadFeatureFlags charge les feature flags depuis app_settings.json puis env vars.
// Ne retourne jamais d'erreur : en cas de problème de lecture, les défauts s'appliquent.
func LoadFeatureFlags(appSettingsPath string) FeatureFlags {
	ff := defaultFeatureFlags()
	parseAppSettingsFlags(appSettingsPath, &ff)
	applyEnvFlags(&ff)
	return ff
}

// parseAppSettingsFlags lit la clé "feature_flags" de app_settings.json si présente.
func parseAppSettingsFlags(path string, ff *FeatureFlags) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw struct {
		FeatureFlags map[string]string `json:"feature_flags"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || len(raw.FeatureFlags) == 0 {
		return
	}
	applyFlagsMap(ff, raw.FeatureFlags)
}

// applyEnvFlags applique les variables d'environnement LEVELUP_FF_<SURFACE>=go|python.
func applyEnvFlags(ff *FeatureFlags) {
	env := make(map[string]string, len(AllSurfaces))
	for _, s := range AllSurfaces {
		key := "LEVELUP_FF_" + strings.ToUpper(string(s))
		if val, ok := os.LookupEnv(key); ok {
			env[string(s)] = strings.ToLower(val)
		}
	}
	applyFlagsMap(ff, env)
}

// applyFlagsMap met à jour ff depuis une map surface→backend.
func applyFlagsMap(ff *FeatureFlags, m map[string]string) {
	fields := ff.surfaceFields()
	for surface, val := range m {
		if ptr, ok := fields[Surface(surface)]; ok {
			*ptr = parseBackend(val)
		}
	}
}

// parseBackend convertit une string en Backend (défaut BackendGo si valeur inconnue).
func parseBackend(s string) Backend {
	if strings.ToLower(s) == string(BackendPython) {
		return BackendPython
	}
	return BackendGo
}
