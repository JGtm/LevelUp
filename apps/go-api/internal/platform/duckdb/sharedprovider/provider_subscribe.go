package sharedprovider

import (
	"context"
	"log/slog"
	"sync"
)

// Subscribe enregistre un callback notifié après chaque transition vers RO.
// Retourne une fonction unsubscribe idempotente (sync.OnceFunc).
//
// Le callback est invoqué synchroniquement par la goroutine qui termine le
// swap, après que p.mu soit relâché — il est donc safe d'appeler
// Get/AcquireWriter depuis le callback (mais PAS Subscribe/Unsubscribe).
func (p *providerImpl) Subscribe(fn Subscriber) func() {
	p.subsMu.Lock()
	defer p.subsMu.Unlock()
	if p.subs == nil {
		p.subs = make(map[int64]Subscriber)
	}
	id := p.nextSubID
	p.nextSubID++
	p.subs[id] = fn

	return sync.OnceFunc(func() {
		p.subsMu.Lock()
		defer p.subsMu.Unlock()
		delete(p.subs, id)
	})
}

// notifyAfterSwap émet un SwapEvent à tous les Subscribers actifs.
// À utiliser pour les transitions RWToRO / ErrorToRO où le state est
// désormais "stable" (RO) et où les Subscribers peuvent légitimement
// appeler Get/AcquireWriter dans leur callback.
//
// IMPORTANT : NE PAS appeler avec p.mu tenu — un Subscriber qui prend
// p.mu (Get/AcquireWriter) deadlockerait. Pour la variante "sous p.mu"
// (PreSwapToRW exclusivement), utiliser notifyAfterSwapLocked.
//
// Sprint B1 commit 19 : ctx est passé aux Subscribers pour propager
// l'event_id du caller (sync.RunDelta typiquement) au callback pool.
// Pour les transitions sans ctx du caller (releaseWriter), le ctx est
// reconstruit avec l'event_id capturé via providerImpl.capturedEventID.
func (p *providerImpl) notifyAfterSwap(ctx context.Context, direction Direction, from, to State) {
	p.notifySubscribers(ctx, direction, from, to)
}

// notifyAfterSwapLocked émet un SwapEvent en supposant que le caller tient
// déjà p.mu. UNIQUEMENT pour DirectionPreSwapToRW : la fenêtre entre
// Close(handle RO Provider) et OpenReadWrite doit rester serialisée pour
// éviter qu'un Subscriber re-ATTACH shared et bloque l'open RW (cf. notes
// dans swapToRW).
//
// Contract : les Subscribers de PreSwapToRW NE DOIVENT PAS appeler
// Get/AcquireWriter dans leur callback (deadlock garanti via p.mu).
// La doc de DirectionPreSwapToRW dans subscriber.go le précise.
func (p *providerImpl) notifyAfterSwapLocked(ctx context.Context, direction Direction, from, to State) {
	if direction != DirectionPreSwapToRW {
		// Garde-fou : utilisation accidentelle hors PreSwapToRW. Log mais
		// continue — pas la peine de bloquer en runtime, le linter / tests
		// devraient flagger l'usage incorrect.
		slog.ErrorContext(ctx, "sharedprovider: notifyAfterSwapLocked called with non-PreSwap direction",
			"direction", direction, "path", p.path)
	}
	p.notifySubscribers(ctx, direction, from, to)
}

// notifySubscribers est le fan-out interne partagé par notifyAfterSwap et
// notifyAfterSwapLocked. Snapshot des subs sous subsMu pour éviter race
// avec Unsubscribe concurrent ; appel des callbacks hors subsMu.
func (p *providerImpl) notifySubscribers(ctx context.Context, direction Direction, from, to State) {
	p.subsMu.Lock()
	if len(p.subs) == 0 {
		p.subsMu.Unlock()
		return
	}
	subs := make([]Subscriber, 0, len(p.subs))
	for _, fn := range p.subs {
		subs = append(subs, fn)
	}
	p.subsMu.Unlock()

	evt := SwapEvent{Direction: direction, From: from, To: to, Path: p.path}
	for _, fn := range subs {
		// Pas de recover ici : un Subscriber qui panic indique un bug applicatif
		// — laisser remonter pour ne pas masquer le problème. Le swap est
		// déjà terminé côté Provider, le state est cohérent.
		fn(ctx, evt)
	}
}
