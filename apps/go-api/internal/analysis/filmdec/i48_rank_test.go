package filmdec

// i48_rank_test.go — INSTRUMENT DE MESURE de l'ÉTAPE 2 du plan
// .ai/V7.5/replay2d/PLAN_EQUIPEMENT_ACTIF.md : « quelle palette pour ce film ? ».
//
// CE QU'IL LIT, ET POURQUOI PERSONNE NE L'AVAIT LU. `consumeBipedDesiredAbilitySet`
// (components_biped_ability.go, i48) reproduit `FUN_1406d0ff0` : R(3) compteur de rotation,
// puis R(1) porte et, SI LA PORTE VAUT 0, **R(6) = l'identité, le rang de palette `sofd`**
// (RECETTE_LOADOUT_2026-07-27 §9 : octet 0xA34 = compteur, octet 0xA35 = identité). Le
// déserialiseur de production consomme ces 6 bits pour rester aligné, et les JETTE. La sonde
// existante (`SetAbilitySetHook`) ne publie que le R(3) et la largeur : l'identité passait
// sous le nez du chantier depuis le début.
//
// CE QUE CELA DÉCIDE. La table de production nomme un champ de 3 BITS lu dans les
// IMAGES-CLÉS (replay/inventory_decode.go), qui rend {3,4,5,6,7}. La palette `sofd` de la
// famille A donne détecteur = 1, mur = 2, grappin = 4, propulseur = 5, répulseur = 6. Les
// deux canaux sont indépendants ; ce que le second dit du même film tranche le premier.
//
// LE CONTRÔLE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT LA MESURE. Sur `000d5950`, le relevé Theater du
// 2026-07-27 donne la composition du match : 3 grappins, 3 propulseurs, 1 capteur de menace,
// 1 mur. Le champ d'image-clé la reproduit en 4:44, 5:44, 6:20, 3:24 — deux valeurs à forte
// occurrence pour les triplets, deux à faible occurrence pour les isolés. Deux prédictions
// s'excluent :
//
//	P1 — le R(6) est le rang famille A : les rangs lus sont {4, 5} en tête et {1, 2} en
//	     queue. Alors le champ d'image-clé N'EST PAS le rang, la table de traduction
//	     index -> rang se lit directement, et les index 3 et 6 se renomment SUR MESURE.
//	P2 — le R(6) reproduit l'index d'image-clé : les rangs lus sont {3, 4, 5, 6}. Alors les
//	     deux canaux disent la même chose, et la palette `sofd` ne s'applique pas à ce titre.
//
// Une troisième issue reste possible et sera dite telle quelle : la porte vaut 1 partout
// (aucune identité transmise dans les deltas), ou la marche n'atteint jamais i48.
//
// LECTURE SEULE, gardé par I48_FILM, sauté partout ailleurs (CI comprise). UN SEUL décodage
// filmdec par process (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I48_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI48PaletteRank$' -timeout 20m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i48FilmEnv = "I48_FILM"

// i48Index est l'index d'itérateur du composant biped-desired-ability-set-component dans
// l'archétype biped (cf. components_biped_ability.go, section i48).
const i48Index = 48

// i48CounterBits / i48RankBits : la grammaire de FUN_1406d0ff0, dans l'ordre du flux.
// La porte est INVERSÉE (consumeGate0R) : le rang n'est présent que si son bit vaut 0.
const (
	i48CounterBits = 3
	i48RankBits    = 6
)

// i48Sample est une lecture d'i48 localisée dans le film. `rank` vaut -1 quand la porte
// vaut 1 : le film ne transmet alors PAS d'identité, et c'est une valeur, pas un trou.
type i48Sample struct {
	slot    uint32
	tsUS    uint64
	counter uint32
	rank    int
}

func TestI48PaletteRank(t *testing.T) {
	dir := os.Getenv(i48FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i48FilmEnv)
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBand(dir, chunks)
	if len(slots) == 0 {
		t.Fatalf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	arch := i48Archetype(t, dir)
	t.Logf("composant %d du registre biped : %q", i48Index, arch.component(i48Index))

	samples, st := i48Scan(dir, chunks, slots, lay, arch)
	if st.records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS delta biped %d · masque∋i48 %d (%.2f %%) · i48 LU %d · i48 illisible %d",
		st.records, st.withI48, 100*float64(st.withI48)/float64(st.records), st.read, st.unread)
	if len(samples) == 0 {
		t.Log("VERDICT : aucune lecture d'i48 exploitable — la marche n'atteint pas le composant")
		return
	}
	i48LogRanks(t, samples)
	i48LogPerSlot(t, samples)
}

// i48Stats compte ce que la marche rencontre. Sans ces dénominateurs, un histogramme de
// rangs ne se juge pas.
type i48Stats struct {
	records, withI48, read, unread int
}

// i48Archetype charge l'archétype biped du registre du film (chunk_00).
func i48Archetype(t *testing.T, dir string) Archetype {
	t.Helper()
	reg0, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 (registre) illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(reg0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	arch, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		t.Fatalf("archétype biped %d absent du registre", BipedTypeIndex)
	}
	return arch
}

// i48Scan parcourt les paquets delta et lit i48 partout où le masque l'annonce.
func i48Scan(
	dir string, chunks []int, slots map[uint32]bool, lay I0Layout, arch Archetype,
) ([]i48Sample, i48Stats) {
	var (
		out []i48Sample
		st  i48Stats
	)
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				st.records++
				if i48InMask(idx) {
					st.withI48++
					if s, got := i48WalkRecord(pay, i0, total, idx, lay, arch); got {
						st.read++
						s.slot, s.tsUS = slot, pk.TimestampUS
						out = append(out, s)
					} else {
						st.unread++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return out, st
}

// i48InMask dit si le masque du record annonce le composant i48.
func i48InMask(idx []int) bool {
	for _, id := range idx {
		if id == i48Index {
			return true
		}
	}
	return false
}

// i48WalkRecord marche les composants du masque avec les désers de PRODUCTION et lit i48 en
// clair : R(3) compteur, R(1) porte, puis R(6) rang si la porte vaut 0. Rend got=false dès
// qu'un composant intermédiaire n'est pas porté ou que la marche déborde du payload — la
// position du curseur ne serait plus digne de confiance.
func i48WalkRecord(
	pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype,
) (s i48Sample, got bool) {
	s.rank = -1
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return s, false
		}
		if id == i48Index {
			if at+i48CounterBits+1 > total {
				return s, false
			}
			s.counter = readBitsAt(pay, at, i48CounterBits)
			if readBitsAt(pay, at+i48CounterBits, 1) == 0 { // porte INVERSÉE
				if at+i48CounterBits+1+i48RankBits > total {
					return s, false
				}
				s.rank = int(readBitsAt(pay, at+i48CounterBits+1, i48RankBits))
			}
			return s, true
		}
		name := arch.component(id)
		if name == "" {
			return s, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return s, false
		}
		at = br.BitPos()
	}
	return s, false
}

// i48LogRanks publie la distribution des rangs transmis et celle du compteur de rotation.
// Le compteur est publié parce qu'il BORNE l'interprétation : la recette le décrit à valeurs
// -1..6, et un compteur hors de cette plage dirait que la lecture est mal placée.
func i48LogRanks(t *testing.T, samples []i48Sample) {
	rankHist := map[int]int{}
	cntHist := map[uint32]int{}
	gated := 0
	for _, s := range samples {
		rankHist[s.rank]++
		cntHist[s.counter]++
		if s.rank < 0 {
			gated++
		}
	}
	t.Logf("i48 : %d lectures · %d SANS identité (porte à 1, %.1f %%)",
		len(samples), gated, 100*float64(gated)/float64(len(samples)))
	t.Logf("i48 rangs transmis : %s", i48RenderInt(rankHist))
	t.Logf("i48 compteur de rotation R(3) : %s", i48RenderU32(cntHist))
}

// i48LogPerSlot publie, par slot, le rang MAJORITAIRE et son histogramme. C'est la forme
// qui se confronte au relevé Theater — lequel nomme des slots, pas des films.
func i48LogPerSlot(t *testing.T, samples []i48Sample) {
	perSlot := map[uint32]map[int]int{}
	for _, s := range samples {
		if s.rank < 0 {
			continue
		}
		if perSlot[s.slot] == nil {
			perSlot[s.slot] = map[int]int{}
		}
		perSlot[s.slot][s.rank]++
	}
	if len(perSlot) == 0 {
		t.Log("i48 par slot : aucune identité transmise, rien à confronter")
		return
	}
	slots := make([]uint32, 0, len(perSlot))
	for sl := range perSlot {
		slots = append(slots, sl)
	}
	sort.Slice(slots, func(a, b int) bool { return slots[a] < slots[b] })
	t.Log("== i48 PAR SLOT — rang majoritaire, puis histogramme complet ==")
	for _, sl := range slots {
		best, bestN, tot := -1, 0, 0
		for v, n := range perSlot[sl] {
			tot += n
			if n > bestN {
				best, bestN = v, n
			}
		}
		t.Logf("  slot %d : rang %d (%d/%d) · %s", sl, best, bestN, tot, i48RenderInt(perSlot[sl]))
	}
}

func i48RenderInt(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := ""
	for _, k := range keys {
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("%d:%d", k, h[k])
	}
	return out
}

func i48RenderU32(h map[uint32]int) string {
	conv := make(map[int]int, len(h))
	for k, v := range h {
		conv[int(k)] = v
	}
	return i48RenderInt(conv)
}
