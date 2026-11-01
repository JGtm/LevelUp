package service_test

// devicecode_completion_poolgrant_test.go — la mise en commun accordée d'office à l'utilisateur
// local d'une installation desktop (ADR 0045 D-8, lot L13.3) : à l'écriture du jeton, jamais sur
// une décision existante, jamais sur une instance qui ne l'a pas demandé.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/service"
)

func TestDeviceCodeCompletion_AccordLocal_PoseEnEcrivantLeJeton(t *testing.T) {
	store := dcStore(t)
	c := service.NewDeviceCodeCompletion(store, nil).WithLocalPoolGrant(true)
	before := time.Now().UTC().Add(-time.Second)

	if ok, err := c.Persist(context.Background(), dcAttempt()); err != nil || !ok {
		t.Fatalf("Persist = (%v, %v), attendu (true, nil)", ok, err)
	}
	stored, err := store.Load(dcXUID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ps := stored.PoolSharing
	if ps == nil || !ps.Granted || ps.Basis != auth.PoolSharingBasisDesktopSingleUser || ps.DecidedAt.Before(before) {
		t.Fatalf("décision = %+v, attendu un accord daté, base %q", ps, auth.PoolSharingBasisDesktopSingleUser)
	}
	if !stored.SharesWithPool() {
		t.Error("le jeton n'entrerait pas dans le round-robin public malgré l'accord")
	}
	if got := store.PoolSharingState(dcXUID); got != domain.PoolSharingGranted {
		t.Errorf("état = %q, attendu %q", got, domain.PoolSharingGranted)
	}
}

// Le jeton qui attend la création du profil reçoit lui aussi l'accord, au moment où il est écrit.
func TestDeviceCodeCompletion_AccordLocal_PoseAuReplayDuJetonEnAttente(t *testing.T) {
	store := dcStore(t)
	tracked := false
	c := service.NewDeviceCodeCompletion(store, gateFor(&tracked, nil)).WithLocalPoolGrant(true)
	c.CompleteForTrackedProfile(context.Background(), dcAttempt())
	if _, err := store.Load(dcXUID); err == nil {
		t.Fatal("le jeton a été écrit pour un profil non suivi")
	}
	c.SeedCredentials(context.Background(), dcXUID)
	stored, err := store.Load(dcXUID)
	if err != nil || stored.PoolSharing == nil || !stored.PoolSharing.Granted {
		t.Fatalf("jeton rejoué = %+v (%v), attendu avec l'accord local", stored, err)
	}
}

// Une décision existante n'est jamais écrasée : un refus reste un refus, un accord garde sa base.
func TestDeviceCodeCompletion_AccordLocal_NEcraseJamaisUneDecision(t *testing.T) {
	for _, granted := range []bool{false, true} {
		store := dcStore(t)
		if err := store.Upsert(&auth.UserTokens{XUID: dcXUID, Gamertag: "Spartan42", OAuthRefreshToken: "rt-ancien"}); err != nil {
			t.Fatal(err)
		}
		if err := store.SetPoolSharing(dcXUID, granted, auth.PoolSharingBasisUserChoice); err != nil {
			t.Fatal(err)
		}
		c := service.NewDeviceCodeCompletion(store, nil).WithLocalPoolGrant(true)
		if _, err := c.Persist(context.Background(), dcAttempt()); err != nil {
			t.Fatal(err)
		}
		stored, err := store.Load(dcXUID)
		if err != nil || stored.PoolSharing == nil {
			t.Fatalf("granted=%v : décision perdue (%+v, %v)", granted, stored, err)
		}
		if stored.PoolSharing.Granted != granted || stored.PoolSharing.Basis != auth.PoolSharingBasisUserChoice {
			t.Errorf("granted=%v : décision = %+v, attendu inchangée (base %q)", granted, stored.PoolSharing, auth.PoolSharingBasisUserChoice)
		}
		if stored.OAuthRefreshToken != "rt-frais" {
			t.Errorf("granted=%v : jeton de rafraîchissement = %q, attendu le nouveau", granted, stored.OAuthRefreshToken)
		}
	}
}

// Sans la capacité (instance hébergée), aucune décision n'est écrite ; sans jeton de rafraîchissement
// non plus (rien à mettre en commun).
func TestDeviceCodeCompletion_AccordLocal_AbsentSansCapaciteOuSansRT(t *testing.T) {
	store := dcStore(t)
	if _, err := service.NewDeviceCodeCompletion(store, nil).WithLocalPoolGrant(false).Persist(context.Background(), dcAttempt()); err != nil {
		t.Fatal(err)
	}
	if stored, _ := store.Load(dcXUID); stored == nil || stored.PoolSharing != nil {
		t.Errorf("sans la capacité : décision = %+v, attendu aucune", stored)
	}

	store = dcStore(t)
	a := dcAttempt()
	a.OAuthRefreshToken = ""
	if _, err := service.NewDeviceCodeCompletion(store, nil).WithLocalPoolGrant(true).Persist(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if stored, _ := store.Load(dcXUID); stored == nil || stored.PoolSharing != nil {
		t.Errorf("sans jeton de rafraîchissement : décision = %+v, attendu aucune", stored)
	}
}
