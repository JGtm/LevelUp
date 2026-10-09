package grammar

// distribuer_test.go — LE DISTRIBUTEUR : UNE MARCHE, N CANAUX (ADR 0037 ; decisions DT2-1 et DT2-2
// du plan de l etape 2, decisions 1 a 5 du lot 2.2). Les phases rendues comme leurs iterateurs, une
// occurrence interpretee seulement quand un canal l interprete dans sa phase, un corps d image-cle
// parcouru seulement quand il est lu, un crochet pose par un seul canal.

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
	prouves, nonParcourus                     int
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
		if rec.Desync == lecture.CorpsNonParcouru {
			r.nonParcourus++
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

// canalTemoin retient ce que la marche lui donne : un canal d image-cle, ou un canal des trames
// ([canalDesTramesTemoin]).
type canalTemoin struct {
	interets    []Interet
	recus       []resumeDePaquet
	interpretes map[int16][]uint8
	// ancresDecalees : les paquets d image-cle dont la marche d ancres exposee ne porte pas les
	// records du paquet, dans le meme ordre.
	ancresDecalees int
	clos           int
	bilan          BilanDeMarche
}

func (c *canalTemoin) Interets() []Interet { return c.interets }

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

func (c *canalTemoin) ImageCle(p *lecture.Paquet, m *MarcheDistribuee) {
	c.recevoir(p, true)
	if len(m.Ancres.Records) != len(p.Records) {
		c.ancresDecalees++
		return
	}
	for i, a := range m.Ancres.Records {
		if uint32(a.Bit) != p.Records[i].Debut || uint32(a.Slot) != p.Records[i].Vie.Slot { //nolint:gosec // ancres bornees
			c.ancresDecalees++
			return
		}
	}
}

func (c *canalTemoin) Clore(b BilanDeMarche) { c.clos, c.bilan = c.clos+1, b }

// canalDesTramesTemoin est le canal temoin qui lit aussi les trames et pose des crochets.
type canalDesTramesTemoin struct {
	canalTemoin
	poser func(obs *Observation, m *MarcheDistribuee)
	m     *MarcheDistribuee
}

func (c *canalDesTramesTemoin) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.m = m
	if c.poser != nil {
		c.poser(obs, m)
	}
}

func (c *canalDesTramesTemoin) Trame(p *lecture.Paquet) { c.recevoir(p, false) }

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

// TestDistribuerRendLesDeuxPhasesCommeLeursIterateurs : sur les deux bobines a trames delta, un
// canal des trames sans interet recoit toutes les images-cles puis toutes les trames, dans l ordre
// du flux : les images-cles des iterateurs, sans corps parcouru, chacune avec sa marche d ancres ;
// les memes trames ; puis le bilan, une fois ; l en-tete est celui du contexte.
func TestDistribuerRendLesDeuxPhasesCommeLeursIterateurs(t *testing.T) {
	for _, court := range []string{"000d5950", "e5adf7b2"} {
		film := bobineDeTrames(t, court)
		fc := contexteDeBobine(film)
		c := &canalDesTramesTemoin{}
		if err := Distribuer(fc, c); err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		attendu := lesPhasesParLeursIterateurs(t, film)
		if len(c.recus) != len(attendu) {
			t.Fatalf("%s : %d paquet(s) distribue(s), %d par les iterateurs", court, len(c.recus), len(attendu))
		}
		images := 0
		for i, r := range c.recus {
			a := attendu[i]
			if r.imageCle {
				images++
				// SANS INTERET, AUCUN CORPS : l identite et l ordre des records sont ceux de la
				// marche complete, rien d autre.
				a.comps, a.prouves, a.nonParcourus = 0, 0, a.records
			}
			if r != a {
				t.Fatalf("%s paquet %d : distribue %+v, iterateur %+v", court, i, r, a)
			}
		}
		if images == 0 || images == len(c.recus) || c.ancresDecalees != 0 || c.clos != 1 || c.bilan.Obs == nil ||
			c.bilan.ChunksLus == 0 || c.m.EnTete != fc.EnTete() {
			t.Errorf("%s : %d image(s)-cle(s) sur %d paquets, %d marche(s) d ancres decalee(s), %d cloture(s), "+
				"bilan %+v, en-tete %+v", court, images, len(c.recus), c.ancresDecalees, c.clos, c.bilan, c.m.EnTete)
		}
	}
}

// TestUnCorpsDImageCleNEstParcouruQueSIlEstLu : decision 3 du lot 2.2. Un canal qui interprete le
// designateur d equipe du joueur gere (ti=9) dans la phase des images-cles fait parcourir les corps
// de ti=9, et eux seuls : chacun a l etendue, la preuve et les composants que la marche complete
// lui donne ; son i0 est interprete, rien d autre ; aucune trame n est marchee.
// MUTATION — parcourir tous les corps dans une distribution (`parcourt` qui rend vrai) : ROUGE.
func TestUnCorpsDImageCleNEstParcouruQueSIlEstLu(t *testing.T) {
	film := bobineDeTrames(t, "e5adf7b2")
	complets := map[[2]int]paquetComplet{}
	fc := contexteDeBobine(film)
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("images-cles : %v", err)
		}
		complets[[2]int{p.Chunk, p.Index}] = paquetComplet{records: append([]lecture.Record(nil), p.Records...),
			comps: append([]lecture.Composant(nil), p.Comps...)}
	}
	c := &canalTemoin{interets: []Interet{{Phase: PhaseImagesCles, TI: managedPlayerTypeIndex,
		Composant: teamDesignatorComponent}}}
	if err := Distribuer(contexteDeBobine(film), c); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	parcourus := 0
	for ti, ix := range c.interpretes {
		for _, i := range ix {
			if ti != managedPlayerTypeIndex || i != 0 {
				t.Fatalf("occurrence (%d, %d) interpretee, seule (9, 0) est un interet", ti, i)
			}
			parcourus++
		}
	}
	for _, r := range c.recus {
		if !r.imageCle {
			t.Fatal("une trame distribuee a un canal qui ne lit que les images-cles")
		}
	}
	if parcourus == 0 || c.bilan.Obs != nil {
		t.Fatalf("%d i0 interprete(s), observation des trames %v : la lecture des corps n est pas exercee",
			parcourus, c.bilan.Obs)
	}
	verifierLesCorpsLus(t, film, complets)
}

// paquetComplet garde les records et les composants d un paquet de la marche complete.
type paquetComplet struct {
	records []lecture.Record
	comps   []lecture.Composant
}

// verifierLesCorpsLus rejoue la distribution aux corps de ti=9 et compare chaque record a celui de
// la marche complete : meme etendue, meme preuve et memes composants pour ti=9 (i0 interprete au
// lieu de delimite), aucun corps pour les autres.
func verifierLesCorpsLus(t *testing.T, film *source.Film, complets map[[2]int]paquetComplet) {
	t.Helper()
	c := &canalDeComparaison{complets: complets, t: t}
	if err := Distribuer(contexteDeBobine(film), c); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	if c.compares == 0 {
		t.Fatal("aucun record ti=9 compare")
	}
}

// canalDeComparaison compare les records d image-cle a ceux de la marche complete.
type canalDeComparaison struct {
	complets map[[2]int]paquetComplet
	compares int
	t        *testing.T
}

func (c *canalDeComparaison) Interets() []Interet {
	return []Interet{{Phase: PhaseImagesCles, TI: managedPlayerTypeIndex, Composant: teamDesignatorComponent}}
}

func (c *canalDeComparaison) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	attendu := c.complets[[2]int{p.Chunk, p.Index}]
	if len(attendu.records) != len(p.Records) {
		c.t.Fatalf("paquet %d:%d : %d record(s), %d dans la marche complete", p.Chunk, p.Index, len(p.Records),
			len(attendu.records))
	}
	for i, r := range p.Records {
		a := attendu.records[i]
		if int(r.TI) != managedPlayerTypeIndex {
			if r.Desync != lecture.CorpsNonParcouru || r.Bits != 0 || r.Comps[0] != r.Comps[1] {
				c.t.Fatalf("paquet %d:%d record %d (ti=%d) : corps parcouru sans canal qui le lise", p.Chunk, p.Index, i, r.TI)
			}
			continue
		}
		c.comparerLesComposants(p.Comps[r.Comps[0]:r.Comps[1]], attendu.comps[a.Comps[0]:a.Comps[1]])
		r.Comps, a.Comps = [2]uint32{}, [2]uint32{}
		if r != a {
			c.t.Fatalf("paquet %d:%d record %d : %+v, marche complete %+v", p.Chunk, p.Index, i, r, a)
		}
		c.compares++
	}
}

// comparerLesComposants compare les composants d un record ti=9 a ceux de la marche complete : i0
// interprete la ou elle le delimite, le reste a l identique.
func (c *canalDeComparaison) comparerLesComposants(lus, complets []lecture.Composant) {
	if len(lus) != len(complets) {
		c.t.Fatalf("%d composant(s), %d dans la marche complete", len(lus), len(complets))
	}
	for k, x := range lus {
		y := complets[k]
		if x.Index == 0 && y.Etat == lecture.EtatDelimite {
			y.Etat = lecture.EtatInterprete
		}
		if x != y {
			c.t.Fatalf("composant %d : %+v, marche complete %+v", k, x, y)
		}
	}
}

func (*canalDeComparaison) Clore(BilanDeMarche) {}

// TestUneOccurrenceInterpreteeEstUnInteretTraverse : DT2-2, et sa phase. Un canal qui interprete le
// bouclier du bipede dans les TRAMES voit ses occurrences traversees des trames marquees
// interpretees, et elles seules ; aucune image-cle n en porte (aucun crochet n y recoit de valeur) ;
// la meme marche sans canal n interprete rien.
// MUTATION — marquer interpretee toute occurrence portee (`composantLu` qui ignore `interesse`) :
// ROUGE. MUTATION — resoudre chaque interet dans les deux phases (`resoudreLesInterets` qui ignore
// `Phase`) : ROUGE.
func TestUneOccurrenceInterpreteeEstUnInteretTraverse(t *testing.T) {
	film := bobineDeTrames(t, "000d5950")
	fc := contexteDeBobine(film)
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatalf("archetype bipede : %v", err)
	}
	voulu := componentIndexOfAny(arch, compObjectShieldVitality)
	if voulu < 0 {
		t.Fatal("la bobine ne declare pas le bouclier du bipede : le test ne prouve rien")
	}
	c := &canalDesTramesTemoin{canalTemoin: canalTemoin{interets: []Interet{
		{TI: BipedTypeIndex, Composant: compObjectShieldVitality}}}}
	if err := Distribuer(fc, c); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	n := 0
	for ti, ix := range c.interpretes {
		for _, i := range ix {
			if ti != BipedTypeIndex || int(i) != voulu {
				t.Fatalf("occurrence (%d, %d) interpretee, seule (%d, %d) est un interet", ti, i, BipedTypeIndex, voulu)
			}
			n++
		}
	}
	for _, r := range c.recus {
		if r.imageCle && r.interpretes != 0 {
			t.Fatalf("image-cle %d:%d : %d occurrence(s) interpretee(s) pour un interet des trames", r.chunk, r.index, r.interpretes)
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

// TestUneDistributionDImageCleLitUnFilmSansRegistre : decision 4 du lot 2.2. Sur la bobine
// historique sans `chunk_00`, un canal d image-cle recoit les ancres, sans corps, et son bilan ;
// un canal des trames rend l erreur du registre.
// MUTATION — une distribution d image-cle qui s arrete sans registre (`distribuerLaDemande` qui
// rend sur l erreur) : ROUGE.
func TestUneDistributionDImageCleLitUnFilmSansRegistre(t *testing.T) {
	film := bobineParBuild(t, "000d5950") // la bobine historique, sans `chunk_00`
	c := &canalTemoin{interets: []Interet{{Phase: PhaseImagesCles, TI: managedPlayerTypeIndex,
		Composant: teamDesignatorComponent}}}
	if err := Distribuer(NewFilmContext(film), c); err != nil {
		t.Fatalf("distribution d image-cle : %v", err)
	}
	records, nonParcourus := 0, 0
	for _, r := range c.recus {
		records += r.records
		nonParcourus += r.nonParcourus
	}
	if records == 0 || nonParcourus != records || c.clos != 1 || c.bilan.ChunksLus == 0 || c.ancresDecalees != 0 {
		t.Errorf("film sans registre : %d record(s), %d sans corps, %d cloture(s), bilan %+v", records, nonParcourus,
			c.clos, c.bilan)
	}
	if err := Distribuer(NewFilmContext(film), &canalDesTramesTemoin{}); err == nil || errors.Is(err, ErrNoFilmChunk) {
		t.Errorf("canal des trames sur un film sans registre : %v, attendu l erreur du registre", err)
	}
}

// TestLesCrochetsDUnCanalRecoiventLaMarche : un crochet pose par un canal est appele pendant la
// marche, sous le paquet en cours ; deux canaux qui posent le meme crochet sont refuses.
func TestLesCrochetsDUnCanalRecoiventLaMarche(t *testing.T) {
	appels, sansPaquet := 0, 0
	poser := func(obs *Observation, m *MarcheDistribuee) {
		obs.EtatMouvementHook = func(EtatMouvementComposant, uint32, []uint64) {
			appels++
			if m.Paquet == nil || m.Paquet.Type != PacketTypeDelta {
				sansPaquet++
			}
		}
	}
	if err := Distribuer(contexteDeBobine(bobineDeTrames(t, "000d5950")), &canalDesTramesTemoin{poser: poser}); err != nil {
		t.Fatalf("distribution : %v", err)
	}
	if appels == 0 || sansPaquet != 0 {
		t.Errorf("%d appel(s) du crochet, %d hors d une trame en cours", appels, sansPaquet)
	}
	err := Distribuer(contexteDeBobine(bobineDeTrames(t, "000d5950")), &canalDesTramesTemoin{poser: poser},
		&canalDesTramesTemoin{poser: poser})
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
