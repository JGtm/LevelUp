package title

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultHaloAuthDescriptor_GoldenParity — oracle (a) : les valeurs du défaut
// Halo doivent correspondre byte-pour-byte aux const ACTUELLES de platform/auth.
// Toute dérive (ici ou là-bas) casse ce test → parité garantie au Contract.
func TestDefaultHaloAuthDescriptor_GoldenParity(t *testing.T) {
	d := DefaultHaloAuthDescriptor()
	cases := map[string]string{
		"xsts_audience":           d.XSTSAudience,
		"spartan_audience":        d.SpartanAudience,
		"spartan_token_url":       d.SpartanTokenURL,
		"clearance_url":           d.ClearanceURL,
		"sisu_app_id":             d.SISUAppID,
		"sisu_title_id":           d.SISUTitleID,
		"xbox_live_relying_party": d.XboxLiveRelyingParty,
	}
	want := map[string]string{
		"xsts_audience":           "https://prod.xsts.halowaypoint.com/",
		"spartan_audience":        "urn:343:s3:services",
		"spartan_token_url":       "https://settings.svc.halowaypoint.com/spartan-token",
		"clearance_url":           "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active",
		"sisu_app_id":             "000000004c20a908",
		"sisu_title_id":           "144209987",
		"xbox_live_relying_party": "http://xboxlive.com",
	}
	for k, got := range cases {
		if got != want[k] {
			t.Errorf("%s = %q, want %q (parité const platform/auth)", k, got, want[k])
		}
	}
	if len(d.OAuthScopes) != 2 || d.OAuthScopes[0] != "Xboxlive.signin" || d.OAuthScopes[1] != "Xboxlive.offline_access" {
		t.Errorf("oauth_scopes = %v, want [Xboxlive.signin Xboxlive.offline_access]", d.OAuthScopes)
	}
	// Le défaut doit toujours être valide.
	if errs := d.validate(); len(errs) != 0 {
		t.Errorf("DefaultHaloAuthDescriptor invalide : %v", errs)
	}
}

const syntheticAuthTOML = `
[meta]
title_slug = "synthetic_test_title"
schema_version = 1

[auth]
xsts_audience = "https://xsts.example.test/"
spartan_audience = "urn:example:services"
spartan_token_url = "https://settings.example.test/spartan-token"
clearance_url = "https://settings.example.test/clearance/titles/syn/active"
sisu_app_id = "deadbeef"
sisu_title_id = "999999"
xbox_live_relying_party = "http://xboxlive.com"
oauth_scopes = ["Xboxlive.signin", "Xboxlive.offline_access"]
`

// TestLoadAuthDescriptor_SyntheticRouting — oracle (b) : un titre déclare des
// valeurs DISTINCTES → le loader les route (pas de cosmétique). Un titre sans
// auth.toml → ErrAuthNotConfigured (dégradation propre, zéro fallback silencieux).
func TestLoadAuthDescriptor_SyntheticRouting(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "config", "titles", "synthetic_test_title")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.toml"), []byte(syntheticAuthTOML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	desc, err := LoadAuthDescriptor(tmp, "synthetic_test_title")
	if err != nil {
		t.Fatalf("LoadAuthDescriptor: %v", err)
	}
	if desc.XSTSAudience != "https://xsts.example.test/" {
		t.Errorf("xsts_audience = %q, want example.test (routing réel)", desc.XSTSAudience)
	}
	if desc.SpartanAudience != "urn:example:services" {
		t.Errorf("spartan_audience = %q", desc.SpartanAudience)
	}
	if desc.SISUTitleID != "999999" {
		t.Errorf("sisu_title_id = %q", desc.SISUTitleID)
	}

	// Titre sans auth.toml → ErrAuthNotConfigured.
	if _, err := LoadAuthDescriptor(tmp, "title_without_auth"); !errors.Is(err, ErrAuthNotConfigured) {
		t.Errorf("titre sans auth.toml → err = %v, want ErrAuthNotConfigured", err)
	}
}

func TestLoadAuthDescriptorFromBytes_Validation(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"meta manquant", "[auth]\nxsts_audience=\"https://a.test\"\n", "title_slug manquant"},
		{
			name: "champ requis manquant",
			doc:  "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[auth]\nxsts_audience=\"https://a.test\"\n",
			want: "manquant",
		},
		{
			name: "clearance non-https",
			doc: "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[auth]\n" +
				"xsts_audience=\"a\"\nspartan_audience=\"b\"\nspartan_token_url=\"https://a.test\"\n" +
				"clearance_url=\"http://a.test\"\nsisu_app_id=\"c\"\nsisu_title_id=\"d\"\noauth_scopes=[\"s\"]\n",
			want: "non-https",
		},
		{
			name: "scopes vides",
			doc: "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[auth]\n" +
				"xsts_audience=\"a\"\nspartan_audience=\"b\"\nspartan_token_url=\"https://a.test\"\n" +
				"clearance_url=\"https://a.test\"\nsisu_app_id=\"c\"\nsisu_title_id=\"d\"\n",
			want: "oauth_scopes vide",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadAuthDescriptorFromBytes("x.toml", []byte(tc.doc))
			if err == nil {
				t.Fatalf("attendu une erreur")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want contains %q", err.Error(), tc.want)
			}
		})
	}
}
