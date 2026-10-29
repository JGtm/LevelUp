package grammar

// equipement_porte_effet_test.go — LES CANDIDATES A L OBJET D EFFET que le canal de l equipement porte rend a la
// suppression d un objet suivi (plan des durees d effets, B.2) : la fenetre, les exclusions, la fin, la cle recyclee.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// canalDEffets : un canal avec un objet porte (2039, porteur 539) qui suit la vie 1123/1 ; `births` sont les
// dernieres creations ti=38 lues par slot.
func canalDEffets(births ...naissanceLue) *canalDeLEquipementPorte {
	c := &canalDeLEquipementPorte{vies: map[uint32]vieDObjetPorte{}, suivis: map[uint32]objetSuivi{},
		naissances: map[uint32]naissanceLue{}, nes38: map[uint32]naissanceLue{}, nommes: map[types.LifeKey][]uint64{}, effets: map[uint32]objetSuivi{},
		lu: equipementPorteLu{quanta: map[types.LifeKey][3]uint64{}, quantaEffets: map[naissanceCle][3]uint64{}}}
	suivi := types.LifeKey{Slot: 1123, Gen: 1}
	c.suivis[1123] = objetSuivi{gen: 1, objet: types.LifeKey{Slot: 2039, Gen: 1}, porteur: 539}
	c.nommes[suivi] = []uint64{1_000_000}
	for _, b := range births {
		c.nes38[b.obj.Life.Slot] = b
	}
	return c
}

func naissance38(slot, gen uint32, ts uint64) naissanceLue {
	return naissanceLue{obj: types.SpawnedObject{Life: types.LifeKey{Slot: slot, Gen: gen}, TimestampUS: ts,
		TypeIndex: archetypeDeLEffet, HasPosition: true, X: 1, Y: 2, Z: 3}, q: [3]uint64{7, 8, 9}, aQ: true}
}

// fin supprime la vie suivie 1123/1 a l instant ts et rend les lectures du canal.
func fin(c *canalDeLEquipementPorte, ts uint64) []types.CarriedEquipmentRead {
	c.suppression(ts, &FrameRecord{Type: recDel, ID: 1<<30 | 1123, Slot: 1123})
	return c.lu.lectures
}

func genres(ls []types.CarriedEquipmentRead) []types.CarriedReadKind {
	var out []types.CarriedReadKind
	for _, l := range ls {
		out = append(out, l.Kind)
	}
	return out
}

// TestCandidatesALEffetFenetre : une vie ti=38 non nommee nee entre 700 ms et 100 ms avant la suppression est
// candidate, bornes comprises ; 1 us de plus ou de moins l exclut.
func TestCandidatesALEffetFenetre(t *testing.T) {
	const ts = uint64(10_000_000)
	cas := []struct {
		nom    string
		ne     uint64
		retenu bool
	}{
		{"borne ancienne (700 ms)", ts - 700_000, true},
		{"trop ancienne (700 ms + 1 us)", ts - 700_001, false},
		{"borne recente (100 ms)", ts - 100_000, true},
		{"trop recente (100 ms - 1 us)", ts - 99_999, false},
		{"au meme instant", ts, false},
	}
	for _, k := range cas {
		t.Run(k.nom, func(t *testing.T) {
			c := canalDEffets(naissance38(1478, 1, k.ne))
			ls := fin(c, ts)
			got := len(ls) == 2 && ls[1].Kind == CarriedEffectBirth
			if got != k.retenu {
				t.Fatalf("retenue %v, attendu %v : %v", got, k.retenu, genres(ls))
			}
		})
	}
}

// TestCandidatesALEffetExclusions : une vie nommee par une pile n est pas candidate (un autre archetype n entre jamais
// dans les creations retenues : cf. TestLireLesEffetsNaissance).
func TestCandidatesALEffetExclusions(t *testing.T) {
	const ts = uint64(10_000_000)
	nommee := naissance38(1500, 1, ts-300_000)
	c := canalDEffets(nommee, naissance38(1502, 1, ts-300_000))
	c.nommes[nommee.obj.Life] = []uint64{nommee.obj.TimestampUS + 50_000}
	ls := fin(c, ts)
	if len(ls) != 2 || ls[1].Spawned[0].Life.Slot != 1502 {
		t.Fatalf("seule la vie 1502 est candidate : %v", genres(ls))
	}
}

// TestCandidatesALEffetLecture : la lecture nomme l objet porte, son porteur et la vie suivie supprimee, porte la
// creation de la candidate, dans l ordre de naissance, et garde ses quanta (par vie ET instant) pour la position.
func TestCandidatesALEffetLecture(t *testing.T) {
	const ts = uint64(10_000_000)
	c := canalDEffets(naissance38(1480, 1, ts-300_000), naissance38(1479, 1, ts-400_000))
	ls := fin(c, ts)
	if len(ls) != 3 || ls[0].Kind != types.CarriedTrackedEnd {
		t.Fatalf("lectures %v", genres(ls))
	}
	a, b := ls[1], ls[2]
	if a.Spawned[0].Life.Slot != 1479 || b.Spawned[0].Life.Slot != 1480 {
		t.Fatalf("ordre de naissance : %d puis %d", a.Spawned[0].Life.Slot, b.Spawned[0].Life.Slot)
	}
	if a.Object != (types.LifeKey{Slot: 2039, Gen: 1}) || a.Carrier != 539 || a.TimestampUS != ts ||
		len(a.Tracked) != 1 || a.Tracked[0] != (types.LifeKey{Slot: 1123, Gen: 1}) {
		t.Fatalf("lecture %+v", a)
	}
	if q := c.lu.quantaEffets[naissanceCle{a.Spawned[0].Life, a.Spawned[0].TimestampUS}]; q != [3]uint64{7, 8, 9} {
		t.Fatalf("quanta %v", q)
	}
}

// TestFinDEffetRendue : la suppression d une candidate retenue est rendue avec l objet porte ; celle d une vie qui
// ne l est pas, ou d une generation recyclee, ne l est pas ; elle n est rendue qu une fois.
func TestFinDEffetRendue(t *testing.T) {
	const ts = uint64(10_000_000)
	c := canalDEffets(naissance38(1478, 1, ts-300_000), naissance38(1600, 1, ts-5_000_000))
	fin(c, ts)
	del := func(at uint64, slot, gen uint32) int {
		n := len(c.lu.lectures)
		c.lireLesEffets(&lecture.Paquet{TS: at, Records: []lecture.Record{{Genre: lecture.GenreSuppression,
			Vie: types.LifeKey{Slot: slot, Gen: gen}}}})
		return len(c.lu.lectures) - n
	}
	if del(ts+7_000_000, 1600, 1) != 0 {
		t.Fatal("une vie hors fenetre n est pas retenue : sa suppression ne se rend pas")
	}
	if del(ts+7_000_000, 1478, 2) != 0 {
		t.Fatal("generation recyclee : la suppression d une autre vie du slot ne se rend pas")
	}
	if del(ts+7_507_000, 1478, 1) != 1 {
		t.Fatal("la suppression de la candidate se rend")
	}
	l := c.lu.lectures[len(c.lu.lectures)-1]
	if l.Kind != CarriedEffectEnd || l.TimestampUS != ts+7_507_000 || l.Object.Slot != 2039 || l.Carrier != 539 ||
		l.Tracked[0] != (types.LifeKey{Slot: 1478, Gen: 1}) {
		t.Fatalf("lecture %+v", l)
	}
	if del(ts+8_000_000, 1478, 1) != 0 {
		t.Fatal("la suppression ne se rend qu une fois")
	}
}

// TestCandidatesALEffetCleRecyclee : la cle d une candidate a pu etre nommee par une pile il y a longtemps (la generation
// ne tient que sur 2 bits) : seul un nommage a moins de 2 s de sa creation l exclut, bornes comprises.
func TestCandidatesALEffetCleRecyclee(t *testing.T) {
	const ts = uint64(100_000_000)
	ne := ts - 300_000
	for _, k := range []struct {
		nom     string
		nomme   uint64
		retenue bool
	}{
		{"nommee 30 s plus tot", ne - 30_000_000, true},
		{"nommee 2 s plus tot", ne - 2_000_000, false},
		{"nommee 2 s + 1 us plus tot", ne - 2_000_001, true},
		{"nommee 2 s plus tard", ne + 2_000_000, false},
		{"nommee 2 s + 1 us plus tard", ne + 2_000_001, true},
	} {
		t.Run(k.nom, func(t *testing.T) {
			c := canalDEffets(naissance38(1478, 1, ne))
			c.nommes[types.LifeKey{Slot: 1478, Gen: 1}] = []uint64{k.nomme}
			ls := fin(c, ts)
			if got := len(ls) == 2; got != k.retenue {
				t.Fatalf("retenue %v, attendu %v", got, k.retenue)
			}
		})
	}
}

// TestLireLesEffetsNaissance : une creation ti=38 de la trame de lecture est retenue sous son slot, a l instant du
// paquet, avec sa generation ; une creation d un autre archetype ne l est pas. Sans composant de position lu, la
// candidate n a pas de quanta.
func TestLireLesEffetsNaissance(t *testing.T) {
	c := canalDEffets()
	c.lireLesEffets(&lecture.Paquet{TS: 9_000_000, Records: []lecture.Record{
		{Genre: lecture.GenreNeuf, TI: 38, Vie: types.LifeKey{Slot: 1478, Gen: 2}},
		{Genre: lecture.GenreNeuf, TI: 41, Vie: types.LifeKey{Slot: 1479, Gen: 1}},
		{Genre: lecture.GenreDelta, TI: 38, Vie: types.LifeKey{Slot: 1480, Gen: 1}}}})
	n, ok := c.nes38[1478]
	if !ok || n.obj.Life.Gen != 2 || n.obj.TimestampUS != 9_000_000 || n.obj.TypeIndex != archetypeDeLEffet || n.aQ {
		t.Fatalf("creation %+v (%v)", n, ok)
	}
	if len(c.nes38) != 1 {
		t.Fatalf("seule la creation ti=38 est retenue : %v", c.nes38)
	}
}

// TestCandidateRetenueParDeuxObjetsGardeLaPremiere : une candidate dans la fenetre de DEUX objets suivis supprimes l un
// apres l autre est attribuee au premier qui la retient ; le second ne la reprend pas, et sa suppression est rendue
// une fois, sous le premier objet porte.
func TestCandidateRetenueParDeuxObjetsGardeLaPremiere(t *testing.T) {
	const ts = uint64(10_000_000)
	c := canalDEffets(naissance38(1478, 1, ts-300_000))
	c.suivis[1130] = objetSuivi{gen: 1, objet: types.LifeKey{Slot: 2050, Gen: 1}, porteur: 540}
	fin(c, ts) // la grenade 1123, objet 2039
	c.suppression(ts+100_000, &FrameRecord{Type: recDel, ID: 1<<30 | 1130, Slot: 1130})
	var naissances int
	for _, l := range c.lu.lectures {
		if l.Kind == CarriedEffectBirth {
			naissances++
		}
	}
	if naissances != 2 {
		t.Fatalf("les deux objets suivis retiennent la candidate : %d lectures de naissance", naissances)
	}
	if e := c.effets[1478]; e.objet.Slot != 2039 || e.porteur != 539 {
		t.Fatalf("la premiere attribution est gardee, pas reprise par le second : %+v", e)
	}
	c.lireLesEffets(&lecture.Paquet{TS: ts + 7_500_000, Records: []lecture.Record{{Genre: lecture.GenreSuppression,
		Vie: types.LifeKey{Slot: 1478, Gen: 1}}}})
	l := c.lu.lectures[len(c.lu.lectures)-1]
	if l.Kind != CarriedEffectEnd || l.Object.Slot != 2039 || l.Carrier != 539 {
		t.Fatalf("fin d effet sous le premier objet porte : %+v", l)
	}
}

// bitsDePayload empaquette des champs (valeur, largeur) MSB d abord, comme le flux du film.
func bitsDePayload(champs ...[2]uint64) []byte {
	var out []byte
	pos := 0
	for _, c := range champs {
		for i := int(c[1]) - 1; i >= 0; i-- {
			if pos%8 == 0 {
				out = append(out, 0)
			}
			if c[0]>>uint(i)&1 == 1 {
				out[len(out)-1] |= 0x80 >> uint(pos%8)
			}
			pos++
		}
	}
	return out
}

func descripteurDeTest() profile.PrecisionDescriptor {
	return profile.PrecisionDescriptor{IndexW: 3, Region: 5, AxisW: [3]uint{8, 8, 8}}
}

// TestLireLesEffetsQuanta : la position de la candidate est celle du composant d INDEX 0 (3 bits de remplissage et un
// composant d un autre index la precedent), lue sur le chemin dominant de la region de la carte ; un composant vide ou
// une autre region ne donne aucun quanta.
func TestLireLesEffetsQuanta(t *testing.T) {
	pay := bitsDePayload([2]uint64{5, 3}, [2]uint64{0, 2}, [2]uint64{5, 3}, [2]uint64{10, 8}, [2]uint64{20, 8}, [2]uint64{30, 8})
	paquet := func(comp lecture.Composant, payload []byte) *lecture.Paquet {
		return &lecture.Paquet{TS: 9_000_000, Payload: payload,
			Comps:   []lecture.Composant{{Index: 3, Debut: 0, Bits: 3}, comp},
			Records: []lecture.Record{{Genre: lecture.GenreNeuf, TI: 38, Vie: types.LifeKey{Slot: 1478, Gen: 1}, Comps: [2]uint32{0, 2}}}}
	}
	cas := []struct {
		nom  string
		comp lecture.Composant
		pay  []byte
		aQ   bool
	}{
		{"composant de position", lecture.Composant{Index: 0, Debut: 3, Bits: 29}, pay, true},
		{"composant de position vide", lecture.Composant{Index: 0, Debut: 3, Bits: 0}, pay, false},
		{"autre composant que la position", lecture.Composant{Index: 2, Debut: 3, Bits: 29}, pay, false},
		{"autre region de la carte", lecture.Composant{Index: 0, Debut: 3, Bits: 29},
			bitsDePayload([2]uint64{5, 3}, [2]uint64{0, 2}, [2]uint64{4, 3}, [2]uint64{10, 8}, [2]uint64{20, 8}, [2]uint64{30, 8}), false},
	}
	for _, k := range cas {
		t.Run(k.nom, func(t *testing.T) {
			c := canalDEffets()
			c.lu.lg = descripteurDeTest()
			c.lireLesEffets(paquet(k.comp, k.pay))
			n := c.nes38[1478]
			if n.aQ != k.aQ || (k.aQ && n.q != [3]uint64{10, 20, 30}) {
				t.Fatalf("quanta %v (%v), attendu %v", n.q, n.aQ, k.aQ)
			}
		})
	}
}

// TestScanEquipementPorteBrancheDeLEffet : la position d une candidate se lit par sa vie ET son instant de creation (la
// cle revient dans le match : deux candidates de meme cle ont deux positions, et les quanta d une VIE NOMMEE de meme
// cle ne sont jamais les siens) ; une vie nommee garde sa table ; et un effet ne s ajoute pas au compte des vies nommees
// positionnees, dont le denominateur est celui des vies nommees.
func TestScanEquipementPorteBrancheDeLEffet(t *testing.T) {
	vie := types.LifeKey{Slot: 1478, Gen: 1}
	nommee := types.LifeKey{Slot: 1123, Gen: 1}
	effet := func(ts uint64) types.CarriedEquipmentRead {
		return types.CarriedEquipmentRead{TimestampUS: ts + 100_000, Kind: CarriedEffectBirth,
			Tracked: []types.LifeKey{nommee}, Spawned: []types.SpawnedObject{{Life: vie, TimestampUS: ts, TypeIndex: 38}}}
	}
	fc := &FilmContext{}
	fc.recup.equipementPorte = &equipementPorteLu{
		lectures: []types.CarriedEquipmentRead{
			{TimestampUS: 1_000_000, Kind: types.CarriedDelta, Spawned: []types.SpawnedObject{{Life: nommee, TimestampUS: 900_000, TypeIndex: 41}}},
			effet(5_000_000), effet(60_000_000)},
		st:           types.CarriedEquipmentStats{SpawnedBirths: 1},
		quanta:       map[types.LifeKey][3]uint64{vie: {200, 201, 202}, nommee: {1, 2, 3}},
		quantaEffets: map[naissanceCle][3]uint64{{vie, 5_000_000}: {10, 20, 30}, {vie, 60_000_000}: {100, 110, 120}},
		lg:           descripteurDeTest(),
	}
	wr := profile.Vec3Range{{Min: 0, Max: 256}, {Min: 0, Max: 256}, {Min: 0, Max: 256}}
	ls, st, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	pos := func(l types.CarriedEquipmentRead) [3]float32 {
		n := l.Spawned[0]
		if !n.HasPosition {
			t.Fatalf("position absente : %+v", l)
		}
		return [3]float32{n.X, n.Y, n.Z}
	}
	if got := pos(ls[0]); got != [3]float32{1.5, 2.5, 3.5} {
		t.Fatalf("vie nommee : %v", got)
	}
	if got := pos(ls[1]); got != [3]float32{10.5, 20.5, 30.5} {
		t.Fatalf("premiere candidate de la cle : %v (les quanta de la vie nommee ou d une autre naissance ne sont pas les siens)", got)
	}
	if got := pos(ls[2]); got != [3]float32{100.5, 110.5, 120.5} {
		t.Fatalf("seconde candidate de la meme cle : %v", got)
	}
	if st.SpawnedPositioned != 1 || st.SpawnedBirths != 1 {
		t.Fatalf("seule la vie nommee se compte : %+v", st)
	}
	// Sans bornes de carte, rien n est positionne.
	if ls, _, _ = ScanEquipementPorte(fc, nil); ls[1].Spawned[0].HasPosition {
		t.Fatal("sans bornes, aucune position")
	}
}
