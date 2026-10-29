// Package filmobs tient les compteurs d'observation du décodeur de film.
//
// La brique ne dépend d'aucune bibliothèque de métriques du dépôt : elle compte ici, localement,
// et laisse l'hôte (le serveur, un binaire) brancher un puits qui republie chaque incrément là où
// il l'expose (`/debug/vars` côté LevelUp). Les NOMS de compteurs sont ceux que la brique a
// toujours portés ; la publication reste identique.
//
// # Contrat
//
//   - AddInt / IncCounter : ajoutent au compteur nommé (créé à la première utilisation) ; delta
//     peut être négatif. Si un puits est posé, il reçoit ensuite le même (nom, delta).
//   - Load : valeur courante du compteur local (0 si absent). Les tests de la brique lisent ceci.
//   - Snapshot : copie des compteurs locaux ; modifier la copie ne touche rien.
//   - SetSink : pose (ou retire, avec nil) le puits. À poser UNE FOIS au démarrage, avant les
//     décodages ; il est sûr à tout moment (pointeur atomique), mais un incrément fait avant la
//     pose n'est pas rejoué vers le puits.
//
// Tout est sûr en concurrence. Les compteurs locaux ne se remettent jamais à zéro : un test lit
// la valeur avant, agit, puis compare la différence.
package filmobs

import (
	"sync"
	"sync/atomic"
)

// compteurs : nom -> *atomic.Int64.
var compteurs sync.Map

// puits : le puits optionnel, nil tant que l'hôte n'en a pas posé.
var puits atomic.Pointer[func(name string, delta int64)]

// SetSink pose le puits qui reçoit chaque incrément en plus du compte local ; nil le retire.
func SetSink(f func(name string, delta int64)) {
	if f == nil {
		puits.Store(nil)
		return
	}
	puits.Store(&f)
}

// IncCounter incrémente de 1 le compteur nommé.
func IncCounter(name string) {
	AddInt(name, 1)
}

// AddInt augmente le compteur nommé de delta (peut être négatif).
func AddInt(name string, delta int64) {
	if v, ok := compteurs.Load(name); ok {
		v.(*atomic.Int64).Add(delta)
	} else {
		actuel, _ := compteurs.LoadOrStore(name, &atomic.Int64{})
		actuel.(*atomic.Int64).Add(delta)
	}
	if f := puits.Load(); f != nil {
		(*f)(name, delta)
	}
}

// Load rend la valeur courante du compteur nommé (0 si absent).
func Load(name string) int64 {
	if v, ok := compteurs.Load(name); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}

// Snapshot rend une copie des compteurs locaux.
func Snapshot() map[string]int64 {
	out := map[string]int64{}
	compteurs.Range(func(k, v any) bool {
		out[k.(string)] = v.(*atomic.Int64).Load()
		return true
	})
	return out
}
