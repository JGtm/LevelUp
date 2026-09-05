package filmdec

// r8_i54_research_test.go — PISTE B, canal 2 : le composant i54
// `biped-mobility-action-component` du BIPEDE, avec sa CHARGE UTILE.
//
// POURQUOI Y REVENIR. Le lot de 2026-08 a conclu « la mobilite n'a pas d'instant d'usage
// par i54 » en ne lisant que `flag1` — c'est-a-dire la PRESENCE d'une action, sans jamais
// lire CE QU'ELLE EST. Le corps de FUN_1408f02c8 est porte depuis 7ter.60
// (`consumeMobilityActionBody`) mais AUCUN de ses champs n'est publie : le deser les
// consomme pour rester aligne et les jette — exactement le defaut d'i48 corrige en 2026-08.
// Le propulseur du jeu s'appelle `ability_evade` dans les tags : une action de mobilite est
// EXACTEMENT ce qu'il est.
//
// CE QUE CET INSTRUMENT AJOUTE : un MIROIR de `consumeMobilityActionBody` qui appelle les
// MEMES primitives de production (`consumeE494Position`, `consumeObjectForwardAndUp`,
// `consume140c1e9d4`) dans le MEME ordre, mais RETIENT les petits champs terminaux —
// R(10), R(10), R(1), R(7), R(2), R(1). La sequence n'est ecrite qu'ici et le jour ou la
// production corrige une largeur, ce miroir devient faux : il porte donc son propre
// CONTROLE (`r8BodyBits`, la largeur totale consommee, comparee a celle du deser de
// production sur le meme record).
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`. `LockProcessDecode`, CGO_ENABLED=0.

import (
	"sort"
	"testing"
)

// r8I54Index est l'index d'iterateur de `biped-mobility-action-component` dans l'archetype
// biped. Resolu par NOM dans le registre du film, jamais cable — cf. r8I54Resolve.
const r8CompMobilityAction = "biped-mobility-action-component"

// r8MobEvent est UNE emission d'i54 sur un record delta de bipede.
type r8MobEvent struct {
	Slot  uint32
	TSUS  uint64
	Flag1 bool
	Flag2 bool
	// Body dit si le corps a ete lu jusqu'au bout (sinon les champs sont sans valeur).
	Body bool
	// Les six champs terminaux du corps, dans l'ordre du flux.
	W10a, W10b uint32
	B1         uint32
	B7         uint32
	B2         uint32
	BLast      uint32
	// Bits est la largeur totale consommee par le corps — le controle du miroir.
	Bits int
}

// r8MobSetup porte la configuration de lecture d'un film (regle des 5 parametres).
type r8MobSetup struct {
	dir      string
	chunks   []int
	slots    SlotBand
	lay      I0Layout
	arch     Archetype
	i54Index int
}

func r8MobResolve(t *testing.T, dir string) r8MobSetup {
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
		t.Fatalf("aucun slot biped dans les keyframes de %s", dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible : %v", err)
	}
	arch, err := bipedArchetypeDir(dir)
	if err != nil {
		t.Fatalf("archetype biped illisible : %v", err)
	}
	idx := -1
	for id := 0; id < 64; id++ {
		if arch.component(id) == r8CompMobilityAction {
			idx = id
			break
		}
	}
	if idx < 0 {
		t.Fatalf("composant %q absent de l'archetype biped de %s", r8CompMobilityAction, dir)
	}
	return r8MobSetup{dir: dir, chunks: chunks, slots: slots, lay: lay, arch: arch, i54Index: idx}
}

// r8MobOffset marche les composants du masque qui precedent i54 avec les desers de
// PRODUCTION et rend la position de bit ou i54 commence, ou -1.
func (s r8MobSetup) r8MobOffset(pay []byte, i0, total int, idx []int) int {
	at := i0 + s.lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return -1
		}
		if id == s.i54Index {
			return at
		}
		name := s.arch.component(id)
		if name == "" {
			return -1
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), s.arch.Level(id))
		if !ported || br.BitPos() > total {
			return -1
		}
		at = br.BitPos()
	}
	return -1
}

// r8ReadMobility lit i54 a la position `at` : les deux drapeaux, puis — si flag1 — le corps
// via le MIROIR. Rend false si la lecture deborde du payload.
func r8ReadMobility(pay []byte, at, total int) (r8MobEvent, bool) {
	if at+2 > total {
		return r8MobEvent{}, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(at)
	ev := r8MobEvent{Flag1: br.ReadBit(), Flag2: br.ReadBit()}
	if !ev.Flag1 {
		return ev, true
	}
	consume1408f0ac4(br)
	start := br.BitPos()
	r8MirrorBody(br, &ev)
	if br.BitPos() > total {
		return ev, false
	}
	ev.Body, ev.Bits = true, br.BitPos()-start
	return ev, true
}

// r8MirrorBody est le MIROIR de `consumeMobilityActionBody` (components_biped_ability.go) :
// meme sequence, memes primitives de production, mais les six champs terminaux sont RETENUS
// au lieu d'etre jetes. Toute divergence de largeur avec la production se voit sur `Bits`.
func r8MirrorBody(br *BitReader, ev *r8MobEvent) {
	if br.ReadBit() {
		br.ReadBits(10)
	}
	if !br.ReadBit() {
		consumeE494Position(br)
		consumeObjectForwardAndUp(br)
	}
	br.ReadBits(64)
	br.ReadBits(32)
	consumeE494Position(br)
	for i := 0; i < 3; i++ {
		consume140c1e9d4(br, 12)
	}
	br.ReadBits(24)
	br.ReadBits(24)
	consume140c1e9d4(br, 12)
	ev.W10a = uint32(br.ReadBits(10))
	ev.W10b = uint32(br.ReadBits(10))
	ev.B1 = uint32(br.ReadBits(1))
	ev.B7 = uint32(br.ReadBits(7))
	ev.B2 = uint32(br.ReadBits(2))
	ev.BLast = uint32(br.ReadBits(1))
}

// r8ScanMobility balaye le film et rend toutes les emissions d'i54, triees par instant.
// L'appelant doit detenir LockProcessDecode.
func r8ScanMobility(t *testing.T, s r8MobSetup) ([]r8MobEvent, int, int) {
	t.Helper()
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	var out []r8MobEvent
	records, with54 := 0, 0
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
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
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				records++
				if r8HasIndex(idx, s.i54Index) {
					with54++
					if at := s.r8MobOffset(pay, i0, total, idx); at >= 0 {
						if ev, ok := r8ReadMobility(pay, at, total); ok {
							ev.Slot, ev.TSUS = slot, pk.TimestampUS
							out = append(out, ev)
						}
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TSUS < out[j].TSUS })
	return out, records, with54
}

func r8HasIndex(idx []int, want int) bool {
	for _, id := range idx {
		if id == want {
			return true
		}
	}
	return false
}
