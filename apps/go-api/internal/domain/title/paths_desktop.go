package title

import "path/filepath"

// DesktopPortPath retourne le fichier qui garde le port du serveur lancé par la coquille desktop :
// une seule ligne, le port en décimal. Le navigateur de la coquille indexe son stockage local par
// origine, port compris ; un port stable d'un lancement à l'autre conserve ce stockage (ADR 0045
// D-9). Absent ou occupé, le serveur choisit un port libre et l'écrit ici.
//
// Ex: data/desktop/port
func (p *PathResolver) DesktopPortPath() string {
	return filepath.Join(p.roots.Data, "desktop", "port")
}

// InstanceLockPath retourne le fichier du verrou d'instance (internal/platform/instancelock) :
// une seule instance du serveur par racine de données en mode desktop et refresh-only (ADR 0045
// D-10). Le fichier reste en place, vide, entre deux lancements ; seul le verrou noyau compte.
//
// Ex: data/levelup.lock
func (p *PathResolver) InstanceLockPath() string {
	return filepath.Join(p.roots.Data, "levelup.lock")
}

// SessionSecretPath retourne le fichier du secret de signature des cookies du mode desktop : généré
// au premier lancement (32 octets aléatoires, en hexadécimal), lu aux suivants, jamais la valeur
// publique par défaut (ADR 0045 D-7). Il vit sous la racine de données, dans le dossier d'authentification.
//
// Ex: data/auth/session_secret
func (p *PathResolver) SessionSecretPath() string {
	return filepath.Join(p.roots.Data, "auth", "session_secret")
}
