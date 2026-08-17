package filmdec

// keyframe_record_walk_test.go — INSTRUMENT DE MESURE de la grammaire du corps d'un record
// d'image-cle (lot R5, cf. .ai/V7.5/replay2d/PLAN_R5_GRAMMAIRE_IMAGE_CLE.md).
//
// CE QU'IL MESURE :
//
//	TestKFGramChain  (C1, C2) — pour chaque record borne de l'oracle `WalkKeyframeWorld`,
//	  la marche du corps atterrit-elle sur une frontiere de record VALIDE, et le CHAINAGE
//	  depuis cette position retombe-t-il EXACTEMENT sur la frontiere que l'oracle annonce ?
//	  Les huit combinaisons de grammaire sont probees sur le meme denominateur. C'est le test
//	  de l'hypothese H1 : si le balayeur saute des records (filtre `field26 == 0`), le
//	  chainage retombe juste et les intercales portent un `field26` non nul.
//
//	TestKFGramGlobal (C3) — un walker DETERMINISTE traverse-t-il un payload d'image-cle de
//	  bout en bout, et retrouve-t-il tous les records de l'oracle ?
//
// Il ne PUBLIE rien : il rend des taux et leurs denominateurs.
//
// LECTURE SEULE, garde par KF_GRAM_FILM, saute partout ailleurs (CI comprise). UN SEUL
// decodage filmdec par process (globaux de paquet) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF_GRAM_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestKFGram' -timeout 30m -v

import (
	"os"
	"sort"
	"testing"
)

const kfGramFilmEnv = "KF_GRAM_FILM"

// kfGramPayloads rend les payloads d'image-cle du film de dir, avec son registre.
func kfGramPayloads(t *testing.T, dir string) (*Registry, [][]byte) {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 (registre) illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	var pays [][]byte
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeKeyframe {
				pays = append(pays, pk.Payload(data))
			}
		}
	}
	if len(pays) == 0 {
		t.Fatalf("aucune image-cle dans %s", dir)
	}
	return reg, pays
}

// kfGramTally compte, pour UNE combinaison de grammaire et UN archetype, ce que la marche a
// rencontre. Sans ces denominateurs, un taux ne se juge pas.
type kfGramTally struct {
	bounded    int // records bornes par un voisin de l'oracle
	desync     int // marches interrompues par un composant non porte
	direct     int // marches atterrissant PILE sur la frontiere de l'oracle
	chained    int // marches y retombant apres des records intercales
	landedHdr  int // marches atterrissant sur un en-tete VALIDE (relache)
	skipped    int // total des records intercales traverses
	skippedF26 int // parmi eux, ceux dont field26 != 0 (invisibles au balayeur)
	lost       int // marches qui ne retombent jamais sur la frontiere
	// stops : cause d'arret du chainage, index parallele a KeyframeWalkStop.
	stops [5]int
	// gaps : distribution des ecarts `want - EndBit` (les 6 plus frequents sont publies).
	gaps map[int]int
}

// exactRate rend le taux d'atterrissage (direct + chaine) sur le denominateur des records
// bornes, en pourcentage.
func (k kfGramTally) exactRate() float64 {
	if k.bounded == 0 {
		return 0
	}
	return 100 * float64(k.direct+k.chained) / float64(k.bounded)
}

// kfGramTargets sont les archetypes mesures : `ti=37` (la cible de C1, celle dont R3 a publie
// le denominateur de 1 226 records) et `ti=38`, l'objet du monde le plus frequent des memes
// payloads, qui sert de SECOND archetype pour C2. Le second est choisi par sa frequence, pas
// par sa commodite : le test publie de toute facon le classement complet.
var kfGramTargets = []int{EquipmentTypeIndex, 38}

func TestKFGramChain(t *testing.T) {
	dir := os.Getenv(kfGramFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", kfGramFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	reg, pays := kfGramPayloads(t, dir)
	t.Logf("== %s : %d images-cles ==", dir, len(pays))
	kfGramLogArchetypes(t, pays)

	for _, ti := range kfGramTargets {
		t.Logf("---- archetype ti=%d ----", ti)
		for k, lay := range EquipmentIdentityLayouts {
			restore := lay.apply()
			tal := kfGramMeasure(reg, pays, ti)
			restore()
			t.Logf("  [%d] %s | bornes %4d · direct %4d · chaine %4d · perdus %4d · desync %4d"+
				" | ATTERRISSAGE %.1f %%", k, lay, tal.bounded, tal.direct, tal.chained,
				tal.lost, tal.desync, tal.exactRate())
			t.Logf("      atterrissages sur un en-tete valide %d · intercales %d (dont"+
				" field26 != 0 : %d)", tal.landedHdr, tal.skipped, tal.skippedF26)
			kfGramLogStops(t, tal)
		}
	}
}

// kfGramLogStops publie la cause d'arret des chainages et les ecarts dominants — sans eux,
// un taux de 0 % ne dit pas OU la marche a lache.
func kfGramLogStops(t *testing.T, tal kfGramTally) {
	t.Helper()
	for s, n := range tal.stops {
		if n > 0 {
			t.Logf("      arret du chainage : %-20s %4d", KeyframeWalkStop(s), n)
		}
	}
	type gv struct{ gap, n int }
	gs := make([]gv, 0, len(tal.gaps))
	for g, n := range tal.gaps {
		gs = append(gs, gv{g, n})
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i].n > gs[j].n })
	if len(gs) > 6 {
		gs = gs[:6]
	}
	for _, g := range gs {
		t.Logf("      ecart want-EndBit = %6d bits : %4d fois", g.gap, g.n)
	}
}

// kfGramMeasure rejoue, pour l'archetype ti, la marche de chaque record borne de l'oracle et
// son chainage vers la frontiere annoncee.
func kfGramMeasure(reg *Registry, pays [][]byte, ti int) kfGramTally {
	tal := kfGramTally{gaps: map[int]int{}}
	for _, pay := range pays {
		recs := WalkKeyframeWorld(pay)
		sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
		for i, r := range recs {
			if r.TI != ti || i+1 >= len(recs) {
				continue
			}
			tal.bounded++
			kfGramOne(reg, pay, r, recs[i+1].Bit, &tal)
		}
	}
	return tal
}

// kfGramOne mesure UN record : sa marche, puis le chainage vers `want`.
func kfGramOne(reg *Registry, pay []byte, r KeyframeRec, want int, tal *kfGramTally) {
	br := NewBitReader(pay)
	br.SetBitPos(r.Bit + keyframeRecordTIBit)
	tr := TraverseEntity(br, reg, 0)
	if tr.DesyncAt >= 0 {
		tal.desync++
		return
	}
	if tr.EndBit == want {
		tal.direct++
		return
	}
	tal.gaps[want-tr.EndBit]++
	if _, ok := readKeyframeHeader(pay, tr.EndBit, len(pay)*8); ok {
		tal.landedHdr++
	}
	ch := ChainKeyframeRecords(pay, reg, tr.EndBit, want, r.Slot)
	if !ch.Reached {
		tal.lost++
		tal.stops[ch.Stop]++
		return
	}
	tal.chained++
	tal.skipped += ch.Skipped
	tal.skippedF26 += ch.SkippedFieldNonZero
}

// kfGramLogArchetypes publie le classement des archetypes par frequence dans les tables
// d'image-cle — c'est lui qui justifie le choix du second archetype de C2.
func kfGramLogArchetypes(t *testing.T, pays [][]byte) {
	t.Helper()
	count := map[int]int{}
	total := 0
	for _, pay := range pays {
		for _, r := range WalkKeyframeWorld(pay) {
			count[r.TI]++
			total++
		}
	}
	tis := make([]int, 0, len(count))
	for ti := range count {
		tis = append(tis, ti)
	}
	sort.Slice(tis, func(i, j int) bool { return count[tis[i]] > count[tis[j]] })
	if len(tis) > 8 {
		tis = tis[:8]
	}
	t.Logf("  records de table (oracle) : %d, archetypes dominants :", total)
	for _, ti := range tis {
		t.Logf("    ti=%-3d %6d", ti, count[ti])
	}
}

// kfGramOffsetMax borne le balayage du decalage du CORPS depuis le debut d'un record. 64 est
// la largeur de l'en-tete etablie ; on balaie au-dela pour ne rien supposer.
const kfGramOffsetMax = 128

// TestKFGramOffset BALAIE le decalage du corps : a quel bit, depuis le debut d'un record, le
// lecteur de record NEW doit-il etre pose pour que sa marche atterrisse EXACTEMENT sur la
// frontiere suivante ? C'est le test de l'hypothese H2 (un bloc lu AVANT le corps), et il ne
// suppose rien : si aucun decalage ne marche, le corps n'est pas un record NEW.
func TestKFGramOffset(t *testing.T) {
	dir := os.Getenv(kfGramFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", kfGramFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	reg, pays := kfGramPayloads(t, dir)
	for _, ti := range kfGramTargets {
		lens, exact, tiOK, bounded := kfGramSweep(reg, pays, ti)
		t.Logf("---- ti=%d : %d records bornes ----", ti, bounded)
		kfGramLogTop(t, "longueur REELLE du record (want - Bit)", lens, 8)
		best, bestN := -1, 0
		for off, n := range exact {
			if n > bestN {
				best, bestN = off, n
			}
		}
		t.Logf("  meilleur decalage de corps : %d bits -> %d marches exactes sur %d",
			best, bestN, bounded)
		for off := 0; off < kfGramOffsetMax; off++ {
			if exact[off] > 0 || tiOK[off] > bounded/2 {
				t.Logf("    decalage %3d : ti relu correct %4d · marches exactes %4d",
					off, tiOK[off], exact[off])
			}
		}
	}
}

// kfGramSweep balaie les decalages de corps pour l'archetype ti et rend, par decalage, le
// nombre de marches exactes et de `ti` relus corrects, plus la distribution des longueurs
// reelles de record.
func kfGramSweep(reg *Registry, pays [][]byte, ti int) (
	lens map[int]int, exact, tiOK [kfGramOffsetMax]int, bounded int,
) {
	lens = map[int]int{}
	for _, pay := range pays {
		recs := WalkKeyframeWorld(pay)
		sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
		for i, r := range recs {
			if r.TI != ti || i+1 >= len(recs) {
				continue
			}
			want := recs[i+1].Bit
			bounded++
			lens[want-r.Bit]++
			for off := 0; off < kfGramOffsetMax; off++ {
				if int(kfReadBits(pay, r.Bit+off, 6)) == ti {
					tiOK[off]++
				}
				br := NewBitReader(pay)
				br.SetBitPos(r.Bit + off)
				if tr := TraverseEntity(br, reg, 0); tr.DesyncAt < 0 && tr.EndBit == want {
					exact[off]++
				}
			}
		}
	}
	return lens, exact, tiOK, bounded
}

// kfGramLogTop publie les `n` valeurs les plus frequentes d'une distribution.
func kfGramLogTop(t *testing.T, label string, hist map[int]int, n int) {
	t.Helper()
	type kv struct{ k, n int }
	xs := make([]kv, 0, len(hist))
	for k, v := range hist {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	if len(xs) > n {
		xs = xs[:n]
	}
	t.Logf("  %s : %d valeurs distinctes, dominantes :", label, len(hist))
	for _, x := range xs {
		t.Logf("    %6d bits : %4d fois", x.k, x.n)
	}
}

func TestKFGramGlobal(t *testing.T) {
	dir := os.Getenv(kfGramFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", kfGramFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	reg, pays := kfGramPayloads(t, dir)
	for k, lay := range EquipmentIdentityLayouts {
		restore := lay.apply()
		parsed, hit, oracle, stops := kfGramGlobalPass(reg, pays)
		restore()
		var pct float64
		if oracle > 0 {
			pct = 100 * float64(hit) / float64(oracle)
		}
		t.Logf("[%d] %s | records parses %6d · records de l'oracle retrouves %6d / %6d"+
			" (%.1f %%)", k, lay, parsed, hit, oracle, pct)
		for s, n := range stops {
			if n > 0 {
				t.Logf("      arret %-20s %4d payloads", KeyframeWalkStop(s), n)
			}
		}
	}
}

// kfGramGlobalPass traverse chaque payload de bout en bout avec le walker deterministe et
// compte les frontieres de l'oracle retrouvees, plus les causes d'arret.
func kfGramGlobalPass(reg *Registry, pays [][]byte) (parsed, hit, oracle int, stops [5]int) {
	for _, pay := range pays {
		recs, stop := WalkKeyframeRecords(pay, reg)
		stops[stop]++
		parsed += len(recs)
		at := make(map[int]bool, len(recs))
		for _, r := range recs {
			at[r.BitStart] = true
		}
		for _, o := range WalkKeyframeWorld(pay) {
			oracle++
			if at[o.Bit] {
				hit++
			}
		}
	}
	return parsed, hit, oracle, stops
}

// TestKFGramVariant BALAIE les huit lectures possibles du CORPS (etat par defaut / porte /
// masque), sous le corruption-check du mode film allume puis eteint. C'est le test de la
// lecture (a) de la decouverte 3 du lot R3 : l'image-cle porte-t-elle un ETAT COMPLET ?
func TestKFGramVariant(t *testing.T) {
	dir := os.Getenv(kfGramFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", kfGramFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	reg, pays := kfGramPayloads(t, dir)
	for _, ti := range kfGramTargets {
		t.Logf("---- ti=%d ----", ti)
		for _, corr := range []bool{false, true} {
			prev := filmComponentCorruptionCheck
			SetFilmComponentCorruptionCheck(corr)
			for k, v := range KeyframeBodyVariants {
				ex, ds, bd, gaps := kfGramVariantPass(reg, pays, ti, v)
				t.Logf("  corruption=%-5v [%d] %s | bornes %4d · exactes %4d · desync %4d",
					corr, k, v, bd, ex, ds)
				kfGramLogTop(t, "    ecart want-EndBit", gaps, 3)
			}
			SetFilmComponentCorruptionCheck(prev)
		}
	}
}

// kfGramVariantPass mesure UNE variante de corps sur tous les records bornes de l'archetype.
func kfGramVariantPass(reg *Registry, pays [][]byte, ti int, v KeyframeBodyVariant) (
	exact, desync, bounded int, gaps map[int]int,
) {
	gaps = map[int]int{}
	for _, pay := range pays {
		recs := WalkKeyframeWorld(pay)
		sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
		for i, r := range recs {
			if r.TI != ti || i+1 >= len(recs) {
				continue
			}
			bounded++
			tr := WalkKeyframeBody(pay, r.Bit, reg, v)
			switch {
			case tr.DesyncAt >= 0:
				desync++
			case tr.EndBit == recs[i+1].Bit:
				exact++
			default:
				gaps[recs[i+1].Bit-tr.EndBit]++
			}
		}
	}
	return exact, desync, bounded, gaps
}
