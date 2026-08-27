package filmdec

// keyframe_objective_fullstate_test.go — INSTRUMENT DU LOT ti11-cadre
// (cf. .ai/V7.5/PLAN_TI11_TEST_CADRE_2026-08-27.md, protocole
// .ai/V7.5/replay2d/registre_film/TI11_PROTOCOLE.md).
//
// LA QUESTION, UNE SEULE : le corps d'un record d'IMAGE-CLE (payload type-2) se lit-il par la
// boucle d'ETAT COMPLET du jeu (WalkKeyframeFullState, portee de FUN_142e2bfd0/FUN_142e2c690)
// pour l'archetype ti=11 (managed-objective) ? Les feuilles ti=11 cablees
// (components_managed_objective.go) sont TRIVIALES et resolues au Ghidra : un echec de landing
// localise donc le mur dans le CADRE, pas dans les feuilles (ce que ti=35 ne pouvait pas isoler).
//
// L'ORACLE. La frontiere du record SUIVANT vient d'une chaine DISJOINTE de toute lecture de
// corps : WalkKeyframeWorld (en-tete 64 bits [id:32][field:26][ti:6], 249/250 vs Cheat Engine).
// Une marche juste atterrit BIT-EXACT sur cette frontiere.
//
// LES FEUILLES NON RESOLUES (i2/i4/i6/i7/i8/i9/i10/i11/i32/i33) sont NEUTRALISEES a 0 bit
// (SetUnportedStubWidth, herite du harnais biped) pour ISOLER le cadre : c'est le meme
// dispositif que la variante v4 du biped. Un pass DIAGNOSTIC sans stub publie ou la marche
// decroche (si elle atteint i2 systematiquement, i0/i1 consomment plausiblement).
//
// LECTURE SEULE, garde par TI11_ROOT : saute partout ailleurs (CI comprise). UN SEUL decodage
// filmdec par process (bascules globales) : le verrou est pris pour tout le test. Un film
// charge a la fois (borne memoire).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI11_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestTI11' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// sortKeyframeRecsByBit trie les records par position bit croissante (defensif).
func sortKeyframeRecsByBit(recs []KeyframeRec) {
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
}

const (
	ti11RootEnv  = "TI11_ROOT"
	ti11FilmsEnv = "TI11_FILMS"
	// ti11ChainMax borne les records intercales explores (meme borne que le harnais biped).
	// ti11TypeIndex (= 11) est defini par sonde_ti11_objectifs_test.go (meme paquet).
	ti11ChainMax = kf35ChainMax
)

// ti11CorpusT1 est le corpus de MESURE du cadre : un film par famille distincte portant des
// records ti=11 BORNES (le gate exige >= 2 familles). MESURE d'inventaire (2026-08-27) : les
// trois films Strongholds testes (7344d24f, 696a9d7c, 10ed320d) rendent ZERO record ti=11 borne
// par WalkKeyframeWorld (l'oracle ne reconstruit pas de record ti=11 adjacent dans leurs
// keyframes) — ils sont donc ecartes du corpus de cadre. Oddball, CTF et KOTH en fournissent
// abondamment (637 / 501 / 89 cumules).
var ti11CorpusT1 = []string{
	"24dbb67d", // Oddball  (115 records ti=11 bornes)
	"64e8adfa", // CTF Catalyst (201)
	"01e1f945", // KOTH Catalyst (25)
}

// ti11FilmNames rend la liste de films a mesurer (defaut = corpus T1, surcharge TI11_FILMS).
func ti11FilmNames() []string {
	if s := os.Getenv(ti11FilmsEnv); s != "" {
		var out []string
		for _, n := range strings.Split(s, ",") {
			if n = strings.TrimSpace(n); n != "" {
				out = append(out, n)
			}
		}
		return out
	}
	return ti11CorpusT1
}

// ti11Root rend la racine des films ou saute le test si la garde d'environnement est absente.
func ti11Root(t *testing.T) string {
	t.Helper()
	root := os.Getenv(ti11RootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", ti11RootEnv)
	}
	return root
}

// ti11Bound est un record ti=11 avec la frontiere annoncee par l'oracle (record suivant).
type ti11Bound struct {
	Rec  KeyframeRec
	Want int
}

// ti11BoundedRecs rend les records ti=11 d'un payload bornes par leur voisin immediat dans la
// liste reconstruite par WalkKeyframeWorld (chaine DISJOINTE de toute lecture de corps).
func ti11BoundedRecs(pay []byte) []ti11Bound {
	recs := WalkKeyframeWorld(pay)
	// WalkKeyframeWorld rend deja les records dans l'ordre du flux (pos croissant) ; on trie
	// par securite, comme le harnais biped.
	sortKeyframeRecsByBit(recs)
	var out []ti11Bound
	for i, r := range recs {
		if r.TI == ti11TypeIndex && i+1 < len(recs) {
			out = append(out, ti11Bound{Rec: r, Want: recs[i+1].Bit})
		}
	}
	return out
}

// ti11UnportedComponents rend les composants de l'archetype ti=11 que le dispatch NE porte PAS
// (probe a flux nul) : ce sont ceux a neutraliser a 0 bit pour isoler le cadre.
func ti11UnportedComponents(reg *Registry) []string {
	arch, ok := reg.Archetype(ti11TypeIndex)
	if !ok {
		return nil
	}
	var out []string
	for i, name := range arch.Components {
		// ti11ComponentIsPorted interroge le dispatch reel (sonde_ti11_objectifs_test.go).
		if !ti11ComponentIsPorted(name, ti11TypeIndex, arch.Level(i)) {
			out = append(out, name)
		}
	}
	return out
}

// ti11StubUnported neutralise a 0 bit tous les composants non portes de ti=11 et rend la
// fonction de restauration. Sans stub, la marche decroche au premier non porte (i2).
func ti11StubUnported(reg *Registry) (names []string, restore func()) {
	names = ti11UnportedComponents(reg)
	for _, n := range names {
		SetUnportedStubWidth(n, 0)
	}
	return names, func() {
		for _, n := range names {
			SetUnportedStubWidth(n, -1)
		}
	}
}

// ti11Tally compte ce qu'une variante a rencontre. `bounded` est le denominateur publie.
type ti11Tally struct {
	bounded  int
	exact    int
	chained  int
	desync   int
	lost     int
	consumed []int
	absGaps  []int
	desyncs  map[string]int
}

func newTI11Tally() ti11Tally { return ti11Tally{desyncs: map[string]int{}} }

func (k ti11Tally) rate() float64 {
	if k.bounded == 0 {
		return 0
	}
	return 100 * float64(k.exact+k.chained) / float64(k.bounded)
}

// ti11CadreVariant : une lecture du corps par la boucle d'etat complet (variable de cadre R7-e).
type ti11CadreVariant struct {
	Label string
	Opt   KeyframeFullStateOpt
}

// ti11CadreVariants : les variables de cadre, allumees une a une puis combinees jusqu'au
// cadre COMPLET du jeu (FUN_142e2bfd0 : en-tete 108 + mots de taille + etat par defaut).
var ti11CadreVariants = []ti11CadreVariant{
	{"C0 en-tete 64 nu", KeyframeFullStateOpt{HeaderBits: 64}},
	{"C1 en-tete 108 nu", KeyframeFullStateOpt{HeaderBits: 108}},
	{"C2 108 + etat par defaut", KeyframeFullStateOpt{HeaderBits: 108, DefaultState: true}},
	{"C3 108 + mots de taille", KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true}},
	{"C4 108 + taille + etat par defaut (cadre JEU FUN_142e2bfd0)",
		KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true, DefaultState: true}},
	{"C5 cadre JEU + LevelShift",
		KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true, DefaultState: true, LevelShift: true}},
	{"C6 en-tete 64 + etat par defaut", KeyframeFullStateOpt{HeaderBits: 64, DefaultState: true}},
}

// ti11WalkOne mesure UN record sous une variante de cadre (stub deja arme). Chaine si besoin.
func ti11WalkOne(reg *Registry, pay []byte, b ti11Bound, opt KeyframeFullStateOpt, tal *ti11Tally) {
	tr := WalkKeyframeFullState(pay, b.Rec.Bit, reg, opt)
	if tr.DesyncAt >= 0 {
		tal.desync++
		if arch, ok := reg.Archetype(ti11TypeIndex); ok && tr.DesyncAt < len(arch.Components) {
			tal.desyncs[fmt.Sprintf("i%d %s", tr.DesyncAt, arch.Components[tr.DesyncAt])]++
		}
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
	if ti11Chain(reg, pay, tr.EndBit, b, opt) {
		tal.chained++
		return
	}
	tal.lost++
}

// ti11Chain enchaine la marche SOUS LA MEME variante jusqu'a la frontiere visee (rattrapage des
// records que le filtre fort de l'oracle ne voit pas). Meme forme que le harnais biped.
func ti11Chain(reg *Registry, pay []byte, from int, b ti11Bound, opt KeyframeFullStateOpt) bool {
	total := len(pay) * 8
	pos, prev := from, b.Rec.Slot
	for n := 0; n < ti11ChainMax; n++ {
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
		tr := WalkKeyframeFullState(pay, pos, reg, opt)
		if tr.DesyncAt >= 0 {
			return false
		}
		pos, prev = tr.EndBit, h.Slot
	}
	return false
}

// ti11PayBounds associe un payload d'image-cle a ses records ti=11 bornes (pre-calcul unique :
// WalkKeyframeWorld est couteux, on ne le rejoue pas par variante).
type ti11PayBounds struct {
	pay    []byte
	bounds []ti11Bound
}

// ti11ComputeBounds pre-calcule, une seule fois par film, les records ti=11 bornes de chaque
// payload d'image-cle (garde les payloads porteurs de records).
func ti11ComputeBounds(pays [][]byte) []ti11PayBounds {
	var out []ti11PayBounds
	for _, pay := range pays {
		if b := ti11BoundedRecs(pay); len(b) > 0 {
			out = append(out, ti11PayBounds{pay: pay, bounds: b})
		}
	}
	return out
}

// ti11PassCadre mesure une variante de cadre sur un film (stub arme par l'appelant).
func ti11PassCadre(reg *Registry, fb []ti11PayBounds, opt KeyframeFullStateOpt) ti11Tally {
	tal := newTI11Tally()
	for _, pb := range fb {
		for _, b := range pb.bounds {
			tal.bounded++
			ti11WalkOne(reg, pb.pay, b, opt, &tal)
		}
	}
	return tal
}

// ti11PassWitness mesure le TEMOIN record NEW (WalkKeyframeBody masque-garde) sur un film.
func ti11PassWitness(reg *Registry, fb []ti11PayBounds, v KeyframeBodyVariant) ti11Tally {
	tal := newTI11Tally()
	for _, pb := range fb {
		for _, b := range pb.bounds {
			tal.bounded++
			tr := WalkKeyframeBody(pb.pay, b.Rec.Bit, reg, v)
			if tr.DesyncAt >= 0 {
				tal.desync++
				continue
			}
			tal.consumed = append(tal.consumed, tr.EndBit-b.Rec.Bit)
			if tr.EndBit == b.Want {
				tal.exact++
				continue
			}
			tal.lost++
		}
	}
	return tal
}

// TestTI11Inventory publie le denominateur : archetype ti=11, ses composants, leur portage, et
// le nombre de records ti=11 BORNES par film. Interdit de SUPPOSER le corpus.
func TestTI11Inventory(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	for _, name := range ti11FilmNames() {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		arch, ok := f.Reg.Archetype(ti11TypeIndex)
		if !ok {
			t.Errorf("film %s : archetype ti=%d absent du registre", name, ti11TypeIndex)
			continue
		}
		unported := ti11UnportedComponents(f.Reg)
		n := 0
		for _, pay := range f.Pays {
			n += len(ti11BoundedRecs(pay))
		}
		t.Logf("film %s : %d composants ti=11 · %d non portes · %d images-cle · %d records ti=11 BORNES",
			name, len(arch.Components), len(unported), len(f.Pays), n)
		if len(unported) > 0 {
			t.Logf("    non portes (stubbes a 0) : %s", strings.Join(unported, " "))
		}
	}
}

// TestTI11CadreLanding est LA MESURE du gate T1 : atterrissage bit-exact des records ti=11 par
// la boucle d'etat complet (stub des non portes), balayage des variables de cadre, sous
// corruption-check eteint puis allume, avec le TEMOIN record NEW. Cumul multi-familles publie.
func TestTI11CadreLanding(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	names := ti11FilmNames()
	// Cumuls par variante (label -> tally cumule), pour le verdict multi-familles.
	cumCadre := map[string]*ti11Tally{}
	var cumWitness ti11Tally
	cumWitness.desyncs = map[string]int{}

	// Boucle films EXTERNE : chaque film n'est charge et ses bornes calculees (WalkKeyframeWorld,
	// couteux) QU'UNE fois ; corr est la variable interne.
	for _, name := range names {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		fb := ti11ComputeBounds(f.Pays)
		stubbed, restore := ti11StubUnported(f.Reg)
		for _, corr := range []bool{false, true} {
			prev := filmComponentCorruptionCheck
			SetFilmComponentCorruptionCheck(corr)
			// Temoin record NEW (mesure sous le meme regime de stub).
			w := ti11PassWitness(f.Reg, fb, KeyframeBodyVariant{DefaultState: true, Gate: true, Mask: true})
			t.Logf("  [%s corr=%v] TEMOIN record NEW : bornes %d · exactes %d · desync %d | %.2f %% (stub %d)",
				name, corr, w.bounded, w.exact, w.desync, w.rate(), len(stubbed))
			if !corr {
				accumulateTI11(&cumWitness, w)
			}
			for _, v := range ti11CadreVariants {
				tal := ti11PassCadre(f.Reg, fb, v.Opt)
				t.Logf("  [%s corr=%v] %s : bornes %d · exactes %d · chainees %d · perdues %d · desync %d | ATTERRISSAGE %.2f %%",
					name, corr, v.Label, tal.bounded, tal.exact, tal.chained, tal.lost, tal.desync, tal.rate())
				key := fmt.Sprintf("corr=%v %s", corr, v.Label)
				c := cumCadre[key]
				if c == nil {
					c = &ti11Tally{desyncs: map[string]int{}}
					cumCadre[key] = c
				}
				accumulateTI11(c, tal)
			}
			SetFilmComponentCorruptionCheck(prev)
		}
		restore()
	}

	t.Logf("======== CUMUL MULTI-FAMILLES (%d films) ========", len(names))
	t.Logf("  TEMOIN record NEW cumule : bornes %d · exactes %d | %.2f %%",
		cumWitness.bounded, cumWitness.exact, cumWitness.rate())
	for _, corr := range []bool{false, true} {
		for _, v := range ti11CadreVariants {
			key := fmt.Sprintf("corr=%v %s", corr, v.Label)
			if c := cumCadre[key]; c != nil {
				t.Logf("  %s : bornes %d · exactes %d · chainees %d | ATTERRISSAGE %.2f %%",
					key, c.bounded, c.exact, c.chained, c.rate())
			}
		}
	}
}

// accumulateTI11 additionne un tally de film dans un cumul.
func accumulateTI11(dst *ti11Tally, src ti11Tally) {
	dst.bounded += src.bounded
	dst.exact += src.exact
	dst.chained += src.chained
	dst.lost += src.lost
	dst.desync += src.desync
}

// TestTI11DiagnosticNoStub publie, SANS stub, ou la marche decroche (quel composant) pour le
// cadre JEU (C4). Si elle atteint i2 systematiquement, i0/i1 consomment plausiblement — ce qui
// distingue « cadre faux des le depart » de « cadre juste mais feuilles non resolues ».
func TestTI11DiagnosticNoStub(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	opt := KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true, DefaultState: true}
	for _, name := range ti11FilmNames() {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		fb := ti11ComputeBounds(f.Pays)
		tal := ti11PassCadre(f.Reg, fb, opt) // sans stub : les non portes desyncent
		t.Logf("  [%s] cadre JEU sans stub : bornes %d · exactes %d · desync %d",
			name, tal.bounded, tal.exact, tal.desync)
		kf35LogTopStr(t, "composant de decrochage (sans stub)", tal.desyncs, 8)
	}
}

// TestTI11GapProfile tranche (a) « cadre faux » vs (b) « cadre juste, feuilles non resolues
// manquantes » : en mode STUB (feuilles non resolues a 0) pour le cadre JEU C4, il publie la
// longueur REELLE mediane du record, la longueur CONSOMMEE mediane, et l'histogramme de l'ecart
// SIGNE (want - EndBit). Une sous-lecture BORNEE et systematique (ecart > 0 petit, ~ la somme
// des largeurs des feuilles manquantes) pointerait (b) ; un ecart disperse ou negatif pointe (a).
func TestTI11GapProfile(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	opt := KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true, DefaultState: true}
	for _, name := range ti11FilmNames() {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		fb := ti11ComputeBounds(f.Pays)
		_, restore := ti11StubUnported(f.Reg)
		var reals, consumed, signedGaps []int
		gapHist := map[int]int{}
		for _, pb := range fb {
			for _, b := range pb.bounds {
				reals = append(reals, b.Want-b.Rec.Bit)
				tr := WalkKeyframeFullState(pb.pay, b.Rec.Bit, f.Reg, opt)
				if tr.DesyncAt >= 0 {
					continue
				}
				consumed = append(consumed, tr.EndBit-b.Rec.Bit)
				g := b.Want - tr.EndBit // > 0 = sous-lecture, < 0 = sur-lecture
				signedGaps = append(signedGaps, g)
				gapHist[g]++
			}
		}
		restore()
		t.Logf("  [%s] cadre JEU C4 (stub) : records %d · longueur REELLE mediane %d · longueur CONSOMMEE mediane %d · ecart SIGNE median %d",
			name, len(reals), kf35Median(reals), kf35Median(consumed), kf35Median(signedGaps))
		kf35LogTopInt(t, "ecart signe want-EndBit (bits), + = sous-lecture", gapHist, 10)
	}
}

// ti11WinningCadre est la variante de cadre qui LANDE (C5 : en-tete 108 + mots de taille + etat
// par defaut + LevelShift), etablie par TestTI11CadreLanding (gate T1 tenu). T2 s'appuie dessus.
var ti11WinningCadre = KeyframeFullStateOpt{HeaderBits: 108, SizeWords: true, DefaultState: true, LevelShift: true}

// TestTI11B1AutoCoherence — GATE T2.1 (B1) : les GlobalID lus (i3 object-reference, i16-31
// sous-objectifs) sont-ils des ENTITES STABLES DANS LE TEMPS ? On ne confronte a aucun oracle
// externe : c'est une auto-coherence (gratuite). Pour chaque objectif (Gen,Slot), i3 doit garder
// la MEME valeur image-cle apres image-cle (un objectif suit le meme objet physique). Denominateur
// = records ti=11 qui ATTERRISSENT bit-exact sous le cadre gagnant. TEMOIN : des GlobalID 32b
// tires au hasard ne doivent PAS tomber dans l'ensemble observe (<= 1 %). Gate : stabilite+validite
// >= 90 %.
func TestTI11B1AutoCoherence(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()
	prev := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false) // le cadre gagnant est corr=false
	defer SetFilmComponentCorruptionCheck(prev)

	// Capture par le hook nomme de ti=11 : i3 et les 16 slots i16-31 du record en cours.
	var curI3 uint64
	var curI3Set bool
	var curSubs []uint64
	SetManagedObjectiveHook(func(f ManagedObjectiveField, values []uint64) {
		switch f {
		case ManagedObjectiveObjectRef:
			if len(values) > 0 {
				curI3, curI3Set = values[0], true
			}
		case ManagedObjectiveSubEntity:
			if len(values) > 0 {
				curSubs = append(curSubs, values[0])
			}
		}
	})
	defer SetManagedObjectiveHook(nil)

	var cumLanded, cumWithI3, cumStable, cumValid int
	observed := map[uint64]bool{} // ensemble des i3 observes (pour le temoin)
	for _, name := range ti11FilmNames() {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		fb := ti11ComputeBounds(f.Pays)
		bySlot := map[[2]int][]uint64{} // (Gen,Slot) -> suite des i3 dans le temps
		landed, withI3 := 0, 0
		for _, pb := range fb {
			for _, b := range pb.bounds {
				curI3, curI3Set, curSubs = 0, false, curSubs[:0]
				tr := WalkKeyframeFullState(pb.pay, b.Rec.Bit, f.Reg, ti11WinningCadre)
				if tr.DesyncAt >= 0 || tr.EndBit != b.Want {
					continue // record non atterri : son etat n'est pas fiable
				}
				landed++
				if !curI3Set {
					continue // i3 absent du masque de ce record
				}
				withI3++
				key := [2]int{b.Rec.Gen, b.Rec.Slot}
				bySlot[key] = append(bySlot[key], curI3)
				observed[curI3] = true
			}
		}
		// Stabilite : part des records dont l'i3 == la valeur MODALE de son (Gen,Slot).
		stable, valid := 0, 0
		for _, seq := range bySlot {
			modeVal, modeCnt := ti11Mode(seq)
			stable += modeCnt
			for _, v := range seq {
				if v != 0 {
					valid++
				}
			}
			_ = modeVal
		}
		rate := func(n int) float64 {
			if withI3 == 0 {
				return 0
			}
			return 100 * float64(n) / float64(withI3)
		}
		t.Logf("  [%s] records atterris %d · avec i3 present %d · objectifs distincts %d | STABILITE i3 %.1f %% · VALIDITE (i3!=0) %.1f %%",
			name, landed, withI3, len(bySlot), rate(stable), rate(valid))
		cumLanded += landed
		cumWithI3 += withI3
		cumStable += stable
		cumValid += valid
	}
	// TEMOIN : 4096 GlobalID 32b tires au hasard (graine fixe) — combien tombent dans l'ensemble
	// observe ? Un ensemble d'entites reelles est creux dans 2^32, donc ~0 %.
	rng := newTI11Rand(0x71114221)
	const draws = 4096
	hits := 0
	for i := 0; i < draws; i++ {
		if observed[uint64(rng())] {
			hits++
		}
	}
	t.Logf("======== CUMUL B1 (%d films) ========", len(ti11FilmNames()))
	t.Logf("  records atterris %d · avec i3 %d · STABILITE i3 cumulee %s · VALIDITE cumulee %s",
		cumLanded, cumWithI3, ti11Pct(cumStable, cumWithI3), ti11Pct(cumValid, cumWithI3))
	t.Logf("  TEMOIN : %d GlobalID 32b au hasard, %d dans l'ensemble observe (|obs|=%d) | %.3f %%",
		draws, hits, len(observed), 100*float64(hits)/float64(draws))
}

// TestTI11FieldDiversity — DIAGNOSTIC d'ALIGNEMENT INTERNE. L'atterrissage bit-exact ne teste
// que le TOTAL de bits consommes (reserve D2 de la spec) ; il ne prouve pas que CHAQUE feuille lit
// son vrai champ. Ce diagnostic publie, pour chaque champ capture (i1 couleur, i3 porteur, i5
// type, i12/i13 progression, i16-31 sous-objectifs), le nombre de valeurs DISTINCTES et un
// echantillon, sur les records qui atterrissent. Un champ qui varie comme attendu (i12 progression
// change dans le temps ; i5 type constant par objectif ; i1 change de camp) atteste l'alignement ;
// un champ FIGE la ou il devrait varier signale un mesalignement interne (ou un champ non peuple).
func TestTI11FieldDiversity(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()
	prev := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false)
	defer SetFilmComponentCorruptionCheck(prev)

	cur := map[ManagedObjectiveField][]uint64{}
	SetManagedObjectiveHook(func(f ManagedObjectiveField, values []uint64) {
		cur[f] = append(cur[f], values...)
	})
	defer SetManagedObjectiveHook(nil)

	fields := []ManagedObjectiveField{
		ManagedObjectiveColor, ManagedObjectiveObjectRef, ManagedObjectiveType,
		ManagedObjectiveProgress, ManagedObjectiveRequired, ManagedObjectiveParent,
		ManagedObjectiveSubEntity,
	}
	for _, name := range ti11FilmNames() {
		f := kf35Load(t, root+"/"+name)
		f.Name = name
		fb := ti11ComputeBounds(f.Pays)
		distinct := map[ManagedObjectiveField]map[uint64]int{}
		for _, fld := range fields {
			distinct[fld] = map[uint64]int{}
		}
		landed := 0
		for _, pb := range fb {
			for _, b := range pb.bounds {
				for k := range cur {
					delete(cur, k)
				}
				tr := WalkKeyframeFullState(pb.pay, b.Rec.Bit, f.Reg, ti11WinningCadre)
				if tr.DesyncAt >= 0 || tr.EndBit != b.Want {
					continue
				}
				landed++
				for _, fld := range fields {
					for _, v := range cur[fld] {
						distinct[fld][v]++
					}
				}
			}
		}
		t.Logf("  [%s] records atterris %d :", name, landed)
		for _, fld := range fields {
			t.Logf("      %-42s : %3d valeurs distinctes %s", fld, len(distinct[fld]), ti11SampleVals(distinct[fld], 6))
		}
	}
}

// ti11SampleVals rend un echantillon (au plus k) des valeurs les plus frequentes d'un histogramme.
func ti11SampleVals(h map[uint64]int, k int) string {
	type kv struct {
		v uint64
		n int
	}
	xs := make([]kv, 0, len(h))
	for v, n := range h {
		xs = append(xs, kv{v, n})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	s := "["
	for i := 0; i < len(xs) && i < k; i++ {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%d×%d", xs[i].v, xs[i].n)
	}
	if len(xs) > k {
		s += " ..."
	}
	return s + "]"
}

// ti11Mode rend la valeur la plus frequente d'une suite et son compte.
func ti11Mode(vs []uint64) (uint64, int) {
	c := map[uint64]int{}
	var best uint64
	bestN := 0
	for _, v := range vs {
		c[v]++
		if c[v] > bestN {
			best, bestN = v, c[v]
		}
	}
	return best, bestN
}

// ti11Pct formate n/d en pourcentage lisible (garde le denominateur visible).
func ti11Pct(n, d int) string {
	if d == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}

// newTI11Rand rend un generateur xorshift32 deterministe (graine != 0) pour le temoin B1.
func newTI11Rand(seed uint32) func() uint32 {
	s := seed
	return func() uint32 {
		s ^= s << 13
		s ^= s >> 17
		s ^= s << 5
		return s
	}
}
