package title

// demo_layout.go — LA disposition de l'arbre du mode démo (lot B5 du plan backlog du
// 2026-09-26, item 8 : « mode démo hermétique côté fichiers »).
//
// # POURQUOI UN SEUL ENDROIT
//
// Avant ce fichier, la traduction « chemin de production -> chemin de la fixture démo »
// vivait en TROIS copies : `cmd/server/demo_paths.go` (demoWarehouseDBPath),
// `internal/config/player_resolver.go` (demoTitleDir) et
// `internal/ops/seed_demo_multititle.go` (demoTitleSubdir). La règle n°6 de CLAUDE.md
// impose, à la 3e copie, un helper ET un garde-rail : celui-ci est
// `internal/archlint/no_demo_layout_translation_test.go`, qui interdit toute traduction
// hors de ce fichier.
//
// # LA DISPOSITION
//
//   - titre par défaut (ou slug vide) : PLAT sous la racine, byte-identique à la démo
//     mono-titre — `<démo>/warehouse/*.duckdb`, `<démo>/players/<dir>/stats.duckdb` ;
//   - titre additionnel : `<démo>/titles/<slug>/…`, miroir du PathResolver de production ;
//   - configuration de la fixture : `<démo>/db_profiles.json`, `<démo>/app_settings.json`,
//     overlay d'un titre `<TitleDir>/settings.json` ;
//   - `<démo>/auth/` : l'authentification de la démo. Le seed ne l'écrit pas : elle est VIDE,
//     la démo ne se connecte à rien (décision D-7) ;
//   - `<TitleDir>/replays/` : les rejeux FIGÉS de la démo, écrits par le seed et seulement lus
//     par le serveur — `artifacts/{short8}.json`, les films sources sous `films/` (même
//     disposition qu'un cache de films : `film_chunks/`, `film_manifests/`, pour pouvoir les
//     recuire), et `index.json` (matchs servis, correspondance des identités) ;
//   - `<démo>/runtime/` : TOUT ce que le serveur écrit pendant qu'il tourne (sessions, logs,
//     caches, état d'administration). Ignoré par git sous `data/demo/` comme sous
//     `tests/fixtures/demo-root/`.
//
// La racine est une valeur : `DemoLayout{}` (racine vide) rend des chemins relatifs, jamais
// une panique — c'est à l'appelant de ne construire une disposition qu'en mode démo.

import "path/filepath"

// DemoLayout est la disposition de l'arbre démo enraciné en Root.
type DemoLayout struct {
	root string
}

// NewDemoLayout construit la disposition démo enracinée en root (LEVELUP_DEMO_FIXTURES_DIR).
func NewDemoLayout(root string) DemoLayout {
	return DemoLayout{root: root}
}

// TitleDir rend le sous-arbre d'un titre : la racine pour le titre par défaut (PLAT),
// `<démo>/titles/<slug>` pour un titre additionnel.
func (d DemoLayout) TitleDir(slug string) string {
	if IsDefaultSlug(slug) {
		return d.root
	}
	return filepath.Join(d.root, "titles", slug)
}

// WarehouseDir rend le dossier des bases partagées d'un titre.
func (d DemoLayout) WarehouseDir(slug string) string {
	return filepath.Join(d.TitleDir(slug), "warehouse")
}

// WarehouseDBPath rend le chemin démo d'une base du warehouse d'un titre par son NOM DE
// FICHIER (shared_matches_v2.duckdb, metadata.duckdb, …). La traduction est
// INCONDITIONNELLE : elle ne dépend jamais de l'existence d'un fichier, ni de production ni de
// fixture (régression du 2026-08-05, cf. cmd/server/demo_paths_test.go).
func (d DemoLayout) WarehouseDBPath(slug, fileName string) string {
	return filepath.Join(d.WarehouseDir(slug), fileName)
}

// SharedDBPath rend la base des matchs partagés démo d'un titre.
func (d DemoLayout) SharedDBPath(slug string) string {
	return d.WarehouseDBPath(slug, "shared_matches_v2.duckdb")
}

// MetadataDBPath rend la base des référentiels démo d'un titre.
func (d DemoLayout) MetadataDBPath(slug string) string {
	return d.WarehouseDBPath(slug, "metadata.duckdb")
}

// SharedSocialDBPath rend la base sociale démo d'un titre.
func (d DemoLayout) SharedSocialDBPath(slug string) string {
	return d.WarehouseDBPath(slug, "shared_social.duckdb")
}

// SharedPVEDBPath rend la base Firefight démo d'un titre.
func (d DemoLayout) SharedPVEDBPath(slug string) string {
	return d.WarehouseDBPath(slug, "shared_pve.duckdb")
}

// PlayersRootDir rend le dossier des joueurs démo d'un titre.
func (d DemoLayout) PlayersRootDir(slug string) string {
	return filepath.Join(d.TitleDir(slug), playersDirName)
}

// PlayerDir rend le dossier d'un joueur démo (dir = nom de dossier du roster, ex. DEMO).
func (d DemoLayout) PlayerDir(slug, dir string) string {
	return filepath.Join(d.PlayersRootDir(slug), dir)
}

// PlayerDBPath rend la base d'un joueur démo.
func (d DemoLayout) PlayerDBPath(slug, dir string) string {
	return filepath.Join(d.PlayerDir(slug, dir), playerDBFileName)
}

// TitleSettingsPath rend l'overlay de réglages d'un titre dans la fixture (miroir de
// PathResolver.TitleSettingsPath). Fichier optionnel : absent, le titre hérite du global.
func (d DemoLayout) TitleSettingsPath(slug string) string {
	return filepath.Join(d.TitleDir(slug), "settings.json")
}

// ReplaysDir rend le dossier des rejeux figés de la démo pour un titre.
func (d DemoLayout) ReplaysDir(slug string) string {
	return filepath.Join(d.TitleDir(slug), "replays")
}

// ReplayArtifactsDir rend le dossier des artefacts de rejeu servis par la démo.
func (d DemoLayout) ReplayArtifactsDir(slug string) string {
	return filepath.Join(d.ReplaysDir(slug), "artifacts")
}

// ReplayArtifactPath rend l'artefact de rejeu démo d'un match (forme courte, comme en
// production).
func (d DemoLayout) ReplayArtifactPath(slug, matchID string) string {
	return filepath.Join(d.ReplayArtifactsDir(slug), ReplayArtifactFileName(matchID))
}

// ReplayFilmsCacheRoot rend la racine du cache de films EMBARQUÉ par la démo : la même
// disposition qu'un cache de films (film_chunks/, film_manifests/), pour que la recuisson
// lise ces films comme elle lit ceux du dépôt.
func (d DemoLayout) ReplayFilmsCacheRoot(slug string) string {
	return filepath.Join(d.ReplaysDir(slug), "films")
}

// ReplayIndexPath rend l'index des rejeux démo : matchs servis et correspondance des
// identités réelles vers les identités démo (jamais servi tel quel).
func (d DemoLayout) ReplayIndexPath(slug string) string {
	return filepath.Join(d.ReplaysDir(slug), "index.json")
}

// DBProfilesPath rend le db_profiles.json émis par le seed de la démo.
func (d DemoLayout) DBProfilesPath() string {
	return filepath.Join(d.root, "db_profiles.json")
}

// AppSettingsPath rend l'app_settings.json émis par le seed de la démo.
func (d DemoLayout) AppSettingsPath() string {
	return filepath.Join(d.root, "app_settings.json")
}

// AuthDir rend le dossier d'authentification de la démo (users, invites, groupes). Le seed ne
// l'écrit pas : la démo ne se connecte à rien.
func (d DemoLayout) AuthDir() string {
	return filepath.Join(d.root, "auth")
}

// WatcherTokensDir rend le magasin de tokens de la démo, sous AuthDir : vide, donc aucun
// compte, aucun pool, aucune connexion.
func (d DemoLayout) WatcherTokensDir() string {
	return filepath.Join(d.AuthDir(), "watcher_tokens")
}

// RuntimeDir rend le dossier de TOUT ce que le serveur écrit en démo pendant qu'il tourne.
func (d DemoLayout) RuntimeDir() string {
	return filepath.Join(d.root, "runtime")
}

// SessionDir rend le dossier des sessions HTTP de la démo.
func (d DemoLayout) SessionDir() string {
	return filepath.Join(d.RuntimeDir(), "sessions")
}

// LogsDir rend le dossier des logs par module (et du crash log) de la démo.
func (d DemoLayout) LogsDir() string {
	return filepath.Join(d.RuntimeDir(), "logs")
}

// RuntimePaths rend un PathResolver enraciné dans RuntimeDir : les fichiers d'exécution que
// le serveur écrit (caches, état d'administration, amis…) y gardent leur sous-chemin de
// production, mais sous `<démo>/runtime/`. Il ne sert JAMAIS à lire des données de référence
// (config/titles, data/titles/*/reference) : celles-là se lisent à la racine du dépôt.
func (d DemoLayout) RuntimePaths() *PathResolver {
	return NewPathResolver(d.RuntimeDir())
}
