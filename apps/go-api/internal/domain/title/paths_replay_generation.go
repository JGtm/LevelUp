package title

import "path/filepath"

// ReplayGenerationQueuePath retourne le fichier des demandes de génération de rejeu en vol (file persistée de
// `replayregen`, reprise à l'ouverture). Il ne porte aucun secret : titre, match, joueur, date.
// Un seul writer : le process serveur.
//
// Ex: data/global/admin_state/replay_generation_queue.json
func (p *PathResolver) ReplayGenerationQueuePath() string {
	return filepath.Join(p.AdminStateDir(), "replay_generation_queue.json")
}
