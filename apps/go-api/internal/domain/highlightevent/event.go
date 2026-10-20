// Package highlightevent porte L'EVENEMENT DE TEMPS FORT tel qu'il entre en base :
// la ligne de `highlight_events`, avec le vocabulaire de son champ `EventType`.
//
// POURQUOI CE PAQUET EXISTE (item 2.5.h du PLAN_DECODEUR_FILM_2026-09-13, decision
// V19 (2), prerequis de V15 (4) mesure au §4 D4 ; ADR 0034). Le type vivait dans
// `internal/analysis`, a cote du lecteur binaire qui le produit
// (`grammar.ParseHighlightEvents`). Or ce lecteur est de la GRAMMAIRE DE FILM : il
// doit descendre sous `internal/games/halo_infinite/film/internal/grammar`. Le faire descendre
// sans sortir d'abord le TYPE laisserait le choix entre deux fautes — `internal/analysis`
// importerait un paquet de titre (garde-rail D9), ou le decodeur importerait
// `internal/analysis` (regle R2 du ratchet des couches). Le type remonte donc D'ABORD,
// et le lecteur descendra ensuite sans toucher un seul de ses consommateurs : c'est le
// chemin exact du lot 2.5.f pour `domain/equipmentusage`.
//
// POURQUOI SOUS `domain/` ET NON SOUS `games/canonical/`. Deux raisons, et la premiere
// se lit dans le depot : `canonical.HighlightEvent` EXISTE DEJA et designe autre chose —
// la forme INTER-TITRES relue depuis la base (`MatchID`, `KillerXUID`, `VictimXUID`,
// `PlayerXUID`, `TimeMS` en int64), celle que consomment `analysis/temporal`,
// `analysis/narrative` et `analysis/timeline`. Le type porte ici est la forme
// d'INGESTION : ce qu'une source rend avant l'ecriture, avec son `TypeHint` et son
// `MedalType` bruts. Les confondre sous un meme nom canonique melangerait la lingua
// franca et le tuyau d'entree. Seconde raison : ce type n'est PAS propre au film. Deux
// producteurs le remplissent — le lecteur du chunk de temps forts
// (`grammar.ParseHighlightEvents`) et l'import OpenSpartan
// (`service.toAnalysisEvent`, depuis `mapper.HighlightEventRow`) — et un seul
// consommateur l'ecrit (`sync.InsertHighlightEvents`). Un type que deux sources
// remplissent et qu'une couche de persistance lit est un type de `domain/`, feuille,
// sans aucun import du depot.
//
// CE PAQUET NE DECODE RIEN. `TypeHint`, `medalSortingWeights` et la deduction
// `inferEventType` restent du cote du lecteur : ce sont des regles de grammaire, pas des
// donnees. Ce paquet ne porte que la forme et le vocabulaire.
package highlightevent

// EventType* sont les valeurs possibles du champ HighlightEvent.EventType.
const (
	EventTypeKill  = "kill"
	EventTypeDeath = "death"
	EventTypeMedal = "medal"
	EventTypeMode  = "mode"
)

// HighlightEvent représente un événement parsé depuis le chunk highlight events.
type HighlightEvent struct {
	XUID      uint64
	Gamertag  string
	EventType string // EventTypeKill | EventTypeDeath | EventTypeMedal | EventTypeMode
	TypeHint  int
	IsMedal   bool
	TimeMS    int
	MedalType int
}
