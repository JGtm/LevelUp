package haloapi

import (
	"context"
	"errors"
	"net/http"
)

// Doer émet une requête HTTP. La brique n'en construit jamais : LevelUp la lui passe, et c'est
// elle qui porte la garde `netguard` (mode démo), le délai et le transport. `*http.Client`
// satisfait l'interface.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Garde : un [Doer] qui peut refuser une sortie AVANT qu'elle n'attende quoi que ce soit (limiteur,
// reprise) l'implémente. La brique la consulte une fois au début de chaque appel ; une erreur
// rendue arrête l'appel telle quelle (la marquer [NonRetryable], comme le refus de `Do`).
type Garde interface {
	Verifie(ctx context.Context) error
}

// ErrDefinitif marque une erreur qu'aucune nouvelle tentative ne corrigera (sortie réseau
// refusée par l'appelant, par exemple le mode démo). Une boucle de reprise de la brique s'arrête
// dessus sans attendre ; un appelant la reconnaît par `errors.Is(err, ErrDefinitif)`.
var ErrDefinitif = errors.New("haloapi: erreur définitive, ne pas retenter")

// NonRetryable enveloppe `err` pour qu'elle soit reconnue à la fois comme [ErrDefinitif] et comme
// elle-même (`errors.Is(x, err)` reste vrai). Le message est celui de `err`, inchangé. `nil` rend
// `nil`.
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return definitif{err: err}
}

// definitif : l'erreur enveloppée par [NonRetryable].
type definitif struct{ err error }

func (e definitif) Error() string { return e.err.Error() }

// Unwrap rend les deux erreurs que `errors.Is` / `errors.As` doivent voir.
func (e definitif) Unwrap() []error { return []error{ErrDefinitif, e.err} }
