// Package logging — observabilité multi-module pour LevelUp.
//
// Fournit un slog.Handler qui dispatche les logs vers :
//   - La sortie console (texte ou JSON, comportement pre-sprint)
//   - Un fichier par module sous `logs/{module}.log` (JSON, pour parsing facile)
//
// Le module d'un log est déterminé par (ordre de priorité) :
//  1. Attribut explicit `module=` sur le record (slog.With("module", "sync"))
//  2. Détection automatique depuis le PC d'appel (package path → nom court)
//  3. Fallback "general" pour les logs non catégorisés
//
// Les modules courants sont déclarés en constantes (cf. Modules ci-dessous)
// pour éviter les typos. Anticipation de l'ajout de nouveaux modules : la
// constante n'est PAS exclusive — un module ad hoc peut être injecté via
// slog.With("module", "mon_nouveau_module") et un fichier sera créé lazy.
//
// Référençabilité cross-module : utiliser logging.WithEventID(ctx, prefix)
// au début d'une opération multi-étape (RunDelta, swap RW, backfill HTTP) ;
// l'event_id est ensuite ajouté à chaque log émis depuis ce ctx (le
// ContextHandler s'en charge), permettant de grep `event_id=...` à travers
// les fichiers logs/{sync,provider,pool,...}.log.
package logging

import (
	"runtime"
	"strings"
	"sync"
)

// Modules connus de l'application. Les fichiers `logs/{module}.log` sont
// créés lazy à la première écriture. Cette liste est indicative — un module
// inconnu se voit créer son propre fichier sans configuration préalable.
const (
	ModuleSync      = "sync"      // sync engine, RunDelta, RunBackfill
	ModuleProvider  = "provider"  // SharedDBProvider (swap RO↔RW, drain, retry)
	ModulePool      = "pool"      // PlayerDB pool (openPlayerDB, swap hooks)
	ModuleScheduler = "scheduler" // auto_sync scheduler
	ModuleHTTP      = "http"      // chi router, middlewares, handlers
	ModuleHandlers  = "handlers"  // endpoints HTTP individuels
	ModuleService   = "service"   // couche service (FriendsOrchestrator, etc.)
	ModuleDuckDB    = "duckdb"    // primitives DuckDB (OpenReadOnly, dblease)
	ModuleAuth      = "auth"      // tokens MSAL, refresh, XSTS
	ModuleAssets    = "assets"    // resolver assets + cache disque
	ModulePrestige  = "prestige"  // challenges + squad prestige
	ModuleMedia     = "media"     // indexation médias + galerie
	ModuleNotif     = "notifications"
	ModuleMigration = "migration" // schémas DB
	ModuleHealth    = "health"    // data_health scheduler
	ModulePersist   = "persist"   // refactor Collect→Persist : queue, workers, persisters
	ModuleBackup    = "backup"    // pkg/duckdbbackup : scheduler restic + exporter
	// ModuleLeaderboard : classement CSR mondial — scraper Halo Waypoint +
	// job snapshot + lecture (tag explicite `module=leaderboard` car le code
	// vit dans les packages halo/duckdb qui ont leur propre module par défaut).
	ModuleLeaderboard = "leaderboard"
	// ModuleCatalog : cron de rafraîchissement du catalogue (noms localisés des
	// playlists/couples/maps/modes + expansion des enfants + poids). Tag explicite
	// car le code vit dans scheduler/api/main qui ont chacun leur module par défaut —
	// on veut un fichier logs/catalog.log unique pour diagnostiquer le cron.
	ModuleCatalog = "catalog"
	// ModuleLab : outil opérateur « Lab » — explorateur d'API
	// Discovery UGC live. Tag explicite car le code vit dans api/server.go
	// (module par défaut "http") ; on veut un fichier logs/lab.log dédié pour
	// diagnostiquer les appels d'exploration Waypoint.
	ModuleLab = "lab"
	// ModuleSnapshot : producteur + lecture des snapshots Parquet immuables
	// (durabilité / lecture découplée du B-swap). Tag explicite car le code vit dans
	// internal/ops (module "general" par défaut) + internal/sync + internal/sync/v2 —
	// on veut un fichier logs/snapshot.log unique pour diagnostiquer cut/fallback/readiness.
	ModuleSnapshot = "snapshot"
	ModuleGeneral  = "general" // fallback pour logs non catégorisés
)

// moduleAttrKey est la clé d'attribut slog reconnue pour spécifier
// manuellement le module ; prend précédence sur la détection auto.
const moduleAttrKey = "module"

// detectModuleFromCaller infère un nom de module à partir du PC d'appel.
// Stratégie : prendre le dernier segment significatif du package path Go.
//
// Exemples :
//   - "levelup/go-api/internal/sync"                              → "sync"
//   - "levelup/go-api/internal/platform/duckdb"                   → "duckdb"
//   - "levelup/go-api/internal/platform/duckdb/sharedprovider"    → "provider"
//     (cas spécial : le nom du package est "sharedprovider" mais l'attribut
//     module canonique est "provider")
//   - "levelup/go-api/internal/api/handlers"                      → "handlers"
//   - "levelup/go-api/internal/api"                               → "http"
//   - "levelup/go-api/internal/scheduler"                         → "scheduler"
//
// Le cache pcToModule évite le coût de runtime.FuncForPC sur chaque log.
func detectModuleFromCaller(pc uintptr) string {
	if v, ok := pcToModule.Load(pc); ok {
		return v.(string)
	}
	module := computeModule(pc)
	pcToModule.Store(pc, module)
	return module
}

var pcToModule sync.Map // map[uintptr]string

// computeModule fait le travail effectif de résolution PC → module name.
func computeModule(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ModuleGeneral
	}
	full := fn.Name() // ex: "levelup/go-api/internal/sync.(*SyncEngine).run"
	// Stripper la fonction : garder la partie package.
	if idx := strings.LastIndex(full, "/"); idx >= 0 {
		// Après le dernier `/`, on a typiquement "sync.(*SyncEngine).run"
		// ou "duckdb.OpenReadOnly". On veut le nom du package.
		afterSlash := full[idx+1:]
		if dotIdx := strings.Index(afterSlash, "."); dotIdx >= 0 {
			pkg := afterSlash[:dotIdx]
			return mapPackageToModule(pkg, full)
		}
	}
	return ModuleGeneral
}

// packageToModuleMap : table de mapping package → module canonique.
// Construite à l'initialisation pour éviter une cascade de switch case
// (réduit la complexité cyclomatique de mapPackageToModule).
var packageToModuleMap = map[string]string{
	ModuleSync:       ModuleSync,
	"sharedprovider": ModuleProvider,
	"duckdb":         ModuleDuckDB,
	"dblease":        ModuleDuckDB,
	"pool":           ModulePool,
	"scheduler":      ModuleScheduler,
	"handlers":       ModuleHandlers,
	"service":        ModuleService,
	"auth":           ModuleAuth,
	"msalauth":       ModuleAuth,
	"assets":         ModuleAssets,
	"prestige":       ModulePrestige,
	"campaign":       ModulePrestige,
	"media":          ModuleMedia,
	"mediaservice":   ModuleMedia,
	"notifications":  ModuleNotif,
	"migration":      ModuleMigration,
	"persist":        ModulePersist,
	"api":            ModuleHTTP,
	"middleware":     ModuleHTTP, // auth/ownership/CSRF/rate-limit → logs/http.log
	"watcher":        ModuleAuth, // tokens watcher fait partie de l'auth flow
	"rta":            ModuleAuth,
	"duckdbbackup":   ModuleBackup,  // pkg/duckdbbackup → logs/backup.log
	"main":           ModuleGeneral, // cmd/server/main.go : boot/shutdown
}

// mapPackageToModule fait le mapping nom-de-package → nom-de-module canonique.
// Plusieurs packages peuvent se grouper sous un seul module (e.g. sharedprovider
// + duckdb → "duckdb" pour les opérations bas-niveau, "provider" pour le swap).
func mapPackageToModule(pkg, fullPath string) string {
	if m, ok := packageToModuleMap[pkg]; ok {
		return m
	}
	return moduleFromPath(fullPath)
}

// moduleFromPath applique l'heuristique secondaire sur le fullPath.
func moduleFromPath(fullPath string) string {
	switch {
	case strings.Contains(fullPath, "observability"):
		return ModuleHealth
	case strings.Contains(fullPath, "data_health"), strings.Contains(fullPath, "health"):
		return ModuleHealth
	case strings.Contains(fullPath, "/auth/"), strings.Contains(fullPath, "/platform/auth"):
		return ModuleAuth
	default:
		return ModuleGeneral
	}
}

// SanitizeModuleName normalise un nom de module pour usage en nom de fichier
// (lowercase, sans caractères dangereux). Garantit qu'on peut faire
// `os.OpenFile(filepath.Join(dir, module+".log"), ...)` sans risque.
func SanitizeModuleName(name string) string {
	if name == "" {
		return ModuleGeneral
	}
	out := strings.Builder{}
	out.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r + ('a' - 'A'))
		default:
			out.WriteRune('_')
		}
	}
	result := out.String()
	if result == "" {
		return ModuleGeneral
	}
	return result
}
