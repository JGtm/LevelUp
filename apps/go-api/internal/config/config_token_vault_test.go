package config

import (
	"errors"
	"runtime"
	"testing"

	"levelup/go-api/internal/platform/auth/tokencipher"
)

// TestTokenVault_LeProfilChoisitLeChiffreur : le chargement de la configuration pose le chiffreur du
// processus d'après la capacité TokenVault, jamais d'après GOOS seul. Le profil desktop demande le
// coffre ; hors Windows il est indisponible (ProcessError) mais le chargement n'échoue pas ; tout
// autre profil est plain, et le dernier profil résolu l'emporte.
func TestTokenVault_LeProfilChoisitLeChiffreur(t *testing.T) {
	t.Cleanup(func() { tokencipher.InstallForProfile(false) })

	for _, c := range []struct{ nom, launch, env, demo string }{
		{"hosted", LaunchModeDefault, "production", ""},
		{"dev", LaunchModeDefault, "", ""},
		{"demo", LaunchModeDefault, "", "true"},
	} {
		t.Run(c.nom, func(t *testing.T) {
			cfg, err := chargerServeur(t, c.launch, c.env, c.demo, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Capabilities.TokenVault {
				t.Fatal("seul le profil desktop offre le coffre")
			}
			if tokencipher.Process().Name() != tokencipher.NamePlain || tokencipher.ProcessError() != nil {
				t.Fatalf("chiffreur = %s, %v", tokencipher.Process().Name(), tokencipher.ProcessError())
			}
		})
	}

	for _, launch := range []string{LaunchModeDesktop, LaunchModeRefreshOnly} {
		t.Run("desktop "+launch, func(t *testing.T) {
			cfg, err := chargerServeur(t, launch, "", "", "", "")
			if err != nil {
				t.Fatalf("le chargement du profil desktop ne doit jamais échouer sur le coffre : %v", err)
			}
			if !cfg.Capabilities.TokenVault {
				t.Fatal("le profil desktop offre le coffre")
			}
			if runtime.GOOS == "windows" {
				if tokencipher.Process().Name() != tokencipher.NameDPAPI || tokencipher.ProcessError() != nil {
					t.Fatalf("chiffreur = %s, %v", tokencipher.Process().Name(), tokencipher.ProcessError())
				}
			} else if !errors.Is(tokencipher.ProcessError(), tokencipher.ErrUnavailable) {
				t.Fatalf("hors Windows, ProcessError = %v", tokencipher.ProcessError())
			}
		})
	}

	if _, err := chargerServeur(t, LaunchModeDefault, "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if tokencipher.Process().Name() != tokencipher.NamePlain || tokencipher.ProcessError() != nil {
		t.Fatal("un chargement hébergé ou de développement remet le chiffreur plain")
	}
}
