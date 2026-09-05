package filmdec

// vehicules_v6_ref2_test.go — INSTRUMENT (lot V6) : LA REFERENCE 2 DE L'EMBARQUEMENT EST-ELLE
// LE VEHICULE ?
//
// D'OU VIENT LA QUESTION. `TestV6Longueur` mesure que le debut de trame ECS d'un paquet a tete
// EMBARQUEMENT vaut 53 bits sur 7/7 instances, et 43 sur 38/38 pour la SORTIE. Or
//
//	embarquement : 9 (config+cont+type) + refs + 6 (siege) + 1 (fin de liste) = 53 -> refs = 37
//	sortie       : 9                    + refs + 6         + 1                = 43 -> refs = 27
//
// et 37 = 11 + 10 + 16, c'est-a-dire les TROIS references PRESENTES aux largeurs deja etablies
// (domaine 2 = 1+8+2, domaine 3 = 1+7+2, domaine 7 = 1+13+2). Pour la sortie, 27 = 13 + 13 + 1 :
// deux refs de domaine 1 avec sonde, et la TROISIEME ABSENTE — exactement ce que le rapport
// V3 avait mesure (« la ref 2 est gardee-absente en pratique »).
//
// AUTREMENT DIT : la ref 2 de l'EMBARQUEMENT, elle, EST PRESENTE, et son domaine (7) est celui
// des objets du monde. C'est le candidat direct pour LE VEHICULE — ce que le chantier resolvait
// jusqu'ici par la position.
//
// LE TEMOIN EST DOUBLE : (a) la meme reference lue avec un decalage d'un bit ; (b) la bande de
// slots d'un AUTRE archetype (les armes au sol, ti=42) au lieu des vehicules.
//
// Garde d'environnement V6_ROOT / V6_FILMS : sans elle, tout SKIP.

import (
	"path/filepath"
	"testing"
)

// v6BoardRefs rend les trois references d'un embarquement de tete, plus le bit de siege.
func v6BoardRefs(pay []byte) (r0, r1, r2 guardedRef, seatBit int) {
	r0, r1, r2 = boardRefs(pay)
	return r0, r1, r2, r2.EndBit
}

// v6Ref2Stats : ce que la passe releve.
type v6Ref2Stats struct {
	boards                       int
	r0Present, r1Present         int
	r2Present                    int
	r2InVehicleBand              int
	r2InWeaponBand               int
	r2Shift1InVehicleBand        int
	occInBand                    int
	seatZero                     int
	endAt53                      int
	r1Values, r2Values, seatVals map[int]int
}

func newV6Ref2() *v6Ref2Stats {
	return &v6Ref2Stats{r1Values: map[int]int{}, r2Values: map[int]int{}, seatVals: map[int]int{}}
}

// TestV6Ref2 : la ref 2 de l'embarquement tombe-t-elle sur un slot de vehicule ?
func TestV6Ref2(t *testing.T) {
	dirs := v6FilmDirs(t)
	s := newV6Ref2()
	filmsWithBoard := 0
	for _, dir := range dirs {
		if s.scanFilm(dir) {
			filmsWithBoard++
		}
	}
	if s.boards == 0 {
		t.Skip("aucun embarquement dans le corpus fourni")
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(s.boards) }
	t.Logf("== V6 REF2 — %d embarquements de tete sur %d films ==", s.boards, filmsWithBoard)
	t.Logf("refs presentes : r0 %d (%.1f %%) · r1 %d (%.1f %%) · r2 %d (%.1f %%)",
		s.r0Present, pct(s.r0Present), s.r1Present, pct(s.r1Present), s.r2Present, pct(s.r2Present))
	t.Logf("fin d'evenement au bit 52 (trame a 53) : %d (%.1f %%)", s.endAt53, pct(s.endAt53))
	t.Logf("occupant (r0) en bande bipede : %d (%.1f %%) · siege = 0 : %d (%.1f %%)",
		s.occInBand, pct(s.occInBand), s.seatZero, pct(s.seatZero))
	t.Logf("REF2 dans la bande VEHICULE (ti=%d) : %d (%.1f %%)",
		VehicleTypeIndex, s.r2InVehicleBand, pct(s.r2InVehicleBand))
	t.Logf("  TEMOIN a) meme ref lue a +1 bit, bande vehicule : %d (%.1f %%)",
		s.r2Shift1InVehicleBand, pct(s.r2Shift1InVehicleBand))
	t.Logf("  TEMOIN b) ref2 dans la bande ARMES AU SOL (ti=%d) : %d (%.1f %%)",
		GroundWeaponTypeIndex, s.r2InWeaponBand, pct(s.r2InWeaponBand))
	t.Logf("valeurs r1 (domaine 3) :%s", v6TopHist(s.r1Values, 10))
	t.Logf("valeurs r2 (domaine 7) :%s", v6TopHist(s.r2Values, 12))
	t.Logf("sieges                 :%s", v6TopHist(s.seatVals, 8))
}

// scanFilm releve les embarquements d'un film et confronte leur ref 2 aux bandes d'archetype.
func (s *v6Ref2Stats) scanFilm(dir string) bool {
	n := CountFilmChunks(dir)
	if n == 0 {
		return false
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	band := bipedSlotBandDir(dir, chunks)
	base := uint32(0)
	if slots := band.Slots(); len(slots) > 0 {
		base = slots[0]
	}
	vehBand := worldObjectSlotBandDir(dir, n, VehicleTypeIndex)
	wpnBand := worldObjectSlotBandDir(dir, n, GroundWeaponTypeIndex)
	found := false
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 8 {
				continue
			}
			pay := p.Payload(data)
			if ty, ok := PacketHeadEventType(pay); !ok || ty != EventBipedBoardVehicle {
				continue
			}
			s.sample(pay, base, band, vehBand, wpnBand)
			found = true
		}
	}
	_ = filepath.Base(dir)
	return found
}

// sample releve un embarquement.
func (s *v6Ref2Stats) sample(pay []byte, base uint32, band SlotBand, vehBand, wpnBand map[uint32]bool) {
	r0, r1, r2, seatBit := v6BoardRefs(pay)
	s.boards++
	if r0.Present {
		s.r0Present++
		if band.Has(base + r0.Index) {
			s.occInBand++
		}
	}
	if r1.Present {
		s.r1Present++
		s.r1Values[int(r1.Index)]++
	}
	if r2.Present {
		s.r2Present++
		s.r2Values[int(r2.Index)]++
		if vehBand[r2.Index] {
			s.r2InVehicleBand++
		}
		if wpnBand[r2.Index] {
			s.r2InWeaponBand++
		}
	}
	if seatBit+vehicleSeatBits <= len(pay)*8 {
		seat := int(readBitsAt(pay, seatBit, vehicleSeatBits))
		s.seatVals[seat]++
		if seat == 0 {
			s.seatZero++
		}
		if seatBit+vehicleSeatBits == 52 {
			s.endAt53++
		}
	}
	// TEMOIN a : la meme reference lue un bit plus loin.
	if r1.Present || r0.Present {
		shifted := readPlainRef(pay, r1.EndBit+1, dom7RefWidth)
		if shifted.Present && vehBand[shifted.Index] {
			s.r2Shift1InVehicleBand++
		}
	}
}
