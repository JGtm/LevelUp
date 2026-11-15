package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// parametreVariante nomme le paramètre de requête qui choisit l'image servie par une route
// d'image de fond ; absent, la route sert le fond lui-même.
const parametreVariante = "variante"

// variantePlan est la valeur de `?variante=` qui demande l'image plan du fond : le plan des
// composantes du maillage de navigation, dans le cadre et au calage de l'image du fond.
const variantePlan = "plan"

// varianteFond lit `?variante=` d'une route d'image de fond et dit si l'image plan est demandée.
// Seule l'absence du paramètre sert le fond. Toute autre forme est refusée (400) : une valeur
// inconnue, une valeur vide, ou le paramètre écrit avec une autre casse (`?Variante=plan`). Un
// client qui se trompe doit le voir, pas recevoir le fond à la place du plan. Rend ok=false
// quand la réponse est déjà écrite.
func varianteFond(ctx context.Context, w http.ResponseWriter, r *http.Request) (plan, ok bool) {
	q := r.URL.Query()
	if !q.Has(parametreVariante) && !casseFausse(q) {
		return false, true
	}
	if q.Get(parametreVariante) == variantePlan && !casseFausse(q) {
		return true, true
	}
	writeError(ctx, w, http.StatusBadRequest, "invalid_request", "variante inconnue : attendu « plan » ou rien")
	return false, false
}

// casseFausse dit si la requête porte le paramètre de variante écrit avec une autre casse.
func casseFausse(q url.Values) bool {
	for cle := range q {
		if cle != parametreVariante && strings.EqualFold(cle, parametreVariante) {
			return true
		}
	}
	return false
}

// imageDeFond sert l'image du fond ou, si `plan`, son image plan, par l'une ou l'autre des deux
// méthodes du service : `fond` pour l'image du fond, `planImage` pour l'image plan.
func imageDeFond(plan bool, fond, planImage func() ([]byte, string, error)) ([]byte, string, error) {
	if plan {
		return planImage()
	}
	return fond()
}
