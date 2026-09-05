// Package persist — bomb_stats_sentinels_test.go : le garde-fou de la RECOPIE.
//
// `bomb_stats_persister.go` recopie quatre chaines qui appartiennent a `analysis` : les deux
// types de fait date (`replay.BombEventArmed` / `replay.BombEventDetonated`) et les deux valeurs
// de vocabulaire de `match_objective_events` (`objectiveevents.ObjectiveTypeBomb` /
// `objectiveevents.RoleScorer`). La recopie est volontaire — faire dependre `persist` du
// decodeur de film pour quatre chaines serait un couplage disproportionne — mais une recopie
// sans garde-rail RE-DIVERGE (regle n6 du depot). Ce test EST le garde-rail : il echoue le jour
// ou l'une des quatre change de valeur la-bas sans changer ici, et une divergence silencieuse
// ecrirait des faits que plus aucun lecteur ne reconnaitrait.
//
// Sans tag de build : test pur, aucune base, il tourne partout.

package persist

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
)

func TestVocabulaireBombeAligne(t *testing.T) {
	cas := []struct {
		nom     string
		ici     string
		la      string
		sourceN string
	}{
		{"event_type arme", BombEventArmed, replay.BombEventArmed, "replay.BombEventArmed"},
		{"event_type explose", BombEventDetonated, replay.BombEventDetonated, "replay.BombEventDetonated"},
		{"objective_type", bombObjectiveType, objectiveevents.ObjectiveTypeBomb, "objectiveevents.ObjectiveTypeBomb"},
		{"role de l acteur", bombEventRoleScorer, objectiveevents.RoleScorer, "objectiveevents.RoleScorer"},
	}
	for _, c := range cas {
		if c.ici != c.la {
			t.Errorf("%s : persist ecrit %q, %s vaut %q — les deux ont diverge, et les faits "+
				"ecrits ne seraient plus reconnus par leurs lecteurs. Aligner ICI, dans le meme "+
				"commit qui change la valeur la-bas.", c.nom, c.ici, c.sourceN, c.la)
		}
	}
}
