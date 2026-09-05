package filmdec

// vehicules_v7_dom1_test.go — INSTRUMENT (lot V7) : LA REFERENCE 0, LUE COMME UNE UNITE.
//
// L'HYPOTHESE, ECRITE AVANT LA MESURE, ET D'OU ELLE VIENT. Le lecteur d'identifiant du moteur
// (`FUN_1406d3140`) ne lit une SONDE que pour le DOMAINE 1, et la sortie de vehicule montre a quoi
// elle sert : sonde = 1 rend un index de 9 bits, sonde = 0 un index de 13 bits. Le chantier lit
// jusqu'ici le cas sonde = 1 comme « index relatif a la bande bipede » (mesure : 237/237 sur les
// sorties, 95,5 % en bande). RESTE LE CAS SONDE = 0, JAMAIS OBSERVE SUR UN EVENEMENT VEHICULE.
//
// Or dans la taxonomie Halo, le BIPEDE et le VEHICULE sont deux specialisations d'UNITE. Si le
// domaine 1 est le domaine des UNITES et non celui des seuls bipedes, alors un evenement qui
// designe un VEHICULE le fait par une reference de domaine 1 — et la sonde est precisement ce qui
// distingue les deux adressages. C'est une hypothese, pas un acquis : elle se mesure ici.
//
// CE QUE L'INSTRUMENT LIT, ET POURQUOI IL EST PLUS FIN QUE `TestV7Refs`. Le balayage aveugle lit
// une valeur a un decalage FIXE, sans savoir si la reference est seulement PRESENTE : quand la
// garde est a zero, il lit les bits du champ suivant. Ici la reference est DECODEE — garde, sonde,
// largeur consequente — et les taux sont CONDITIONNES a sa presence. Un signal dilue par 70 % de
// references absentes redevient visible.
//
// LE CRITERE ET SES TEMOINS sont ceux du lot : le slot resolu tombe dans la bande `ti=40` ET
// l'instant du paquet tombe dans la FENETRE DE DISPARITION de cette vie (dernier recensement ..
// premiere preuve d'absence) ; temoin temporel a +60 s, temoin de cible sur la bande bipede.
//
// Garde d'environnement V7_ROOT / V7_FILMS : sans elle, tout SKIP.

import (
	"path/filepath"
	"sort"
	"testing"
)

// v7Dom1 accumule ce qu'on releve de la reference 0 d'un type, lue en domaine 1.
//
// LA COLONNE QUI COMPTE EST `vehEnd / veh` : parmi les evenements dont la reference 0 DESIGNE UN
// VEHICULE, la part qui tombe dans la fenetre de disparition de CE vehicule. Exiger que TOUS les
// evenements d'un type visent un vehicule serait un critere faux — un evenement de destruction
// d'OBJET vise aussi des grenades, des bidons et des bipedes. C'est la PURETE CONDITIONNELLE qui
// signe la destruction, et son temoin est le meme evenement teste 60 s plus tard.
type v7Dom1 struct {
	n              int
	absent         int // garde a zero
	sonde1, sonde0 int
	s1Biped, s1Veh int // sonde=1 : base bipede + index(9)
	s1Hors         int // sonde=1 et slot dans AUCUNE des deux bandes
	s1End, s1Shift int // sonde=1 ET cible vehicule : fenetre de disparition, reel et temoin
	// tSonde1 / tDansBande : temoin de cadrage tente (la meme reference relue UN BIT PLUS LOIN).
	//
	// IL EST DEGENERE, ET LA COLONNE EST GARDEE POUR LE DIRE. Mesure : 99,0 a 100,0 % pour les
	// types 0, 1, 7 et 36 — le meme taux que le reel. Ce n'est PAS un echec de la partition, c'est
	// un temoin inapplicable par construction : relue au bit 10, la garde tombe sur la SONDE
	// (= 1 pour ces types) et la sonde tombe sur le bit de poids fort de l'index vrai. Le temoin
	// ne retient donc QUE les references d'index >= 256, c'est-a-dire exactement les references
	// VEHICULE, et il lit ensuite `(index - 256) * 2`, qui retombe mecaniquement dans la bande
	// bipede. LE VRAI CHIFFRE EST AILLEURS : c'est `s1Veh / (s1Veh + s1Hors)`, la part des index
	// HORS bande bipede qui tombent dans la bande vehicule (99,6 a 100,0 % mesures, contre 3 a
	// 16 % attendus par hasard).
	tSonde1, tDansBande int
	s0Veh, s0Bip, s0Nu  int // sonde=0 : index(13) lu comme slot absolu
	s0End, s0Shift      int // sonde=0 ET cible vehicule : idem
	s0Vals              map[uint32]int
	// fracs porte, pour chaque evenement visant un vehicule VIVANT, la position relative de
	// l'instant DANS la vie visee (0 = naissance, 1 = derniere preuve de presence). Une
	// destruction se lit a 1,0 ; un degat se repartit.
	fracs []float64
	// cibles compte les vies distinctes visees, et evtParVie les evenements par vie.
	cibles map[string]int
}

func newV7Dom1() *v7Dom1 { return &v7Dom1{s0Vals: map[uint32]int{}, cibles: map[string]int{}} }

// v7Quartiles rend min / q1 / mediane / q3 / max d'une serie (triee sur place).
func v7Quartiles(v []float64) (lo, q1, med, q3, hi float64) {
	if len(v) == 0 {
		return 0, 0, 0, 0, 0
	}
	sort.Float64s(v)
	at := func(f float64) float64 {
		i := int(f * float64(len(v)-1))
		return v[i]
	}
	return v[0], at(0.25), at(0.5), at(0.75), v[len(v)-1]
}

// note enregistre la position RELATIVE de l'instant dans la vie visee, et la vie elle-meme.
func (d *v7Dom1) note(veh v7Bande, slot uint32, at uint64) {
	l, ok := veh.lifeAt(slot, at)
	if !ok || l.hi <= l.lo {
		return
	}
	d.fracs = append(d.fracs, float64(at-l.lo)/float64(l.hi-l.lo))
	d.cibles[itoa(int(slot))+"@"+itoa(int(l.last/1_000_000))]++
}

// v7Dom1Scan balaie un film et alimente le releve par type de tete.
func v7Dom1Scan(dir string, acc map[int]*v7Dom1) (int, int) {
	k := ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI)
	veh := v7BandeFrom(k)
	chunks := make([]int, 0, CountFilmChunks(dir))
	for i := 1; i <= CountFilmChunks(dir); i++ {
		chunks = append(chunks, i)
	}
	if len(chunks) == 0 {
		return 0, 0
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
			if !present {
				continue
			}
			d := acc[ty]
			if d == nil {
				d = newV7Dom1()
				acc[ty] = d
			}
			v7Dom1One(d, pay, p.TimestampUS, base, bip, veh)
		}
	}
	return len(bip), len(veh.band)
}

// v7Dom1One decode la reference 0 d'UN evenement et met a jour le releve.
func v7Dom1One(d *v7Dom1, pay []byte, at uint64, base uint32, bip map[uint32]bool, veh v7Bande) {
	d.n++
	if tr := readDom1Ref(pay, eventPayloadStartBit+1); tr.Present && tr.Sonde == 1 {
		d.tSonde1++
		if s := base + tr.Index; bip[s] || veh.band[s] {
			d.tDansBande++
		}
	}
	r := readDom1Ref(pay, eventPayloadStartBit)
	if !r.Present {
		d.absent++
		return
	}
	if r.Sonde == 1 {
		d.sonde1++
		slot := base + r.Index
		if bip[slot] {
			d.s1Biped++
		}
		if veh.band[slot] {
			d.s1Veh++
			if veh.ending(slot, at) {
				d.s1End++
			}
			if veh.ending(slot, at+v7ShiftUS) {
				d.s1Shift++
			}
			d.note(veh, slot, at)
		}
		if !bip[slot] && !veh.band[slot] {
			d.s1Hors++
		}
		return
	}
	d.sonde0++
	slot := r.Index
	d.s0Vals[slot]++
	switch {
	case veh.band[slot]:
		d.s0Veh++
		if veh.ending(slot, at) {
			d.s0End++
		}
		if veh.ending(slot, at+v7ShiftUS) {
			d.s0Shift++
		}
	case bip[slot]:
		d.s0Bip++
	default:
		d.s0Nu++
	}
}

// TestV7Dom1 — LA TABLE. Une ligne par type de tete.
func TestV7Dom1(t *testing.T) {
	dirs := v7FilmDirs(t)
	acc := map[int]*v7Dom1{}
	for _, dir := range dirs {
		nb, nv := v7Dom1Scan(dir, acc)
		t.Logf("film %-10s bande bipede %3d · bande vehicule %3d",
			filepath.Base(filepath.Clean(dir)), nb, nv)
	}
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 DOM1 — reference 0 lue en domaine 1 (garde, sonde, largeur consequente) ==")
	t.Logf("%-5s %-7s %-7s %-7s | sonde=1 : %-6s %-5s %-5s %-7s %-7s %-8s | sonde=0 : %-5s %-5s %-5s %-7s %-7s",
		"type", "n", "absente", "sonde1", "bipede", "VEH", "hors", "FIN|veh", "tem+60",
		"TEMOIN+1b", "veh", "bip", "hors", "FIN|veh", "tem+60")
	for _, ty := range tys {
		d := acc[ty]
		if d.n == 0 {
			continue
		}
		p := func(n, tot int) float64 {
			if tot == 0 {
				return 0
			}
			return 100 * float64(n) / float64(tot)
		}
		t.Logf("%-5d %-7d %6.1f%% %6.1f%% | %5.1f%% %5d %5d %6.1f%% %6.1f%% %7.1f%% | %5d %5d %5d %6.1f%% %6.1f%%",
			ty, d.n, p(d.absent, d.n), p(d.sonde1, d.n),
			p(d.s1Biped, d.sonde1), d.s1Veh, d.s1Hors, p(d.s1End, d.s1Veh), p(d.s1Shift, d.s1Veh),
			p(d.tDansBande, d.tSonde1),
			d.s0Veh, d.s0Bip, d.s0Nu, p(d.s0End, d.s0Veh), p(d.s0Shift, d.s0Veh))
	}
	t.Logf("-- position RELATIVE dans la vie visee (0 = naissance, 1 = derniere preuve) --")
	for _, ty := range tys {
		d := acc[ty]
		if len(d.fracs) < 5 {
			continue
		}
		lo, q1, med, q3, hi := v7Quartiles(d.fracs)
		t.Logf("type %-4d n=%-5d cibles=%-4d evts/cible=%.1f · min %.2f q1 %.2f MED %.2f q3 %.2f max %.2f",
			ty, len(d.fracs), len(d.cibles),
			float64(len(d.fracs))/float64(v7Max(len(d.cibles), 1)), lo, q1, med, q3, hi)
	}
}
