// Package playerdirectory — fs.go : le témoin disque de l'annuaire.
//
// Seule implémentation « réelle » de l'interface FS. Tous les chemins passent
// par PathResolver (CLAUDE.md « Architecture des Données » : jamais de
// `filepath.Join(..., "data", ...)` à la main), et rien n'est ouvert : on ne
// fait que constater l'existence d'un dossier ou d'un fichier. Aucun paquet
// DuckDB n'est importé ici — une player DB se CONSTATE, elle ne s'ouvre pas
// (modèle mono-writer, ADR 0013).
package playerdirectory

import (
	"os"

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
