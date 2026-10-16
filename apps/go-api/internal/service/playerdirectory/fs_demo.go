// Package playerdirectory — fs_demo.go : le témoin disque de la DÉMO.
//
// La démo n'a pas le témoin du dépôt ([NewPathFS]) : il balaierait les dossiers de joueurs
// réels du poste. Elle a le sien, adossé à SA disposition ([title.DemoLayout]) : la fixture
// nomme ses dossiers autrement que les clés de profil (demo-player -> DEMO), la
// correspondance vient du roster démo ([DemoPlayerDir]). Un dossier de la fixture que le
// roster ne nomme pas est un orphelin, comme sur le disque d'une instance réelle.
//
// Lecture seule : la démo ne purge rien ([RemovePlayerDir] refuse), et rien n'est jamais ouvert.
package playerdirectory

import (
	"errors"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// DemoPlayerDir apparie la clé d'un profil démo et le dossier de sa fixture.
type DemoPlayerDir struct {
	Key string // clé de profil (player_slug), ex. demo-player
	Dir string // dossier de la fixture, ex. DEMO
}

// errDemoReadOnly : la purge d'un dossier n'a pas de sens sur la fixture publiée.
var errDemoReadOnly = errors.New("playerdirectory: demo fixture is read-only")

// demoFS lit la disposition de la démo à travers le roster démo.
type demoFS struct {
	layout title.DemoLayout
	dirOf  map[string]string // clé de profil (minuscules) -> dossier
	keyOf  map[string]string // dossier (minuscules) -> clé de profil
}

// NewDemoFS construit le témoin disque de la démo pour sa disposition et son roster.
func NewDemoFS(layout title.DemoLayout, roster []DemoPlayerDir) FS {
	f := &demoFS{layout: layout, dirOf: make(map[string]string, len(roster)), keyOf: make(map[string]string, len(roster))}
	for _, r := range roster {
		f.dirOf[strings.ToLower(r.Key)] = r.Dir
		f.keyOf[strings.ToLower(r.Dir)] = r.Key
	}
	return f
}

func (f *demoFS) dir(key string) (string, bool) {
	d, ok := f.dirOf[strings.ToLower(key)]
	return d, ok && key != ""
}

// PlayerDirExists : le dossier de la fixture du joueur de cette clé existe-t-il ?
func (f *demoFS) PlayerDirExists(titleSlug, key string) bool {
	d, ok := f.dir(key)
	if !ok {
		return false
	}
	info, err := os.Stat(f.layout.PlayerDir(titleSlug, d))
	return err == nil && info.IsDir()
}

// PlayerDBPath rend le chemin de la player DB démo du joueur, "" si la clé n'est pas au roster.
func (f *demoFS) PlayerDBPath(titleSlug, key string) string {
	d, ok := f.dir(key)
	if !ok {
		return ""
	}
	return f.layout.PlayerDBPath(titleSlug, d)
}

// PlayerDBExists : la player DB démo du joueur existe-t-elle ?
func (f *demoFS) PlayerDBExists(titleSlug, key string) bool {
	path := f.PlayerDBPath(titleSlug, key)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ListPlayerDirs rend les dossiers de la fixture pour ce titre ; ceux du roster sous leur clé de
// profil (que l'annuaire compare aux clés de profil), les autres sous leur nom de dossier.
func (f *demoFS) ListPlayerDirs(titleSlug string) ([]string, error) {
	entries, err := os.ReadDir(f.layout.PlayersRootDir(titleSlug))
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
		if key, ok := f.keyOf[strings.ToLower(e.Name())]; ok {
			names = append(names, key)
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// RemovePlayerDir refuse toujours : la démo est en lecture seule.
func (f *demoFS) RemovePlayerDir(string, string) error { return errDemoReadOnly }
