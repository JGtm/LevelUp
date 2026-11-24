package pool

// pool_background.go — la résolution des comptes en tâche de fond (PoolOptions.ResolveInBackground).
//
// Au démarrage, résoudre tous les comptes avant d'ouvrir le serveur coûtait la somme des vagues de six
// résolutions (jusqu'à 20 s chacune : 122 s mesurées avec douze comptes). Ici le pool naît vide et un
// seul coordinateur résout les comptes ; chacun entre dans le pool dès sa résolution.
//
// CE QUE LE POOL NE FAIT JAMAIS PENDANT CE TEMPS :
//   - servir un jeton avant qu'il soit résolu : une acquisition attend, ou échoue avec son contexte ;
//   - servir un compte à la mise en commun sans son consentement : l'état de consentement d'un compte à
//     résoudre vit dans `pending`, appliqué à son slot à la création, et SetShared le tient à jour ;
//   - laisser un propriétaire sans accord emprunter les jetons des autres : IsWithheld lit `pending`.
//
// Verrous : slotMu puis fill.mu, toujours dans cet ordre.

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// closeFillWait borne l'attente de Close sur la fin de la résolution annulée : une résolution qui
// ignorerait son contexte ne bloque pas l'arrêt, mais le journal le dit.
const closeFillWait = 15 * time.Second

// bootFill : l'état d'une résolution de fond. Nil sur un pool construit sans ResolveInBackground.
type bootFill struct {
	mu      sync.Mutex
	running bool
	title   string
	// pending : les comptes encore à résoudre, par clé de slot. Porte l'état de consentement courant.
	pending map[string]CredentialSource
	// wake est fermé, puis remplacé, à chaque slot installé, échec constaté ou fin de résolution.
	wake   chan struct{}
	cancel context.CancelFunc
	done   chan struct{}
}

func newBootFill(title string, sorted []CredentialSource, cancel context.CancelFunc) *bootFill {
	f := &bootFill{
		running: len(sorted) > 0,
		title:   title,
		pending: make(map[string]CredentialSource, len(sorted)),
		wake:    make(chan struct{}),
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	for _, src := range sorted {
		f.pending[gtKey(title, src.Gamertag)] = src
	}
	return f
}

// key rend la clé de slot du gamertag, celle des comptes à résoudre.
func (f *bootFill) key(gamertag string) string { return gtKey(f.title, gamertag) }

// signalLocked réveille tous les acquéreurs en attente. fill.mu tenu.
func (f *bootFill) signalLocked() {
	close(f.wake)
	f.wake = make(chan struct{})
}

// pendingSource rend la source encore à résoudre de cette clé, avec son consentement courant.
func (f *bootFill) pendingSource(key string) (CredentialSource, bool) {
	if f == nil {
		return CredentialSource{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	src, ok := f.pending[key]
	return src, ok
}

// take retire la source de la liste des comptes à résoudre et la rend telle qu'elle est maintenant
// (consentement à jour). Rend faux si elle n'y était plus.
func (f *bootFill) take(key string) (CredentialSource, bool) {
	if f == nil {
		return CredentialSource{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	src, ok := f.pending[key]
	if ok {
		delete(f.pending, key)
		f.signalLocked()
	}
	return src, ok
}

// setShared applique un changement de consentement au compte encore à résoudre qui porte ce xuid.
// slotMu tenu par l'appelant : le slot ne peut pas apparaître entre sa recherche et cet appel.
func (f *bootFill) setShared(xuid string, shared bool) bool {
	if f == nil || xuid == "" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, src := range f.pending {
		if src.XUID == xuid {
			src.PoolExcluded = !shared
			f.pending[key] = src
			return true
		}
	}
	return false
}

// watch rend le canal de réveil courant et dit si la résolution est en cours. À prendre AVANT l'essai
// d'acquisition dont l'échec mène à l'attente.
func (f *bootFill) watch() (<-chan struct{}, bool) {
	if f == nil {
		return nil, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wake, f.running
}

// signal réveille les acquisitions en attente, si une résolution est en cours.
func (f *bootFill) signal() {
	if f == nil {
		return
	}
	f.mu.Lock()
	if f.running {
		f.signalLocked()
	}
	f.mu.Unlock()
}

// waitForUsableSlot attend, pendant la résolution de fond, que l'acquisition publique ait une chance
// d'aboutir : un réveil du pool (slot installé, retour de santé, consentement rendu, fin de résolution), ou
// l'échéance du plus proche repos de slot (429) qui finit après `essaiA`, l'instant de l'essai infructueux
// (un repos déjà échu à l'attente compte : l'attente repart aussitôt). Rend faux si ctx se termine avant.
func (p *poolImpl) waitForUsableSlot(ctx context.Context, wake <-chan struct{}, essaiA time.Time) bool {
	var timeout <-chan time.Time
	if d, ok := p.untilNextRestEnd(essaiA); ok {
		timer := time.NewTimer(d)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-wake:
		return true
	case <-timeout:
		return true
	case <-ctx.Done():
		return false
	}
}

// untilNextRestEnd rend le délai (nul si déjà échu) avant la fin du plus proche repos de slot (429) qui finit
// après `depuis`, faux s'il n'y en a pas.
func (p *poolImpl) untilNextRestEnd(depuis time.Time) (time.Duration, bool) {
	p.slotMu.RLock()
	slots := p.slots
	p.slotMu.RUnlock()
	var next time.Time
	for _, s := range slots {
		s.mu.RLock()
		until := s.rateLimitedUntil
		s.mu.RUnlock()
		if until.After(depuis) && (next.IsZero() || until.Before(next)) {
			next = until
		}
	}
	if next.IsZero() {
		return 0, false
	}
	return max(0, time.Until(next)), true
}

// waitIfPending attend, pour une acquisition épinglée, un changement de la résolution (slot installé, échec, fin) :
// seulement si ce compte est encore à résoudre. Rend faux, sans attendre, sinon ou quand ctx est terminé.
func (f *bootFill) waitIfPending(ctx context.Context, key string) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	if _, ok := f.pending[key]; !ok || !f.running {
		f.mu.Unlock()
		return false
	}
	return f.waitWake(ctx)
}

// waitWake libère fill.mu (tenu à l'appel) et attend le prochain réveil ou la fin de ctx.
func (f *bootFill) waitWake(ctx context.Context) bool {
	ch := f.wake
	f.mu.Unlock()
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// finish clôt la résolution : plus aucun compte à résoudre, les attentes sont réveillées.
func (f *bootFill) finish() {
	f.mu.Lock()
	f.running = false
	f.pending = map[string]CredentialSource{}
	f.signalLocked()
	f.mu.Unlock()
	close(f.done)
}

// startBackgroundPool construit un pool vide dont les comptes de sorted se résolvent en tâche de fond.
func startBackgroundPool(
	ctx context.Context, resolver Resolver, title string, sorted []CredentialSource, opts PoolOptions,
) Pool {
	fillCtx, cancel := context.WithCancel(ctx)
	// Capacité du tour de rôle : tous les comptes à venir, plus la marge des ajouts à chaud.
	p := newPoolImpl(resolver, title, nil, map[string]int{}, len(sorted)+hotAddPublicSlots, opts)
	p.fill = newBootFill(title, sorted, cancel)
	slog.InfoContext(ctx, "pool: créé vide, comptes en cours de résolution",
		"sources", len(sorted), "perTokenRPS", opts.PerTokenRPS)
	if len(sorted) == 0 {
		p.fill.finish()
		cancel()
	} else {
		go p.runBootFill(fillCtx, sorted, opts.MaxSize)
	}
	go p.refresherLoop(context.Background())
	return p
}

// runBootFill résout les comptes et installe chaque slot dès sa résolution, puis clôt la résolution.
func (p *poolImpl) runBootFill(ctx context.Context, sorted []CredentialSource, maxSize int) {
	debut := time.Now()
	failed := 0
	runBootResolutionsNotify(ctx, p.resolver, sorted, maxSize, func(i int, o bootOutcome) {
		src := sorted[i]
		if o.err != nil {
			failed++
			slog.WarnContext(ctx, "pool: impossible de résoudre token au boot, skip slot",
				"gamertag", src.Gamertag, "err", o.err)
			p.fill.take(p.fill.key(src.Gamertag))
			return
		}
		p.installBootSlot(ctx, src, o.resolved)
	})
	// Interrompue : constaté AVANT de relâcher le contexte de la résolution (cancel le rendrait toujours annulé).
	interrompue := ctx.Err() != nil
	size := p.Size()
	attrs := []any{"size", size, "sources", len(sorted), "echecs", failed,
		"resolve_ms", time.Since(debut).Milliseconds(), "interrompue", interrompue}
	switch {
	case interrompue:
		slog.WarnContext(ctx, "pool: résolution des comptes interrompue", attrs...)
	case size == 0:
		slog.WarnContext(ctx, "pool: aucun compte résolu — authentifier un joueur (SSO Xbox ou `go run ./cmd/token-capture/ <GT>`) : le pool se remplira à sa connexion ou au re-scan",
			attrs...)
	default:
		slog.InfoContext(ctx, "pool: comptes résolus", attrs...)
	}
	// Le bilan est écrit avant la clôture : ceux qui attendent la fin (WaitResolved) le trouvent au journal.
	p.fill.finish()
	p.fill.cancel()
}

// installBootSlot installe le slot d'un compte résolu en tâche de fond. Le consentement est celui du
// moment (pending), jamais celui du scan. Un slot déjà posé par un ajout à chaud plus récent (re-scan,
// connexion) est conservé tel quel.
func (p *poolImpl) installBootSlot(ctx context.Context, src CredentialSource, resolved *ResolvedTokens) {
	key := p.fill.key(src.Gamertag)
	p.slotMu.Lock()
	defer p.slotMu.Unlock()
	current, wasPending := p.fill.take(key)
	if _, exists := p.slotsByKey[key]; exists {
		return
	}
	if !wasPending {
		current = src
	}
	p.appendSlotLocked(ctx, current, resolved, key)
}

// WaitResolved attend la fin de la résolution de fond. Rend vrai tout de suite sans résolution de fond ;
// faux si ctx se termine avant la fin.
func (p *poolImpl) WaitResolved(ctx context.Context) bool {
	if p.fill == nil {
		return true
	}
	select {
	case <-p.fill.done:
		return true
	case <-ctx.Done():
		return false
	}
}

// WaitResolved attend que le pool p ait fini de résoudre ses comptes en tâche de fond : les tâches de
// démarrage qui ne se rejouent pas avant longtemps (catalogue, noms d'assets) l'appellent avant leur
// première passe plutôt que de trouver un pool vide. Vrai tout de suite pour un pool sans résolution de
// fond (ou nil) ; faux si ctx se termine avant la fin.
func WaitResolved(ctx context.Context, p Pool) bool {
	w, ok := p.(interface{ WaitResolved(context.Context) bool })
	if !ok {
		return true
	}
	return w.WaitResolved(ctx)
}

// stopFill annule la résolution de fond et en attend la fin (Close).
func (p *poolImpl) stopFill() {
	if p.fill == nil {
		return
	}
	p.fill.cancel()
	select {
	case <-p.fill.done:
	case <-time.After(closeFillWait):
		slog.WarnContext(context.Background(), "pool: la résolution des comptes n'a pas rendu la main à l'arrêt",
			"attente", closeFillWait)
	}
}
