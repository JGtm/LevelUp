// Tests de la résolution per-titre du provider shared CÔTÉ ÉCRITURE (B-swap).
//
// Symétrie avec config/player_resolver_shared_pertitle_test.go (côté LECTURE) :
// la lecture obtient son SharedReader via cfg.SharedManager.For(path) (caché par
// path dans le Manager) ; l'écriture h5 obtient son provider via
// sharedProviderForPath(cfg, path) → cfg.SharedManager.For(path, tz).
//
// PROPRIÉTÉ VERROUILLÉE ICI : pour le MÊME fichier shared h5, écriture et lecture
// résolvent la MÊME instance de Provider. C'est la condition nécessaire pour que
// le B-swap coordonne RO et RW in-process (sinon DuckDB "different configuration").
// Aucun test ne la verrouillait côté écriture : une régression future (ex. un
// filepath.Abs ajouté d'un seul côté) divergerait silencieusement les deux paths.
//
// Build tag cgo : ouvre un provider DuckDB réel via le Manager.

//go:build cgo

package livesync

import (
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"

	"levelup/go-api/internal/config"

	_ "github.com/duckdb/duckdb-go/v2"
)

// provisionSharedH5 crée le shared_matches_v2.duckdb h5 au path title-aware sous
// repoRoot (mkdir + schéma) puis FERME le handle de setup, de sorte que le Manager
// puisse l'ouvrir lui-même. Réutilise le helper de boot canonique sync.OpenSharedDB
// (mkdir + OpenReadWrite + EnsureSharedSchema) — même chaîne que le serveur réel.
func provisionSharedH5(t *testing.T, repoRoot string) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).SharedDBPath(halo5.TitleSlug)
	db, err := syncpkg.OpenSharedDB(path)
	if err != nil {
		t.Fatalf("OpenSharedDB(h5 shared): %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close setup handle: %v", err)
	}
	return path
}

// TestSharedProviderForPath_ReadEqualsWrite verrouille la propriété read==write :
// sharedProviderForPath (chemin écriture) retourne EXACTEMENT le même Provider que
// cfg.SharedManager.For(path) (ce que la lecture via sharedReaderForTitle obtient).
func TestSharedProviderForPath_ReadEqualsWrite(t *testing.T) {
	repoRoot := t.TempDir()
	sharedPath := provisionSharedH5(t, repoRoot)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	cfg := &config.AppConfig{
		RepoRoot:      repoRoot,
		SharedManager: mgr,
		UserTimezone:  "Europe/Paris",
	}

	// Côté écriture : le provider per-titre résolu par le câblage live-sync h5.
	writeProvider := sharedProviderForPath(cfg, sharedPath)
	if writeProvider == nil {
		t.Fatalf("sharedProviderForPath doit être non-nil quand le shared h5 existe")
	}

	// Côté lecture : exactement l'appel que fait config.sharedReaderForTitle —
	// For(path) caché par path dans le MÊME Manager.
	readProvider, err := cfg.SharedManager.For(sharedPath, cfg.UserTimezone)
	if err != nil {
		t.Fatalf("Manager.For(sharedPath): %v", err)
	}

	// La propriété centrale : read et write pointent vers la MÊME instance. Si un
	// seul des deux côtés normalisait le path différemment (Abs, Clean, séparateur),
	// le Manager ouvrirait deux Provider distincts → B-swap non coordonné.
	if writeProvider != readProvider {
		t.Errorf("écriture et lecture doivent résoudre le MÊME provider pour %q "+
			"(read==write requis pour le B-swap)", sharedPath)
	}

	// Idempotence : un 2e appel écriture rend toujours la même instance (caché).
	if again := sharedProviderForPath(cfg, sharedPath); again != writeProvider {
		t.Errorf("sharedProviderForPath non idempotent : 2 instances pour le même path")
	}
}

// TestSharedProviderForPath_NilManager : sans Manager (mode legacy / kill-switch
// LEVELUP_USE_SHARED_PROVIDER=0), le chemin écriture retourne nil → fallback legacy.
func TestSharedProviderForPath_NilManager(t *testing.T) {
	repoRoot := t.TempDir()
	sharedPath := provisionSharedH5(t, repoRoot)

	cfg := &config.AppConfig{RepoRoot: repoRoot} // SharedManager == nil
	if p := sharedProviderForPath(cfg, sharedPath); p != nil {
		t.Errorf("SharedManager nil : sharedProviderForPath doit être nil (fallback legacy), got %v", p)
	}
}
