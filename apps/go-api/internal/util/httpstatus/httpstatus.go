// Package httpstatus lit le statut HTTP qu'une erreur d'appel à un service distant cite dans son
// message. Les clients de l'API Halo rendent des erreurs dont le texte porte « HTTP 500 » ou
// « statut HTTP 429 » ; les couches qui classent un échec (niveau de journal, synthèse d'une passe)
// n'importent pas ces clients, elles lisent le statut ici.
package httpstatus

import (
	"net/http"
	"regexp"
	"strconv"
)

var motif = regexp.MustCompile(`\bHTTP (\d{3})\b`)

// FromError rend le statut HTTP cité par le message de err, 0 si err est nil ou n'en cite aucun.
func FromError(err error) int {
	if err == nil {
		return 0
	}
	m := motif.FindStringSubmatch(err.Error())
	if m == nil {
		return 0
	}
	code, convErr := strconv.Atoi(m[1])
	if convErr != nil {
		return 0
	}
	return code
}

// IsServerError dit si le statut est une panne du service distant (5xx).
func IsServerError(status int) bool { return status >= http.StatusInternalServerError && status <= 599 }

// IsTooManyRequests dit si err cite un 429 : le service distant demande de ralentir.
func IsTooManyRequests(err error) bool { return FromError(err) == http.StatusTooManyRequests }
