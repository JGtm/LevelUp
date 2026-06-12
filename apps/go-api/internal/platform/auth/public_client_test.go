package auth

import "testing"

// TestIsPublicAzureClient verrouille la décision secret/no-secret : les apps publiques
// connues (LevelUp, halo-tools) ne doivent jamais recevoir de client_secret (AADSTS90023) ;
// tout autre client_id est présumé confidentiel.
func TestIsPublicAzureClient(t *testing.T) {
	cases := []struct {
		name     string
		clientID string
		want     bool
	}{
		{"LevelUp (public)", LevelUpClientID, true},
		{"halo-tools (public, défaut token-capture)", HaloToolsClientID, true},
		{"app confidentielle tierce", "16ed74ce-5373-4b39-8d66-f0d1c5f9c8fe", false},
		{"client_id vide", "", false},
	}
	for _, c := range cases {
		if got := IsPublicAzureClient(c.clientID); got != c.want {
			t.Errorf("%s: IsPublicAzureClient(%q) = %v, want %v", c.name, c.clientID, got, c.want)
		}
	}
}
