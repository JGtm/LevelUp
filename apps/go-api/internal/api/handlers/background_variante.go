package handlers

import (
	"context"
	"net/http"
)

// parametreVariante nomme le paramètre de requête qui choisit l'image servie par une route
// d'image de fond ; absent, la route sert le fond lui-même.
const parametreVariante = "variante"

// variantePlan est la valeur de `?variante=` qui demande l'image plan du fond : le plan des
// composantes du maillage de navigation, dans le cadre et au calage de l'image du fond.
const variantePlan = "plan"

// varianteFond lit `?variante=` d'une route d'image de fond et dit si l'image plan est demandée.
// Une valeur inconnue est refusée (400) : un client qui se trompe de nom doit le voir, pas
// recevoir le fond à la place du plan. Rend ok=false quand la réponse est déjà écrite.
func varianteFond(ctx context.Context, w http.ResponseWriter, r *http.Request) (plan, ok bool) {
	switch v := r.URL.Query().Get(parametreVariante); v {
	case "":
		return false, true
	case variantePlan:
		return true, true
	}
	writeError(ctx, w, http.StatusBadRequest, "invalid_request", "variante inconnue : attendu « plan » ou rien")
	return false, false
}

// imageDeFond sert l'image du fond ou, si `plan`, son image plan, par l'une ou l'autre des deux
// méthodes du service : `fond` pour l'image du fond, `planImage` pour l'image plan.
func imageDeFond(plan bool, fond, planImage func() ([]byte, string, error)) ([]byte, string, error) {
	if plan {
		return planImage()
	}
	return fond()
}
