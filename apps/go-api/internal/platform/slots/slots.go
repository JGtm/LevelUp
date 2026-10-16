// Package slots — nombre borné de titulaires simultanés d'un travail coûteux.
//
// Un Pool de n places sert de sémaphore global : les envois de médias (2 en vol) et le
// transcodage ffmpeg (1 à la fois) s'y rangent pour qu'aucune salve de requêtes ne
// multiplie la mémoire ou les processus. Acquire attend son tour et rend la main quand
// son contexte finit : l'appelant décide de l'attente maximale.
package slots

import (
	"context"
	"sync/atomic"
)

// Pool est un sémaphore de capacité fixe.
type Pool struct {
	places  chan struct{}
	waiting atomic.Int32 // demandes bloquées faute de place
}

// New crée un Pool de n places (n < 1 est ramené à 1).
func New(n int) *Pool {
	if n < 1 {
		n = 1
	}
	return &Pool{places: make(chan struct{}, n)}
}

// Acquire prend une place, en attendant si le pool est plein. La fonction rendue libère
// la place ; l'appeler plusieurs fois est sans effet. Erreur : le contexte, si son
// échéance tombe avant qu'une place se libère.
func (p *Pool) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case p.places <- struct{}{}:
	default:
		if err := p.wait(ctx); err != nil {
			return nil, err
		}
	}
	released := make(chan struct{})
	return func() {
		select {
		case <-released:
		default:
			close(released)
			<-p.places
		}
	}, nil
}

// wait attend une place, compté parmi les demandes en attente le temps de l'attente.
func (p *Pool) wait(ctx context.Context) error {
	p.waiting.Add(1)
	defer p.waiting.Add(-1)
	select {
	case p.places <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InUse rend le nombre de places tenues à cet instant.
func (p *Pool) InUse() int { return len(p.places) }

// Waiting rend le nombre de demandes bloquées en attente d'une place à cet instant.
func (p *Pool) Waiting() int { return int(p.waiting.Load()) }
