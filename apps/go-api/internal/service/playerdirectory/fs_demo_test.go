package playerdirectory

// fs_demo_test.go — le témoin disque de la démo (lot L8-2 du plan sécurité du 2026-10-15,
// reste de recos-d). La démo n'a PAS le témoin du dépôt (il balaierait les dossiers de joueurs
// du poste), mais un témoin adossé à sa propre disposition : la fixture nomme ses dossiers
// autrement que ses clés de profil (demo-player -> DEMO), la correspondance vient du roster démo.

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// fixtureDemo crée une racine démo : DEMO (avec base), DEMO2 (sans base) et Stray (inconnu du roster).
func fixtureDemo(t *testing.T) (title.DemoLayout, []DemoPlayerDir) {
	t.Helper()
	root := t.TempDir()
	layout := title.NewDemoLayout(root)
	for _, d := range []string{"DEMO", "DEMO2", "Stray"} {
		if err := os.MkdirAll(layout.PlayerDir(testTitle, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(layout.PlayerDBPath(testTitle, "DEMO"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return layout, []DemoPlayerDir{{Key: "demo-player", Dir: "DEMO"}, {Key: "demo-player-2", Dir: "DEMO2"}}
}

func TestDemoFS_ConstateLaFixtureParCleDeProfil(t *testing.T) {
	layout, roster := fixtureDemo(t)
	fs := NewDemoFS(layout, roster)

	if !fs.PlayerDirExists(testTitle, "demo-player") || !fs.PlayerDirExists(testTitle, "Demo-Player-2") {
		t.Error("le dossier d'un joueur du roster doit être constaté par sa clé de profil (casse ignorée)")
	}
	if fs.PlayerDirExists(testTitle, "DEMO") {
		t.Error("un nom de dossier n'est pas une clé de profil")
	}
	if fs.PlayerDirExists(testTitle, "inconnu") || fs.PlayerDirExists(testTitle, "") {
		t.Error("clé inconnue ou vide : pas de dossier")
	}
	if !fs.PlayerDBExists(testTitle, "demo-player") || fs.PlayerDBExists(testTitle, "demo-player-2") {
		t.Error("la base n'existe que pour demo-player")
	}
	if got, want := fs.PlayerDBPath(testTitle, "demo-player"), layout.PlayerDBPath(testTitle, "DEMO"); got != want {
		t.Errorf("PlayerDBPath = %q, attendu %q", got, want)
	}
	if fs.PlayerDBPath(testTitle, "inconnu") != "" {
		t.Error("PlayerDBPath d'une clé inconnue doit être vide")
	}
}

func TestDemoFS_ListePlayerDirs_NommeLesDossiersDuRosterParLeurCle(t *testing.T) {
	layout, roster := fixtureDemo(t)
	got, err := NewDemoFS(layout, roster).ListPlayerDirs(testTitle)
	if err != nil {
		t.Fatalf("ListPlayerDirs : %v", err)
	}
	sort.Strings(got)
	if want := []string{"Stray", "demo-player", "demo-player-2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListPlayerDirs = %v, attendu %v", got, want)
	}
	// Titre absent de la fixture : liste vide, pas d'erreur.
	if got, err := NewDemoFS(layout, roster).ListPlayerDirs("halo_5"); err != nil || len(got) != 0 {
		t.Errorf("titre sans dossier joueurs : %v, %v", got, err)
	}
}

func TestDemoFS_NeSupprimeJamais(t *testing.T) {
	layout, roster := fixtureDemo(t)
	if err := NewDemoFS(layout, roster).RemovePlayerDir(testTitle, "Stray"); err == nil {
		t.Error("la démo est en lecture seule : la purge d'un dossier doit être refusée")
	}
	if _, err := os.Stat(filepath.Join(layout.PlayersRootDir(testTitle), "Stray")); err != nil {
		t.Errorf("le dossier a été touché : %v", err)
	}
}

func TestDirectory_Demo_SeulUnDossierHorsRosterEstOrphelin(t *testing.T) {
	layout, roster := fixtureDemo(t)
	d := New(Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{
			{PlayerSlug: "demo-player", Gamertag: "DemoPlayer", XUID: "0000000000000000", TitleSlug: testTitle},
			{PlayerSlug: "demo-player-2", Gamertag: "DemoPlayer2", XUID: "0000000000000001", TitleSlug: testTitle},
		}},
		FS:     NewDemoFS(layout, roster),
		Titles: []string{testTitle},
	})
	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List : %v", err)
	}
	var orphans []string
	refs := map[string][2]bool{}
	for _, rec := range resp.Identities {
		for _, o := range rec.OrphanDirs {
			orphans = append(orphans, o.Name)
		}
		for _, p := range rec.Profiles {
			refs[p.Key] = [2]bool{p.DirExists, p.DBExists}
		}
	}
	if !reflect.DeepEqual(orphans, []string{"Stray"}) {
		t.Errorf("orphelins = %v, attendu [Stray] (DEMO et DEMO2 sont ceux du roster)", orphans)
	}
	if refs["demo-player"] != [2]bool{true, true} || refs["demo-player-2"] != [2]bool{true, false} {
		t.Errorf("témoins disque des profils = %v", refs)
	}
}
