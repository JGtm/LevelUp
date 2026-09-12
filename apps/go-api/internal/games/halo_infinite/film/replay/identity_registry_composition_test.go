package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// identity_registry_composition_test.go — LE REGISTRE COMPOSE DES LECTURES, ET RIEN D'AUTRE.
//
// Ce fichier RETOURNE `killpos_bridge_test.go` (baseline gelee, trois entrees corrigees dans le
// meme commit, lot P2.3 2026-09-08) : `ResolveSlotXUID` a disparu — le registre est desormais le
// seul producteur du pont, et le collecteur de sync l'appelle comme la cuisson. Les scenarios
// sont INCHANGES, seule la porte d'entree change.
//
// Il verrouille la COMPOSITION (positions -> vies -> pont), pas l'algorithme lui-meme, deja
// couvert par lives_test.go et closures_test.go. Le scenario nominal reprend VOLONTAIREMENT celui
// de TestNameLivesByDeathsJoinsOnEnd : une seconde fixture inventee prouverait moins qu'une
// fixture deja eprouvee traversee par le nouveau chemin d'entree.

func TestRegistreComposeLaLectureSeule(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(513, 20_000_000, 0, 0, 0), posAt(513, 21_000_000, 0, 0, 0),
	}
	deaths := []Death{{XUID: 111, TimeMS: 2_000 - 500}, {XUID: 222, TimeMS: 21_000 - 500}}
	idx := PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 5}

	reg := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: deaths, PlayerIndices: idx})
	pont := reg.PontEpure()

	if pont[512] != 111 || pont[513] != 222 {
		t.Fatalf("pont inattendu : %+v", pont)
	}
	if reg.ViesNommeesParLaLecture() != 2 || reg.CollisionsDeSlot() != 0 {
		t.Errorf("rapport inattendu : nommees=%d collisions=%d",
			reg.ViesNommeesParLaLecture(), reg.CollisionsDeSlot())
	}
}

// TestRegistreSansMortsRendUnPontVide — PAS DE REPLI : sans fil des morts, le pont est vide,
// jamais devine.
func TestRegistreSansMortsRendUnPontVide(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(512, 1_000_000, 0, 0, 0)}
	idx := PlayerIndexTable{ByXUID: map[uint64]int{111: 0}}

	reg := BuildIdentityRegistry(IdentityInput{Positions: pos, PlayerIndices: idx})

	if len(reg.PontEpure()) != 0 {
		t.Fatalf("attendu un pont vide sans fil des morts, obtenu %+v", reg.PontEpure())
	}
	if reg.ViesNommeesParLaLecture() != 0 {
		t.Errorf("vies nommees = %d, attendu 0", reg.ViesNommeesParLaLecture())
	}
}

// TestRegistreSansIndexDeJoueurRendUnPontVide — meme regle, second maillon absent.
func TestRegistreSansIndexDeJoueurRendUnPontVide(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0)}
	deaths := []Death{{XUID: 111, TimeMS: 1_500}}

	reg := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: deaths})

	if len(reg.PontEpure()) != 0 {
		t.Fatalf("attendu un pont vide sans index de joueur, obtenu %+v", reg.PontEpure())
	}
}
