package xboxlive

import "fmt"

// HTTPError : réponse non-200 d'une page de succès, typée pour inspection (`errors.As`). Son message
// est `HTTP {statut}: {corps}` : le corps ENTIER de la réponse, tel que reçu (diagnostic de
// l'appelant ; jamais journalisé par la brique).
type HTTPError struct {
	StatusCode int
	URL        string
	Body       []byte
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body) }

// ErreurTransport : le [haloapi.Doer] a échoué à émettre la requête (réseau, délai, contexte). Son
// message est celui de l'erreur du Doer, inchangé ; `URL` permet à l'appelant de le composer. Une
// erreur [haloapi.ErrDefinitif] (sortie refusée par l'appelant) n'est jamais enveloppée.
type ErreurTransport struct {
	URL string
	Err error
}

func (e *ErreurTransport) Error() string { return e.Err.Error() }

func (e *ErreurTransport) Unwrap() error { return e.Err }
