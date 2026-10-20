package grammar

// movement_states_comptes_test.go — LES COMPTES DE PAQUETS DU CANAL DES ETATS DE MOUVEMENT : un
// debut de liste lu a la fin de la vue A n est pas une localisation.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestUnDebutLuNEstNiLocaliseNiOuvertParUnNeuf : chaque paquet a liste d evenements se compte selon
// la facon dont le debut de sa liste a ete trouve ; un debut lu dans la vue A compte parmi les
// paquets a evenements, ni localise ni ouvert par un NEW de tete ; un paquet non localise est
// saute.
func TestUnDebutLuNEstNiLocaliseNiOuvertParUnNeuf(t *testing.T) {
	st := &types.MovementStateStats{}
	sc := &movementStateScanner{st: st}
	for _, d := range []lecture.DebutDeVueB{lecture.DebutEnTete, lecture.DebutParVueA, lecture.DebutParSignature,
		lecture.DebutParChaine, lecture.DebutParFermeture, lecture.DebutNonLocalise} {
		sc.Trame(&lecture.Paquet{Debut: d})
	}
	if st.EventPackets != 5 || st.EventPacketsLocated != 3 || st.EventPacketsNewRecordStart != 2 ||
		st.EventPacketsUnlocated != 1 || st.Packets != 5 {
		t.Fatalf("a evenements %d (5), localises %d (3), ouverts par un NEW %d (2), non localises %d (1), marches %d (5)",
			st.EventPackets, st.EventPacketsLocated, st.EventPacketsNewRecordStart, st.EventPacketsUnlocated, st.Packets)
	}
}
