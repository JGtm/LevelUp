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
	// FilmArchive : dossier de l'archive des films (setting `film_archive_dir`, lot L14), chemin
	// absolu ; vide = <Data>/cache, la disposition d'avant le réglage. Cf. FilmArchiveDir.
	FilmArchive string
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

// FilmArchiveDir rend la racine de l'ARCHIVE DES FILMS : le dossier qui contient `film_chunks/` ET
// `film_manifests/` (l'unité d'archive est le film, ADR 0045 D-12). Le réglage `film_archive_dir`
// la pose ; sans lui c'est la racine du cache applicatif (<Data>/cache), exactement le chemin
// d'avant le réglage. Les faits décodés (`film_facts`) et les artefacts de rejeu restent sous
// CacheRootDir : ils se refont du film, l'archive ne se refait pas.
//
// Le résolveur ne sait rien de `libs/film/filmcache` : la disposition interne de l'archive (les
// deux sous-dossiers) n'est déclarée que là, et cette méthode ne rend que sa racine.
func (p *PathResolver) FilmArchiveDir() string { return p.roots.FilmArchiveDir() }

// FilmArchiveDir rend la racine de l'archive des films de ces racines (cf. la méthode du résolveur).
func (r Roots) FilmArchiveDir() string {
	if r.FilmArchive != "" {
		return r.FilmArchive
	}
	return r.DefaultFilmArchiveDir()
}

// DefaultFilmArchiveDir rend l'emplacement de l'archive sans réglage : la racine du cache
// applicatif, là où `film_chunks/` et `film_manifests/` ont toujours vécu.
func (r Roots) DefaultFilmArchiveDir() string { return filepath.Join(r.Data, "cache") }

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

// --- Ressources et overlays : où vit chaque famille (amendement A-2) ---

// referenceDir rend le dossier de RÉFÉRENCE d'un titre (catalogues de cartes, profils de film,
// énergie des équipements) : une ressource livrée avec le programme, lue seulement. Sa forme
// relative `data/titles/{slug}/reference` est conservée sous la racine de ressources pour
// qu'un arbre livré et un checkout se ressemblent ; il n'est PAS sous Data.
func (p *PathResolver) referenceDir(titleSlug string) string {
	return filepath.Join(p.roots.Resources, "data", "titles", titleSlug, "reference")
}

// generatedReferenceDir rend le dossier des overlays que le RUNTIME écrit à côté du référentiel
// (cartes rattrapées au fetch d'un film) : de la donnée, donc sous Data, même s'il porte la
// forme `reference/generated` du référentiel qu'il complète.
func (p *PathResolver) generatedReferenceDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "generated")
}

// PrebuiltMetadataZipPath rend l'archive de la base metadata pré-construite d'un titre, livrée
// avec le programme et extraite vers MetadataDBPath au premier démarrage. Ressource.
// Ex: data/titles/halo_infinite/warehouse/metadata-prebuilt.zip (sous la racine de ressources)
func (p *PathResolver) PrebuiltMetadataZipPath(titleSlug string) string {
	return filepath.Join(p.roots.Resources, "data", "titles", titleSlug, "warehouse", "metadata-prebuilt.zip")
}
