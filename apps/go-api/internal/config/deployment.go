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
	// LaunchModeRefreshOnly : processus de rafraîchissement des jetons seul (tâche planifiée du
	// poste, ADR 0045 D-10). Il ne sert rien et n'ouvre aucune base, mais il agit sur les données
	// de l'utilisateur desktop : il résout le MÊME profil que [LaunchModeDesktop].
	LaunchModeRefreshOnly = "refresh-only"
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
	// ShellEntryGate : l'accès au serveur passe par l'entrée de la coquille (ADR 0045 D-3) : route
	// /desktop/enter, cookie lié au port, Host strict, CORS limité à l'origine du port lié. Hors de
	// cette capacité rien de cela n'est monté.
	ShellEntryGate bool
	// PoolGrantedToLocalUser : la mise en commun du jeton est accordée d'office à l'utilisateur local
	// (ADR 0045 D-8, exception à ADR 0038 D4) : le seul jeton du pool est le sien et ne quitte pas le
	// poste. Distincte de PoolTokens, qui dit que le choix est DEMANDÉ à l'utilisateur.
	PoolGrantedToLocalUser bool
	// WakeSequence : l'instance rattrape ce que l'arrêt de la machine lui a fait manquer — vérifie les
	// jetons et tire un cycle de sync en retard au lancement et à la sortie de veille (ADR 0045 D-10).
	// Faux sur un serveur hébergé : il redémarre à chaque déploiement et n'est jamais en veille.
	WakeSequence bool
	// TokenVault : les fichiers de jetons sont scellés par le coffre du système (DPAPI, compte Windows
	// courant) au lieu d'être écrits en clair (ADR 0045 D-11, amende ADR 0023). Le chargement de la
	// configuration en dérive le chiffreur du processus (tokencipher.InstallForProfile) : c'est le profil
	// qui choisit, jamais GOOS seul. Le profil desktop est Windows seul ; le serveur refuse de démarrer
	// sous cette capacité si le coffre est indisponible.
	TokenVault bool
	// CompactChrome : l'interface adopte le chrome d'une application de bureau — barre latérale à la place de
	// la barre haute, fenêtre « À propos » à la place du pied de page, accueil sans image de couverture,
	// sans les éléments qui n'ont de sens que sur un site (déconnexion, téléchargement, liens du développeur).
	// Le client branche dessus, jamais sur le profil (D-4).
	CompactChrome bool
	// AdoptSystemLocale : le fuseau horaire et la langue d'interface du système servent de défaut aux
	// réglages `user_timezone` et `lang` absents d'app_settings.json (premier lancement d'une
	// application locale hors de France). Faux sur un serveur : le fuseau et la langue y sont ceux de
	// l'instance, pas ceux de la machine qui l'héberge.
	AdoptSystemLocale bool
	// FrequentCheckpoint : le serveur CHECKPOINT toutes les quelques minutes les bases qu'il tient en
	// lecture-écriture (`desktop_checkpoint_interval`), pour qu'une coupure brutale du poste ne laisse
	// qu'un court WAL. Faux sur un serveur hébergé, que le déploiement et l'arrêt propre vident.
	FrequentCheckpoint bool
	// LoginViaBrowser : la connexion Microsoft se fait dans le navigateur par défaut, sans code à
	// recopier (code d'autorisation + PKCE, adresse de retour loopback servie par ce serveur, ADR 0045
	// D-22). Ouvre GET /auth/xbox/callback sans cookie de la coquille, sous le garde du `state`.
	// Hors desktop, le code d'appareil ou le SSO du site restent les seuls chemins.
	LoginViaBrowser bool
	// CompareAccounts : l'instance suit plusieurs comptes, donc les pages qui les confrontent entre eux
	// (classements de la communauté) ont de quoi s'afficher. Faux sur une application de bureau, qui
	// n'en suit qu'un (ADR 0045 D-7).
	CompareAccounts bool
	// GameProcessWatch : le serveur sonde la liste des processus du poste pour savoir si le jeu tourne
	// (noms d'exécutables `process_names` des titres, énumération seule, jamais de lecture de mémoire) :
	// le « mode jeu » suspend la cuisson des rejeux et les tâches lourdes pendant la partie, et la
	// fermeture du jeu déclenche une synchronisation. Réglage `game_process_watch`. Faux sur un serveur,
	// qui n'héberge pas le jeu de l'utilisateur.
	GameProcessWatch bool
	// ShellIntegration : le serveur pilote l'intégration de la coquille avec le poste, par les réglages
	// `native_notifications`, `close_to_tray` et `start_with_windows` (événement stdout `shell_settings`,
	// ADR 0045) : notifications Windows, icône de la zone de notification, démarrage avec la session.
	// Faux hors de la coquille : un navigateur n'a ni zone de notification ni entrée de démarrage.
	ShellIntegration bool
	// ScheduledRefresh : le serveur peut poser une tâche planifiée Windows qui lance `-mode refresh-only`
	// pour garder les jetons vivants quand l'application est fermée (réglage `scheduled_refresh`).
	ScheduledRefresh bool
	// ParquetBackup : le serveur sauvegarde périodiquement les bases en Parquet vers un dossier choisi par
	// l'utilisateur (réglages `backup_dir`, `backup_interval`, `backup_keep`) et sait les restaurer. Faux sur un
	// serveur hébergé, qui a sa propre sauvegarde (restic) ; la démo n'a rien à protéger.
	ParquetBackup bool
	// ImportFromSite : le serveur peut importer les données d'un joueur depuis une instance hébergée
	// (jeton d'export à usage unique, archive Parquet vérifiée, écriture par les chemins d'écriture existants).
	ImportFromSite bool
	// ExportForDesktop : l'instance livre au joueur CONNECTÉ ses données pour une application de bureau
	// (GET /export/desktop). Vraie sur un serveur hébergé ; l'application de bureau, qui n'a qu'un compte, et la
	// démo, qui n'en a aucun de réel, ne l'offrent pas.
	ExportForDesktop bool
	// DiagnosticReport : le serveur produit un rapport de diagnostic (archive de journaux, versions et état,
	// sans secret) que l'utilisateur joint à un signalement.
	DiagnosticReport bool
	// MemoryLimit : le plafond mémoire des bases DuckDB se règle (`memory_limit_mb`) et la mémoire du
	// processus se mesure.
	MemoryLimit bool
	// ReplayRegen : le serveur compte les rejeux rangés en retard sur la version de l'application et les
	// régénère, un film à la fois, sur demande de l'utilisateur (file de fond, mode jeu respecté ; ADR 0045).
	// Faux sur un serveur hébergé, dont les rejeux se cuisent au fil de la synchronisation ou par un ouvrier.
	ReplayRegen bool
	// ResourcePackFatal : le paquet de ressources installé DOIT être celui du moteur : l'empreinte du manifeste est
	// comparée à celle que le binaire embarque au démarrage, et une différence refuse le démarrage (même discipline
	// qu'une migration fatale). Vraie sur un serveur hébergé et sur la démo, dont l'image embarque moteur et paquet
	// ensemble ; fausse sur desktop, qui démarre sur le paquet présent et télécharge le bon (capacité UpdateApp),
	// et en développement, qui n'embarque aucune empreinte.
	ResourcePackFatal bool
	// AutoSyncByDefault : la synchronisation automatique (`spnkr_auto_sync_enabled`) est ACTIVE tant que
	// l'utilisateur n'a rien écrit : l'application de bureau suit un compte et doit se tenir à jour sans
	// réglage préalable. Défaut des clés absentes d'app_settings.json (settings.Store.WithAutoSyncDefault) ;
	// une valeur écrite gagne toujours. Faux sur un serveur, où l'administrateur l'active.
	AutoSyncByDefault bool
	// KeepMediaSources : l'original d'une vidéo convertie n'est jamais supprimé par l'application. Sur un
	// poste, le dossier des captures appartient à l'utilisateur et l'explorateur y fait foi : la
	// suppression de l'original après conversion est interdite, le réglage correspondant est masqué et
	// LEVELUP_MEDIA_DELETE_SOURCE est ignorée ([AppConfig.MediaDeleteSource]).
	KeepMediaSources bool
	// RevealFilesInExplorer : l'application peut ouvrir l'explorateur de fichiers du système sur un
	// média (elle tourne sur le poste de l'utilisateur, pas sur un serveur distant).
	RevealFilesInExplorer bool
	// TrashMediaSources : supprimer un média depuis l'application envoie son original à la Corbeille du
	// système quand le support en a une, au lieu de l'effacer définitivement. Faux sur un serveur : la
	// suppression y est définitive.
	TrashMediaSources bool
	// PickFolder : la page peut demander à la coquille d'ouvrir la boîte « Parcourir » du système pour
	// choisir un dossier (une seule commande IPC, limitée à la boucle locale, ADR 0045). Vraie sur le
	// profil desktop seulement : un navigateur n'a pas ce pont, le champ reste alors un texte à saisir.
	PickFolder bool
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
// du serveur ([LaunchModeDefault], [LaunchModeDesktop] ou [LaunchModeRefreshOnly]).
//
// Ordre, premier qui correspond : les modes desktop et refresh-only donnent desktop ;
// LEVELUP_DEMO_MODE=true donne demo ;
// un LEVELUP_ENV déclaré donne hosted ; sinon dev. Un mode desktop ou refresh-only avec la démo est
// REFUSÉ (les deux n'ont aucun sens commun). Sous desktop, LEVELUP_AUTH_MODE et LEVELUP_API_HOST
// sont ignorés avec un avertissement : une variable oubliée dans l'environnement d'un
// utilisateur ne doit pas changer le modèle de sécurité d'une application locale.
func resolveDeployment(cfg *AppConfig, launchMode string) (*AppConfig, error) {
	switch launchMode {
	case LaunchModeDefault, LaunchModeDesktop, LaunchModeRefreshOnly:
	default:
		return nil, fmt.Errorf("mode de lancement %q inconnu pour le profil de déploiement", launchMode)
	}
	demo := cfg.DemoMode // mode déclaré (LEVELUP_DEMO_MODE), pas un choix de source de données
	desktopRun := launchMode == LaunchModeDesktop || launchMode == LaunchModeRefreshOnly
	switch {
	case desktopRun && demo:
		return nil, fmt.Errorf("-mode %s est incompatible avec LEVELUP_DEMO_MODE=true : "+
			"une application locale à un seul utilisateur n'a rien de commun avec la démo publique", launchMode)
	case desktopRun:
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
		UseAccounts:            accounts,
		SignInWithXbox:         !desktop && cfg.AuthMode == authModeXbox,
		UseInvitations:         accounts,
		UseGroups:              !desktop && !demo,
		PoolTokens:             !desktop && !demo,
		UseAdmin:               !desktop,
		UseResticBackup:        !desktop && !demo,
		UseDiscord:             !desktop && !demo,
		UseOutboundNetwork:     !demo,
		UpdateApp:              desktop,
		OfferWindowsDownload:   !desktop,
		EditSettings:           !demo,
		UploadMedia:            !demo,
		ShellEntryGate:         desktop,
		PoolGrantedToLocalUser: desktop,
		WakeSequence:           desktop,
		TokenVault:             desktop,
		CompactChrome:          desktop,
		AdoptSystemLocale:      desktop,
		FrequentCheckpoint:     desktop,
		LoginViaBrowser:        desktop,
		CompareAccounts:        !desktop,
		GameProcessWatch:       desktop,
		ShellIntegration:       desktop,
		ScheduledRefresh:       desktop,
		AutoSyncByDefault:      desktop,
		ResourcePackFatal:      !desktop && cfg.Deployment != DeploymentDev,
		ParquetBackup:          desktop,
		ImportFromSite:         desktop,
		ExportForDesktop:       !desktop && !demo,
		DiagnosticReport:       desktop,
		MemoryLimit:            desktop,
		ReplayRegen:            desktop,
		KeepMediaSources:       desktop,
		RevealFilesInExplorer:  desktop,
		TrashMediaSources:      desktop,
		PickFolder:             desktop,
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
