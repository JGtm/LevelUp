package grammar

// vue_a_detonations_test.go — LES MESSAGES DE DETONATION/IMPACT DE LA VUE A ET LEUR CAPTURE
// (vue_a_detonations.go, detonations.go).

import (
	"math"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// ecrireUneDetonation ecrit un message de detonation/impact : continuation, genre, la reference 0
// (domaine 5) quand positive, les references 1 et 2 (domaines 8 et 7, jamais presentes en
// pratique) absentes, puis la charge : `avant` la position, une position de niveau a porte posee
// (table DEFAUT du build, aux largeurs `larg`), puis `apres`.
func (w *bitWriter) ecrireUneDetonation(genre, ref0 int, avant, apres func(w *bitWriter), larg [3]uint) {
	w.bit(1)
	w.bits(uint64(genre), LargeurGenreVueA)
	if ref0 < 0 {
		w.bit(0)
	} else {
		w.bit(1)
		w.bits(uint64(ref0), int(varWidthBits(5)))
		w.bits(1, 2) // generation
	}
	w.bit(0) // ref1 (domaine 8) absente
	w.bit(0) // ref2 (domaine 7) absente
	avant(w)
	w.ecrirePositionDeNiveau(true, 0, larg) // porte posee : table DEFAUT du build
	apres(w)
}

// TestLaVueARangeLesDetonations : un `projectile_detonate` (genre 5) nommant sa reference de
// projectile (domaine 5) et une position (porte posee) se lit jusqu a son terminateur, et la vue A
// range le message avec sa reference et sa position dequantifiee aux bornes DEFAUT du build —
// exactement les bits qu avant ce lot [chargeDetonation] consommait puis jetait
// (TestLesPositionsAIndexSeLisentSurLaRegionJouee verifie deja cette consommation). MUTATIONS — le
// domaine de la reference 0 change (varWidthBits(5) -> une autre categorie), la porte de position
// inversee, le niveau 0xf lu a 0x10 : ROUGE.
func TestLaVueARangeLesDetonations(t *testing.T) {
	larg := profile.LargeursAxeParDefautDuBuild(niveauDetonation)
	var w bitWriter
	w.bit(1) // bit de configuration
	w.ecrireUneDetonation(GenreProjectileDetonate, 42,
		func(w *bitWriter) { w.bits(0, 6); w.bit(1); w.bit(0) }, // FUN_140809454, variante, FUN_14080d69c
		func(w *bitWriter) {
			w.bits(0, largeurDirection+5+1+largeurDetonationIndex)
			w.bit(0) // consumeGateR(10+8) : gate fermee
			w.bit(0) // ReadBit() largeur : largeurDeuxiemeDirection
			w.bits(0, largeurDeuxiemeDirection+2)
		}, larg)
	w.bit(0) // terminateur
	w.bits(0, 16)
	a := lireSous(w.buf, grammaireRecente())
	if !a.Porte || len(a.Detonations) != 1 {
		t.Fatalf("vue A : portee %v, %d detonation(s), attendu portee et 1", a.Porte, len(a.Detonations))
	}
	d := a.Detonations[0]
	if d.Genre != GenreProjectileDetonate || d.Ref0 != 42 || !aLeDrapeauDetonation(d, lecture.DetonationRef0) {
		t.Errorf("detonation lue %+v, attendu ref0 42", d)
	}
	if !aLeDrapeauDetonation(d, lecture.DetonationPositionLue) {
		t.Fatalf("position non lue : %+v", d)
	}
	want := DequantBipedAxis(0, 0, profile.I0Layout{AxisW: larg}, profile.QuantRangeParDefautDuBuild())
	if math.Abs(float64(d.X-want)) > 1e-6 || math.Abs(float64(d.Y-want)) > 1e-6 || math.Abs(float64(d.Z-want)) > 1e-6 {
		t.Errorf("position dequantifiee %+v, attendu %.4f sur les trois axes (quantum 0)", d, want)
	}
	if d.Debut != 1 {
		t.Errorf("debut = %d, attendu 1 (la continuation qui l annonce)", d.Debut)
	}
}

// TestLeCanalDesDetonationsPublieSansSeuil : le canal publie CHAQUE message lu, qu il porte ou non
// sa reference et sa position — c est l appelant (detonations.go doit documenter, jamais filtrer en
// silence) qui decide de ce qu il fait d une detonation sans l un des deux. MUTATION — une
// detonation sans reference ni position ecartee du resultat : ROUGE.
func TestLeCanalDesDetonationsPublieSansSeuil(t *testing.T) {
	c := &canalDesDetonations{}
	pq := &lecture.Paquet{TS: 2000}
	pq.VueA.Detonations = []lecture.MessageDeDetonation{
		{Genre: GenreProjectileDetonate, Ref0: 7, Gen0: 1,
			Drapeaux: lecture.DetonationRef0 | lecture.DetonationPositionLue, X: 1, Y: 2, Z: 3},
		{Genre: GenreProjectileImpactEffect}, // ni reference ni position
	}
	c.Tete(pq)
	if c.st.Read != 2 || c.st.WithProjectile != 1 || c.st.WithPosition != 1 {
		t.Fatalf("stats %+v, attendu 2 lus, 1 avec reference, 1 avec position", c.st)
	}
	if len(c.out) != 2 {
		t.Fatalf("%d detonations publiees, attendu 2 (lecture seule, sans seuil)", len(c.out))
	}
	d0 := c.out[0]
	if d0.TimestampUS != 2000 || d0.Projectile != (types.LifeKey{Slot: 7, Gen: 1}) || !d0.HasProjectile ||
		!d0.HasPosition || d0.X != 1 || d0.Y != 2 || d0.Z != 3 {
		t.Errorf("detonation %+v", d0)
	}
	d1 := c.out[1]
	if d1.HasProjectile || d1.HasPosition {
		t.Errorf("detonation sans reference ni position publiee avec l un des deux : %+v", d1)
	}
}
