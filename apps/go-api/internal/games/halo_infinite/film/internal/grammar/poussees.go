package grammar

// poussees.go — LES POUSSEES D EQUIPEMENT D UN FILM : la paire 105 -> 104 de la vue A, son auteur et
// son impulsion (lot F du plan d ajustements du 2026-10-14).
//
// LA REGLE DE L ECRIVAIN. Pour chaque objet que le souffle deplace, le jeu ecrit un
// `EquipmentObjectKnockedBack` (105 : l objet, l unite qui pousse) ; quand l objet est un joueur, il
// ecrit a sa suite, dans la meme liste, un `EquipmentKnockbackPlayer` (104 : le joueur, l impulsion).
// Une poussee est donc la PAIRE : un 104 precede dans sa liste d un 105 de meme reference 0. Un 104
// sans ce 105 n est pas rendu, il est compte ([types.KnockbackStats.Unpaired]).
//
// LA BASE DES REFERENCES EST 512, la meme que celle des types 103 et 117 : l index de la reference 0
// plus 512 est le slot du bipede pousse, celui dont la piste accelere dans les 300 ms qui suivent le
// paquet (mesure du 2026-10-14 sur les temoins 0d9a9af9, 215e7022 et a6ae19fb).
//
// LECTURE SEULE, sans seuil : ce balayage rend ce que la vue A ecrit. Il n y a pas de poussee hors
// d une vue A lue (une liste arretee avant ses messages de poussee ne les rend pas).

import (
	"cmp"
	"math"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// knockbackRefBase est la base des references du domaine 0 des genres 104 et 105 (cf. l en-tete).
const knockbackRefBase = 512

// Les bornes de la norme de l impulsion, lues dans l executable : `FUN_14116c344` passe
// `DAT_143cd8648` (0,05) et `DAT_143cd8f60` (20,0) a `FUN_14076d528`, qui les donne a `FUN_14076d6dc`
// avec la largeur 10 ; `DAT_143cd8374` (1,0) et `DAT_143cd84b0` (0,5) sont les deux constantes de sa
// loi logarithmique.
const (
	normePousseeMin     = 0.05
	normePousseeMax     = 20.0
	normePousseeUn      = 1.0
	normePousseeDemi    = 0.5
	normePousseeLargeur = 10
)

// ScanKnockbacks lit les poussees d equipement d un film DEJA CHARGE, dans la passe des tetes, triees
// par instant puis par slot pousse.
func ScanKnockbacks(fc *FilmContext) ([]types.Knockback, types.KnockbackStats) {
	c := &canalDesPoussees{}
	distribuerLesTetesSeules(fc, []Canal{c})
	return c.out, c.st
}

// canalDesPoussees recueille les messages de poussee de la vue A de chaque trame.
type canalDesPoussees struct {
	out []types.Knockback
	st  types.KnockbackStats
}

func (*canalDesPoussees) Interets() []Interet { return nil }

func (c *canalDesPoussees) Tete(p *lecture.Paquet) {
	ms := p.VueA.Poussees
	for i, m := range ms {
		switch m.Genre {
		case GenreObjetRepousse:
			c.st.Read105++
		case GenrePousseeDuJoueur:
			c.st.Read104++
			k, ok := apparierLaPoussee(ms[:i], m)
			if !ok {
				c.st.Unpaired++
				continue
			}
			k.TimestampUS = p.TS
			c.compter(k)
			c.out = append(c.out, k)
		}
	}
}

func (c *canalDesPoussees) compter(k types.Knockback) {
	c.st.Paired++
	if !k.HasAuthor {
		c.st.WithoutAuthor++
	}
	if !k.HasImpulse {
		c.st.WithoutImpulse++
	}
}

func (c *canalDesPoussees) Clore(BilanDeMarche) { trierLesPoussees(c.out) }

// apparierLaPoussee rend la poussee du 104 `m` quand un 105 de meme reference 0 le precede dans sa
// liste (`avant`) : le plus proche, l ecrivain les ecrivant a la suite.
func apparierLaPoussee(avant []lecture.MessageDePoussee, m lecture.MessageDePoussee) (types.Knockback, bool) {
	if !aLeDrapeau(m, lecture.PousseeRef0) {
		return types.Knockback{}, false
	}
	for j := len(avant) - 1; j >= 0; j-- {
		o := avant[j]
		if o.Genre != GenreObjetRepousse || !aLeDrapeau(o, lecture.PousseeRef0) || o.Ref[0] != m.Ref[0] || o.Gen[0] != m.Gen[0] {
			continue
		}
		k := types.Knockback{Pushed: vieDeLaReference(m.Ref[0], m.Gen[0])}
		if aLeDrapeau(o, lecture.PousseeRef1) {
			k.Author, k.HasAuthor = vieDeLaReference(o.Ref[1], o.Gen[1]), true
		}
		if aLeDrapeau(m, lecture.PousseeImpulsionLue) {
			k.Impulse, k.HasImpulse = impulsionDeLaPoussee(m.Direction, m.Norme), true
		}
		return k, true
	}
	return types.Knockback{}, false
}

// vieDeLaReference applique la base des references au slot.
func vieDeLaReference(index uint16, gen uint8) types.LifeKey {
	return types.LifeKey{Slot: uint32(index) + knockbackRefBase, Gen: uint32(gen)}
}

// impulsionDeLaPoussee rend le vecteur de `FUN_14076d528` : la direction unitaire `FUN_1406d8288`
// sur 19 bits, multipliee par la norme de [normeDeLaPoussee].
func impulsionDeLaPoussee(direction uint32, norme uint16) [3]float32 {
	x, y, z := DecodeAimVector(direction, largeurDirection)
	n := float32(normeDeLaPoussee(norme))
	return [3]float32{x * n, y * n, z * n}
}

// normeDeLaPoussee porte `FUN_14076d6dc` : 0 rend le minimum, `2^10 - 1` et au-dela le maximum ;
// entre les deux, `exp((q + 0,5) * ln(1 - min + max) / 2^10) - (1 - min)`.
func normeDeLaPoussee(q uint16) float64 {
	n := 1 << normePousseeLargeur
	switch {
	case q == 0:
		return normePousseeMin
	case int(q) >= n-1:
		return normePousseeMax
	}
	decalage := normePousseeUn - normePousseeMin
	pas := math.Log(decalage+normePousseeMax) / float64(n)
	return math.Exp(float64(q)*pas+pas*normePousseeDemi) - decalage
}

// trierLesPoussees range les poussees dans un ordre TOTAL : instant, puis slot pousse ; les ex aequo
// restants dans l ordre du film.
func trierLesPoussees(out []types.Knockback) {
	slices.SortStableFunc(out, func(a, b types.Knockback) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Pushed.Slot, b.Pushed.Slot))
	})
}
