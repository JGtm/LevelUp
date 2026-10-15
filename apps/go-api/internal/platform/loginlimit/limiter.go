// Package loginlimit borne les tentatives d'authentification par mot de passe.
//
// Deux compteurs indépendants, en mémoire du processus : par identifiant normalisé
// (casse et espaces de bord ignorés, comme le magasin d'utilisateurs) et par IP
// cliente. Au-delà de MaxFailures échecs dans une fenêtre glissante de Window,
// la clé est bloquée pendant Lockout ; une tentative bloquée n'est ni vérifiée ni
// comptée. Un succès remet à zéro les deux clés de la tentative.
//
// Le compteur par identifiant protège un compte attaqué depuis de nombreuses IP ;
// le compteur par IP freine une IP qui essaie de nombreux comptes. Les compteurs ne
// survivent pas à un redémarrage : la borne vise la force brute en ligne, pas
// l'audit.
package loginlimit

import (
	"strings"
	"sync"
	"time"
)

const (
	// MaxFailures est le nombre d'échecs tolérés dans la fenêtre : l'échec qui
	// l'atteint bloque la clé, la tentative suivante reçoit le refus.
	MaxFailures = 5
	// Window est la fenêtre glissante sur laquelle les échecs sont comptés.
	Window = 15 * time.Minute
	// Lockout est la durée du blocage, comptée depuis l'échec qui l'a déclenché.
	Lockout = 15 * time.Minute

	// maxIdentifierRunes borne la longueur d'un identifiant utilisé comme clé : un
	// identifiant plus long est tronqué (la mémoire d'une clé reste bornée quel que
	// soit le corps envoyé).
	maxIdentifierRunes = 64
	// maxKeys borne le nombre de clés suivies par compteur. Au-delà, les clés
	// expirées sont purgées, puis la clé non bloquée la moins récente est évincée
	// (une clé bloquée ne l'est qu'en dernier recours).
	maxKeys = 10_000
)

// Limiter compte les échecs d'authentification par identifiant et par IP. Sûr pour
// un usage concurrent.
type Limiter struct {
	mu           sync.Mutex
	now          func() time.Time
	byIdentifier map[string]*entry
	byIP         map[string]*entry
}

// entry est l'état d'une clé : échecs encore dans la fenêtre (du plus ancien au plus
// récent) et fin de blocage éventuelle.
type entry struct {
	failures     []time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

// New rend un limiteur vide sur l'horloge système.
func New() *Limiter {
	return newWithClock(time.Now)
}

func newWithClock(now func() time.Time) *Limiter {
	return &Limiter{
		now:          now,
		byIdentifier: make(map[string]*entry),
		byIP:         make(map[string]*entry),
	}
}

// Blocked dit si l'identifiant ou l'IP est bloqué, et pour combien de temps encore
// (la plus longue des deux attentes). Une clé vide n'est jamais bloquée.
func (l *Limiter) Blocked(identifier, ip string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	wait := max(remaining(l.byIdentifier, normalizeIdentifier(identifier), now),
		remaining(l.byIP, ip, now))
	return wait, wait > 0
}

// RecordFailure compte un échec pour l'identifiant et pour l'IP. Une clé vide est
// ignorée.
func (l *Limiter) RecordFailure(identifier, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	recordFailure(l.byIdentifier, normalizeIdentifier(identifier), now)
	recordFailure(l.byIP, ip, now)
}

// RecordSuccess remet à zéro l'identifiant et l'IP d'une authentification réussie.
func (l *Limiter) RecordSuccess(identifier, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byIdentifier, normalizeIdentifier(identifier))
	delete(l.byIP, ip)
}

// normalizeIdentifier ramène un identifiant à sa clé : sans espaces de bord, en
// minuscules (même règle que la clé du magasin d'utilisateurs), tronqué.
func normalizeIdentifier(identifier string) string {
	key := strings.ToLower(strings.TrimSpace(identifier))
	if r := []rune(key); len(r) > maxIdentifierRunes {
		key = string(r[:maxIdentifierRunes])
	}
	return key
}

// remaining rend le temps de blocage restant de la clé (0 si elle n'est pas
// bloquée) et retire l'entrée dont le blocage est échu.
func remaining(table map[string]*entry, key string, now time.Time) time.Duration {
	if key == "" {
		return 0
	}
	e, ok := table[key]
	if !ok || e.blockedUntil.IsZero() {
		return 0
	}
	if !now.Before(e.blockedUntil) {
		delete(table, key)
		return 0
	}
	return e.blockedUntil.Sub(now)
}

// recordFailure ajoute un échec à la clé et la bloque quand la fenêtre en compte
// MaxFailures.
func recordFailure(table map[string]*entry, key string, now time.Time) {
	if key == "" {
		return
	}
	e, ok := table[key]
	if !ok {
		makeRoom(table, now)
		e = &entry{}
		table[key] = e
	}
	e.lastSeen = now
	e.failures = append(pruneFailures(e.failures, now), now)
	if len(e.failures) >= MaxFailures {
		e.blockedUntil = now.Add(Lockout)
		e.failures = nil
	}
}

// pruneFailures retire les échecs sortis de la fenêtre.
func pruneFailures(failures []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-Window)
	i := 0
	for i < len(failures) && !failures[i].After(cutoff) {
		i++
	}
	return failures[i:]
}

// makeRoom libère une place dans une table pleine : purge des clés expirées, puis
// éviction de la clé non bloquée la moins récente, et en dernier recours de la clé
// bloquée dont le blocage finit le plus tôt.
func makeRoom(table map[string]*entry, now time.Time) {
	if len(table) < maxKeys {
		return
	}
	for key, e := range table {
		if !now.Before(e.blockedUntil) && len(pruneFailures(e.failures, now)) == 0 {
			delete(table, key)
		}
	}
	if len(table) < maxKeys {
		return
	}
	delete(table, evictionCandidate(table, now))
}

// evictionCandidate choisit la clé à évincer d'une table pleine sans clé expirée.
func evictionCandidate(table map[string]*entry, now time.Time) string {
	var idle, blocked string
	var idleSeen, blockedEnd time.Time
	for key, e := range table {
		if now.Before(e.blockedUntil) {
			if blocked == "" || e.blockedUntil.Before(blockedEnd) {
				blocked, blockedEnd = key, e.blockedUntil
			}
			continue
		}
		if idle == "" || e.lastSeen.Before(idleSeen) {
			idle, idleSeen = key, e.lastSeen
		}
	}
	if idle != "" {
		return idle
	}
	return blocked
}
