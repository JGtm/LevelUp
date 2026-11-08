// config_desktop.go — réglages du serveur lancé par un processus parent (coquille desktop,
// pré-commande d'air en développement). Dans un fichier dédié : config.go est gelé en taille
// (config_size_ratchet_test.go).
package config

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DesktopWebDistDirName : sous-dossier de la racine de ressources qui porte le front compilé de
// l'application installée (le bundle NSIS le dépose là). LEVELUP_WEB_DIST l'emporte quand il est posé.
const DesktopWebDistDirName = "web"

// DesktopConfig est embarquée dans AppConfig : ses champs s'y lisent directement.
type DesktopConfig struct {
	// DesktopSecret : secret partagé avec le processus qui lance le serveur. Lit
	// LEVELUP_DESKTOP_SECRET. VIDE PAR DÉFAUT : la route POST /api/v1/internal/shutdown n'est
	// alors pas montée (un serveur hébergé n'a pas cette route). Jamais journalisé ni exposé
	// (ni /bootstrap, ni les journaux de configuration).
	DesktopSecret string
	// ShutdownRequester demande l'arrêt propre du serveur (reason : cause, journalisée).
	// Posé par main (bootServer.requestShutdown) après la construction de la config, comme
	// TitleReadyNotifier, avant celle du routeur ; nil en CLI/tests, où la route d'arrêt
	// répond 404.
	ShutdownRequester func(reason string)
	// UpdatePublicKey : la clé publique de mise à jour (le champ `pubkey` de tauri.conf.json), posée par la
	// coquille dans LEVELUP_UPDATE_PUBKEY : elle vérifie la signature du paquet de ressources, la même que celle de
	// l'installeur. Vide : aucun paquet ne peut être téléchargé.
	UpdatePublicKey string
	// SystemLocale : fuseau et langue du système, lus sous la capacité AdoptSystemLocale
	// (valeur zéro sinon). Défaut des clés absentes d'app_settings.json (settings.Store.WithSystemDefaults).
	SystemLocale SystemLocale
	// CheckpointInterval : intervalle du CHECKPOINT périodique des bases tenues en RW, lu sous la
	// capacité FrequentCheckpoint (zéro sinon : la tâche n'est pas lancée).
	CheckpointInterval time.Duration
}

// withDesktopConfig lit les réglages desktop de l'environnement dans cfg et le rend.
func withDesktopConfig(cfg *AppConfig) *AppConfig {
	cfg.DesktopSecret = strings.TrimSpace(getEnvOrDefault("LEVELUP_DESKTOP_SECRET", ""))
	cfg.UpdatePublicKey = strings.TrimSpace(getEnvOrDefault("LEVELUP_UPDATE_PUBKEY", ""))
	return cfg
}

// resolveDesktop est la fin du chargement : lit les réglages desktop, résout le profil de
// déploiement ([resolveDeployment]) et, sous le profil desktop, pose le secret de session du
// fichier (ADR 0045 D-7) et l'origine web du port courant.
func resolveDesktop(cfg *AppConfig, launchMode string) (*AppConfig, error) {
	cfg, err := resolveDeployment(withDesktopConfig(cfg), launchMode)
	if err != nil {
		return nil, err
	}
	installTokenCipher(cfg)
	adoptSystemLocale(cfg)
	adoptCheckpointInterval(cfg)
	if !cfg.Capabilities.ShellEntryGate {
		return cfg, nil
	}
	if cfg.WebDistDir == "" {
		cfg.WebDistDir = filepath.Join(cfg.Roots().Resources, DesktopWebDistDirName)
	}
	if err := applyDesktopSessionSecret(cfg); err != nil {
		return nil, err
	}
	cfg.SetBoundPort(cfg.APIPort)
	return cfg, nil
}

// SetBoundPort enregistre le port réellement lié par le serveur (le mode desktop reprend un port
// stable ou en choisit un libre : le port configuré n'est qu'une valeur de départ) et, sous
// l'entrée de la coquille, en dérive les origines autorisées : l'origine exacte de ce port, jamais
// LEVELUP_CORS_ORIGINS (ADR 0045 D-3). À appeler avant la construction du routeur.
func (c *AppConfig) SetBoundPort(port int) {
	c.APIPort = port
	if c.Capabilities.ShellEntryGate {
		c.CORSOrigins = []string{c.DesktopOrigin()}
	}
}

// DesktopAuthority rend l'autorité exacte que porte l'en-tête Host d'une requête légitime de la
// coquille : l'hôte de boucle locale et le port lié.
func (c *AppConfig) DesktopAuthority() string {
	return net.JoinHostPort(c.APIHost, strconv.Itoa(c.APIPort))
}

// DesktopOrigin rend l'origine web de la coquille (schéma, hôte et port liés), la seule que CORS
// et le contrôle d'origine acceptent sous l'entrée de la coquille.
func (c *AppConfig) DesktopOrigin() string {
	return "http://" + c.DesktopAuthority()
}

// validateStartup est le contrôle statique de Validate : les modes d'authentification et
// d'inscription, puis, sous l'entrée de la coquille, un secret de session qui n'est ni le défaut
// public ni trop court (le profil desktop échappe au refus de production, il ne doit pas
// échapper à celui-là).
func (c *AppConfig) validateStartup() error {
	if err := c.validateModes(); err != nil {
		return err
	}
	if !c.Capabilities.ShellEntryGate {
		return nil
	}
	switch {
	case c.SessionSecret == DefaultSessionSecret:
		return fmt.Errorf("mode desktop : le secret de session est la valeur publique par défaut")
	case len(c.SessionSecret) < minSessionSecretLen:
		return fmt.Errorf("mode desktop : secret de session trop court (%d octets, minimum %d)",
			len(c.SessionSecret), minSessionSecretLen)
	}
	return nil
}
