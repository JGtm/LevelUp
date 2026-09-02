package filmdec

// i54_research_test.go — INSTRUMENT DE MESURE de l'item 1.4 du plan
// .ai/V7.5/replay2d/PLAN_PARITE_REJEU_2D.md (lot 1, investigation bornée) : les
// évènements du composant i54 « biped-mobility-action » donnent-ils un USAGE DATÉ DE
// CAPACITÉ (équipement actif : surbouclier, camouflage, translocateur — décision
// produit 3 du plan), 100 % offline, sur le film témoin 000d5950 ?
//
// CE QU'IL MESURE, sans rien modifier :
//  1. la PRÉSENCE d'i54 dans le masque des records delta de biped — même détecteur
//     d'en-tête que ScanFilmBipedPositions (production), donc mêmes limites : les
//     records à masque dense (> 7 composants) lui échappent, la mesure est un
//     PLANCHER de la fréquence réelle d'i54 ;
//  2. le FLAG1 d'i54 (le gate du corps, lu DANS le flux — cf. consumeBipedMobilityAction)
//     quand tous les composants qui précèdent i54 dans le record sont portés par
//     consumeByName : flag1==1 = « une action est transmise à cet instant ».
//
// CRITÈRE DE DÉCISION (posé AVANT la mesure) : dans un Slayer 8 joueurs (~10 min), un
// usage d'équipement actif est RARE (ordre 0-10 par joueur et par match). Si les
// épisodes flag1==1 se comptent en dizaines par slot ou par minute, c'est le bruit des
// actions de MOBILITÉ (glissade, escalade, saut — le corps d'i54 porte des positions et
// des vecteurs de déplacement, cf. consumeMobilityActionBody), PAS un usage de
// capacité : verdict NON, ligne au registre RE.
//
// LECTURE SEULE, gardé par I54_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I54_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI54MobilityActionUsage$' -timeout 10m -v

import (
	"os"
	"sort"
	"testing"
)

const i54FilmEnv = "I54_FILM"

// i54Index est l'index d'itérateur du composant biped-mobility-action dans
// l'archétype biped (cf. components_biped_ability.go, section i54).
const i54Index = 54

func TestI54MobilityActionUsage(t *testing.T) {
	dir := os.Getenv(i54FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i54FilmEnv)
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
	if len(slots) == 0 {
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
	t.Logf("composant %d du registre biped : %q", i54Index, arch.component(i54Index))

	type i54Event struct {
		slot  uint32
		tsUS  uint64
		flag1 int // 1 / 0 / -1 = illisible (composant intermédiaire non porté)
	}
	var (
		records    int // records delta biped reconnus (masque explicite <= 7 composants)
		with54     int // dont le masque contient l'index 54
		flag1On    int
		flag1Off   int
		flagNoRead int
		events     []i54Event
		firstTS    = ^uint64(0)
		lastTS     uint64
		// slotFirst date la PREMIÈRE apparition de chaque slot dans les deltas : un slot
		// est UNE VIE, l'offset épisode-début-de-vie discrimine « action de spawn »
		// (offsets tous ~0) d'« action en cours de vie » (offsets répartis).
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
				if pk.TimestampUS < firstTS {
					firstTS = pk.TimestampUS
				}
				if pk.TimestampUS > lastTS {
					lastTS = pk.TimestampUS
				}
				if seen, ok := slotFirst[slot]; !ok || pk.TimestampUS < seen {
					slotFirst[slot] = pk.TimestampUS
				}
				if i54InMask(idx) {
					with54++
					f := i54Flag1(pay, i0, total, idx, lay, arch)
					switch f {
					case 1:
						flag1On++
					case 0:
						flag1Off++
					default:
						flagNoRead++
					}
					events = append(events, i54Event{slot: slot, tsUS: pk.TimestampUS, flag1: f})
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	durMin := float64(lastTS-firstTS) / 60e6
	t.Logf("RECORDS delta biped %d · masque∋i54 %d · flag1==1 %d · flag1==0 %d · flag1 illisible %d",
		records, with54, flag1On, flag1Off, flagNoRead)
	t.Logf("fenêtre du film : %.1f min · slots biped de la bande : %d", durMin, len(slots))

	// ÉPISODES par slot : des records flag1==1 consécutifs du même slot à < 1 s d'écart
	// portent LA MÊME action (état répliqué tant qu'il est actif), pas autant d'usages
	// distincts. C'est le chiffre à confronter au critère de décision.
	bySlot := map[uint32][]uint64{}
	for _, e := range events {
		if e.flag1 == 1 {
			bySlot[e.slot] = append(bySlot[e.slot], e.tsUS)
		}
	}
	slotIDs := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slotIDs = append(slotIDs, s)
	}
	sort.Slice(slotIDs, func(i, j int) bool { return slotIDs[i] < slotIDs[j] })
	episodes := 0
	var startOffsets []float64 // offsets (s) des débuts d'épisode depuis le début de la vie
	for _, s := range slotIDs {
		ts := bySlot[s]
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		slotEpisodes := 0
		var offs []float64
		var last uint64
		for k, x := range ts {
			if k == 0 || x-last > 1_000_000 {
				slotEpisodes++
				offs = append(offs, float64(x-slotFirst[s])/1e6)
			}
			last = x
		}
		episodes += slotEpisodes
		startOffsets = append(startOffsets, offs...)
		rate := 0.0
		if durMin > 0 {
			rate = float64(slotEpisodes) / durMin
		}
		t.Logf("  slot %d : %d records flag1==1 · %d épisodes (> 1 s d'écart) · %.2f épisode/min · débuts à %v s de la vie",
			s, len(ts), slotEpisodes, rate, offs)
	}
	// DISCRIMINANT « action de spawn » : si tous les débuts d'épisode sont à ~0 s de la
	// vie, i54 date l'apparition, pas une action du joueur.
	nearSpawn := 0
	for _, o := range startOffsets {
		if o < 2.0 {
			nearSpawn++
		}
	}
	t.Logf("DISCRIMINANT spawn : %d/%d débuts d'épisode à moins de 2 s du début de la vie",
		nearSpawn, len(startOffsets))
	if durMin > 0 {
		t.Logf("TOTAL : %d épisodes flag1==1 sur %d slots · %.2f épisode/min toutes vies confondues",
			episodes, len(slotIDs), float64(episodes)/durMin)
	}
	t.Logf("RAPPEL du critère : usage d'équipement = ordre 0-10 par joueur et par match ; " +
		"des dizaines d'épisodes par slot ou par minute = bruit de mobilité, verdict NON")
}

// i54InMask dit si la liste d'index du masque contient i54.
func i54InMask(idx []int) bool {
	for _, id := range idx {
		if id == i54Index {
			return true
		}
	}
	return false
}

// i54Flag1 lit flag1 (le gate du corps d'i54, premier bit du composant — cf.
// consumeBipedMobilityAction) en marchant les composants du masque qui précèdent i54
// avec les désers de production (consumeByName), depuis la fin d'i0 (queue i0TailBits
// comprise, même chemin que scanRecordDirs). Rend -1 quand un composant intermédiaire
// n'est pas porté ou que la marche déborde du payload : la position du curseur ne
// serait plus digne de confiance.
func i54Flag1(pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype) int {
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return -1
		}
		if id == i54Index {
			if at+1 > total {
				return -1
			}
			return int(readBitsAt(pay, at, 1))
		}
		name := arch.component(id)
		if name == "" {
			return -1
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return -1
		}
		at = br.BitPos()
	}
	return -1
}
