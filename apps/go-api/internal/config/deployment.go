// deployment.go — le profil de déploiement (ADR 0044) : UN concept, résolu UNE fois.
//
// Quatre manières de lancer le serveur : hosted (instance publique ou privée derrière un
// reverse proxy), desktop (serveur lancé par la coquille Windows, utilisateur unique), demo
// (démo publique en lecture seule), dev (poste de développement). Les trois réglages qui
// décrivaient jusqu'ici « quelle instance tourne » (LEVELUP_ENV, LEVELUP_DEMO_MODE,
// LEVELUP_AUTH_MODE) et le mode de lancement (drapeau -mode, passé en paramètre : ce paquet ne
// lit jamais os.Args) entrent dans [resolveDeployment], qui rend le profil et tout ce qui en
// découle : mode d'authentification, hôte d'écoute, capacités fines.
//
// Règle de lecture (D-4) : un consommateur branche sur une CAPACITÉ ([DeploymentCapabilities]),
// jamais sur la valeur du profil, dont le nom ne sert qu'à l'affichage et au diagnostic. Aucun
// code hors de ce fichier ne compare une valeur [Deployment] ; le garde-rail
// archlint/deployment_single_resolver_test.go l'impose.
package config

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

// Deployment : la manière dont ce processus serveur est lancé.
type Deployment string

// Les quatre profils.
const (
	DeploymentHosted  Deployment = "hosted"
	DeploymentDesktop Deployment = "desktop"
	DeploymentDemo    Deployment = "demo"
	DeploymentDev     Deployment = "dev"
)

// Deployments rend les quatre profils, dans un ordre stable (tables, tests).
func Deployments() []Deployment {
	return []Deployment{DeploymentHosted, DeploymentDev, DeploymentDemo, DeploymentDesktop}
}

// Modes de lancement acceptés par [LoadServer] (valeurs du drapeau -mode de cmd/server).
const (
	// LaunchModeDefault : serveur ordinaire, le profil se déduit de l'environnement.
	LaunchModeDefault = ""
	// LaunchModeDesktop : serveur lancé par la coquille desktop.
	LaunchModeDesktop = "desktop"
)

const (
	// authModeDesktop : mode d'authentification DÉRIVÉ du profil desktop. Jamais une valeur
	// admise de LEVELUP_AUTH_MODE (cf. authModeAccepted).
	authModeDesktop = "desktop"
	// desktopListenHost : l'hôte d'écoute du profil desktop, quoi que dise LEVELUP_API_HOST.
	desktopListenHost = "127.0.0.1"
	// environmentDevelopment : valeur de LEVELUP_ENV qui ne déclare PAS d'environnement.
	environmentDevelopment = "development"
)

// DeploymentCapabilities : les faits fins sur lesquels le reste du code branche, décidés ici
// et nulle part ailleurs. Une capacité vraie veut dire « cette instance l'offre », pas « le
// visiteur y a droit » (l'autorisation d'une requête reste à authz et aux middlewares).
type DeploymentCapabilities struct {
	// UseAccounts : les comptes et les inscriptions existent (mode d'authentification
	// password ou xbox).
	UseAccounts bool
	// SignInWithXbox : la connexion se fait par le SSO Xbox (mode xbox).
	SignInWithXbox bool
	// UseInvitations : des codes d'invitation relient les comptes et les groupes.
	UseInvitations bool
	// UseGroups : les groupes (accès mutuel entre comptes, ADR 0029) sont offerts.
	UseGroups bool
	// PoolTokens : la mise en commun des jetons (ADR 0038) est proposée à l'utilisateur.
	PoolTokens bool
	// UseAdmin : l'espace d'administration de l'instance est offert.
	UseAdmin bool
	// UseResticBackup : la sauvegarde restic des bases est offerte.
	UseResticBackup bool
	// UseDiscord : le relais d'alertes Discord est offert.
	UseDiscord bool
	// UseOutboundNetwork : le serveur peut sortir vers les services tiers (API Halo, Microsoft).
	UseOutboundNetwork bool
	// UpdateApp : l'application se met à jour elle-même (mise à jour signée de la coquille).
	UpdateApp bool
	// OfferWindowsDownload : l'instance propose le téléchargement de l'application Windows.
	OfferWindowsDownload bool
	// EditSettings : les réglages de l'instance sont modifiables (figés en démo).
	EditSettings bool
	// UploadMedia : le dépôt de captures est offert (refusé en démo).
	UploadMedia bool
}

// DeploymentProfile est embarqué dans AppConfig : le profil résolu et ses capacités.
type DeploymentProfile struct {
	// Deployment : le profil, fixé par [resolveDeployment]. Pour l'affichage et le diagnostic
	// seulement (D-4).
	Deployment Deployment
	// Capabilities : les faits fins du profil, fixés avec lui.
	Capabilities DeploymentCapabilities
}

// environmentDeclared : LEVELUP_ENV déclare-t-il un environnement (autre que vide ou
// « development ») ? Le critère que partagent le profil hosted et IsExposedDeployment.
func environmentDeclared(env string) bool {
	env = strings.TrimSpace(env)
	return env != "" && !strings.EqualFold(env, environmentDevelopment)
}

// resolveDeployment résout le profil de cfg (lu depuis l'environnement par load) et en dérive
// le mode d'authentification, l'hôte d'écoute et les capacités. launchMode : drapeau -mode
// du serveur ([LaunchModeDefault] ou [LaunchModeDesktop]).
//
// Ordre, premier qui correspond : le mode desktop donne desktop ; LEVELUP_DEMO_MODE=true donne
// demo ; un LEVELUP_ENV déclaré donne hosted ; sinon dev. Le mode desktop avec la démo est
// REFUSÉ (les deux n'ont aucun sens commun). Sous desktop, LEVELUP_AUTH_MODE et LEVELUP_API_HOST
// sont ignorés avec un avertissement : une variable oubliée dans l'environnement d'un
// utilisateur ne doit pas changer le modèle de sécurité d'une application locale.
func resolveDeployment(cfg *AppConfig, launchMode string) (*AppConfig, error) {
	switch launchMode {
	case LaunchModeDefault, LaunchModeDesktop:
	default:
		return nil, fmt.Errorf("mode de lancement %q inconnu pour le profil de déploiement", launchMode)
	}
	demo := cfg.DemoMode // mode déclaré (LEVELUP_DEMO_MODE), pas un choix de source de données
	switch {
	case launchMode == LaunchModeDesktop && demo:
		return nil, fmt.Errorf("-mode desktop est incompatible avec LEVELUP_DEMO_MODE=true : " +
			"une application locale à un seul utilisateur n'a rien de commun avec la démo publique")
	case launchMode == LaunchModeDesktop:
		cfg.Deployment = DeploymentDesktop
		forceDesktopAuthAndHost(cfg)
	case demo:
		cfg.Deployment = DeploymentDemo
	case environmentDeclared(cfg.Environment):
		cfg.Deployment = DeploymentHosted
	default:
		cfg.Deployment = DeploymentDev
	}
	cfg.Capabilities = capabilitiesOf(cfg)
	return cfg, nil
}

// forceDesktopAuthAndHost pose les valeurs dérivées du profil desktop : l'authentification
// « desktop » (ni compte ni propriété des joueurs, comme none) et la boucle locale. Une valeur
// d'environnement qui les contredit est ignorée et journalisée.
func forceDesktopAuthAndHost(cfg *AppConfig) {
	if raw := getEnvOrDefault("LEVELUP_AUTH_MODE", ""); raw != "" {
		slog.Warn("deployment: LEVELUP_AUTH_MODE ignoré en mode desktop (authentification dérivée du profil)",
			"valeur", raw)
	}
	cfg.AuthMode = authModeDesktop
	if raw := getEnvOrDefault("LEVELUP_API_HOST", ""); raw != "" && !isLoopbackHost(raw) {
		slog.Warn("deployment: LEVELUP_API_HOST ignoré en mode desktop (boucle locale imposée)",
			"valeur", raw, "hote", desktopListenHost)
	}
	cfg.APIHost = desktopListenHost
}

// capabilitiesOf dérive les capacités du profil de cfg. Hors desktop, les capacités de
// comptes suivent le mode d'authentification déjà lu, comme les lecteurs d'avant.
func capabilitiesOf(cfg *AppConfig) DeploymentCapabilities {
	desktop := cfg.Deployment == DeploymentDesktop
	demo := cfg.Deployment == DeploymentDemo
	accounts := !desktop && (cfg.AuthMode == authModePassword || cfg.AuthMode == authModeXbox)
	return DeploymentCapabilities{
		UseAccounts:          accounts,
		SignInWithXbox:       !desktop && cfg.AuthMode == authModeXbox,
		UseInvitations:       accounts,
		UseGroups:            !desktop && !demo,
		PoolTokens:           !desktop && !demo,
		UseAdmin:             !desktop,
		UseResticBackup:      !desktop && !demo,
		UseDiscord:           !desktop && !demo,
		UseOutboundNetwork:   !demo,
		UpdateApp:            desktop,
		OfferWindowsDownload: !desktop,
		EditSettings:         !demo,
		UploadMedia:          !demo,
	}
}

// authModeAccepted : le mode d'authentification de cfg est-il une valeur admise ? Les valeurs
// de LEVELUP_AUTH_MODE, plus « desktop » quand le profil le dérive (et seulement alors).
func (c *AppConfig) authModeAccepted() bool {
	mode := normalizeMode(c.AuthMode, defaultAuthMode)
	return slices.Contains(knownAuthModes, mode) || (mode == authModeDesktop && c.Deployment == DeploymentDesktop)
}

// refusesUnsafeConfig : Validate refuse-t-il ici une configuration non sûre ? Seulement le
// profil hosted en production : la démo n'a rien à protéger, et desktop est sûr par
// construction (boucle locale, secret par lancement, ADR 0045).
func (c *AppConfig) refusesUnsafeConfig() bool {
	return c.IsProduction() && !c.DemoMode && c.Deployment != DeploymentDesktop
}

// PublishedAuthMode : le mode d'authentification tel que /bootstrap le publie. Le contrat
// d'API reste none, password ou xbox : le mode dérivé « desktop » (aucune authentification)
// se publie « none ».
func (c *AppConfig) PublishedAuthMode() string {
	if c.AuthMode == authModeDesktop {
		return defaultAuthMode
	}
	return c.AuthMode
}
