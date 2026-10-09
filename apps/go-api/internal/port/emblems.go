package port

import "context"

// EmblemURLLoader — l'URL de l'emblème Spartan de chaque gamertag (fiche « Ma part à l'objectif »
// des Séries temporelles). Implémenté par internal/platform/duckdb.SquadV2LoaderAdapter, le même
// chargeur que les fiches de médailles de l'Escouade. Dégradation silencieuse par contrat : un
// joueur sans emblème connu reçoit une entrée vide, jamais une erreur.
type EmblemURLLoader interface {
	LoadEmblemURLs(ctx context.Context, titleSlug string, gamertags []string) map[string]string
}
