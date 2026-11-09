// Package highlightevent porte L'EVENEMENT DE TEMPS FORT tel que le decodeur de film le lit dans
// le chunk des temps forts (`grammar.ParseHighlightEvents`), avec le vocabulaire de son champ
// `EventType`.
//
// LA BRIQUE POSSEDE CE QU ELLE PRODUIT (plan D2, decision D2-B). LevelUp garde sa propre forme,
// `internal/domain/highlightevent`, parce qu elle a d autres producteurs que le film (import
// OpenSpartan, rattrapage des medailles) et que `internal/analysis` la lit. Les deux formes ont
// LES MEMES CHAMPS, dans le meme ordre, avec les memes types, et les memes valeurs de
// `EventType*` : la conversion vit dans l adaptateur de titre `internal/games/halo_infinite`
// (ADR 0012, `HighlightEventDepuisFilm`), dont le test d aller-retour par reflexion refuse toute
// divergence de champ.
//
// POURQUOI UN PAQUET ET NON `film/types`. Ce paquet est une FEUILLE hors des couches, comme
// `playerposition` et `objectiveevent`. `film/types` entre par ses octets dans l empreinte de la
// couche `source` (golden de perimetre de `source`) : y ajouter un fichier ferait bouger
// l empreinte d une couche que ce type ne concerne pas. Ici, seules `grammar` (producteur) et
// `facts/killsource` (lecteur) le hachent.
//
// CE PAQUET NE DECODE RIEN. `TypeHint`, les poids de tri des medailles et la deduction du type
// d evenement restent dans la grammaire : ce sont des regles de lecture, pas des donnees.
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
