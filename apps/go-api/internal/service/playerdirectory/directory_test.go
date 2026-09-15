package playerdirectory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/port"
)

const (
	testTitle      = "halo_infinite"
	testOtherTitle = "halo_5"
)

// ─── Fakes des cinq lecteurs ─────────────────────────────────────────────────

type fakeProfiles struct {
	players []domain.PlayerSummary
	err     error
}

func (f *fakeProfiles) LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(titleFilter) == 0 || titleFilter[0] == "" {
		return f.players, nil
	}
	out := make([]domain.PlayerSummary, 0, len(f.players))
	for _, p := range f.players {
		if p.TitleSlug == titleFilter[0] {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeProfiles) HasTrackedProfile(titleSlug, xuid string) (bool, error) {
	players, err := f.LoadPlayers(titleSlug)
	if err != nil {
		return false, err
	}
	for _, p := range domain.SyncablePlayers(players) {
		if p.XUID == xuid {
			return true, nil
		}
	}
	return false, nil
}

type fakeAccounts struct {
	users []domain.AdminUserSummary
	err   error
}

func (f *fakeAccounts) List() ([]domain.AdminUserSummary, error) { return f.users, f.err }

type fakeTokens struct {
	tokens map[string]*auth.UserTokens
	err    error
}

func (f *fakeTokens) LoadAll() (map[string]*auth.UserTokens, error) { return f.tokens, f.err }

type fakeWatched struct {
	refs []domain.WatchedPlayerRef
}

func (f *fakeWatched) WatchedPlayers() []domain.WatchedPlayerRef { return f.refs }

// fakeFS : dossiers joueur par titre + player DB présentes (clé "titre/nom").
type fakeFS struct {
	dirs map[string][]string
	dbs  map[string]bool
	err  map[string]error
}

func (f *fakeFS) PlayerDirExists(titleSlug, key string) bool {
	for _, name := range f.dirs[titleSlug] {
		if strings.EqualFold(name, key) {
			return true
		}
	}
	return false
}

func (f *fakeFS) PlayerDBExists(titleSlug, key string) bool { return f.dbs[titleSlug+"/"+key] }

func (f *fakeFS) PlayerDBPath(titleSlug, key string) string {
	if key == "" {
		return ""
	}
	return titleSlug + "/" + key + "/stats.duckdb"
}

func (f *fakeFS) ListPlayerDirs(titleSlug string) ([]string, error) {
	if err, ok := f.err[titleSlug]; ok {
		return nil, err
	}
	return f.dirs[titleSlug], nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

var fixedNow = func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) }

func newTestDirectory(t *testing.T, d Deps) *Directory {
	t.Helper()
	if d.Titles == nil {
		d.Titles = []string{testTitle, testOtherTitle}
	}
	d.Now = fixedNow
	return New(d)
}

func trackedProfile(gamertag, xuid, titleSlug string) domain.PlayerSummary {
	return domain.PlayerSummary{
		PlayerSlug:  gamertag,
		Gamertag:    gamertag,
		XUID:        xuid,
		TitleSlug:   titleSlug,
		SyncEnabled: true,
	}
}

func recordFor(t *testing.T, resp domain.AdminIdentitiesResponse, xuid string) domain.IdentityRecord {
	t.Helper()
	for _, rec := range resp.Identities {
		if rec.XUID == xuid {
			return rec
		}
	}
	t.Fatalf("aucune identité pour le xuid %q (rendu : %+v)", xuid, resp.Identities)
	return domain.IdentityRecord{}
}

func codes(rec domain.IdentityRecord) []string {
	out := make([]string, 0, len(rec.Anomalies))
	for _, a := range rec.Anomalies {
		out = append(out, a.Code)
	}
	return out
}

func hasCode(rec domain.IdentityRecord, code string) bool {
	for _, a := range rec.Anomalies {
		if a.Code == code {
			return true
		}
	}
	return false
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestList_IdentiteComplete_AucuneAnomalie : profil suivi + compte + credentials
// + suivi live sur le MÊME titre + dossier présent = rien à signaler.
func TestList_IdentiteComplete_AucuneAnomalie(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
		Accounts: &fakeAccounts{users: []domain.AdminUserSummary{
			{Username: "spartan", Role: domain.RoleAdmin, Gamertag: "Spartan", XUID: "111"},
		}},
		Tokens: &fakeTokens{tokens: map[string]*auth.UserTokens{
			"111": {XUID: "111", Gamertag: "Spartan", OAuthRefreshToken: "rt", UpdatedAt: fixedNow()},
		}},
		Watched: &fakeWatched{refs: []domain.WatchedPlayerRef{{XUID: "111", Gamertag: "Spartan", TitleSlug: testTitle}}},
		FS:      &fakeFS{dirs: map[string][]string{testTitle: {"Spartan"}}, dbs: map[string]bool{testTitle + "/Spartan": true}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Identities) != 1 {
		t.Fatalf("identités = %d, want 1 (%+v)", len(resp.Identities), resp.Identities)
	}
	rec := recordFor(t, resp, "111")
	if len(rec.Anomalies) != 0 {
		t.Fatalf("anomalies = %v, want aucune", codes(rec))
	}
	if rec.Gamertag != "Spartan" || rec.Account == nil || rec.Token == nil {
		t.Fatalf("identité incomplète : %+v", rec)
	}
	if !rec.Token.HasRefreshToken || rec.Token.ReauthRequired {
		t.Fatalf("token = %+v", rec.Token)
	}
	if len(rec.Profiles) != 1 || !rec.Profiles[0].DirExists || !rec.Profiles[0].DBExists {
		t.Fatalf("profils = %+v", rec.Profiles)
	}
	if len(rec.Watched) != 1 || rec.Watched[0] != testTitle {
		t.Fatalf("suivi live = %v", rec.Watched)
	}
	if resp.GeneratedAt != "2026-09-15T12:00:00Z" {
		t.Fatalf("generated_at = %q", resp.GeneratedAt)
	}
	if resp.Counts[countIdentities] != 1 || resp.Counts[countWarnings] != 0 || resp.Counts[countInfos] != 0 {
		t.Fatalf("compteurs = %v", resp.Counts)
	}
}

// TestList_CompteSansProfil : l'état EXACT du compte du 2026-07-23 — un compte,
// des credentials, un dossier écrit par le sync, et aucun profil.
func TestList_CompteSansProfil(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{},
		Accounts: &fakeAccounts{users: []domain.AdminUserSummary{
			{Username: "inconnu", Role: domain.RoleUser, Gamertag: "Inconnu", XUID: "999"},
		}},
		Tokens: &fakeTokens{tokens: map[string]*auth.UserTokens{
			"999": {XUID: "999", Gamertag: "Inconnu", OAuthRefreshToken: "rt"},
		}},
		FS: &fakeFS{dirs: map[string][]string{testTitle: {"Inconnu"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "999")
	if !hasCode(rec, domain.AnomalyAccountWithoutProfile) {
		t.Fatalf("anomalies = %v, want %s", codes(rec), domain.AnomalyAccountWithoutProfile)
	}
	if !hasCode(rec, domain.AnomalyPlayerDirOrphan) {
		t.Fatalf("anomalies = %v, want %s", codes(rec), domain.AnomalyPlayerDirOrphan)
	}
	if hasCode(rec, domain.AnomalyTokenOrphan) {
		t.Fatalf("token_orphan ne doit PAS être posé quand un compte existe : %v", codes(rec))
	}
	if len(rec.OrphanDirs) != 1 || rec.OrphanDirs[0].Name != "Inconnu" {
		t.Fatalf("dossiers orphelins = %+v", rec.OrphanDirs)
	}
	if resp.Counts[domain.AnomalyAccountWithoutProfile] != 1 || resp.Counts[countWarnings] != 2 {
		t.Fatalf("compteurs = %v", resp.Counts)
	}
}

// TestList_TokenSeul : des credentials sans compte ni profil — plus personne ne
// les réclame.
func TestList_TokenSeul(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{},
		Accounts: &fakeAccounts{},
		Tokens: &fakeTokens{tokens: map[string]*auth.UserTokens{
			"777": {XUID: "777", Gamertag: "Oublie", ReauthRequired: true, LastAuthError: "invalid_grant"},
		}},
		FS: &fakeFS{},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "777")
	if !hasCode(rec, domain.AnomalyTokenOrphan) {
		t.Fatalf("anomalies = %v, want %s", codes(rec), domain.AnomalyTokenOrphan)
	}
	if rec.Token == nil || rec.Token.HasRefreshToken || !rec.Token.ReauthRequired {
		t.Fatalf("token = %+v", rec.Token)
	}
	if rec.Token.LastAuthError != "invalid_grant" {
		t.Fatalf("last_auth_error = %q", rec.Token.LastAuthError)
	}
	if rec.Gamertag != "Oublie" {
		t.Fatalf("gamertag = %q (le token est la 3e source de repli)", rec.Gamertag)
	}
}

// TestList_DossierOrphelinSansAucunRegistre : un dossier joueur que PLUS AUCUN
// registre ne réclame reste visible — le masquer reproduirait le trou que
// l'annuaire ferme.
func TestList_DossierOrphelinSansAucunRegistre(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{},
		FS:       &fakeFS{dirs: map[string][]string{testTitle: {"Fantome"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Identities) != 1 {
		t.Fatalf("identités = %+v, want 1", resp.Identities)
	}
	rec := resp.Identities[0]
	if rec.XUID != "" || rec.Gamertag != "Fantome" {
		t.Fatalf("identité = %+v", rec)
	}
	if !hasCode(rec, domain.AnomalyPlayerDirOrphan) {
		t.Fatalf("anomalies = %v", codes(rec))
	}
	if rec.Anomalies[0].Detail != testTitle+"/Fantome" {
		t.Fatalf("detail = %q (contexte machine attendu)", rec.Anomalies[0].Detail)
	}
}

// TestList_DossierCasseDifferente_PasOrphelin : la comparaison de clé est
// insensible à la casse, comme dbprofiles.File.FindKey.
func TestList_DossierCasseDifferente_PasOrphelin(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
		FS:       &fakeFS{dirs: map[string][]string{testTitle: {"SPARTAN"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "111")
	if hasCode(rec, domain.AnomalyPlayerDirOrphan) {
		t.Fatalf("anomalies = %v, want aucun orphelin", codes(rec))
	}
	if len(resp.Identities) != 1 {
		t.Fatalf("identités = %+v, want 1 (pas de doublon de casse)", resp.Identities)
	}
}

// TestList_ProfilAmiSansCompte : un ami suivi par l'administrateur n'a ni compte
// ni credentials — deux `info`, aucun `warning`.
func TestList_ProfilAmiSansCompte(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Ami", "222", testTitle)}},
		Accounts: &fakeAccounts{},
		Tokens:   &fakeTokens{},
		FS:       &fakeFS{dirs: map[string][]string{testTitle: {"Ami"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "222")
	if !hasCode(rec, domain.AnomalyProfileWithoutAccount) || !hasCode(rec, domain.AnomalyProfileWithoutToken) {
		t.Fatalf("anomalies = %v", codes(rec))
	}
	for _, a := range rec.Anomalies {
		if a.Severity != domain.AnomalySeverityInfo {
			t.Fatalf("anomalie %s en sévérité %s, want info", a.Code, a.Severity)
		}
	}
	if resp.Counts[countWarnings] != 0 || resp.Counts[countInfos] != 2 {
		t.Fatalf("compteurs = %v", resp.Counts)
	}
}

// TestList_SuiviLiveSansProfilSurCeTitre : un profil Halo 5 n'autorise pas un
// suivi live Halo Infinite (la question se pose PAR TITRE).
func TestList_SuiviLiveSansProfilSurCeTitre(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testOtherTitle)}},
		Watched:  &fakeWatched{refs: []domain.WatchedPlayerRef{{XUID: "111", Gamertag: "Spartan", TitleSlug: testTitle}}},
		FS:       &fakeFS{dirs: map[string][]string{testOtherTitle: {"Spartan"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "111")
	if !hasCode(rec, domain.AnomalyWatchedWithoutProfile) {
		t.Fatalf("anomalies = %v, want %s", codes(rec), domain.AnomalyWatchedWithoutProfile)
	}
	if len(resp.Identities) != 1 {
		t.Fatalf("identités = %+v, want 1 (le suivi live rejoint la ligne du xuid)", resp.Identities)
	}
}

// TestList_TriEtCompteurs : les identités à `warning` passent devant.
func TestList_TriEtCompteurs(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{
			trackedProfile("Alpha", "111", testTitle),
			trackedProfile("Zoulou", "333", testTitle),
		}},
		Accounts: &fakeAccounts{users: []domain.AdminUserSummary{
			{Username: "alpha", Role: domain.RoleAdmin, Gamertag: "Alpha", XUID: "111"},
			{Username: "zoulou", Role: domain.RoleUser, Gamertag: "Zoulou", XUID: "333"},
			{Username: "orphelin", Role: domain.RoleUser, Gamertag: "Orphelin", XUID: "999"},
		}},
		Tokens: &fakeTokens{tokens: map[string]*auth.UserTokens{
			"111": {XUID: "111", OAuthRefreshToken: "rt"},
			"333": {XUID: "333", OAuthRefreshToken: "rt"},
		}},
		FS: &fakeFS{dirs: map[string][]string{testTitle: {"Alpha", "Zoulou"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Identities) != 3 {
		t.Fatalf("identités = %d, want 3", len(resp.Identities))
	}
	if resp.Identities[0].XUID != "999" {
		t.Fatalf("première ligne = %q, want 999 (warning d'abord)", resp.Identities[0].XUID)
	}
	if resp.Counts[countIdentities] != 3 || resp.Counts[countWarnings] != 1 {
		t.Fatalf("compteurs = %v", resp.Counts)
	}
}

// TestList_ErreurDeRegistre_Remontee : un registre illisible interrompt — un
// annuaire amputé inventerait des anomalies fausses.
func TestList_ErreurDeRegistre_Remontee(t *testing.T) {
	boom := errors.New("boom")
	cases := map[string]Deps{
		"profils":     {Profiles: &fakeProfiles{err: boom}},
		"comptes":     {Profiles: &fakeProfiles{}, Accounts: &fakeAccounts{err: boom}},
		"credentials": {Profiles: &fakeProfiles{}, Tokens: &fakeTokens{err: boom}},
	}
	for name, deps := range cases {
		t.Run(name, func(t *testing.T) {
			d := newTestDirectory(t, deps)
			if _, err := d.List(context.Background()); !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
		})
	}
}

// TestList_BalayageDisqueEnEchec_Degrade : le disque, lui, dégrade par titre —
// les registres restent lisibles.
func TestList_BalayageDisqueEnEchec_Degrade(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
		FS:       &fakeFS{err: map[string]error{testTitle: errors.New("disque")}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "111")
	if hasCode(rec, domain.AnomalyPlayerDirOrphan) {
		t.Fatalf("anomalies = %v : aucun orphelin ne peut être conclu d'un balayage échoué", codes(rec))
	}
}

// TestList_SansWatcher : pas de daemon (CLI, serveur sans watcher) → aucun suivi
// live, aucune anomalie inventée.
func TestList_SansWatcher(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
		FS:       &fakeFS{dirs: map[string][]string{testTitle: {"Spartan"}}},
	})

	resp, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rec := recordFor(t, resp, "111")
	if len(rec.Watched) != 0 {
		t.Fatalf("suivi live = %v, want vide", rec.Watched)
	}
	if hasCode(rec, domain.AnomalyWatchedWithoutProfile) {
		t.Fatalf("anomalies = %v", codes(rec))
	}
}

// TestGet : par xuid, ErrIdentityNotFound sinon.
func TestGet(t *testing.T) {
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{players: []domain.PlayerSummary{trackedProfile("Spartan", "111", testTitle)}},
	})

	rec, err := d.Get(context.Background(), "111")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.Gamertag != "Spartan" {
		t.Fatalf("rec = %+v", rec)
	}
	if _, err := d.Get(context.Background(), "000"); !errors.Is(err, port.ErrIdentityNotFound) {
		t.Fatalf("err = %v, want ErrIdentityNotFound", err)
	}
	if _, err := d.Get(context.Background(), ""); !errors.Is(err, port.ErrIdentityNotFound) {
		t.Fatalf("xuid vide : err = %v, want ErrIdentityNotFound", err)
	}
}

// TestHasTrackedProfile_Delegue : l'annuaire ne re-filtre pas — il pose la
// question au lecteur de profils (une seule définition de « suivi »).
func TestHasTrackedProfile_Delegue(t *testing.T) {
	profiles := &fakeProfiles{players: []domain.PlayerSummary{
		trackedProfile("Spartan", "111", testTitle),
		{PlayerSlug: "Pause", Gamertag: "Pause", XUID: "222", TitleSlug: testTitle, SyncEnabled: false},
	}}
	d := newTestDirectory(t, Deps{Profiles: profiles})

	for _, tc := range []struct {
		name  string
		xuid  string
		title string
		want  bool
	}{
		{"profil suivi", "111", testTitle, true},
		{"profil en pause", "222", testTitle, false},
		{"autre titre", "111", testOtherTitle, false},
		{"xuid inconnu", "000", testTitle, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.HasTrackedProfile(context.Background(), tc.title, tc.xuid)
			if err != nil {
				t.Fatalf("HasTrackedProfile: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
