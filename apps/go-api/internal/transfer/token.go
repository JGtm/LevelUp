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
	// ExportInterval : un joueur n'obtient qu'un export RÉUSSI par heure.
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
	mu       sync.Mutex
	tokens   map[string]pending
	lastRun  map[string]time.Time
	inFlight map[string]bool
}

// NewTokenStore crée un magasin vide.
func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: map[string]pending{}, lastRun: map[string]time.Time{}, inFlight: map[string]bool{}}
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

// Redeem consomme un jeton : il n'est valable qu'une fois et avant son échéance. Le joueur est alors EN COURS
// d'export ; l'appelant rend la main par [TokenStore.Finish], qui seul fait entrer l'export dans la limite d'un
// par heure (un export qui échoue ne consomme pas le quota). ErrRateLimited : un export réussi date de moins
// d'une heure ou un export du joueur est en cours (le jeton reste alors valable). Journalisé.
func (s *TokenStore) Redeem(token string, now time.Time) (Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep(now)
	fp := fingerprint(token)
	p, ok := s.tokens[fp]
	if !ok {
		return Grant{}, ErrTokenInvalid
	}
	if !s.begin(p.grant.Xuid, now) {
		return Grant{}, ErrRateLimited
	}
	delete(s.tokens, fp)
	slog.Info("transfer: jeton d'export consommé", "xuid", p.grant.Xuid)
	return p.grant, nil
}

// AllowSession ouvre un export demandé par la session connectée (sans jeton) : true si le joueur n'a pas
// d'export réussi dans l'heure ni d'export en cours ; l'appelant le termine par [TokenStore.Finish].
func (s *TokenStore) AllowSession(xuid string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.begin(xuid, now) {
		return false
	}
	slog.Info("transfer: export de la session", "xuid", xuid)
	return true
}

// Finish termine l'export du joueur ouvert par Redeem ou AllowSession. ok : l'export a abouti, la limite d'un
// par heure part de now ; sinon le joueur peut redemander tout de suite.
func (s *TokenStore) Finish(xuid string, ok bool, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inFlight, xuid)
	if ok {
		s.lastRun[xuid] = now
	}
}

// begin marque l'export du joueur en cours, ou refuse (verrou tenu) : un export réussi de moins d'une heure ou un
// export déjà en cours.
func (s *TokenStore) begin(xuid string, now time.Time) bool {
	if last, ran := s.lastRun[xuid]; ran && now.Sub(last) < ExportInterval {
		slog.Warn("transfer: export refusé, limite d'un par heure", "xuid", xuid, "dernier", last)
		return false
	}
	if s.inFlight[xuid] {
		slog.Warn("transfer: export refusé, un export du joueur est en cours", "xuid", xuid)
		return false
	}
	s.inFlight[xuid] = true
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
