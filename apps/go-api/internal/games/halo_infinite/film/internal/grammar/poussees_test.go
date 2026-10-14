package grammar

// poussees_test.go — LES MESSAGES DE POUSSEE DE LA VUE A ET LEUR APPARIEMENT (vue_a_poussees.go,
// poussees.go).

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ecrireUnePoussee ecrit un message de poussee : continuation, genre, trois references (les deux
// premieres presentes quand leur index est positif), puis la charge que `charge` ecrit.
func (w *bitWriter) ecrireUnePoussee(genre int, ref0, ref1 int, charge func(w *bitWriter)) {
	w.bit(1)
	w.bits(uint64(genre), LargeurGenreVueA)
	for _, r := range [2]int{ref0, ref1} {
		if r < 0 {
			w.bit(0)
			continue
		}
		w.bit(1)
		w.bits(uint64(r), int(varWidthBits(0)))
		w.bits(1, 2) // generation
	}
	w.bit(0) // reference 2 absente
	charge(w)
}

// TestLaVueARangeLesPoussees : une liste 105 puis 104 de meme reference 0 se lit jusqu a son
// terminateur, et la vue A range les deux messages avec leurs references et leur charge.
// MUTATIONS — la direction et la norme du 104 interverties, la reference 1 du 105 jetee : ROUGES.
func TestLaVueARangeLesPoussees(t *testing.T) {
	var w bitWriter
	w.bit(1) // bit de configuration
	w.ecrireUnePoussee(GenreObjetRepousse, 7, 1, func(w *bitWriter) {
		w.bit(1)
		w.bits(0x3c38cb9c, 32)
	})
	w.ecrireUnePoussee(GenrePousseeDuJoueur, 7, -1, func(w *bitWriter) {
		w.bit(0)
		w.bits(156974, largeurDirection)
		w.bits(652, largeurMagnitudePoussee)
	})
	w.bit(0) // terminateur
	w.bits(0, 16)
	a := lireSous(w.buf, grammaireRecente())
	if !a.Porte || len(a.Poussees) != 2 {
		t.Fatalf("vue A : portee %v, %d poussee(s), attendu portee et 2", a.Porte, len(a.Poussees))
	}
	o, p := a.Poussees[0], a.Poussees[1]
	if o.Genre != GenreObjetRepousse || o.Ref != [2]uint16{7, 1} || o.Mot != 0x3c38cb9c ||
		!aLeDrapeau(o, lecture.PousseeRef0) || !aLeDrapeau(o, lecture.PousseeRef1) || !aLeDrapeau(o, lecture.PousseeMotLu) {
		t.Errorf("105 lu %+v", o)
	}
	if p.Genre != GenrePousseeDuJoueur || p.Ref[0] != 7 || aLeDrapeau(p, lecture.PousseeRef1) ||
		p.Direction != 156974 || p.Norme != 652 || !aLeDrapeau(p, lecture.PousseeImpulsionLue) {
		t.Errorf("104 lu %+v", p)
	}
	if o.Debut != 1 {
		t.Errorf("debut du 105 = %d, attendu 1 (la continuation qui l annonce)", o.Debut)
	}
}

// TestLesPousseesSontAppariees : un 104 n est rendu que precede de son 105 (meme reference 0, meme
// generation) dans la meme liste ; son auteur est la reference 1 du 105, base 512.
func TestLesPousseesSontAppariees(t *testing.T) {
	o := lecture.MessageDePoussee{Genre: GenreObjetRepousse, Ref: [2]uint16{7, 1}, Gen: [2]uint8{1, 1},
		Drapeaux: lecture.PousseeRef0 | lecture.PousseeRef1}
	p := lecture.MessageDePoussee{Genre: GenrePousseeDuJoueur, Ref: [2]uint16{7}, Gen: [2]uint8{1},
		Direction: 156974, Norme: 652, Drapeaux: lecture.PousseeRef0 | lecture.PousseeImpulsionLue}
	autre := p
	autre.Ref[0] = 9
	c := &canalDesPoussees{}
	pq := &lecture.Paquet{TS: 1000}
	pq.VueA.Poussees = []lecture.MessageDePoussee{p, o, p, autre}
	c.Tete(pq)
	st := c.st
	if st.Read104 != 3 || st.Read105 != 1 || st.Paired != 1 || st.Unpaired != 2 {
		t.Fatalf("stats %+v : attendu 3 x 104, 1 x 105, 1 paire, 2 non appariees", st)
	}
	k := c.out[0]
	if k.TimestampUS != 1000 || k.Pushed != (types.LifeKey{Slot: 519, Gen: 1}) || !k.HasAuthor ||
		k.Author != (types.LifeKey{Slot: 513, Gen: 1}) || !k.HasImpulse {
		t.Errorf("poussee %+v", k)
	}
	n := math.Hypot(math.Hypot(float64(k.Impulse[0]), float64(k.Impulse[1])), float64(k.Impulse[2]))
	if math.Abs(n-6.0) > 0.01 {
		t.Errorf("norme de l impulsion %.3f, attendu 6,00", n)
	}
}

// TestLaNormeDeLaPousseeSuitLaLoiDuJeu : `FUN_14076d6dc` sur 10 bits entre 0,05 et 20 — les deux
// bornes et les deux normes du repulseur lues sur le parc (652 -> 6,00 ; 627 -> 5,50).
func TestLaNormeDeLaPousseeSuitLaLoiDuJeu(t *testing.T) {
	for _, c := range []struct {
		q    uint16
		want float64
	}{{0, 0.05}, {1023, 20}, {652, 6.00}, {627, 5.50}} {
		if got := normeDeLaPoussee(c.q); math.Abs(got-c.want) > 0.005 {
			t.Errorf("norme(%d) = %.4f, attendu %.2f", c.q, got, c.want)
		}
	}
}

// TestUneAutreGenerationNeSAppariePas : le 105 qui precede porte le meme index mais une autre
// generation — un autre objet du meme slot ; le 104 n est pas apparie. MUTATION : la comparaison de
// generation retiree de [apparierLaPoussee] -> ROUGE.
func TestUneAutreGenerationNeSAppariePas(t *testing.T) {
	o := lecture.MessageDePoussee{Genre: GenreObjetRepousse, Ref: [2]uint16{7, 1}, Gen: [2]uint8{2, 1},
		Drapeaux: lecture.PousseeRef0 | lecture.PousseeRef1}
	p := lecture.MessageDePoussee{Genre: GenrePousseeDuJoueur, Ref: [2]uint16{7}, Gen: [2]uint8{1},
		Direction: 156974, Norme: 652, Drapeaux: lecture.PousseeRef0 | lecture.PousseeImpulsionLue}
	if k, ok := apparierLaPoussee([]lecture.MessageDePoussee{o}, p); ok {
		t.Fatalf("apparie a un 105 d une autre generation : %+v", k)
	}
}
