package grammar

// distribuer_test.go — LE DISTRIBUTEUR : UNE MARCHE, N CANAUX (ADR 0037 ; decisions DT2-1 et DT2-2
// du plan de l etape 2). Les deux phases rendues comme leurs iterateurs, une occurrence interpretee
// seulement quand un canal l interprete, un crochet pose par un seul canal.

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineDeTrames charge la mini-bobine `court` de killsource, qui porte des trames delta.
func bobineDeTrames(t *testing.T, court string) *source.Film {
	t.Helper()
	dir := filepath.Join("..", "facts", "killsource", "testdata", "minibobine_"+court)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bobine absente (%s) : %v", dir, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	return film
}

// resumeDePaquet est ce qu un test garde d un paquet rendu, au-dela de son tour.
type resumeDePaquet struct {
	imageCle                                  bool
	chunk, index, records, comps, interpretes int
	verdict                                   lecture.Verdict
	prouves                                   int
}

// resumer resume un paquet, et rend les occurrences interpretees qu il porte, par archetype.
func resumer(p *lecture.Paquet, imageCle bool) (resumeDePaquet, map[int16][]uint8) {
	r := resumeDePaquet{imageCle: imageCle, chunk: p.Chunk, index: p.Index, records: len(p.Records),
		comps: len(p.Comps), verdict: p.Fermeture.Verdict}
	interpretes := map[int16][]uint8{}
	for _, rec := range p.Records {
		if rec.Preuve == lecture.PreuveFerme {
			r.prouves++
		}
		for _, c := range p.Comps[rec.Comps[0]:rec.Comps[1]] {
			if c.Etat == lecture.EtatInterprete {
				r.interpretes++
				interpretes[rec.TI] = append(interpretes[rec.TI], c.Index)
			}
		}
	}
	return r, interpretes
}

// canalTemoin retient ce que la marche lui donne.
type canalTemoin struct {
	interets    []Interet
	poser       func(obs *Observation, m *MarcheDistribuee)
	m           *MarcheDistribuee
	recus       []resumeDePaquet
	interpretes map[int16][]uint8
	clos        int
	bilan       BilanDeMarche
}

func (c *canalTemoin) Interets() []Interet { return c.interets }

func (c *canalTemoin) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.m = m
	if c.poser != nil {
		c.poser(obs, m)
	}
}

func (c *canalTemoin) recevoir(p *lecture.Paquet, imageCle bool) {
	r, it := resumer(p, imageCle)
	c.recus = append(c.recus, r)
	if c.interpretes == nil {
		c.interpretes = map[int16][]uint8{}
	}
	for ti, ix := range it {
		c.interpretes[ti] = append(c.interpretes[ti], ix...)
	}
}

func (c *canalTemoin) ImageCle(p *lecture.Paquet) { c.recevoir(p, true) }
func (c *canalTemoin) Trame(p *lecture.Paquet)    { c.recevoir(p, false) }
func (c *canalTemoin) Clore(b BilanDeMarche)      { c.clos, c.bilan = c.clos+1, b }

// lesPhasesParLeursIterateurs rend les paquets des deux phases tels que leurs iterateurs les rendent,
// sur un contexte neuf.
func lesPhasesParLeursIterateurs(t *testing.T, film *source.Film) []resumeDePaquet {
	t.Helper()
	var out []resumeDePaquet
	fc := contexteDeBobine(film)
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("images-cles : %v", err)
		}
		r, _ := resumer(p, true)
		out = append(out, r)
	}
	for p, err := range fc.Trames(nil) {
		if err != nil {
			t.Fatalf("trames : %v", err)
		}
		r, _ := resumer(p, false)
		out = append(out, r)
	}
	return out
}

// TestDistribuerRendLesDeuxPhasesCommeLeursIterateurs : sur les deux bobines a trames delta, le
// canal recoit toutes les images-cles puis toutes les trames, dans l ordre du flux, les memes que
// les iterateurs rendent, puis le bilan, une fois ; l en-tete est celui du contexte.
func TestDistribuerRendLesDeuxPhasesCommeLeursIterateurs(t *testing.T) {
	for _, court := range []string{"000d5950", "e5adf7b2"} {
		film := bobineDeTrames(t, court)
		fc := contexteDeBobine(film)
		c := &canalTemoin{}
		if err := Distribuer(fc, c); err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		attendu := lesPhasesParLeursIterateurs(t, film)
		if !reflect.DeepEqual(c.recus, attendu) {
			t.Errorf("%s : %d paquet(s) distribue(s), %d par les iterateurs, ou un ordre different",
				court, len(c.recus), len(attendu))
		}
		images, trames := 0, 0
		for _, r := range c.recus {
			if r.imageCle {
				if trames > 0 {
					t.Fatalf("%s : une image-cle apres une trame", court)
				}
				images++
			} else {
				trames++
			}
		}
		if images == 0 || trames == 0 || c.clos != 1 || c.bilan.Obs == nil || c.m.EnTete != fc.EnTete() {
			t.Errorf("%s : %d image(s)-cle(s), %d trame(s), %d cloture(s), bilan %+v, en-tete %+v",
				court, images, trames, c.clos, c.bilan, c.m.EnTete)
		}
	}
}

// TestUneOccurrenceInterpreteeEstUnInteretTraverse : DT2-2. Un canal qui interprete le composant
// d accroupissement du bipede voit ses occurrences traversees marquees interpretees, et elles
// seules ; la meme marche sans canal n interprete rien.
// MUTATION — marquer interpretee toute occurrence portee (`composantLu` qui ignore `interesse`) :
// ROUGE.
func TestUneOccurrenceInterpreteeEstUnInteretTraverse(t *testing.T) {
	film := bobineDeTrames(t, "000d5950")
	fc := contexteDeBobine(film)
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatalf("archetype bipede : %v", err)
	}
	voulu := componentIndexOfAny(arch, crouchComponentName, crouchComponentNameAlt)
	if voulu < 0 {
		t.Fatal("la bobine ne declare pas l accroupissement du bipede : le test ne prouve rien")
	}
	c := &canalTemoin{interets: []Interet{{TI: BipedTypeIndex, Composant: crouchComponentName},
		{TI: BipedTypeIndex, Composant: crouchComponentNameAlt}}}
	if err := Distribuer(fc, c); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	n := 0
	for ti, ix := range c.interpretes {
		for _, i := range ix {
			if ti != BipedTypeIndex || int(i) != voulu {
				t.Fatalf("occurrence (%d, %d) interpretee, seule (%d, %d) est un interet", ti, i,
					BipedTypeIndex, voulu)
			}
			n++
		}
	}
	if n == 0 {
		t.Fatal("aucune occurrence interpretee : la marque n est pas exercee")
	}
	for _, r := range lesPhasesParLeursIterateurs(t, film) {
		if r.interpretes != 0 {
			t.Fatalf("paquet %d:%d : %d occurrence(s) interpretee(s) sans canal", r.chunk, r.index, r.interpretes)
		}
	}
}

// TestLesCrochetsDUnCanalRecoiventLaMarche : un crochet pose par un canal est appele pendant la
// marche, sous le paquet en cours ; deux canaux qui posent le meme crochet sont refuses.
func TestLesCrochetsDUnCanalRecoiventLaMarche(t *testing.T) {
	appels, sansPaquet := 0, 0
	c := &canalTemoin{}
	c.poser = func(obs *Observation, m *MarcheDistribuee) {
		obs.EtatMouvementHook = func(EtatMouvementComposant, uint32, []uint64) {
			appels++
			if m.Paquet == nil || m.Paquet.Type != PacketTypeDelta {
				sansPaquet++
			}
		}
	}
	if err := Distribuer(contexteDeBobine(bobineDeTrames(t, "000d5950")), c); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	if appels == 0 || sansPaquet != 0 {
		t.Errorf("%d appel(s) du crochet, %d hors d une trame en cours", appels, sansPaquet)
	}
	double := &canalTemoin{poser: c.poser}
	err := Distribuer(contexteDeBobine(bobineDeTrames(t, "000d5950")), &canalTemoin{poser: c.poser}, double)
	if !errors.Is(err, ErrCrochetDejaPose) {
		t.Errorf("deux canaux posent le meme crochet : %v, attendu %v", err, ErrCrochetDejaPose)
	}
}

// TestLesCrochetsSontExportes : le distributeur fond les crochets par reflexion ; un crochet non
// exporte ne pourrait pas l etre.
func TestLesCrochetsSontExportes(t *testing.T) {
	typ := reflect.TypeFor[Observation]()
	crochets := 0
	for i := range typ.NumField() {
		if f := typ.Field(i); f.Type.Kind() == reflect.Func {
			crochets++
			if !f.IsExported() {
				t.Errorf("crochet non exporte : %s", f.Name)
			}
		}
	}
	if crochets == 0 {
		t.Fatal("aucun crochet dans l observation : le test ne prouve rien")
	}
}
