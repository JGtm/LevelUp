package filmdec

// vehicules_v6_chaine_test.go — INSTRUMENT (lot V6) : la GRAMMAIRE D'ENCHAINEMENT de la liste
// d'evenements. Sous garde d'environnement (V6_ROOT / V6_FILMS) : sans elle, tout SKIP.
//
// LA QUESTION. Le modele de paquet est
//
//	[1 bit config] [ ( 1 [R(7) type] [3 refs gardees] [charge] )* 0 ] [trame ECS]
//
// Le decodeur actuel ne lit que l'evenement de TETE. Ce fichier MESURE ce qui suit un
// evenement dont la longueur est CONNUE au bit pres — l'embarquement (type 8) et la sortie
// (type 22), dont la charge est `R(6) siege`. Le bit qui suit le siege doit etre le bit de
// CONTINUATION du deuxieme evenement de la liste.
//
// LE TEMOIN EST INTEGRE : le meme releve est fait a -1 et +1 bit du bit de continuation
// suppose. Si le cadrage est bon, l'histogramme des types au bon decalage doit ressembler a
// l'histogramme des types de TETE (memes familles), et les deux decalages voisins doivent
// rendre du bruit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// v6Root / v6Films : garde d'environnement.
func v6Root(t *testing.T) string {
	t.Helper()
	d := os.Getenv("V6_ROOT")
	if d == "" {
		t.Skip("V6_ROOT non defini")
	}
	return d
}

func v6FilmDirs(t *testing.T) []string {
	t.Helper()
	root := v6Root(t)
	if l := os.Getenv("V6_FILMS"); l != "" {
		var out []string
		for _, s := range splitComma(l) {
			out = append(out, filepath.Join(root, s))
		}
		return out
	}
	return evtListFilmDirs(root)
}

// v6EventEnd rend le bit qui suit IMMEDIATEMENT la charge d'un evenement vehicule dont la
// tete commence au bit `startBit` (le bit de continuation), et ok=false si le type n'est pas
// un evenement vehicule. C'est la SEULE longueur d'evenement connue au bit pres.
func v6EventEnd(pay []byte, startBit, typ int) (int, bool) {
	body := startBit + 1 + eventTypeBits
	var seatBit int
	switch typ {
	case EventUnitExitVehicle:
		r0 := readDom1Ref(pay, body)
		r1 := readDom1Ref(pay, r0.EndBit)
		seatBit = readPlainRef(pay, r1.EndBit, dom7RefWidth).EndBit
	case EventBipedBoardVehicle:
		r0 := readPlainRef(pay, body, dom2RefWidth)
		r1 := readPlainRef(pay, r0.EndBit, dom3RefWidth)
		seatBit = readPlainRef(pay, r1.EndBit, dom7RefWidth).EndBit
	default:
		return 0, false
	}
	end := seatBit + vehicleSeatBits
	if end+1+eventTypeBits > len(pay)*8 {
		return end, false
	}
	return end, true
}

// v6Chain : ce qu'une passe releve.
type v6Chain struct {
	heads          int            // paquets a tete board/exit exploitables
	contSet        int            // bit de continuation a 1 apres la charge
	typesAt        map[int]int    // histogramme du R(7) au bon decalage
	typesMinus     map[int]int    // temoin : decalage -1
	typesPlus      map[int]int    // temoin : decalage +1
	contSetMinus   int            // temoin : bit de continuation a -1
	contSetPlus    int            // temoin : bit de continuation a +1
	tailBits       map[int]int    // distribution de (bits restants apres la charge)
	headHist       map[int]int    // histogramme des types de TETE (reference de forme)
	secondVehicule int            // deuxieme evenement de type 8 ou 22
	secondByType   map[int]int    // detail des deuxiemes evenements vehicule
	filmsSeen      map[string]int // films visites
}

func newV6Chain() *v6Chain {
	return &v6Chain{
		typesAt: map[int]int{}, typesMinus: map[int]int{}, typesPlus: map[int]int{},
		tailBits: map[int]int{}, headHist: map[int]int{}, secondByType: map[int]int{},
		filmsSeen: map[string]int{},
	}
}

// v6ScanFilm remplit le releve pour un film.
func (m *v6Chain) scanFilm(dir string) {
	n := CountFilmChunks(dir)
	id := filepath.Base(filepath.Clean(dir))
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta || p.Size < 2 {
				continue
			}
			pay := p.Payload(chunk)
			typ, present := PacketHeadEventType(pay)
			if !present {
				continue
			}
			m.headHist[typ]++
			if typ != EventBipedBoardVehicle && typ != EventUnitExitVehicle {
				continue
			}
			end, ok := v6EventEnd(pay, 1, typ)
			if !ok {
				continue
			}
			m.heads++
			m.filmsSeen[id]++
			m.tailBits[len(pay)*8-end]++
			m.sample(pay, end)
		}
	}
}

// sample releve le bit de continuation et le type au bon decalage et aux deux temoins.
func (m *v6Chain) sample(pay []byte, end int) {
	rd := func(at int) (int, int) {
		return int(readBitsAt(pay, at, 1)), int(readBitsAt(pay, at+1, eventTypeBits))
	}
	cont, typ2 := rd(end)
	if cont == 1 {
		m.contSet++
		m.typesAt[typ2]++
		if typ2 == EventBipedBoardVehicle || typ2 == EventUnitExitVehicle {
			m.secondVehicule++
			m.secondByType[typ2]++
		}
	}
	if end >= 1 {
		if cm, tm := rd(end - 1); cm == 1 {
			m.contSetMinus++
			m.typesMinus[tm]++
		}
	}
	if cp, tp := rd(end + 1); cp == 1 {
		m.contSetPlus++
		m.typesPlus[tp]++
	}
}

// v6TopHist rend les n premieres entrees d'un histogramme, triees par effectif.
func v6TopHist(h map[int]int, n int) string {
	type kv struct{ k, v int }
	var all []kv
	for k, v := range h {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].k < all[j].k
	})
	s := ""
	for i := 0; i < len(all) && i < n; i++ {
		s += fmt.Sprintf(" %d×%d", all[i].k, all[i].v)
	}
	return s
}

// v6Concentration rend la part des 8 types les plus frequents (mesure de non-uniformite).
func v6Concentration(h map[int]int) (float64, int) {
	total, distinct := 0, len(h)
	type kv struct{ k, v int }
	var all []kv
	for k, v := range h {
		all = append(all, kv{k, v})
		total += v
	}
	if total == 0 {
		return 0, 0
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	top := 0
	for i := 0; i < len(all) && i < 8; i++ {
		top += all[i].v
	}
	return 100 * float64(top) / float64(total), distinct
}

// TestV6Chaine : le bit qui suit la charge d'un board/exit est-il un bit de continuation ?
func TestV6Chaine(t *testing.T) {
	dirs := v6FilmDirs(t)
	m := newV6Chain()
	for _, d := range dirs {
		m.scanFilm(d)
	}
	if m.heads == 0 {
		t.Skip("aucun evenement vehicule de tete dans le corpus fourni")
	}
	t.Logf("== V6 CHAINAGE — %d films demandes, %d films porteurs, %d tetes board/exit ==",
		len(dirs), len(m.filmsSeen), m.heads)
	pct := func(n int) float64 { return 100 * float64(n) / float64(m.heads) }
	t.Logf("continuation apres la charge : %d (%.1f %%) — temoin -1 : %d (%.1f %%) · temoin +1 : %d (%.1f %%)",
		m.contSet, pct(m.contSet), m.contSetMinus, pct(m.contSetMinus), m.contSetPlus, pct(m.contSetPlus))

	cAt, dAt := v6Concentration(m.typesAt)
	cM, dM := v6Concentration(m.typesMinus)
	cP, dP := v6Concentration(m.typesPlus)
	cH, dH := v6Concentration(m.headHist)
	t.Logf("concentration top8 — TETE %.1f %% (%d types) · AU BON DECALAGE %.1f %% (%d types) · "+
		"temoin -1 %.1f %% (%d) · temoin +1 %.1f %% (%d)", cH, dH, cAt, dAt, cM, dM, cP, dP)
	t.Logf("types de TETE (TOUS, registre du lot) :%s", v6TopHist(m.headHist, 200))
	t.Logf("types 2e evenement     :%s", v6TopHist(m.typesAt, 12))
	t.Logf("temoin -1 bit          :%s", v6TopHist(m.typesMinus, 12))
	t.Logf("temoin +1 bit          :%s", v6TopHist(m.typesPlus, 12))
	t.Logf("2e evenement VEHICULE  : %d (dont%s)", m.secondVehicule, v6TopHist(m.secondByType, 4))
	t.Logf("bits restants apres la charge (top) :%s", v6TopHist(m.tailBits, 16))
}
