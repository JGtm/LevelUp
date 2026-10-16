package auth

import (
	"errors"
	"testing"
)

func newSharingStore(t *testing.T, users ...*UserTokens) *MultiUserTokenStore {
	t.Helper()
	s := NewMultiUserTokenStore(t.TempDir())
	for _, u := range users {
		if err := s.Upsert(u); err != nil {
			t.Fatalf("Upsert(%s): %v", u.XUID, err)
		}
	}
	return s
}

func TestSharesWithPool_UndecidedIsNotShared(t *testing.T) {
	var nilTokens *UserTokens
	if nilTokens.SharesWithPool() {
		t.Fatal("nil UserTokens ne partage pas")
	}
	if (&UserTokens{}).SharesWithPool() {
		t.Fatal("aucune décision doit valoir refus")
	}
	if (&UserTokens{PoolSharing: &PoolSharing{Granted: false}}).SharesWithPool() {
		t.Fatal("un retrait ne partage pas")
	}
	if !(&UserTokens{PoolSharing: &PoolSharing{Granted: true}}).SharesWithPool() {
		t.Fatal("un accord partage")
	}
}

func TestSetPoolSharing_GrantThenWithdraw(t *testing.T) {
	s := newSharingStore(t, &UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: "rt-a"})

	if err := s.SetPoolSharing("111", true, PoolSharingBasisUserChoice); err != nil {
		t.Fatalf("grant: %v", err)
	}
	got, _ := s.Load("111")
	if !got.SharesWithPool() || got.PoolSharing.Basis != PoolSharingBasisUserChoice || got.PoolSharing.DecidedAt.IsZero() {
		t.Fatalf("accord mal enregistré: %+v", got.PoolSharing)
	}
	if got.Gamertag != "Alice" || got.OAuthRefreshToken != "rt-a" {
		t.Fatalf("les autres champs doivent être préservés: %+v", got)
	}

	if err := s.SetPoolSharing("111", false, PoolSharingBasisUserChoice); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	got, _ = s.Load("111")
	if got.SharesWithPool() {
		t.Fatal("le retrait doit couper le partage")
	}
	if got.PoolSharing == nil {
		t.Fatal("un retrait reste une décision enregistrée, pas une absence de décision")
	}
}

func TestSetPoolSharing_RefusesUnknownUserAndEmptyBasis(t *testing.T) {
	s := newSharingStore(t, &UserTokens{XUID: "111", OAuthRefreshToken: "rt"})

	if err := s.SetPoolSharing("999", true, PoolSharingBasisUserChoice); !errors.Is(err, ErrUserTokensNotFound) {
		t.Fatalf("xuid inconnu: want ErrUserTokensNotFound, got %v", err)
	}
	if _, err := s.Load("999"); !errors.Is(err, ErrUserTokensNotFound) {
		t.Fatal("une décision ne doit jamais créer de fichier de jetons")
	}
	if err := s.SetPoolSharing("111", true, ""); err == nil {
		t.Fatal("une décision sans base doit être refusée")
	}
	if err := s.SetPoolSharing("../x", true, PoolSharingBasisUserChoice); err == nil {
		t.Fatal("xuid non sûr doit être refusé")
	}
}

func TestUpsert_PartialWriteKeepsPoolSharing(t *testing.T) {
	s := newSharingStore(t, &UserTokens{XUID: "111", OAuthRefreshToken: "rt"})
	if err := s.SetPoolSharing("111", true, PoolSharingBasisUserChoice); err != nil {
		t.Fatal(err)
	}

	// Upsert partiel (rafraîchissement XSTS) : ne porte pas PoolSharing.
	if err := s.Upsert(&UserTokens{XUID: "111", XSTSToken: "xsts"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load("111")
	if !got.SharesWithPool() {
		t.Fatal("un Upsert partiel ne doit pas effacer le consentement")
	}

	// Un retrait n'est pas non plus annulé par un Upsert partiel.
	if err := s.SetPoolSharing("111", false, PoolSharingBasisUserChoice); err != nil {
		t.Fatal(err)
	}
	if err := s.Upsert(&UserTokens{XUID: "111", XSTSToken: "xsts2"}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Load("111")
	if got.SharesWithPool() || got.PoolSharing == nil {
		t.Fatalf("le retrait doit survivre à un Upsert partiel: %+v", got.PoolSharing)
	}
}

func TestGrandfatherPoolSharing(t *testing.T) {
	s := newSharingStore(t,
		&UserTokens{XUID: "1", Gamertag: "Undecided", OAuthRefreshToken: "rt1"},
		&UserTokens{XUID: "2", Gamertag: "Refused", OAuthRefreshToken: "rt2"},
		&UserTokens{XUID: "3", Gamertag: "NoToken"},
		&UserTokens{XUID: "4", Gamertag: "Granted", OAuthRefreshToken: "rt4"},
	)
	if err := s.SetPoolSharing("2", false, PoolSharingBasisUserChoice); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPoolSharing("4", true, PoolSharingBasisUserChoice); err != nil {
		t.Fatal(err)
	}

	n, err := s.GrandfatherPoolSharing(PoolSharingBasisGrandfathered)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("seul le joueur sans décision et avec jeton est marqué, got %d", n)
	}

	u1, _ := s.Load("1")
	if !u1.SharesWithPool() || u1.PoolSharing.Basis != PoolSharingBasisGrandfathered {
		t.Fatalf("joueur sans décision: %+v", u1.PoolSharing)
	}
	u2, _ := s.Load("2")
	if u2.SharesWithPool() {
		t.Fatal("un refus enregistré ne doit jamais être écrasé")
	}
	u3, _ := s.Load("3")
	if u3.PoolSharing != nil {
		t.Fatal("un joueur sans jeton n'est pas marqué")
	}
	u4, _ := s.Load("4")
	if u4.PoolSharing.Basis != PoolSharingBasisUserChoice {
		t.Fatal("une décision existante garde sa base")
	}

	again, err := s.GrandfatherPoolSharing(PoolSharingBasisGrandfathered)
	if err != nil || again != 0 {
		t.Fatalf("idempotent: want 0 got %d err=%v", again, err)
	}
}
