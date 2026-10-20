// Package auth — attempt_claim.go : remise d'une tentative autorisée à la session
// qui l'a lancée, une seule fois.
package auth

import "time"

// ClaimAuthorized remet à la session sessionID la tentative autorisée attemptID, une
// seule fois : sous verrou, la tentative doit appartenir à cette session et être au
// statut authorized ; elle passe alors au statut provisioned et ses secrets (jetons
// Halo, Microsoft, XSTS) sont effacés du registre. La copie rendue les porte encore,
// pour la stratégie post-flux. Faux si la tentative est absente, d'une autre session
// ou déjà remise : deux sondages concurrents ne connectent pas deux fois.
func (s *AttemptStore) ClaimAuthorized(attemptID, sessionID string) (*Attempt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[attemptID]
	if !ok || a.SessionID != sessionID || a.Status != AttemptStatusAuthorized {
		return nil, false
	}
	claimed := *a
	a.Status = AttemptStatusProvisioned
	clearAttemptSecrets(a)
	return &claimed, true
}

// Fail fait passer la tentative au statut failed avec le code donné et efface ses
// secrets. Sans effet si la tentative est absente.
func (s *AttemptStore) Fail(attemptID, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.attempts[attemptID]; ok {
		a.Status = AttemptStatusFailed
		a.ErrorCode = code
		clearAttemptSecrets(a)
	}
}

// Rebind rattache la tentative à la session newSessionID (identifiant renouvelé à la
// connexion) : le sondage suivant, qui porte le nouveau cookie, la retrouve. Sans
// effet si la tentative est absente.
func (s *AttemptStore) Rebind(attemptID, newSessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[attemptID]
	if !ok {
		return
	}
	if s.bySession[a.SessionID] == attemptID {
		delete(s.bySession, a.SessionID)
	}
	a.SessionID = newSessionID
	s.bySession[newSessionID] = attemptID
}

// clearAttemptSecrets efface les jetons portés par une tentative. L'appelant tient s.mu.
func clearAttemptSecrets(a *Attempt) {
	a.SpartanToken = ""
	a.ClearanceToken = ""
	a.SpartanExpiresAt = time.Time{}
	a.MicrosoftAccessToken = ""
	a.OAuthRefreshToken = ""
	a.XSTSRTAToken = ""
	a.XSTSRTAUserHash = ""
	a.XSTSRTAExpiresAt = time.Time{}
}
