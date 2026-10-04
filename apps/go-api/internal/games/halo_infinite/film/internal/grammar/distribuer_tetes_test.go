package grammar

// distribuer_tetes_test.go — LA TETE DE CHAQUE TRAME, LUE UNE FOIS (lot 2.3 du plan de l etape 2) :
// la vue A rangee, la passe des tetes, et l egalite avec la tete lue hors de la structure.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// TestRangerLaTete : les trois formes de la tete — liste vide (vue terminee sur la continuation),
// un message (vue arretee apres son genre), une tete qui ne tient pas dans le payload (vue arretee
// sans genre, et la lecture tolerante d aujourd hui pour les canaux).
// MUTATION — ranger la vue A sans son genre (`consumeVueA` sans `Genres`) : ROUGE.
func TestRangerLaTete(t *testing.T) {
	cas := []struct {
		nom     string
		payload []byte
		vueA    lecture.VueA
		tete    teteDeTrame
	}{
		{"liste vide", []byte{0x80, 0x00}, lecture.VueA{Debut: 1, Bits: 1, Etat: lecture.VueTerminee}, teteDeTrame{}},
		// config 1, continuation 1, genre 36 = 0b0100100 : 0xD2 puis le bit de poids faible du genre
		{"un message", []byte{0xD2, 0x00}, lecture.VueA{Debut: 1, Bits: 8, Etat: lecture.VueArretee, Genres: []uint8{36}},
			teteDeTrame{liste: true, genre: 36}},
		{"tete tronquee", []byte{0xD2}, lecture.VueA{Debut: 1, Bits: 1, Etat: lecture.VueArretee},
			teteDuPayload([]byte{0xD2})},
	}
	for _, c := range cas {
		p := lecture.Paquet{Payload: c.payload}
		rangerLaTete(&p)
		if p.VueA.Debut != c.vueA.Debut || p.VueA.Bits != c.vueA.Bits || p.VueA.Etat != c.vueA.Etat ||
			!reflect.DeepEqual(append([]uint8(nil), p.VueA.Genres...), append([]uint8(nil), c.vueA.Genres...)) {
			t.Errorf("%s : vue A %+v, attendu %+v", c.nom, p.VueA, c.vueA)
		}
		if got := teteDe(&p); got != c.tete {
			t.Errorf("%s : tete %+v, attendu %+v", c.nom, got, c.tete)
		}
	}
}

// canalDeTeteTemoin retient la tete de chaque trame qu on lui donne.
type canalDeTeteTemoin struct {
	vues  []teteVue
	clos  int
	bilan BilanDeMarche
}

// teteVue : ce qu un canal de tete garde d une trame.
type teteVue struct {
	chunk, index int
	vueA         lecture.VueA
	tete         teteDeTrame
	duPayload    teteDeTrame
}

func (*canalDeTeteTemoin) Interets() []Interet { return nil }

func (c *canalDeTeteTemoin) Tete(p *lecture.Paquet) {
	a := p.VueA
	a.Genres = append([]uint8(nil), a.Genres...)
	c.vues = append(c.vues, teteVue{chunk: p.Chunk, index: p.Index, vueA: a, tete: teteDe(p),
		duPayload: teteDuPayload(p.Payload)})
}

func (c *canalDeTeteTemoin) Clore(b BilanDeMarche) { c.clos, c.bilan = c.clos+1, b }

// TestLaPasseDesTetesEstLaTeteDeLaMarche : sur les deux bobines a trames delta, la passe des tetes
// (aucun canal des trames) rend les memes trames, avec la meme vue A, que la marche complete ; la
// tete rangee est celle que la lecture hors structure rend ; la passe ne marche ni les images-cles
// ni les records.
// MUTATION — la marche complete qui ne range plus la tete d une trame a liste (seule la
// continuation, sans genre) : ROUGE.
func TestLaPasseDesTetesEstLaTeteDeLaMarche(t *testing.T) {
	for _, court := range []string{"000d5950", "e5adf7b2"} {
		film := bobineDeTrames(t, court)
		fc := contexteDeBobine(film)
		seule := &canalDeTeteTemoin{}
		if err := Distribuer(fc, seule); err != nil {
			t.Fatalf("%s : passe des tetes : %v", court, err)
		}
		if fc.marches != nil || seule.bilan.Obs != nil || seule.clos != 1 || seule.bilan.ChunksLus == 0 {
			t.Errorf("%s : memoire d ancres %v, observation %v, %d cloture(s), bilan %+v : la passe des "+
				"tetes a marche autre chose que les tetes", court, fc.marches != nil, seule.bilan.Obs, seule.clos,
				seule.bilan)
		}
		avecMarche := &canalDeTeteTemoin{}
		if err := Distribuer(contexteDeBobine(film), &canalDesTramesTemoin{}, avecMarche); err != nil {
			t.Fatalf("%s : marche complete : %v", court, err)
		}
		if len(seule.vues) == 0 || !reflect.DeepEqual(seule.vues, avecMarche.vues) {
			t.Fatalf("%s : %d tete(s) par la passe, %d par la marche, ou differentes", court, len(seule.vues),
				len(avecMarche.vues))
		}
		listes := 0
		for _, v := range seule.vues {
			if v.tete != v.duPayload && v.duPayload.liste {
				t.Fatalf("%s paquet %d:%d : tete %+v, %+v hors structure", court, v.chunk, v.index, v.tete, v.duPayload)
			}
			if v.tete.liste != v.duPayload.liste {
				t.Fatalf("%s paquet %d:%d : continuation %v, %v hors structure", court, v.chunk, v.index,
					v.tete.liste, v.duPayload.liste)
			}
			if v.tete.liste {
				listes++
			}
		}
		if listes == 0 {
			t.Errorf("%s : aucune trame a liste d evenements : la tete n est pas exercee", court)
		}
	}
}
