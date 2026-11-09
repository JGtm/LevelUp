package halo5

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPError : réponse non-200 d'un service Halo 5 (ou 200 au corps vide d'un rendu PNG), typée pour
// inspection (`errors.As`). URL est celle de la requête, `?auth=st` compris.
type HTTPError struct {
	StatusCode int
	URL        string
	Err        error
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.URL) }

func (e *HTTPError) Unwrap() error { return e.Err }

// Les causes posées dans [HTTPError.Err] (textes du client d'avant la brique).
var (
	errJetonInvalide    = errors.New("token invalide/expire")
	errRessourceAbsente = errors.New("ressource absente")
	errCorpsVide        = errors.New("corps vide")
)

func estStatutAuth(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

func estStatutAbsent(code int) bool {
	return code == http.StatusNotFound || code == http.StatusGone
}
