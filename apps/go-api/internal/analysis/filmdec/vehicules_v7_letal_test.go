package filmdec

// vehicules_v7_letal_test.go — INSTRUMENT (lot V7) : Y A-T-IL UN DRAPEAU DE LETALITE dans la
// charge d'un evenement qui VISE un vehicule ?
//
// D'OU VIENT LA QUESTION. Le rapport V3 (§ 6, voie 2) nomme « l'evenement de degat de type 0 de
// la liste, jamais decode » comme la voie ouverte pour dater la destruction. Le lot V7 a etabli
// que ce type 0 designe bien une UNITE par sa reference 0, et que 3 099 de ses 18 684 instances
// (12 films) visent un VEHICULE. Mais leur instant se repartit dans toute la vie du vehicule
// (position relative mediane 0,73, quartiles 0,46-0,87) : la plupart sont des degats subis, pas
// la destruction. SI la destruction est dans le type 0, elle est distinguee par un DRAPEAU.
//
// LE PRECEDENT EST DANS LE DEPOT : `fire_events.go` lit cinq drapeaux aux bits 108..112 du meme
// record, dont trois commandent la presence du champ de visee. Un bit de charge qui vaut
// « la cible est morte » aurait exactement cette forme.
//
// L'ORACLE, ECRIT AVANT LA MESURE. On cherche un bit `b` tel que, sur les instances qui visent un
// vehicule :
//
//	P(l'instant tombe dans la fenetre de disparition du vehicule vise | bit b = 1) >= 90 %
//	P(la meme chose | bit b = 0) reste au plancher
//
// et tel que le nombre d'instances a `b = 1` soit du bon ordre (quelques unites a quelques
// dizaines par film — une destruction est rare).
//
// LE TEMOIN : la meme partition, mesuree contre la fenetre de disparition DECALEE de +60 s. Un
// bit qui separe vraiment doit voir son temoin s'effondrer. Le second temoin est la table
// elle-meme : 250 bits sont balayes, donc le MEILLEUR de 250 tirages doit etre juge comme tel —
// c'est pourquoi la colonne `n(b=1)` est publiee : un bit qui ne separe que trois instances
// n'apprend rien.
//
// Garde d'environnement V7_ROOT / V7_FILMS (+ V7_TYPES) : sans elle, tout SKIP.

import (
	"os"
	"sort"
	"strconv"
	"testing"
)

// v7LetalMaxBit borne le balayage : le record de tir (type 36) lit jusqu'au bit 142, 256 couvre
// large sans exploser le cout.
const v7LetalMaxBit = 256

// v7LetalTypes : les types depouilles par defaut — ceux dont `TestV7Chaine` a montre que la
// reference 0 se lit proprement en domaine 1 ET qui visent parfois un vehicule.
var v7LetalTypes = []int{0, 1, 5, 7, 36}

// v7LetalCell est la table 2x2 d'un bit : instances a 1 / a 0, et parmi elles celles dont
// l'instant tombe dans la fenetre de disparition du vehicule vise (reel, puis temoin +60 s).
type v7LetalCell struct{ n1, e1, s1, n0, e0 int }

// v7LetalAcc accumule un type.
type v7LetalAcc struct {
	n     int
	bits  []v7LetalCell
	ends  int // instances dont l'instant est deja dans la fenetre (base de comparaison)
	shift int
}

func newV7LetalAcc() *v7LetalAcc {
	return &v7LetalAcc{bits: make([]v7LetalCell, v7LetalMaxBit)}
}

// v7LetalScan balaie un film.
func v7LetalScan(dir string, want map[int]bool, acc map[int]*v7LetalAcc) {
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
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present || !want[ty] {
				continue
			}
			r := readDom1Ref(pay, eventPayloadStartBit)
			if !r.Present || r.Sonde != 1 {
				continue
			}
			slot := base + r.Index
			if !veh.band[slot] {
				continue
			}
			a := acc[ty]
			if a == nil {
				a = newV7LetalAcc()
				acc[ty] = a
			}
			v7LetalOne(a, pay, veh.ending(slot, p.TimestampUS),
				veh.ending(slot, p.TimestampUS+v7ShiftUS))
		}
	}
}

// v7LetalOne range UNE instance dans la table de chaque bit.
func v7LetalOne(a *v7LetalAcc, pay []byte, end, shift bool) {
	a.n++
	if end {
		a.ends++
	}
	if shift {
		a.shift++
	}
	bits := len(pay) * 8
	for b := eventPayloadStartBit; b < v7LetalMaxBit && b < bits; b++ {
		c := &a.bits[b]
		if readBitsAt(pay, b, 1) == 1 {
			c.n1++
			if end {
				c.e1++
			}
			if shift {
				c.s1++
			}
			continue
		}
		c.n0++
		if end {
			c.e0++
		}
	}
}

// TestV7Letal — LE BALAYAGE DE DRAPEAU.
func TestV7Letal(t *testing.T) {
	dirs := v7FilmDirs(t)
	list := v7LetalTypes
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
	want := map[int]bool{}
	for _, v := range list {
		want[v] = true
	}
	acc := map[int]*v7LetalAcc{}
	for _, d := range dirs {
		v7LetalScan(d, want, acc)
	}
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 LETAL — bits %d..%d, sur les instances qui VISENT un vehicule ==",
		eventPayloadStartBit, v7LetalMaxBit-1)
	for _, ty := range tys {
		a := acc[ty]
		if a.n == 0 {
			continue
		}
		t.Logf("type %d · %d instances visant un vehicule · deja en fenetre de fin : %.1f %% "+
			"(temoin +60 s : %.1f %%)", ty, a.n, 100*float64(a.ends)/float64(a.n),
			100*float64(a.shift)/float64(a.n))
		t.Logf("   %-5s %-8s %-9s %-9s %-9s", "bit", "n(b=1)", "FIN|b=1", "FIN|b=0", "tem+60|b=1")
		for _, b := range v7LetalTop(a, 8) {
			c := a.bits[b]
			p := func(x, tot int) float64 {
				if tot == 0 {
					return 0
				}
				return 100 * float64(x) / float64(tot)
			}
			t.Logf("   %-5d %-8d %8.1f%% %8.1f%% %8.1f%%",
				b, c.n1, p(c.e1, c.n1), p(c.e0, c.n0), p(c.s1, c.n1))
		}
	}
}

// v7LetalTop rend les bits de plus fort ECART de taux (b=1 contre b=0), en exigeant un effectif
// minimal a b=1 — un bit qui ne separe que trois instances n'apprend rien.
func v7LetalTop(a *v7LetalAcc, k int) []int {
	minN := v7Max(a.n/100, 5)
	type sc struct {
		b int
		v float64
	}
	var all []sc
	for b := range a.bits {
		c := a.bits[b]
		if c.n1 < minN || c.n0 == 0 {
			continue
		}
		all = append(all, sc{b, float64(c.e1)/float64(c.n1) - float64(c.e0)/float64(c.n0)})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].b < all[j].b
	})
	var out []int
	for i := 0; i < len(all) && i < k; i++ {
		out = append(out, all[i].b)
	}
	return out
}
