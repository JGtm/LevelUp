// Package auth — attempt_browser.go : tentatives de connexion par le navigateur (code
// d'autorisation + PKCE), dans le registre des tentatives du code d'appareil.
//
// POURQUOI LE MÊME REGISTRE. La page de la coquille lance la connexion, le navigateur par défaut
// la termine : le retour de Microsoft arrive sans cookie ni session. La tentative est donc liée à
// la session de la coquille dès le départ, retrouvée au retour par son `state` (secret
// aléatoire, usage unique, durée bornée), puis remise UNE fois à cette session par le sondage
// que le code d'appareil utilise déjà (ClaimAuthorized) : même point d'arrivée, même stratégie
// post-flux, aucun second chemin d'écriture des jetons.
package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// BrowserStateTTL borne l'attente du retour de Microsoft : au-delà, le `state` est refusé et la
// tentative expire (la page propose de recommencer).
const BrowserStateTTL = 10 * time.Minute

// Issues du retour de Microsoft (ClaimBrowserState).
var (
	// ErrBrowserStateUnknown : le state est inconnu, ou déjà consommé (usage unique).
	ErrBrowserStateUnknown = errors.New("auth: browser login state unknown or already used")
	// ErrBrowserStateExpired : le state a dépassé BrowserStateTTL.
	ErrBrowserStateExpired = errors.New("auth: browser login state expired")
)

// StartBrowser ouvre une tentative de connexion par le navigateur pour la session sessionID.
// Une tentative de navigateur encore en attente pour cette session est remplacée (un second
// clic repart d'un state neuf, l'ancien ne vaut plus rien). Refus ErrTooManyAttempts aux
// mêmes plafonds que le code d'appareil. locale : langue de la page appelante.
func (s *AttemptStore) StartBrowser(sessionID, ip, state, verifier, locale string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()

	if id, ok := s.bySession[sessionID]; ok {
		if prev := s.attempts[id]; prev != nil && prev.Browser && prev.Status == AttemptStatusPending {
			s.retireBrowserLocked(prev, AttemptStatusExpired)
		}
	}
	if s.pendingLocked(nil) >= MaxPendingAttempts || s.pendingLocked(&ip) >= MaxPendingPerIP {
		return nil, ErrTooManyAttempts
	}
	a := &Attempt{
		AttemptID:    uuid.New().String(),
		SessionID:    sessionID,
		ClientIP:     ip,
		StartedAt:    time.Now(),
		Status:       AttemptStatusPending,
		Browser:      true,
		BrowserState: state,
		CodeVerifier: verifier,
		Locale:       locale,
	}
	s.attempts[a.AttemptID] = a
	s.bySession[sessionID] = a.AttemptID
	s.byState[state] = a.AttemptID
	return a, nil
}

// ClaimBrowserState consomme le state rendu par Microsoft : sous verrou, la tentative doit être
// en attente et dans sa durée de vie. Rend une copie qui porte encore le code_verifier (la seule
// fois) ; le state et le verifier disparaissent du registre. ErrBrowserStateUnknown pour un state
// inconnu ou déjà servi, ErrBrowserStateExpired (tentative passée à expired) au-delà de BrowserStateTTL.
func (s *AttemptStore) ClaimBrowserState(state string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byState[state]
	if !ok || state == "" {
		return nil, ErrBrowserStateUnknown
	}
	a := s.attempts[id]
	if a == nil || !a.Browser || a.Status != AttemptStatusPending {
		delete(s.byState, state)
		return nil, ErrBrowserStateUnknown
	}
	if time.Since(a.StartedAt) > BrowserStateTTL {
		s.retireBrowserLocked(a, AttemptStatusExpired)
		return nil, ErrBrowserStateExpired
	}
	claimed := *a
	delete(s.byState, state)
	a.BrowserState = ""
	a.CodeVerifier = ""
	return &claimed, nil
}

// ExpireStaleBrowser fait passer à expired la tentative de navigateur en attente dont le state a
// dépassé BrowserStateTTL, pour que le sondage de la page le sache sans attendre un retour qui ne
// viendra pas. Sans effet pour une autre tentative ou une tentative absente.
func (s *AttemptStore) ExpireStaleBrowser(attemptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.attempts[attemptID]
	if a != nil && a.Browser && a.Status == AttemptStatusPending && time.Since(a.StartedAt) > BrowserStateTTL {
		s.retireBrowserLocked(a, AttemptStatusExpired)
	}
}

// retireBrowserLocked clôt une tentative de navigateur au statut donné et efface sa poignée de
// main. L'appelant tient s.mu.
func (s *AttemptStore) retireBrowserLocked(a *Attempt, status string) {
	if a.BrowserState != "" {
		delete(s.byState, a.BrowserState)
	}
	a.Status = status
	a.BrowserState = ""
	a.CodeVerifier = ""
}
