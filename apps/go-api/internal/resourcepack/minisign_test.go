package resourcepack

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La signature de testdata est produite par `tauri signer sign` (la vraie chaîne de livraison) : ce
// test prouve que le vérificateur lit ce que l'outil écrit, pas seulement ce que le test fabrique.
func TestVerifieLaSignatureProduiteParTauri(t *testing.T) {
	pub, _ := os.ReadFile(filepath.Join("testdata", "tauri_test_pubkey.txt"))
	sig, _ := os.ReadFile(filepath.Join("testdata", "signed_payload.bin.sig"))
	key, err := ParsePublicKey(strings.TrimSpace(string(pub)))
	if err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join("testdata", "signed_payload.bin")
	if err := key.VerifyFile(payload, string(sig)); err != nil {
		t.Fatalf("signature de Tauri refusée : %v", err)
	}
	altered := filepath.Join(t.TempDir(), "p.bin")
	if err := os.WriteFile(altered, []byte("hello pack!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := key.VerifyFile(altered, string(sig)); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("contenu altéré : %v", err)
	}
}

func TestSignatureDeTestEtCleEtrangere(t *testing.T) {
	k, other := newTestKey(t), newTestKey(t)
	f := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(f, []byte("contenu"), 0o644); err != nil {
		t.Fatal(err)
	}
	sig := k.Sign(t, f)
	pk, err := ParsePublicKey(k.Public())
	if err != nil {
		t.Fatal(err)
	}
	if err := pk.VerifyFile(f, sig); err != nil {
		t.Fatalf("signature valide refusée : %v", err)
	}
	foreign, _ := ParsePublicKey(other.Public())
	if err := foreign.VerifyFile(f, sig); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("clé étrangère : %v", err)
	}
	for _, bad := range []string{"", "pas du base64 !", "AAAA"} {
		if err := pk.VerifyFile(f, bad); err == nil {
			t.Fatalf("signature %q acceptée", bad)
		}
	}
	if _, err := ParsePublicKey("AAAA"); err == nil {
		t.Fatal("clé publique mal formée acceptée")
	}
}
