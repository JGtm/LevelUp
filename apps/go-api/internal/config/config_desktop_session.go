// config_desktop_session.go — le secret de session du mode desktop (ADR 0045 D-7, A12).
//
// Le secret qui signe les cookies ne vient pas de l'environnement dans ce profil : il est généré au
// premier lancement, écrit dans <DataRoot>/auth/session_secret et relu ensuite, de sorte qu'un
// relancement garde ses cookies valides et que la valeur publique par défaut ne serve jamais.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/platform/atomicfile"
)

const (
	// sessionSecretBytes : taille du secret généré, avant l'encodage hexadécimal (256 bits).
	sessionSecretBytes = 32
	// sessionSecretFileMode : le secret n'est lisible que de son propriétaire. Les bits de mode ne
	// sont que consultatifs sous Windows, où le fichier tient dans le profil de l'utilisateur.
	sessionSecretFileMode = 0o600
)

// applyDesktopSessionSecret pose dans cfg le secret de session du fichier, qu'il crée au premier
// lancement. LEVELUP_SESSION_SECRET est ignoré avec un avertissement : une variable oubliée dans
// l'environnement de l'utilisateur ne doit pas remplacer le secret propre à ce poste.
func applyDesktopSessionSecret(cfg *AppConfig) error {
	if getEnvOrDefault("LEVELUP_SESSION_SECRET", "") != "" {
		slog.Warn("deployment: LEVELUP_SESSION_SECRET ignoré en mode desktop (secret de session gardé dans le fichier de données)")
	}
	secret, err := loadOrCreateSessionSecret(cfg.PathResolver().SessionSecretPath())
	if err != nil {
		return err
	}
	cfg.SessionSecret = secret
	return nil
}

// loadOrCreateSessionSecret rend le secret gardé dans path, qu'il crée (32 octets aléatoires en
// hexadécimal, mode 0600) quand le fichier est absent. Un fichier présent mais illisible est une
// erreur, jamais un secret régénéré en silence ; un fichier présent dont le contenu est trop court
// ou égal au défaut public est remplacé et l'avertissement le dit (les cookies déjà posés cessent
// d'être valides, la coquille repasse par /desktop/enter).
func loadOrCreateSessionSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		secret := strings.TrimSpace(string(raw))
		if len(secret) >= minSessionSecretLen && secret != DefaultSessionSecret {
			return secret, nil
		}
		slog.Warn("session_secret: fichier inutilisable (vide, trop court ou valeur publique), un secret neuf est généré",
			"path", path, "octets", len(secret))
	case errors.Is(err, os.ErrNotExist):
	default:
		return "", fmt.Errorf("lecture du secret de session %s : %w", path, err)
	}
	buf := make([]byte, sessionSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("tirage du secret de session : %w", err)
	}
	secret := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("dossier du secret de session %s : %w", path, err)
	}
	if err := atomicfile.WriteFileStrict(path, []byte(secret+"\n"), sessionSecretFileMode); err != nil {
		return "", fmt.Errorf("écriture du secret de session %s : %w", path, err)
	}
	return secret, nil
}
