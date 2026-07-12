// Test de non-régression du fix « known-set indisponible / different configuration ».
//
// loadKnownMatchIDs et loadXUIDAliasesSeed ne font que des SELECT : ils doivent lire
// via provider.Get (RO), JAMAIS via un swap RW. Le bug prod : ces lectures forçaient
// un AcquireWriter (swap RO→RW→RO) ; avec 4 joueurs h5 en parallèle + des lecteurs
// HTTP sur le shared h5, l'OpenReadWrite échouait (« Can't open a connection to same
// database file with a different configuration than existing connections »), le delta
// tombait (« collecte sans delta »).
//
// PROPRIÉTÉ VERROUILLÉE ICI : un lecteur RO concurrent tenu ouvert n'empêche PAS le
// known-set de réussir, et le provider reste en StateRO (aucun swap déclenché). Avec
// l'ancien code (AcquireWriter), le waitForDrain aurait attendu ce lecteur jusqu'au
// drain timeout puis échoué.
//
//go:build cgo

package livesync

import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"

	"levelup/go-api/internal/config"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedKnownSharedH5 provisionne le shared h5 puis y insère quelques match_registry +
// xuid_aliases via un handle RW éphémère (fermé avant que le Manager n'ouvre le RO).
func seedKnownSharedH5(t *testing.T, repoRoot string) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).SharedDBPath(halo5.TitleSlug)
	db, err := syncpkg.OpenSharedDB(path)
	if err != nil {
		t.Fatalf("OpenSharedDB(h5 shared): %v", err)
	}
	ctx := context.Background()
	for _, id := range []string{"m1", "m2", "m3"} {
		if _, err := db.SQLDb().ExecContext(ctx,
			"INSERT INTO match_registry (match_id, start_time) VALUES (?, now())", id); err != nil {
			t.Fatalf("insert match_registry %s: %v", id, err)
		}
	}
	if _, err := db.SQLDb().ExecContext(ctx,
		"INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)", "2533274800000001", "PlayerOne"); err != nil {
		t.Fatalf("insert xuid_aliases: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close setup handle: %v", err)
	}
	return path
}

// TestLoadKnownMatchIDs_ReadOnlyNoSwap : known-set lit en RO même avec un lecteur
// concurrent tenu ouvert ; le provider ne quitte JAMAIS StateRO (pas de swap RW).
func TestLoadKnownMatchIDs_ReadOnlyNoSwap(t *testing.T) {
	repoRoot := t.TempDir()
	sharedPath := seedKnownSharedH5(t, repoRoot)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	cfg := &config.AppConfig{RepoRoot: repoRoot, SharedManager: mgr, UserTimezone: "Europe/Paris"}

	provider := sharedProviderForPath(cfg, sharedPath)
	if provider == nil {
		t.Fatalf("provider per-titre attendu non-nil")
	}

	// Un lecteur RO concurrent tenu ouvert (simule une requête HTTP servant le
	// shared h5 pendant le sync). Avec l'ancien code writer, le known-set aurait
	// dû drainer ce lecteur → drain timeout → échec.
	heldDB, releaseHeld, err := provider.Get(context.Background())
	if err != nil {
		t.Fatalf("provider.Get (lecteur concurrent): %v", err)
	}
	defer releaseHeld()
	if err := heldDB.PingContext(context.Background()); err != nil {
		t.Fatalf("ping lecteur concurrent: %v", err)
	}

	known, err := loadKnownMatchIDs(context.Background(), provider, sharedPath)
	if err != nil {
		t.Fatalf("loadKnownMatchIDs (lecteur concurrent tenu): %v", err)
	}
	for _, id := range []string{"m1", "m2", "m3"} {
		if !known[id] {
			t.Errorf("known-set: match %q manquant (%v)", id, known)
		}
	}
	if st := provider.State(); st != sharedprovider.StateRO {
		t.Errorf("provider.State() = %v, want StateRO (aucun swap RW ne doit être déclenché)", st)
	}

	// aliases-seed : même chemin RO, doit aussi réussir avec le lecteur tenu.
	seed := loadXUIDAliasesSeed(context.Background(), provider, sharedPath)
	if seed["PlayerOne"] != "2533274800000001" {
		t.Errorf("aliases-seed: PlayerOne = %q, want 2533274800000001", seed["PlayerOne"])
	}
	if st := provider.State(); st != sharedprovider.StateRO {
		t.Errorf("après aliases-seed, provider.State() = %v, want StateRO", st)
	}
}
