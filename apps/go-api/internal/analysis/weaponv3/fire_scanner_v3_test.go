package weaponv3

// fire_scanner_v3_test.go — tests fixture (sans cache film) du scanner fire v3.
//
// Construit un bitstream synthétique portant UN fire-event au marqueur 11-bit, à
// pi/slot/weapon choisis, et vérifie : (1) chaque layout lit le pi attendu ;
// (2) FirePi5SpanBefore se RÉDUIT à FirePi4High quand le bit emprunté (avant b5)
// vaut 0 (= propriété qui rend le 5-bit sûr sur les matchs ≤16 joueurs) ;
// (3) FirePi5SpanBefore débloque un pi∈[16,31] que le 4-bit plafonne à 15 ;
// (4) le recall relâché (3-bit + canon) trouve l'event quand le 11-bit ne casse pas.

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

// fireBitWriter — petit écrivain de bits MSB-first pour fabriquer un fixture.
type fireBitWriter struct {
	bits []int
}

func (w *fireBitWriter) put(value uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, int((value>>uint(i))&1))
	}
}

func (w *fireBitWriter) pad(n int) {
	for i := 0; i < n; i++ {
		w.bits = append(w.bits, 0)
	}
}

// bytes assemble les bits en octets (complète le dernier octet par des zéros).
func (w *fireBitWriter) bytes() []byte {
	out := make([]byte, (len(w.bits)+7)/8)
	for i, b := range w.bits {
		if b != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}

// buildFireFixture fabrique un chunk avec UN fire-event : un préambule de
// padding pour que le marqueur ne tombe pas à un bord, le marqueur 11-bit, puis
// les champs (b5 à eventStart+32, weapon à eventStart+40). bitBeforeB5 = le bit
// emprunté par FirePi5SpanBefore (eventStart+31). weapon = id 64-bit complet.
func buildFireFixture(t *testing.T, prePadBits int, b5 byte, bitBeforeB5 int, weapon uint64) []byte {
	t.Helper()
	w := &fireBitWriter{}
	w.pad(prePadBits)                    // décalage (le scanner cherche à toute position)
	w.put(fireMarkerBits, fireMarkerLen) // marqueur 11-bit -> eventStart = bitPos+3
	// Après le marqueur (11 bits) le curseur est à eventStart+8. On remplit jusqu'à
	// eventStart+31 (23 bits) ; le bit emprunté par FirePi5SpanBefore occupe
	// eventStart+31, puis l'octet b5 à eventStart+32 et le weapon-id à eventStart+40.
	w.pad(23)                       // de eventStart+8 à eventStart+31 (exclu)
	w.put(uint64(bitBeforeB5&1), 1) // eventStart+31 = bit emprunté (avant b5)
	w.put(uint64(b5), 8)            // eventStart+32..39 = octet b5
	w.put(weapon, 64)               // eventStart+40..103 = weapon-id
	w.pad(64)                       // marge post-event (burst/hit + sécurité borne)
	return w.bytes()
}

const fireTestWeapon = uint64(0x2B1824D5)<<32 | 0x42c9679f // BR75 + suffixe commun

// estZero — estimateur trivial (timestamp constant), suffisant pour les tests.
func estZero(int) float64 { return 0 }

func TestScanFireEventsV3_PiLayouts(t *testing.T) {
	const pi, slot = 6, 1
	b5 := byte(pi<<4 | slot)

	cases := []struct {
		name    string
		layout  FirePiLayout
		bitPrev int
		wantPI  int
	}{
		{"4High pi=6", FirePi4High, 0, 6},
		{"5SpanBefore bitPrev=0 -> =4High", FirePi5SpanBefore, 0, 6},
		{"5SpanBefore bitPrev=1 -> +16", FirePi5SpanBefore, 1, 22},
		{"5HighInB5 = b5>>3", FirePi5HighInB5, 0, int(b5 >> 3)},
		{"5LowInB5 = b5&0x1f", FirePi5LowInB5, 0, int(b5 & 0x1f)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := buildFireFixture(t, 5, b5, c.bitPrev, fireTestWeapon)
			evs := ScanFireEventsV3(data, estZero, c.layout, false)
			if len(evs) == 0 {
				t.Fatalf("aucun fire-event trouvé (layout=%d)", c.layout)
			}
			// Le 1er event doit porter le pi attendu + l'arme + le slot.
			ev := evs[0]
			if ev.PlayerIndex != c.wantPI {
				t.Errorf("pi=%d, attendu %d", ev.PlayerIndex, c.wantPI)
			}
			if ev.Slot != slot {
				t.Errorf("slot=%d, attendu %d", ev.Slot, slot)
			}
			// L'arme reconstruite doit être le BR75 (high-32 0x2B1824D5).
			weaponInt := uint64(ev.WeaponBytes[0])<<56 | uint64(ev.WeaponBytes[1])<<48 |
				uint64(ev.WeaponBytes[2])<<40 | uint64(ev.WeaponBytes[3])<<32 |
				uint64(ev.WeaponBytes[4])<<24 | uint64(ev.WeaponBytes[5])<<16 |
				uint64(ev.WeaponBytes[6])<<8 | uint64(ev.WeaponBytes[7])
			if high, known := CanonWeaponID(weaponInt); !known || high != 0x2B1824D5 {
				t.Errorf("arme high-32 inattendue : high=0x%08x known=%v", high, known)
			}
		})
	}
}

// TestScanFireEventsV3_SpanBeforeReducesToV2 vérifie la propriété de sûreté : sur
// un pi≤15 avec bit emprunté=0, FirePi5SpanBefore lit le MÊME pi que FirePi4High
// (= pourquoi le 5-bit ne régresse pas les petits matchs où bit-31=0 pour le pi).
func TestScanFireEventsV3_SpanBeforeReducesToV2(t *testing.T) {
	for pi := 0; pi <= 15; pi++ {
		b5 := byte(pi << 4) // slot=0
		data := buildFireFixture(t, 3, b5, 0, fireTestWeapon)
		v4 := ScanFireEventsV3(data, estZero, FirePi4High, false)
		v5 := ScanFireEventsV3(data, estZero, FirePi5SpanBefore, false)
		if len(v4) == 0 || len(v5) == 0 {
			t.Fatalf("pi=%d : event manquant (v4=%d v5=%d)", pi, len(v4), len(v5))
		}
		if v4[0].PlayerIndex != pi || v5[0].PlayerIndex != pi {
			t.Errorf("pi=%d : v4=%d v5=%d (attendu %d/%d)", pi, v4[0].PlayerIndex, v5[0].PlayerIndex, pi, pi)
		}
	}
}

// TestScanFireEventsV3_MatchesB5OnV2Layout vérifie que ScanFireEventsV3 en
// FirePi4High produit le MÊME nombre d'events + mêmes pi que analysis.ScanFireEventsB5
// (le port v3 ne change QUE la lecture du pi ; en 4-bit il est identique à la v2).
func TestScanFireEventsV3_MatchesB5OnV2Layout(t *testing.T) {
	data := buildFireFixture(t, 7, byte(9<<4|2), 0, fireTestWeapon)
	v2 := analysis.ScanFireEventsB5(data, estZero)
	v3 := ScanFireEventsV3(data, estZero, FirePi4High, false)
	if len(v2) != len(v3) {
		t.Fatalf("cardinalité v2=%d v3=%d", len(v2), len(v3))
	}
	for i := range v2 {
		if v2[i].PlayerIndex != v3[i].PlayerIndex || v2[i].Slot != v3[i].Slot ||
			v2[i].WeaponBytes != v3[i].WeaponBytes {
			t.Errorf("event %d : v2(pi=%d slot=%d) != v3(pi=%d slot=%d)",
				i, v2[i].PlayerIndex, v2[i].Slot, v3[i].PlayerIndex, v3[i].Slot)
		}
	}
}

// TestScanFireEventsV3_Relax3FindsEvent vérifie que le recall relâché (3-bit +
// canon high-32) retrouve le fire-event du fixture. Le marqueur 11-bit commence
// par "101" donc le 3-bit le matche aussi ; le canon valide l'arme BR75.
func TestScanFireEventsV3_Relax3FindsEvent(t *testing.T) {
	data := buildFireFixture(t, 5, byte(3<<4), 0, fireTestWeapon)
	evs := ScanFireEventsV3(data, estZero, FirePi4High, true)
	if len(evs) == 0 {
		t.Fatal("recall relâché : aucun fire-event trouvé")
	}
	found := false
	for _, ev := range evs {
		if ev.PlayerIndex == 3 {
			found = true
		}
	}
	if !found {
		t.Errorf("recall relâché : pi=3 attendu absent (events=%d)", len(evs))
	}
}
