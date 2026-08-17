package replay

// equipment_uses_join_test.go — LA JOINTURE du lot D phase 0 : rattacher les lectures de
// charge a une VIE D'OBJET IDENTIFIEE, en tirer les signaux candidats, et les confronter a
// l'oracle du grappin.
//
// POURQUOI UNE INSTANCE ET PAS LA SEULE PAIRE (slot, generation). La generation ne fait que
// 2 bits : un slot repasse par la meme paire au cours d'un match, et `confirmPlacements` le
// sait deja (sa cle de deduplication porte le debut de la vie). Rattacher une lecture a la
// paire seule fondrait deux objets distincts du meme socle en un — et fabriquerait des
// « transitions » entre deux objets qui n'ont jamais rien partage. La lecture va donc a la
// DERNIERE pose dont le record de creation la precede, et le comptage BRUT par paire est
// publie a cote : c'est l'ecart entre les deux qui dit ce que la confirmation coute.
//
// LE PARTAGE REEL / FANTOME EST CETTE JOINTURE MEME. L'en-tete de record d'objet du monde
// n'est pas selectif (equipment_placements.go) : une lecture dont la cle ne retombe sur
// AUCUNE vie confirmee par l'oracle de position est, au mieux, un objet dont la naissance
// n'a pas ete lue ; au pire du bruit d'ancrage. Les deux distributions sont publiees.
//
// UN DECREMENT EST DATE PAR SA SECONDE LECTURE, jamais par la premiere : c'est l'instant ou la
// nouvelle valeur est annoncee. La fenetre d'appariement (± 0,5 s) absorbe le pas de
// replication ; la dater a la premiere lecture la reculerait d'un pas inconnu.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// eqLife est une VIE d'objet d'equipement IDENTIFIEE : sa pose lui donne un GlobalID `eqip`,
// donc une famille, et ses lectures lui donnent des signaux datables.
type eqLife struct {
	key      filmdec.EquipmentLifeKey
	inst     int
	globalID uint32
	family   string
	t0US     uint64
	t1US     uint64
	samples  []filmdec.EquipmentStateSample
}

// eqSignal est UN instant candidat porte par une vie identifiee — le grain de l'appariement.
type eqSignal struct {
	life     int
	atUS     uint64
	from, to uint64
}

// eqUsesBuildLives rattache chaque lecture d'etat a la pose qui la precede sur la meme cle.
// Rend aussi, par indice de lecture, si elle a trouve une vie (le partage reel / fantome).
func eqUsesBuildLives(
	placements []filmdec.EquipmentPlacement, families map[uint32]string,
	samples []filmdec.EquipmentStateSample,
) ([]eqLife, []bool) {
	byKey := map[filmdec.EquipmentLifeKey][]int{}
	lives := make([]eqLife, 0, len(placements))
	sorted := append([]filmdec.EquipmentPlacement(nil), placements...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].T0US < sorted[j].T0US })
	for _, p := range sorted {
		fam := families[p.GlobalID]
		if fam == "" {
			fam = equipmentFamilyOther
		}
		lives = append(lives, eqLife{
			key: p.Life, inst: len(byKey[p.Life]), globalID: p.GlobalID, family: fam,
			t0US: p.T0US, t1US: p.T1US,
		})
		byKey[p.Life] = append(byKey[p.Life], len(lives)-1)
	}
	attached := make([]bool, len(samples))
	for i, s := range samples {
		k := filmdec.EquipmentLifeKey{Slot: s.Slot, Gen: s.Gen}
		idx := -1
		for _, li := range byKey[k] {
			if lives[li].t0US <= s.TimestampUS+eqUsesCreationSlackUS {
				idx = li
			}
		}
		if idx >= 0 {
			lives[idx].samples = append(lives[idx].samples, s)
			attached[i] = true
		}
	}
	for i := range lives {
		ss := lives[i].samples
		sort.SliceStable(ss, func(a, b int) bool { return ss[a].TimestampUS < ss[b].TimestampUS })
	}
	return lives, attached
}

// eqUsesLogLives publie ce que la jointure a rattache, famille par famille — le denominateur
// de tout le reste.
func eqUsesLogLives(t *testing.T, lives []eqLife) {
	t.Helper()
	type row struct{ vies, i27vies, i27lect, i26vies int }
	byFam := map[string]*row{}
	attached := 0
	for _, l := range lives {
		r := byFam[l.family]
		if r == nil {
			r = &row{}
			byFam[l.family] = r
		}
		r.vies++
		attached += len(l.samples)
		n := eqUsesCountField(l, filmdec.EquipCharges)
		r.i27lect += n
		if n > 0 {
			r.i27vies++
		}
		if eqUsesCountField(l, filmdec.EquipEnergyDelay) > 0 {
			r.i26vies++
		}
	}
	fams := make([]string, 0, len(byFam))
	for f := range byFam {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	t.Logf("== JOINTURE == %d vies d'objet IDENTIFIEES · %d lectures rattachees", len(lives), attached)
	for _, f := range fams {
		r := byFam[f]
		t.Logf("  %-22s %3d vies · %3d portent i27 (%d lectures) · %3d portent i26",
			f, r.vies, r.i27vies, r.i27lect, r.i26vies)
	}
}

func eqUsesCountField(l eqLife, f filmdec.EquipmentField) int {
	n := 0
	for _, s := range l.samples {
		if s.Present[f] {
			n++
		}
	}
	return n
}

// eqUsesChargeDrops extrait les DECROISSANCES de `charges-remaining` par vie identifiee.
func eqUsesChargeDrops(lives []eqLife) []eqSignal {
	return eqUsesSteps(lives, filmdec.EquipCharges, false)
}

// eqUsesDelayRises extrait les HAUSSES d'`energy-delay-ticks-left` : un compte a rebours qui
// REPART. C'est le repli prescrit par le plan quand les charges ne datent rien.
func eqUsesDelayRises(lives []eqLife) []eqSignal {
	return eqUsesSteps(lives, filmdec.EquipEnergyDelay, true)
}

// eqUsesSteps rend les marches d'un champ sur chaque vie : hausses si `up`, sinon baisses.
func eqUsesSteps(lives []eqLife, f filmdec.EquipmentField, up bool) []eqSignal {
	var out []eqSignal
	for i, l := range lives {
		prev, has := uint64(0), false
		for _, s := range l.samples {
			if !s.Present[f] {
				continue
			}
			v := s.Val[f]
			if has && ((up && v > prev) || (!up && v < prev)) {
				out = append(out, eqSignal{life: i, atUS: s.TimestampUS, from: prev, to: v})
			}
			prev, has = v, true
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].atUS < out[b].atUS })
	return out
}

// eqUsesBirths rend la NAISSANCE de chaque vie identifiee. Ce n'est pas un canal prescrit :
// c'est le controle qui explique un negatif. Le registre a deja refute les naissances ti=37
// TOUTES FAMILLES CONFONDUES (densite 4,7-5,3/s, temoins au niveau du reel) ; restreintes aux
// objets dont l'identite `eqip` est le grappin, elles sont mille fois plus rares, et le temoin
// decale dit alors si la coincidence vaut quelque chose.
func eqUsesBirths(lives []eqLife) []eqSignal {
	out := make([]eqSignal, 0, len(lives))
	for i, l := range lives {
		out = append(out, eqSignal{life: i, atUS: l.t0US})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].atUS < out[b].atUS })
	return out
}

// eqUsesFamilySig filtre des signaux par appartenance (ou non) a une famille.
func eqUsesFamilySig(sig []eqSignal, lives []eqLife, fam string, in bool) []eqSignal {
	var out []eqSignal
	for _, s := range sig {
		if (lives[s.life].family == fam) == in {
			out = append(out, s)
		}
	}
	return out
}

// eqUsesNear rend la vie du signal le plus proche de `at` dans la fenetre, s'il existe.
// `sig` est trie par instant : la recherche est bornee, pas un balayage complet.
func eqUsesNear(sig []eqSignal, at uint64) (int, bool) {
	lo := sort.Search(len(sig), func(k int) bool { return sig[k].atUS+eqUsesWindowUS >= at })
	best, bestD, ok := 0, uint64(0), false
	for k := lo; k < len(sig) && sig[k].atUS <= at+eqUsesWindowUS; k++ {
		d := eqUsesGap(sig[k].atUS, at)
		if !ok || d < bestD {
			best, bestD, ok = sig[k].life, d, true
		}
	}
	return best, ok
}

func eqUsesGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// eqPair est UN evenement de grappin apparie a un signal.
type eqPair struct {
	read int
	life int
}

// eqUsesOracle joue UN canal candidat contre l'oracle du grappin, avec ses deux temoins :
// (a) les memes evenements decales de +7 s, (b) les memes evenements contre les signaux des
// AUTRES familles. Le seuil est celui du plan, ecrit avant la mesure.
func eqUsesOracle(
	t *testing.T, nom string, reads []filmdec.GrappleRead, sig, autres []eqSignal,
) []eqPair {
	t.Helper()
	n, wit, cross := 0, 0, 0
	var pairs []eqPair
	for i, r := range reads {
		if li, ok := eqUsesNear(sig, r.TimestampUS); ok {
			n++
			pairs = append(pairs, eqPair{read: i, life: li})
		}
		if _, ok := eqUsesNear(sig, r.TimestampUS+eqUsesWitnessUS); ok {
			wit++
		}
		if _, ok := eqUsesNear(autres, r.TimestampUS); ok {
			cross++
		}
	}
	t.Logf("  canal %-28s %d signaux `grapple` · APPARIES %s -> %s · temoin (a) +%d s %s"+
		" · temoin (b) autres familles %s", nom, len(sig), eqUsesPct(n, len(reads)),
		eqUsesVerdict(len(reads) > 0 && float64(n) >= eqUsesPairMin*float64(len(reads))),
		eqUsesWitnessUS/1_000_000, eqUsesPct(wit, len(reads)), eqUsesPct(cross, len(reads)))
	return pairs
}

// eqUsesGrappleUses compte les GESTES : une paire tir/accroche a <= 0,5 s vaut un usage.
func eqUsesGrappleUses(reads []filmdec.GrappleRead) int {
	bySlot := map[uint32][]filmdec.GrappleRead{}
	for _, r := range reads {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	n := 0
	for _, list := range bySlot {
		sort.SliceStable(list, func(a, b int) bool { return list[a].TimestampUS < list[b].TimestampUS })
		var last uint64
		first := true
		for _, r := range list {
			if first || r.TimestampUS-last > eqUsesWindowUS {
				n++
			}
			last, first = r.TimestampUS, false
		}
	}
	return n
}

// eqUsesGrappleIDs rend les GlobalID `eqip` des objets de famille `grapple` du film.
func eqUsesGrappleIDs(lives []eqLife) (string, int) {
	ids := map[uint32]int{}
	n := 0
	for _, l := range lives {
		if l.family == eqUsesGrappleFamily {
			ids[l.globalID]++
			n++
		}
	}
	keys := make([]uint32, 0, len(ids))
	for k := range ids {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("0x%08x:%d vies", k, ids[k]))
	}
	return strings.Join(parts, " "), n
}

// eqUsesBridge joue D.0.3 : l'objet apparie a >= 2 evenements du MEME slot est « celui de S ».
// La coherence mesuree est l'absence de second pretendant sur la meme vie.
func eqUsesBridge(t *testing.T, nom string, pairs []eqPair, reads []filmdec.GrappleRead) {
	t.Helper()
	bySlot := map[int]map[uint32]int{}
	for _, p := range pairs {
		if bySlot[p.life] == nil {
			bySlot[p.life] = map[uint32]int{}
		}
		bySlot[p.life][reads[p.read].Slot]++
	}
	attribuees, coherentes, ambigues := 0, 0, 0
	for _, slots := range bySlot {
		cand := 0
		for _, n := range slots {
			if n >= 2 {
				cand++
			}
		}
		if cand == 0 {
			continue
		}
		attribuees++
		if cand == 1 {
			coherentes++
		} else {
			ambigues++
		}
	}
	t.Logf("  pont par %-24s %d vies appariees · %d attribuees (>= 2 evenements d'un meme"+
		" slot) · %d ambigues · COHERENCE %s -> %s", nom, len(bySlot), attribuees, ambigues,
		eqUsesPct(coherentes, attribuees),
		eqUsesVerdict(attribuees > 0 &&
			float64(coherentes) >= eqUsesCoherenceMin*float64(attribuees)))
}

// eqUsesGeneralise joue D.0.4 : usages par famille, et controle croise pose <-> decrement.
func eqUsesGeneralise(
	t *testing.T, lives []eqLife, drops []eqSignal,
	placements []filmdec.EquipmentPlacement, families map[uint32]string,
) {
	t.Helper()
	t.Log("== D.0.4 GENERALISATION == decroissances de charge par vie d'objet et par famille")
	perFam := map[string][]int{}
	count := map[int]int{}
	for _, d := range drops {
		count[d.life]++
	}
	for i, l := range lives {
		perFam[l.family] = append(perFam[l.family], count[i])
	}
	fams := make([]string, 0, len(perFam))
	for f := range perFam {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	for _, f := range fams {
		list := perFam[f]
		sort.Ints(list)
		tot, nz := 0, 0
		for _, n := range list {
			tot += n
			if n > 0 {
				nz++
			}
		}
		t.Logf("  %-22s %3d vies · %3d avec >= 1 decroissance · %4d decroissances · max/vie %d",
			f, len(list), nz, tot, list[len(list)-1])
	}
	eqUsesCross(t, drops, lives, placements, families)
}

// eqUsesCross confronte chaque POSE a une decroissance de la MEME famille : une pose de mur
// consomme une charge, ou bien la these ne tient pas.
func eqUsesCross(
	t *testing.T, drops []eqSignal, lives []eqLife,
	placements []filmdec.EquipmentPlacement, families map[uint32]string,
) {
	t.Helper()
	byFam := map[string][]filmdec.EquipmentPlacement{}
	for _, p := range placements {
		fam := families[p.GlobalID]
		if fam == "" {
			fam = equipmentFamilyOther
		}
		byFam[fam] = append(byFam[fam], p)
	}
	fams := make([]string, 0, len(byFam))
	for f := range byFam {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	t.Logf("  CONTROLE CROISE pose <-> decroissance de la meme famille (fenetre ± %d ms,"+
		" seuil %.0f %%) :", eqUsesWindowUS/1000, 100*eqUsesCrossMin)
	for _, f := range fams {
		fd := eqUsesFamilySig(drops, lives, f, true)
		if len(fd) == 0 {
			t.Logf("    %-22s %3d poses · AUCUNE decroissance dans cette famille :"+
				" non calculable", f, len(byFam[f]))
			continue
		}
		n, wit := 0, 0
		for _, p := range byFam[f] {
			if _, ok := eqUsesNear(fd, p.T0US); ok {
				n++
			}
			if _, ok := eqUsesNear(fd, p.T0US+eqUsesWitnessUS); ok {
				wit++
			}
		}
		t.Logf("    %-22s %s -> %s · temoin +%d s %s", f, eqUsesPct(n, len(byFam[f])),
			eqUsesVerdict(float64(n) >= eqUsesCrossMin*float64(len(byFam[f]))),
			eqUsesWitnessUS/1_000_000, eqUsesPct(wit, len(byFam[f])))
	}
}

// eqUsesEnergyDelay repond a la derniere question de D.0.4 : le compte a rebours d'i26
// DEBUTE-t-il au decrement ? Un delai qui repart a chaque usage MONTE au moment ou la charge
// tombe ; un delai qui ne fait que descendre ne date rien.
func eqUsesEnergyDelay(t *testing.T, lives []eqLife, drops, rises []eqSignal) {
	t.Helper()
	n := 0
	for _, d := range drops {
		for _, r := range rises {
			if r.life == d.life && eqUsesGap(r.atUS, d.atUS) <= eqUsesWindowUS {
				n++
				break
			}
		}
	}
	seul := 0
	byLife := map[int]bool{}
	for _, d := range drops {
		byLife[d.life] = true
	}
	for i := range lives {
		if !byLife[i] && eqUsesCountField(lives[i], filmdec.EquipEnergyDelay) > 0 {
			seul++
		}
	}
	t.Logf("== D.0.4 RELATION i26 <-> DECROISSANCE DE CHARGE == %s des decroissances sont"+
		" accompagnees d'une HAUSSE d'i26 sur la MEME vie dans la fenetre ± %d ms"+
		" · %d hausses d'i26 au total · %d vies portent i26 sans aucune decroissance de charge",
		eqUsesPct(n, len(drops)), eqUsesWindowUS/1000, len(rises), seul)
}

// eqUsesOwners attribue les poses a leur POSEUR — `equipmentOwner`, LA fonction de production.
// Sans bornes monde, la distance n'est pas une distance : la mesure est declaree non
// calculable plutot que rendue dans une unite muette.
func eqUsesOwners(
	t *testing.T, pos []filmdec.BipedPosition, placements []filmdec.EquipmentPlacement,
	families map[uint32]string,
) {
	t.Helper()
	if len(pos) == 0 {
		t.Logf("== D.0.4 POSES PAR POSEUR == NON CALCULABLE : le poseur (`equipmentOwner`,"+
			" seuil de proximite en METRES) exige les bornes de la carte. Renseigner %s"+
			" (et %s si la signature de largeurs est ambigue).", eqUsesBoundsEnv, eqUsesMapEnv)
		return
	}
	perSlot := map[uint32]map[string]int{}
	sans := 0
	for _, p := range placements {
		fam := families[p.GlobalID]
		if fam == "" {
			fam = equipmentFamilyOther
		}
		slot, _, ok := equipmentOwner(pos, p)
		if !ok {
			sans++
			continue
		}
		if perSlot[slot] == nil {
			perSlot[slot] = map[string]int{}
		}
		perSlot[slot][fam]++
	}
	totals := map[string]int{}
	for _, m := range perSlot {
		for f, n := range m {
			totals[f] += n
		}
	}
	t.Logf("== D.0.4 POSES PAR POSEUR == %d slots poseurs (VIES, pas joueurs) · %d poses sans"+
		" poseur a portee · total par famille : %s", len(perSlot), sans, eqUsesFamLine(totals))
	t.Logf("  mediane de poses par slot poseur : %.1f", eqUsesMedianePoses(perSlot))
}

// eqUsesMedianePoses rend la mediane du nombre de poses par slot poseur.
func eqUsesMedianePoses(perSlot map[uint32]map[string]int) float64 {
	var counts []int
	for _, m := range perSlot {
		n := 0
		for _, v := range m {
			n += v
		}
		counts = append(counts, n)
	}
	if len(counts) == 0 {
		return 0
	}
	sort.Ints(counts)
	return float64(counts[len(counts)/2])
}

func eqUsesFamLine(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// eqUsesWriteTSV depose les pieces justificatives. Sans EQ_OUT, rien n'est ecrit : un
// instrument de mesure n'a pas a semer des fichiers dans le depot de qui le lance.
func eqUsesWriteTSV(t *testing.T, short string, lives []eqLife, drops []eqSignal) {
	t.Helper()
	out := os.Getenv(eqUsesOutEnv)
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("dossier de sortie %s : %v", out, err)
	}
	count := map[int]int{}
	for _, d := range drops {
		count[d.life]++
	}
	var b strings.Builder
	b.WriteString("slot\tgen\tinstance\tglobal_id\tfamille\tt0_us\tt1_us\tlectures\t" +
		"lectures_i27\tmin_i27\tmax_i27\tlectures_i26\tdecroissances_i27\n")
	for i, l := range lives {
		n, lo, hi := 0, uint64(0), uint64(0)
		for _, s := range l.samples {
			if !s.Present[filmdec.EquipCharges] {
				continue
			}
			v := s.Val[filmdec.EquipCharges]
			if n == 0 || v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
			n++
		}
		fmt.Fprintf(&b, "%d\t%d\t%d\t0x%08x\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			l.key.Slot, l.key.Gen, l.inst, l.globalID, l.family, l.t0US, l.t1US,
			len(l.samples), n, lo, hi, eqUsesCountField(l, filmdec.EquipEnergyDelay), count[i])
	}
	eqUsesWriteFile(t, filepath.Join(out, short+"_vies.tsv"), b.String())

	var d strings.Builder
	d.WriteString("at_us\tslot\tgen\tinstance\tfamille\tglobal_id\tde\tvers\n")
	for _, x := range drops {
		l := lives[x.life]
		fmt.Fprintf(&d, "%d\t%d\t%d\t%d\t%s\t0x%08x\t%d\t%d\n",
			x.atUS, l.key.Slot, l.key.Gen, l.inst, l.family, l.globalID, x.from, x.to)
	}
	eqUsesWriteFile(t, filepath.Join(out, short+"_decroissances.tsv"), d.String())
}

func eqUsesWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", path, err)
	}
	t.Logf("  piece ecrite : %s", path)
}
