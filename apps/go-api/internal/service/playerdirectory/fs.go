// Package playerdirectory — fs.go : le témoin disque de l'annuaire.
//
// Seule implémentation « réelle » de l'interface FS. Tous les chemins passent
// par PathResolver (CLAUDE.md « Architecture des Données » : jamais de
// `filepath.Join(..., "data", ...)` à la main), et rien n'est jamais OUVERT :
// on constate l'existence d'un dossier ou d'un fichier, et — à la purge d'une
// identité seulement — on retire un dossier joueur. Aucun paquet DuckDB n'est
// importé ici (ratchet `no_duckdb_import_playerdirectory_test.go`) : une player
// DB se CONSTATE et se SUPPRIME, elle ne s'ouvre pas (modèle mono-writer,
// ADR 0013).
package playerdirectory

import (
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// pathFS lit le disque à travers le PathResolver d'un dépôt.
type pathFS struct {
	paths *title.PathResolver
}

// NewPathFS construit le témoin disque pour la racine du dépôt donnée.
func NewPathFS(repoRoot string) FS {
	return &pathFS{paths: title.NewPathResolver(repoRoot)}
}

// PlayerDirExists : le dossier du joueur existe-t-il pour ce titre ?
func (f *pathFS) PlayerDirExists(titleSlug, key string) bool {
	if key == "" {
		return false
	}
	info, err := os.Stat(f.paths.PlayerDir(titleSlug, key))
	return err == nil && info.IsDir()
}

// PlayerDBPath rend le chemin de la player DB du joueur pour ce titre, qu'elle
// existe ou non : c'est le chemin QUE LA SYNC utilisera. Rendu par Onboard pour
// que l'appelant sache où le joueur écrira, sans jamais ouvrir le fichier.
func (f *pathFS) PlayerDBPath(titleSlug, key string) string {
	if key == "" {
		return ""
	}
	return f.paths.PlayerDBPath(titleSlug, key)
}

// PlayerDBExists : la player DB du joueur existe-t-elle pour ce titre ?
func (f *pathFS) PlayerDBExists(titleSlug, key string) bool {
	if key == "" {
		return false
	}
	info, err := os.Stat(f.paths.PlayerDBPath(titleSlug, key))
	return err == nil && !info.IsDir()
}

// ListPlayerDirs rend les noms des dossiers joueur présents pour ce titre.
// Racine absente = titre jamais synchronisé : ce n'est pas une erreur, c'est
// une liste vide.
func (f *pathFS) ListPlayerDirs(titleSlug string) ([]string, error) {
	entries, err := os.ReadDir(f.paths.PlayersRootDir(titleSlug))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// RemovePlayerDir supprime le dossier joueur `name` du titre donné, avec tout ce
// qu'il contient. Utilisé par la purge d'identité pour les dossiers ORPHELINS —
// ceux qu'aucun profil ne déclare (les dossiers de profils, eux, partent avec
// leur entrée, via ProfileService.PurgeIdentityData).
//
// Le chemin vient de PathResolver et `name` doit en être la dernière composante
// et rien d'autre : un nom vide, `.`/`..` ou porteur d'un séparateur est REFUSÉ.
// `PathResolver.PlayerDir` ne fait qu'un `filepath.Join`, qui nettoie un `..` en
// remontant d'un cran — c'est-à-dire hors de la racine des joueurs. Les noms
// traités viennent tous d'un `os.ReadDir`, donc aucun ne devrait l'être ; la
// garde existe pour que cela reste vrai si un appelant change. Dossier déjà
// absent = succès (la purge est idempotente).
func (f *pathFS) RemovePlayerDir(titleSlug, name string) error {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("playerdirectory: nom de dossier joueur invalide: %q", name)
	}
	return os.RemoveAll(f.paths.PlayerDir(titleSlug, name))
}
