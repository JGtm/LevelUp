// Package auth — xsts_provenance_test.go : provenance du RpsTicket sur le chemin
// XSTS RTA (watcher). Le succès complet dépend d'un appel réseau Xbox Live réel :
// on teste ce qui est testable sans réseau — la dégradation propre et l'absence
// d'écriture quand rien n'a été mesuré.
package auth

import (
	"context"
	"testing"
)

// TestAcquireXSTSForRTAWithProvenance_DegradeSansStore : sans store ni xuid, le
// helper se comporte exactement comme AcquireXSTSForRTA (aucune panique, aucune
// écriture). L'appel réseau échoue en test, ce qui est l'issue attendue ici.
func TestAcquireXSTSForRTAWithProvenance_DegradeSansStore(t *testing.T) {
	if _, err := AcquireXSTSForRTAWithProvenance(context.Background(), nil, "", "at-bidon"); err == nil {
		t.Fatal("attendu échec réseau (endpoint Xbox injoignable en test)")
	}
	store := newTestMultiUserStore(t)
	if _, err := AcquireXSTSForRTAWithProvenance(context.Background(), store, "", "at-bidon"); err == nil {
		t.Fatal("attendu échec réseau (xuid vide)")
	}
}

// TestAcquireXSTSForRTAWithProvenance_EchecNEcritRien : un échange qui ne mesure
// rien (échec amont) ne doit PAS toucher la provenance persistée — sinon un
// incident réseau effacerait une mesure valide.
func TestAcquireXSTSForRTAWithProvenance_EchecNEcritRien(t *testing.T) {
	store := newTestMultiUserStore(t)
	if err := store.Upsert(&UserTokens{
		XUID: "12345", Gamertag: "TestUser",
		TokenClientFamily: TokenFamilyXboxNative,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := AcquireXSTSForRTAWithProvenance(context.Background(), store, "12345", "at-bidon"); err == nil {
		t.Fatal("attendu échec réseau (endpoint Xbox injoignable en test)")
	}

	got, err := store.Load("12345")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.TokenClientFamily != TokenFamilyXboxNative {
		t.Errorf("provenance = %q, want %q — un échange raté a écrasé la mesure", got.TokenClientFamily, TokenFamilyXboxNative)
	}
}
