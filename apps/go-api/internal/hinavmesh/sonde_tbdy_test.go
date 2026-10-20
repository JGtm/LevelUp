package hinavmesh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SONDE DE TBDY — l ETAT DE L ART du blocage d Absolution, au 2026-08-31.
//
// CE QU ELLE A ETABLI, ET QUI CORRIGE LE DIAGNOSTIC PRECEDENT. Le decodeur echoue sur « type 68
// (hkPropertyId), membre 50, indice de nom 98 hors des 98 chaines ». On a longtemps lu cet ecart
// d un comme une ORIGINE D INDEXATION (table 1-based). C est FAUX : hkPropertyId n a pas cinquante
// membres. Le flux TBDY est DESYNCHRONISE, et l indice hors bornes n est que le premier symptome
// visible. Chercher l origine du DECALAGE, pas celle de l indexation.
//
// CE QUI EST MESURE ICI, et que la prochaine tentative peut reprendre sans le refaire :
//
//   - le flux est SAIN jusqu a l entree 66 incluse : les indices de nom des membres croissent
//     regulierement (48..54, puis 55..58, puis 59..60), les offsets et les types sont coherents ;
//   - l entree 67 lit « 196609 membres » a partir des octets 44 00 29 07 08 08 c3 00 01 ... ;
//   - l entree SUIVANTE semble commencer treize octets plus loin (45 00 2b = type 69, parent 0,
//     drapeaux 0x2b), ce qui donne la taille reelle de l entree 68 : 13 octets ;
//   - les DRAPEAUX DE MEMBRE distinguent les deux generations : Isolation ne connait que 0x20,
//     0x22, 0x24 et 0x25 ; Absolution ajoute 0x21 (une fois) et 0x23 (trois fois).
//
// PISTE ESSAYEE ET REFUTEE LE 2026-08-31 : lire un entier de plus quand le drapeau de membre porte
// le bit 0x01 sans le bit 0x04 (donc sur 0x21 et 0x23, jamais sur le 0x25 d Isolation). Elle est
// bien inerte sur Isolation, mais elle fait echouer Absolution PLUS TOT — a l offset 208 au lieu
// de 818. L entier de plus n est donc pas gouverne par ces bits-la.
//
// La sonde reste versionnee parce qu elle porte la mesure : sans elle, la prochaine tentative
// recommencerait par l hypothese deja refutee.
func TestSondeCorpsDeTypes(t *testing.T) {
	const racine = `C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`
	for _, cas := range []struct{ nom, blob string }{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260.blob"},
		{"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352.blob"},
	} {
		brut, err := os.ReadFile(filepath.Join(racine, cas.blob))
		if err != nil {
			t.Skipf("blob absent (%v)", err)
		}
		charge, err := decompresse(brut)
		if err != nil {
			t.Fatalf("%s : %v", cas.nom, err)
		}
		decoupe, err := regions(charge)
		if err != nil {
			t.Fatalf("%s : %v", cas.nom, err)
		}
		var region []byte
		for _, r := range decoupe {
			if len(r) >= 8 && string(r[4:8]) == sectionTAG0 {
				region = r
				break
			}
		}
		sections := map[string][2]int{}
		if err := parcoursSections(region, 0, len(region), sections, 0); err != nil {
			t.Fatalf("%s : %v", cas.nom, err)
		}
		var secT, secF [2]int
		for _, p := range sectionsChaines {
			if a, ok := sections[p[0]]; ok {
				if b, ok2 := sections[p[1]]; ok2 {
					secT, secF = a, b
					break
				}
			}
		}
		nomsTypes := chaines(region, secT)
		types, err := lireNomsTypes(region, sections["TNA1"], nomsTypes)
		if err != nil {
			t.Fatalf("%s : TNA1 : %v", cas.nom, err)
		}
		t.Logf("%s : %d types, %d noms de champs", cas.nom, len(types), len(chaines(region, secF)))

		flagsVus := map[int]int{}
		sec := sections["TBDY"]
		l := &lecteurEmpaquete{buf: region[sec[0] : sec[0]+sec[1]]}
		for n := 0; l.err == nil && !l.fini() && n < 400; n++ {
			debut := l.pos
			idx := l.entier()
			if l.err != nil || idx <= 0 || idx >= len(types) {
				t.Logf("  entree %d a l offset %d : indice de type %d HORS BORNES (%d types)", n, debut, idx, len(types))
				break
			}
			parent := l.entier()
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
			nbM := 0
			if dr&drapeauMembres != 0 {
				nbM = l.entier()
				if nbM < 0 || nbM > 64 {
					fin := debut + 24
					if fin > len(l.buf) {
						fin = len(l.buf)
					}
					t.Logf("  >>> entree %d, offset %d : type %d (%s), parent %d, drapeaux 0x%02x, %d MEMBRES — invraisemblable ; octets % x",
						n, debut, idx, types[idx].Nom, parent, dr, nbM, l.buf[debut:fin])
					break
				}
				var det []string
				for j := 0; j < nbM && l.err == nil; j++ {
					iNom := l.entier()
					fl := l.entier()
					of := l.entier()
					ty := l.entier()
					det = append(det, fmt.Sprintf("nom=%d fl=0x%x off=%d typ=%d", iNom, fl, of, ty))
				}
				if n >= 60 {
					t.Logf("      membres : %s", strings.Join(det, " | "))
				}
				for _, d := range det {
					var fl int
					fmt.Sscanf(d[strings.Index(d, "fl=0x")+5:], "%x", &fl)
					flagsVus[fl]++
				}
			}
			if dr&drapeauInterfaces != 0 {
				nbI := l.entier()
				if nbI < 0 || nbI > 32 {
					t.Logf("  >>> entree %d, offset %d : type %d (%s), %d INTERFACES — invraisemblable",
						n, debut, idx, types[idx].Nom, nbI)
					break
				}
				for j := 0; j < nbI && l.err == nil; j++ {
					l.entier()
					l.entier()
				}
			}
			_ = flagsVus
			if n < 8 || n >= 60 {
				t.Logf("  entree %d : type %d (%s), parent %d, drapeaux 0x%02x, %d membres",
					n, idx, types[idx].Nom, parent, dr, nbM)
			}
		}
		t.Logf("%s : drapeaux de MEMBRE vus : %v", cas.nom, flagsVus)
	}
}
