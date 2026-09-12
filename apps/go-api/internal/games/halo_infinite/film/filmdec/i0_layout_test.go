package filmdec

import "testing"

// synthFlipProfile fabrique un profil de bascule conforme au modèle physique : gate à zéro,
// puis pour chaque champ une chaîne géométrique du MSB (plancher) vers le LSB (saturation
// ~50 %). C'est exactement la forme mesurée sur les films (cf. i0_layout.go).
func synthFlipProfile(gate int, widths []int, tail int) []float64 {
	out := make([]float64, 0, 72)
	for i := 0; i < gate; i++ {
		out = append(out, 0)
	}
	for _, w := range widths {
		for k := 0; k < w; k++ {
			r := 0.5 / float64(uint64(1)<<uint(w-1-k))
			if r < 0.0009 {
				r = 0.0009 // plancher observé (respawns / téléportations rares)
			}
			out = append(out, r)
		}
	}
	for i := 0; i < tail; i++ {
		out = append(out, 0)
	}
	return out
}

// TestI0Boundaries_ReadsFieldStarts : le détecteur retrouve les MSB des trois axes et la fin
// d'i0 sur les découpages réellement mesurés (Cliffhanger, Catalyst, Aquarius, Illusion).
func TestI0Boundaries_ReadsFieldStarts(t *testing.T) {
	cases := []struct {
		name   string
		widths []int
		want   []int
	}{
		{"cliffhanger", []int{13, 13, 14}, []int{18, 31, 45}},
		{"catalyst", []int{15, 15, 15}, []int{20, 35, 50}},
		{"aquarius", []int{13, 12, 11}, []int{18, 30, 41}},
		{"illusion", []int{18, 18, 17}, []int{23, 41, 58}},
	}
	for _, c := range cases {
		prof := synthFlipProfile(DefaultI0GateBits, c.widths, 4)
		got := i0Boundaries(prof)
		if len(got) != len(c.want) {
			t.Fatalf("%s : %d frontière(s) %v, attendu %v", c.name, len(got), got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s : frontières %v, attendu %v", c.name, got, c.want)
				break
			}
		}
	}
}

// TestI0Boundaries_NoFieldNoBoundary : un profil plat (aucun champ quantifié continu) ne
// produit aucune frontière — le détecteur refuse plutôt que d'inventer un découpage.
func TestI0Boundaries_NoFieldNoBoundary(t *testing.T) {
	flat := make([]float64, 48)
	for i := range flat {
		flat[i] = 0.5
	}
	if got := i0Boundaries(flat); len(got) != 0 {
		t.Errorf("profil plat : frontières %v, attendu aucune", got)
	}
}

// TestI0LayoutGeometry : offsets, longueur totale et validation des largeurs.
func TestI0LayoutGeometry(t *testing.T) {
	lay := I0Layout{GateBits: DefaultI0GateBits, AxisW: [3]uint{15, 15, 15}}
	if got := lay.TotalBits(); got != 50 {
		t.Errorf("TotalBits = %d, attendu 50", got)
	}
	for ax, want := range []int{5, 20, 35} {
		if got := lay.AxisOffset(ax); got != want {
			t.Errorf("AxisOffset(%d) = %d, attendu %d", ax, got, want)
		}
	}
	if !lay.Valid() {
		t.Error("15/15/15 devrait être valide")
	}
	if (I0Layout{GateBits: 5, AxisW: [3]uint{15, 27, 15}}).Valid() {
		t.Error("une largeur > 26 (cap moteur) devrait être rejetée")
	}
	if (I0Layout{GateBits: 2, AxisW: [3]uint{13, 13, 14}}).Valid() {
		t.Error("un gate plus court que spine+useDefault devrait être rejeté")
	}
}
