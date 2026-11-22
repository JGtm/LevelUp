package port

import "context"

// HistoryCompletionReader dit, en lecture seule, si la fin de l'historique d'un joueur a déjà été
// vue (clé `history_complete_at` de sa base, règle de `sync/historystate`). Il sert à
// l'ordonnanceur quand la passe d'un joueur a échoué avant de pouvoir le dire : sans fin rangée,
// le cycle suivant doit venir vite.
//
// Contrat : jamais d'écriture ni de bail ; une base joueur absente vaut « incomplet » (aucune passe
// n'a encore parcouru l'historique) ; une lecture en échec rend l'erreur et l'appelant la journalise.
type HistoryCompletionReader interface {
	HistoryComplete(ctx context.Context, titleSlug, gamertag string) (bool, error)
}
