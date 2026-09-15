package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
)

const (
	identityXUID     = "2533274796795729"
	identityGamertag = "Inconnu"
)

// newIdentityFixture seme une instance de test sur disque : un profil suivi, un
// compte, des credentials et un dossier joueur — l'etat que la commande lit.
func newIdentityFixture(t *testing.T) *config.AppConfig {
	t.Helper()
	root := t.TempDir()
	paths := titlePkg.NewPathResolver(root)
	profilesPath := filepath.Join(root, "db_profiles.json")

	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	write(profilesPath, `{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {
      "`+identityGamertag+`": {"db_path": "x", "xuid": "`+identityXUID+`"},
      "Admin": {"db_path": "y", "xuid": "111"}
    }
  }
}`)
	write(paths.PlayerDBPath("halo_infinite", identityGamertag), "player-db")

	cfg := &config.AppConfig{
		RepoRoot:       root,
		DBProfilesPath: profilesPath,
		AuthDir:        filepath.Join(root, "data", "auth"),
	}
	users := userstore.NewStore(filepath.Join(cfg.AuthDir, "users.json"))
	if _, err := users.CreateFromXbox(identityGamertag, identityXUID); err != nil {
		t.Fatalf("compte: %v", err)
	}
	tokens := auth_platform.NewMultiUserTokenStore(paths.WatcherTokensDir())
	if err := tokens.Upsert(&auth_platform.UserTokens{
		XUID: identityXUID, Gamertag: identityGamertag, OAuthRefreshToken: "rt",
	}); err != nil {
		t.Fatalf("credentials: %v", err)
	}
	return cfg
}

// `identity list` montre les deux profils locaux, sans paniquer sur celui qui n'a
// ni compte ni credentials.
func TestIdentityList_ImprimeLAnnuaire(t *testing.T) {
	cfg := newIdentityFixture(t)
	var out bytes.Buffer

	if err := runIdentityList(cfg, &out); err != nil {
		t.Fatalf("runIdentityList: %v", err)
	}
	texte := out.String()
	for _, attendu := range []string{identityXUID, identityGamertag, "Admin", "halo_infinite", "2 identite(s)"} {
		if !strings.Contains(texte, attendu) {
			t.Errorf("sortie sans %q :\n%s", attendu, texte)
		}
	}
}

// Sans `--yes`, la commande SIMULE : elle imprime le rapport complet, sort 0, et
// ne supprime rien.
func TestIdentityPurge_SansYesEstUneSimulation(t *testing.T) {
	cfg := newIdentityFixture(t)
	paths := titlePkg.NewPathResolver(cfg.RepoRoot)
	var out bytes.Buffer

	if err := runIdentityPurge(cfg, []string{identityXUID}, &out); err != nil {
		t.Fatalf("runIdentityPurge: %v", err)
	}
	texte := out.String()
	if !strings.Contains(texte, "SIMULATION") {
		t.Errorf("le rapport doit se dire en simulation :\n%s", texte)
	}
	for _, attendu := range []string{"profile", "token", "account", "a faire", "base partagee"} {
		if !strings.Contains(texte, attendu) {
			t.Errorf("rapport sans %q :\n%s", attendu, texte)
		}
	}
	if _, err := os.Stat(paths.PlayerDir("halo_infinite", identityGamertag)); err != nil {
		t.Error("le dossier joueur a ete supprime par une simulation")
	}
	if _, err := os.Stat(filepath.Join(paths.WatcherTokensDir(), identityXUID+".json")); err != nil {
		t.Error("les credentials ont ete supprimes par une simulation")
	}
}

// Avec `--yes`, la commande execute et le dit.
func TestIdentityPurge_AvecYesExecute(t *testing.T) {
	cfg := newIdentityFixture(t)
	paths := titlePkg.NewPathResolver(cfg.RepoRoot)
	var out bytes.Buffer

	if err := runIdentityPurge(cfg, []string{identityXUID, "--yes"}, &out); err != nil {
		t.Fatalf("runIdentityPurge: %v", err)
	}
	if texte := out.String(); !strings.Contains(texte, "EXECUTION") || !strings.Contains(texte, "fait") {
		t.Errorf("le rapport doit dire ce qui a ete fait :\n%s", texte)
	}
	if _, err := os.Stat(paths.PlayerDir("halo_infinite", identityGamertag)); err == nil {
		t.Error("le dossier joueur devrait avoir disparu")
	}
	if _, err := os.Stat(filepath.Join(paths.WatcherTokensDir(), identityXUID+".json")); err == nil {
		t.Error("les credentials devraient avoir disparu")
	}
}

// Un xuid absent de tous les registres : erreur explicite qui renvoie a `list`.
func TestIdentityPurge_XUIDInconnu(t *testing.T) {
	cfg := newIdentityFixture(t)
	var out bytes.Buffer

	err := runIdentityPurge(cfg, []string{"000"}, &out)
	if err == nil {
		t.Fatal("un xuid inconnu doit produire une erreur")
	}
	if !strings.Contains(err.Error(), "identity list") {
		t.Errorf("l'erreur doit renvoyer a la commande de listage : %v", err)
	}
}

// Routage : une sous-commande inconnue, et l'absence de sous-commande, sont dites.
func TestRunIdentity_Routage(t *testing.T) {
	cfg := newIdentityFixture(t)

	if err := runIdentity(cfg, nil); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("sans sous-commande, l'usage doit etre rendu : %v", err)
	}
	if err := runIdentity(cfg, []string{"supprime-tout"}); err == nil ||
		!strings.Contains(err.Error(), "inconnue") {
		t.Errorf("une sous-commande inconnue doit etre dite : %v", err)
	}
}
