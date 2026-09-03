package filmdec

// i56_drops_test.go — INSTRUMENT DE MESURE : UNE CHUTE D'ÉNERGIE DE CAPACITÉ EST-ELLE UN
// ÉVÉNEMENT DATABLE EN ELLE-MÊME ?
//
// CE QUI LE DISTINGUE D'`i56_energy_test.go`. L'instrument du 2026-08-14 mesure la
// COÏNCIDENCE entre les épisodes `i54` et les chutes d'`i56`, sur UN film et 176 lectures —
// son propre journal le qualifie de trop maigre (4,5 % contre témoins 0,0 % et 3,0 %). La
// question posée ici est autre et se répond sans `i54` : combien de chutes, sur combien de
// lectures, sur combien de vies, et de quelle forme ? Un signal qui ne chute jamais ne date
// rien ; un signal qui chute à chaque transmission ne date rien non plus.
//
// CE QUE L'INSTRUMENT LIT, ET PAR QUI. Les valeurs viennent du DÉSERIALISEUR DE PRODUCTION
// (`consumeBipedSpartanAbilityEnergy`, qui les publie depuis le 2026-08-15 par
// `SetAbilityEnergyHook`) et non d'une relecture posée à côté de lui : deux lecteurs du même
// champ divergent le jour où l'un des deux est corrigé.
//
// LES DEUX ENCODAGES, séparés comme le demande RECETTE_LOADOUT §9. Le consommateur
// `FUN_140F8F300` montre que la même valeur 7 bits se lit de deux façons selon la capacité :
// continu `v / 127.0f`, ou discret `(v >> 4) & 0xF` charges entières plus `(v & 0xF)` de
// recharge fractionnaire. On ne sait PAS, depuis le flux, laquelle s'applique à un porteur
// donné — l'instrument classe donc les CHUTES, pas les capacités : chute du quartet de poids
// fort (compatible avec « une charge consommée ») contre chute des seuls bits de poids
// faible (compatible avec une jauge continue, ou avec une recharge fractionnaire).
//
// LE CONTRÔLE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT LA MESURE. La coïncidence chute <-> épisode
// `i54` est rejouée sur la population élargie, avec les MÊMES témoins décalés de ±5 s que
// les mesures précédentes. Si le taux réel n'écrase pas les deux témoins, la coïncidence est
// celle de deux signaux fréquents et non une relation — et le verdict reste NON.
//
// LECTURE SEULE, gardé par I56_DROPS_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I56_DROPS_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI56DropsAreEvents$' -timeout 30m -v

import (
	"os"
	"sort"
	"testing"
)

const i56DropsFilmEnv = "I56_DROPS_FILM"

// i56dSample est une lecture d'i56 localisée, telle que le déserialiseur l'a publiée.
type i56dSample struct {
	slot uint32
	tsUS uint64
	mask uint32
	ch   [AbilityEnergyCharges]int
}

func TestI56DropsAreEvents(t *testing.T) {
	dir := os.Getenv(i56DropsFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i56DropsFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	env := i56dPrepare(t, dir)
	var hook i56dSample
	got := false
	prev := abilityEnergyHook
	SetAbilityEnergyHook(func(mask uint32, ch [AbilityEnergyCharges]int) {
		hook.mask, hook.ch, got = mask, ch, true
	})
	defer SetAbilityEnergyHook(prev)

	energy, epis, st := i56dScan(t, dir, env, &hook, &got)
	t.Logf("RECORDS delta biped %d · masque∋i56 %d (%.3f %%) · i56 LU %d · i56 illisible %d "+
		"· masque∋i54 %d · épisodes i54 %d",
		st.records, st.with56, 100*float64(st.with56)/float64(max(st.records, 1)),
		st.read56, st.unread56, st.with54, len(epis))
	if len(energy) == 0 {
		t.Log("VERDICT : aucune lecture d'i56 — rien à dater sur ce film")
		return
	}
	i56dLogShape(t, energy)
	drops := i56dDeltas(t, energy)
	i56dCorrelate(t, epis, drops)
}

// i56dEnv porte ce que le balayage doit connaître (règle des 5 paramètres).
type i56dEnv struct {
	chunks []int
	slots  SlotBand
	lay    I0Layout
	arch   Archetype
}

type i56dStats struct {
	records, with56, with54, read56, unread56 int
}

func i56dPrepare(t *testing.T, dir string) i56dEnv {
	t.Helper()
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
	t.Logf("i56 = %q · i54 = %q · %d slots de bipède",
		arch.component(i56Index), arch.component(i54Index), slots.Count())
	return i56dEnv{chunks: chunks, slots: slots, lay: lay, arch: arch}
}

// i56dScan parcourt les paquets delta, lit i56 par le déserialiseur de production partout où
// le masque l'annonce, et capture au passage le flag1 d'i54 (le gate lu dans le flux).
func i56dScan(
	t *testing.T, dir string, env i56dEnv, hook *i56dSample, got *bool,
) ([]i56dSample, []i56Episode, i56dStats) {
	var (
		out    []i56dSample
		st     i56dStats
		i54On  = map[uint32][]uint64{}
		minRec = bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + env.lay.TotalBits()
	)
	for _, c := range env.chunks {
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
			for p := 0; p+minRec <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, env.slots, true, env.lay)
				if !ok {
					p++
					continue
				}
				st.records++
				has54, has56 := i54InMask(idx), i56InMask(idx)
				if has54 {
					st.with54++
				}
				if has56 {
					st.with56++
				}
				if has54 || has56 {
					*got = false
					flag1 := i56dWalk(pay, i0, total, idx, env)
					if flag1 == 1 {
						i54On[slot] = append(i54On[slot], pk.TimestampUS)
					}
					if has56 {
						if *got {
							st.read56++
							out = append(out, i56dSample{
								slot: slot, tsUS: pk.TimestampUS,
								mask: hook.mask, ch: hook.ch,
							})
						} else {
							st.unread56++
						}
					}
				}
				p = i0 + env.lay.TotalBits()
			}
		}
	}
	if st.records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	return out, i56Episodes(i54On), st
}

// i56dWalk marche les composants du masque avec les désers de PRODUCTION — c'est cette marche
// qui déclenche le hook d'i56 — et rend le flag1 d'i54 lu au passage. S'arrête dès qu'un
// composant intermédiaire n'est pas porté ou que la marche déborde du payload.
func i56dWalk(pay []byte, i0, total int, idx []int, env i56dEnv) int {
	flag1 := -1
	at := i0 + env.lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return flag1
		}
		if id == i54Index && at+1 <= total {
			flag1 = int(readBitsAt(pay, at, 1))
		}
		name := env.arch.component(id)
		if name == "" {
			return flag1
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), env.arch.Level(id))
		if !ported || br.BitPos() > total {
			return flag1
		}
		at = br.BitPos()
		if id == i56Index {
			return flag1
		}
	}
	return flag1
}

// i56dLogShape publie la forme du signal : masques, valeurs, slots porteurs. Sans cette
// forme, un nombre de chutes ne se juge pas.
func i56dLogShape(t *testing.T, energy []i56dSample) {
	maskHist := map[uint64]int{}
	valHist := map[uint64]int{}
	slotsSeen := map[uint32]bool{}
	for _, e := range energy {
		maskHist[uint64(e.mask)]++
		slotsSeen[e.slot] = true
		for _, v := range e.ch {
			if v != AbilityEnergyUnarmed {
				valHist[uint64(v)]++
			}
		}
	}
	t.Logf("i56 : %d lectures sur %d slots · masques %s",
		len(energy), len(slotsSeen), equipRenderU64(maskHist))
	t.Logf("i56 valeurs transmises : %s", equipRenderU64(valHist))
}

// i56dDeltas extrait les CHUTES et les REMONTÉES par slot et par emplacement de charge, et
// sépare les deux encodages de RECETTE_LOADOUT §9.
func i56dDeltas(t *testing.T, energy []i56dSample) []i56Drop {
	type key struct {
		slot uint32
		ch   int
	}
	series := map[key][]i56dSample{}
	for _, e := range energy {
		for c := 0; c < AbilityEnergyCharges; c++ {
			if e.ch[c] != AbilityEnergyUnarmed {
				series[key{e.slot, c}] = append(series[key{e.slot, c}], e)
			}
		}
	}
	var drops []i56Drop
	rises, pairs, nibbleDrops, lowDrops := 0, 0, 0, 0
	perSlot := map[uint32]int{}
	for k, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			prev, cur := ss[i-1].ch[k.ch], ss[i].ch[k.ch]
			pairs++
			if cur > prev {
				rises++
				continue
			}
			if cur == prev {
				continue
			}
			nib := (cur>>4)&0xF < (prev>>4)&0xF
			if nib {
				nibbleDrops++
			} else {
				lowDrops++
			}
			perSlot[k.slot]++
			drops = append(drops, i56Drop{
				slot: k.slot, tsUS: ss[i].tsUS, from: prev, to: cur, nibble: nib,
			})
		}
	}
	sort.Slice(drops, func(a, b int) bool { return drops[a].tsUS < drops[b].tsUS })
	t.Logf("SÉRIES (slot, emplacement) %d · paires consécutives %d", len(series), pairs)
	t.Logf("CHUTES %d sur %d paires · REMONTÉES %d · quartet HAUT (charges entières) %d "+
		"· quartet BAS seul (jauge continue ou recharge) %d",
		len(drops), pairs, rises, nibbleDrops, lowDrops)
	if len(perSlot) > 0 {
		t.Logf("CHUTES PAR VIE (slot) : %d slots porteurs de chute · %.2f chutes par slot porteur",
			len(perSlot), float64(len(drops))/float64(len(perSlot)))
	}
	return drops
}

// i56dCorrelate rejoue la coïncidence chute <-> épisode i54 avec les témoins décalés de
// ±5 s, sur la population élargie.
func i56dCorrelate(t *testing.T, eps []i56Episode, drops []i56Drop) {
	if len(eps) == 0 || len(drops) == 0 {
		t.Logf("COÏNCIDENCE non calculable : %d épisodes i54, %d chutes i56", len(eps), len(drops))
		return
	}
	hit := func(shift int64) int {
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
		return n
	}
	real, plus, minus := hit(0), hit(i56ControlShiftUS), hit(-i56ControlShiftUS)
	tot := len(eps)
	t.Logf("COÏNCIDENCE (±%.0f s) : réel %d/%d (%.1f %%) · témoin +%.0f s %d (%.1f %%) "+
		"· témoin -%.0f s %d (%.1f %%)",
		float64(i56CoincidenceWindowUS)/1e6, real, tot, 100*float64(real)/float64(tot),
		float64(i56ControlShiftUS)/1e6, plus, 100*float64(plus)/float64(tot),
		float64(i56ControlShiftUS)/1e6, minus, 100*float64(minus)/float64(tot))
	t.Log("RAPPEL du critère : si le taux réel n'écrase pas les deux témoins décalés, " +
		"la coïncidence n'est pas une relation — verdict NON, aucun effet affiché")
}
