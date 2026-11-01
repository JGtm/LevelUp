package wakeup

import (
	"context"
	"errors"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/sync/deadtoken"
)

// stubProvider : un fournisseur dont le rafraîchissement OAuth est scripté par joueur (par jeton).
type stubProvider struct {
	mu    sync.Mutex
	calls []string                // jetons présentés, dans l'ordre
	reply map[string]refreshReply // par refresh token présenté
}

type refreshReply struct {
	access, rotated string
	err             error
}

func (p *stubProvider) InitDeviceFlow(context.Context) (auth.DeviceFlow, error) { return nil, nil }
func (p *stubProvider) TryOAuthRefresh(context.Context, string) (string, error) { return "", nil }
func (p *stubProvider) Exchange(context.Context, string) (*auth.ExchangeResult, error) {
	return nil, errors.New("l'échange Halo n'a rien à faire dans une vérification de jeton")
}
func (p *stubProvider) TryOAuthRefreshWithRotation(_ context.Context, rt string) (string, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, rt)
	r := p.reply[rt]
	return r.access, r.rotated, r.err
}

// newTokenCheck rend une vérification sur un magasin neuf garni de (xuid -> refresh token).
func newTokenCheck(t *testing.T, tokens map[string]string, players []domain.PlayerSummary, p *stubProvider) (*TokenCheck, *auth.MultiUserTokenStore) {
	t.Helper()
	store := auth.NewMultiUserTokenStore(t.TempDir())
	for xuid, rt := range tokens {
		if err := store.Upsert(&auth.UserTokens{XUID: xuid, Gamertag: "gt" + xuid, OAuthRefreshToken: rt}); err != nil {
			t.Fatalf("magasin : %v", err)
		}
	}
	return &TokenCheck{
		Players:  func() ([]domain.PlayerSummary, error) { return players, nil },
		Store:    store,
		Provider: p,
		Gate:     deadtoken.New(),
	}, store
}

func tracked(xuid string) domain.PlayerSummary {
	return domain.PlayerSummary{Gamertag: "gt" + xuid, XUID: xuid, SyncEnabled: true}
}

// TestTokenCheck_Vivant : le refresh token répond ; le jeton tourné est écrit au magasin, la bannière
// de reconnexion éventuelle est levée.
func TestTokenCheck_Vivant(t *testing.T) {
	p := &stubProvider{reply: map[string]refreshReply{"rt-1": {access: "at", rotated: "rt-2"}}}
	c, store := newTokenCheck(t, map[string]string{"100": "rt-1"}, []domain.PlayerSummary{tracked("100")}, p)

	report := c.Check(context.Background(), nil)

	if got := report.Results[0].Outcome; got != TokenAlive {
		t.Fatalf("issue = %q, attendu %q", got, TokenAlive)
	}
	u, err := store.Load("100")
	if err != nil {
		t.Fatal(err)
	}
	if u.OAuthRefreshToken != "rt-2" {
		t.Errorf("refresh token du magasin = %q, attendu le jeton tourné rt-2", u.OAuthRefreshToken)
	}
	if u.ReauthRequired {
		t.Error("reauth_required posé après un rafraîchissement réussi")
	}
}

// TestTokenCheck_JetonRemplaceApresMort : un jeton constaté mort puis remplacé (reconnexion) est
// présenté de nouveau ; son succès lève la bannière de reconnexion.
func TestTokenCheck_JetonRemplaceApresMort(t *testing.T) {
	dead := &auth.OAuthExchangeError{ErrorCode: "invalid_grant"}
	p := &stubProvider{reply: map[string]refreshReply{"rt-1": {err: dead}, "rt-9": {access: "at", rotated: "rt-10"}}}
	c, store := newTokenCheck(t, map[string]string{"100": "rt-1"}, []domain.PlayerSummary{tracked("100")}, p)

	if got := c.Check(context.Background(), nil).Results[0].Outcome; got != TokenDead {
		t.Fatalf("premier passage = %q, attendu %q", got, TokenDead)
	}
	if err := store.UpdateOAuthRefreshToken("100", "rt-9"); err != nil {
		t.Fatal(err)
	}
	if got := c.Check(context.Background(), nil).Results[0].Outcome; got != TokenAlive {
		t.Fatalf("passage après remplacement = %q, attendu %q", got, TokenAlive)
	}
	if store.IsReauthRequired("100") {
		t.Error("reauth_required reste posé alors que le nouveau jeton vit")
	}
}

// TestTokenCheck_Mort : invalid_grant pose reauth_required (la bannière) et le jeton n'est pas
// re-présenté à Microsoft au passage suivant.
func TestTokenCheck_Mort(t *testing.T) {
	dead := &auth.OAuthExchangeError{ErrorCode: "invalid_grant", Description: "AADSTS70008: expired"}
	p := &stubProvider{reply: map[string]refreshReply{"rt-1": {err: dead}}}
	c, store := newTokenCheck(t, map[string]string{"100": "rt-1"}, []domain.PlayerSummary{tracked("100")}, p)

	report := c.Check(context.Background(), nil)
	if got := report.Results[0].Outcome; got != TokenDead {
		t.Fatalf("issue = %q, attendu %q", got, TokenDead)
	}
	if !store.IsReauthRequired("100") {
		t.Error("reauth_required n'est pas posé pour un jeton révoqué")
	}
	u, _ := store.Load("100")
	if u.OAuthRefreshToken != "rt-1" {
		t.Errorf("le jeton a été modifié : %q (jamais de re-capture, ADR 0023)", u.OAuthRefreshToken)
	}

	report = c.Check(context.Background(), nil)
	if got := report.Results[0].Outcome; got != TokenAlreadyDead {
		t.Errorf("second passage = %q, attendu %q", got, TokenAlreadyDead)
	}
	if len(p.calls) != 1 {
		t.Errorf("le jeton mort a été présenté %d fois à Microsoft, attendu 1", len(p.calls))
	}
}

// TestTokenCheck_ReseauAbsent : une erreur de transport ne marque RIEN — ni bannière, ni dernier échec,
// ni modification du jeton — et se range « injoignable » (reprise ultérieure).
func TestTokenCheck_ReseauAbsent(t *testing.T) {
	cas := map[string]error{
		"transport": errors.New(`Post "https://login.microsoftonline.com/consumers/oauth2/v2.0/token": dial tcp: lookup login.microsoftonline.com: no such host`),
		"délai":     context.DeadlineExceeded,
		"5xx":       &auth.OAuthExchangeError{ErrorCode: "temporarily_unavailable", Description: "retry"},
	}
	for name, cause := range cas {
		t.Run(name, func(t *testing.T) {
			p := &stubProvider{reply: map[string]refreshReply{"rt-1": {err: cause}}}
			c, store := newTokenCheck(t, map[string]string{"100": "rt-1"}, []domain.PlayerSummary{tracked("100")}, p)

			report := c.Check(context.Background(), nil)

			if got := report.Results[0].Outcome; got != TokenUnreachable {
				t.Fatalf("issue = %q, attendu %q", got, TokenUnreachable)
			}
			if !report.NetworkAbsent() {
				t.Error("aucune réponse obtenue : le réseau doit être déclaré absent")
			}
			u, _ := store.Load("100")
			if u.ReauthRequired || u.LastAuthErrorClass != "" || u.OAuthRefreshToken != "rt-1" {
				t.Errorf("le magasin a été touché : reauth=%v dernier échec=%q jeton=%q",
					u.ReauthRequired, u.LastAuthErrorClass, u.OAuthRefreshToken)
			}
		})
	}
}

// TestTokenCheck_ConfigNEstPasUneRevocation : un refus de configuration est consigné mais ne lève pas
// la bannière de reconnexion (une reconnexion ne le réglerait pas).
func TestTokenCheck_ConfigNEstPasUneRevocation(t *testing.T) {
	cfgErr := &auth.OAuthExchangeError{ErrorCode: "invalid_client", Description: "AADSTS7000215"}
	p := &stubProvider{reply: map[string]refreshReply{"rt-1": {err: cfgErr}}}
	c, store := newTokenCheck(t, map[string]string{"100": "rt-1"}, []domain.PlayerSummary{tracked("100")}, p)

	report := c.Check(context.Background(), nil)

	if got := report.Results[0].Outcome; got != TokenRefused {
		t.Fatalf("issue = %q, attendu %q", got, TokenRefused)
	}
	if store.IsReauthRequired("100") {
		t.Error("bannière de reconnexion levée pour une erreur de configuration")
	}
}

// TestTokenCheck_PorteDesProfilsSuivis : seuls les profils déclarés, suivis, avec un jeton connu sont
// vérifiés ; un même xuid sur deux titres l'est une fois ; un profil sans xuid se résout par le magasin.
func TestTokenCheck_PorteDesProfilsSuivis(t *testing.T) {
	p := &stubProvider{reply: map[string]refreshReply{
		"rt-1": {access: "at", rotated: "rt-1b"}, "rt-2": {access: "at"}, "rt-3": {access: "at"},
	}}
	players := []domain.PlayerSummary{
		tracked("100"),
		{Gamertag: "gt100", XUID: "100", SyncEnabled: true, TitleSlug: "halo_5"}, // même xuid, autre titre
		{Gamertag: "gt200", XUID: "200", SyncEnabled: false},                     // en pause
		{Gamertag: "gt300", XUID: "300", SyncEnabled: true, AuthOnly: true},      // profil de jeton seul
		{Gamertag: "gt400", SyncEnabled: true},                                   // xuid inconnu du profil
		tracked("500"),                                                           // suivi mais sans entrée au magasin
	}
	c, _ := newTokenCheck(t, map[string]string{"100": "rt-1", "200": "rt-2", "300": "rt-3", "400": "rt-3"}, players, p)

	report := c.Check(context.Background(), nil)

	got := map[string]TokenOutcome{}
	for _, r := range report.Results {
		got[r.XUID] = r.Outcome
	}
	want := map[string]TokenOutcome{"100": TokenAlive, "400": TokenAlive, "500": TokenAbsent}
	if len(got) != len(want) {
		t.Fatalf("issues = %v, attendu %v", got, want)
	}
	for xuid, o := range want {
		if got[xuid] != o {
			t.Errorf("xuid %s : %q, attendu %q", xuid, got[xuid], o)
		}
	}
	if n := countOf(p.calls, "rt-1"); n != 1 {
		t.Errorf("le jeton du xuid 100 a été présenté %d fois, attendu 1", n)
	}
}

// TestTokenCheck_Reprise : only restreint la vérification aux joueurs restés sans réponse.
func TestTokenCheck_Reprise(t *testing.T) {
	p := &stubProvider{reply: map[string]refreshReply{"rt-1": {access: "at"}, "rt-2": {access: "at"}}}
	c, _ := newTokenCheck(t, map[string]string{"100": "rt-1", "200": "rt-2"},
		[]domain.PlayerSummary{tracked("100"), tracked("200")}, p)

	report := c.Check(context.Background(), []string{"200"})

	if len(report.Results) != 1 || report.Results[0].XUID != "200" {
		t.Fatalf("résultats = %+v, attendu le seul xuid 200", report.Results)
	}
	if len(p.calls) != 1 || p.calls[0] != "rt-2" {
		t.Errorf("jetons présentés = %v, attendu rt-2 seul", p.calls)
	}
}

func countOf(list []string, v string) int {
	n := 0
	for _, s := range list {
		if s == v {
			n++
		}
	}
	return n
}
