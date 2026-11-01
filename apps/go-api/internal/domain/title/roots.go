package title

// roots.go — les TROIS RACINES du programme (ADR 0008, amendement « two roots »).
//
//   - Resources : ce qui accompagne le programme, en lecture seule par convention (config/,
//     static/, build web, référentiel du rejeu data/titles/{slug}/reference/, metadata-prebuilt.zip).
//   - Data : tout ce que le programme crée ou modifie (bases, sessions, auth, caches, sauvegardes,
//     overlays générés).
//   - Logs : les journaux.
//
// Par défaut Data vaut <Resources>/data et Logs <Resources>/logs : c'est la disposition d'un
// checkout, byte-identique à celle d'avant les trois racines. La résolution depuis
// l'environnement se fait UNE fois, dans internal/config (qui importe ce paquet, pas l'inverse) ;
// ce paquet ne lit aucune variable d'environnement.

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
)

// Roots réunit les trois racines résolues.
type Roots struct {
	Resources string
	Data      string
	Logs      string
	// DataSet : la racine de données a été posée explicitement (LEVELUP_DATA_DIR). Seul effet :
	// db_profiles.json, app_settings.json et .env.local vivent en tête de Data au lieu de la
	// tête de Resources (amendement A-3).
	DataSet bool
}

// DefaultRoots rend les racines d'un checkout enraciné en resources.
func DefaultRoots(resources string) Roots {
	return Roots{
		Resources: resources,
		Data:      filepath.Join(resources, "data"),
		Logs:      filepath.Join(resources, "logs"),
	}
}

// StateFilesDir rend le dossier de db_profiles.json, app_settings.json et .env.local : la tête
// de Data quand la racine de données est explicite, la tête de Resources sinon.
func (r Roots) StateFilesDir() string {
	if r.DataSet {
		return r.Data
	}
	return r.Resources
}

// withDefaults complète les racines vides à partir de Resources.
func (r Roots) withDefaults() Roots {
	def := DefaultRoots(r.Resources)
	if r.Data == "" {
		r.Data = def.Data
	}
	if r.Logs == "" {
		r.Logs = def.Logs
	}
	return r
}

// SameRoot dit si a et b désignent le même dossier (comparaison lexicale après nettoyage,
// insensible à la casse sous Windows).
func SameRoot(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// NewPathResolverRoots crée un PathResolver à racines explicites. Les racines Data et Logs
// vides se complètent à partir de Resources.
func NewPathResolverRoots(roots Roots, registry ...*Registry) *PathResolver {
	return newPathResolver(roots.withDefaults(), registry)
}

// Roots rend les trois racines du résolveur.
func (p *PathResolver) Roots() Roots { return p.roots }

// ResourcesRoot rend la racine des ressources livrées avec le programme.
func (p *PathResolver) ResourcesRoot() string { return p.roots.Resources }

// DataRoot rend la racine des données écrites par le programme.
func (p *PathResolver) DataRoot() string { return p.roots.Data }

// LogsRoot rend la racine des journaux.
func (p *PathResolver) LogsRoot() string { return p.roots.Logs }

// processRoots : les racines du processus, installées par internal/config à sa résolution.
//
// PONT TRANSITOIRE. Une centaine de services reçoivent une racine nue (`repoRoot string`) et
// bâtissent eux-mêmes leur PathResolver ; tant qu'ils ne reçoivent pas le résolveur, une
// racine de données distincte ne leur parviendrait pas. NewPathResolver(racine) rend donc les
// racines du processus quand `racine` est la racine de ressources installée, et DefaultRoots
// sinon. Retrait : quand l'allowlist du garde-rail archlint/repo_root_readers_test.go est vide.
var processRoots atomic.Pointer[Roots]

// InstallProcessRoots enregistre les racines du processus (appelé par internal/config).
func InstallProcessRoots(r Roots) {
	r = r.withDefaults()
	processRoots.Store(&r)
}

// ResetProcessRoots retire les racines du processus (tests).
func ResetProcessRoots() { processRoots.Store(nil) }

// rootsFor rend les racines d'un NewPathResolver(repoRoot) : celles du processus si repoRoot en
// est la racine de ressources, la disposition par défaut sinon.
func rootsFor(repoRoot string) Roots {
	if p := processRoots.Load(); p != nil && SameRoot(p.Resources, repoRoot) {
		return *p
	}
	return DefaultRoots(repoRoot)
}
