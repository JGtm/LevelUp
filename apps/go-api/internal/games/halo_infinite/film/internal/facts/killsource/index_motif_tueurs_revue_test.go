package killsource

// index_motif_tueurs_revue_test.go — UN TUEUR SANS MORT NE FAIT PLUS TOMBER LE LIEN PAR MOTIF DU
// FILM (revue adverse du lot J7, constat 2 sur FK-3).
//
// FK-3 fait chercher au motif les joueurs qui tuent sans mourir. Un remplacant qui reprend l indice
// d un partant (present dans la table) et qui tue sans mourir est alors lu au MEME indice que lui :
// collision, `desaccords++`, et [indexParMotif.refuserSiElleSeContredit] vidait TOUT l epinglage par
// motif du film — le mecanisme de `b1ad85eb` d avant le lot 5.2b.1. Un tel candidat ne sert qu a
// COMPLETER : quand sa lecture se contredit ou tombe sur un indice deja pris, il est ecarte SEUL, et
// compte.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : traiter les tueurs sans mort comme les autres xuids dans
// [indexParMotif.retenirLesLectures].

import (
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestMotif_UnTueurSansMortNAnnulePasLEpinglageDesAutres(t *testing.T) {
	// A (xuid 1) : le partant, siege 5 de la table ; B (xuid 3) : meurt au feed ; R (xuid 2) : le
	// remplacant, qui TUE sans mourir.
	slots := []types.PlayerSlot{{FilmIndex: 5, XUID: 1, Gamertag: "A"}}
	kf := buildFeed([]highlightevent.HighlightEvent{
		{EventType: highlightevent.EventTypeKill, XUID: 2, Gamertag: "R", TimeMS: 1000},
		{EventType: highlightevent.EventTypeDeath, XUID: 3, Gamertag: "B", TimeMS: 1000},
	})
	nom, xuids := xuidsNommesParLeFilm(slots, kf)
	tueurs := tueursSansMort(slots, kf, nom)
	if !tueurs[2] || tueurs[1] || tueurs[3] {
		t.Fatalf("tueurs sans mort = %v, attendu le seul xuid 2", tueurs)
	}

	for _, cas := range []struct {
		nom string
		vus map[uint64]map[int]int
	}{
		{"le remplacant lu a l indice du partant", map[uint64]map[int]int{
			1: {5: 3}, 2: {5: 4}, 3: {6: 7}}},
		{"le remplacant lu a deux indices", map[uint64]map[int]int{
			1: {5: 3}, 2: {5: 1, 9: 2}, 3: {6: 7}}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			m := indexParMotif{nomParIndex: map[int]string{}}
			m.retenirLesLectures(cas.vus, xuids, nom, tueurs)
			m.refuserSiElleSeContredit()
			if m.nomParIndex[5] != "A" || m.nomParIndex[6] != "B" || len(m.nomParIndex) != 2 {
				t.Errorf("index retenus = %v, attendu {5: A, 6: B} — le tueur sans mort a fait tomber "+
					"l epinglage des autres", m.nomParIndex)
			}
			if m.desaccords != 0 || m.tueursEcartes != 1 {
				t.Errorf("desaccords %d, tueurs ecartes %d — attendu 0 et 1", m.desaccords, m.tueursEcartes)
			}
		})
	}
}
