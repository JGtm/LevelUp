// Package coalesce — recalculs coûteux sérialisés et absorbés par clé.
package coalesce

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MaxRunDuration borne chaque recalcul (et chaque rattrapage) lancé par une Runs.
const MaxRunDuration = 10 * time.Minute

// DrainWait borne l'attente, à l'arrêt du process, des recalculs en vol que Drain vient
// d'annuler (cmd/server). Elle reste bien sous le budget de la séquence d'arrêt
// (cmd/server/shutdown.go, borné sous le stop_grace_period de docker-compose.yml ; air :
// kill_delay de 5 s) pour que la suite (files de persistance, fermeture des bases) ait lieu.
const DrainWait = 2 * time.Second

// ErrDraining est rendue à toute demande déposée après le début de l'arrêt (Drain) : aucun
// recalcul n'est lancé ni absorbé.
var ErrDraining = errors.New("coalesce: arrêt en cours, recalcul refusé")

// Runs sérialise et absorbe les recalculs demandés pour une même clé
// (une base joueur). Une exclusion de match déclenche un recalcul de tout l'historique du
// joueur : des demandes répétées ne doivent ni s'empiler ni tourner en parallèle.
//
// Contrat de Do : à son retour sans erreur, un recalcul DÉMARRÉ APRÈS le dépôt de la
// demande s'est terminé. Le drapeau d'exclusion étant persisté avant l'appel, ce recalcul
// le voit.
//   - aucun recalcul en cours pour la clé : un recalcul démarre ;
//   - un recalcul est en cours : la demande est notée (dirty) ; à la fin du recalcul en
//     cours, UN seul recalcul de rattrapage sert toutes les demandes notées.
//
// Le recalcul tourne dans sa propre goroutine, sous un contexte DÉTACHÉ de celui du
// demandeur qui l'a lancé (context.WithoutCancel : ses valeurs, jamais son annulation ni
// son échéance) : l'abandon de ce demandeur n'interrompt ni le recalcul ni le rattrapage
// dû aux autres. Chaque recalcul porte sa propre échéance (MaxRunDuration) : à son terme,
// le contexte passé à fn est annulé et l'erreur est journalisée puis rendue aux
// demandeurs. Chaque demandeur attend au plus jusqu'à l'annulation de son propre contexte,
// qu'il reçoit alors en erreur.
//
// Les recalculs en vol de toutes les Runs du process sont suivis : Drain, appelé à l'arrêt
// avant la fermeture des bases, refuse toute nouvelle demande (ErrDraining), annule les
// recalculs en vol (cause ErrDraining : aucun rattrapage n'est plus lancé) et attend leur
// fin, au plus jusqu'à l'échéance qu'on lui donne.
//
// Les demandeurs reçoivent l'erreur du dernier recalcul. Une clé dont le dernier recalcul a
// échoué est retenue (FailedLast) jusqu'au prochain succès : l'appelant sait alors qu'une
// demande identique à l'état déjà persisté doit tout de même recalculer.
// Sa valeur zéro est prête.
type Runs struct {
	mu     sync.Mutex
	runs   map[string]*coalescedRun
	failed map[string]struct{}
	// maxDuration remplace MaxRunDuration (tests) ; zéro : MaxRunDuration.
	maxDuration time.Duration
}

// coalescedRun est l'état d'une clé dont un recalcul est en cours.
type coalescedRun struct {
	dirty bool          // une demande est arrivée pendant le recalcul courant
	done  chan struct{} // fermé quand le dernier recalcul est terminé
	err   error         // erreur du dernier recalcul, lisible après done
}

// Do exécute fn pour key en respectant le contrat ci-dessus. ctx borne l'attente du
// demandeur ; il n'interrompt jamais un recalcul. Après le début de l'arrêt (Drain), rend
// ErrDraining sans rien lancer ni absorber.
func (c *Runs) Do(ctx context.Context, key string, fn func(context.Context) error) error {
	tracker := inflight
	c.mu.Lock()
	if run, busy := c.runs[key]; busy {
		if tracker.draining() {
			c.mu.Unlock()
			return ErrDraining
		}
		run.dirty = true
		c.mu.Unlock()
		return run.wait(ctx)
	}
	if !tracker.begin() {
		c.mu.Unlock()
		return ErrDraining
	}
	run := &coalescedRun{done: make(chan struct{})}
	if c.runs == nil {
		c.runs = make(map[string]*coalescedRun)
	}
	c.runs[key] = run
	c.mu.Unlock()

	go func() {
		defer tracker.end()
		c.loop(context.WithoutCancel(ctx), tracker, key, run, fn)
	}()
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
// puis publie le résultat aux demandeurs. Une fois l'arrêt commencé, aucun rattrapage n'est
// lancé : les demandes absorbées reçoivent ErrDraining.
func (c *Runs) loop(ctx context.Context, tracker *inflightRuns, key string, run *coalescedRun, fn func(context.Context) error) {
	for {
		err := c.runBounded(ctx, tracker, key, fn)
		c.mu.Lock()
		if run.dirty && err == nil && tracker.draining() {
			err = ErrDraining
		}
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

// runBounded exécute un recalcul sous son échéance et sous l'arrêt du process (Drain annule
// son contexte, cause ErrDraining). Un recalcul interrompu par l'une ou l'autre rend une
// erreur qui la nomme, journalisée ; celle de l'arrêt enveloppe ErrDraining.
func (c *Runs) runBounded(ctx context.Context, tracker *inflightRuns, key string, fn func(context.Context) error) error {
	bound := c.maxDuration
	if bound <= 0 {
		bound = MaxRunDuration
	}
	stopCtx, stop := context.WithCancelCause(ctx)
	defer stop(nil)
	unwatch := context.AfterFunc(tracker.shutdownContext(), func() { stop(ErrDraining) })
	defer unwatch()
	runCtx, cancel := context.WithTimeout(stopCtx, bound)
	defer cancel()
	err := runGuarded(runCtx, key, fn)
	switch {
	case err == nil:
	case errors.Is(context.Cause(runCtx), ErrDraining):
		err = fmt.Errorf("coalesce: recalcul de %q interrompu par l'arrêt : %w (%w)", key, ErrDraining, err)
		slog.WarnContext(ctx, "coalesce: recalcul interrompu par l'arrêt", "key", key, "err", err)
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		err = fmt.Errorf("coalesce: recalcul de %q interrompu à son échéance (%s) : %w", key, bound, err)
		slog.ErrorContext(ctx, "coalesce: recalcul interrompu à son échéance",
			"key", key, "max_duration", bound.String(), "err", err)
	}
	return err
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

// Drain refuse toute nouvelle demande de toutes les Runs du process (ErrDraining), annule les
// recalculs en vol (leur contexte, cause ErrDraining ; aucun rattrapage n'est plus lancé) puis
// attend leur fin, au plus jusqu'à l'annulation de ctx (erreur qui l'enveloppe). À appeler une
// fois, à l'arrêt, avant la fermeture des bases, sous une échéance courte (DrainWait).
func Drain(ctx context.Context) error {
	return inflight.drain(ctx)
}

// inflight suit les recalculs en vol de toutes les Runs du process.
var inflight = &inflightRuns{}

// inflightRuns compte les boucles de recalcul en vol, ferme l'admission à l'arrêt et annule
// alors les recalculs en vol. Sa valeur zéro est prête.
type inflightRuns struct {
	mu     sync.Mutex
	n      int
	closed bool
	idle   chan struct{} // fermé quand n retombe à zéro pendant un drain ; nil sinon
	// shutdown est annulé (cause ErrDraining) au début de l'arrêt ; créé à la première demande.
	shutdown       context.Context
	cancelShutdown context.CancelCauseFunc
}

// shutdownContext rend le contexte annulé au début de l'arrêt.
func (t *inflightRuns) shutdownContext() context.Context {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initShutdownLocked()
	return t.shutdown
}

func (t *inflightRuns) initShutdownLocked() {
	if t.shutdown == nil {
		t.shutdown, t.cancelShutdown = context.WithCancelCause(context.Background())
	}
}

// begin admet une boucle de recalcul ; false une fois l'arrêt commencé.
func (t *inflightRuns) begin() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return false
	}
	t.n++
	return true
}

// end retire une boucle terminée et réveille le drain quand plus aucune n'est en vol.
func (t *inflightRuns) end() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n--
	if t.n == 0 && t.idle != nil {
		close(t.idle)
		t.idle = nil
	}
}

// draining dit si l'arrêt a commencé.
func (t *inflightRuns) draining() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func (t *inflightRuns) drain(ctx context.Context) error {
	t.mu.Lock()
	t.closed = true
	t.initShutdownLocked()
	t.cancelShutdown(ErrDraining)
	if t.n == 0 {
		t.mu.Unlock()
		return nil
	}
	if t.idle == nil {
		t.idle = make(chan struct{})
	}
	idle := t.idle
	t.mu.Unlock()
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		t.mu.Lock()
		pending := t.n
		t.mu.Unlock()
		return fmt.Errorf("coalesce: %d recalcul(s) encore en vol à l'échéance de l'arrêt : %w", pending, ctx.Err())
	}
}
