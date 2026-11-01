package service

// bootstrap_capabilities_profile_test.go — /bootstrap publie les capacités du profil de
// déploiement (ADR 0044 D-3, lot L10.2) sans rien changer aux neuf capacités d'avant : pour
// chaque combinaison LEVELUP_ENV x LEVELUP_DEMO_MODE x LEVELUP_AUTH_MODE, elles valent ce que
// valait la formule d'avant le profil (témoin T3).

import (
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// capacitesAvantLeProfil : la formule d'avant L10, recopiée à la main (cfg.DemoMode, authz.Enforced).
func capacitesAvantLeProfil(demo bool, authMode string, mediaEnabled bool) domain.CapabilityMap {
	return domain.CapabilityMap{
		CanReadLocalData:    true,
		CanRunSync:          !demo,
		CanUseLiveHalo:      !demo,
		CanManageSettings:   true,
		CanResetMediaIndex:  true,
		CanViewMedia:        mediaEnabled,
		CanSelfProvision:    !authz.Enforced(demo, authMode),
		CanStartInitialSync: !demo,
		CanManageInstance:   true,
	}
}

func chargerPourBootstrap(t *testing.T, launchMode, env, demo, auth string) *config.AppConfig {
	t.Helper()
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	t.Setenv("LEVELUP_ENV", env)
	t.Setenv("LEVELUP_DEMO_MODE", demo)
	t.Setenv("LEVELUP_AUTH_MODE", auth)
	t.Setenv("LEVELUP_API_HOST", "")
	cfg, err := config.LoadServer(launchMode)
	if err != nil {
		t.Fatalf("LoadServer : %v", err)
	}
	return cfg
}

// neufCapacites remet à zéro les capacités du profil pour comparer les neuf d'avant.
func neufCapacites(c domain.CapabilityMap) domain.CapabilityMap {
	return domain.CapabilityMap{
		CanReadLocalData: c.CanReadLocalData, CanRunSync: c.CanRunSync, CanUseLiveHalo: c.CanUseLiveHalo,
		CanManageSettings: c.CanManageSettings, CanResetMediaIndex: c.CanResetMediaIndex,
		CanViewMedia: c.CanViewMedia, CanSelfProvision: c.CanSelfProvision,
		CanStartInitialSync: c.CanStartInitialSync, CanManageInstance: c.CanManageInstance,
	}
}

func TestBuildCapabilities_CombinaisonsActuellesInchangees(t *testing.T) {
	for _, env := range []string{"", "development", "production"} {
		for _, demo := range []string{"", "true"} {
			for _, auth := range []string{"", "none", "password", "xbox"} {
				t.Run(fmt.Sprintf("env=%q/demo=%q/auth=%q", env, demo, auth), func(t *testing.T) {
					cfg := chargerPourBootstrap(t, config.LaunchModeDefault, env, demo, auth)
					caps := buildCapabilities(cfg, map[string]interface{}{})
					want := capacitesAvantLeProfil(demo == "true", cfg.AuthMode, true)
					if got := neufCapacites(caps); got != want {
						t.Errorf("neuf capacités d'avant changées :\n got %+v\nwant %+v", got, want)
					}
					if caps.Deployment != string(cfg.Deployment) || caps.Deployment == "" {
						t.Errorf("deployment = %q, attendu le nom du profil %q", caps.Deployment, cfg.Deployment)
					}
					c := cfg.Capabilities
					if caps.CanUseAccounts != c.UseAccounts || caps.CanSignInWithXbox != c.SignInWithXbox ||
						caps.CanPoolTokens != c.PoolTokens || caps.CanEditSettings != c.EditSettings ||
						caps.CanUploadMedia != c.UploadMedia || caps.CanUseRestic != c.UseResticBackup {
						t.Errorf("capacités du profil non recopiées : %+v pour %+v", caps, c)
					}
				})
			}
		}
	}
}

// TestBuildCapabilities_Desktop : l'utilisateur seul garde la synchro, les profils et la
// création de profil (aucune propriété à appliquer) ; ni compte, ni mise en commun, ni sauvegarde.
func TestBuildCapabilities_Desktop(t *testing.T) {
	cfg := chargerPourBootstrap(t, config.LaunchModeDesktop, "production", "", "xbox")
	caps := buildCapabilities(cfg, map[string]interface{}{})
	want := capacitesAvantLeProfil(false, "none", true)
	if got := neufCapacites(caps); got != want {
		t.Errorf("neuf capacités en desktop :\n got %+v\nwant %+v", got, want)
	}
	if caps.Deployment != "desktop" || caps.CanUseAccounts || caps.CanPoolTokens || caps.CanUseRestic ||
		caps.CanUseDiscord || !caps.CanUpdateApp || caps.CanOfferWindowsDownload {
		t.Errorf("capacités desktop inattendues : %+v", caps)
	}
}

// TestBootstrap_AuthModePublieSansDesktop : le contrat d'API garde none, password, xbox — le mode
// dérivé desktop se publie none.
func TestBootstrap_AuthModePublieSansDesktop(t *testing.T) {
	cfg := chargerPourBootstrap(t, config.LaunchModeDesktop, "", "", "")
	if got := cfg.PublishedAuthMode(); got != "none" || strings.Contains(got, "desktop") {
		t.Errorf("auth_mode publié = %q, attendu none", got)
	}
	cfg = chargerPourBootstrap(t, config.LaunchModeDefault, "", "", "xbox")
	if got := cfg.PublishedAuthMode(); got != "xbox" {
		t.Errorf("auth_mode publié = %q, attendu xbox", got)
	}
}
