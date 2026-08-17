package filmdec

// keyframe_fullstate_loop_test.go — INSTRUMENT DU LOT R7-e
// (cf. .ai/V7.5/replay2d/PLAN_R7E_BOUCLE_ETAT_COMPLET.md).
//
// LA QUESTION : la boucle d'ETAT COMPLET du jeu (`FUN_142e2bfd0` -> `FUN_1428e2b68` ->
// `FUN_142e2c690`, lue par R7-d) portee TELLE QUELLE sur le payload type-2 atterrit-elle
// bit-exact ? Cinq variables, allumees UNE A LA FOIS :
//
//	(a) l'ORDRE des composants   — table nommee de 64 entrees contre ordre `chunk_00`
//	(b) l'EN-TETE par entite     — 108 bits + deux `R(32)` de taille, contre 64 bits
//	(c) le CONTROLE par composant — `R(1) [+R(32)]` sous le drapeau film
//	(d) `DAT_144e61ea0`          — vec3 brut 96 bits contre 3 x axisW quantifies
//	(e) `i0`                     — la grammaire de l'ECRIVAIN (`FUN_14320678c`)
//
// CE QU'IL NE FAIT PAS : il ne publie AUCUNE donnee, n'ecrit RIEN sur disque, ne touche a
// aucun schema. LECTURE SEULE, garde par KF35_ROOT (meme garde que R7-a/R7-b/R7-d).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestKF7E' -timeout 90m -v

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"os"
	"sort"
	"testing"
)

// kf7eInflate rend le chunk_00 DEFLATE d'un film, tel que `ParseRegistryChunk` le lit — mais
// en OCTETS BRUTS : c'est la seule facon de confronter le layout suppose par `registry.go`
// (`[u32 kind][u32 flags][nom @ +8]`) a celui que `FUN_142e2c690` lit en memoire
// (`[nom @ +0x00][u32 niveau @ +0x100]`, entree de 0x104, 64 par archetype).
func kf7eInflate(t *testing.T, dir string) []byte {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("zlib : %v", err)
	}
	dec, err := io.ReadAll(zr)
	if err != nil && len(dec) == 0 {
		t.Fatalf("inflate : %v", err)
	}
	return dec
}

// kf7eCString rend la chaine NUL-terminee a `off`, bornee a `max` octets.
func kf7eCString(d []byte, off, max int) string {
	if off < 0 || off >= len(d) {
		return ""
	}
	end := off + max
	if end > len(d) {
		end = len(d)
	}
	if i := bytes.IndexByte(d[off:end], 0); i >= 0 {
		return string(d[off : off+i])
	}
	return string(d[off:end])
}

// TestKF7ETableLayout tranche la variable (a) SANS supposer : il lit les octets bruts du bloc
// d'archetype du bipede et confronte les DEUX layouts. Si le layout du jeu est le bon, le
// `kind` que `registry.go` lit en `+0` est TOUJOURS nul (queue de bourrage du nom precedent)
// et le `flags` qu'il lit en `+4` est le NIVEAU du composant PRECEDENT — un decalage d'un cran
// sur toutes les largeurs quantifiees.
func TestKF7ETableLayout(t *testing.T) {
	root := os.Getenv(kf35RootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", kf35RootEnv)
	}
	d := kf7eInflate(t, root+"/"+kf35OracleFilms[0])
	t.Logf("chunk_00 inflate : %d octets · %% 0x4100 = %d · (len-8) %% 0x4100 = %d · blocs %d",
		len(d), len(d)%archetypeBlockSize, (len(d)-8)%archetypeBlockSize, len(d)/archetypeBlockSize)

	base := bipedDefaultStateTypeIndex * archetypeBlockSize
	nonZeroKind, nonZeroTail := 0, 0
	for s := 0; s < archetypeBlockSlots; s++ {
		off := base + s*registrySlotSize
		if off+registrySlotSize > len(d) {
			break
		}
		kind := binary.LittleEndian.Uint32(d[off:])
		flags := binary.LittleEndian.Uint32(d[off+4:])
		nameGo := kf7eCString(d, off+8, registrySlotSize-8)
		// Layout du JEU : l'entree k commence au nom que registry.go lit en +8, et son
		// niveau est le u32 situe 0x100 octets plus loin.
		lvlGame := uint32(0)
		if off+8+0x100+4 <= len(d) {
			lvlGame = binary.LittleEndian.Uint32(d[off+8+0x100:])
		}
		if kind != 0 {
			nonZeroKind++
		}
		if nameGo == "" {
			continue
		}
		nonZeroTail++
		if s < 12 || s >= 60 {
			t.Logf("  slot %2d | registry.go kind=%d flags=%d | JEU niveau(+0x100)=%d | %s",
				s, kind, flags, lvlGame, nameGo)
		}
	}
	t.Logf("  -> %d slots nommes · %d slots avec kind != 0 (layout registry.go)", nonZeroTail, nonZeroKind)
}

// TestKF7ELevelShift publie, composant par composant, le niveau que `registry.go` sert
// (`Flags[i]`) contre celui que le jeu passe au deser (`u32 @ entree_i + 0x100`, qui est le
// `Flags[i+1]` de `registry.go` si le layout du jeu est le bon). C'est la mesure de l'ecart,
// pas son postulat.
func TestKF7ELevelShift(t *testing.T) {
	root := os.Getenv(kf35RootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", kf35RootEnv)
	}
	for _, name := range kf35OracleFilms {
		d := kf7eInflate(t, root+"/"+name)
		reg, err := ParseRegistryChunk(d)
		if err != nil {
			t.Fatalf("registre %s : %v", name, err)
		}
		arch, ok := reg.Archetype(bipedDefaultStateTypeIndex)
		if !ok {
			t.Fatalf("archetype %d absent", bipedDefaultStateTypeIndex)
		}
		diff := 0
		for i := range arch.Components {
			if arch.Level(i) != arch.Level(i+1) {
				diff++
			}
		}
		t.Logf("[%s] %d composants · %d niveaux differents entre Flags[i] et Flags[i+1]",
			name, len(arch.Components), diff)
		for i, c := range arch.Components {
			if arch.Level(i) == arch.Level(i+1) {
				continue
			}
			t.Logf("    i%-2d %-58s registry.go L=%d | JEU L=%d", i, c, arch.Level(i), arch.Level(i+1))
		}
	}
}

// ---------------------------------------------------------------------------------------
// LA MESURE — les cinq variables, allumees une a une.
// ---------------------------------------------------------------------------------------

// kf7eCase est UNE configuration mesuree : un libelle, les options de marche, et les
// bascules globales qu'elle installe.
type kf7eCase struct {
	Label string
	Opt   KeyframeFullStateOpt
	Corr  bool // (c) le controle par composant du mode film
	I0    bool // (e) la grammaire d'ECRIVAIN d'`i0`
	Scope bool // (d) la portee `DAT_144e61ea0` (vec3 brut 96 bits au lieu du quantifie)
}

// kf7eTally compte ce qu'une configuration a rencontre sur un film. Memes denominateurs et
// memes definitions que `kf35Tally` (R7-a/R7-b) : `bounded` est le denominateur publie.
type kf7eTally struct {
	bounded, exact, chained, desync, lost int
	consumed, absGaps                     []int
	breaks                                map[string]int
}

func newKF7ETally() kf7eTally { return kf7eTally{breaks: map[string]int{}} }

func (k kf7eTally) rate() float64 {
	if k.bounded == 0 {
		return 0
	}
	return 100 * float64(k.exact+k.chained) / float64(k.bounded)
}

// kf7eWalkOne rejoue le corps d'UN record sous la configuration donnee, puis mesure.
func kf7eWalkOne(f kf35Film, pay []byte, b kf35Bound, c kf7eCase, tal *kf7eTally) {
	tr := WalkKeyframeFullState(pay, b.Rec.Bit, f.Reg, c.Opt)
	if tr.DesyncAt >= 0 {
		tal.desync++
		return
	}
	tal.consumed = append(tal.consumed, tr.EndBit-b.Rec.Bit)
	if tr.EndBit == b.Want {
		tal.exact++
		return
	}
	gap := b.Want - tr.EndBit
	if gap < 0 {
		gap = -gap
	}
	tal.absGaps = append(tal.absGaps, gap)
	tal.breaks[kf35Break(tr, b.Want)]++
	if kf7eChain(f, pay, tr.EndBit, b, c) {
		tal.chained++
		return
	}
	tal.lost++
}

// kf7eChain enchaine la marche SOUS LA MEME CONFIGURATION jusqu'a la frontiere visee : c'est
// le rattrapage des records que le filtre fort du balayeur ne voit pas (meme borne que R7-a).
func kf7eChain(f kf35Film, pay []byte, from int, b kf35Bound, c kf7eCase) bool {
	total := len(pay) * 8
	pos, prev := from, b.Rec.Slot
	for n := 0; n < kf35ChainMax; n++ {
		if pos == b.Want {
			return true
		}
		if pos > b.Want || pos+keyframeHeaderBits > total {
			return false
		}
		h, ok := readKeyframeHeader(pay, pos, total)
		if !ok || h.Slot <= prev {
			return false
		}
		tr := WalkKeyframeFullState(pay, pos, f.Reg, c.Opt)
		if tr.DesyncAt >= 0 {
			return false
		}
		pos, prev = tr.EndBit, h.Slot
	}
	return false
}

// kf7ePass mesure UNE configuration sur UN film, bascules globales installees et restaurees.
func kf7ePass(f kf35Film, c kf7eCase) kf7eTally {
	prevCorr, prevI0 := filmComponentCorruptionCheck, keyframeWriterI0Grammar
	prevScope := SetKeyframeBaselineScope(c.Scope)
	SetFilmComponentCorruptionCheck(c.Corr)
	SetKeyframeWriterI0Grammar(c.I0)
	defer func() {
		SetFilmComponentCorruptionCheck(prevCorr)
		SetKeyframeWriterI0Grammar(prevI0)
		SetKeyframeBaselineScope(prevScope)
	}()
	tal := newKF7ETally()
	for _, pay := range f.Pays {
		for _, b := range kf35BoundedRecs(pay) {
			tal.bounded++
			kf7eWalkOne(f, pay, b, c, &tal)
		}
	}
	return tal
}

// kf7eCases construit la matrice A/B : une REFERENCE (la lecture v4 de R7-a/R7-d portee par
// la nouvelle marche, en-tete 64 bits, rien d'autre), puis chaque variable SEULE, puis les
// cumuls dans l'ordre du plan.
func kf7eCases() []kf7eCase {
	ref := KeyframeFullStateOpt{HeaderBits: keyframeHeaderBits}
	lvl := KeyframeFullStateOpt{HeaderBits: keyframeHeaderBits, LevelShift: true}
	hdr := KeyframeFullStateOpt{HeaderBits: keyframeFullStateHeaderBits}
	hdrSz := KeyframeFullStateOpt{HeaderBits: keyframeFullStateHeaderBits, SizeWords: true}
	hdrSzDs := KeyframeFullStateOpt{HeaderBits: keyframeFullStateHeaderBits, SizeWords: true, DefaultState: true}
	all := KeyframeFullStateOpt{HeaderBits: keyframeFullStateHeaderBits, SizeWords: true, LevelShift: true}
	return []kf7eCase{
		{Label: "REF    en-tete 64, sans rien (= v4 de R7-a/R7-d)", Opt: ref},
		{Label: "(a)    REF + niveaux decales (layout du JEU)", Opt: lvl},
		{Label: "(b1)   en-tete 108 bits SEUL", Opt: hdr},
		{Label: "(b2)   en-tete 108 + deux R(32) de taille", Opt: hdrSz},
		{Label: "(b3)   en-tete 108 + tailles + etat par defaut", Opt: hdrSzDs},
		{Label: "(c)    REF + controle par composant", Opt: ref, Corr: true},
		{Label: "(e)    REF + grammaire d'ECRIVAIN d'i0", Opt: ref, I0: true},
		{Label: "(a+e)  REF + niveaux decales + i0 ecrivain", Opt: lvl, I0: true},
		{Label: "(b2+e) en-tete 108 + tailles + i0 ecrivain", Opt: hdrSz, I0: true},
		{Label: "(a+b2+e) niveaux + en-tete 108 + tailles + i0", Opt: all, I0: true},
		{Label: "(a+b2+c+e) TOUT sauf l'etat par defaut", Opt: all, Corr: true, I0: true},
		{Label: "(d)    REF + portee DAT_144e61ea0 (brut 96)", Opt: ref, Scope: true},
		{Label: "(d+e)  REF + portee + i0 ecrivain", Opt: ref, I0: true, Scope: true},
		{Label: "(b2+d+e) en-tete 108 + tailles + portee + i0", Opt: hdrSz, I0: true, Scope: true},
		{Label: "(b3+c+e) TOUT (etat par defaut compris)", Opt: hdrSzDs, Corr: true, I0: true},
	}
}

// TestKF7EFullStateLoop est LA MESURE de la phase 2 : chaque configuration, sur les trois
// films du corpus ferme, largeurs de la carte installees et trous neutralises.
func TestKF7EFullStateLoop(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)

	for _, f := range films {
		kf7eOneFilm(t, f)
	}
}

// kf7eOneFilm joue toute la matrice sur UN film.
func kf7eOneFilm(t *testing.T, f kf35Film) {
	t.Helper()
	_, restorePrec := kf35bInstallPrecision(t, f.Name)
	defer restorePrec()
	stubbed, restoreStubs := kf35ApplyStubs(f, kf7dVariant)
	defer restoreStubs()
	t.Logf("======== %s — composants neutralises : %d ========", f.Name, len(stubbed))

	for _, c := range kf7eCases() {
		tal := kf7ePass(f, c)
		t.Logf("  %-42s exactes %4d · chainees %4d · desync %4d / %4d | ATTERRISSAGE %5.2f %%"+
			" | longueur MEDIANE %5d · ecart absolu MEDIAN %5d",
			c.Label, tal.exact, tal.chained, tal.desync, tal.bounded, tal.rate(),
			kf35Median(tal.consumed), kf35Median(tal.absGaps))
		kf7eLogBreaks(t, tal.breaks, 4)
	}
}

// kf7eLogBreaks publie les `n` points de decrochage les plus frequents.
func kf7eLogBreaks(t *testing.T, hist map[string]int, n int) {
	t.Helper()
	if len(hist) == 0 {
		return
	}
	type kv struct {
		k string
		n int
	}
	xs := make([]kv, 0, len(hist))
	for k, v := range hist {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	if len(xs) > n {
		xs = xs[:n]
	}
	for _, x := range xs {
		t.Logf("        %-62s %4d fois", x.k, x.n)
	}
}

// TestKF7EProfileI0 publie la largeur consommee par `i0` sous la grammaire d'ECRIVAIN, face a
// la largeur PREDITE par le decoupage de la carte. Sans lui, « (e) ameliore » ne se verifie pas.
func TestKF7EProfileI0(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)
	prevCorr := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false)
	defer SetFilmComponentCorruptionCheck(prevCorr)

	for _, f := range films {
		kf7eProfileOne(t, f)
	}
}

func kf7eProfileOne(t *testing.T, f kf35Film) {
	t.Helper()
	lay, restorePrec := kf35bInstallPrecision(t, f.Name)
	defer restorePrec()
	_, restoreStubs := kf35ApplyStubs(f, kf7dVariant)
	defer restoreStubs()

	for _, on := range []bool{false, true} {
		prev := keyframeWriterI0Grammar
		SetKeyframeWriterI0Grammar(on)
		stats := kf35bProfile(f, kf7dVariant)
		SetKeyframeWriterI0Grammar(prev)
		for _, s := range stats {
			if s.Name != kf7dI0 {
				continue
			}
			t.Logf("  [%s] i0 ecrivain=%-5v · vu %d fois · largeur MEDIANE %d bits"+
				" · franchissements %d | PREDITE (6 + %d+%d+%d) = %d",
				f.Name, on, s.Seen, kf35Median(s.Bits), s.Break,
				lay.AxisW[0], lay.AxisW[1], lay.AxisW[2], kf7dPredicted(lay))
			break
		}
	}
}
