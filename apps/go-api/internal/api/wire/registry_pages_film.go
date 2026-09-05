package wire

// registry_pages_film.go — LA PORTE DES DEUX PROJECTIONS DE L'ARTEFACT servies à la
// Match View : la timeline objectif v3 (`/objective-events`) et les positions joueurs
// aux images-clés (`/positions`).
//
// # CE QUE CE FICHIER CORRIGE
//
// Le câblage était INCONDITIONNEL (`registry_pages.go`, avant le 2026-09-05), sous un
// commentaire qui promettait pourtant l'inverse : « Titre sans film / tables absentes →
// le repo remonte ErrCapabilityNotSupported et l'endpoint rend un 503 propre ». Cette
// promesse ne pouvait pas être tenue :
//
//  1. le service ne dégrade que si le repo est `nil` (`GetObjectiveEvents` /
//     `GetMatchPositions`, service/match_view_service.go) — or il ne l'était jamais ;
//  2. les tables du film sont créées pour TOUS les titres (le registre de migrations
//     `shared` commun tourne sur chaque titre additionnel), donc jamais absentes ;
//  3. sur une table existante mais vide, `LoadMatch` rend `(nil, nil)` — pas une erreur.
//
// Résultat mesuré (registre `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`, constat D2) : sur
// Halo 5, les deux routes répondaient 200 `[]`. Le client ne pouvait pas distinguer « ce
// titre ne sait pas produire ce calque » de « ce match-là n'en a pas », et le compteur
// `http_capability_not_supported_total` restait à zéro.
//
// # LA RÈGLE
//
// Ces deux surfaces sont des PROJECTIONS DE L'ARTEFACT DE REJEU : sans artefact, elles
// n'ont pas de substrat. Leur porte est donc `film.replay_artifact`, la clé qui gouverne
// la production de cet artefact (décision utilisateur du 2026-09-05 : pas de cuisson sans
// clé, l'affichage suit). Titre sans la clé ⇒ repos NON câblés ⇒ le service rend
// `games.ErrCapabilityNotSupported` ⇒ le handler rend un 503 `capability_not_supported`.
// La doc de `registry_pages.go` devient vraie, et par la seule voie qui pouvait la rendre
// vraie.
//
// Jamais de `slug ==` : la clé est lue dans la CapabilityMap du titre du joueur.

import (
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// withFilmArtifactRepos câble les deux loaders du film si — et seulement si — la
// CapabilityMap du titre déclare `film.replay_artifact`. Sinon elle rend le service tel
// quel : ses deux getters trouvent leur repo à nil et rendent
// `games.ErrCapabilityNotSupported`, que le handler traduit en 503.
//
// La CapabilityMap et les deux repos sont PASSÉS, jamais résolus ici : c'est ce qui permet
// au garde-rail de rejouer la décision sur les capabilities.toml LIVRÉS, avec des loaders
// factices et sans la moindre base de données (registry_pages_film_test.go). Les deux
// constructeurs `duckdb.New*Repo` du seul appelant sont des enveloppes d'un pointeur — les
// évaluer quand la porte est fermée ne coûte rien et garde l'appel lisible.
func withFilmArtifactRepos(
	svc *service.MatchViewService,
	caps games.CapabilityMap,
	objectiveEvents port.ObjectiveEventsRepository,
	playerPositions port.PlayerPositionsRepository,
) *service.MatchViewService {
	if !caps.Has(games.CapFilmReplayArtifact) {
		return svc
	}
	return svc.
		WithObjectiveEventsRepo(objectiveEvents).
		WithPlayerPositionsRepo(playerPositions)
}

// filmArtifactReposFor est le point d'appel du wire : il résout la CapabilityMap du titre du
// joueur (jamais un slug) et construit les loaders réels.
func (r *ServiceRegistry) filmArtifactReposFor(
	svc *service.MatchViewService, pdb *duckdb.PlayerDB,
) *service.MatchViewService {
	return withFilmArtifactRepos(svc, r.capabilitiesForPDB(pdb),
		duckdb.NewObjectiveEventsRepo(pdb), duckdb.NewPlayerPositionsRepo(pdb))
}
