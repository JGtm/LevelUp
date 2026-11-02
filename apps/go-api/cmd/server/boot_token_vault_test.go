package main

import (
	"errors"
	"testing"

	"levelup/go-api/internal/config"
)

// TestTokenVaultRefusal : un profil qui exige le coffre des jetons ne démarre pas sans lui (le profil
// desktop est Windows seul, ADR 0045 D-2) ; un profil qui ne l'exige pas ignore l'échec.
func TestTokenVaultRefusal(t *testing.T) {
	indispo := errors.New("coffre indisponible")
	avec := &config.AppConfig{}
	avec.Capabilities.TokenVault = true
	sans := &config.AppConfig{}

	if err := tokenVaultRefusal(avec, indispo); !errors.Is(err, indispo) {
		t.Errorf("coffre exigé et indisponible : %v, attendu un refus", err)
	}
	if err := tokenVaultRefusal(avec, nil); err != nil {
		t.Errorf("coffre exigé et disponible : %v", err)
	}
	if err := tokenVaultRefusal(sans, indispo); err != nil {
		t.Errorf("coffre non exigé : %v", err)
	}
}
