package hinavmesh

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// SONDE DE RESYNCHRONISATION DE TBDY.
//
// Plutot que de deviner la grammaire de l entree fautive, on cherche OU le flux redevient sain :
// pour chaque longueur candidate de l entree, on tente de lire tout le reste de la section. La
// longueur qui permet d aller jusqu au bout — et une seule doit y parvenir — donne la taille
// reelle de l entree, donc ce qu on ne sait pas lire.
func TestSondeResynchronisationTBDY(t *testing.T) {
	region, sections, types, nomsChamps := chargePourSonde(t,
		"78da545f-a168-4a5e-9c8d-dd379067c352.blob")
	sec := sections["TBDY"]
	corps := region[sec[0] : sec[0]+sec[1]]

	pos := 0
	for tour := 0; tour < 40; tour++ {
		fin, entrees, err := litEntrees(corps, pos, types, nomsChamps)
		if err == nil {
			t.Logf("SECTION LUE ENTIEREMENT depuis l offset %d (%d entrees)", pos, entrees)
			return
		}
		t.Logf("blocage a l offset %d apres %d entrees : %v", fin, entrees, err)
		octets := corps[fin:min(fin+28, len(corps))]
		t.Logf("   octets : % x", octets)

		var bonnes []int
		for saut := 1; saut <= 40 && fin+saut < len(corps); saut++ {
			if _, _, e := litEntrees(corps, fin+saut, types, nomsChamps); e == nil {
				bonnes = append(bonnes, saut)
			}
		}
		if len(bonnes) == 0 {
			t.Logf("   AUCUN saut de 1 a 40 octets ne resynchronise — l anomalie n est pas isolee")
			return
		}
		t.Logf("   sauts qui menent jusqu au bout : %v (le plus court = taille reelle de l entree)", bonnes)
		return
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// litEntrees deroule TBDY depuis un offset et rend l offset de blocage, le nombre d entrees lues
// et l erreur eventuelle.
func litEntrees(corps []byte, depuis int, types tableTypes, nomsChamps []string) (int, int, error) {
	l := &lecteurEmpaquete{buf: corps, pos: depuis}
	n := 0
	for l.err == nil && !l.fini() {
		debut := l.pos
		idx := l.entier()
		if l.err != nil {
			return debut, n, l.err
		}
		if idx <= 0 || idx >= len(types) {
			return debut, n, fmt.Errorf("indice de type %d hors des %d types", idx, len(types))
		}
		l.entier() // parent
		dr := l.entier()
		if dr&drapeauReserve != 0 {
			return debut, n, fmt.Errorf("drapeau 0x80 sur le type %d", idx)
		}
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
				return debut, n, fmt.Errorf("type %d (%s) : %d membres", idx, types[idx].Nom, nb)
			}
			for j := 0; j < nb && l.err == nil; j++ {
				iNom := l.entier()
				l.entier()
				l.entier()
				l.entier()
				if iNom < 0 || iNom >= len(nomsChamps) {
					return debut, n, fmt.Errorf("type %d (%s) membre %d : nom %d hors des %d chaines",
						idx, types[idx].Nom, j, iNom, len(nomsChamps))
				}
			}
		}
		if dr&drapeauInterfaces != 0 {
			nb := l.entier()
			if nb < 0 || nb > 32 {
				return debut, n, fmt.Errorf("type %d : %d interfaces", idx, nb)
			}
			for j := 0; j < nb && l.err == nil; j++ {
				l.entier()
				l.entier()
			}
		}
		if l.err != nil {
			return debut, n, l.err
		}
		n++
	}
	return l.pos, n, nil
}

// chargePourSonde ouvre un blob et rend region, sections, types et noms de champs.
func chargePourSonde(t *testing.T, blob string) ([]byte, map[string][2]int, tableTypes, []string) {
	t.Helper()
	const racine = `C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`
	brut, err := os.ReadFile(filepath.Join(racine, blob))
	if err != nil {
		t.Skipf("blob absent (%v)", err)
	}
	charge, err := decompresse(brut)
	if err != nil {
		t.Fatal(err)
	}
	decoupe, err := regions(charge)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
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
	types, err := lireNomsTypes(region, sections["TNA1"], chaines(region, secT))
	if err != nil {
		t.Fatal(err)
	}
	return region, sections, types, chaines(region, secF)
}
