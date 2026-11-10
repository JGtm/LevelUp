package halo5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/JGtm/halo-api"
)

// En-têtes et valeurs du protocole Halo 5.
const (
	enteteAccept  = "Accept"
	enteteSpartan = "X-343-Authorization-Spartan"
	enteteUA      = "User-Agent"
	userAgent     = "cpprestsdk/2.4.0"
	typeJSON      = "application/json"
	typeImage     = "image/png,image/*"
	// Préfixes des messages d'erreur du transport (noms des deux chemins d'avant la brique).
	nomJSON    = "doGet"
	nomBinaire = "doGetBinary"
)

// Auth : le jeton d'une requête Halo 5, passé à CHAQUE appel. L'en-tête
// `X-343-Authorization-Spartan` est posé même vide ; Halo 5 n'emploie pas de jeton de clearance.
type Auth struct {
	Spartan string
}

// String masque le jeton (un `%v` ne le révèle jamais).
func (a Auth) String() string { return "Auth{Spartan:" + haloapi.Masque(a.Spartan) + "}" }

// LogValue masque le jeton dans un journal `slog`.
func (a Auth) LogValue() slog.Value {
	return slog.GroupValue(slog.String("spartan", haloapi.Masque(a.Spartan)))
}

// Client : le client des services Halo 5. Valeur immuable ; les `With*` rendent une copie, sûre en
// concurrence tant que les [haloapi.Doer] et le [haloapi.Limiter] reçus le sont.
type Client struct {
	doerJSON    haloapi.Doer
	doerBinaire haloapi.Doer
	endpoints   Endpoints
	retry       haloapi.Retry
	limiter     haloapi.Limiter
}

// New construit un client sur les deux `Doer` reçus (obligatoires : la brique n'en construit
// aucun) : `doerJSON` pour les réponses JSON, `doerBinaire` pour les rendus PNG. Politique de
// reprise `retry`, adresses par défaut ([EndpointsParDefaut]), aucun limiteur.
func New(doerJSON, doerBinaire haloapi.Doer, retry haloapi.Retry) *Client {
	return &Client{doerJSON: doerJSON, doerBinaire: doerBinaire, endpoints: EndpointsParDefaut(), retry: retry}
}

// WithEndpoints rend une copie du client sur les adresses `e`.
func (c *Client) WithEndpoints(e Endpoints) *Client {
	clone := *c
	clone.endpoints = e
	return &clone
}

// WithLimiter rend une copie du client sur le limiteur `l` (nil : aucun limiteur).
func (c *Client) WithLimiter(l haloapi.Limiter) *Client {
	clone := *c
	clone.limiter = l
	return &clone
}

// Endpoints rend les adresses du client.
func (c *Client) Endpoints() Endpoints { return c.endpoints }

// Limiter rend le limiteur du client (nil : aucun).
func (c *Client) Limiter() haloapi.Limiter { return c.limiter }

// requete : un GET de service, JSON ou binaire.
type requete struct {
	url     string // sans `auth=st`
	auth    Auth
	binaire bool
}

// essai : l'issue d'UNE requête.
type essai struct {
	corps       []byte
	contentType string
	err         error
	retryAfter  time.Duration
	// definitif : ne pas retenter (requête impossible à construire, erreur [haloapi.ErrDefinitif],
	// 401/403/404/410, 200 au corps vide d'un rendu, statut que la politique ne retente pas).
	definitif bool
}

// reponse : ce qu'UNE réponse reçue livre au classement.
type reponse struct {
	statut      int
	corps       []byte
	errLecture  error
	retryAfter  time.Duration
	contentType string
}

// getJSON exécute un GET JSON et rend le corps.
func (c *Client) getJSON(ctx context.Context, auth Auth, url string) ([]byte, error) {
	corps, _, err := c.faire(ctx, requete{url: url, auth: auth})
	return corps, err
}

// getBinaire exécute un GET binaire et rend le corps et son Content-Type.
func (c *Client) getBinaire(ctx context.Context, auth Auth, url string) ([]byte, string, error) {
	return c.faire(ctx, requete{url: url, auth: auth, binaire: true})
}

// faire exécute un GET sous la politique de reprise du client : garde du `Doer`, limiteur, essais,
// attentes. Une seule boucle pour les deux chemins.
func (c *Client) faire(ctx context.Context, rq requete) ([]byte, string, error) {
	doer := c.doerDe(rq)
	if err := verifierGarde(ctx, doer); err != nil {
		return nil, "", err
	}
	url := avecAuthST(rq.url)
	r := c.retry
	if r.LimiteurUneFois {
		if err := c.attendreLimiteur(ctx); err != nil {
			return nil, "", err
		}
	}
	var derniere error
	for n := 0; n < r.Essais; n++ {
		if !r.LimiteurUneFois {
			if err := c.attendreLimiteur(ctx); err != nil {
				return nil, "", err
			}
		}
		e := c.essayer(ctx, doer, rq, url)
		if e.err == nil {
			return e.corps, e.contentType, nil
		}
		if e.definitif {
			return nil, "", e.err
		}
		derniere = e.err
		if !r.Attendre(ctx, n, e.retryAfter) {
			return nil, "", ctx.Err()
		}
	}
	return nil, "", fmt.Errorf("%s %s: %d tentatives echouees: %w", nomDe(rq), url, r.Essais, derniere)
}

// doerDe rend le `Doer` du chemin de la requête.
func (c *Client) doerDe(rq requete) haloapi.Doer {
	if rq.binaire {
		return c.doerBinaire
	}
	return c.doerJSON
}

// nomDe rend le préfixe des messages d'erreur du chemin de la requête.
func nomDe(rq requete) string {
	if rq.binaire {
		return nomBinaire
	}
	return nomJSON
}

// verifierGarde consulte la garde du `Doer` ([haloapi.Garde]) s'il en porte une : une sortie
// refusée l'est avant toute attente du limiteur ou de reprise, et l'erreur de la garde est rendue
// telle quelle.
func verifierGarde(ctx context.Context, doer haloapi.Doer) error {
	if g, ok := doer.(haloapi.Garde); ok {
		return g.Verifie(ctx)
	}
	return nil
}

// attendreLimiteur attend un jeton du limiteur ; son erreur est rendue telle quelle, sauf si la
// politique l'ignore ([haloapi.Retry.IgnoreErreurLimiteur]).
func (c *Client) attendreLimiteur(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	if err := c.limiter.Wait(ctx); err != nil && !c.retry.IgnoreErreurLimiteur {
		return err
	}
	return nil
}

// essayer envoie UNE requête et classe son issue.
func (c *Client) essayer(ctx context.Context, doer haloapi.Doer, rq requete, url string) essai {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return essai{err: fmt.Errorf("%s new request: %w", nomDe(rq), err), definitif: true}
	}
	poserEntetes(req, rq)
	resp, err := doer.Do(req)
	if err != nil {
		return essai{err: err, definitif: errors.Is(err, haloapi.ErrDefinitif)}
	}
	corps, errLecture := io.ReadAll(resp.Body)
	_ = resp.Body.Close() // corps déjà lu ; une erreur de fermeture ne change pas l'issue
	return c.classer(rq, url, reponse{
		statut:      resp.StatusCode,
		corps:       corps,
		errLecture:  errLecture,
		retryAfter:  haloapi.ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
		contentType: resp.Header.Get("Content-Type"),
	})
}

// classer rend l'issue d'une réponse reçue. Le statut est lu AVANT l'erreur de lecture du corps.
func (c *Client) classer(rq requete, url string, rep reponse) essai {
	switch {
	case rep.statut == http.StatusOK:
		return classerSucces(rq, url, rep)
	case estStatutAuth(rep.statut):
		return essai{err: &HTTPError{StatusCode: rep.statut, URL: url, Err: errJetonInvalide}, definitif: true}
	case estStatutAbsent(rep.statut):
		return essai{err: &HTTPError{StatusCode: rep.statut, URL: url, Err: errRessourceAbsente}, definitif: true}
	}
	he := &HTTPError{StatusCode: rep.statut, URL: url, Err: fmt.Errorf("HTTP %d", rep.statut)}
	return essai{err: he, retryAfter: rep.retryAfter, definitif: !c.retry.RetenteStatut(rep.statut)}
}

// classerSucces : un 200 au corps coupé est retenté (sans fenêtre Retry-After) ; un 200 au corps
// vide est une erreur définitive du chemin binaire.
func classerSucces(rq requete, url string, rep reponse) essai {
	if rep.errLecture != nil {
		return essai{err: rep.errLecture}
	}
	if rq.binaire && len(rep.corps) == 0 {
		return essai{err: &HTTPError{StatusCode: rep.statut, URL: url, Err: errCorpsVide}, definitif: true}
	}
	return essai{corps: rep.corps, contentType: rep.contentType}
}

// poserEntetes pose Accept (JSON ou image), le jeton Spartan (même vide) et le User-Agent. Ni
// `343-clearance` ni `Accept-Encoding` (gzip transparent).
func poserEntetes(req *http.Request, rq requete) {
	accept := typeJSON
	if rq.binaire {
		accept = typeImage
	}
	req.Header.Set(enteteAccept, accept)
	req.Header.Set(enteteSpartan, rq.auth.Spartan)
	req.Header.Set(enteteUA, userAgent)
}

// avecAuthST ajoute `auth=st` à la requête de l'URL.
func avecAuthST(url string) string {
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	return url + sep + "auth=st"
}
