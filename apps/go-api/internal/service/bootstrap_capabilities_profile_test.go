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
		caps.CanUseDiscord || !caps.CanUpdateApp || caps.CanOfferWindowsDownload || !caps.CanUseCompactChrome {
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

// TestBuildCapabilities_ChromeDeBureauSeulementEnDesktop : la capacité de chrome compact et la
// section app_info (version, racine des données) ne sortent que du profil desktop ; les trois
// autres profils ne publient ni l'une ni l'autre (une instance hébergée ne donne pas le chemin
// de ses données).
func TestBuildCapabilities_ChromeDeBureauSeulementEnDesktop(t *testing.T) {
	cas := []struct {
		nom        string
		launch     string
		env        string
		demo       string
		wantChrome bool
	}{
		{"hosted", config.LaunchModeDefault, "production", "", false},
		{"dev", config.LaunchModeDefault, "", "", false},
		{"demo", config.LaunchModeDefault, "", "true", false},
		{"desktop", config.LaunchModeDesktop, "", "", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cfg := chargerPourBootstrap(t, c.launch, c.env, c.demo, "")
			cfg.AppVersion = "9.8.7"
			if got := buildCapabilities(cfg, map[string]interface{}{}).CanUseCompactChrome; got != c.wantChrome {
				t.Errorf("can_use_compact_chrome = %v, attendu %v", got, c.wantChrome)
			}
			info := buildAppInfo(cfg)
			if (info != nil) != c.wantChrome {
				t.Fatalf("app_info publié = %v, attendu %v", info != nil, c.wantChrome)
			}
			if info != nil && (info.Version != "9.8.7" || info.DataRoot != cfg.Roots().Data || info.DataRoot == "") {
				t.Errorf("app_info = %+v, attendu version 9.8.7 et la racine des données %q", info, cfg.Roots().Data)
			}
		})
	}
}

// TestBuildCapabilities_ConnexionNavigateurEtComparaisonDeComptes : la connexion par le navigateur
// n'est offerte que par le profil desktop ; la comparaison de comptes l'est partout SAUF en desktop
// (un seul compte suivi).
func TestBuildCapabilities_ConnexionNavigateurEtComparaisonDeComptes(t *testing.T) {
	cas := []struct {
		nom         string
		launch      string
		env         string
		demo        string
		wantBrowser bool
		wantCompare bool
	}{
		{"hosted", config.LaunchModeDefault, "production", "", false, true},
		{"dev", config.LaunchModeDefault, "", "", false, true},
		{"demo", config.LaunchModeDefault, "", "true", false, true},
		{"desktop", config.LaunchModeDesktop, "", "", true, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cfg := chargerPourBootstrap(t, c.launch, c.env, c.demo, "")
			caps := buildCapabilities(cfg, map[string]interface{}{})
			if caps.CanLoginViaBrowser != c.wantBrowser || caps.CanCompareAccounts != c.wantCompare {
				t.Errorf("can_login_via_browser = %v, can_compare_accounts = %v ; attendu %v, %v",
					caps.CanLoginViaBrowser, caps.CanCompareAccounts, c.wantBrowser, c.wantCompare)
			}
		})
	}
}
