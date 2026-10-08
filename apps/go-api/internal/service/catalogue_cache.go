package service

// catalogue_cache.go — LES CATALOGUES VERSIONNÉS DÉCODÉS UNE FOIS PAR PROCESSUS.
//
// Les fichiers de référence VERSIONNÉS lus à la requête (catalogue de callouts, catalogue de bornes
// des cartes) ne changent qu'au déploiement, donc au redémarrage du processus. Les
// décoder à chaque requête — chaque clic sur une zone de l'onglet Tactique, chaque ouverture d'un
// rejeu — payait ce décodage à chaque fois. Un échec n'est pas gardé : la lecture suivante réessaie.
// Le catalogue rendu est partagé entre les requêtes et se lit sans être modifié. Un fichier écrit
// par le runtime (catalogue généré des zones) a son propre cache, à empreinte (`cacheAEmpreinte`).

import (
	"os"
	"sync"
	"time"
)

// cacheParChemin garde le catalogue décodé de chaque chemin. Sûr en concurrence : le premier lecteur
// d'un chemin le décode, les suivants attendent puis le lisent.
type cacheParChemin[T any] struct {
	mu        sync.Mutex
	parChemin map[string]*T
	charger   func(chemin string) (*T, error)
}

// nouveauCacheParChemin construit le cache d'un catalogue et de sa fonction de lecture.
func nouveauCacheParChemin[T any](charger func(chemin string) (*T, error)) *cacheParChemin[T] {
	return &cacheParChemin[T]{parChemin: map[string]*T{}, charger: charger}
}

// lire rend le catalogue décodé d'un chemin, décodé à la première lecture seulement.
func (c *cacheParChemin[T]) lire(chemin string) (*T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cat, ok := c.parChemin[chemin]; ok {
		return cat, nil
	}
	cat, err := c.charger(chemin)
	if err != nil {
		return nil, err
	}
	c.parChemin[chemin] = cat
	return cat, nil
}

// cacheAEmpreinte garde le catalogue décodé d'un fichier QUI CHANGE à l'exécution (le catalogue
// généré des zones, que le rattrapage enrichit au fil de l'eau) : il est relu dès que sa date de
// modification ou sa taille bouge, et seulement alors. Un fichier absent rend l'erreur de `os.Stat`
// (`fs.ErrNotExist`) ; un échec de lecture n'est pas gardé.
type cacheAEmpreinte[T any] struct {
	mu        sync.Mutex
	parChemin map[string]lectureAEmpreinte[T]
	charger   func(chemin string) (*T, error)
}

// lectureAEmpreinte : un catalogue décodé et l'empreinte du fichier qui l'a donné.
type lectureAEmpreinte[T any] struct {
	modifie time.Time
	taille  int64
	cat     *T
}

// nouveauCacheAEmpreinte construit le cache d'un catalogue changeant et de sa fonction de lecture.
func nouveauCacheAEmpreinte[T any](charger func(chemin string) (*T, error)) *cacheAEmpreinte[T] {
	return &cacheAEmpreinte[T]{parChemin: map[string]lectureAEmpreinte[T]{}, charger: charger}
}

// lire rend le catalogue décodé d'un chemin, relu si le fichier a changé depuis la lecture gardée.
func (c *cacheAEmpreinte[T]) lire(chemin string) (*T, error) {
	st, err := os.Stat(chemin)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if l, ok := c.parChemin[chemin]; ok && l.modifie.Equal(st.ModTime()) && l.taille == st.Size() {
		return l.cat, nil
	}
	cat, err := c.charger(chemin)
	if err != nil {
		return nil, err
	}
	c.parChemin[chemin] = lectureAEmpreinte[T]{modifie: st.ModTime(), taille: st.Size(), cat: cat}
	return cat, nil
}
