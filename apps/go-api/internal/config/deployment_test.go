package config

// deployment_test.go — le profil de déploiement (ADR 0044, lot L10.6) : chaque combinaison des
// réglages d'AVANT (LEVELUP_ENV x LEVELUP_DEMO_MODE x LEVELUP_AUTH_MODE) donne son profil et les
// capacités d'avant ; le profil desktop force la boucle locale, accepte la validation et laisse
// authz.Enforced faux ; mode desktop + démo est refusé.

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/authz"
)

// chargerServeur charge la configuration serveur sous un environnement contrôlé : racine de
// dépôt vide (aucun .env.local qui rejouerait une variable), variables du profil posées.
func chargerServeur(t *testing.T, launchMode, env, demo, authMode, host string) (*AppConfig, error) {
	t.Helper()
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	t.Setenv("LEVELUP_ENV", env)
	t.Setenv("LEVELUP_DEMO_MODE", demo)
	t.Setenv("LEVELUP_AUTH_MODE", authMode)
	t.Setenv("LEVELUP_API_HOST", host)
	return LoadServer(launchMode)
}

// profilAttendu : la règle d'ordre de l'ADR 0044 D-1, réécrite à la main (hors mode desktop).
func profilAttendu(env, demo string) Deployment {
	switch {
	case strings.ToLower(demo) == "true":
		return DeploymentDemo
	case env != "" && env != "development":
		return DeploymentHosted
	default:
		return DeploymentDev
	}
}

// TestDeployment_TableDesCombinaisonsActuelles : chaque combinaison d'avant donne le profil
// attendu, laisse DemoMode / AuthMode / APIHost tels qu'avant, et des capacités qui suivent
// exactement les anciennes lectures (comptes = authentification password ou xbox, réglages
// et dépôt figés en démo seulement).
func TestDeployment_TableDesCombinaisonsActuelles(t *testing.T) {
	envs := []string{"", "development", "production", "staging"}
	demos := []string{"", "false", "true", "TRUE"}
	auths := []string{"", "none", "password", "xbox", "Xbox"}
	for _, env := range envs {
		for _, demo := range demos {
			for _, auth := range auths {
				name := fmt.Sprintf("env=%q/demo=%q/auth=%q", env, demo, auth)
				t.Run(name, func(t *testing.T) {
					cfg, err := chargerServeur(t, LaunchModeDefault, env, demo, auth, "")
					if err != nil {
						t.Fatalf("LoadServer : %v", err)
					}
					wantDemo := strings.ToLower(demo) == "true"
					wantAuth := normalizeMode(auth, defaultAuthMode)
					if got := profilAttendu(env, demo); cfg.Deployment != got {
						t.Errorf("profil = %q, attendu %q", cfg.Deployment, got)
					}
					if cfg.DemoMode != wantDemo || cfg.AuthMode != wantAuth || cfg.APIHost != "127.0.0.1" {
						t.Errorf("réglages bruts changés : demo=%v auth=%q hôte=%q", cfg.DemoMode, cfg.AuthMode, cfg.APIHost)
					}
					accounts := wantAuth == "password" || wantAuth == "xbox"
					c := cfg.Capabilities
					if c.UseAccounts != accounts || c.UseInvitations != accounts || c.SignInWithXbox != (wantAuth == "xbox") {
						t.Errorf("capacités de comptes = %+v pour l'authentification %q", c, wantAuth)
					}
					if c.EditSettings == wantDemo || c.UploadMedia == wantDemo || c.PoolTokens == wantDemo {
						t.Errorf("capacités figées en démo seulement : %+v pour demo=%v", c, wantDemo)
					}
					if c.CompactChrome {
						t.Errorf("hors desktop : chrome de bureau offert : %+v", c)
					}
					if c.UpdateApp || !c.OfferWindowsDownload {
						t.Errorf("hors desktop : mise à jour offerte ou téléchargement absent : %+v", c)
					}
					if got, want := authz.Enforced(cfg.DemoMode, cfg.AuthMode), !wantDemo && accounts; got != want {
						t.Errorf("authz.Enforced = %v, attendu %v", got, want)
					}
				})
			}
		}
	}
}

// TestDeployment_CapacitesParProfil : les capacités des quatre profils, ligne par ligne.
func TestDeployment_CapacitesParProfil(t *testing.T) {
	cas := []struct {
		profil Deployment
		env    string
		demo   string
		launch string
		want   DeploymentCapabilities
	}{
		{DeploymentHosted, "production", "", LaunchModeDefault, DeploymentCapabilities{
			UseGroups: true, PoolTokens: true, UseAdmin: true, UseResticBackup: true, UseDiscord: true,
			UseOutboundNetwork: true, OfferWindowsDownload: true, EditSettings: true, UploadMedia: true}},
		{DeploymentDev, "", "", LaunchModeDefault, DeploymentCapabilities{
			UseGroups: true, PoolTokens: true, UseAdmin: true, UseResticBackup: true, UseDiscord: true,
			UseOutboundNetwork: true, OfferWindowsDownload: true, EditSettings: true, UploadMedia: true}},
		{DeploymentDemo, "", "true", LaunchModeDefault, DeploymentCapabilities{
			UseAdmin: true, OfferWindowsDownload: true}},
		{DeploymentDesktop, "", "", LaunchModeDesktop, DeploymentCapabilities{
			UseOutboundNetwork: true, UpdateApp: true, EditSettings: true, UploadMedia: true,
			ShellEntryGate: true, PoolGrantedToLocalUser: true, WakeSequence: true, TokenVault: true,
			CompactChrome: true}},
	}
	for _, c := range cas {
		t.Run(string(c.profil), func(t *testing.T) {
			cfg, err := chargerServeur(t, c.launch, c.env, c.demo, "", "")
			if err != nil {
				t.Fatalf("LoadServer : %v", err)
			}
			if cfg.Deployment != c.profil {
				t.Fatalf("profil = %q, attendu %q", cfg.Deployment, c.profil)
			}
			if cfg.Capabilities != c.want {
				t.Errorf("capacités = %+v\nattendu    %+v", cfg.Capabilities, c.want)
			}
		})
	}
}

// TestDeployment_LeModeDesktopGagneSurLEnvironnement : l'ordre de résolution de l'ADR 0044 D-1 —
// -mode desktop passe avant LEVELUP_ENV et le reste.
func TestDeployment_LeModeDesktopGagneSurLEnvironnement(t *testing.T) {
	for _, env := range []string{"", "development", "production"} {
		cfg, err := chargerServeur(t, LaunchModeDesktop, env, "", "xbox", "0.0.0.0")
		if err != nil {
			t.Fatalf("env=%q : %v", env, err)
		}
		if cfg.Deployment != DeploymentDesktop {
			t.Errorf("env=%q : profil %q, attendu desktop", env, cfg.Deployment)
		}
	}
}

// TestDeployment_DesktopForceLaBoucleLocale : quelle que soit LEVELUP_API_HOST, l'hôte d'écoute
// du profil desktop est la boucle locale ; une valeur qui contredit est journalisée UNE fois.
func TestDeployment_DesktopForceLaBoucleLocale(t *testing.T) {
	for _, host := range []string{"", "0.0.0.0", "192.168.1.20", "::", "localhost", "::1", "127.0.0.1"} {
		t.Run(fmt.Sprintf("host=%q", host), func(t *testing.T) {
			var journal bytes.Buffer
			ancien := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&journal, nil)))
			defer slog.SetDefault(ancien)

			cfg, err := chargerServeur(t, LaunchModeDesktop, "", "", "", host)
			if err != nil {
				t.Fatalf("LoadServer : %v", err)
			}
			if cfg.APIHost != "127.0.0.1" || cfg.ServerAddr() != "127.0.0.1:8000" {
				t.Errorf("hôte = %q (%s), attendu 127.0.0.1", cfg.APIHost, cfg.ServerAddr())
			}
			conflit := host != "" && !isLoopbackHost(host)
			if n := strings.Count(journal.String(), "LEVELUP_API_HOST ignoré"); (n == 1) != conflit || n > 1 {
				t.Errorf("avertissements = %d pour l'hôte %q (conflit=%v)\n%s", n, host, conflit, journal.String())
			}
		})
	}
}

// TestDeployment_DesktopIgnoreLeModeDAuthentification : un LEVELUP_AUTH_MODE oublié dans
// l'environnement n'altère pas le modèle de sécurité d'une application locale ; la valeur
// dérivée est acceptée par Validate, y compris sous LEVELUP_ENV=production, et n'impose rien.
func TestDeployment_DesktopIgnoreLeModeDAuthentification(t *testing.T) {
	for _, auth := range []string{"", "none", "password", "xbox", "n-importe-quoi"} {
		t.Run(fmt.Sprintf("auth=%q", auth), func(t *testing.T) {
			var journal bytes.Buffer
			ancien := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&journal, nil)))
			defer slog.SetDefault(ancien)

			cfg, err := chargerServeur(t, LaunchModeDesktop, "production", "", auth, "")
			if err != nil {
				t.Fatalf("LoadServer : %v", err)
			}
			if cfg.AuthMode != authModeDesktop || cfg.PublishedAuthMode() != "none" {
				t.Errorf("AuthMode = %q (publié %q), attendu desktop (publié none)", cfg.AuthMode, cfg.PublishedAuthMode())
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate refuse le profil desktop sous production : %v", err)
			}
			if authz.Enforced(cfg.DemoMode, cfg.AuthMode) {
				t.Error("authz.Enforced vrai en desktop : la propriété des joueurs s'appliquerait à l'utilisateur seul")
			}
			if !authz.AuthDisabled(cfg.AuthMode) {
				t.Error("authz.AuthDisabled faux en desktop : les gardes de session s'appliqueraient")
			}
			if cfg.Capabilities.UseAccounts || cfg.Capabilities.SignInWithXbox {
				t.Errorf("comptes offerts en desktop : %+v", cfg.Capabilities)
			}
			if warned := strings.Contains(journal.String(), "LEVELUP_AUTH_MODE ignoré"); warned != (auth != "") {
				t.Errorf("avertissement = %v pour LEVELUP_AUTH_MODE=%q", warned, auth)
			}
		})
	}
}

// TestDeployment_DesktopEtDemoRefuses : -mode desktop avec la démo n'a pas de sens commun.
func TestDeployment_DesktopEtDemoRefuses(t *testing.T) {
	cfg, err := chargerServeur(t, LaunchModeDesktop, "", "true", "", "")
	if err == nil || cfg != nil {
		t.Fatalf("attendu un refus, obtenu cfg=%v err=%v", cfg, err)
	}
	if !strings.Contains(err.Error(), "LEVELUP_DEMO_MODE") {
		t.Errorf("l'erreur doit nommer LEVELUP_DEMO_MODE : %v", err)
	}
}

// TestDeployment_RefreshOnlyResoutLeProfilDesktop : le processus de rafraîchissement agit sur les
// données de l'utilisateur desktop : même profil, mêmes capacités (dont la séquence de réveil, que
// seule la capacité commande), avec la démo refusée comme pour desktop.
func TestDeployment_RefreshOnlyResoutLeProfilDesktop(t *testing.T) {
	desktop, err := chargerServeur(t, LaunchModeDesktop, "", "", "", "")
	if err != nil {
		t.Fatalf("LoadServer desktop : %v", err)
	}
	refresh, err := chargerServeur(t, LaunchModeRefreshOnly, "production", "", "xbox", "0.0.0.0")
	if err != nil {
		t.Fatalf("LoadServer refresh-only : %v", err)
	}
	if refresh.Deployment != DeploymentDesktop || refresh.Capabilities != desktop.Capabilities {
		t.Errorf("refresh-only = %q %+v, attendu le profil desktop %+v", refresh.Deployment, refresh.Capabilities, desktop.Capabilities)
	}
	if refresh.AuthMode != "desktop" || refresh.APIHost != "127.0.0.1" {
		t.Errorf("refresh-only : auth=%q hôte=%q, attendu desktop et la boucle locale", refresh.AuthMode, refresh.APIHost)
	}
	if _, err := chargerServeur(t, LaunchModeRefreshOnly, "", "true", "", ""); err == nil {
		t.Error("refresh-only avec la démo accepté")
	}
}

// TestDeployment_ModeDeLancementInconnuRefuse : seuls les modes livrés entrent dans le résolveur.
func TestDeployment_ModeDeLancementInconnuRefuse(t *testing.T) {
	for _, mode := range []string{"Desktop", "Refresh-Only", "autre"} {
		if _, err := chargerServeur(t, mode, "", "", "", ""); err == nil {
			t.Errorf("mode %q accepté", mode)
		}
	}
}

// TestDeployment_LeModeDerivePasUneValeurDeLEnvironnement : « desktop » n'est une valeur
// admise de LEVELUP_AUTH_MODE sur aucun profil — il n'existe que dérivé du profil desktop.
func TestDeployment_LeModeDerivePasUneValeurDeLEnvironnement(t *testing.T) {
	for _, env := range []string{"", "production"} {
		cfg, err := chargerServeur(t, LaunchModeDefault, env, "", "desktop", "")
		if err != nil {
			t.Fatalf("LoadServer : %v", err)
		}
		if err := cfg.Validate(); err == nil {
			t.Errorf("env=%q : LEVELUP_AUTH_MODE=desktop accepté hors du profil desktop", env)
		}
	}
}

// TestDeployment_ValidateEnProductionInchange : hors desktop, la validation de production reste
// celle d'avant (secret par défaut refusé en hosted, jamais en démo).
func TestDeployment_ValidateEnProductionInchange(t *testing.T) {
	hosted, err := chargerServeur(t, LaunchModeDefault, "production", "", "none", "")
	if err != nil {
		t.Fatalf("LoadServer : %v", err)
	}
	if err := hosted.Validate(); err == nil {
		t.Error("hosted en production avec le secret par défaut : Validate doit refuser")
	}
	demo, err := chargerServeur(t, LaunchModeDefault, "production", "true", "none", "")
	if err != nil {
		t.Fatalf("LoadServer : %v", err)
	}
	if err := demo.Validate(); err != nil {
		t.Errorf("la démo en production n'est jamais refusée : %v", err)
	}
}
