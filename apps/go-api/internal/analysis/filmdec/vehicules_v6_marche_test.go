package filmdec

// vehicules_v6_marche_test.go — INSTRUMENT (lot V6) : LA MARCHE DE LA LISTE, PAR LA SIGNATURE.
//
// POURQUOI CETTE FORME-LA, ET PAS LA MARCHE SEQUENTIELLE. Marcher la liste evenement par
// evenement suppose de connaitre la LONGUEUR DE CHARGE de chaque type rencontre. Deux mesures du
// lot ferment cette porte :
//
//	(a) le score d'ancrage aval (profondeur de marche ECS) ne designe le VRAI debut de trame que
//	    dans 23,3 % des cas par paquet — insuffisant pour deduire une longueur ;
//	(b) le seul type dont la longueur est connue au bit pres (board/exit) montre que la liste se
//	    TERMINE juste apres lui : bit de continuation a 0 sur 99 % des 100 tetes mesurees.
//
// (b) est ce qui rend la signature exploitable : QUAND un evenement vehicule est dans la liste,
// IL EN EST LE DERNIER. Sa fin est donc suivie du bit de fin de liste, et son cadrage complet est
// contraint :
//
//	board : [cont=1][R(7)=8]  [g=1][8][2] [g=1][7][2] [g=1][13][2] [R(6) siege] [fin=0]
//	exit  : [cont=1][R(7)=22] [g=1][s=1][9][2] [g=1][s=1][9][2] [g=0] [R(6) siege] [fin=0]
//
// soit 18 bits (board) et 20 bits (exit) CONTRAINTS a une valeur precise, avant meme de compter
// l'appartenance de l'occupant a la bande bipede. Un balayage de 400 decalages sur 40 000 paquets
// fait 1,6 x 10^7 essais : le plancher de faux positifs attendu est de l'ordre de la dizaine, et
// il est MESURE par le temoin.
//
// LE TEMOIN EST LE DECALAGE D'UN BIT : le meme balayage, corps de l'evenement lu un bit plus
// loin. Il doit s'effondrer.
//
// Garde d'environnement V6_ROOT / V6_FILMS : sans elle, tout SKIP.

import (
	"testing"
)

// v6ScanMaxBit borne le balayage : au-dela, on n'est plus dans la liste d'evenements mais dans
// la trame ECS. Le plus long evenement dont la longueur soit connue fait 52 bits ; le record de
// tir lit jusqu'au bit 143. 400 laisse la place a une liste de plusieurs evenements longs.
const v6ScanMaxBit = 400

// v6MaxEventBits borne la lecture d'un candidat : le plus long des deux evenements vehicule
// (embarquement) fait 1 + 7 + 11 + 10 + 16 + 6 + 1 = 52 bits. La marge protege `readBitsAt`,
// qui indexe SANS borne (piege documente dans fire_events.go).
const v6MaxEventBits = 64

// v6Candidate est un evenement vehicule reconnu par sa signature complete.
type v6Candidate struct {
	Bit          int
	Kind         int
	OccupantSlot uint32
	Seat         uint32
	Head         bool // le candidat est l'evenement de TETE (bit 1)
}

// v6TryVehicleEventAt teste la signature d'un evenement vehicule commencant au bit `b` (le bit de
// continuation). `shift` decale le CORPS de l'evenement (temoin : 1). Rend ok=false des qu'une
// contrainte tombe.
func v6TryVehicleEventAt(
	pay []byte, b, shift int, base uint32, band map[uint32]bool,
) (v6Candidate, bool) {
	total := len(pay) * 8
	if b < 1 || b+shift+v6MaxEventBits > total {
		return v6Candidate{}, false
	}
	if readBitsAt(pay, b, 1) != 1 {
		return v6Candidate{}, false
	}
	typ := int(readBitsAt(pay, b+1, eventTypeBits))
	body := b + 1 + eventTypeBits + shift
	var seatBit int
	var slot uint32
	switch typ {
	case EventUnitExitVehicle:
		r0 := readDom1Ref(pay, body)
		if !r0.Present || r0.Sonde != 1 {
			return v6Candidate{}, false
		}
		r1 := readDom1Ref(pay, r0.EndBit)
		if !r1.Present || r1.Sonde != 1 {
			return v6Candidate{}, false
		}
		r2 := readPlainRef(pay, r1.EndBit, dom7RefWidth)
		if r2.Present {
			return v6Candidate{}, false // mesure : la ref 2 de la sortie est gardee-absente 38/38
		}
		seatBit, slot = r2.EndBit, base+r0.Index
	case EventBipedBoardVehicle:
		r0 := readPlainRef(pay, body, dom2RefWidth)
		if !r0.Present {
			return v6Candidate{}, false
		}
		r1 := readPlainRef(pay, r0.EndBit, dom3RefWidth)
		if !r1.Present {
			return v6Candidate{}, false
		}
		r2 := readPlainRef(pay, r1.EndBit, dom7RefWidth)
		if !r2.Present {
			return v6Candidate{}, false // mesure : les trois refs de l'embarquement sont presentes 22/22
		}
		seatBit, slot = r2.EndBit, base+r0.Index
	default:
		return v6Candidate{}, false
	}
	if seatBit+vehicleSeatBits+1 > total {
		return v6Candidate{}, false
	}
	seat := readBitsAt(pay, seatBit, vehicleSeatBits)
	if seat >= 8 {
		return v6Candidate{}, false // mesure : 100 % des sieges attestes sont dans 0..7
	}
	if readBitsAt(pay, seatBit+vehicleSeatBits, 1) != 0 {
		return v6Candidate{}, false // l'evenement vehicule TERMINE la liste (mesure : 99 %)
	}
	if !band[slot] {
		return v6Candidate{}, false
	}
	return v6Candidate{Bit: b, Kind: typ, OccupantSlot: slot, Seat: seat, Head: b == 1}, true
}

// v6FilmID rend l'identifiant court d'un repertoire de film.
func v6FilmID(dir string) string {
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			return dir[i+1:]
		}
	}
	return dir
}

// v6MarcheStats : ce que la passe releve.
type v6MarcheStats struct {
	packets                           int
	headBoard, headExit               int
	deepBoard, deepExit               int
	deepSeat0                         int
	witness, witnessSeat0             []int
	deepBits                          map[int]int
	deepSeats                         map[int]int
	perFilmDeepBoard, perFilmDeepExit map[string]int
	perFilmHeadBoard, perFilmHeadExit map[string]int
	multiPerPacket                    map[int]int
}

func newV6Marche() *v6MarcheStats {
	return &v6MarcheStats{deepBits: map[int]int{}, deepSeats: map[int]int{},
		perFilmDeepBoard: map[string]int{}, perFilmDeepExit: map[string]int{},
		perFilmHeadBoard: map[string]int{}, perFilmHeadExit: map[string]int{},
		multiPerPacket: map[int]int{},
		witness:        make([]int, len(v6WitnessShifts)),
		witnessSeat0:   make([]int, len(v6WitnessShifts))}
}

// v6BandOf rend la bande bipede d'un film et sa base.
func v6BandOf(dir string, n int) (map[uint32]bool, uint32) {
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	band := bipedSlotBand(dir, chunks)
	base, first := uint32(0), true
	for s := range band {
		if first || s < base {
			base, first = s, false
		}
	}
	return band, base
}

// scanFilm balaie un film.
func (m *v6MarcheStats) scanFilm(dir, id string) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return
	}
	band, base := v6BandOf(dir, n)
	if len(band) == 0 {
		return
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 8 {
				continue
			}
			m.scanPacket(p.Payload(data), id, base, band)
		}
	}
}

// v6WitnessShifts : les decalages TEMOINS du corps de l'evenement. Zero est le reel.
var v6WitnessShifts = []int{1, 2, 3, 4}

// scanPacket balaie un payload et compte les candidats reels et temoins.
func (m *v6MarcheStats) scanPacket(pay []byte, id string, base uint32, band map[uint32]bool) {
	m.packets++
	hits := 0
	maxBit := v6ScanMaxBit
	if lim := len(pay)*8 - v6MaxEventBits - 1; lim < maxBit {
		maxBit = lim
	}
	for b := 1; b <= maxBit; b++ {
		if c, ok := v6TryVehicleEventAt(pay, b, 0, base, band); ok {
			m.tally(c, id)
			hits++
		}
		if b == 1 {
			continue // au bit 1, le temoin lit la charge d'un VRAI evenement : biais connu
		}
		for i, sh := range v6WitnessShifts {
			if c, ok := v6TryVehicleEventAt(pay, b, sh, base, band); ok {
				m.witness[i]++
				if c.Seat == 0 {
					m.witnessSeat0[i]++
				}
			}
		}
	}
	m.multiPerPacket[hits]++
}

// tally range un candidat.
func (m *v6MarcheStats) tally(c v6Candidate, id string) {
	switch {
	case c.Head && c.Kind == EventBipedBoardVehicle:
		m.headBoard++
		m.perFilmHeadBoard[id]++
	case c.Head:
		m.headExit++
		m.perFilmHeadExit[id]++
	case c.Kind == EventBipedBoardVehicle:
		m.deepBoard++
		m.perFilmDeepBoard[id]++
		m.deepBits[c.Bit]++
		m.deepSeats[int(c.Seat)]++
	default:
		m.deepExit++
		m.perFilmDeepExit[id]++
		m.deepBits[c.Bit]++
		m.deepSeats[int(c.Seat)]++
	}
	if !c.Head && c.Seat == 0 {
		m.deepSeat0++
	}
}

// TestV6Marche : y a-t-il des evenements vehicule AILLEURS qu'en tete de liste ?
func TestV6Marche(t *testing.T) {
	dirs := v6FilmDirs(t)
	m := newV6Marche()
	for _, d := range dirs {
		m.scanFilm(d, v6FilmID(d))
	}
	t.Logf("== V6 MARCHE — %d films, %d paquets delta balayes (decalages 1..%d) ==",
		len(dirs), m.packets, v6ScanMaxBit)
	t.Logf("TETE   : board %d · exit %d", m.headBoard, m.headExit)
	t.Logf("HORS TETE (signature complete) : board %d · exit %d", m.deepBoard, m.deepExit)
	t.Logf("HORS TETE, siege = 0 seulement       : %d", m.deepSeat0)
	for i, sh := range v6WitnessShifts {
		t.Logf("TEMOIN corps decale de +%d bit(s) : %d (dont siege 0 : %d)",
			sh, m.witness[i], m.witnessSeat0[i])
	}
	t.Logf("bits de depart des candidats hors tete :%s", v6TopHist(m.deepBits, 16))
	t.Logf("sieges des candidats hors tete        :%s", v6TopHist(m.deepSeats, 8))
	t.Logf("candidats par paquet                  :%s", v6TopIntHist(m.multiPerPacket, 8))
	for _, d := range dirs {
		id := v6FilmID(d)
		if m.perFilmHeadExit[id]+m.perFilmDeepExit[id]+m.perFilmHeadBoard[id]+m.perFilmDeepBoard[id] == 0 {
			continue
		}
		t.Logf("  %s : tete b=%d e=%d · hors tete b=%d e=%d", id,
			m.perFilmHeadBoard[id], m.perFilmHeadExit[id],
			m.perFilmDeepBoard[id], m.perFilmDeepExit[id])
	}
}
