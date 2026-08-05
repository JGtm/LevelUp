// Package persist — weapon_shots_sentinels_test.go : le garde-fou de la RECOPIE.
//
// `weaponSentinelMax` recopie une connaissance qui appartient a `internal/analysis` (les
// identifiants 0/1/2 sont les sentinelles grenade/melee/vehicule). La recopie est volontaire —
// faire dependre `persist` d `analysis` pour trois entiers serait un couplage disproportionne —
// mais une recopie sans garde-rail RE-DIVERGE (regle n6 du depot). Ce test EST le garde-rail :
// il echoue le jour ou `analysis` ajoute, retire ou deplace une sentinelle.
//
// Sans tag de build : c est un test pur, aucune base, il tourne partout.

package persist

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func TestSentinellesArmesAlignees(t *testing.T) {
	for id := range analysis.SentinelIDs {
		if id > weaponSentinelMax {
			t.Errorf("analysis.SentinelIDs contient %d, au-dela de weaponSentinelMax=%d — "+
				"le persister laisserait passer cette sentinelle et fabriquerait une jointure "+
				"fausse avec metadata.weapon_labels. Relever la constante ICI, dans le meme "+
				"commit qui ajoute la sentinelle la-bas.", id, weaponSentinelMax)
		}
	}
	// Le sens inverse compte aussi : si `analysis` RETIRAIT une sentinelle, la constante
	// deviendrait trop haute et le persister refuserait des identifiants filmshell legitimes.
	if len(analysis.SentinelIDs) != weaponSentinelMax+1 {
		t.Errorf("analysis.SentinelIDs a %d entrees pour weaponSentinelMax=%d (attendu %d) — "+
			"les deux ont divergé, le persister refuse ou laisse passer les mauvais identifiants",
			len(analysis.SentinelIDs), weaponSentinelMax, weaponSentinelMax+1)
	}
}
