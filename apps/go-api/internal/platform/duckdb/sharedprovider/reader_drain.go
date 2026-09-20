package sharedprovider

// reader_drain.go — comptage des lecteurs en vol et attente de leur drain.
//
// POURQUOI PAS UN sync.WaitGroup (incident du 2026-09-16, deux crashs).
// Le drain était un `sync.WaitGroup` attendu dans une goroutine jetable :
//
//	go func() { p.readersWG.Wait(); close(done) }()
//	select { case <-done: ...; case <-ctx.Done(): return ctx.Err() }
//
// Sur expiration du drain, `waitForDrain` rendait la main en LAISSANT cette
// goroutine bloquée dans `Wait`. `rollbackFromDraining` repassait en StateRO, les
// `Get` reprenaient leurs `Add(1)`, et quand le compteur retombait à zéro la
// `Wait` orpheline se réveillait — un `Add(1)` concurrent dans cette fenêtre
// déclenche alors « panic: sync: WaitGroup is reused before previous Wait has
// returned » (sync/waitgroup.go). Le `Close()` best-effort avait le même motif.
//
// MODÈLE RETENU : un compteur entier protégé par `p.mu` (le même mutex qui garde
// déjà la machine à états — l'incrément d'un lecteur et la lecture de l'état sont
// donc ATOMIQUES ensemble, propriété qu'avait déjà l'ancien code) et un canal de
// drain créé à la demande. Le DERNIER `release()` ferme ce canal quand un drain
// attend ; une expiration l'abandonne sous `p.mu`. Aucune goroutine d'attente,
// donc aucun réveil différé : un drain abandonné ne peut plus interférer avec les
// suivants.
//
// Les métriques (`readersInUse`) et la machine à états (ADR 0013/0016) sont
// inchangées.

// trackReaderLocked comptabilise un lecteur qui vient d'obtenir le handle.
// Doit être appelé avec p.mu tenu, AVANT de le relâcher — sinon un swap
// concurrent drainerait sans nous attendre.
func (p *providerImpl) trackReaderLocked() {
	p.readers++
	readersInUse.Add(1)
}

// releaseReader comptabilise la fin d'un lecteur et réveille le drain en attente
// si c'était le dernier. Appelé une seule fois par lecteur (sync.OnceFunc).
func (p *providerImpl) releaseReader() {
	p.mu.Lock()
	p.readers--
	if p.readers <= 0 && p.drainCh != nil {
		close(p.drainCh)
		p.drainCh = nil
	}
	p.mu.Unlock()
	readersInUse.Add(-1)
}

// beginDrain déclare l'attente du drain des lecteurs en vol et rend le canal
// fermé quand il n'en reste plus. Sans lecteur en vol, le canal rendu est DÉJÀ
// fermé (drain immédiat). Deux demandes concurrentes (AcquireWriter + Close)
// partagent le même canal : la fermeture les réveille toutes les deux.
func (p *providerImpl) beginDrain() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.readers <= 0 {
		return newClosedChan()
	}
	if p.drainCh == nil {
		p.drainCh = make(chan struct{})
	}
	return p.drainCh
}

// abandonDrain renonce à l'attente rendue par beginDrain (expiration, annulation
// du ctx). Le canal n'est jamais fermé : plus personne ne l'écoute, et le
// prochain beginDrain en ouvrira un neuf. No-op si un autre drain a déjà pris la
// place (comparaison d'identité du canal).
func (p *providerImpl) abandonDrain(ch <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.drainCh != nil && (<-chan struct{})(p.drainCh) == ch {
		p.drainCh = nil
	}
}
