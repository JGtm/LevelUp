package hinavmesh

import (
	"os"
	"path/filepath"
	"testing"
)

// SONDE DE LA VALEUR « FORMAT ». L entree fautive d Absolution est la SEULE de sa section : un
// saut de 13 octets — sa taille exacte — resynchronise le flux jusqu au bout. Reste a savoir ce
// que ces 13 octets contiennent. Cette sonde releve, pour chaque entree qui porte le drapeau
// 0x01, la valeur lue juste apres les drapeaux : si l entree fautive est la seule a en porter une
// d une certaine forme, on tient le discriminant.
func TestSondeValeurFormat(t *testing.T) {
	for _, cas := range []struct{ nom, blob string }{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260.blob"},
		{"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352.blob"},
	} {
		if _, err := os.Stat(filepath.Join(
			`C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`, cas.blob)); err != nil {
			t.Skipf("blob absent (%v)", err)
		}
		region, sections, types, nomsChamps := chargePourSonde(t, cas.blob)
		sec := sections["TBDY"]
		corps := region[sec[0] : sec[0]+sec[1]]
		l := &lecteurEmpaquete{buf: corps}
		formats := map[int]int{}
		for l.err == nil && !l.fini() {
			idx := l.entier()
			if l.err != nil || idx <= 0 || idx >= len(types) {
				break
			}
			l.entier()
			dr := l.entier()
			var format = -1
			if dr&drapeauFormat != 0 {
				format = l.entier()
				formats[format]++
			}
			for _, d := range []int{drapeauSousType, drapeauVersion} {
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
					t.Logf("%s : ARRET sur le type %d (%s), drapeaux 0x%02x, format %d",
						cas.nom, idx, types[idx].Nom, dr, format)
					break
				}
				for j := 0; j < nb && l.err == nil; j++ {
					iNom := l.entier()
					l.entier()
					l.entier()
					l.entier()
					if iNom >= len(nomsChamps) {
						break
					}
				}
			}
			if dr&drapeauInterfaces != 0 {
				nb := l.entier()
				for j := 0; j < nb && l.err == nil; j++ {
					l.entier()
					l.entier()
				}
			}
		}
		t.Logf("%s : valeurs de FORMAT vues (valeur:occurrences) %v", cas.nom, formats)
	}
}
