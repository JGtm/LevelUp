// Package historyfail classe l'échec d'une lecture de l'API Halo faite par le rattrapage de
// l'historique, pour que le compte de passes en échec (qui finit par abandonner un match ou une page)
// ne compte que les défauts du match ou de la page, jamais une panne de l'API ou du réseau.
//
// Sans lui, trois cycles de suite pendant une panne (10 min chacun) abandonnaient des matchs sains. Deux
// verdicts, tous deux « cet échec ne compte aucune tentative » :
//
//   - [IsTransient] : 429, délai dépassé, réseau coupé, pool sans jeton, arrêt du cycle. Ce n'est jamais le
//     défaut du match, quelle que soit la part des matchs touchés. Un 5xx n'en fait PAS partie : un match
//     que l'API refuse toujours en 5xx existe, et sans abandon il garderait le joueur au rattrapage à vie ;
//   - [Wide] : trop de matchs en échec dans la même passe pour que ce soient les matchs, c'est une panne
//     (5xx compris, mais aussi écriture refusée, décodage en série).
//
// Les verdicts de la brique halo-api (statut HTTP typé, Infinite et Halo 5) et les erreurs de la
// bibliothèque standard font foi ; aucun texte d'erreur n'est lu.
package historyfail

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/JGtm/halo-api/halo5"
	"github.com/JGtm/halo-api/infinite"

	"levelup/go-api/internal/platform/auth/pool"
)

const (
	// WidePercent : part, en pour cent, des matchs d'une passe au-delà de laquelle les échecs sont ceux
	// d'une panne et non de matchs précis.
	WidePercent = 10
	// WideMinFailed : nombre d'échecs en dessous duquel la part ne dit rien.
	WideMinFailed = 5
)

// IsTransient dit si l'erreur de lecture est une panne passagère de la liaison ou une limitation de
// débit, jamais le défaut du match. nil n'en est pas une : un match non rangé sans erreur compte.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if code, ok := httpStatus(err); ok {
		return code == http.StatusTooManyRequests || code == http.StatusRequestTimeout
	}
	var netErr net.Error
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, pool.ErrNoHealthySlot) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.As(err, &netErr)
}

// httpStatus lit le statut HTTP typé de la brique (Halo Infinite, puis Halo 5).
func httpStatus(err error) (int, bool) {
	var inf *infinite.HTTPError
	if errors.As(err, &inf) {
		return inf.StatusCode, true
	}
	var h5 *halo5.HTTPError
	if errors.As(err, &h5) {
		return h5.StatusCode, true
	}
	return 0, false
}

// Wide dit si les `failed` matchs en échec d'une passe sont ceux d'une panne : au moins WideMinFailed, et
// plus de WidePercent % des matchs de la passe. La passe se mesure à `listed` matchs listés, au moins `budget`
// (le budget de la passe) : une dernière passe de quelques matchs, où des matchs durablement refusés sont
// seuls à rester, ne passe pas pour une panne et finit par les abandonner.
func Wide(failed, listed, budget int) bool {
	return failed >= WideMinFailed && failed*100 > max(listed, budget)*WidePercent
}

// Counts dit si l'échec compte une tentative vers l'abandon : ni passager, ni noyé dans une passe de masse.
func Counts(wide bool, err error) bool {
	return !wide && !IsTransient(err)
}
