//go:build research

package grammar

// avant_feuille4_54_research_test.go — LOT 5.4 : LA FEUILLE 4 DE L ETAT PAR DEFAUT DE `ti=40`.
//
// Extrait d `avant_chassis_54_research_test.go` le 2026-09-21 (seuil de 500 lignes). Ce n est PAS
// le chemin de production : la feuille 4 n existe qu aux IMAGES-CLES, porte `bVar14` posee, alors
// que le cap publie vient d `i2` a la cadence des deltas. Conserve parce qu il porte le negatif
// mesure (D4 (5.4)).

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------------------------
// LA FEUILLE 4 DE L ETAT PAR DEFAUT — lisible sur mini-bobines, mais ce n est PAS le chemin
// de production (image-cle seulement, porte `bVar14` posee).
// ---------------------------------------------------------------------------------------------

func TestAvantFeuille4MiniBobines(t *testing.T) {
	var zs, caps []float64
	total, aPlat := 0, 0
	for _, court := range closureMiniFilms() {
		fc, _ := avantContexte(t, court)
		if fc == nil {
			continue
		}
		if restore, err := InstallFilmFormatMPP(fc); err == nil {
			defer restore()
		}
		ctx := fc.ContexteDeLecture()
		n, plats := 0, 0
		for _, num := range fc.ChunkNumbers() {
			data, packets, ok := fc.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range packets {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				avantFeuille4UnPaquet(pk.Payload(data), ctx, &n, &plats, &zs, &caps)
			}
		}
		t.Logf("  %-10s feuille 4 lue sur %4d record(s) ti=40, dont %d a plat (haut = (0,0,1))",
			court, n, plats)
		total, aPlat = total+n, aPlat+plats
	}
	if total == 0 {
		t.Skip("aucune feuille 4 lisible sur les mini-bobines")
	}
	t.Logf("  CUMUL      %d feuille(s) 4, %d a plat (%.1f %%)", total, aPlat,
		100*float64(aPlat)/float64(total))
	if len(zs) > 0 {
		t.Logf("  |z| de la direction lue : mediane %.3f, part au-dessus de 0,9 = %.1f %% (n = %d)",
			avantMediane(zs), 100*avantPart(zs, func(z float64) bool { return z > 0.9 }), len(zs))
	}
	avantHistogrammeCaps(t, caps)
}

// avantFeuille4UnPaquet lit la feuille 4 de chaque record `ti=40` d image-cle dont la porte
// `bVar14` est posee, et rend la direction et l angle EN VALEUR.
func avantFeuille4UnPaquet(pay []byte, ctx ContexteDeLecture, n, plats *int, zs, caps *[]float64) {
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != VehicleTypeIndex {
			continue
		}
		br := LecteurSur(pay)
		br.PoserContexte(ctx)
		br.SetBitPos(b.Bit + br.cadre().EnTeteBits)
		mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil, bornee a 32
		if int32(br.ReadBits(mot)) <= 0 {       //nolint:gosec // n1, comparaison SIGNEE
			continue
		}
		consumeVersionPrefix(br)              // feuille 1
		consumeMultiplayerPropertiesBlock(br) // feuille 2
		if !br.ReadBit() {                    // feuille 3 : la porte bVar14
			continue
		}
		consumeSimStateHandleTail(br) // feuille 4, premiere moitie : la position absolue
		// feuille 4, seconde moitie : FUN_140c1e79c = R(1) porte ; si 0 -> R(19) ; puis R(8).
		up := [3]float32{0, 0, 1}
		if br.ReadBit() {
			*plats++
		} else {
			v, ok := DecodeAimVectorChecked(uint32(br.ReadBits(19)), 19)
			if !ok {
				continue
			}
			up = v
		}
		roll := RollAngleFromRaw(uint32(br.ReadBits(8)), 8)
		*n++
		*zs = append(*zs, math.Abs(float64(up[2])))
		f := ForwardFromUpRoll(up, roll)
		*caps = append(*caps, avantAzimut(float64(f[0]), float64(f[1])))
	}
}

// avantHistogrammeCaps : un cap RECONSTRUIT doit se repartir sur le tour. Une lecture a la
// mauvaise largeur, elle, s effondre sur quelques valeurs ou sur une seule face du cubemap.
func avantHistogrammeCaps(t *testing.T, caps []float64) {
	t.Helper()
	if len(caps) == 0 {
		return
	}
	var cases [12]int
	for _, c := range caps {
		i := int(c / 30)
		if i > 11 {
			i = 11
		}
		cases[i]++
	}
	t.Logf("  caps reconstruits, par secteur de 30 deg (n = %d) : %v", len(caps), cases)
}
