package tokencipher

import (
	"bytes"
	"errors"
	"runtime"
	"testing"
)

func TestPlain_OctetsInchanges(t *testing.T) {
	c := Plain()
	if c.Name() != NamePlain {
		t.Fatalf("nom = %q", c.Name())
	}
	in := []byte("jeton")
	sealed, err := c.Seal(in)
	if err != nil || !bytes.Equal(sealed, in) {
		t.Fatalf("Seal = %q, %v", sealed, err)
	}
	sealed[0] = 'X'
	if in[0] != 'j' {
		t.Error("Seal ne doit pas aliaser l'entrée")
	}
	if out, err := c.Open([]byte("jeton")); err != nil || string(out) != "jeton" {
		t.Fatalf("Open = %q, %v", out, err)
	}
}

func TestForVault_ParLeProfil(t *testing.T) {
	c, err := ForVault(false)
	if err != nil || c.Name() != NamePlain {
		t.Fatalf("sans coffre : %v, %v", c, err)
	}
	c, err = ForVault(true)
	if runtime.GOOS == "windows" {
		if err != nil || c.Name() != NameDPAPI {
			t.Fatalf("coffre sous Windows : %v, %v", c, err)
		}
		return
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("coffre hors Windows = %v, attendu ErrUnavailable", err)
	}
}

// TestInstallForProfile : le profil choisit, le dernier profil résolu l'emporte, l'indisponibilité du
// coffre n'est jamais une panique et se lit par ProcessError.
func TestInstallForProfile(t *testing.T) {
	t.Cleanup(func() { InstallForProfile(false) })

	InstallForProfile(false)
	if Process().Name() != NamePlain || ProcessError() != nil {
		t.Fatalf("profil sans coffre : %s, %v", Process().Name(), ProcessError())
	}
	InstallForProfile(true)
	if runtime.GOOS == "windows" {
		if Process().Name() != NameDPAPI || ProcessError() != nil {
			t.Fatalf("profil avec coffre sous Windows : %s, %v", Process().Name(), ProcessError())
		}
	} else if Process().Name() != NamePlain || !errors.Is(ProcessError(), ErrUnavailable) {
		t.Fatalf("profil avec coffre hors Windows : %s, %v", Process().Name(), ProcessError())
	}
	InstallForProfile(false)
	if Process().Name() != NamePlain || ProcessError() != nil {
		t.Fatal("le dernier profil résolu l'emporte")
	}
}
