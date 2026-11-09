package parquetbackup

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/userstore"
)

// fileUsers : le nom du fichier des comptes, tel que le magasin des comptes le nomme (jamais écrit ici).
var fileUsers = filepath.Base(userstore.FilePathIn("x"))

// Chemins relatifs des fichiers de configuration dans un jeu (sous `config/`).
const (
	relDBProfiles  = "db_profiles.json"
	relAppSettings = "app_settings.json"
	relAuthDir     = "auth"
	relTokensDir   = "auth/watcher_tokens"
	fileGroups     = "groups.json"
)

// configSource : un fichier de configuration à copier, sa place dans le jeu et son chemin en production.
type configSource struct {
	rel string
	src string
}

// configSources liste les fichiers de configuration et d'authentification à joindre au jeu : les profils
// suivis, les réglages, les comptes, les groupes et les fichiers de jetons (chiffrés par le coffre du
// système quand le profil en a un : copiés tels quels, jamais déchiffrés). Un fichier absent est omis.
func configSources(paths *title.PathResolver, authDir string) []configSource {
	candidates := []configSource{
		{relDBProfiles, paths.DBProfilesPath()},
		{relAppSettings, paths.AppSettingsPath()},
		{relAuthDir + "/" + fileUsers, userstore.FilePathIn(authDir)},
		{relAuthDir + "/" + fileGroups, filepath.Join(authDir, fileGroups)},
	}
	if entries, err := os.ReadDir(paths.WatcherTokensDir()); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			candidates = append(candidates, configSource{
				rel: relTokensDir + "/" + e.Name(),
				src: filepath.Join(paths.WatcherTokensDir(), e.Name()),
			})
		}
	}
	var out []configSource
	for _, c := range candidates {
		if st, err := os.Stat(c.src); err == nil && !st.IsDir() {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

// copyConfigFiles copie les fichiers de configuration dans <setDir>/config et les mesure.
func copyConfigFiles(setDir string, sources []configSource) ([]FileEntry, error) {
	entries := make([]FileEntry, 0, len(sources))
	for _, s := range sources {
		rel := configDirName + "/" + s.rel
		if err := copyFile(s.src, filepath.Join(setDir, filepath.FromSlash(rel))); err != nil {
			return nil, fmt.Errorf("copie de %s : %w", s.rel, err)
		}
		e, err := fileEntryOf(setDir, rel)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// copyFile copie src vers dst (dossier créé), en 0600 : ces fichiers portent des comptes et des jetons.
func copyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // chemin issu du résolveur de chemins
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // dst construit sous le jeu
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// configDestination rend le chemin de restauration d'un fichier de configuration du jeu (chemin relatif
// au jeu, sous `config/`), ou false pour un fichier inconnu : seuls les noms que [configSources] produit
// sont restaurés, un manifeste ne peut pas désigner un chemin arbitraire.
func configDestination(paths *title.PathResolver, authDir, rel string) (string, bool) {
	rel = strings.TrimPrefix(rel, configDirName+"/")
	switch {
	case rel == relDBProfiles:
		return paths.DBProfilesPath(), true
	case rel == relAppSettings:
		return paths.AppSettingsPath(), true
	case rel == relAuthDir+"/"+fileUsers, rel == relAuthDir+"/"+fileGroups:
		return filepath.Join(authDir, path.Base(rel)), true
	case path.Dir(rel) == relTokensDir && strings.HasSuffix(rel, ".json") && path.Base(rel) == filepath.Base(rel):
		return filepath.Join(paths.WatcherTokensDir(), path.Base(rel)), true
	}
	return "", false
}
