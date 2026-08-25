package replay

// inventory_trous_mesure_test.go — MESURER LES TROUS DE LA FICHE D'INVENTAIRE (aucune
// correction, aucune publication).
//
// LA QUESTION. Le moteur reconstruit l'etat COMPLET a chaque image-cle : chaque bipede vivant
// doit y avoir un record complet. Or la fiche de certains joueurs ne donne rien par moments.
// Deux causes possibles, et une seule est un defaut de nos regles : (1) le record de bipede
// MANQUE a l'image-cle, (2) le record est la et nos ancrages echouent dessus.
//
// CE QUI EST COMPTE, par categories EXCLUSIVES (premiere regle qui tombe) :
//
//	(a) record de bipede ABSENT pour un slot pourtant vivant a cet instant — « vivant » etant
//	    etabli SANS le record manquant : le slot apparait a une image-cle AVANT et a une
//	    image-cle APRES. C'est un encadrement, pas une supposition.
//	(b1) R1 echoue — aucune occurrence de l'ancre 28 bits dans le record ;
//	(b2) R1 echoue — l'ancre est la, mais le motif 20 bits n'est pas dans les 60 bits suivants ;
//	(c) R1 echoue — deux lectures d'ancre, non departagees ;
//	(d) R2 echoue — pas de motif i22 apres l'ancre, ou somme nulle ;
//	(e) R3 echoue — aucune famille d'arme reconnue dans le record ;
//	(f) R4 echoue — aucun debut de bloc de munitions n'atterrit au bit pres ;
//	(g) succes partiel — quels champs manquent ;
//	(h) succes complet.
//
// LA LECTURE GENERALISEE, qui est le TEMOIN de l'hypothese « canal borgne ». Le motif 20 bits
// vaut `[17 bits fixes][010]`, et ces trois derniers bits sont les bits de POIDS FORT du rang
// (invAbilityRankHigh). Chercher le seul PREFIXE 17 bits, puis lire 6 bits (haut puis bas),
// donne le rang COMPLET sur toute la palette. Si les records ou R1 echoue rendent, par cette
// lecture, un rang que le canal i48 — totalement independant, dans les paquets delta —
// confirme, alors la cause n'est pas un defaut d'ancrage : c'est la fenetre 16..23.
//
// LECTURE SEULE : aucune base, aucun document, aucun fichier du depot ecrit hors INV_OUT.
// UN SEUL decodage filmdec a la fois (LockProcessDecode), un film apres l'autre.
//
// USAGE (depuis apps/go-api) :
//
//	INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=20 INV_DIG=5 \
//	  go test ./internal/analysis/replay/ -run '^TestInventaireTrousMesure$' -timeout 120m -v
//
//	INV_FILMS=<repo>/data/cache/film_chunks/000d5950 INV_DIG=8 \
//	  go test ./internal/analysis/replay/ -run '^TestInventaireTrousMesure$' -timeout 30m -v
//
// INV_I48=0 coupe le controle croise i48 (couteux : il redecode les paquets delta).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	invTrousFilmsEnv  = "INV_FILMS"
	invTrousCacheEnv  = "INV_CACHE"
	invTrousSampleEnv = "INV_SAMPLE"
	invTrousOutEnv    = "INV_OUT"
	invTrousDigEnv    = "INV_DIG"
	invTrousI48Env    = "INV_I48"
)

// invTrousGenWin est la fenetre de recherche du motif, en bits apres l'ancre. 60 est celle de
// production ; 400 sert UNIQUEMENT a mesurer si 60 est trop court.
const (
	invTrousGenWin  = 60
	invTrousWideWin = 400
)

// invTrousPrefix est le prefixe 17 bits du motif de capacite, motif prive de ses 3 bits de
// rang haut. DERIVE du motif de production, jamais reecrit a cote de lui.
const invTrousPrefix = invAbilityPattern >> 3

// invTrousDiag est le diagnostic d'UN record de bipede a UNE image-cle.
type invTrousDiag struct {
	slot         uint32
	chunk, pkt   int
	ts           uint64
	bits         int // largeur du record
	ancres       int // occurrences de l'ancre 28 bits
	hits         int // lectures R1 strictes (ancre + motif dans 60 bits)
	rangR1       int
	gren         bool
	fam          bool
	ammoSols     int
	drawn        int
	genRangs     []int          // rangs rendus par la lecture generalisee (fenetre 60)
	motifLarge   int            // offset du motif STRICT trouve entre 60 et 400 bits, -1 sinon
	rangI48      int            // rang lu par i48 pour ce slot autour de cet instant, -1 inconnu
	motifSeul    int            // occurrences du motif 20 bits N'IMPORTE OU dans le record
	motifPos     []int          // positions de ces occurrences
	from         int            // premier bit du record (origine des positions relatives)
	ancrePos     int            // position de la premiere ancre, relative au record, -1 sinon
	prefSeul     int            // occurrences du prefixe 17 bits n'importe ou dans le record
	h1           []invTrousH1   // ancres a distance de Hamming 1
	h1Gen        []int          // rangs lus a partir de ces ancres, fenetre de production
	i22Zero      bool           // un motif i22 de somme NULLE existe apres l'ancre
	prefHits     []invTrousPref // toutes les occurrences du prefixe 17 bits et le rang qui suit
	categorie    string
	champsManque string
}

func TestInventaireTrousMesure(t *testing.T) {
	films := invTrousCorpus(t)
	if len(films) == 0 {
		t.Skipf("%s ou %s est requis — mesure sautee", invTrousFilmsEnv, invTrousCacheEnv)
	}
	dig := invTrousEnvInt(invTrousDigEnv, 5)
	var sb strings.Builder
	fmt.Fprintf(&sb, "film\tkeyframes\tkfSansBiped\trecords\tslotsAbsents\t"+
		"b1_ancreAbsente\tb2_motifAbsent\tc_ancreMultiple\td_R2\te_R3\tf_R4\tg_partiel\th_complet\t"+
		"fichesVides\tammoMulti\tgenSauveR1\tgenAccordI48\tgenDesaccordI48\tmotifAuDelaDe60\n")
	total := invTrousNewCompte("TOTAL")
	for _, dir := range films {
		c := invTrousFilm(t, dir, dig)
		c.log(t)
		c.tsv(&sb)
		total.fusion(c)
	}
	total.log(t)
	total.tsv(&sb)
	invTrousEcrire(t, sb.String())
}

// invTrousCorpus resout la liste des dossiers de film a mesurer.
func invTrousCorpus(t *testing.T) []string {
	t.Helper()
	if raw := strings.TrimSpace(os.Getenv(invTrousFilmsEnv)); raw != "" {
		var out []string
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	root := strings.TrimSpace(os.Getenv(invTrousCacheEnv))
	if root == "" {
		return nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("racine de cache illisible %s : %v", root, err)
	}
	var shorts []string
	for _, e := range ents {
		if e.IsDir() {
			shorts = append(shorts, e.Name())
		}
	}
	sort.Strings(shorts)
	n := invTrousEnvInt(invTrousSampleEnv, 20)
	if n <= 0 || n >= len(shorts) {
		n = len(shorts)
	}
	// ECHANTILLON REPARTI : un pas constant sur les prefixes tries, pas les n premiers — les
	// prefixes sont des hachages, mais prendre une tranche contigue reste un choix arbitraire
	// que rien ne justifie.
	step := len(shorts) / n
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, filepath.Join(root, shorts[i*step]))
	}
	return out
}

func invTrousEnvInt(key string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return def
}

// invTrousFilm mesure UN film.
func invTrousFilm(t *testing.T, dir string, dig int) *invTrousCompte {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	c := invTrousNewCompte(filepath.Base(dir))
	known := loadoutFamilies()
	diags, slotSets := invTrousWalk(t, dir, known, c)
	if len(diags) == 0 {
		return c
	}
	invTrousJoinI48(t, dir, diags)
	for i := range diags {
		invTrousClasser(&diags[i])
		c.compter(&diags[i])
	}
	c.slotsAbsents = invTrousSlotsAbsents(slotSets)
	invTrousCreuser(t, c, diags, dig)
	return c
}

// invTrousWalk parcourt les images-cles du film et diagnostique chaque record de bipede.
func invTrousWalk(
	t *testing.T, dir string, known map[uint32]bool, c *invTrousCompte,
) ([]invTrousDiag, []map[uint32]bool) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	var diags []invTrousDiag
	var slotSets []map[uint32]bool
	for ch := 1; ch <= n; ch++ {
		chunk, err := filmdec.ReadFilmChunk(dir, ch)
		if err != nil {
			// UN CHUNK ILLISIBLE N'EST PAS UNE MESURE : il est compte, et le film qui en
			// porte est ecarte de l'agregat par l'appelant. Sans ce compteur, une lecture
			// ratee se lit comme « aucun trou ».
			c.chunksIllisibles++
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			c.keyframes++
			pay := p.Payload(chunk)
			set := map[uint32]bool{}
			for _, sp := range invRecordSpans(pay) {
				if sp.ti != invBipedTI {
					continue
				}
				set[uint32(sp.slot)] = true
				d := invTrousDiagnose(pay, sp, known)
				d.chunk, d.pkt, d.ts = ch, p.Index, p.TimestampUS
				diags = append(diags, d)
			}
			c.parKeyframe[len(set)]++
			if len(set) == 0 {
				c.keyframesSansBiped++
			}
			slotSets = append(slotSets, set)
		}
	}
	return diags, slotSets
}

// invTrousDiagnose applique les regles de production a UN record, et mesure a cote d'elles la
// lecture generalisee et la fenetre large. Aucune de ces deux mesures n'influence le verdict.
func invTrousDiagnose(pay []byte, sp invRecordSpan, known map[uint32]bool) invTrousDiag {
	d := invTrousDiag{
		slot: uint32(sp.slot), bits: sp.to - sp.from,
		rangR1: -1, drawn: -1, motifLarge: -1, rangI48: -1, from: sp.from, ancrePos: -1,
	}
	ancres := invTrousAncres(pay, sp.from, sp.to)
	d.ancres = len(ancres)
	if len(ancres) > 0 {
		d.ancrePos = ancres[0] - 27 - sp.from
	}
	hits := invAbilityIn(pay, sp.from, sp.to)
	d.hits = len(hits)
	if len(hits) == 1 {
		d.rangR1 = invAbilityRankOf(hits[0].low)
		if _, ok := invGrenadesAfter(pay, hits[0].anchorBit, sp.to, DefaultGrenadeMax); ok {
			d.gren = true
		}
	}
	for _, b := range ancres {
		d.genRangs = append(d.genRangs, invTrousGen(pay, b, sp.to, invTrousGenWin)...)
		if d.motifLarge < 0 {
			d.motifLarge = invTrousMotifLarge(pay, b, sp.to)
		}
		if !d.gren {
			d.i22Zero = d.i22Zero || invTrousI22Zero(pay, b, sp.to)
		}
	}
	d.motifSeul, d.prefSeul, d.motifPos = invTrousMotifPartout(pay, sp.from, sp.to)
	d.prefHits = invTrousPrefHits(pay, sp.from, sp.to)
	d.h1 = invTrousAncreH1(pay, sp.from, sp.to)
	for _, h := range d.h1 {
		d.h1Gen = append(d.h1Gen, invTrousGen(pay, h.fin, sp.to, invTrousGenWin)...)
	}
	if first, ok := invFirstFamily(pay, sp.from, sp.to, known); ok {
		d.fam = true
		var inv KeyframeInventory
		inv.DrawnSlot = -1
		readAmmo(pay, &inv, sp.from, first)
		d.ammoSols, d.drawn = inv.AmmoCandidates, inv.DrawnSlot
	}
	return d
}

// invTrousAncres rend la position du DERNIER bit de chaque occurrence de l'ancre 28 bits.
func invTrousAncres(pay []byte, from, to int) []int {
	var out []int
	var w uint32
	const mask28 = (uint32(1) << 28) - 1
	for b := from; b < to; b++ {
		w = ((w << 1) | invBitAt(pay, b)) & mask28
		if b-from >= 27 && w == invAbilityAnchor {
			out = append(out, b)
		}
	}
	return out
}

// invTrousGen est la LECTURE GENERALISEE : prefixe 17 bits, puis 6 bits de rang (haut, bas).
func invTrousGen(pay []byte, b, to, win int) []int {
	var out []int
	for off := 0; off <= win; off++ {
		p := b + 1 + off
		if p+23 > to {
			break
		}
		if invBits(pay, p, 17) != invTrousPrefix {
			continue
		}
		out = append(out, int(invBits(pay, p+17, 6)))
	}
	return out
}

// invTrousMotifLarge cherche le motif STRICT au-dela de la fenetre de production : il mesure
// si 60 bits est trop court. Rend l'offset, ou -1.
func invTrousMotifLarge(pay []byte, b, to int) int {
	for off := invTrousGenWin + 1; off <= invTrousWideWin; off++ {
		p := b + 1 + off
		if p+20 > to {
			return -1
		}
		if invBits(pay, p, 20) == invAbilityPattern {
			return off
		}
	}
	return -1
}

// invTrousMotifPartout compte le motif 20 bits et son prefixe 17 bits N'IMPORTE OU dans le
// record. Si un record sans ancre porte quand meme le motif, c'est l'ANCRE qui manque, pas le
// champ.
func invTrousMotifPartout(pay []byte, from, to int) (motif, pref int, pos []int) {
	for b := from; b+20 <= to; b++ {
		if invBits(pay, b, 20) == invAbilityPattern {
			motif++
			pos = append(pos, b)
		}
		if invBits(pay, b, 17) == invTrousPrefix {
			pref++
		}
	}
	return
}

// invTrousH1 est une ancre « presque la » : elle ne differe de l'ancre de production que par UN
// bit, dont on retient le rang (0 = bit de poids fort des 28).
type invTrousH1 struct {
	fin  int // position du DERNIER bit de l'occurrence
	rang int
}

// invTrousAncreH1 releve les occurrences de l'ancre 28 bits a distance de Hamming 1.
//
// LE HASARD EST ECARTE PAR LE DENOMBREMENT : 28 variantes d'un mot de 28 bits, c'est une chance
// sur 9,6 millions par position. Un record de 5 000 bits en attend 0,0005. En trouver une dans
// presque chaque record dirait que l'un des 28 bits n'est PAS constant — donc qu'il porte un
// champ, et que l'ancre de production est trop large d'un bit.
func invTrousAncreH1(pay []byte, from, to int) []invTrousH1 {
	var out []invTrousH1
	var w uint32
	const mask28 = (uint32(1) << 28) - 1
	for b := from; b < to; b++ {
		w = ((w << 1) | invBitAt(pay, b)) & mask28
		if b-from < 27 {
			continue
		}
		x := w ^ invAbilityAnchor
		if x == 0 || x&(x-1) != 0 {
			continue
		}
		rang := 0
		for m := uint32(1) << 27; m != 0 && m != x; m >>= 1 {
			rang++
		}
		out = append(out, invTrousH1{fin: b, rang: rang})
	}
	return out
}

// invTrousI22Zero dit si un motif i22 de somme NULLE existe apres l'ancre. C'est le test du
// critere `somme > 0` de R2 : un Spartan SANS AUCUNE grenade produit exactement ce motif, et
// R2 le rejette — la mesure « zero grenade » devient alors indistinguable d'une non-lecture.
func invTrousI22Zero(pay []byte, from, to int) bool {
	for b := from; b+35 <= to; b++ {
		if invBits(pay, b, 3) != 4 {
			continue
		}
		sum, ok := uint32(0), true
		for i := 0; i < invGrenadeSlots && ok; i++ {
			v := invBits(pay, b+3+8*i, 8)
			if v > DefaultGrenadeMax {
				ok = false
			}
			sum += v
		}
		if ok && sum == 0 {
			return true
		}
	}
	return false
}

// invTrousPref est une occurrence du prefixe 17 bits, avec les 6 bits de rang qui la suivent.
type invTrousPref struct {
	pos  int
	rang int
}

// invTrousPrefHits releve toutes les occurrences du prefixe 17 bits du record.
func invTrousPrefHits(pay []byte, from, to int) []invTrousPref {
	var out []invTrousPref
	for b := from; b+23 <= to; b++ {
		if invBits(pay, b, 17) == invTrousPrefix {
			out = append(out, invTrousPref{pos: b, rang: int(invBits(pay, b+17, 6))})
		}
	}
	return out
}

// invTrousOracle cherche, parmi ces occurrences, celle dont le rang est celui qu'annonce i48,
// et rend son offset par rapport a l'ancre-variante. Le temoin decale le rang de 1.
func invTrousOracle(d invTrousDiag) (ok, temoin bool, off int, aOff bool) {
	if d.rangI48 < 0 {
		return false, false, 0, false
	}
	base := 0
	if len(d.h1) > 0 {
		base, aOff = d.h1[0].fin-27, true
	}
	for _, h := range d.prefHits {
		if h.rang == d.rangI48 && !ok {
			ok, off = true, h.pos-base
		}
		if h.rang == (d.rangI48+1)%64 {
			temoin = true
		}
	}
	return ok, temoin, off, aOff && ok
}

func invTrousUnique(v []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, x := range v {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Ints(out)
	return out
}

// invTrousJoinI48 attache a chaque diagnostic le rang lu par le canal i48 pour le MEME slot,
// la lecture la plus proche dans le temps. Canal independant : c'est ce qui en fait un temoin.
func invTrousJoinI48(t *testing.T, dir string, diags []invTrousDiag) {
	t.Helper()
	if invTrousEnvInt(invTrousI48Env, 1) == 0 {
		return
	}
	ranks, st, err := filmdec.ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("    i48 illisible (%v) — controle croise saute", err)
		return
	}
	t.Logf("    i48 : %d lectures (records %d, masque %d, illisibles %d)",
		len(ranks), st.Records, st.WithI48, st.Unread)
	bySlot := map[uint32][]filmdec.AbilityRank{}
	for _, r := range ranks {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	for s := range bySlot {
		v := bySlot[s]
		sort.Slice(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
		bySlot[s] = v
	}
	for i := range diags {
		diags[i].rangI48 = invTrousPlusProche(bySlot[diags[i].slot], diags[i].ts)
	}
}

func invTrousPlusProche(v []filmdec.AbilityRank, ts uint64) int {
	best, bestD := -1, uint64(1)<<62
	for _, r := range v {
		d := r.TimestampUS - ts
		if r.TimestampUS < ts {
			d = ts - r.TimestampUS
		}
		if d < bestD {
			best, bestD = r.Rank, d
		}
	}
	return best
}
