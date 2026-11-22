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

// Les valeurs de `?variante=` qu'une route d'image de fond accepte, en plus de son absence.
const (
	// variantePlan demande l'image plan du fond : le plan des composantes du maillage de
	// navigation, dans le cadre et au calage de l'image du fond.
	variantePlan = "plan"
	// varianteRelief demande l'image du relief : hauteurs et drapeaux, dans le même cadre et au
	// même calage. Ses données vont par la route sœur `background.relief.json`.
	varianteRelief = "relief"
)

// cacheFondDeCarte est le Cache-Control de tout ce qu'une route de fond sert en octets (image du
// fond, plan, relief et ses données) : donnée de RÉFÉRENCE versionnée, qui ne change qu'à une
// re-cuisson, jamais en cours de session. `private` parce que les routes sont derrière l'ownership
// joueur.
const cacheFondDeCarte = "private, max-age=3600"

// typeDonneesRelief est le type MIME des données du relief : leur format est fixe.
const typeDonneesRelief = "application/json"

// varianteFond lit `?variante=` d'une route d'image de fond et rend la variante demandée : "" pour
// le fond lui-même, sinon l'une des constantes ci-dessus. Seule l'absence du paramètre sert le fond.
// Toute autre forme est refusée (400) : une valeur inconnue, une valeur vide, ou le paramètre écrit
// avec une autre casse (`?Variante=plan`). Un client qui se trompe doit le voir, pas recevoir le fond
// à la place de l'image demandée. Rend ok=false quand la réponse est déjà écrite.
func varianteFond(ctx context.Context, w http.ResponseWriter, r *http.Request) (variante string, ok bool) {
	q := r.URL.Query()
	if !q.Has(parametreVariante) && !casseFausse(q) {
		return "", true
	}
	if v := q.Get(parametreVariante); (v == variantePlan || v == varianteRelief) && !casseFausse(q) {
		return v, true
	}
	writeError(ctx, w, http.StatusBadRequest, "invalid_request",
		"variante inconnue : attendu « "+variantePlan+" », « "+varianteRelief+" » ou rien")
	return "", false
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

// imageDeFond sert l'image du fond ou, selon `variante`, son image plan ou son relief, par la
// méthode du service qui lui répond : `fond`, `plan` ou `relief`.
func imageDeFond(variante string, fond, plan, relief func() ([]byte, string, error)) ([]byte, string, error) {
	switch variante {
	case variantePlan:
		return plan()
	case varianteRelief:
		return relief()
	}
	return fond()
}
