// Package coalesce — recalculs coûteux sérialisés et absorbés par clé.
package coalesce

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Runs sérialise et absorbe les recalculs demandés pour une même clé
// (une base joueur). Une exclusion de match déclenche un recalcul de tout l'historique du
// joueur : des demandes répétées ne doivent ni s'empiler ni tourner en parallèle.
//
// Contrat de Do : à son retour sans erreur de contexte, un recalcul DÉMARRÉ APRÈS le dépôt
// de la demande s'est terminé. Le drapeau d'exclusion étant persisté avant l'appel, ce
// recalcul le voit.
//   - aucun recalcul en cours pour la clé : un recalcul démarre ;
//   - un recalcul est en cours : la demande est notée (dirty) ; à la fin du recalcul en
//     cours, UN seul recalcul de rattrapage sert toutes les demandes notées.
//
// Le recalcul tourne dans sa propre goroutine, sous un contexte DÉTACHÉ de celui du
// demandeur qui l'a lancé (context.WithoutCancel : ses valeurs, jamais son annulation ni
// son échéance) : l'abandon de ce demandeur n'interrompt ni le recalcul ni le rattrapage
// dû aux autres. fn borne elle-même ses attentes. Chaque demandeur attend au plus jusqu'à
// l'annulation de son propre contexte, qu'il reçoit alors en erreur.
//
// Les demandeurs reçoivent l'erreur du dernier recalcul. Une clé dont le dernier recalcul a
// échoué est retenue (FailedLast) jusqu'au prochain succès : l'appelant sait alors qu'une
// demande identique à l'état déjà persisté doit tout de même recalculer.
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

// Do exécute fn pour key en respectant le contrat ci-dessus. ctx borne l'attente du
// demandeur ; il n'interrompt jamais un recalcul.
func (c *Runs) Do(ctx context.Context, key string, fn func(context.Context) error) error {
	c.mu.Lock()
	if run, busy := c.runs[key]; busy {
		run.dirty = true
		c.mu.Unlock()
		return run.wait(ctx)
	}
	run := &coalescedRun{done: make(chan struct{})}
	if c.runs == nil {
		c.runs = make(map[string]*coalescedRun)
	}
	c.runs[key] = run
	c.mu.Unlock()

	go c.loop(context.WithoutCancel(ctx), key, run, fn)
	return run.wait(ctx)
}

// wait attend la fin du dernier recalcul de run, au plus jusqu'à l'annulation de ctx.
func (run *coalescedRun) wait(ctx context.Context) error {
	select {
	case <-run.done:
		return run.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// loop enchaîne le recalcul et ses rattrapages tant que des demandes ont été absorbées,
// puis publie le résultat aux demandeurs.
func (c *Runs) loop(ctx context.Context, key string, run *coalescedRun, fn func(context.Context) error) {
	for {
		err := runGuarded(ctx, key, fn)
		c.mu.Lock()
		if run.dirty && err == nil {
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
		return
	}
}

// runGuarded exécute fn ; une panique devient une erreur (journalisée) : les demandeurs en
// attente reçoivent un résultat et le process ne tombe pas.
func runGuarded(ctx context.Context, key string, fn func(context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("coalesce: recalcul de %q en panique : %v", key, r)
			slog.ErrorContext(ctx, "coalesce: recalcul en panique", "key", key, "err", err)
		}
	}()
	return fn(ctx)
}

// FailedLast dit si le dernier recalcul de key a échoué (et qu'aucun n'a réussi depuis).
func (c *Runs) FailedLast(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, failed := c.failed[key]
	return failed
}
