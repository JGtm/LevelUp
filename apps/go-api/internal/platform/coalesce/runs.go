// Package coalesce — recalculs coûteux sérialisés et absorbés par clé.
package coalesce

import (
	"context"
	"sync"
)

// Runs sérialise et absorbe les recalculs demandés pour une même clé
// (une base joueur). Une exclusion de match déclenche un recalcul de tout l'historique du
// joueur : des demandes répétées ne doivent ni s'empiler ni tourner en parallèle.
//
// Contrat de Do : à son retour, un recalcul DÉMARRÉ APRÈS le dépôt de la demande s'est
// terminé. Le drapeau d'exclusion étant persisté avant l'appel, ce recalcul le voit.
//   - aucun recalcul en cours pour la clé : fn tourne dans l'appelant ;
//   - un recalcul est en cours : la demande est notée (dirty) et attend ; à la fin du
//     recalcul en cours, UN seul recalcul de rattrapage sert toutes les demandes notées.
//
// Les demandeurs absorbés reçoivent l'erreur du dernier recalcul. Une clé dont le dernier
// recalcul a échoué est retenue (FailedLast) jusqu'au prochain succès : l'appelant sait
// alors qu'une demande identique à l'état déjà persisté doit tout de même recalculer.
// Sa valeur zéro est prête.
type Runs struct {
	mu     sync.Mutex
	runs   map[string]*coalescedRun
	failed map[string]struct{}
}

// coalescedRun est l'état d'une clé dont un recalcul est en cours.
type coalescedRun struct {
	dirty bool          // une demande est arrivée pendant le recalcul courant
	done  chan struct{} // fermé quand le dernier recalcul est terminé
	err   error         // erreur du dernier recalcul, lisible après done
}

// Do exécute fn pour key en respectant le contrat ci-dessus. ctx borne l'attente d'un
// demandeur absorbé ; il n'interrompt pas le recalcul en cours.
func (c *Runs) Do(ctx context.Context, key string, fn func(context.Context) error) error {
	c.mu.Lock()
	if run, busy := c.runs[key]; busy {
		run.dirty = true
		c.mu.Unlock()
		select {
		case <-run.done:
			return run.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	run := &coalescedRun{done: make(chan struct{})}
	if c.runs == nil {
		c.runs = make(map[string]*coalescedRun)
	}
	c.runs[key] = run
	c.mu.Unlock()

	for {
		err := fn(ctx)
		c.mu.Lock()
		if run.dirty && err == nil && ctx.Err() == nil {
			run.dirty = false
			c.mu.Unlock()
			continue
		}
		run.err = err
		delete(c.runs, key)
		if err != nil {
			if c.failed == nil {
				c.failed = make(map[string]struct{})
			}
			c.failed[key] = struct{}{}
		} else {
			delete(c.failed, key)
		}
		c.mu.Unlock()
		close(run.done)
		return err
	}
}

// FailedLast dit si le dernier recalcul de key a échoué (et qu'aucun n'a réussi depuis).
func (c *Runs) FailedLast(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, failed := c.failed[key]
	return failed
}
