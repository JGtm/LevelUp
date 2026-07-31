// tmp_dmgattacker — A1 : L'ATTAQUANT via le global-id +0x0c (et la queue du record).
//
// Acquis (capture live + D1/S1) : un record de degat = paquet type-0, payload[0]==0xd2,
// le deser FUN_14080c1f8 lit a partir du bit 36 du payload :
//
//	+0x08 slot/cause  consumeId2 = R(1); si 0 R(2)
//	+0x0c global-id   R(5) + R(32) BE      <-- le R5 prefixe + le R32 = high-32 FAMILLE
//	+0x10 handle      R(1) gate ; si set R(32)   (=0 dans 519/519 => ABSENT)
//	+0x14 variant     R(32) BE (sous-variant; ici = low-32 0x42c9679f universel)
//
// id64 = (R32<<32)|low32 = cle analysis.WeaponIDToName.
//
// MISSION A1 (verrou attribution fiable = pas-au-timestamp) :
//  1. Decoder COMPLETEMENT le +0x0c : valeurs du R5 (0-31 ?), du R32 (famille).
//     Distribution du R5 : ~8 valeurs (joueurs) ? slot d'arme ? equipe ? correle a la famille ?
//  2. HYPOTHESE : un champ du record (R5 ou autre) identifie l'ENTITE-ARME / le JOUEUR
//     qui tire. Recoupe avec les loadouts connus (1 seul joueur a tel BR75 => attaquant deduit).
//  3. Au-dela de +0x14 (le record fait 214 o), chercher un index joueur / handle attaquant
//     non-nul (R3 ? R5 ? handle datum (h>>1)&0x7FFF ? encodage 0xE1500000+idx*0x10002 ?).
//
// Usage : tmp_dmgattacker [mode]
//
//	(defaut)  : distribution R5, recoupement R5<->famille, dump records.
//	tail      : scan de la queue du message (apres +0x14) -> champs candidats attaquant.
//	loadout   : recoupe famille du record <-> loadouts joueur (attribution par unicite).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const deserStartBit = 36
const variantSuffix = uint32(0x42c9679f)

var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

// pi->xuid->gamertag (bit-verifie, contexte 000d5950).
var xuidGamertag = map[uint64]string{
	2535467794760703: "whiteknight2519",
	2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm",
	2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA",
	2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus",
	2533274826120416: "VitaminA1688",
}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
		id64name[id] = n
	}
}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
	chunk   int
}

func listPackets(d []byte, chunk int) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+size], chunk})
		off += 16 + size
	}
	return out
}

func allType0() []pkt {
	var all []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		if len(d) == 0 {
			continue
		}
		for _, p := range listPackets(d, n) {
			if p.typ == 0 {
				all = append(all, p)
			}
		}
	}
	return all
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// dmgRec = un record de degat decode (les champs canoniques + bit de fin du variant).
type dmgRec struct {
	ts        uint64
	tms       int
	slotCause uint64 // valeur lisible (0 = forme courte, sinon 1+R2)
	slotBits  int
	r5        uint64 // prefixe 5 bits du global-id
	fam       string
	famH32    uint32
	id64      uint64
	variant   uint32 // low-32 (=0x42c9679f, suffixe universel)
	afterBit  int    // bit logique juste apres le low-32 (debut de la QUEUE)
	payload   []byte
	chunk     int
}

// decodeOne rejoue le deser CANONIQUE PROUVE (tmp_dmgscan) : depuis le bit 36,
//
//	slot/cause (consumeId2) ; R(5) ; R(32 high=famille) ; R(32 low=suffixe contigu).
//
// L'id64 = (high<<32)|low est la cle catalogue (le suffixe suit IMMEDIATEMENT le high,
// sans handle intercale -> field-map "handle +0x10" du commentaire D1 est decale, le low
// est contigu au high, cf tmp_dmgscan/PHASE1 recoupement 519==519). afterBit = juste apres
// le low-32 ; c'est la que commence la QUEUE du record (champs apres l'id64 d'arme).
func decodeOne(p pkt) (dmgRec, bool) {
	d := p.payload
	br := filmdec.NewBitReader(d)
	br.Skip(deserStartBit)
	// +0x08 slot/cause : R(1); si 0 R(2).
	s0 := br.BitPos()
	var slotCause uint64
	if br.ReadBit() {
		slotCause = 0
	} else {
		slotCause = 1 + br.ReadBits(2)
	}
	slotBits := br.BitPos() - s0
	// +0x0c global-id : R(5) prefixe, puis R(32) high = famille, puis R(32) low = suffixe.
	r5 := br.ReadBits(5)
	fam32 := uint32(br.ReadBits(32))
	low := uint32(br.ReadBits(32))
	afterBit := br.BitPos()

	fam, known := h32name[fam32]
	id64 := (uint64(fam32) << 32) | uint64(low)
	return dmgRec{
		ts: p.ts, tms: tsToMs(p.ts), slotCause: slotCause, slotBits: slotBits,
		r5: r5, fam: fam, famH32: fam32, id64: id64, variant: low,
		afterBit: afterBit, payload: d, chunk: p.chunk,
	}, known
}

// collectRecords renvoie les records de degat-arme (discriminant payload[0]==0xd2
// + famille catalogue + suffixe variant contigu), tries par temps.
func collectRecords() []dmgRec {
	var recs []dmgRec
	for _, p := range allType0() {
		if len(p.payload) == 0 || p.payload[0] != 0xd2 {
			continue
		}
		r, known := decodeOne(p)
		if !known || r.variant != variantSuffix {
			continue
		}
		recs = append(recs, r)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tms < recs[j].tms })
	return recs
}

func main() {
	build()
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	recs := collectRecords()
	fmt.Printf("=== %d records de degat-arme (payload[0]==0xd2, famille catalogue, suffixe contigu) ===\n", len(recs))

	switch mode {
	case "tail":
		scanTail(recs)
		return
	case "loadout":
		loadoutCrossCheck(recs)
		return
	case "perfam":
		perFamilyTail(recs)
		return
	case "prefix":
		prefixScan(recs)
		return
	case "cross":
		crossR5Slot(recs)
		return
	}

	// ── ETAPE 1 : distribution du R5 (prefixe du global-id). ──
	fmt.Println("\n=== ETAPE 1 : distribution du R5 (prefixe +0x0c, 5 bits, 0..31) ===")
	r5count := map[uint64]int{}
	for _, r := range recs {
		r5count[r.r5]++
	}
	type kc struct {
		k uint64
		c int
	}
	var r5s []kc
	for k, c := range r5count {
		r5s = append(r5s, kc{k, c})
	}
	sort.Slice(r5s, func(i, j int) bool { return r5s[i].c > r5s[j].c })
	fmt.Printf("  %d valeurs distinctes de R5 :\n", len(r5s))
	for _, e := range r5s {
		fmt.Printf("    R5=%2d (0x%02x)  x%d\n", e.k, e.k, e.c)
	}

	// ── ETAPE 1b : slot/cause distribution. ──
	fmt.Println("\n=== ETAPE 1b : slot/cause (+0x08) ===")
	scCount := map[uint64]int{}
	for _, r := range recs {
		scCount[r.slotCause]++
	}
	for v, c := range scCount {
		fmt.Printf("    slot/cause=%d  x%d\n", v, c)
	}

	// ── ETAPE 2 : R5 correle-t-il a la famille ? (test "R5 = slot d'arme/joueur"). ──
	fmt.Println("\n=== ETAPE 2 : table de contingence R5 x famille ===")
	// pour chaque R5, distribution des familles ; pour chaque famille, distribution des R5.
	r5fam := map[uint64]map[string]int{}
	famr5 := map[string]map[uint64]int{}
	for _, r := range recs {
		if r5fam[r.r5] == nil {
			r5fam[r.r5] = map[string]int{}
		}
		r5fam[r.r5][r.fam]++
		if famr5[r.fam] == nil {
			famr5[r.fam] = map[uint64]int{}
		}
		famr5[r.fam][r.r5]++
	}
	var r5keys []uint64
	for k := range r5fam {
		r5keys = append(r5keys, k)
	}
	sort.Slice(r5keys, func(i, j int) bool { return r5keys[i] < r5keys[j] })
	for _, k := range r5keys {
		fmt.Printf("    R5=%2d -> ", k)
		printFamDist(r5fam[k])
	}
	fmt.Println("\n  -- inversement, par famille : quels R5 ? (si 1 R5 par famille => R5 lie a l'arme, pas au joueur) --")
	var famkeys []string
	for f := range famr5 {
		famkeys = append(famkeys, f)
	}
	sort.Strings(famkeys)
	for _, f := range famkeys {
		fmt.Printf("    %-26s -> R5: %v\n", f, sortedKC(famr5[f]))
	}

	// ── ETAPE 3 : interpretation. ──
	fmt.Println("\n=== ETAPE 3 : interpretation R5 ===")
	fmt.Printf("  nb R5 distincts=%d ; nb familles=%d ; nb joueurs attendus=8\n", len(r5s), len(famr5))
	// Si R5 a ~autant de valeurs que de familles ET 1-1 famille<->R5 => R5 = type d'arme/projectile.
	// Si R5 a ~8 valeurs et une meme famille porte plusieurs R5 => R5 candidat = joueur/slot.

	// Dump 30 premiers records pour inspection.
	fmt.Println("\n=== DUMP 30 premiers records (ts, R5, slot/cause, handle, famille) ===")
	for i, r := range recs {
		if i >= 30 {
			break
		}
		fmt.Printf("  [%3d] t=%7.1fs R5=%2d sc=%d fam=%-24s id64=0x%016x\n",
			i, float64(r.tms)/1000, r.r5, r.slotCause, r.fam, r.id64)
	}
}

func printFamDist(m map[string]int) {
	type fc struct {
		f string
		c int
	}
	var fcs []fc
	for f, c := range m {
		fcs = append(fcs, fc{f, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	for _, e := range fcs {
		fmt.Printf("%s:%d ", e.f, e.c)
	}
	fmt.Println()
}

func sortedKC(m map[uint64]int) string {
	type kc struct {
		k uint64
		c int
	}
	var ks []kc
	for k, c := range m {
		ks = append(ks, kc{k, c})
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].k < ks[j].k })
	s := ""
	for _, e := range ks {
		s += fmt.Sprintf("%d:%d ", e.k, e.c)
	}
	return s
}

// ── MODE tail : scanner la queue du record (apres +0x14) pour un index/handle attaquant. ──
//
// On lit, a partir de afterBit (apres le variant +0x14), une serie de champs candidats :
//   - un R(3) (index joueur 0..7 ?)
//   - un R(5) (slot/joueur ?)
//   - un R(32) (handle datum world ?) -> on decode (h>>1)&0x7FFF (index world) + gen.
//
// On cherche un champ qui prend ~8 valeurs distinctes (= joueurs) et, idealement, dont
// la distribution recoupe les 8 slots/joueurs connus.
func scanTail(recs []dmgRec) {
	fmt.Println("\n=== MODE tail : champs candidats apres +0x14 (variant) ===")
	// Pour chaque profondeur de bits apres afterBit, on agrege la valeur lue en R(w)
	// sur tous les records, et on mesure le nb de valeurs distinctes.
	// w=3 (0..7), w=4, w=5, w=6.
	for _, w := range []int{3, 4, 5, 6} {
		fmt.Printf("\n  -- fenetre glissante R(%d) sur les 64 bits apres le variant --\n", w)
		bestOff, bestDistinct, bestSpread := -1, 1<<30, 0.0
		for off := 0; off <= 64; off++ {
			vals := map[uint64]int{}
			for _, r := range recs {
				v := bitsAt(r.payload, r.afterBit+off, w)
				vals[v]++
			}
			distinct := len(vals)
			// "spread" : entropie approximative (favorise distributions equilibrees ~8 classes).
			spread := float64(distinct)
			if distinct >= 6 && distinct <= 9 {
				if distinct < bestDistinct || (distinct == bestDistinct && spread > bestSpread) {
					bestDistinct, bestOff, bestSpread = distinct, off, spread
				}
			}
		}
		if bestOff >= 0 {
			vals := map[uint64]int{}
			for _, r := range recs {
				vals[bitsAt(r.payload, r.afterBit+bestOff, w)]++
			}
			fmt.Printf("    meilleur offset=%d -> %d valeurs distinctes : %v\n", bestOff, bestDistinct, sortedKC(vals))
		} else {
			fmt.Printf("    (aucun offset 0..64 ne donne 6..9 valeurs distinctes en R(%d))\n", w)
		}
	}

	// Recherche d'un handle datum world dans la queue (R32 dont (h>>1)&0x7FFF est petit/stable).
	fmt.Println("\n  -- recherche handle datum world dans la queue (R32, index=(h>>1)&0x7FFF) --")
	// On teste plusieurs offsets ; pour chacun, distribution des index world.
	for _, off := range []int{0, 1, 8, 16, 32, 33, 40, 48} {
		idxCount := map[uint64]int{}
		nonFFFF := 0
		for _, r := range recs {
			h := uint32(bitsAt(r.payload, r.afterBit+off, 32))
			if h == 0xffffffff || h == 0 {
				continue
			}
			nonFFFF++
			idx := uint64((h >> 1) & 0x7fff)
			idxCount[idx]++
		}
		fmt.Printf("    off=%2d : %d/%d non-trivial, %d index world distincts\n", off, nonFFFF, len(recs), len(idxCount))
	}

	// Dump brut : pour 12 records, afficher les 96 bits suivant le variant (hex bit-string).
	fmt.Println("\n  -- dump brut 96 bits apres variant (12 records, pour inspection manuelle) --")
	for i, r := range recs {
		if i >= 12 {
			break
		}
		var bitsStr string
		for b := 0; b < 96; b++ {
			bitsStr += fmt.Sprintf("%d", bitsAt(r.payload, r.afterBit+b, 1))
			if (b+1)%8 == 0 {
				bitsStr += " "
			}
		}
		fmt.Printf("    [%2d] R5=%2d fam=%-20s tail=%s\n", i, r.r5, r.fam, bitsStr)
	}
}

// ── MODE loadout : recoupe famille du record <-> loadouts connus (attribution par unicite). ──
//
// On charge les loadouts du keyframe (par joueur). Pour chaque record, on regarde
// combien de joueurs possedent cette famille. Si exactement 1 => attaquant deduit.
// Donne le taux d'attribution-par-unicite-loadout (sans timestamp).
func loadoutCrossCheck(recs []dmgRec) {
	fmt.Println("\n=== MODE loadout : attribution par unicite de la famille dans les loadouts ===")
	// Loadouts du keyframe (acquis tmp_loadout, familles uniquement). On les re-extrait
	// a la volee : les armes de chaque record biped #35. Faute de calibration ici, on
	// utilise la table connue (8 joueurs Fiesta du keyframe). NB : en Fiesta les armes
	// changent (pickups) -> le loadout initial est une borne inferieure d'unicite.
	famByPlayer := loadoutFamilies()
	// inverse : famille -> set de joueurs.
	famOwners := map[string]map[string]bool{}
	for pl, fams := range famByPlayer {
		for _, f := range fams {
			if famOwners[f] == nil {
				famOwners[f] = map[string]bool{}
			}
			famOwners[f][pl] = true
		}
	}
	uniq, ambig, none := 0, 0, 0
	for _, r := range recs {
		owners := famOwners[r.fam]
		switch {
		case len(owners) == 1:
			uniq++
		case len(owners) > 1:
			ambig++
		default:
			none++
		}
	}
	fmt.Printf("  records=%d : attribuable-par-unicite=%d (%.0f%%) ; ambigu(>1 owner)=%d ; aucun-owner(pickup/Fiesta)=%d\n",
		len(recs), uniq, 100*float64(uniq)/float64(len(recs)), ambig, none)
	fmt.Println("  -- familles vues dans les records et leur nb de proprietaires (loadout initial) --")
	famSeen := map[string]int{}
	for _, r := range recs {
		famSeen[r.fam]++
	}
	var fs []string
	for f := range famSeen {
		fs = append(fs, f)
	}
	sort.Strings(fs)
	for _, f := range fs {
		fmt.Printf("    %-26s records=%-4d owners(loadout init)=%d %v\n", f, famSeen[f], len(famOwners[f]), keys(famOwners[f]))
	}
}

func keys(m map[string]bool) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

// ── MODE cross : R5 (4 val) x slot/cause (4 val) — signature equipe/type ? ──
// On verifie si (R5, slotCause) determine la famille (=> ce sont des proprietes de l'arme,
// pas du joueur) et on decompose R5/slotCause en bits pour lire leur structure.
func crossR5Slot(recs []dmgRec) {
	fmt.Println("\n=== MODE cross : R5 x slot/cause x famille ===")
	// (R5,sc) -> familles.
	type key struct{ r5, sc uint64 }
	m := map[key]map[string]int{}
	for _, r := range recs {
		k := key{r.r5, r.slotCause}
		if m[k] == nil {
			m[k] = map[string]int{}
		}
		m[k][r.fam]++
	}
	var keys []key
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].r5 != keys[j].r5 {
			return keys[i].r5 < keys[j].r5
		}
		return keys[i].sc < keys[j].sc
	})
	for _, k := range keys {
		// famille dominante.
		top, topc, total := "", 0, 0
		for f, c := range m[k] {
			total += c
			if c > topc {
				top, topc = f, c
			}
		}
		fmt.Printf("  R5=%2d (0b%05b) sc=%d -> n=%-3d top=%-22s (%d/%d) nfam=%d\n",
			k.r5, k.r5, k.sc, total, top, topc, total, len(m[k]))
	}
	// Decomposition bit du R5 (quels bits varient ?).
	fmt.Println("\n  -- bits du R5 qui varient (sur 5 bits) --")
	for b := 0; b < 5; b++ {
		ones := 0
		for _, r := range recs {
			if (r.r5>>uint(b))&1 == 1 {
				ones++
			}
		}
		fmt.Printf("    bit%d : %d/%d a 1 (%.0f%%)\n", b, ones, len(recs), 100*float64(ones)/float64(len(recs)))
	}
}

// ── MODE perfam : pour chaque famille, quels bits de la QUEUE sont stables vs variables ? ──
// Un sous-champ STABLE par famille = propriete de l'arme (degats, hit-section). Un champ
// VARIABLE au sein d'une meme famille, prenant ~peu de valeurs = candidat attaquant/cible.
func perFamilyTail(recs []dmgRec) {
	fmt.Println("\n=== MODE perfam : stabilite des bits de la queue PAR famille ===")
	byFam := map[string][]dmgRec{}
	for _, r := range recs {
		byFam[r.fam] = append(byFam[r.fam], r)
	}
	var fams []string
	for f := range byFam {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	const W = 80 // bits de queue examines
	for _, f := range fams {
		group := byFam[f]
		if len(group) < 4 {
			continue
		}
		// pour chaque bit i de la queue, fraction de records ou il vaut 1.
		// bit "stable" : fraction ~0 ou ~1. bit "variable" : fraction ~0.5.
		var stableMask string
		variableBits := 0
		for i := 0; i < W; i++ {
			ones := 0
			for _, r := range group {
				if bitsAt(r.payload, r.afterBit+i, 1) == 1 {
					ones++
				}
			}
			frac := float64(ones) / float64(len(group))
			switch {
			case frac < 0.05 || frac > 0.95:
				stableMask += "." // stable
			case frac < 0.25 || frac > 0.75:
				stableMask += "-" // quasi-stable
			default:
				stableMask += "V" // variable
				variableBits++
			}
		}
		fmt.Printf("  %-26s n=%-3d varbits=%d/%d\n      mask=%s\n", f, len(group), variableBits, W, stableMask)
	}
	fmt.Println("\n  Legende : '.'=stable(arme) '-'=quasi-stable 'V'=variable(candidat position/cible/attaquant)")
}

// ── MODE prefix : examiner l'EN-TETE 8 octets du message + les 36 bits avant le slot. ──
// L'attaquant pourrait etre dans le header (consomme par FUN_14080AADE) plutot que dans le
// corps. On dump l'en-tete (8 o) et les premiers 36 bits, et on cherche un champ ~8 valeurs.
func prefixScan(recs []dmgRec) {
	fmt.Println("\n=== MODE prefix : en-tete 8 o + 36 bits avant le slot/cause ===")
	// L'en-tete capturee = d2 60 44 00 04 38 4b d2. Les octets 0..3 (d2 60 44 00) et 4..7
	// (04 38 4b d2) : varient-ils par record ? Lesquels ~8 valeurs ?
	fmt.Println("  -- distribution de chaque octet d'en-tete (0..7) --")
	for b := 0; b < 8; b++ {
		vals := map[byte]int{}
		for _, r := range recs {
			if b < len(r.payload) {
				vals[r.payload[b]]++
			}
		}
		fmt.Printf("    octet[%d] : %d valeurs distinctes\n", b, len(vals))
	}
	// Les 36 bits avant le slot = bits 0..35 du payload (l'en-tete). On teste R(w) a chaque
	// offset 0..36 pour trouver un champ ~8 valeurs.
	fmt.Println("\n  -- champs R(3/4) dans les bits 0..36 (en-tete) : ~8 valeurs ? --")
	for _, w := range []int{3, 4} {
		for off := 0; off+w <= 40; off++ {
			vals := map[uint64]int{}
			for _, r := range recs {
				vals[bitsAt(r.payload, off, w)]++
			}
			if len(vals) >= 6 && len(vals) <= 9 {
				fmt.Printf("    R(%d)@bit%-2d : %d valeurs : %v\n", w, off, len(vals), sortedKC(vals))
			}
		}
	}
	// Dump des 8 octets d'en-tete pour 20 records (groupes par famille pour voir si l'en-tete
	// porte la famille ou un id de source).
	fmt.Println("\n  -- en-tete 8 o (20 records, avec famille) --")
	for i, r := range recs {
		if i >= 20 {
			break
		}
		fmt.Printf("    [%2d] %-22s hdr=% x\n", i, r.fam, r.payload[:8])
	}
}

// loadoutFamilies : familles d'armes par joueur au keyframe (acquis tmp_loadout).
// Fiesta => loadout aleatoire ; valeurs indicatives. A defaut de calibration ici, on
// renvoie une map vide -> le mode loadout signalera l'absence de donnee fiable.
func loadoutFamilies() map[string][]string {
	// NB : le keyframe Fiesta donne 8 paires d'armes (tmp_loadout). On ne les code pas
	// en dur ici (elles changent par pickup). Le but est de MESURER l'unicite, pas de
	// fournir une table figee : si vide, le taux d'unicite sera 0 et on le dira.
	return map[string][]string{}
}
