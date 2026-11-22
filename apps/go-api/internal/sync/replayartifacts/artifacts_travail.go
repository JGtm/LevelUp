package replayartifacts

// artifacts_travail.go — LE LOT DU CYCLE : un match à construire et ce que la base en sait.
// Extrait d'artifacts.go pour tenir le seuil de taille de fichier ; mêmes déclarations, même paquet.

import (
	"context"
	"database/sql"
	"log/slog"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// buildWork : un match à construire, avec ses identités de carte candidates et les faits que
// seule la base connaît (lignes de match, scores des deux camps, nom de variante).
type buildWork struct {
	matchID  string
	mapNames []string
	facts    port.MatchFacts
	// priorite : rang dans la file de l'ouvrier (domain.BuildPriority*), posé par la sélection.
	priorite int
}

// attachMatchFacts lit, DANS LE MÊME SEGMENT DE LECTURE que la sélection, ce que la base sait
// de chaque match du lot — le pont d'identité des joueurs et celui des camps.
//
// POURQUOI ICI ET PAS AU MOMENT DE CONSTRUIRE : le décodage d'un film dure des secondes à des
// minutes, et rien ne doit tenir un handle partagé pendant ce temps-là (même règle que l'action
// admin). Deux SELECT courts par match, puis la base est relâchée.
//
// Une lecture qui échoue DÉGRADE ce match seul, journalisée : l'artefact sortira sans compteurs
// de joueur ni actions d'objectif, ce qui vaut mieux que pas d'artefact du tout.
func attachMatchFacts(ctx context.Context, sharedDB *sql.DB, work []buildWork) {
	if sharedDB == nil {
		return
	}
	var repo port.ReplayFactsRepo = duckdbpkg.NewReplayFactsRepo(sharedDB)
	for i := range work {
		facts, err := repo.FactsForMatch(ctx, work[i].matchID)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: faits de match illisibles — rejeu sans courbe de score complète",
				"err", err, "match_id", work[i].matchID)
			continue
		}
		work[i].facts = facts
	}
}

// etatArtefact dit ce que vaut l'artefact déjà sur disque pour ce match : est-il à la version
// COURANTE, et porte-t-il ce que les faits permettent d'y mettre.
//
// UNE SEULE ÉCRITURE DE LA RÈGLE POUR LES DEUX CHEMINS post-sync (mise en file et construction
// locale). En deux exemplaires elle aurait divergé au premier ajustement, et un chemin serait
// resté à figer des artefacts appauvris pendant que l'autre les répare.
//
// `complet` est une PRÉSOMPTION, pas une preuve : sans lignes de match, il n'y a rien de mieux
// à espérer d'une reconstruction, donc l'artefact est réputé complet ; avec des lignes, l'absence
// de compteurs de joueur le fait présumer appauvri (cf. l'en-tête de
// `replaybuild.Digest.HasPlayerCounters` pour les trois vacuités légitimes que cette présomption
// peut confondre).
// UNE SEULE LECTURE DU FICHIER, ET C'EST L'ITEM 5.3 DE PLAN_CUISSON_PERF : les deux questions
// (« à jour ? », « avec des compteurs ? ») ouvraient chacune l'artefact — deux `os.ReadFile` et
// deux désérialisations d'un document de ~2 Mo, pour chaque match de chaque cycle, sur les DEUX
// chemins post-sync. Le digest répond aux deux d'une seule lecture.
func etatArtefact(path string, facts port.MatchFacts) (aJour, complet bool) {
	d, ok := replaybuild.ArtifactDigest(path)
	if !ok || !d.UpToDate() {
		return false, false
	}
	return true, len(facts.Players) == 0 || d.HasPlayerCounters()
}
