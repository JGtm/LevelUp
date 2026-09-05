package filmdec

// i56_energy_test.go — INSTRUMENT DE MESURE de l'ÉTAPE 3 (actualisée) du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : l'ÉTAT ACTIF d'une capacité d'armure.
//
// POURQUOI CET INSTRUMENT PLUTÔT QUE `i57`. Le plan désignait `i57` comme la source de
// l'état actif. Cette piste est RÉFUTÉE : le POC avait exploité son bit 0, qui valait 1 sur
// 386 lectures sur 386 — un interrupteur qui ne bascule jamais n'informe de rien. Depuis,
// l'investigation du 2026-08-13 (i54_research_test.go) a trouvé dans `i54`
// « biped-mobility-action » un ÉVÉNEMENT DATÉ : 67 épisodes discrets d'environ 0,6 s, 1 à 3
// par vie, sur 39 vies sur 99 du film 000d5950. Elle a conclu « non décidable » parce
// qu'AUCUNE IDENTITÉ d'équipement ne voyage avec l'événement.
//
// CE QUE CET INSTRUMENT TESTE, et c'est la reprise n°1 inscrite au REGISTRE_REPORTS :
// l'identité n'a pas besoin de voyager avec l'événement — elle est déjà connue par ailleurs
// (la capacité ÉQUIPÉE se lit à l'image-clé, cf. replay/inventory_decode.go). Ce qui reste à
// prouver, c'est que l'épisode `i54` est bien un USAGE D'ÉQUIPEMENT et non une escalade ou
// une glissade. Le témoin indépendant est `i56` « biped-spartan-ability-energy » : masque
// R(3) puis 7 bits par charge armée (cf. consumeBipedSpartanAbilityEnergy). Une utilisation
// de capacité doit faire CHUTER cette énergie.
//
// LE CONTRÔLE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT LA MESURE. On compte la part des épisodes `i54`
// qui ont une chute d'énergie `i56` du MÊME SLOT dans une fenêtre de ±1 s. Ce taux est
// comparé à un TÉMOIN DÉCALÉ : les mêmes épisodes glissés de +5 s et de -5 s. Si le taux
// réel n'écrase pas le taux décalé, la coïncidence est celle de deux signaux fréquents et
// non une relation — verdict NON, aucun effet affiché.
//
// LECTURE SEULE, gardé par I56_FILM, sauté partout ailleurs (CI comprise). UN SEUL décodage
// filmdec par process (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I56_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI56AbilityEnergyVsI54$' -timeout 20m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i56FilmEnv = "I56_FILM"

// i56Index est l'index d'itérateur du composant biped-spartan-ability-energy dans
// l'archétype biped (cf. components_biped_ability.go, section i56).
const i56Index = 56

// i56Charges est le nombre d'emplacements de charge décrits par le masque R(3). Alias de la
// constante de production depuis le 2026-08-15 : deux littéraux pour la même largeur, c'est
// une divergence en attente.
const i56Charges = AbilityEnergyCharges

// i56CoincidenceWindowUS est la demi-fenêtre de coïncidence épisode <-> chute d'énergie.
// 1 s : un épisode i54 dure ~0,6 s, et l'énergie n'est retransmise qu'aux records qui
// portent i56 — la fenêtre doit couvrir l'épisode entier sans avaler ses voisins.
const i56CoincidenceWindowUS = 1_000_000

// i56ControlShiftUS est le décalage du témoin. 5 s : bien au-delà de la fenêtre, bien en
// deçà d'une vie moyenne.
const i56ControlShiftUS = 5_000_000

// i56Sample est une lecture d'énergie de capacité localisée dans le film.
type i56Sample struct {
	slot uint32
	tsUS uint64
	mask uint32
	ch   [i56Charges]int // valeur 7 bits transmise, -1 si le bit du masque n'est pas armé
}

func TestI56AbilityEnergyVsI54(t *testing.T) {
	dir := os.Getenv(i56FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i56FilmEnv)
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	if slots.Count() == 0 {
		t.Fatalf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
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
	t.Logf("composant %d du registre biped : %q", i56Index, arch.component(i56Index))
	t.Logf("composant %d du registre biped : %q", i54Index, arch.component(i54Index))

	var (
		records   int
		with56    int
		with54    int
		read56    int
		unread56  int
		energy    []i56Sample
		i54On     = map[uint32][]uint64{}
		slotFirst = map[uint32]uint64{}
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
				records++
				if seen, ok := slotFirst[slot]; !ok || pk.TimestampUS < seen {
					slotFirst[slot] = pk.TimestampUS
				}
				has54, has56 := i54InMask(idx), i56InMask(idx)
				if has54 {
					with54++
				}
				if has56 {
					with56++
				}
				if has54 || has56 {
					flag1, s, got := i56WalkRecord(pay, i0, total, idx, lay, arch)
					if flag1 == 1 {
						i54On[slot] = append(i54On[slot], pk.TimestampUS)
					}
					if has56 {
						if got {
							read56++
							s.slot, s.tsUS = slot, pk.TimestampUS
							energy = append(energy, s)
						} else {
							unread56++
						}
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS delta biped %d · masque∋i54 %d · masque∋i56 %d · i56 LU %d · i56 illisible %d",
		records, with54, with56, read56, unread56)
	if len(energy) == 0 {
		t.Log("VERDICT : aucune lecture d'i56 exploitable — la corrélation ne peut pas être calculée")
		return
	}
	i56LogDistribution(t, energy)
	drops := i56Drops(t, energy)
	episodes := i56Episodes(i54On)
	i56Correlate(t, episodes, drops)
}

// i56LogDistribution publie la forme du signal d'énergie : masques observés, valeurs, et
// nombre de slots porteurs. Sans cette forme, un taux de coïncidence ne veut rien dire.
func i56LogDistribution(t *testing.T, energy []i56Sample) {
	maskHist := map[uint32]int{}
	valHist := map[int]int{}
	slotsSeen := map[uint32]bool{}
	for _, e := range energy {
		maskHist[e.mask]++
		slotsSeen[e.slot] = true
		for _, v := range e.ch {
			if v >= 0 {
				valHist[v]++
			}
		}
	}
	masks := make([]uint32, 0, len(maskHist))
	for m := range maskHist {
		masks = append(masks, m)
	}
	sort.Slice(masks, func(a, b int) bool { return masks[a] < masks[b] })
	var parts []string
	for _, m := range masks {
		parts = append(parts, fmt.Sprintf("%d:%d", m, maskHist[m]))
	}
	t.Logf("i56 : %d lectures sur %d slots · masques %v", len(energy), len(slotsSeen), parts)
	vals := make([]int, 0, len(valHist))
	for v := range valHist {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	var vparts []string
	for _, v := range vals {
		vparts = append(vparts, fmt.Sprintf("%d(q%d.%d):%d", v, (v>>4)&0xF, v&0xF, valHist[v]))
	}
	t.Logf("i56 valeurs transmises (valeur(quartet_haut.quartet_bas):n) : %v", vparts)
}

// i56Drop est une chute d'énergie transmise pour un slot et un emplacement de charge.
type i56Drop struct {
	slot   uint32
	tsUS   uint64
	from   int
	to     int
	nibble bool // la chute porte sur le quartet de poids fort (« une utilisation »)
}

// i56Drops extrait les chutes de valeur transmise, par slot et par emplacement.
func i56Drops(t *testing.T, energy []i56Sample) []i56Drop {
	type key struct {
		slot uint32
		ch   int
	}
	series := map[key][]i56Sample{}
	for _, e := range energy {
		for c := 0; c < i56Charges; c++ {
			if e.ch[c] >= 0 {
				series[key{e.slot, c}] = append(series[key{e.slot, c}], e)
			}
		}
	}
	var out []i56Drop
	nibbleDrops := 0
	for k, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			prev, cur := ss[i-1].ch[k.ch], ss[i].ch[k.ch]
			if cur >= prev {
				continue
			}
			nib := (cur>>4)&0xF < (prev>>4)&0xF
			if nib {
				nibbleDrops++
			}
			out = append(out, i56Drop{slot: k.slot, tsUS: ss[i].tsUS, from: prev, to: cur, nibble: nib})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].tsUS < out[b].tsUS })
	t.Logf("CHUTES d'énergie i56 : %d au total, dont %d sur le quartet de poids fort (« utilisation »)",
		len(out), nibbleDrops)
	return out
}

// i56Episode est un épisode i54 (flag1==1) : un début daté pour un slot.
type i56Episode struct {
	slot uint32
	tsUS uint64
}

// i56Episodes regroupe les records flag1==1 en épisodes (> 1 s d'écart = épisode suivant),
// exactement comme l'instrument i54 du 2026-08-13.
func i56Episodes(on map[uint32][]uint64) []i56Episode {
	var out []i56Episode
	for slot, ts := range on {
		sort.Slice(ts, func(a, b int) bool { return ts[a] < ts[b] })
		var last uint64
		for k, x := range ts {
			if k == 0 || x-last > 1_000_000 {
				out = append(out, i56Episode{slot: slot, tsUS: x})
			}
			last = x
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].tsUS < out[b].tsUS })
	return out
}

// i56Correlate confronte les épisodes i54 aux chutes i56, contre un témoin décalé.
func i56Correlate(t *testing.T, eps []i56Episode, drops []i56Drop) {
	if len(eps) == 0 || len(drops) == 0 {
		t.Logf("VERDICT : %d épisodes i54, %d chutes i56 — corrélation non calculable", len(eps), len(drops))
		return
	}
	hit := func(shift int64) (int, int) {
		n := 0
		for _, e := range eps {
			ts := int64(e.tsUS) + shift
			for _, d := range drops {
				if d.slot != e.slot {
					continue
				}
				diff := int64(d.tsUS) - ts
				if diff < 0 {
					diff = -diff
				}
				if diff <= i56CoincidenceWindowUS {
					n++
					break
				}
			}
		}
		return n, len(eps)
	}
	real, tot := hit(0)
	plus, _ := hit(i56ControlShiftUS)
	minus, _ := hit(-i56ControlShiftUS)
	t.Logf("ÉPISODES i54 %d · CHUTES i56 %d", len(eps), len(drops))
	t.Logf("COÏNCIDENCE (±%.1f s) : réel %d/%d (%.1f %%) · témoin +%.0f s %d (%.1f %%) · témoin -%.0f s %d (%.1f %%)",
		float64(i56CoincidenceWindowUS)/1e6, real, tot, 100*float64(real)/float64(tot),
		float64(i56ControlShiftUS)/1e6, plus, 100*float64(plus)/float64(tot),
		float64(i56ControlShiftUS)/1e6, minus, 100*float64(minus)/float64(tot))
	t.Log("RAPPEL du critère : si le taux réel n'écrase pas les deux témoins décalés, " +
		"la coïncidence n'est pas une relation — verdict NON, aucun effet affiché")
}

// i56InMask dit si la liste d'index du masque contient i56.
func i56InMask(idx []int) bool {
	for _, id := range idx {
		if id == i56Index {
			return true
		}
	}
	return false
}

// i56WalkRecord marche les composants du masque avec les désers de PRODUCTION, capture au
// passage le flag1 d'i54 (le gate lu dans le flux) et lit i56 en clair : masque R(3) puis
// 7 bits par bit armé (cf. consumeBipedSpartanAbilityEnergy). Rend got=false dès qu'un
// composant intermédiaire n'est pas porté ou que la marche déborde du payload — la position
// du curseur ne serait plus digne de confiance.
func i56WalkRecord(
	pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype,
) (flag1 int, s i56Sample, got bool) {
	flag1 = -1
	for c := 0; c < i56Charges; c++ {
		s.ch[c] = -1
	}
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return flag1, s, false
		}
		if id == i54Index && at+1 <= total {
			flag1 = int(readBitsAt(pay, at, 1))
		}
		if id == i56Index {
			if at+3 > total {
				return flag1, s, false
			}
			s.mask = readBitsAt(pay, at, 3)
			p := at + 3
			for c := 0; c < i56Charges; c++ {
				if s.mask&(1<<uint(c)) == 0 {
					continue
				}
				if p+7 > total {
					return flag1, s, false
				}
				s.ch[c] = int(readBitsAt(pay, p, 7))
				p += 7
			}
			return flag1, s, true
		}
		name := arch.component(id)
		if name == "" {
			return flag1, s, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return flag1, s, false
		}
		at = br.BitPos()
	}
	return flag1, s, false
}
