package filmdec

// vehicules_v7_chaine_test.go — INSTRUMENT (lot V7) : LES TROIS REFERENCES D'UN EVENEMENT, LUES
// EN CHAINE COMME DES REFERENCES D'UNITE.
//
// POURQUOI CE QUATRIEME ANGLE. `TestV7Dom1` n'a lu que la reference 0, et il a etabli un fait
// structurel : le DOMAINE 1 EST CELUI DES UNITES, bipedes ET vehicules confondus (partition
// « bande bipede ou bande vehicule » a 100,0 % sur les types 1 et 7, 99,96 % sur le type 0,
// 99,8 % sur le type 36, 12 films). Mais la reference 0 d'un evenement de degat est l'ATTAQUANT
// (97,8 % bipede pour le type 36, le tir) : ce n'est pas la qu'une CIBLE se lit.
//
// La decomposition mesuree au lot V6 dit ou chercher : la SORTIE occupe 9 + [13 + 13 + 1] + 6,
// c'est-a-dire DEUX references de domaine 1 puis une absente ; et le decodeur de tir place son
// premier champ de charge au bit 36, soit exactement 9 + 27. LES DEUX PREMIERES REFERENCES SONT
// DONC DES UNITES, et la seconde n'a jamais ete regardee. C'est ce que ce fichier fait :
// il chaine `readDom1Ref` trois fois et publie la partition de CHACUNE.
//
// LE CRITERE ET LES TEMOINS ne changent pas : une reference qui designe un vehicule dont
// l'instant tombe dans la FENETRE DE DISPARITION (dernier recensement .. premiere preuve
// d'absence) ; temoin temporel a +60 s ; temoin de cadrage = la meme chaine relue un bit plus
// loin (`TestV7Dom1`).
//
// Garde d'environnement V7_ROOT / V7_FILMS : sans elle, tout SKIP.

import (
	"path/filepath"
	"sort"
	"testing"
)

// v7RefStat est la partition d'UNE reference d'un type d'evenement.
type v7RefStat struct {
	n, absente, sonde1     int
	biped, veh, hors       int
	end, shift             int
	s0Biped, s0Veh, s0Hors int
}

// v7Chaine accumule les trois references d'un type.
type v7Chaine3 struct {
	n    int
	refs [3]v7RefStat
}

// v7ChaineClasse range une reference decodee et rend le slot resolu.
func v7ChaineClasse(
	st *v7RefStat, r guardedRef, at uint64, base uint32, bip map[uint32]bool, veh v7Bande,
) {
	st.n++
	if !r.Present {
		st.absente++
		return
	}
	if r.Sonde == 0 {
		switch slot := r.Index; {
		case veh.band[slot]:
			st.s0Veh++
		case bip[slot]:
			st.s0Biped++
		default:
			st.s0Hors++
		}
		return
	}
	st.sonde1++
	slot := base + r.Index
	switch {
	case bip[slot]:
		st.biped++
	case veh.band[slot]:
		st.veh++
		if veh.ending(slot, at) {
			st.end++
		}
		if veh.ending(slot, at+v7ShiftUS) {
			st.shift++
		}
	default:
		st.hors++
	}
}

// v7ChaineScan balaie un film.
func v7ChaineScan(dir string, acc map[int]*v7Chaine3) {
	veh := v7BandeFrom(ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI))
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	if len(chunks) == 0 {
		return
	}
	bip := bipedSlotBand(dir, chunks)
	base := ^uint32(0)
	for s := range bip {
		if s < base {
			base = s
		}
	}
	if len(bip) == 0 {
		base = 0
	}
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 8 {
				continue
			}
			pay := p.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present {
				continue
			}
			a := acc[ty]
			if a == nil {
				a = &v7Chaine3{}
				acc[ty] = a
			}
			a.n++
			r0 := readDom1Ref(pay, eventPayloadStartBit)
			r1 := readDom1Ref(pay, r0.EndBit)
			r2 := readDom1Ref(pay, r1.EndBit)
			for i, r := range [3]guardedRef{r0, r1, r2} {
				v7ChaineClasse(&a.refs[i], r, p.TimestampUS, base, bip, veh)
			}
		}
	}
}

// TestV7Chaine3 — LA TABLE. Trois lignes par type de tete.
func TestV7Chaine3(t *testing.T) {
	dirs := v7FilmDirs(t)
	acc := map[int]*v7Chaine3{}
	for _, dir := range dirs {
		v7ChaineScan(dir, acc)
		t.Logf("film %s lu", filepath.Base(filepath.Clean(dir)))
	}
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 CHAINE — trois references chainees, lues en domaine 1 ==")
	t.Logf("%-5s %-4s %-7s %-8s | sonde=1 : %-7s %-6s %-6s %-8s %-8s | sonde=0 : %-6s %-6s %-6s",
		"type", "ref", "n", "absente", "bipede", "VEH", "hors", "FIN|VEH", "tem+60",
		"veh", "bip", "hors")
	for _, ty := range tys {
		a := acc[ty]
		for i := 0; i < 3; i++ {
			st := a.refs[i]
			if st.n == 0 {
				continue
			}
			p := func(x, tot int) float64 {
				if tot == 0 {
					return 0
				}
				return 100 * float64(x) / float64(tot)
			}
			t.Logf("%-5d %-4d %-7d %7.1f%% | %6.1f%% %6d %6d %7.1f%% %7.1f%% | %6d %6d %6d",
				ty, i, st.n, p(st.absente, st.n),
				p(st.biped, st.sonde1), st.veh, st.hors, p(st.end, st.veh), p(st.shift, st.veh),
				st.s0Veh, st.s0Biped, st.s0Hors)
		}
	}
}
