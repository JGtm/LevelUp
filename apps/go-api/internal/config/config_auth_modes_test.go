package config

import (
	"strings"
	"testing"
)

// TestAuthModes_NormalizedAndValidated : LEVELUP_AUTH_MODE et LEVELUP_REGISTRATION
// sont lus en minuscules, espaces de bord retirés ; une valeur inconnue est refusée au
// démarrage (Validate), quel que soit l'environnement. Registre sécurité C5-5 : avec
// `Xbox`, la propriété des joueurs s'éteignait sans un mot.
func TestAuthModes_NormalizedAndValidated(t *testing.T) {
	cases := []struct {
		name, auth, registration   string
		wantAuth, wantRegistration string
		wantErr                    string
	}{
		{name: "défauts", wantAuth: "none", wantRegistration: "invite"},
		{name: "prod", auth: "xbox", registration: "invite", wantAuth: "xbox", wantRegistration: "invite"},
		{name: "casse xbox", auth: "Xbox", wantAuth: "xbox", wantRegistration: "invite"},
		{name: "espaces et majuscules", auth: "  PASSWORD ", registration: " Open", wantAuth: "password", wantRegistration: "open"},
		{name: "closed", auth: "none", registration: "CLOSED", wantAuth: "none", wantRegistration: "closed"},
		{name: "blanc seul = défaut", auth: "   ", registration: "  ", wantAuth: "none", wantRegistration: "invite"},
		{name: "mode inconnu", auth: "sso", wantErr: "LEVELUP_AUTH_MODE"},
		{name: "faute de frappe", auth: "xbx", wantErr: "LEVELUP_AUTH_MODE"},
		{name: "inscription inconnue", auth: "xbox", registration: "public", wantErr: "LEVELUP_REGISTRATION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
			t.Setenv("LEVELUP_AUTH_MODE", tc.auth)
			t.Setenv("LEVELUP_REGISTRATION", tc.registration)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load : %v", err)
			}
			verr := cfg.Validate()
			if tc.wantErr != "" {
				if verr == nil || !strings.Contains(verr.Error(), tc.wantErr) {
					t.Fatalf("Validate = %v, attendu un refus citant %s", verr, tc.wantErr)
				}
				return
			}
			if verr != nil {
				t.Fatalf("Validate = %v, attendu nil (hors production)", verr)
			}
			if cfg.AuthMode != tc.wantAuth || cfg.RegistrationMode != tc.wantRegistration {
				t.Errorf("modes = (%q, %q), attendu (%q, %q)", cfg.AuthMode, cfg.RegistrationMode, tc.wantAuth, tc.wantRegistration)
			}
		})
	}
}
