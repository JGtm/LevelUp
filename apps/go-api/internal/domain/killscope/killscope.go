// Package killscope — LA PORTÉE DES LIGNES DE `shared.match_kill_events`, en un seul endroit.
//
// Les colonnes `read_path` / `read_origin` disent CE SUR QUOI PORTE une ligne : par quelle voie
// elle a été lue, et de quelle mesure elle rend compte. Elles ne sont pas décoratives — c'est sur
// `read_path` que se décide la PRÉSÉANCE entre producteurs (le film gagne, toujours), donc deux
// copies qui divergent d'un caractère rendraient la préséance aveugle sans rien casser d'autre.
//
// ─── POURQUOI UN PAQUET, ET PAS UNE CONSTANTE DANS `persist` ───────────────────────────────
//
// Quatre écrivains portent ce vocabulaire : `persist` (producteur live), `migration` (reprise
// des couples), `sync/killcollector` (producteur crédit-seul hors ligne) et `ops` (base de
// démonstration). `migration` NE PEUT PAS importer `persist` : les tests internes de `persist`
// (`package persist`) importent `migration`, donc la dépendance inverse serait un cycle
// d'import à la compilation des tests. La conséquence a été une copie privée dans `migration`,
// avec un commentaire renvoyant à un test de verrouillage qui n'a jamais existé (constat J4R-3).
//
// Ce paquet est la réponse : une feuille SANS AUCUN IMPORT, donc importable par n'importe qui,
// sans cycle possible. Le garde-rail qui interdit les copies est
// `internal/archlint/no_raw_kill_scope_literal_test.go` ; les valeurs de fil sont épinglées par
// `killscope_test.go` (changer une valeur ici casse un test qui dit pourquoi).
package killscope

// Les voies de lecture (`read_path`) des producteurs CRÉDIT, et leur origine commune.
//
// Le crédit, c'est « qui le jeu a-t-il crédité de cette mort » — ce sur quoi les statistiques
// sont bâties. Deux producteurs le lisent, à deux endroits différents, et c'est exactement ce que
// la distinction `read_path` / `read_origin` encode : MÊME mesure, MÊME origine, voies distinctes.
const (
	// ReadPathLiveFeed : le kill-feed rendu par le pipeline de sync DU TITRE — kill-feed de
	// l'API pour Halo Infinite, carnage natif pour Halo 5. Voie du producteur live
	// (`persist.CreditKillEventsFromPairs`) ET de la reprise des couples historiques
	// (`migration.applyKillEventsFromPairs`) : les deux rendent compte des mêmes couples, au
	// même titre, l'un au fil de l'eau et l'autre en rattrapage.
	ReadPathLiveFeed = "kill-feed"

	// ReadPathCreditBackfill : les couples RECONSTITUÉS hors ligne depuis `highlight_events`
	// (`sync/killcollector.CreditCollector`). Voie distincte de la précédente parce que la
	// lecture l'est : ici les couples sont appariés par une fenêtre temporelle, ils ne sont pas
	// donnés tels quels par le titre.
	ReadPathCreditBackfill = "highlight-events"

	// OriginCreditOnly : « le crédit, et rien que le crédit ». Origine COMMUNE aux trois
	// écrivains ci-dessus : aucun d'eux ne mesure la source du dégât fatal (elle se lit dans le
	// film), aucun ne connaît l'assistant. Une ligne portant cette origine dit ce qu'elle sait
	// ET ce qu'elle ignore.
	OriginCreditOnly = "credit-seul"
)
