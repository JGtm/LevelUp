// Package verrous — UN VERROU PAR CLÉ, créé à la demande, pour la vie du processus.
//
// Le motif revient partout où un travail doit être exclusif PAR clé et non globalement : un
// chemin de base (bail d'écriture, `platform/dblease`), un titre (passe post-sync de la source
// des kills, rattrapage des zones nommées), un chemin de base cible (indexation des médias).
// Il est écrit ICI une fois ; le garde-rail `archlint/no_keyed_mutex_registry_test.go` interdit
// d'en réécrire une copie à la main (`map[string]*sync.Mutex`, `LoadOrStore(…, &sync.Mutex{})`).
//
// LE REGISTRE NE RÉTRÉCIT JAMAIS, et c'est le contrat : retirer le verrou d'une clé pendant
// qu'un appelant le tient en ferait naître un second pour la même clé, et l'exclusion tomberait.
// Il compte un verrou par clé ACTIVE du processus (quelques titres, quelques chemins de base).
//
// Le verrou rendu est un `*sync.Mutex` ordinaire : l'appelant choisit `Lock` (attendre son tour)
// ou `TryLock` (passer son tour quand un autre travaille déjà).
package verrous

import "sync"

// Registre associe un verrou à chaque clé. Sa valeur zéro est prête à l'emploi et il est sûr en
// concurrence.
type Registre struct {
	mu     sync.Mutex
	parCle map[string]*sync.Mutex
}

// De rend le verrou d'une clé, créé au premier appel ; deux appels pour la même clé rendent le
// MÊME verrou.
func (r *Registre) De(cle string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mu, ok := r.parCle[cle]; ok {
		return mu
	}
	if r.parCle == nil {
		r.parCle = map[string]*sync.Mutex{}
	}
	mu := &sync.Mutex{}
	r.parCle[cle] = mu
	return mu
}
