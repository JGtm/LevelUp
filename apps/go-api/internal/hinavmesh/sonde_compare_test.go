package hinavmesh

import (
	"os"
	"path/filepath"
	"testing"
)

// SONDE COMPARATIVE : la MEME entree dans les deux generations.
//
// L entree fautive d Absolution est celle du type `hkPropertyId`. Si Isolation — qui se decode
// entierement — porte le meme type, la comparaison octet a octet de leurs deux entrees dit
// exactement ce que la generation TSTR/FSTR ajoute. C est le temoin le plus direct disponible :
// deux encodages du meme objet, l un lisible, l autre non.
func TestSondeCompareEntreeDuMemeType(t *testing.T) {
	const cible = "hkPropertyId"
	for _, cas := range []struct{ nom, blob string }{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260.blob"},
		{"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352.blob"},
	} {
		if _, err := os.Stat(filepath.Join(
			`C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`, cas.blob)); err != nil {
			t.Skipf("blob absent (%v)", err)
		}
		region, sections, types, _ := chargePourSonde(t, cas.blob)
		var vise int
		for i, ty := range types {
			if ty.Nom == cible {
				vise = i
			}
		}
		if vise == 0 {
			t.Logf("%s : type %q ABSENT de la table (%d types)", cas.nom, cible, len(types))
			continue
		}
		sec := sections["TBDY"]
		corps := region[sec[0] : sec[0]+sec[1]]
		l := &lecteurEmpaquete{buf: corps}
		for l.err == nil && !l.fini() {
			debut := l.pos
			idx := l.entier()
			if l.err != nil || idx <= 0 || idx >= len(types) {
				break
			}
			if idx == vise {
				fin := debut + 20
				if fin > len(corps) {
					fin = len(corps)
				}
				t.Logf("%s : %s = type %d, entree a l offset %d : % x",
					cas.nom, cible, idx, debut, corps[debut:fin])
				break
			}
			l.entier()
			dr := l.entier()
			for _, d := range []int{drapeauFormat, drapeauSousType, drapeauVersion} {
				if dr&d != 0 {
					l.entier()
				}
			}
			if dr&drapeauTaille != 0 {
				l.entier()
				l.entier()
			}
			if dr&drapeauInconnu != 0 {
				l.entier()
			}
			if dr&drapeauMembres != 0 {
				nb := l.entier()
				if nb < 0 || nb > 64 {
					break
				}
				for j := 0; j < nb*4 && l.err == nil; j++ {
					l.entier()
				}
			}
			if dr&drapeauInterfaces != 0 {
				nb := l.entier()
				for j := 0; j < nb*2 && l.err == nil; j++ {
					l.entier()
				}
			}
		}
	}
}
