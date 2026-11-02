//go:build windows

package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth/tokencipher"
)

// TestVault_FichierScelleParLeCoffreWindows : le magasin au chiffreur DPAPI réel migre un fichier
// plain à l'écriture, n'écrit aucun secret en clair, relit, et le refus d'un magasin plain sur ce
// fichier est typé.
func TestVault_FichierScelleParLeCoffreWindows(t *testing.T) {
	dpapi, err := tokencipher.NewDPAPI()
	if err != nil {
		t.Fatal(err)
	}
	dir := tempTokenDir(t)

	plain := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
	if err := plain.Upsert(seedTokens("111")); err != nil {
		t.Fatal(err)
	}
	vault := NewMultiUserTokenStoreWithCipher(dir, dpapi)
	if got, err := vault.Load("111"); err != nil || got.OAuthRefreshToken != secretRT {
		t.Fatalf("lecture d'un fichier plain par le coffre : %+v, %v", got, err)
	}
	if err := vault.UpdateTokenClientFamily("111", TokenFamilyXboxNative); err != nil {
		t.Fatal(err)
	}

	raw := readFileString(t, dir, "111")
	for _, secret := range []string{secretRT, secretAccess, secretXSTS, secretHash} {
		if strings.Contains(raw, secret) {
			t.Fatalf("secret %q en clair après migration :\n%s", secret, raw)
		}
	}
	if !strings.Contains(raw, `"cipher": "dpapi"`) {
		t.Fatalf("champ cipher absent :\n%s", raw)
	}
	if got, err := vault.Load("111"); err != nil || got.OAuthRefreshToken != secretRT ||
		got.XSTSToken != secretXSTS || got.TokenClientFamily != TokenFamilyXboxNative {
		t.Fatalf("aller-retour DPAPI : %+v, %v", got, err)
	}
	if _, err := plain.Load("111"); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("magasin plain sur fichier DPAPI = %v, attendu ErrTokenFileSealed", err)
	}
}

// TestVault_EtatDuWatcherScelleParLeCoffreWindows : le fichier d'état du watcher passe par le même
// DPAPI : aucun secret en clair après la migration, aller-retour, refus typé d'un magasin plain.
func TestVault_EtatDuWatcherScelleParLeCoffreWindows(t *testing.T) {
	dpapi, err := tokencipher.NewDPAPI()
	if err != nil {
		t.Fatal(err)
	}
	p := tempWatcherStatePath(t)
	if err := NewTokenStoreWithCipher(p, tokencipher.Plain()).Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}
	vault := NewTokenStoreWithCipher(p, dpapi)
	if got, err := vault.Load(); err != nil || got.XSTSToken != secretXSTS {
		t.Fatalf("lecture d'un fichier plain par le coffre : %+v, %v", got, err)
	}
	if err := vault.UpdateOAuth(strings.Repeat("n", 20), time.Hour); err != nil {
		t.Fatal(err)
	}
	raw := readPath(t, p)
	for _, secret := range []string{secretXSTS, secretHash, secretAccess, strings.Repeat("n", 20)} {
		if strings.Contains(raw, secret) {
			t.Fatalf("secret %q en clair après migration :\n%s", secret, raw)
		}
	}
	if !strings.Contains(raw, `"cipher": "dpapi"`) {
		t.Fatalf("champ cipher absent :\n%s", raw)
	}
	if got, err := vault.Load(); err != nil || got.AccessToken != strings.Repeat("n", 20) || got.XSTSUserHash != secretHash {
		t.Fatalf("aller-retour DPAPI : %+v, %v", got, err)
	}
	if _, err := NewTokenStoreWithCipher(p, tokencipher.Plain()).Load(); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("magasin plain sur état DPAPI = %v, attendu ErrTokenFileSealed", err)
	}
}
