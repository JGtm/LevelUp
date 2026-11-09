package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/service"
)

const dcXUID = "2535471234567890"

func dcStore(t *testing.T) *auth.MultiUserTokenStore {
	t.Helper()
	return auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
}

func dcAttempt() *auth.Attempt {
	return &auth.Attempt{
		Gamertag:             "Spartan42",
		XUID:                 dcXUID,
		MicrosoftAccessToken: "ms-access",
		OAuthRefreshToken:    "rt-frais",
		XSTSRTAToken:         "xsts-rta",
		XSTSRTAUserHash:      "uhs",
		XSTSRTAExpiresAt:     time.Now().Add(55 * time.Minute),
	}
}

func gateFor(tracked *bool, seen *[]string) domain.ProfileGate {
	return func(_ context.Context, titleSlug, xuid string) bool {
		if seen != nil {
			*seen = append(*seen, titleSlug+"/"+xuid)
		}
		return *tracked
	}
}

func TestAuthCompletion_Persist_EcritLeJetonDeRafraichissement(t *testing.T) {
	store := dcStore(t)
	c := service.NewAuthCompletion(store, nil)

	ok, err := c.Persist(context.Background(), dcAttempt())
	if err != nil || !ok {
		t.Fatalf("Persist = (%v, %v), attendu (true, nil)", ok, err)
	}
	stored, err := store.Load(dcXUID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.OAuthRefreshToken != "rt-frais" || stored.XSTSToken != "xsts-rta" || stored.AccessToken != "ms-access" {
		t.Errorf("jetons persistés incomplets : %+v", stored)
	}
	if stored.PoolSharing != nil {
		t.Errorf("aucune décision de mise en commun ne doit être écrite (ADR 0038), reçu %+v", stored.PoolSharing)
	}
}

func TestAuthCompletion_Persist_RTSeulSansXSTS(t *testing.T) {
	store := dcStore(t)
	c := service.NewAuthCompletion(store, nil)
	a := dcAttempt()
	a.XSTSRTAToken = "" // AcquireXSTSForRTA a échoué : le RT, lui, est durable.

	if ok, err := c.Persist(context.Background(), a); err != nil || !ok {
		t.Fatalf("Persist = (%v, %v), attendu (true, nil)", ok, err)
	}
	stored, err := store.Load(dcXUID)
	if err != nil || stored.OAuthRefreshToken != "rt-frais" {
		t.Fatalf("le RT doit être persisté sans XSTS RTA : %+v, %v", stored, err)
	}
}

func TestAuthCompletion_Persist_RienAEcrire(t *testing.T) {
	store := dcStore(t)
	c := service.NewAuthCompletion(store, nil)
	a := dcAttempt()
	a.XSTSRTAToken, a.OAuthRefreshToken = "", ""

	if ok, err := c.Persist(context.Background(), a); ok || err != nil {
		t.Fatalf("Persist = (%v, %v), attendu (false, nil)", ok, err)
	}
	if _, err := store.Load(dcXUID); err == nil {
		t.Error("rien ne doit être écrit sans jeton")
	}
}

func TestAuthCompletion_Persist_PreserveLeRTExistant(t *testing.T) {
	store := dcStore(t)
	if err := store.Upsert(&auth.UserTokens{XUID: dcXUID, Gamertag: "Spartan42", OAuthRefreshToken: "rt-ancien"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	c := service.NewAuthCompletion(store, nil)
	a := dcAttempt()
	a.OAuthRefreshToken = ""

	if _, err := c.Persist(context.Background(), a); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	stored, _ := store.Load(dcXUID)
	if stored.OAuthRefreshToken != "rt-ancien" {
		t.Errorf("RT = %q, le RT existant doit être préservé", stored.OAuthRefreshToken)
	}
	if stored.XSTSToken != "xsts-rta" {
		t.Errorf("XSTS = %q, doit être mis à jour", stored.XSTSToken)
	}
}

func TestAuthCompletion_ProfilSuivi_EcritImmediatement(t *testing.T) {
	store := dcStore(t)
	tracked := true
	var seen []string
	c := service.NewAuthCompletion(store, gateFor(&tracked, &seen))

	if !c.CompleteForTrackedProfile(context.Background(), dcAttempt()) {
		t.Fatal("profil suivi : le magasin doit être écrit")
	}
	stored, err := store.Load(dcXUID)
	if err != nil || stored.OAuthRefreshToken != "rt-frais" {
		t.Fatalf("RT non persisté : %+v, %v", stored, err)
	}
	if len(seen) != 1 || seen[0] != "halo_infinite/"+dcXUID {
		t.Errorf("la porte doit être interrogée par (titre par défaut, xuid), reçu %v", seen)
	}
}

func TestAuthCompletion_ProfilNonSuivi_RienSurDisqueJusquAuProfil(t *testing.T) {
	store := dcStore(t)
	tracked := false
	c := service.NewAuthCompletion(store, gateFor(&tracked, nil))

	if c.CompleteForTrackedProfile(context.Background(), dcAttempt()) {
		t.Fatal("profil non suivi : rien ne doit être écrit")
	}
	if _, err := store.Load(dcXUID); err == nil {
		t.Fatal("le magasin ne doit contenir aucun fichier pour un profil non suivi")
	}

	// Le wizard déclare ensuite le profil : l'annuaire rejoue les jetons.
	c.SeedCredentials(context.Background(), dcXUID)
	stored, err := store.Load(dcXUID)
	if err != nil || stored.OAuthRefreshToken != "rt-frais" {
		t.Fatalf("jetons non rejoués à la création du profil : %+v, %v", stored, err)
	}

	// Rejouer deux fois ne plante pas (les jetons ont été consommés).
	c.SeedCredentials(context.Background(), dcXUID)
}

func TestAuthCompletion_SeedSansJetonEnAttente(t *testing.T) {
	store := dcStore(t)
	c := service.NewAuthCompletion(store, nil)

	c.SeedCredentials(context.Background(), dcXUID)
	if _, err := store.Load(dcXUID); err == nil {
		t.Error("aucun jeton en attente : rien ne doit être écrit")
	}
}

func TestAuthCompletion_SeedNeServtPasUnAutreXUID(t *testing.T) {
	store := dcStore(t)
	tracked := false
	c := service.NewAuthCompletion(store, gateFor(&tracked, nil))
	c.CompleteForTrackedProfile(context.Background(), dcAttempt())

	c.SeedCredentials(context.Background(), "2535400000000099")
	if _, err := store.Load(dcXUID); err == nil {
		t.Error("le profil d'un autre xuid ne doit pas déclencher l'écriture")
	}
	if _, err := store.Load("2535400000000099"); err == nil {
		t.Error("aucun fichier ne doit exister pour l'autre xuid")
	}
}

func TestAuthCompletion_SansGate_GardeEnAttente(t *testing.T) {
	store := dcStore(t)
	c := service.NewAuthCompletion(store, nil)

	if c.CompleteForTrackedProfile(context.Background(), dcAttempt()) {
		t.Fatal("sans porte, le défaut sûr est de ne rien écrire")
	}
	if _, err := store.Load(dcXUID); err == nil {
		t.Error("rien ne doit être écrit sans porte")
	}
}

// stubInner : stratégie interne programmable.
type stubInner struct {
	err   error
	calls int
}

func (s *stubInner) OnAuthSuccess(context.Context, *auth.Attempt, *domain.SessionData) error {
	s.calls++
	return s.err
}

func TestDeviceCodeLinkStrategy_ModeNone_SessionSansCompteEtProfilSuivi(t *testing.T) {
	store := dcStore(t)
	tracked := true
	inner := &stubInner{err: auth.ErrSessionNotAuthenticated} // mode "none" : aucun compte local
	s := service.NewAuthLinkStrategy(inner, service.NewAuthCompletion(store, gateFor(&tracked, nil)))

	err := s.OnAuthSuccess(context.Background(), dcAttempt(), &domain.SessionData{})
	if !errors.Is(err, auth.ErrSessionNotAuthenticated) {
		t.Fatalf("la stratégie interne doit être répercutée, reçu %v", err)
	}
	stored, lerr := store.Load(dcXUID)
	if lerr != nil || stored.OAuthRefreshToken != "rt-frais" {
		t.Fatalf("mode none : le RT doit être persisté malgré ErrSessionNotAuthenticated : %+v, %v", stored, lerr)
	}
}

func TestDeviceCodeLinkStrategy_RefusInterne_NEcritRien(t *testing.T) {
	store := dcStore(t)
	tracked := true
	inner := &stubInner{err: auth.ErrInstanceLocked}
	s := service.NewAuthLinkStrategy(inner, service.NewAuthCompletion(store, gateFor(&tracked, nil)))

	err := s.OnAuthSuccess(context.Background(), dcAttempt(), &domain.SessionData{})
	if !errors.Is(err, auth.ErrInstanceLocked) {
		t.Fatalf("le refus doit être répercuté, reçu %v", err)
	}
	if _, lerr := store.Load(dcXUID); lerr == nil {
		t.Error("une connexion refusée ne doit rien écrire dans le magasin")
	}
}
