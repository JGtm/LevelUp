package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/transfer"
)

// newBoundSiteImport relie un import neuf à une configuration dont db_profiles.json porte `profiles`
// (JSON du titre par défaut, vide : aucun profil).
func newBoundSiteImport(t *testing.T, profiles string) *SiteImport {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	doc := `{"version": "3.0", "profiles": {"halo_infinite": {` + profiles + `}}}`
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &SiteImport{runner: transfer.NewRunner()}
	var wg sync.WaitGroup
	s.Bind(context.Background(), &wg, &config.AppConfig{DBProfilesPath: path, RepoRoot: dir}, nil)
	return s
}

// TestSiteImport_StatusIdleEnReponseDAPI : un import neuf est au repos, sans ligne insérée.
func TestSiteImport_StatusIdleEnReponseDAPI(t *testing.T) {
	st := (&SiteImport{runner: transfer.NewRunner()}).Status()
	if st.Phase != "idle" || st.Matches != 0 || len(st.Inserted) != 0 || st.ErrorCode != "" {
		t.Fatalf("état d'un import neuf = %+v, attendu idle sans ligne", st)
	}
}

// TestSiteImport_StartRefusDuJoueurLocal : sans liaison l'import est indisponible ; lié, il refuse un
// joueur absent, plusieurs joueurs sans nom, et un joueur sans identifiant Xbox, par les erreurs du port.
func TestSiteImport_StartRefusDuJoueurLocal(t *testing.T) {
	req := port.SiteImportRequest{URL: "https://example.invalid", Token: "t"}
	if err := (&SiteImport{runner: transfer.NewRunner()}).Start(req); !errors.Is(err, ErrSiteImportUnbound) {
		t.Fatalf("import non relié : %v, attendu ErrSiteImportUnbound", err)
	}
	two := `"alice": {"db_path": "a.duckdb", "xuid": "111", "waypoint_player": "alice"},
	        "bob": {"db_path": "b.duckdb", "xuid": "222", "waypoint_player": "bob"}`
	noXUID := `"carol": {"db_path": "c.duckdb", "waypoint_player": "carol"}`
	cases := []struct {
		name     string
		profiles string
		gamertag string
		want     error
	}{
		{"aucun joueur", ``, "", port.ErrSiteImportNoPlayer},
		{"gamertag inconnu", two, "zoe", port.ErrSiteImportNoPlayer},
		{"plusieurs joueurs sans nom", two, "", port.ErrSiteImportAmbiguous},
		{"joueur sans xuid", noXUID, "", port.ErrSiteImportPlayerNoXID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := req
			r.Gamertag = c.gamertag
			if err := newBoundSiteImport(t, c.profiles).Start(r); !errors.Is(err, c.want) {
				t.Fatalf("Start = %v, attendu %v", err, c.want)
			}
		})
	}
}
