// tmp_cecapture — LIRE la capture CE brute (40 o par composant dispatche) et en extraire la
// VERITE que notre decodeur hors ligne doit reproduire.
//
// CE QUE LA CAPTURE EST. Un crochet pose sur le dispatch des composants (0x14076CD11,
// juste avant `call [rax+28]`) journalise, pour CHAQUE composant que le jeu deserialise :
// l'entite, l'archetype, l'index du composant, le curseur de bits AVANT lecture, et 16
// octets bruts lus depuis le tampon du film. Rien n'est infere : c'est le moteur lui-meme
// qui parle.
//
// LES DEUX GRANDEURS QUI EN SORTENT, et pourquoi elles sont decisives :
//
//  1. LE MASQUE. L'ensemble des compIndex vus pour une meme entite dans un meme record EST
//     le masque de presence. C'est la grandeur qu'on n'avait jamais mesuree, et c'est la
//     ou est la faute : la verite dit i22 present dans 0,19 %% des records de bipede, notre
//     decodeur le lit dans 12 %% — un exces de 63 fois qu'aucune correction de largeur ne
//     peut produire. On cherchait des largeurs, le defaut est dans la LISTE.
//
//  2. LES LARGEURS. La difference entre le curseur du composant N+1 et celui du composant N
//     EST la largeur consommee par le composant N, exactement. Mesuree, pas portee depuis
//     Ghidra.
//
// DECOUPAGE EN RECORDS. Le crochet ne voit pas les frontieres de record ; on les reconstruit
// par la regle : un nouveau record commence quand l'entite change OU quand le curseur
// recule (le lecteur est reparti en arriere, donc c'est un autre flux). Cette regle est
// conservatrice — elle peut FUSIONNER deux records consecutifs de la meme entite dont les
// curseurs s'enchainent, jamais en couper un en deux.
//
// CE QUE CET OUTIL NE FAIT PAS : il ne corrige rien. Il mesure et publie. La correction du
// decodeur vient apres, sur pieces.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// recSize est la taille d'un enregistrement de capture, fixee par le cave assembleur.
const recSize = 40

// hit est un composant dispatche, tel que le moteur l'a journalise.
type hit struct {
	EID       uint32
	TypeIndex uint32
	CompIndex uint32
	Param4    uint32
	BitCursor uint32
	SkipCount uint32
	Sig       [16]byte
}

// record est une suite de composants appartenant a la meme entite et au meme flux.
type record struct {
	EID       uint32
	TypeIndex uint32
	Hits      []hit
}

// Mask rend la liste triee des composants presents.
func (r record) Mask() []uint32 {
	out := make([]uint32, 0, len(r.Hits))
	seen := map[uint32]bool{}
	for _, h := range r.Hits {
		if !seen[h.CompIndex] {
			seen[h.CompIndex] = true
			out = append(out, h.CompIndex)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func main() {
	path := flag.String("in", "", "dump binaire du tampon de capture CE")
	count := flag.Int("n", 0, "nombre de records valides (0 = deduire du contenu)")
	focusTI := flag.Int("ti", 35, "archetype a detailler (35 = bipede)")
	worldOut := flag.String("worlddump", "", "ecrire un world_dump_full.txt (entite:archetype) a ce chemin et sortir")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_cecapture -in <dump.bin> [-n N] [-ti 35]")
		os.Exit(2)
	}

	hits, err := readHits(*path, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("CAPTURE CE — %s\n\n", *path)
	fmt.Printf("  %d composants dispatches\n", len(hits))
	if len(hits) == 0 {
		os.Exit(1)
	}

	recs := splitRecords(hits)
	fmt.Printf("  %d records reconstruits\n\n", len(recs))

	if *worldOut != "" {
		if err := writeWorldDump(hits, *worldOut); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	reportArchetypes(recs)
	reportMasks(recs, uint32(*focusTI))
	reportWidths(hits, uint32(*focusTI))
	reportInventoryLocus(recs, uint32(*focusTI))
	reportRecordKind(recs, uint32(*focusTI))
	reportPresence(recs, uint32(*focusTI))
}

// readHits lit le dump binaire. Les enregistrements au-dela du compteur sont des zeros
// laisses par l'allocation : on s'arrete au premier enregistrement entierement nul, sauf si
// l'appelant impose un compte explicite.
func readHits(path string, n int) ([]hit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	total := len(raw) / recSize
	if n > 0 && n < total {
		total = n
	}
	out := make([]hit, 0, total)
	for i := 0; i < total; i++ {
		b := raw[i*recSize : (i+1)*recSize]
		var h hit
		h.EID = binary.LittleEndian.Uint32(b[0:])
		h.TypeIndex = binary.LittleEndian.Uint32(b[4:])
		h.CompIndex = binary.LittleEndian.Uint32(b[8:])
		h.Param4 = binary.LittleEndian.Uint32(b[12:])
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		h.SkipCount = binary.LittleEndian.Uint32(b[20:])
		copy(h.Sig[:], b[24:40])
		if n == 0 && h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break // queue non ecrite
		}
		out = append(out, h)
	}
	return out, nil
}

// splitRecords reconstruit les frontieres de record. Voir l'en-tete pour la regle et sa
// limite (fusion possible, jamais de coupure abusive).
func splitRecords(hits []hit) []record {
	var out []record
	var cur *record
	for _, h := range hits {
		newRec := cur == nil || cur.EID != h.EID || h.BitCursor < lastCursor(*cur)
		if newRec {
			if cur != nil {
				out = append(out, *cur)
			}
			cur = &record{EID: h.EID, TypeIndex: h.TypeIndex}
		}
		cur.Hits = append(cur.Hits, h)
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

func lastCursor(r record) uint32 {
	if len(r.Hits) == 0 {
		return 0
	}
	return r.Hits[len(r.Hits)-1].BitCursor
}

// reportArchetypes publie le poids de chaque archetype — quelle part du flux chacun occupe.
func reportArchetypes(recs []record) {
	byTI := map[uint32]int{}
	compsByTI := map[uint32]int{}
	for _, r := range recs {
		byTI[r.TypeIndex]++
		compsByTI[r.TypeIndex] += len(r.Hits)
	}
	fmt.Println("  ARCHETYPES (typeIndex) —")
	fmt.Printf("  %8s  %10s  %12s  %s\n", "ti", "records", "composants", "moy. comp/record")
	tis := keysOf(byTI)
	sort.Slice(tis, func(i, j int) bool { return byTI[tis[i]] > byTI[tis[j]] })
	for _, ti := range tis {
		fmt.Printf("  %8d  %10d  %12d  %.2f\n",
			ti, byTI[ti], compsByTI[ti], float64(compsByTI[ti])/float64(byTI[ti]))
	}
	fmt.Println()
}

// reportMasks publie les masques les plus frequents de l'archetype vise. C'EST LA MESURE
// CENTRALE : si notre decodeur invente des composants, c'est ici que ca se voit.
func reportMasks(recs []record, ti uint32) {
	freq := map[string]int{}
	n := 0
	for _, r := range recs {
		if r.TypeIndex != ti {
			continue
		}
		n++
		freq[fmt.Sprint(r.Mask())]++
	}
	if n == 0 {
		fmt.Printf("  MASQUES ti=%d — aucun record de cet archetype.\n\n", ti)
		return
	}
	fmt.Printf("  MASQUES les plus frequents de ti=%d (%d records) —\n", ti, n)
	type kv struct {
		k string
		v int
	}
	var rows []kv
	for k, v := range freq {
		rows = append(rows, kv{k, v})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].v > rows[j].v })
	for i, r := range rows {
		if i >= 15 {
			fmt.Printf("  … %d autres masques distincts\n", len(rows)-15)
			break
		}
		fmt.Printf("  %6.2f %%  %5d  %s\n", 100*float64(r.v)/float64(n), r.v, r.k)
	}
	fmt.Printf("  -> %d masques DISTINCTS sur %d records\n\n", len(rows), n)
	reportMaskSizes(recs, ti, n)
}

// reportMaskSizes publie la distribution des TAILLES de masque. C'est le temoin qui
// depart les deux branches de consumeMask : la branche eparse porte un compteur de 3 bits
// et ne peut donc excéder 7 composants ; la branche dense (R(64)) en poserait ~32 en
// moyenne. Si la verite ne depasse jamais 7 pour cet archetype, toute lecture dense sur
// un record de cet archetype est une faute, pas une variante legitime.
func reportMaskSizes(recs []record, ti uint32, n int) {
	hist := map[int]int{}
	over, maxSz := 0, 0
	for _, r := range recs {
		if r.TypeIndex != ti {
			continue
		}
		s := len(r.Mask())
		hist[s]++
		if s > 7 {
			over++
		}
		if s > maxSz {
			maxSz = s
		}
	}
	fmt.Printf("  TAILLES DE MASQUE ti=%d — max observe %d composants\n", ti, maxSz)
	sizes := make([]int, 0, len(hist))
	for s := range hist {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	for _, s := range sizes {
		bar := ""
		w := hist[s] * 40 / n
		for i := 0; i < w; i++ {
			bar += "#"
		}
		fmt.Printf("  %3d comp  %7d  %6.2f %%  %s\n",
			s, hist[s], 100*float64(hist[s])/float64(n), bar)
	}
	fmt.Printf("  -> AU-DELA DE 7 (impossible en branche eparse) : %d records = %.3f %%\n\n",
		over, 100*float64(over)/float64(n))
}

// reportWidths publie, par composant, la largeur consommee mesuree par difference de
// curseurs. Une largeur CONSTANTE signe un composant a grammaire fixe ; une largeur
// variable signe une grammaire a portes.
func reportWidths(hits []hit, ti uint32) {
	// largeur du composant courant = curseur du suivant - curseur du courant, a condition
	// que le suivant appartienne au meme flux (meme entite, curseur qui avance).
	widths := map[uint32]map[int]int{}
	for i := 0; i+1 < len(hits); i++ {
		a, b := hits[i], hits[i+1]
		if a.TypeIndex != ti || a.EID != b.EID || b.BitCursor < a.BitCursor {
			continue
		}
		w := int(b.BitCursor - a.BitCursor)
		if w > 4096 {
			continue // frontiere de record mal reconstruite : on ne pollue pas la mesure
		}
		if widths[a.CompIndex] == nil {
			widths[a.CompIndex] = map[int]int{}
		}
		widths[a.CompIndex][w]++
	}
	if len(widths) == 0 {
		fmt.Printf("  LARGEURS ti=%d — pas de couple mesurable.\n\n", ti)
		return
	}
	fmt.Printf("  LARGEURS MESUREES ti=%d (bits consommes par composant) —\n", ti)
	fmt.Printf("  %6s  %8s  %10s  %s\n", "comp", "n", "distinctes", "les plus frequentes")
	comps := keysOf(widths)
	sort.Slice(comps, func(i, j int) bool { return comps[i] < comps[j] })
	for _, c := range comps {
		m := widths[c]
		tot := 0
		for _, v := range m {
			tot += v
		}
		type kv struct{ w, n int }
		var rows []kv
		for w, n := range m {
			rows = append(rows, kv{w, n})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
		s := ""
		for i, r := range rows {
			if i >= 4 {
				s += " …"
				break
			}
			if i > 0 {
				s += "  "
			}
			s += fmt.Sprintf("%d bits (%.0f %%)", r.w, 100*float64(r.n)/float64(tot))
		}
		fmt.Printf("  i%-5d  %8d  %10d  %s\n", c, tot, len(m), s)
	}
	fmt.Println()
}

// reportPresence publie le taux de presence de chaque composant — la grandeur a confronter
// directement au decodeur hors ligne.
func reportPresence(recs []record, ti uint32) {
	present := map[uint32]int{}
	n := 0
	for _, r := range recs {
		if r.TypeIndex != ti {
			continue
		}
		n++
		for _, c := range r.Mask() {
			present[c]++
		}
	}
	if n == 0 {
		return
	}
	fmt.Printf("  TAUX DE PRESENCE par composant, ti=%d (%d records) —\n", ti, n)
	fmt.Printf("  %6s  %10s  %10s  %s\n", "comp", "records", "taux", "")
	comps := keysOf(present)
	sort.Slice(comps, func(i, j int) bool { return present[comps[i]] > present[comps[j]] })
	for _, c := range comps {
		note := ""
		switch c {
		case 22:
			note = "  <- unit-grenade-counts (COMPTES DE GRENADES)"
		case 47:
			note = "  <- biped-desired-grenade-set (TYPES EQUIPES)"
		case 48:
			note = "  <- biped-desired-ability-set (CAPACITE)"
		case 42:
			note = "  <- biped-desired-weapon-set (ARME)"
		case 56:
			note = "  <- biped-spartan-ability-energy"
		}
		fmt.Printf("  i%-5d  %10d  %9.2f %%%s\n",
			c, present[c], 100*float64(present[c])/float64(n), note)
	}
	fmt.Println()
}

// reportInventoryLocus teste UNE hypothese precise : les composants d'inventaire
// n'apparaissent QUE dans les records a masque dense (> 7 composants), c'est-a-dire les
// retransmissions d'etat complet au spawn.
//
// LE TEMOIN. Si l'hypothese est vraie, la part des records denses parmi ceux qui portent
// i22 doit ecraser la part des records denses dans la population generale (0,12 %). Un
// composant qui serait reparti au hasard afficherait la meme part que la population.
// C'est un rapport de vraisemblance, pas une simple coincidence de comptes.
func reportInventoryLocus(recs []record, ti uint32) {
	type stat struct{ total, dense int }
	watch := map[uint32]string{
		22: "unit-grenade-counts", 47: "biped-desired-grenade-set",
		48: "biped-desired-ability-set", 42: "biped-desired-weapon-set",
		56: "biped-spartan-ability-energy", 43: "weapon-state-type-info",
		0:  "object-position (temoin: composant COMMUN)",
		25: "unit-command-tick (temoin: composant COMMUN)",
	}
	st := map[uint32]*stat{}
	popTotal, popDense := 0, 0
	for _, r := range recs {
		if r.TypeIndex != ti {
			continue
		}
		m := r.Mask()
		dense := len(m) > 7
		popTotal++
		if dense {
			popDense++
		}
		for _, c := range m {
			if _, ok := watch[c]; !ok {
				continue
			}
			if st[c] == nil {
				st[c] = &stat{}
			}
			st[c].total++
			if dense {
				st[c].dense++
			}
		}
	}
	if popTotal == 0 {
		return
	}
	base := 100 * float64(popDense) / float64(popTotal)
	fmt.Printf("  LOCUS DE L'INVENTAIRE ti=%d — part de records DENSES (>7 comp)\n", ti)
	fmt.Printf("  population generale : %.3f %% (%d/%d)\n\n", base, popDense, popTotal)
	fmt.Printf("  %6s  %8s  %8s  %9s  %s\n", "comp", "records", "denses", "part", "composant")
	cs := keysOf(st)
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
	for _, c := range cs {
		s := st[c]
		fmt.Printf("  i%-5d  %8d  %8d  %8.2f %%  %s\n",
			c, s.total, s.dense, 100*float64(s.dense)/float64(s.total), watch[c])
	}
	fmt.Println()
}

// reportRecordKind teste l'hypothese qui commande le correctif : les records a masque large
// ne sont PAS des deltas a masque dense, ce sont des records NEW (nouvelle entite, etat
// complet transmis) — autrement dit des reapparitions de joueur.
//
// LE DISCRIMINANT DISPONIBLE. Le crochet ne voit pas le type de record, mais il capture
// param_4 (le recordStateParam passe a chaque deserialiseur). Si NEW et DELTA se
// distinguent par ce parametre, sa distribution differera entre records larges et records
// courts. Si elle est IDENTIQUE, l'hypothese NEW tombe et il faut vraiment chercher une
// branche dense.
//
// C'est un test refutable : une distribution identique refute, pas « n'apporte rien ».
func reportRecordKind(recs []record, ti uint32) {
	large := map[uint32]int{}
	small := map[uint32]int{}
	nL, nS := 0, 0
	for _, r := range recs {
		if r.TypeIndex != ti {
			continue
		}
		dst, n := small, &nS
		if len(r.Mask()) > 7 {
			dst, n = large, &nL
		}
		*n++
		for _, h := range r.Hits {
			dst[h.Param4]++
		}
	}
	if nL == 0 || nS == 0 {
		return
	}
	fmt.Printf("  NATURE DES RECORDS LARGES ti=%d — distribution de param_4\n", ti)
	fmt.Printf("  %d records larges (>7 comp) contre %d courts\n\n", nL, nS)
	fmt.Printf("  %8s  %14s  %14s\n", "param_4", "larges", "courts")
	seen := map[uint32]bool{}
	for k := range large {
		seen[k] = true
	}
	for k := range small {
		seen[k] = true
	}
	ks := keysOf(seen)
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	totL, totS := sumOf(large), sumOf(small)
	for _, k := range ks {
		fmt.Printf("  %8d  %7d %5.1f %%  %7d %5.1f %%\n", k,
			large[k], 100*float64(large[k])/float64(totL),
			small[k], 100*float64(small[k])/float64(totS))
	}
	fmt.Println()
}

func sumOf(m map[uint32]int) int {
	t := 0
	for _, v := range m {
		t += v
	}
	if t == 0 {
		return 1
	}
	return t
}

// writeWorldDump ecrit la table entite -> archetype au format `world_dump_full.txt`, celui
// que le decodeur hors ligne sait deja charger.
//
// POURQUOI C'EST DECISIF. Un record DELTA ne porte PAS son typeIndex : l'archetype vient du
// World, c'est-a-dire de la table slot -> archetype. Le bootstrap hors ligne
// (`WalkKeyframeWorld`) ne lie que 123 slots sur le film ancre — il cherche des ancres
// valides de facon heuristique et s'arrete a la premiere invalide. Resultat mesure : ZERO
// record de bipede decode, la ou le moteur en traite 159 772.
//
// La capture CE porte cette table gratuitement : chaque composant journalise l'entite ET son
// archetype, lus dans la RAM du moteur. On prend le MODE par entite (et non la premiere
// valeur vue) pour qu'un enregistrement aberrant isole ne fixe pas un archetype faux.
//
// CE QUE CE DUMP EST, ET N'EST PAS : c'est un ORACLE, pas une solution. Il debloque le
// decodage du film capture et il donne le temoin exact contre lequel mesurer
// `WalkKeyframeWorld`. Il ne rend PAS le rejeu hors ligne possible sur les 948 autres films —
// pour ceux-la il faudra reparer le bootstrap keyframe, et ce dump est precisement ce qui
// permettra de savoir quels slots il rate.
func writeWorldDump(hits []hit, path string) error {
	byEID := map[uint32]map[uint32]int{}
	for _, h := range hits {
		if byEID[h.EID] == nil {
			byEID[h.EID] = map[uint32]int{}
		}
		byEID[h.EID][h.TypeIndex]++
	}
	ids := keysOf(byEID)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var b strings.Builder
	fmt.Fprintf(&b, "# world_dump derive de la capture CE du dispatch (entite:archetype modal).\n")
	fmt.Fprintf(&b, "# %d entites, %d composants journalises.\n", len(ids), len(hits))
	ambig := 0
	for _, id := range ids {
		best, bestN, tot := uint32(0), 0, 0
		for ti, n := range byEID[id] {
			tot += n
			if n > bestN || (n == bestN && ti < best) {
				bestN, best = n, ti
			}
		}
		if bestN < tot {
			ambig++
		}
		fmt.Fprintf(&b, "%d:%d\n", id, best)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("ecriture %s : %w", path, err)
	}
	fmt.Printf("  world_dump -> %s\n", path)
	fmt.Printf("  %d entites liees, dont %d a archetype non unanime (mode retenu)\n", len(ids), ambig)
	return nil
}

func keysOf[V any](m map[uint32]V) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
