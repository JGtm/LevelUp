package filmdec

// lot1_monde_chrono_research_test.go — LOT 1 : la resolution ref -> slot -> bipede de
// damage_aftermath (0xC0) se fait aujourd'hui contre l'etat du monde en FIN de chunk (tous
// les tick-frames appliques puis snapshot ; taux mesure 82-89 %). Un bipede mort en cours
// de chunk (DEL avant la fin) est ALORS deja delie : la ref d'un evenement anterieur a sa
// mort ne resout plus. Cet instrument reconstruit l'etat du monde CHRONOLOGIQUE — les seuls
// tick-frames ANTERIEURS a l'evenement — et mesure si le taux de resolution MONTE.
//
// METHODE (memes refs, meme jeu de bases que victime_slot ; seule change l'instant du monde) :
//   - APRES (chronologique) : passe unique en ordre de paquets ; le monde avance tick-frame
//     par tick-frame ; a chaque 0xC0 on resout contre le monde A CET INSTANT.
//   - AVANT (fin de chunk) : le monde de la meme passe une fois TOUS les tick-frames
//     appliques (== l'etat que l'instrument de production utilise) ; on re-resout les memes
//     evenements contre lui. C'est le temoin de non-regression (doit reproduire ~82 %).
// La base retenue par ref est celle qui MAXIMISE le taux AVANT (le choix de production) ;
// on publie AVANT et APRES a CETTE base, par film.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"testing"
)

// lot1chBases : jeu de bases candidat (identique a victime_slot pour comparabilite).
var lot1chBases = []int{0, 128, 256, 384, 448, 480, 500, 508, 510, 512, 514, 516, 520, 544, 576}

// lot1chReferenceBase : base de la bande bipede etablie par l'instrument A (calibration par
// la vitalite, base a couverture max = 512 sur les trois films temoins). Sert de reference
// commune pour le AVANT/APRES quand l'argmax "monde" tombe sur une base voisine (bande contigue).
const lot1chReferenceBase = 512

// lot1DamageRefs rend les index bruts des deux references domaine-1 d'un payload 0xC0, et
// ok=false si le paquet n'est pas un damage_aftermath (type 0).
func lot1DamageRefs(pay []byte) (idx0 int, has0 bool, idx1 int, has1 bool, ok bool) {
	idx0, idx1 = -1, -1
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 0 {
		return
	}
	ok = true
	if i0, o0 := lot1RefDom1(br); o0 {
		idx0, has0 = int(i0), true
	}
	if i1, o1 := lot1RefDom1(br); o1 {
		idx1, has1 = int(i1), true
	}
	return
}

// lot1chEvt : un evenement 0xC0 dont on garde les index pour la re-resolution fin-de-chunk.
type lot1chEvt struct {
	idx0, idx1 int
}

// lot1chAccum : atterrissages bipedes par base, pour une reference.
type lot1chAccum struct {
	present int
	avant   map[int]int
	apres   map[int]int
}

func newLot1chAccum() *lot1chAccum {
	return &lot1chAccum{avant: map[int]int{}, apres: map[int]int{}}
}

func lot1chIsBiped(w *World, base, idx int) bool {
	if idx < 0 {
		return false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}

func TestLot1MondeChrono(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	cfg := DefaultFrameConfig()
	acc0, acc1 := newLot1chAccum(), newLot1chAccum()

	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		var events []lot1chEvt
		// PASSE CHRONOLOGIQUE : monde avance en ordre de paquets ; resolution APRES a l'instant.
		for _, pk := range pks {
			pay := pk.Payload(data)
			switch {
			case pk.Type == PacketTypeKeyframe:
				for _, r := range WalkKeyframeWorld(pay) {
					w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
				}
			case pk.Type == PacketTypeDelta && pk.Size >= 1 && pay[0]&0x40 == 0:
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, w, cfg) // avance le monde (tick-frame)
			case pk.Type == PacketTypeDelta && pk.Size >= 2 && pay[0] == 0xC0:
				idx0, has0, idx1, has1, ok := lot1DamageRefs(pay)
				if !ok {
					continue
				}
				events = append(events, lot1chEvt{idx0: idx0, idx1: idx1})
				if has0 {
					acc0.present++
					for _, b := range lot1chBases {
						if lot1chIsBiped(w, b, idx0) {
							acc0.apres[b]++
						}
					}
				}
				if has1 {
					acc1.present++
					for _, b := range lot1chBases {
						if lot1chIsBiped(w, b, idx1) {
							acc1.apres[b]++
						}
					}
				}
			}
		}
		// FIN DE CHUNK : w porte desormais TOUS les tick-frames — re-resoudre les memes events.
		for _, e := range events {
			if e.idx0 >= 0 {
				for _, b := range lot1chBases {
					if lot1chIsBiped(w, b, e.idx0) {
						acc0.avant[b]++
					}
				}
			}
			if e.idx1 >= 0 {
				for _, b := range lot1chBases {
					if lot1chIsBiped(w, b, e.idx1) {
						acc1.avant[b]++
					}
				}
			}
		}
	}

	line := func(label string, a *lot1chAccum, b int) {
		av, ap := a.avant[b], a.apres[b]
		t.Logf("    %s base=%d : AVANT %d/%d = %.1f %% · APRES %d/%d = %.1f %% · GAIN %+.1f pts",
			label, b, av, a.present, lot1Pct(av, a.present), ap, a.present, lot1Pct(ap, a.present),
			lot1Pct(ap, a.present)-lot1Pct(av, a.present))
	}
	report := func(name string, a *lot1chAccum) {
		best, bestHits := lot1chBases[0], -1
		for _, b := range lot1chBases {
			if a.avant[b] > bestHits {
				best, bestHits = b, a.avant[b]
			}
		}
		t.Logf("%s (%d references presentes)", name, a.present)
		line("base 'meilleur AVANT'", a, best)
		// Base 512 = bande bipede etablie par l'instrument A (calibration vitalite, 3 films).
		if best != lot1chReferenceBase {
			line("base 512 (calibree A)", a, lot1chReferenceBase)
		}
	}
	t.Logf("== resolution ref -> bipede : monde fin-de-chunk vs chronologique (%d chunks) ==", n)
	report("REF0", acc0)
	report("REF1", acc1)
}
