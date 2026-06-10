package auth

import (
	"context"
	"sync"
	"time"
)

// CachedHeaderProvider mémoïse un header d'authentification (typiquement le header
// RTA "XBL3.0 x=<hash>;<token>" pour PeopleHub) et le reconstruit seulement après
// expiration du TTL. Thread-safe : le PeopleHubResolver peut être appelé en
// parallèle (fan-out de l'agrégateur), une seule reconstruction à la fois.
//
// Le `build` encapsule le chemin auth coûteux (charger le token → un access_token
// frais → AcquireXSTSForRTA → header) ; le caller (cmd/server) le fournit. Sa
// méthode Header satisfait le `headerFn` de NewPeopleHubResolver.
type CachedHeaderProvider struct {
	mu        sync.Mutex
	build     func(ctx context.Context) (string, error)
	ttl       time.Duration
	now       func() time.Time
	header    string
	expiresAt time.Time
}

// NewCachedHeaderProvider construit le provider. ttl <= 0 → défaut 3h (Spartan/XSTS
// ~4h, marge de sécurité). `build` est obligatoire.
func NewCachedHeaderProvider(ttl time.Duration, build func(ctx context.Context) (string, error)) *CachedHeaderProvider {
	if ttl <= 0 {
		ttl = 3 * time.Hour
	}
	return &CachedHeaderProvider{build: build, ttl: ttl, now: time.Now}
}

// Header retourne le header mémoïsé, le (re)construisant si absent ou expiré.
func (p *CachedHeaderProvider) Header(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.header != "" && p.now().Before(p.expiresAt) {
		return p.header, nil
	}
	h, err := p.build(ctx)
	if err != nil {
		return "", err
	}
	p.header = h
	p.expiresAt = p.now().Add(p.ttl)
	return h, nil
}
