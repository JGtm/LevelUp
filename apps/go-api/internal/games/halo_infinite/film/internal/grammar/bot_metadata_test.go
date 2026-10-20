package grammar

// bot_metadata_test.go — LES FRAGMENTS DU BALAYAGE (bot_metadata.go, « les fragments du balayage »).
//
//	FR-SUFFIXE    `43 Hollis` (slot 0, bid 0), suffixe strict de `343 Hollis` que la marche lit :
//	              ecarte, compte.
//	FR-ECRIT      un nom que la marche lit lui-meme n est jamais ecarte, meme suffixe d un autre.
//	FR-AUTRE      un nom balaye qui n est le suffixe d aucun nom lu reste (il se compte ailleurs).
//
// Un paquet qui ne ferme pas n ecarte rien : `TestUneLargeurFausseNEcartePasLesFragments`
// (`facts/killsource/botmeta_equipe_test.go`), sur une charge fabriquee par la grammaire de l ecrivain.

import (
	"reflect"
	"testing"
)

func TestFragmentDuBalayageEcarte(t *testing.T) { // FR-SUFFIXE, FR-ECRIT, FR-AUTRE
	balayees := []EntreeDeBotBalayee{
		{Slot: 24, BotID: 59, Nom: "343 Hollis", Bit: 900},
		{Slot: 0, BotID: 0, Nom: "43 Hollis", Bit: 17000},
		{Slot: 25, BotID: 3, Nom: "43 Bob", Bit: 18000},
		{Slot: 0, BotID: 0, Nom: "Inconnu", Bit: 19000},
	}
	ecrites := []EntreeDeBotEcrite{
		{Slot: 24, BotID: 59, Nom: "343 Hollis"},
		{Slot: 25, BotID: 3, Nom: "43 Bob"},
		{Slot: 26, BotID: 4, Nom: "343 Bob"},
	}
	out, n := ecarterLesFragments(balayees, ecrites)
	attendu := []EntreeDeBotBalayee{balayees[0], balayees[2], balayees[3]}
	if n != 1 || !reflect.DeepEqual(out, attendu) {
		t.Fatalf("%d ecarte(s), restent %+v ; attendu 1 et %+v", n, out, attendu)
	}
}
