package pool

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/time/rate"
)

// sharedLeaseData est leaseData réservé au round-robin public : un slot exclu
// (propriétaire sans consentement) n'est pas acquérable, quel que soit son état de santé.
// L'état d'exclusion et la santé sont lus sous UN SEUL RLock.
func (s *slot) sharedLeaseData(now time.Time) (resolved *ResolvedTokens, gamertag string, limiter *rate.Limiter, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.excluded || !s.healthy || now.Before(s.rateLimitedUntil) {
		return nil, "", nil, false
	}
	return s.resolved, s.gamertag, s.limiter, true
}

// slotFor retourne le slot d'un gamertag (nil si absent), index lu sous slotMu.
func (p *poolImpl) slotFor(gamertag string) *slot {
	p.slotMu.RLock()
	defer p.slotMu.RUnlock()
	idx, ok := p.slotsByKey[gtKey(p.titleSlug, gamertag)]
	if !ok {
		return nil
	}
	return p.slots[idx]
}

// IsWithheld implémente Pool.IsWithheld().
func (p *poolImpl) IsWithheld(gamertag string) bool {
	s := p.slotFor(gamertag)
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.excluded
}

// slotForXUID retourne le slot qui porte ce xuid (nil si aucun). Parcours linéaire : le
// pool compte quelques dizaines de slots au plus, et l'appel suit une action utilisateur.
func (p *poolImpl) slotForXUID(xuid string) *slot {
	if xuid == "" {
		return nil
	}
	p.slotMu.RLock()
	defer p.slotMu.RUnlock()
	for _, s := range p.slots {
		if s.xuidSnapshot() == xuid {
			return s
		}
	}
	return nil
}

// SetShared implémente Pool.SetShared().
func (p *poolImpl) SetShared(xuid string, shared bool) bool {
	s := p.slotForXUID(xuid)
	if s == nil {
		return false
	}
	s.mu.Lock()
	changed := s.excluded == shared
	s.excluded = !shared
	gamertag := s.gamertag
	s.mu.Unlock()
	if changed {
		slog.InfoContext(context.Background(), "pool: consentement de mise en commun appliqué",
			"gamertag", gamertag, "xuid", xuid, "shared", shared)
	}
	return true
}
