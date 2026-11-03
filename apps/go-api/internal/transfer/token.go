package transfer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// Réglages du jeton d'export et de la limite.
const (
	// TokenTTL : durée de vie d'un jeton d'export (délivré dans les réglages du site, collé dans l'application).
	TokenTTL = 15 * time.Minute
	// ExportInterval : un joueur n'obtient qu'un export par heure.
	ExportInterval = time.Hour
	// maxPendingTokens borne la mémoire du magasin (jetons non échus).
	maxPendingTokens = 1000
)

// Erreurs du magasin.
var (
	ErrTokenInvalid = errors.New("transfer: jeton d'export inconnu, échu ou déjà utilisé")
	ErrRateLimited  = errors.New("transfer: un export par heure et par joueur")
)

// Grant : ce qu'un jeton d'export ouvre : l'export du joueur qui l'a demandé, et de lui seul.
type Grant struct {
	Xuid, Gamertag string
}

type pending struct {
	grant   Grant
	expires time.Time
}

// TokenStore garde les jetons d'export à usage unique et la limite d'un export par heure et par joueur. Tout
// est en mémoire : un redémarrage du serveur invalide les jetons en attente (le joueur en redemande un).
// Seule l'empreinte SHA-256 d'un jeton est gardée, jamais le jeton.
type TokenStore struct {
	mu      sync.Mutex
	tokens  map[string]pending
	lastRun map[string]time.Time
}

// NewTokenStore crée un magasin vide.
func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: map[string]pending{}, lastRun: map[string]time.Time{}}
}

func fingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Issue délivre un jeton court à usage unique pour le joueur grant, valable [TokenTTL]. Un joueur qui en
// redemande un invalide le précédent (un seul jeton en attente par joueur).
func (s *TokenStore) Issue(g Grant, now time.Time) (token string, expires time.Time, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token = hex.EncodeToString(buf)
	expires = now.Add(TokenTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep(now)
	for fp, p := range s.tokens {
		if p.grant.Xuid == g.Xuid {
			delete(s.tokens, fp)
		}
	}
	if len(s.tokens) >= maxPendingTokens {
		return "", time.Time{}, errors.New("transfer: trop de jetons en attente")
	}
	s.tokens[fingerprint(token)] = pending{grant: g, expires: expires}
	return token, expires, nil
}

// Redeem consomme un jeton : il n'est valable qu'une fois et avant son échéance. La limite d'un export par
// heure s'applique à la consommation (ErrRateLimited, le jeton reste alors valable). Journalisé.
func (s *TokenStore) Redeem(token string, now time.Time) (Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep(now)
	fp := fingerprint(token)
	p, ok := s.tokens[fp]
	if !ok {
		return Grant{}, ErrTokenInvalid
	}
	if last, ran := s.lastRun[p.grant.Xuid]; ran && now.Sub(last) < ExportInterval {
		slog.Warn("transfer: export refusé, limite d'un par heure", "xuid", p.grant.Xuid, "dernier", last)
		return Grant{}, ErrRateLimited
	}
	delete(s.tokens, fp)
	s.lastRun[p.grant.Xuid] = now
	slog.Info("transfer: jeton d'export consommé", "xuid", p.grant.Xuid)
	return p.grant, nil
}

// AllowSession applique la limite d'un export par heure à un export demandé par la session connectée
// (sans jeton) ; true et le compte est tenu.
func (s *TokenStore) AllowSession(xuid string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if last, ran := s.lastRun[xuid]; ran && now.Sub(last) < ExportInterval {
		slog.Warn("transfer: export refusé, limite d'un par heure", "xuid", xuid, "dernier", last)
		return false
	}
	s.lastRun[xuid] = now
	slog.Info("transfer: export de la session", "xuid", xuid)
	return true
}

// sweep retire les jetons échus (verrou tenu).
func (s *TokenStore) sweep(now time.Time) {
	for fp, p := range s.tokens {
		if !now.Before(p.expires) {
			delete(s.tokens, fp)
		}
	}
}
