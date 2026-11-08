package ops

import "sync"

// RootState résume ce que la dernière passe d'indexation a constaté du dossier
// des captures d'un joueur : de quoi afficher le bandeau des fichiers introuvables.
type RootState struct {
	// Unavailable : le dossier racine n'était pas accessible, la passe n'a rien
	// marqué introuvable.
	Unavailable bool
	// BulkPending : nombre de médias dont le marquage est suspendu par le garde-fou
	// de disparition massive, en attente de confirmation.
	BulkPending int
}

// rootStateKey identifie le dossier d'un joueur dans un titre.
type rootStateKey struct{ title, gamertag string }

// rootStates : mémoire du process, perdue au redémarrage. Ce n'est pas une donnée à
// conserver : la passe suivante, lancée au démarrage, la reconstitue.
var rootStates = struct {
	sync.Mutex
	state   map[rootStateKey]RootState
	confirm map[rootStateKey]bool
}{state: map[rootStateKey]RootState{}, confirm: map[rootStateKey]bool{}}

// RecordRootState enregistre l'état constaté par la dernière passe du joueur.
func RecordRootState(title, gamertag string, s RootState) {
	rootStates.Lock()
	defer rootStates.Unlock()
	rootStates.state[rootStateKey{title, gamertag}] = s
}

// GetRootState rend l'état enregistré du joueur ; zéro si aucune passe n'en a
// rendu compte.
func GetRootState(title, gamertag string) RootState {
	rootStates.Lock()
	defer rootStates.Unlock()
	return rootStates.state[rootStateKey{title, gamertag}]
}

// ConfirmBulkMissing note que l'utilisateur accepte la disparition massive du
// joueur : la prochaine passe qui marque la lira (ConsumeBulkConfirmation).
func ConfirmBulkMissing(title, gamertag string) {
	rootStates.Lock()
	defer rootStates.Unlock()
	rootStates.confirm[rootStateKey{title, gamertag}] = true
}

// ConsumeBulkConfirmation rend la confirmation en attente du joueur et la remet à
// zéro : une confirmation ne vaut que pour une passe.
func ConsumeBulkConfirmation(title, gamertag string) bool {
	rootStates.Lock()
	defer rootStates.Unlock()
	k := rootStateKey{title, gamertag}
	v := rootStates.confirm[k]
	delete(rootStates.confirm, k)
	return v
}
