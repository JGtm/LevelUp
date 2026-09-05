package filmdec

// vehicules_v7_cible_test.go — INSTRUMENT (lot V7) : LE DEPOUILLEMENT INSTANCE PAR INSTANCE des
// types CANDIDATS, avec leur reference 0 resolue et sa place dans la vie du vehicule vise.
//
// D'OU VIENNENT LES CANDIDATS. `TestV7Correlation` mesure, sur 40 films, l'effectif de chaque type
// de tete dans les films SANS vehicule et dans les films AVEC. Les deux evenements vehicule connus
// s'y comportent comme l'oracle l'annonce (type 22 sortie : 0,04 sans / 23,5 avec, r = 0,97 ;
// type 8 embarquement : 0,00 / 1,08, r = 0,92), ce qui valide la methode. Les types NEUFS qui
// partagent cette signature sont les candidats de ce fichier.
//
// CE QUE L'INSTRUMENT AJOUTE. Deux acquis du lot rendent le depouillement possible :
//
//  1. LE DOMAINE 1 EST CELUI DES UNITES, BIPEDES *ET* VEHICULES. Mesure (`TestV7Dom1`, 2 films) :
//     sur les references 0 de domaine 1 a sonde = 1, la somme « slot dans la bande bipede » +
//     « slot dans la bande vehicule » vaut 319/319 pour le type 7, 2 376/2 399 pour le type 36 et
//     2 381/2 393 pour le type 0 — il ne reste presque RIEN hors des deux bandes. La base est la
//     meme (le minimum de la bande bipede) et l'index de 9 bits porte au-dela : un vehicule EST
//     adressable par une reference de domaine 1. Le lot V6 avait cherche le vehicule en domaine 7
//     (la ref 2 de l'embarquement) et l'y avait refute ; il est en domaine 1.
//  2. LA FENETRE DE DISPARITION d'une vie recensee borne sa destruction : dernier recensement <=
//     t <= premiere image-cle qui ne la recense plus.
//
// LE VERDICT SE LIT SUR TROIS COLONNES : la part des instances qui RESOLVENT un vehicule, la part
// de celles-la qui tombent dans la fenetre de disparition de CE vehicule, et le TEMOIN a +60 s.
//
// Garde d'environnement V7_ROOT / V7_FILMS (+ V7_MAX, V7_TYPES) : sans elle, tout SKIP.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// v7CibleDefaut : les types depouilles par defaut. Les deux premiers sont les CONTROLES POSITIFS
// (embarquement et sortie, dont la reponse est connue) ; les suivants sont les candidats de la
// table de correlation ; les trois derniers sont les types dont `TestV7Dom1` a montre qu'ils
// visent parfois un vehicule (0 = degat, 7, 1).
var v7CibleDefaut = []int{8, 22, 2, 40, 41, 118, 3, 23, 47, 100, 105, 106, 117, 0, 7, 1}

// v7CibleInst est une instance depouillee.
type v7CibleInst struct {
	film             string
	atUS             uint64
	slot             uint32
	classe           string // VEH / BIP / hors / absente
	ending, endShift bool
	frac             float64
	lastUS, goneUS   uint64
}

// v7CibleAcc accumule les instances d'un type.
type v7CibleAcc struct {
	insts []v7CibleInst
	films map[string]int
}

// v7CibleTypes lit la liste des types demandes.
func v7CibleTypes(t *testing.T) map[int]bool {
	t.Helper()
	list := v7CibleDefaut
	if s := os.Getenv("V7_TYPES"); s != "" {
		list = nil
		for _, x := range splitComma(s) {
			v, err := strconv.Atoi(x)
			if err != nil {
				t.Fatalf("V7_TYPES : %q n'est pas un entier", x)
			}
			list = append(list, v)
		}
	}
	out := map[int]bool{}
	for _, v := range list {
		out[v] = true
	}
	return out
}

// v7CibleScan depouille un film.
func v7CibleScan(dir string, want map[int]bool, acc map[int]*v7CibleAcc) {
	id := filepath.Base(filepath.Clean(dir))
	veh := v7BandeFrom(ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI))
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	if len(chunks) == 0 {
		return
	}
	bip := bipedSlotBandMapDir(dir, chunks)
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
			if p.Type != PacketTypeDelta || p.Size < 6 {
				continue
			}
			pay := p.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present || !want[ty] {
				continue
			}
			a := acc[ty]
			if a == nil {
				a = &v7CibleAcc{films: map[string]int{}}
				acc[ty] = a
			}
			a.films[id]++
			a.insts = append(a.insts, v7CibleInst1(id, pay, p.TimestampUS, base, bip, veh))
		}
	}
}

// v7CibleInst1 resout la reference 0 d'une instance.
func v7CibleInst1(
	id string, pay []byte, at uint64, base uint32, bip map[uint32]bool, veh v7Bande,
) v7CibleInst {
	in := v7CibleInst{film: id, atUS: at, classe: "absente"}
	r := readDom1Ref(pay, eventPayloadStartBit)
	if !r.Present {
		return in
	}
	slot := r.Index
	if r.Sonde == 1 {
		slot = base + r.Index
	}
	in.slot = slot
	switch {
	case veh.band[slot]:
		in.classe = "VEH"
	case bip[slot]:
		in.classe = "BIP"
	default:
		in.classe = "hors"
	}
	if in.classe != "VEH" {
		return in
	}
	in.ending, in.endShift = veh.ending(slot, at), veh.ending(slot, at+v7ShiftUS)
	if l, ok := veh.lifeAt(slot, at); ok {
		in.lastUS, in.goneUS = l.last, l.gone
		if l.hi > l.lo {
			in.frac = float64(at-l.lo) / float64(l.hi-l.lo)
		}
	}
	return in
}

// TestV7Cible — LE DEPOUILLEMENT.
func TestV7Cible(t *testing.T) {
	dirs := v7CorrelDirs(t)
	want := v7CibleTypes(t)
	acc := map[int]*v7CibleAcc{}
	read := 0
	for _, d := range dirs {
		if CountFilmChunks(d) == 0 {
			continue
		}
		v7CibleScan(d, want, acc)
		read++
	}
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 CIBLE — %d films lus · types %v ==", read, tys)
	t.Logf("%-5s %-6s %-6s %-7s %-7s %-7s %-7s %-9s %-9s",
		"type", "n", "films", "absente", "VEH", "BIP", "hors", "FIN|VEH", "tem+60")
	for _, ty := range tys {
		a := acc[ty]
		var abs, veh, bip, hors, fin, shift int
		for _, in := range a.insts {
			switch in.classe {
			case "absente":
				abs++
			case "VEH":
				veh++
				if in.ending {
					fin++
				}
				if in.endShift {
					shift++
				}
			case "BIP":
				bip++
			default:
				hors++
			}
		}
		p := func(x, tot int) float64 {
			if tot == 0 {
				return 0
			}
			return 100 * float64(x) / float64(tot)
		}
		t.Logf("%-5d %-6d %-6d %6.1f%% %6.1f%% %6.1f%% %6.1f%% %8.1f%% %8.1f%%",
			ty, len(a.insts), len(a.films), p(abs, len(a.insts)), p(veh, len(a.insts)),
			p(bip, len(a.insts)), p(hors, len(a.insts)), p(fin, veh), p(shift, veh))
	}
	if os.Getenv("V7_DETAIL") == "" {
		return
	}
	for _, ty := range tys {
		t.Logf("-- type %d, instance par instance --", ty)
		for _, in := range acc[ty].insts {
			t.Logf("   %-10s t=%9.2f s slot=%-5d %-8s fin=%-5v tem=%-5v frac=%.2f "+
				"dernier=%9.2f absent=%9.2f", in.film, float64(in.atUS)/1e6, in.slot, in.classe,
				in.ending, in.endShift, in.frac, float64(in.lastUS)/1e6, float64(in.goneUS)/1e6)
		}
	}
}
