package infinite

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrAuth : sentinelle d'un refus d'authentification des services Halo (HTTP 401/403). Un
// [*HTTPError] de statut 401 ou 403 est reconnu par `errors.Is(err, ErrAuth)` ; un
// [*BlobHTTPError] ne l'est JAMAIS (le CDN des films est public : son 401/403 ne dit rien des
// jetons).
var ErrAuth = errors.New("infinite: refus d'authentification (401/403)")

// ErrFilmGone : sentinelle d'une ressource ABSENTE côté Halo (HTTP 404/410) : film (manifeste ou
// blob), skill, CSR. Un [*HTTPError] ou un [*BlobHTTPError] de statut 404 ou 410 est reconnu par
// `errors.Is(err, ErrFilmGone)`.
var ErrFilmGone = errors.New("infinite: ressource absente (404/410)")

// Les causes posées dans [HTTPError.Err] (textes repris du client de synchronisation).
var (
	errJetonsRefuses    = errors.New("tokens invalides/expirés")
	errRessourceAbsente = errors.New("ressource absente")
)

// HTTPError : réponse non-200 d'un service Halo authentifié, typée pour inspection.
type HTTPError struct {
	StatusCode int
	URL        string
	Err        error
	// RetryAfter : fenêtre demandée par l'en-tête Retry-After (429/503), déjà parsée et bornée par
	// [maxRetryAfter]. 0 = en-tête absent ou illisible.
	RetryAfter time.Duration
	// Body : corps de la réponse en erreur, tel que reçu (diagnostic ; jamais journalisé par la
	// brique).
	Body []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.URL)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// Is reconnaît [ErrAuth] (401/403) et [ErrFilmGone] (404/410).
func (e *HTTPError) Is(target error) bool {
	switch target {
	case ErrAuth:
		return estStatutAuth(e.StatusCode)
	case ErrFilmGone:
		return estStatutAbsent(e.StatusCode)
	}
	return false
}

// BlobHTTPError : échec HTTP DÉFINITIF d'un téléchargement de blob sur le CDN des films.
//
// Type DISTINCT de [HTTPError], jamais un [HTTPError] enveloppé : le CDN est public et non
// authentifié. Un 401/403/503 venant de lui ne dit rien de la santé des jetons ni de l'API Halo ;
// un appelant qui marque un jeton malade sur un [HTTPError] 401/403 ne doit pas le faire sur une
// panne d'edge du CDN.
type BlobHTTPError struct {
	StatusCode int
	URL        string
	// Attempts : nombre de requêtes réellement envoyées (un statut non retentable rencontré au 3e
	// essai donne 3 ; le nombre d'essais de la politique quand ils sont épuisés).
	Attempts int
}

func (e *BlobHTTPError) Error() string {
	return fmt.Sprintf("downloadBlob HTTP %d après %d tentative(s): %s", e.StatusCode, e.Attempts, e.URL)
}

// Is reconnaît [ErrFilmGone] (404/410) ; jamais [ErrAuth].
func (e *BlobHTTPError) Is(target error) bool {
	return target == ErrFilmGone && estStatutAbsent(e.StatusCode)
}

// ErrChunkIncomplet : le CDN a servi un blob de film d'une autre taille que celle que le manifeste
// annonce (`ChunkSize`, taille du blob BRUT). Ce n'est pas un film perdu ([IsFilmGoneErr] le laisse
// passer) : l'appelant le retente à son cycle suivant.
type ErrChunkIncomplet struct {
	URL     string
	Annonce int // `ChunkSize` du manifeste (octets du blob brut)
	Recu    int // octets du blob brut reçu
}

func (e *ErrChunkIncomplet) Error() string {
	return fmt.Sprintf("blob de film incomplet (%d octets reçus, %d annoncés) : %s", e.Recu, e.Annonce, e.URL)
}

// IsAuthError dit si `err` est un refus d'authentification d'un service Halo (401/403). Un blob du
// CDN n'en est jamais un.
func IsAuthError(err error) bool { return errors.Is(err, ErrAuth) }

// IsNotFound dit si `err` signale une ressource absente côté Halo (404/410), service ou blob.
func IsNotFound(err error) bool { return errors.Is(err, ErrFilmGone) }

// IsFilmGoneErr : le même verdict que [IsNotFound], sous le nom que lisent les appelants qui
// RETENTENT un film (manifeste ou blob 404/410 = film définitivement perdu ; une expiration
// partielle, un blob mort à côté de blobs lisibles, est classée définitive elle aussi).
func IsFilmGoneErr(err error) bool { return IsNotFound(err) }

func estStatutAuth(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

func estStatutAbsent(code int) bool {
	return code == http.StatusNotFound || code == http.StatusGone
}

// maxRetryAfter borne la valeur de l'en-tête Retry-After : une valeur aberrante ne fige pas
// l'appelant indéfiniment.
const maxRetryAfter = 5 * time.Minute

// ParseRetryAfter lit l'en-tête Retry-After (RFC 7231) : un entier de secondes ou une date HTTP.
// Rend une durée dans [0, maxRetryAfter] ; 0 si absent ou illisible.
func ParseRetryAfter(h string, now time.Time) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil {
		if secs <= 0 {
			return 0
		}
		if d := time.Duration(secs) * time.Second; d <= maxRetryAfter {
			return d
		}
		return maxRetryAfter
	}
	if t, err := http.ParseTime(h); err == nil {
		d := t.Sub(now)
		if d <= 0 {
			return 0
		}
		return min(d, maxRetryAfter)
	}
	return 0
}
