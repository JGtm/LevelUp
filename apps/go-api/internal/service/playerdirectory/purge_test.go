package playerdirectory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// Ces tests montent une instance RÉELLE sur disque — db_profiles.json, users.json,
// groups.json, watcher_tokens/{xuid}.json, un dossier joueur avec sa player DB, et
// une base partagée — parce que l'invariant qu'ils gardent est un invariant de
// FICHIERS : ce qui disparaît, et surtout ce qui ne doit PAS bouger d'un octet.

const (
	purgeXUID     = "2533274796795729"
	purgeGamertag = "Inconnu"
	purgeAdminGT  = "Admin"
	purgeAdminXID = "111"
)

// purgeFixture est l'instance de test et ce qu'il faut pour l'inspecter.
type purgeFixture struct {
	repoRoot   string
	dir        *Directory
	users      *userstore.Store
	groups     *groupstore.GroupStore
	tokens     *auth.MultiUserTokenStore
	paths      *titlePkg.PathResolver
	sharedPath string
	groupID    string
}

// newPurgeFixture sème l'état EXACT laissé par l'incident du 2026-07-23 : un
// compte, des credentials, un profil de suivi avec son dossier et sa player DB,
// un dossier ORPHELIN sur un autre titre, une appartenance à un groupe — et une
// base partagée qui ne doit pas bouger.
func newPurgeFixture(t *testing.T) *purgeFixture {
	t.Helper()
	root := t.TempDir()
	paths := titlePkg.NewPathResolver(root)
	f := &purgeFixture{repoRoot: root, paths: paths}

	// db_profiles.json v3 : le joueur purgé sur le titre par défaut, l'admin à côté
	// (sans lui, retirer le dernier titre actif du joueur serait le seul cas testé).
	profilesPath := filepath.Join(root, "db_profiles.json")
	writeFile(t, profilesPath, `{
  "version": "3.0",
  "admin": "`+purgeAdminGT+`",
  "profiles": {
    "`+testTitle+`": {
      "`+purgeGamertag+`": {"db_path": "x", "xuid": "`+purgeXUID+`"},
      "`+purgeAdminGT+`": {"db_path": "y", "xuid": "`+purgeAdminXID+`"}
    }
  }
}`)

	// Dossier joueur + player DB factice (le témoin de ce qu'un sync a écrit).
	writeFile(t, paths.PlayerDBPath(testTitle, purgeGamertag), "player-db")
	// Dossier ORPHELIN sur l'autre titre : aucun profil ne le déclare.
	writeFile(t, paths.PlayerDBPath(testOtherTitle, purgeGamertag), "orphan-db")
	// Base partagée : c'est elle qui ne doit JAMAIS bouger (ADR 0035 D6).
	f.sharedPath = paths.SharedDBPath(testTitle)
	writeFile(t, f.sharedPath, "matchs de tout le monde, append-only")

	f.users = userstore.NewStore(filepath.Join(root, "data", "auth", "users.json"))
	if _, err := f.users.CreateFromXbox(purgeGamertag, purgeXUID); err != nil {
		t.Fatalf("compte: %v", err)
	}
	f.tokens = auth.NewMultiUserTokenStore(paths.WatcherTokensDir())
	if err := f.tokens.Upsert(&auth.UserTokens{
		XUID: purgeXUID, Gamertag: purgeGamertag, OAuthRefreshToken: "rt",
	}); err != nil {
		t.Fatalf("credentials: %v", err)
	}
	f.groups = groupstore.NewGroupStore(filepath.Join(root, "data", "auth", "groups.json"))
	g, err := f.groups.Create("Famille", purgeAdminXID, purgeAdminGT)
	if err != nil {
		t.Fatalf("groupe: %v", err)
	}
	f.groupID = g.ID
	if err := f.groups.AddMember(g.ID, purgeXUID, purgeGamertag); err != nil {
		t.Fatalf("membre: %v", err)
	}

	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	profiles := service.NewProfileService(profilesPath, root)
	f.dir = New(Deps{
		Profiles: cfg,
		Accounts: f.users,
		Tokens:   f.tokens,
		FS:       NewPathFS(root),
		Titles:   []string{testTitle, testOtherTitle},
		Purge: PurgeDeps{
			Profiles: profiles,
			Tokens:   f.tokens,
			Groups:   f.groups,
			Accounts: f.users,
		},
	})
	return f
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func stepsOfKind(report domain.PurgeReport, kind string) []domain.PurgeStep {
	var out []domain.PurgeStep
	for _, s := range report.Steps {
		if s.Kind == kind {
			out = append(out, s)
		}
	}
	return out
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// La purge complète : tout ce qui décrit l'identité disparaît, et la base
// partagée ne bouge pas d'un octet (ADR 0035 D6).
func TestPurge_ToutSaufLaBasePartagee(t *testing.T) {
	f := newPurgeFixture(t)
	avant := sha256File(t, f.sharedPath)

	report, err := f.dir.Purge(context.Background(), purgeXUID, domain.PurgeOptions{})
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if apres := sha256File(t, f.sharedPath); apres != avant {
		t.Fatalf("la base partagee a ete modifiee : %s -> %s", avant, apres)
	}

	if exists(f.paths.PlayerDir(testTitle, purgeGamertag)) {
		t.Error("le dossier du profil devrait avoir disparu")
	}
	if exists(f.paths.PlayerDir(testOtherTitle, purgeGamertag)) {
		t.Error("le dossier orphelin devrait avoir disparu")
	}
	if exists(filepath.Join(f.paths.WatcherTokensDir(), purgeXUID+".json")) {
		t.Error("les credentials devraient avoir disparu")
	}
	if _, err := f.users.GetByXUID(purgeXUID); err == nil {
		t.Error("le compte devrait avoir disparu")
	}
	g, err := f.groups.Get(f.groupID)
	if err != nil {
		t.Fatalf("groupe: %v", err)
	}
	if g.HasMember(purgeXUID) {
		t.Error("le xuid ne devrait plus etre membre du groupe")
	}
	// Le profil restant (l'admin) est intact : une purge vise UNE identité.
	cfg := &config.AppConfig{DBProfilesPath: filepath.Join(f.repoRoot, "db_profiles.json")}
	players, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatalf("LoadPlayers: %v", err)
	}
	if len(players) != 1 || players[0].Gamertag != purgeAdminGT {
		t.Fatalf("profils restants = %+v, attendu le seul admin", players)
	}

	for _, s := range report.Steps {
		if !s.Done {
			t.Errorf("etape non executee : %+v", s)
		}
	}
	if len(stepsOfKind(report, domain.PurgeStepProfile)) != 1 ||
		len(stepsOfKind(report, domain.PurgeStepOrphanDir)) != 1 ||
		len(stepsOfKind(report, domain.PurgeStepToken)) != 1 ||
		len(stepsOfKind(report, domain.PurgeStepGroup)) != 1 ||
		len(stepsOfKind(report, domain.PurgeStepAccount)) != 1 {
		t.Fatalf("rapport incomplet : %+v", report.Steps)
	}
	if report.Gamertag != purgeGamertag || report.DryRun {
		t.Errorf("en-tete du rapport = %+v", report)
	}
}

// Le DERNIER titre actif d'un joueur : `RemoveEntry` le refuserait
// (ErrLastActiveTitle), et le profil resterait en place. La purge, elle, fait
// disparaître le joueur — c'est la raison d'être de PurgeIdentityData.
func TestPurge_DernierTitreActifRetireQuandMeme(t *testing.T) {
	f := newPurgeFixture(t)

	if _, err := f.dir.Purge(context.Background(), purgeXUID, domain.PurgeOptions{}); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	cfg := &config.AppConfig{DBProfilesPath: filepath.Join(f.repoRoot, "db_profiles.json")}
	players, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatalf("LoadPlayers: %v", err)
	}
	for _, p := range players {
		if p.XUID == purgeXUID {
			t.Fatalf("le profil du joueur purge est encore la : %+v", p)
		}
	}
}

// Simulation : le rapport est COMPLET, et rien n'a bougé sur le disque.
func TestPurge_DryRunNeSupprimeRien(t *testing.T) {
	f := newPurgeFixture(t)
	avant := sha256File(t, f.sharedPath)

	report, err := f.dir.Purge(context.Background(), purgeXUID, domain.PurgeOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if !report.DryRun {
		t.Error("le rapport doit se dire en simulation")
	}
	if len(report.Steps) != 5 {
		t.Fatalf("le rapport de simulation doit lister les memes etapes : %+v", report.Steps)
	}
	for _, s := range report.Steps {
		if s.Done || s.Err != "" {
			t.Errorf("une etape simulee n'est ni faite ni en echec : %+v", s)
		}
	}

	if sha256File(t, f.sharedPath) != avant {
		t.Error("la base partagee a bouge en simulation")
	}
	if !exists(f.paths.PlayerDir(testTitle, purgeGamertag)) {
		t.Error("le dossier du profil a ete supprime en simulation")
	}
	if !exists(f.paths.PlayerDir(testOtherTitle, purgeGamertag)) {
		t.Error("le dossier orphelin a ete supprime en simulation")
	}
	if !exists(filepath.Join(f.paths.WatcherTokensDir(), purgeXUID+".json")) {
		t.Error("les credentials ont ete supprimes en simulation")
	}
	if _, err := f.users.GetByXUID(purgeXUID); err != nil {
		t.Error("le compte a ete supprime en simulation")
	}
	g, _ := f.groups.Get(f.groupID)
	if !g.HasMember(purgeXUID) {
		t.Error("le membre a ete retire du groupe en simulation")
	}
}

// Un compte administrateur ne se purge pas par cette porte.
func TestPurge_RefuseUnAdministrateur(t *testing.T) {
	f := newPurgeFixture(t)
	if err := f.users.SetRole(purgeGamertag, domain.RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	avant := sha256File(t, f.sharedPath)

	_, err := f.dir.Purge(context.Background(), purgeXUID, domain.PurgeOptions{})
	if !errors.Is(err, ErrPurgeAdminRefused) {
		t.Fatalf("ErrPurgeAdminRefused attendue, recu %v", err)
	}
	if !exists(f.paths.PlayerDir(testTitle, purgeGamertag)) {
		t.Error("rien ne doit avoir ete supprime")
	}
	if sha256File(t, f.sharedPath) != avant {
		t.Error("la base partagee a bouge")
	}
}

// Une étape en échec n'arrête pas les suivantes : le rapport reste complet et
// l'erreur agrège. Ici, le retrait des credentials échoue.
func TestPurge_EtapeEnEchecNArretePasLesSuivantes(t *testing.T) {
	f := newPurgeFixture(t)
	f.dir.purge.Tokens = failingTokens{}

	report, err := f.dir.Purge(context.Background(), purgeXUID, domain.PurgeOptions{})
	if err == nil {
		t.Fatal("l'erreur agregee doit remonter")
	}
	tokenSteps := stepsOfKind(report, domain.PurgeStepToken)
	if len(tokenSteps) != 1 || tokenSteps[0].Done || tokenSteps[0].Err == "" {
		t.Fatalf("l'etape credentials doit etre en echec et le dire : %+v", tokenSteps)
	}
	accountSteps := stepsOfKind(report, domain.PurgeStepAccount)
	if len(accountSteps) != 1 || !accountSteps[0].Done {
		t.Fatalf("l'etape compte doit s'executer malgre l'echec precedent : %+v", accountSteps)
	}
	if _, err := f.users.GetByXUID(purgeXUID); err == nil {
		t.Error("le compte devrait avoir disparu")
	}
	if exists(f.paths.PlayerDir(testTitle, purgeGamertag)) {
		t.Error("le dossier du profil devrait avoir disparu")
	}
}

type failingTokens struct{}

func (failingTokens) Remove(string) error { return errors.New("fichier verrouille") }

// Un xuid inconnu de tous les registres : refus net, aucune écriture.
func TestPurge_XUIDInconnuEtVide(t *testing.T) {
	f := newPurgeFixture(t)

	if _, err := f.dir.Purge(context.Background(), "", domain.PurgeOptions{}); !errors.Is(err, ErrPurgeInvalidXUID) {
		t.Fatalf("ErrPurgeInvalidXUID attendue, recu %v", err)
	}
	if _, err := f.dir.Purge(context.Background(), "000", domain.PurgeOptions{}); err == nil {
		t.Fatal("un xuid inconnu doit etre refuse")
	}
	if !exists(f.paths.PlayerDir(testTitle, purgeGamertag)) {
		t.Error("rien ne doit avoir ete supprime")
	}
}

// Le suivi live part EN PREMIER, et par xuid : le double enregistre l'ordre.
func TestPurge_SuiviLiveRetireEnPremier(t *testing.T) {
	ordre := &purgeOrdre{}
	d := New(Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
		Watched:  &fakeWatched{refs: []domain.WatchedPlayerRef{{XUID: "111", Gamertag: "Spartan", TitleSlug: testTitle}}},
		FS:       &fakeFS{dirs: map[string][]string{testTitle: {"Spartan"}}},
		Titles:   []string{testTitle},
		Purge: PurgeDeps{
			Watcher:  ordre,
			Profiles: ordre,
		},
	})

	report, err := d.Purge(context.Background(), "111", domain.PurgeOptions{})
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if len(ordre.appels) != 2 || ordre.appels[0] != "watcher" || ordre.appels[1] != "profiles" {
		t.Fatalf("ordre des appels = %v, attendu watcher puis profiles", ordre.appels)
	}
	if steps := stepsOfKind(report, domain.PurgeStepWatcher); len(steps) != 1 || !steps[0].Done {
		t.Fatalf("etape de suivi live = %+v", steps)
	}
}

// purgeOrdre note l'ordre des écritures destructrices.
type purgeOrdre struct {
	appels []string
}

func (p *purgeOrdre) RemovePlayer(_ context.Context, _ string) []string {
	p.appels = append(p.appels, "watcher")
	return []string{testTitle}
}

func (p *purgeOrdre) PurgeIdentityData(_ string, titleSlugs []string) (map[string]bool, error) {
	p.appels = append(p.appels, "profiles")
	out := map[string]bool{}
	for _, s := range titleSlugs {
		out[s] = true
	}
	return out, nil
}
