package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
)

func poolSharingFixture(t *testing.T) (*config.AppConfig, *auth_platform.MultiUserTokenStore) {
	t.Helper()
	cfg := &config.AppConfig{RepoRoot: t.TempDir(), AuthDir: t.TempDir()}
	dir := cfg.WatcherTokensDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth_platform.NewMultiUserTokenStore(dir)
	for _, u := range []*auth_platform.UserTokens{
		{XUID: "1", Gamertag: "Ami", OAuthRefreshToken: "rt1"},
		{XUID: "2", Gamertag: "Refus", OAuthRefreshToken: "rt2"},
		{XUID: "3", Gamertag: "Inconnu", OAuthRefreshToken: "rt3"}, // jeton présent, accord NON recueilli
	} {
		if err := store.Upsert(u); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetPoolSharing("2", false, auth_platform.PoolSharingBasisUserChoice); err != nil {
		t.Fatal(err)
	}
	return cfg, store
}

func TestPoolSharingGrandfather_DryRunWritesNothing(t *testing.T) {
	cfg, store := poolSharingFixture(t)
	var out bytes.Buffer

	if err := runPoolSharingTo(cfg, []string{"grandfather", "--players", "ami,Refus"}, &out); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "SIMULATION : 1 joueur(s)") {
		t.Fatalf("la simulation doit annoncer 1 joueur:\n%s", out.String())
	}
	if got := store.PoolSharingState("1"); got != domain.PoolSharingUndecided {
		t.Fatalf("une simulation n'écrit rien, état = %q", got)
	}
}

func TestPoolSharingGrandfather_YesRecordsOnlyUndecided(t *testing.T) {
	cfg, store := poolSharingFixture(t)
	var out bytes.Buffer

	if err := runPoolSharingTo(cfg, []string{"grandfather", "--players", "Ami,2", "--yes"}, &out); err != nil {
		t.Fatal(err)
	}

	if got := store.PoolSharingState("1"); got != domain.PoolSharingGranted {
		t.Fatalf("le joueur sans décision doit être marqué, état = %q", got)
	}
	if got := store.PoolSharingState("2"); got != domain.PoolSharingDeclined {
		t.Fatalf("un refus enregistré ne doit jamais être écrasé, état = %q", got)
	}
}

func TestPoolSharingList_ShowsEachPlayerState(t *testing.T) {
	cfg, _ := poolSharingFixture(t)
	var out bytes.Buffer

	if err := runPoolSharingTo(cfg, []string{"list"}, &out); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "Ami") || !strings.Contains(got, "undecided") ||
		!strings.Contains(got, "Refus") || !strings.Contains(got, "declined") {
		t.Fatalf("liste incomplète:\n%s", got)
	}
}

func TestPoolSharing_UnknownSubcommand(t *testing.T) {
	cfg, _ := poolSharingFixture(t)
	if err := runPoolSharingTo(cfg, []string{"nope"}, &bytes.Buffer{}); err == nil {
		t.Fatal("une sous-commande inconnue doit échouer")
	}
	if err := runPoolSharingTo(cfg, nil, &bytes.Buffer{}); err == nil {
		t.Fatal("sans sous-commande, l'usage doit être rendu en erreur")
	}
}

func TestPoolSharingGrandfather_YesLeavesUndesignatedPlayersUndecided(t *testing.T) {
	cfg, store := poolSharingFixture(t)

	if err := runPoolSharingTo(cfg, []string{"grandfather", "--players", "Ami", "--yes"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	if got := store.PoolSharingState("3"); got != domain.PoolSharingUndecided {
		t.Fatalf("un joueur non désigné ne doit pas être marqué, état = %q", got)
	}
}

func TestPoolSharingGrandfather_RequiresExplicitDesignation(t *testing.T) {
	cfg, store := poolSharingFixture(t)
	for _, args := range [][]string{
		{"grandfather", "--yes"},
		{"grandfather", "--players", "Ami", "--all", "--yes"},
	} {
		if err := runPoolSharingTo(cfg, args, &bytes.Buffer{}); err == nil {
			t.Fatalf("%v : la désignation des joueurs doit être exigée", args)
		}
	}
	if got := store.PoolSharingState("1"); got != domain.PoolSharingUndecided {
		t.Fatalf("aucune écriture sans désignation, état = %q", got)
	}
}

func TestPoolSharingGrandfather_UnknownPlayerAbortsBeforeWriting(t *testing.T) {
	cfg, store := poolSharingFixture(t)

	err := runPoolSharingTo(cfg, []string{"grandfather", "--players", "Ami,Fautedefrappe", "--yes"}, &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "Fautedefrappe") {
		t.Fatalf("un nom inconnu doit faire échouer la commande en le nommant, got %v", err)
	}
	if got := store.PoolSharingState("1"); got != domain.PoolSharingUndecided {
		t.Fatalf("aucune écriture partielle, état = %q", got)
	}
}

func TestPoolSharingGrandfather_AllMarksEveryUndecided(t *testing.T) {
	cfg, store := poolSharingFixture(t)

	if err := runPoolSharingTo(cfg, []string{"grandfather", "--all", "--yes"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	for _, x := range []string{"1", "3"} {
		if got := store.PoolSharingState(x); got != domain.PoolSharingGranted {
			t.Fatalf("xuid %s : état = %q, want granted", x, got)
		}
	}
	if got := store.PoolSharingState("2"); got != domain.PoolSharingDeclined {
		t.Fatalf("--all n'écrase pas un refus, état = %q", got)
	}
}
