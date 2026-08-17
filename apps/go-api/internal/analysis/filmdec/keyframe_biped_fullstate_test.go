package filmdec

// keyframe_biped_fullstate_test.go — INSTRUMENT DU LOT R7-a
// (cf. .ai/V7.5/replay2d/PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md).
//
// LA QUESTION, UNE SEULE : le corps d'un record d'IMAGE-CLE (payload type-2) est-il l'ETAT
// COMPLET de l'objet — la concatenation des deserialiseurs « leaf » de TOUS les composants
// de l'archetype, dans l'ordre du registre (chunk_00), SANS masque ni porte ?
//
// POURQUOI LE BIPEDE (ti=35). C'est l'archetype dont la couverture de portage est la plus
// haute du depot (les composants biped sont portes dans `components_biped_*.go`,
// `components_movement.go`, `components_object*.go`, ...). Si l'hypothese « etat complet »
// est vraie quelque part, c'est la qu'elle se voit ; si elle casse la, elle casse partout.
// L'inventaire (TestKF35Inventory) publie la couverture REELLE, composant par composant,
// au lieu de la supposer.
//
// L'ORACLE. La position du record SUIVANT est connue par une chaine DISJOINTE de toute
// lecture de corps : `WalkKeyframeWorld` (balayage d'en-tetes 64 bits `[id:32][field:26]
// [ti:6]`, 249/250 entites contre un oracle Cheat Engine). Une marche juste atterrit
// BIT-EXACT sur cette frontiere. Le lot R5 a etabli que le balayeur SAUTE les records dont
// `field26 != 0` : le chainage (meme variante, <= 16 intercales) le rattrape.
//
// CE QU'IL NE FAIT PAS : il ne publie AUCUNE donnee, il ne modifie AUCUN fichier partage du
// decodeur, il n'ecrit RIEN sur disque. Il rend des taux et leurs denominateurs.
//
// LECTURE SEULE, garde par KF35_ROOT : saute partout ailleurs (CI comprise). UN SEUL
// decodage filmdec par process (bascules globales) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestKF35' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	kf35RootEnv  = "KF35_ROOT"
	kf35FilmsEnv = "KF35_FILMS"
	// kf35ChainMax borne les records intercales explores avant de declarer une marche
	// perdue (meme borne que `keyframeChainMax`, lot R5).
	kf35ChainMax = 16
	// kf35StubRounds borne la convergence de la variante v4 (une passe par composant non
	// porte decouvert) : l'archetype a au plus `archetypeBlockSlots` composants.
	kf35StubRounds = archetypeBlockSlots
)

// kf35OracleFilms est le CORPUS FERME du lot : les trois films oracles de R3/R5/R6.
var kf35OracleFilms = []string{"000d5950", "00502e52", "07aa428d"}

// kf35Variant est UNE lecture possible du corps, telle que le plan R7-a l'ordonne.
type kf35Variant struct {
	Label string
	Body  KeyframeBodyVariant
	// Stub : sauter (largeur 0) les composants dont le deser n'est pas porte, au lieu de
	// s'arreter dessus — mesure l'ECART RESIDUEL une fois les trous neutralises.
	Stub bool
}

// kf35Variants : les quatre lectures du plan, precedees de leur TEMOIN DE CONTROLE — la
// lecture « record NEW » que R5 a refutee sur `ti=37`/`ti=38`. Sans ce temoin, la longueur
// consommee par l'etat complet ne se compare a rien.
var kf35Variants = []kf35Variant{
	{Label: "v0 TEMOIN record NEW (etat par defaut + porte + masque)",
		Body: KeyframeBodyVariant{DefaultState: true, Gate: true, Mask: true}},
	{Label: "v0b TEMOIN record NEW, composants non portes sautes (0 bit)",
		Body: KeyframeBodyVariant{DefaultState: true, Gate: true, Mask: true}, Stub: true},
	{Label: "v1 64 leaf nus (ni etat par defaut, ni porte, ni masque)",
		Body: KeyframeBodyVariant{}},
	{Label: "v2 etat par defaut + 64 leaf",
		Body: KeyframeBodyVariant{DefaultState: true}},
	{Label: "v3 etat par defaut + porte R(1) + 64 leaf",
		Body: KeyframeBodyVariant{DefaultState: true, Gate: true}},
	{Label: "v4 64 leaf nus, composants non portes sautes (0 bit)",
		Body: KeyframeBodyVariant{}, Stub: true},
}

// kf35Film porte les payloads d'image-cle d'un film et son registre.
type kf35Film struct {
	Name string
	Reg  *Registry
	Pays [][]byte
}

// kf35Films rend le corpus, ou saute le test si la garde d'environnement est absente.
func kf35Films(t *testing.T) []kf35Film {
	t.Helper()
	root := os.Getenv(kf35RootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", kf35RootEnv)
	}
	names := kf35OracleFilms
	if s := os.Getenv(kf35FilmsEnv); s != "" {
		names = strings.Split(s, ",")
	}
	out := make([]kf35Film, 0, len(names))
	for _, n := range names {
		f := kf35Load(t, root+"/"+strings.TrimSpace(n))
		f.Name = strings.TrimSpace(n)
		out = append(out, f)
	}
	return out
}

// kf35Load lit le registre (chunk_00) et TOUS les payloads d'image-cle du film.
func kf35Load(t *testing.T, dir string) kf35Film {
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
	f := kf35Film{Reg: reg}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeKeyframe {
				f.Pays = append(f.Pays, pk.Payload(data))
			}
		}
	}
	if len(f.Pays) == 0 {
		t.Fatalf("aucune image-cle dans %s", dir)
	}
	return f
}

// kf35Tally compte ce qu'une variante a rencontre sur un film. Sans denominateur, un taux
// ne se juge pas : `bounded` est le denominateur publie partout.
type kf35Tally struct {
	bounded   int // records ti=35 bornes par un voisin de l'oracle
	exact     int // marches atterrissant PILE sur la frontiere
	chained   int // marches y retombant apres des intercales (meme variante)
	landedHdr int // marches atterrissant sur un en-tete VALIDE (critere relache)
	desync    int // marches arretees par un composant non porte
	lost      int // marches qui ne retombent jamais sur la frontiere
	gaps      map[int]int
	breaks    map[string]int
	desyncs   map[string]int
	// consumed : longueur EN BITS que la marche a consommee (en-tete compris), pour les
	// marches qui vont au bout. absGaps : valeur absolue des ecarts. Leurs MEDIANES disent,
	// en une grandeur, si la lecture est « presque juste » ou hors sujet — un histogramme
	// d'ecarts tous differents ne le dit pas.
	consumed []int
	absGaps  []int
}

func newKF35Tally() kf35Tally {
	return kf35Tally{gaps: map[int]int{}, breaks: map[string]int{}, desyncs: map[string]int{}}
}

// kf35Median rend la mediane d'un echantillon (0 si vide). Il TRIE la tranche recue.
func kf35Median(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	sort.Ints(xs)
	return xs[len(xs)/2]
}

// rate rend le taux d'atterrissage (direct + chaine) en pourcentage.
func (k kf35Tally) rate() float64 {
	if k.bounded == 0 {
		return 0
	}
	return 100 * float64(k.exact+k.chained) / float64(k.bounded)
}

// kf35Bounded rend les records ti=35 d'un payload avec la frontiere annoncee par l'oracle.
type kf35Bound struct {
	Rec  KeyframeRec
	Want int
}

func kf35BoundedRecs(pay []byte) []kf35Bound {
	recs := WalkKeyframeWorld(pay)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	var out []kf35Bound
	for i, r := range recs {
		if r.TI == bipedDefaultStateTypeIndex && i+1 < len(recs) {
			out = append(out, kf35Bound{Rec: r, Want: recs[i+1].Bit})
		}
	}
	return out
}

// kf35Break nomme le POINT DE DECROCHAGE d'une marche qui n'atterrit pas juste : le
// composant pendant lequel la frontiere reelle a ete franchie (sur-lecture), ou la
// sous-lecture residuelle. C'est ce que le plan demande de publier en histogramme.
func kf35Break(tr EntityTrace, want int) string {
	if tr.EndBit < want {
		return "SOUS-LECTURE (marche finie avant la frontiere)"
	}
	last := "avant le premier composant (en-tete / etat par defaut)"
	for _, c := range tr.Comps {
		if c.StartBit > want {
			break
		}
		last = fmt.Sprintf("i%d %s", c.Index, c.Name)
	}
	return last
}

// kf35Walk rejoue le corps d'UN record sous la variante donnee, puis mesure.
func kf35Walk(f kf35Film, pay []byte, b kf35Bound, v kf35Variant, tal *kf35Tally) {
	tr := WalkKeyframeBody(pay, b.Rec.Bit, f.Reg, v.Body)
	if tr.DesyncAt >= 0 {
		tal.desync++
		if tr.DesyncAt < len(f.Reg.Archetypes[bipedDefaultStateTypeIndex].Components) {
			tal.desyncs[fmt.Sprintf("i%d %s", tr.DesyncAt,
				f.Reg.Archetypes[bipedDefaultStateTypeIndex].Components[tr.DesyncAt])]++
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
		tal.absGaps = append(tal.absGaps, -gap)
	} else {
		tal.absGaps = append(tal.absGaps, gap)
	}
	tal.gaps[gap]++
	tal.breaks[kf35Break(tr, b.Want)]++
	if _, ok := readKeyframeHeader(pay, tr.EndBit, len(pay)*8); ok {
		tal.landedHdr++
	}
	if kf35Chain(f, pay, tr.EndBit, b, v) {
		tal.chained++
		return
	}
	tal.lost++
}

// kf35Chain enchaine la marche SOUS LA MEME VARIANTE jusqu'a la frontiere visee : c'est le
// rattrapage des records que le filtre fort du balayeur (`field26 == 0`) ne voit pas.
func kf35Chain(f kf35Film, pay []byte, from int, b kf35Bound, v kf35Variant) bool {
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
		tr := WalkKeyframeBody(pay, pos, f.Reg, v.Body)
		if tr.DesyncAt >= 0 {
			return false
		}
		pos, prev = tr.EndBit, h.Slot
	}
	return false
}

// kf35Pass mesure UNE variante sur UN film.
func kf35Pass(f kf35Film, v kf35Variant) kf35Tally {
	tal := newKF35Tally()
	for _, pay := range f.Pays {
		for _, b := range kf35BoundedRecs(pay) {
			tal.bounded++
			kf35Walk(f, pay, b, v, &tal)
		}
	}
	return tal
}

// kf35ApplyStubs neutralise, par convergence, TOUS les composants non portes de
// l'archetype bipede : il rejoue la variante, note le composant de desync, lui donne une
// largeur de 0 bit (`SetUnportedStubWidth`, hook de calibration DEJA present dans le
// decodeur), et recommence tant qu'un nouveau trou apparait. Il rend la liste des
// composants neutralises et la fonction de restauration.
func kf35ApplyStubs(f kf35Film, v kf35Variant) (stubbed []string, restore func()) {
	arch, _ := f.Reg.Archetype(bipedDefaultStateTypeIndex)
	seen := map[string]bool{}
	for round := 0; round < kf35StubRounds; round++ {
		added := false
		for _, pay := range f.Pays {
			for _, b := range kf35BoundedRecs(pay) {
				tr := WalkKeyframeBody(pay, b.Rec.Bit, f.Reg, v.Body)
				if tr.DesyncAt < 0 || tr.DesyncAt >= len(arch.Components) {
					continue
				}
				name := arch.Components[tr.DesyncAt]
				if !seen[name] {
					seen[name] = true
					stubbed = append(stubbed, fmt.Sprintf("i%d %s", tr.DesyncAt, name))
					SetUnportedStubWidth(name, 0)
					added = true
				}
			}
		}
		if !added {
			break
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	return stubbed, func() {
		for _, n := range names {
			SetUnportedStubWidth(n, -1)
		}
	}
}

// TestKF35Inventory publie l'ETAT DES LIEUX SUR PIECES : la liste ordonnee des composants de
// l'archetype bipede, leur portage tel que le dispatch de production le declare, et le
// denominateur de records ti=35 par film. C'est lui qui interdit de SUPPOSER « les 64
// composants sont tous portes ».
func TestKF35Inventory(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	arch, ok := films[0].Reg.Archetype(bipedDefaultStateTypeIndex)
	if !ok {
		t.Fatalf("archetype ti=%d absent du registre", bipedDefaultStateTypeIndex)
	}
	t.Logf("== archetype ti=%d : %d composants ==", bipedDefaultStateTypeIndex, len(arch.Components))
	unported := 0
	for i, name := range arch.Components {
		zero := NewBitReader(make([]byte, 512))
		_, _, ported := consumeByName(zero, name, bipedDefaultStateTypeIndex, arch.Level(i))
		mark := "porte"
		if !ported {
			mark, unported = "NON PORTE", unported+1
		}
		t.Logf("  i%-2d %-58s %-9s (%d bits sur flux nul)", i, name, mark, zero.BitPos())
	}
	t.Logf("  -> %d composants non portes sur %d (sonde a flux nul : un deser a branchement"+
		" sur valeur peut etre porte ici et non porte sur donnee reelle)", unported, len(arch.Components))

	for _, f := range films {
		n, lens := 0, map[int]int{}
		var all []int
		for _, pay := range f.Pays {
			for _, b := range kf35BoundedRecs(pay) {
				n++
				lens[b.Want-b.Rec.Bit]++
				all = append(all, b.Want-b.Rec.Bit)
			}
		}
		t.Logf("  film %s : %d tables d'image-cle · %d records ti=%d BORNES · longueur REELLE"+
			" mediane %d bits", f.Name, len(f.Pays), n, bipedDefaultStateTypeIndex,
			kf35Median(all))
		kf35LogTopInt(t, "longueur REELLE du record (want - Bit)", lens, 8)
	}
}

// TestKF35FullState est LA MESURE : les quatre variantes du plan, sous le corruption-check
// du mode film eteint puis allume, sur les trois films du corpus.
func TestKF35FullState(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	for _, corr := range []bool{false, true} {
		prev := filmComponentCorruptionCheck
		SetFilmComponentCorruptionCheck(corr)
		t.Logf("======== corruption-check du mode film = %v ========", corr)
		for _, v := range kf35Variants {
			for _, f := range films {
				kf35Report(t, f, v)
			}
		}
		SetFilmComponentCorruptionCheck(prev)
	}
}

// kf35Report joue une variante sur un film et publie son compte-rendu complet.
func kf35Report(t *testing.T, f kf35Film, v kf35Variant) {
	t.Helper()
	if v.Stub {
		stubbed, restore := kf35ApplyStubs(f, v)
		defer restore()
		t.Logf("  [%s] %s — composants neutralises (0 bit) : %d", f.Name, v.Label, len(stubbed))
		for _, s := range stubbed {
			t.Logf("      %s", s)
		}
	}
	tal := kf35Pass(f, v)
	t.Logf("  [%s] %s", f.Name, v.Label)
	t.Logf("      bornes %4d · exactes %4d · chainees %4d · perdues %4d · desync %4d"+
		" | ATTERRISSAGE %.2f %%", tal.bounded, tal.exact, tal.chained, tal.lost,
		tal.desync, tal.rate())
	t.Logf("      atterrissages sur un en-tete valide : %d", tal.landedHdr)
	t.Logf("      longueur consommee MEDIANE %d bits · ecart absolu MEDIAN %d bits",
		kf35Median(tal.consumed), kf35Median(tal.absGaps))
	kf35LogTopInt(t, "ecart want-EndBit (bits)", tal.gaps, 6)
	kf35LogTopStr(t, "point de decrochage", tal.breaks, 6)
	kf35LogTopStr(t, "composant de desync", tal.desyncs, 6)
}

// kf35LogTopInt publie les `n` valeurs les plus frequentes d'une distribution entiere.
func kf35LogTopInt(t *testing.T, label string, hist map[int]int, n int) {
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
	t.Logf("      %s : %d valeurs distinctes", label, len(hist))
	for _, x := range xs {
		t.Logf("        %8d : %4d fois", x.k, x.n)
	}
}

// kf35LogTopStr publie les `n` entrees les plus frequentes d'une distribution nommee.
func kf35LogTopStr(t *testing.T, label string, hist map[string]int, n int) {
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
	t.Logf("      %s : %d valeurs distinctes", label, len(hist))
	for _, x := range xs {
		t.Logf("        %-62s %4d fois", x.k, x.n)
	}
}
