package filmdec

// vehicules_v8_refveh_test.go — INSTRUMENT (lot V8) : LA REFERENCE VEHICULE DES EVENEMENTS
// D OCCUPATION, RE-MESUREE SUR CORPUS.
//
// CE QU IL VERIFIE, ET POURQUOI IL EXISTE. Le lot V7 a mesure que la reference 1 de la SORTIE est
// le VEHICULE (105 / 105 sur 12 films, § 7 de son rapport) — mais il l a mesure avec un decodeur
// D INSTRUMENT, en chainant `readDom1Ref` a la main. Le lot V8 fait de cette lecture une donnee de
// PRODUCTION (`VehicleEvent.VehicleSlot`) : la mesure doit donc etre refaite PAR LE DECODEUR DE
// PRODUCTION, sans quoi rien ne garantit que les deux lisent la meme chose.
//
// IL REPOND AUSSI A LA QUESTION LAISSEE OUVERTE POUR L EMBARQUEMENT : ses trois references sont en
// domaines 2/3/7 (lecture Ghidra du 2026-09-02) et le lot V6 avait refute la ref 2 comme vehicule
// (0/22 en bande `ti=40`). A la lumiere de « domaine 1 = unites », l instrument RE-EXAMINE les
// trois emplacements — chaque valeur lue est testee BRUTE et RAPPORTEE A LA BASE bipede contre la
// bande `ti=40` —, et il essaie en plus la variante ou la ref 1 se lirait en DOMAINE 1 (avec
// sonde). Si l embarquement portait un slot `ti=40` mal attribue, il sortirait la.
//
// LES TEMOINS, ecrits avant la mesure :
//
//	CHANCE ANALYTIQUE — une valeur tiree au hasard dans l espace des index de 9 bits qui SORT de
//	  la bande bipede tombe dans la bande `ti=40` avec la probabilite `|bande veh| / (512 - |bande
//	  bipede|)`, soit 3 a 16 % selon le film. C est ce chiffre que 100 % doit ecraser ;
//	PERMUTATION — chaque sortie est appariee au vehicule de la sortie SUIVANTE du meme film. Le
//	  taux d appartenance a la bande ne bouge pas (les deux tirent dans la meme bande, c est
//	  attendu), mais le taux de RESOLUTION D UNE VIE VIVANTE A L INSTANT, lui, doit chuter : c est
//	  le temoin qui juge la resolution `(slot, instant) -> vie`.
//
// Garde d environnement V7_ROOT / V7_FILMS (le harnais du chantier) : sans elle, tout SKIP.

import (
	"fmt"
	"path/filepath"
	"testing"
)

// v8Vie est une vie recensee d un slot `ti=40` : sa generation et sa fenetre de presence, meme
// regle que `replay.vehicleLife` (tolerance de recensement de ~20 s).
type v8Vie struct {
	gen    uint32
	lo, hi uint64
}

// v8ViesParSlot rend les vies recensees, indexees par slot.
func v8ViesParSlot(k WorldObjectKeyframes) map[uint32][]v8Vie {
	out := map[uint32][]v8Vie{}
	for key, seen := range k.SeenUS {
		if len(seen) == 0 {
			continue
		}
		v := v8Vie{gen: key.Gen, lo: 0, hi: 0}
		if seen[0] > v7CensusTolUS {
			v.lo = seen[0] - v7CensusTolUS
		}
		last := seen[len(seen)-1]
		v.hi = v7FirstAfter(k.TimesUS, last)
		if v.hi == 0 {
			v.hi = last + v7CensusTolUS
		}
		out[key.Slot] = append(out[key.Slot], v)
	}
	return out
}

// v8Compte est le depouillement d un film pour UN type d evenement.
type v8Compte struct {
	n, refPresente          int
	enBandeVeh, enBandeBip  int
	horsDesDeuxBandes       int
	vieTrouvee, vieMultiple int
	genAccord               int
	permVie, permGen        int
	// horsFenetreMaxS est le plus grand ecart (en secondes) entre l instant d une sortie et la
	// fenetre de vie la plus proche de son slot, parmi les sorties qu AUCUNE fenetre ne couvre.
	horsFenetre     int
	horsFenetreMaxS float64
	// genVies / genRefs sont les histogrammes des generations (2 bits) : celles des vies
	// recensees de la bande `ti=40`, et celles lues dans les references de sortie. Ils sont
	// publies parce que SANS EUX l accord des generations ne se juge pas — une generation
	// quasi constante rendrait l accord vrai sans rien prouver.
	genVies, genRefs [4]int
	// slotsPlusieursVies compte les slots `ti=40` qui portent PLUS D UNE vie recensee : ce sont
	// les seuls ou `(slot, gen)` peut differer de `slot`.
	slotsPlusieursVies, slotsBande int
}

// v8Ajoute classe une reference vehicule et sa resolution en vie.
func (c *v8Compte) ajoute(slot uint32, valide bool, gen uint32, at uint64,
	veh map[uint32]bool, bip map[uint32]bool, vies map[uint32][]v8Vie) {
	c.n++
	if !valide {
		return
	}
	c.refPresente++
	c.genRefs[gen&3]++
	switch {
	case veh[slot]:
		c.enBandeVeh++
	case bip[slot]:
		c.enBandeBip++
	default:
		c.horsDesDeuxBandes++
	}
	n, g := 0, uint32(0)
	for _, v := range vies[slot] {
		if at >= v.lo && at <= v.hi {
			n, g = n+1, v.gen
		}
	}
	if n >= 1 {
		c.vieTrouvee++
		if g == gen {
			c.genAccord++
		}
	}
	if n > 1 {
		c.vieMultiple++
	}
	if n == 0 {
		c.horsFenetre++
		if d := v8EcartFenetre(vies[slot], at); d > c.horsFenetreMaxS {
			c.horsFenetreMaxS = d
		}
	}
}

// v8EcartFenetre rend l ecart EN SECONDES entre un instant et la fenetre de vie la plus proche du
// slot, ou -1 quand le slot n a aucune vie recensee.
func v8EcartFenetre(vies []v8Vie, at uint64) float64 {
	best := -1.0
	for _, v := range vies {
		d := uint64(0)
		switch {
		case at < v.lo:
			d = v.lo - at
		case at > v.hi:
			d = at - v.hi
		}
		s := float64(d) / 1e6
		if best < 0 || s < best {
			best = s
		}
	}
	return best
}

// v8Board est le depouillement des TROIS emplacements de reference d un EMBARQUEMENT, chacun lu
// de quatre facons : valeur brute et valeur rapportee a la base, sous la grammaire portee
// (domaines 2/3/7) puis, pour la ref 1, sous la variante DOMAINE 1.
type v8Board struct {
	n          int
	presente   [3]int
	brutVeh    [3]int
	baseVeh    [3]int
	baseBip    [3]int
	d1Presente int
	d1Veh      int
	d1Bip      int
}

func (b *v8Board) ajoute(pay []byte, base uint32, veh, bip map[uint32]bool) {
	b.n++
	r := [3]guardedRef{}
	r[0], r[1], r[2] = boardRefs(pay)
	for i, x := range r {
		if !x.Present {
			continue
		}
		b.presente[i]++
		if veh[x.Index] {
			b.brutVeh[i]++
		}
		if veh[base+x.Index] {
			b.baseVeh[i]++
		}
		if bip[base+x.Index] {
			b.baseBip[i]++
		}
	}
	// VARIANTE : la ref 1 lue en DOMAINE 1 (garde, sonde, R(sonde?9:13), gen) — la lecture qui
	// vaut pour la SORTIE. Si l embarquement nommait son vehicule, ce serait le premier endroit.
	d1 := readDom1Ref(pay, r[0].EndBit)
	if !d1.Present {
		return
	}
	b.d1Presente++
	s := d1.Index
	if d1.Sonde == 1 {
		s = base + d1.Index
	}
	if veh[s] {
		b.d1Veh++
	}
	if bip[s] {
		b.d1Bip++
	}
}

// v8ScanFilm depouille un film : sorties (par le DECODEUR DE PRODUCTION) et embarquements (refs
// brutes). Rend aussi la chance analytique du film.
func v8ScanFilm(dir string) (sortie v8Compte, board v8Board, chance float64, ok bool) {
	kf := ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI)
	if len(kf.Band) == 0 {
		return sortie, board, 0, false
	}
	vies := v8ViesParSlot(kf)
	sortie.slotsBande = len(kf.Band)
	for _, vs := range vies {
		if len(vs) > 1 {
			sortie.slotsPlusieursVies++
		}
		for _, v := range vs {
			sortie.genVies[v.gen&3]++
		}
	}
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
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
	if d := 512 - len(bip); d > 0 {
		chance = 100 * float64(len(kf.Band)) / float64(d)
	}
	var slots, gens []uint32
	var instants []uint64
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(data)
			ev, decode := decodeVehicleEvent(pay, base, NewSlotBand(bip))
			if !decode {
				continue
			}
			if ev.Kind == EventBipedBoardVehicle {
				board.ajoute(pay, base, kf.Band, bip)
				continue
			}
			sortie.ajoute(ev.VehicleSlot, ev.VehicleSlotValid, ev.VehicleGen, p.TimestampUS,
				kf.Band, bip, vies)
			if ev.VehicleSlotValid {
				slots, gens = append(slots, ev.VehicleSlot), append(gens, ev.VehicleGen)
				instants = append(instants, p.TimestampUS)
			}
		}
	}
	// TEMOIN PAR PERMUTATION : le vehicule de la sortie SUIVANTE, au MEME instant. Deux colonnes,
	// et c est la SECONDE qui juge : « une vie couvre l instant » est un temoin FAIBLE (un
	// vehicule vit presque tout le film, donc n importe quel slot repond oui), tandis que
	// « la generation de la reference egale celle de la vie » ne peut etre satisfaite par un
	// appariement au hasard qu une fois sur quatre (la generation fait 2 bits).
	for i := range slots {
		j := (i + 1) % len(slots)
		for _, v := range vies[slots[j]] {
			if instants[i] >= v.lo && instants[i] <= v.hi {
				sortie.permVie++
				if v.gen == gens[i] {
					sortie.permGen++
				}
				break
			}
		}
	}
	return sortie, board, chance, true
}

// TestV8RefVehicule — LA TABLE. Une ligne par film pour la sortie, une pour l embarquement.
func TestV8RefVehicule(t *testing.T) {
	dirs := v7FilmDirs(t)
	var tot, totB = v8Compte{}, v8Board{}
	t.Logf("== V8 — LA SORTIE NOMME SON VEHICULE (decodeur de PRODUCTION) ==")
	t.Logf("%-10s %-6s %-8s %-8s %-8s %-7s %-8s %-8s %-8s %-8s", "film", "n", "refPres",
		"bandeVEH", "bandeBIP", "hors", "vie", "genOK", "PERMUT", "chance")
	for _, dir := range dirs {
		s, b, chance, ok := v8ScanFilm(dir)
		if !ok {
			t.Logf("%-10s : aucune bande ti=40 — saute", filepath.Base(filepath.Clean(dir)))
			continue
		}
		t.Logf("%-10s %-6d %-8d %-8d %-8d %-7d %-8d %-8d %-8d %6.1f %%",
			filepath.Base(filepath.Clean(dir)), s.n, s.refPresente, s.enBandeVeh, s.enBandeBip,
			s.horsDesDeuxBandes, s.vieTrouvee, s.genAccord, s.permVie, chance)
		tot.n += s.n
		tot.refPresente += s.refPresente
		tot.enBandeVeh += s.enBandeVeh
		tot.enBandeBip += s.enBandeBip
		tot.horsDesDeuxBandes += s.horsDesDeuxBandes
		tot.vieTrouvee += s.vieTrouvee
		tot.vieMultiple += s.vieMultiple
		tot.genAccord += s.genAccord
		tot.permVie += s.permVie
		tot.permGen += s.permGen
		tot.horsFenetre += s.horsFenetre
		if s.horsFenetreMaxS > tot.horsFenetreMaxS {
			tot.horsFenetreMaxS = s.horsFenetreMaxS
		}
		tot.slotsBande += s.slotsBande
		tot.slotsPlusieursVies += s.slotsPlusieursVies
		for g := 0; g < 4; g++ {
			tot.genVies[g] += s.genVies[g]
			tot.genRefs[g] += s.genRefs[g]
		}
		totB.n += b.n
		totB.d1Presente += b.d1Presente
		totB.d1Veh += b.d1Veh
		totB.d1Bip += b.d1Bip
		for i := 0; i < 3; i++ {
			totB.presente[i] += b.presente[i]
			totB.brutVeh[i] += b.brutVeh[i]
			totB.baseVeh[i] += b.baseVeh[i]
			totB.baseBip[i] += b.baseBip[i]
		}
	}
	pc := func(x, n int) string {
		if n == 0 {
			return "-"
		}
		return fmt.Sprintf("%d (%.1f %%)", x, 100*float64(x)/float64(n))
	}
	t.Logf("TOTAL sorties %d — ref presente %s · bande VEH %s · bande BIPEDE %s · hors %s",
		tot.n, pc(tot.refPresente, tot.n), pc(tot.enBandeVeh, tot.refPresente),
		pc(tot.enBandeBip, tot.refPresente), pc(tot.horsDesDeuxBandes, tot.refPresente))
	t.Logf("TOTAL resolution — vie recensee a l instant %s (dont fenetres MULTIPLES %d) · "+
		"generation de la ref = celle de la vie %s",
		pc(tot.vieTrouvee, tot.refPresente), tot.vieMultiple, pc(tot.genAccord, tot.vieTrouvee))
	t.Logf("TOTAL hors fenetre : %s — ecart maximal a la fenetre la plus proche %.1f s",
		pc(tot.horsFenetre, tot.refPresente), tot.horsFenetreMaxS)
	t.Logf("TEMOIN PERMUTATION (vehicule de la sortie suivante) — une vie couvre l instant %s "+
		"(temoin FAIBLE) · generation en accord %s (hasard attendu 25 %%)",
		pc(tot.permVie, tot.refPresente), pc(tot.permGen, tot.permVie))
	t.Logf("GENERATIONS (2 bits) — vies recensees %v · references de sortie %v · slots de bande %d "+
		"dont %d portent PLUS D UNE vie", tot.genVies, tot.genRefs, tot.slotsBande,
		tot.slotsPlusieursVies)
	t.Logf("== V8 — L EMBARQUEMENT PORTE-T-IL SON VEHICULE ? (n = %d) ==", totB.n)
	for i := 0; i < 3; i++ {
		t.Logf("  ref %d (domaine %s) : presente %s · valeur BRUTE en bande VEH %s · "+
			"base+valeur en bande VEH %s · base+valeur en bande BIPEDE %s",
			i, [3]string{"2", "3", "7"}[i], pc(totB.presente[i], totB.n),
			pc(totB.brutVeh[i], totB.presente[i]), pc(totB.baseVeh[i], totB.presente[i]),
			pc(totB.baseBip[i], totB.presente[i]))
	}
	t.Logf("  VARIANTE ref 1 lue en DOMAINE 1 : presente %s · bande VEH %s · bande BIPEDE %s",
		pc(totB.d1Presente, totB.n), pc(totB.d1Veh, totB.d1Presente),
		pc(totB.d1Bip, totB.d1Presente))
}
