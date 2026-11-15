package port

// last_match.go — la fraîcheur des données d'un titre : le dernier match persisté de chaque joueur suivi.

import (
	"context"
	"time"
)

// LastMatchReader lit le dernier match persisté de chaque joueur d'un titre.
type LastMatchReader interface {
	// LastMatchByXUID rend, en UNE requête groupée par titre sur la base partagée située à sharedDBPath,
	// l'instant canonique (UTC) du dernier match de chaque xuid. Un joueur sans match est absent du
	// résultat ; aucune base de joueur n'est résolue. Une base absente ou illisible est une erreur.
	LastMatchByXUID(ctx context.Context, sharedDBPath, titleSlug string, xuids []string) (map[string]time.Time, error)
}
