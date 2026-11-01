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
