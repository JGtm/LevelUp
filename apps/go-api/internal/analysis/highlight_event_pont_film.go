package analysis

import "levelup/go-api/internal/domain/highlightevent"

// PONT TRANSITOIRE VERS `film/` — A SUPPRIMER AU LOT 2.5.e.
//
// CE QUE C'EST. Cinq renvois, sans une ligne de logique, vers
// `internal/domain/highlightevent` : le type du temps fort et les quatre valeurs de son
// champ `EventType`, qui ont quitte ce paquet a l'item 2.5.h. Tout consommateur
// title-agnostic du depot est passe au nouveau nom ; ce pont ne sert QU'AUX fichiers de
// `internal/games/halo_infinite/film/`.
//
// POURQUOI IL EXISTE, ET POURQUOI IL N'EST PAS UN CHOIX. Deux contraintes du lot se
// croisent exactement sur ces fichiers :
//
//  1. `film/facts/killsource/feed.go` et `film/replay/deaths_source.go` lisent ces
//     symboles en PRODUCTION, et `film/facts/killsource` est l'une des QUATRE racines
//     hachees par `GrammarRev` (`film/grammar/grammar_rev_fingerprint_test.go`).
//     Re-pointer `feed.go` ferait monter l'empreinte de grammaire — or l'item 2.5.h ne
//     touche aucun paquet hache et doit laisser `GrammarRev` immobile.
//  2. Le lot 2.5.b requalifie EN PARALLELE tout `film/` (couche `profile`, ~200 sites) :
//     la frontiere de fichiers de 2.5.h s'arrete au bord de `film/`.
//
// Le pont est donc la forme la plus petite qui laisse compiler des fichiers qu'on ne
// peut ni toucher ni casser. Il ne retient rien : `analysis` n'utilise plus ces renvois
// nulle part, le lecteur et ses voisins citent `highlightevent` directement.
//
// QUAND IL DISPARAIT, ET COMMENT ON LE SAURA. Au lot 2.5.e, qui re-pointe les
// consommateurs de `film/` derriere la facade exportee : le jour ou plus aucun fichier
// de `film/` ne cite `analysis.HighlightEvent` ni `analysis.EventType*`, ce fichier n'a
// plus d'objet. Ce n'est pas une promesse : `TestPontTempsFortVersFilmNEstPasPerime`
// (meme paquet) LIT les sources de `film/` et ECHOUE le jour ou le compte tombe a zero,
// en demandant la suppression de ce fichier. Ouvert le 2026-09-16, item 2.5.h.
type HighlightEvent = highlightevent.HighlightEvent

// EventType* : voir le pont ci-dessus. Valeurs canoniques dans
// `internal/domain/highlightevent`.
const (
	EventTypeKill  = highlightevent.EventTypeKill
	EventTypeDeath = highlightevent.EventTypeDeath
	EventTypeMedal = highlightevent.EventTypeMedal
	EventTypeMode  = highlightevent.EventTypeMode
)
