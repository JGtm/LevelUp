package haloapitest

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Enregistreur est un http.RoundTripper qui note la ligne de chaque requête reçue
// ([LigneDeRequete]) et rend la réponse de [Enregistreur.Repondre]. Sûr pour un usage concurrent.
type Enregistreur struct {
	// Repondre rend la réponse à la requête numéro n (à partir de 1). Nil : 200 au corps vide.
	Repondre func(n int, req *http.Request) *http.Response
	// SurAppel, s'il est non nil, est appelé avec le numéro n une fois la requête notée et avant
	// que la réponse ne soit rendue.
	SurAppel func(n int)

	mu       sync.Mutex
	requetes []string
}

// RoundTrip note la requête, appelle SurAppel puis rend la réponse, dont le champ Request est posé.
func (e *Enregistreur) RoundTrip(req *http.Request) (*http.Response, error) {
	e.mu.Lock()
	e.requetes = append(e.requetes, LigneDeRequete(req))
	n := len(e.requetes)
	e.mu.Unlock()
	if e.SurAppel != nil {
		e.SurAppel(n)
	}
	rep := Reponse(http.StatusOK, "")
	if e.Repondre != nil {
		rep = e.Repondre(n, req)
	}
	rep.Request = req
	return rep, nil
}

// Requetes rend une copie des lignes notées, dans l'ordre d'arrivée.
func (e *Enregistreur) Requetes() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.requetes...)
}

// LigneDeRequete écrit « METHODE URL [Nom=valeur; ...] » : l'URL absolue, puis les en-têtes tels que
// le client les a posés, triés par nom canonique (plusieurs valeurs séparées par une virgule).
func LigneDeRequete(req *http.Request) string {
	noms := make([]string, 0, len(req.Header))
	for nom := range req.Header {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	paires := make([]string, 0, len(noms))
	for _, nom := range noms {
		paires = append(paires, nom+"="+strings.Join(req.Header[nom], ","))
	}
	return req.Method + " " + req.URL.String() + " [" + strings.Join(paires, "; ") + "]"
}

// Reponse construit une réponse de statut code et de corps corps ; entetes est une suite de paires
// nom, valeur (une valeur orpheline est ignorée).
func Reponse(code int, corps string, entetes ...string) *http.Response {
	rep := &http.Response{
		StatusCode: code,
		Status:     fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(corps)),
	}
	for i := 0; i+1 < len(entetes); i += 2 {
		rep.Header.Set(entetes[i], entetes[i+1])
	}
	return rep
}
