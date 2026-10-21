package title

import "path/filepath"

// PassesGaugePath retourne le chemin du fichier de la jauge « passes par match » : le rapport
// lignes brutes / lignes gardées de chaque table du registre de compaction, par titre, écrit par
// le contrôle de santé des données (un seul writer : le process serveur).
//
// IL EST UN FICHIER ET PAS UNE ROUTE : son lecteur est le script hôte de nuit
// (scripts/compact-passes-nuit.sh), qui décide sur lui d'arrêter le serveur pour compacter. Le
// fichier se lit depuis l'hôte sans authentification ni port ouvert, et sans ouvrir la base.
// Format texte `clé=valeur` (une table par ligne `table=`), lisible par `sed` sans `jq`.
//
// Ex: data/global/admin_state/passes_par_match.txt
func (p *PathResolver) PassesGaugePath() string {
	return filepath.Join(p.AdminStateDir(), "passes_par_match.txt")
}
