package service

// catalogue_cache.go — LES CATALOGUES VERSIONNÉS DÉCODÉS UNE FOIS PAR PROCESSUS.
//
// Les fichiers de référence lus à la requête (catalogue de callouts, catalogue de bornes des
// cartes) sont versionnés : ils ne changent qu'au déploiement, donc au redémarrage du processus. Les
// décoder à chaque requête — chaque clic sur une zone de l'onglet Tactique, chaque ouverture d'un
// rejeu — payait ce décodage à chaque fois. Un échec n'est pas gardé : la lecture suivante réessaie.
// Le catalogue rendu est partagé entre les requêtes et se lit sans être modifié.

import "sync"

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
