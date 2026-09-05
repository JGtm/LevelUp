package filmdec

// i48_manques_research_test.go — INSTRUMENT DE MESURE (pas de production). Lot R2 du
// PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.
//
// LA QUESTION. Le canal i48 (biped-desired-ability-set-component) manque ~5 % de ses
// emissions, auto-mesure par les sauts du compteur R(3) (equipment_changes.go). Ces
// manques sont-ils des pertes du FILM (les octets n'existent pas — incompressible) ou des
// pertes du SCANNER (les octets existent mais le balayage strict les rejette — corrigeable) ?
//
// LA METHODE. Sur une fenetre bornee [US_MIN, US_MAX], deux balayages du MEME flux :
//
//  1. STRICT — replique bit a bit du balayage de production (walkAbilityEmissions) :
//     matchBipedHeader (tag=1, i0 absolu, region), masque a comptage 2..7 commencant a 0,
//     marche des composants jusqu'a i48, saut post-match a i0+TotalBits. En PLUS de la
//     production, il note les records i48 dont la marche echoue (Unread) et le composant
//     fautif.
//  2. RELACHE — le meme flux, position de bit par position de bit, SANS saut post-match,
//     chaque garde stricte relachee et ETIQUETEE sur le candidat : tag!=1, bit16!=0,
//     masque dense R(64) (bit de porte a 1), comptage=1, masque sans i0, i0 non absolu
//     (pregate!=0), region etrangere. Les candidats dont l'i0 est absolu et de la bonne
//     region sont MARCHES avec les desers de production jusqu'a i48 (compteur + rang lus).
//
// LE VERDICT est la confrontation : le compteur R(3) de l'emission manquante est PREDIT
// (compteur precedent + 1 modulo 8) AVANT le balayage relache. Un candidat marche, au bon
// slot, dans la fenetre, portant le compteur predit = les octets existent (perte SCANNER).
// Aucun candidat plausible sur toute la fenetre = perte FILM, avec denominateurs (paquets
// et bits balayes, candidats rejetes et pourquoi).
//
// GARDE : I48M_FILM (patron TRANSLOC_FILM — sans elle le test se saute, les films ne sont
// pas versionnes). Fenetre OBLIGATOIRE (balayage borne). UN decodage filmdec par process.
//
//	CGO_ENABLED=0 I48M_FILM=<depot>/data/cache/film_chunks/1b2d9e08 \
//	  I48M_SLOT=535 I48M_US_MIN=146862000 I48M_US_MAX=194162000 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI48ManquesFenetre$' -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"
)

const (
	i48mFilmEnv = "I48M_FILM"
	i48mSlotEnv = "I48M_SLOT"
	i48mMinEnv  = "I48M_US_MIN"
	i48mMaxEnv  = "I48M_US_MAX"
)

// i48mSetup porte la configuration resolue une fois pour un film.
type i48mSetup struct {
	dir    string
	chunks []int
	slots  SlotBand
	lay    I0Layout
	arch   Archetype
	idx48  int
	minRec int
}

// i48mResolve resout la configuration du film ; l'index d'i48 vient du NOM de registre.
func i48mResolve(t *testing.T, dir string) i48mSetup {
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
	return i48mSetup{
		dir: dir, chunks: chunks, slots: slots, lay: lay, arch: arch,
		idx48:  eqAbilityIndex(t, arch),
		minRec: bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits(),
	}
}

// i48mCand est un record candidat — strict ou relache — rattache a ses octets.
type i48mCand struct {
	Slot       uint32
	Chunk, Pkt int
	TS         uint64
	Off        int // offset BIT du debut du record dans le payload du paquet
	Variant    string
	Guards     string // gardes strictes violees ; vide = conforme production
	Idx        []int
	Walked     bool
	StopID     int // dernier composant consomme quand la marche n'atteint pas i48
	Counter    uint32
	Rank       int
}

// i48mKey identifie un record (pour dedupe strict/relache).
func (c i48mCand) i48mKey() string {
	return fmt.Sprintf("%d/%d/%d", c.Chunk, c.Pkt, c.Off)
}

// i48mPackets itere les paquets delta de la fenetre ; rend paquets et octets visites.
func i48mPackets(s i48mSetup, usMin, usMax uint64, f func(c int, pk FilmPacket, pay []byte)) (int, int) {
	pkts, bytes := 0, 0
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.TimestampUS < usMin || pk.TimestampUS > usMax {
				continue
			}
			pkts++
			bytes += pk.Size
			f(c, pk, pk.Payload(data))
		}
	}
	return pkts, bytes
}

// i48mHook installe la sonde i48 et rend (capture, restauration).
func i48mHook() (*struct {
	counter uint32
	rank    int
	got     bool
}, func()) {
	last := &struct {
		counter uint32
		rank    int
		got     bool
	}{}
	prev := abilitySetHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		last.counter, last.rank, last.got = uint32(counter), rank, true
	})
	return last, func() { SetAbilitySetHook(prev) }
}

// i48mStrict replique le balayage de PRODUCTION sur la fenetre : memes gardes, meme saut
// post-match. Rend les emissions decodees ET les records i48 dont la marche echoue.
func i48mStrict(s i48mSetup, usMin, usMax uint64) (ems, unread []i48mCand) {
	last, restore := i48mHook()
	defer restore()
	i48mPackets(s, usMin, usMax, func(c int, pk FilmPacket, pay []byte) {
		total := len(pay) * 8
		for p := 0; p+s.minRec <= total; {
			i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
			if !ok {
				p++
				continue
			}
			if maskHas(idx, s.idx48) {
				cand := i48mCand{
					Slot: slot, Chunk: c, Pkt: pk.Index, TS: pk.TimestampUS,
					Off: p, Variant: "count", Idx: idx, StopID: -1,
				}
				last.got = false
				stop := -1
				walkRecordComponents(pay, i0, total, idx, s.lay, s.arch, func(id int) bool {
					stop = id
					return id != s.idx48
				})
				if stop == s.idx48 && last.got {
					cand.Walked, cand.Counter, cand.Rank = true, last.counter, last.rank
					ems = append(ems, cand)
				} else {
					cand.StopID = stop
					unread = append(unread, cand)
				}
			}
			p = i0 + s.lay.TotalBits()
		}
	})
	return ems, unread
}

// i48mDenseIdx lit un masque dense R(64) a la position bit at, dans l'ordre du flux
// (bit k = composant k) ou inverse (bit k = composant 63-k), et rend les index leves.
func i48mDenseIdx(pay []byte, at int, msb bool) []int {
	var idx []int
	for k := 0; k < 64; k++ {
		if readBitsAt(pay, at+k, 1) == 1 {
			comp := k
			if msb {
				comp = 63 - k
			}
			idx = append(idx, comp)
		}
	}
	sort.Ints(idx)
	return idx
}

// i48mMatchAt tente la grammaire GENERALISEE a la position p : variantes comptage (1..7)
// et masque dense R(64) (deux ordres de bits). Rend (candidat, position i0, ok). Gardes
// conservees (le bruit noierait tout sans elles) : prefixe=1, slot dans la bande, indices
// STRICTEMENT croissants, masque contenant i48.
func i48mMatchAt(s i48mSetup, pay []byte, p, total int) (i48mCand, int, bool) {
	if readBitsAt(pay, p, 1) != 1 {
		return i48mCand{}, 0, false
	}
	slot := readBitsAt(pay, p+1, bipedSlotBits)
	if !s.slots.Has(slot) {
		return i48mCand{}, 0, false
	}
	tag := readBitsAt(pay, p+14, 2)
	bit16 := readBitsAt(pay, p+16, 1)
	gate := readBitsAt(pay, p+17, 1)
	cand := i48mCand{Slot: slot, Off: p, StopID: -1, Rank: AbilitySetNoRank}
	guards := ""
	if tag != 1 {
		guards += fmt.Sprintf("tag=%d ", tag)
	}
	if bit16 != 0 {
		guards += "bit16=1 "
	}
	var i0 int
	if gate == 0 {
		mc := int(readBitsAt(pay, p+18, 3))
		if mc < 1 || mc > bipedMaxMaskCnt {
			return i48mCand{}, 0, false
		}
		if p+bipedHeaderBits+bipedIndexBits*mc > total {
			return i48mCand{}, 0, false
		}
		idx, ok := i48mAscending(pay, p+bipedHeaderBits, mc)
		if !ok || !maskHas(idx, s.idx48) {
			return i48mCand{}, 0, false
		}
		if mc < bipedMinMaskCnt {
			guards += "mc=1 "
		}
		if idx[0] != 0 {
			guards += "noI0 "
		}
		cand.Variant, cand.Idx = "count", idx
		i0 = p + bipedHeaderBits + bipedIndexBits*mc
	} else {
		// Variante dense : [1][14 id][2 tag][1 porte=1][R(64) masque] puis i0.
		if p+18+64 > total {
			return i48mCand{}, 0, false
		}
		ok := false
		for _, msb := range []bool{false, true} {
			idx := i48mDenseIdx(pay, p+18, msb)
			if len(idx) < 2 || len(idx) > 40 || idx[0] != 0 || !maskHas(idx, s.idx48) {
				continue
			}
			cand.Variant, cand.Idx, ok = fmt.Sprintf("dense/msb=%v", msb), idx, true
			break
		}
		if !ok {
			return i48mCand{}, 0, false
		}
		guards += "dense "
		i0 = p + 18 + 64
	}
	if i0+s.lay.TotalBits() > total {
		return i48mCand{}, 0, false
	}
	const preGate = i0SpineBits + i0UseDefaultBits
	if cand.Idx[0] == 0 {
		if v := readBitsAt(pay, i0, preGate); v != 0 {
			guards += fmt.Sprintf("pregate=%d ", v)
		}
		if v := readBitsAt(pay, i0+preGate, s.lay.GateBits-preGate); v != s.lay.Region {
			guards += fmt.Sprintf("region=%d ", v)
		}
	}
	cand.Guards = guards
	return cand, i0, true
}

// i48mAscending est ascendingFromZero SANS l'exigence du premier index a 0 (garde relachee
// « noI0 ») ; la croissance STRICTE, elle, est conservee — c'est l'ancre anti-bruit.
func i48mAscending(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := 0; k < count; k++ {
		idx := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if idx <= prev {
			return nil, false
		}
		prev = idx
		out = append(out, idx)
	}
	return out, true
}

// i48mWalk marche un candidat jusqu'a i48 avec les desers de PRODUCTION. La marche n'est
// tentee que si son point de depart est sur : i0 absolu de la bonne region (sinon la
// largeur d'i0 n'est pas connue), ou masque sans i0 (depart juste apres les indices).
func i48mWalk(s i48mSetup, pay []byte, cand *i48mCand, i0, total int) {
	last, restore := i48mHook()
	defer restore()
	if cand.Idx[0] == 0 {
		for _, g := range []string{"pregate", "region"} {
			if containsGuard(cand.Guards, g) {
				return // largeur d'i0 inconnue : marche non tentee, candidat garde tel quel
			}
		}
		stop := -1
		walkRecordComponents(pay, i0, total, cand.Idx, s.lay, s.arch, func(id int) bool {
			stop = id
			return id != s.idx48
		})
		if stop == s.idx48 && last.got {
			cand.Walked, cand.Counter, cand.Rank = true, last.counter, last.rank
		} else {
			cand.StopID = stop
		}
		return
	}
	// Masque sans i0 : les composants commencent immediatement apres les indices.
	br := NewBitReader(pay)
	br.SetBitPos(cand.Off + bipedHeaderBits + bipedIndexBits*len(cand.Idx))
	for _, id := range cand.Idx {
		name := s.arch.component(id)
		if name == "" {
			return
		}
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), s.arch.Level(id))
		if !ported || br.BitPos() > total {
			return
		}
		cand.StopID = id
		if id == s.idx48 {
			if last.got {
				cand.Walked, cand.Counter, cand.Rank = true, last.counter, last.rank
				cand.StopID = -1
			}
			return
		}
	}
}

// containsGuard dit si la liste de gardes violees contient l'etiquette donnee.
func containsGuard(guards, name string) bool {
	for i := 0; i+len(name) <= len(guards); i++ {
		if guards[i:i+len(name)] == name {
			return true
		}
	}
	return false
}

// i48mRelaxed balaye la fenetre position de bit par position de bit, SANS saut post-match,
// et rend tous les candidats i48 generalises. Rend aussi (paquets, octets) balayes.
func i48mRelaxed(s i48mSetup, usMin, usMax uint64) ([]i48mCand, int, int) {
	var out []i48mCand
	pkts, bytes := i48mPackets(s, usMin, usMax, func(c int, pk FilmPacket, pay []byte) {
		total := len(pay) * 8
		for p := 0; p+bipedHeaderBits+bipedIndexBits <= total; p++ {
			cand, i0, ok := i48mMatchAt(s, pay, p, total)
			if !ok {
				continue
			}
			cand.Chunk, cand.Pkt, cand.TS = c, pk.Index, pk.TimestampUS
			i48mWalk(s, pay, &cand, i0, total)
			out = append(out, cand)
		}
	})
	return out, pkts, bytes
}

// i48mEnvUint lit une variable d'environnement entiere obligatoire.
func i48mEnvUint(t *testing.T, name string) uint64 {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s obligatoire : le balayage relache doit etre BORNE (fenetre en us)", name)
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		t.Fatalf("%s invalide : %v", name, err)
	}
	return n
}

// TestI48ManquesFenetre — le cas index : une fenetre, un slot, UNE emission manquante
// annoncee par le compteur. Verdict : SCANNER (candidat aux octets retrouves) ou FILM.
func TestI48ManquesFenetre(t *testing.T) {
	dir := os.Getenv(i48mFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", i48mFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	usMin, usMax := i48mEnvUint(t, i48mMinEnv), i48mEnvUint(t, i48mMaxEnv)
	slotFilter := uint64(0)
	if v := os.Getenv(i48mSlotEnv); v != "" {
		slotFilter, _ = strconv.ParseUint(v, 10, 32)
	}
	s := i48mResolve(t, dir)
	t.Logf("FILM %s · fenetre US [%d, %d] · slot cible %d · i48=index %d", dir, usMin, usMax, slotFilter, s.idx48)

	ems, unread := i48mStrict(s, usMin, usMax)
	strictSeen := map[string]bool{}
	for _, e := range ems {
		strictSeen[e.i48mKey()] = true
		if slotFilter == 0 || e.Slot == uint32(slotFilter) {
			t.Logf("STRICT  slot %d  c%d rang %s  @%dus (chunk %d pkt %d off %d)",
				e.Slot, e.Counter, eqRankLabel(e.Rank), e.TS, e.Chunk, e.Pkt, e.Off)
		}
	}
	for _, u := range unread {
		t.Logf("STRICT-UNREAD  slot %d  @%dus (chunk %d pkt %d off %d) marche arretee apres i%d",
			u.Slot, u.TS, u.Chunk, u.Pkt, u.Off, u.StopID)
	}
	t.Logf("STRICT : %d emissions decodees, %d records i48 illisibles (Unread)", len(ems), len(unread))
	i48mPredictionLog(t, ems, uint32(slotFilter))

	cands, pkts, bytes := i48mRelaxed(s, usMin, usMax)
	t.Logf("RELACHE : %d paquets delta, %d octets balayes bit a bit, %d candidats i48", pkts, bytes, len(cands))
	i48mCandLog(t, cands, strictSeen, uint32(slotFilter))
}

// i48mPredictionLog enonce la prediction AVANT le verdict : les compteurs des emissions
// manquantes entre chaque paire d'emissions strictes du slot dont le pas n'est pas 1.
func i48mPredictionLog(t *testing.T, ems []i48mCand, slot uint32) {
	t.Helper()
	bySlot := map[uint32][]i48mCand{}
	for _, e := range ems {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	for sl, list := range bySlot {
		if slot != 0 && sl != slot {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].TS < list[j].TS })
		for i := 1; i < len(list); i++ {
			step := (int(list[i].Counter) - int(list[i-1].Counter) + 8) % 8
			if step == 1 || step == 0 {
				continue
			}
			var want []uint32
			for k := 1; k < step; k++ {
				want = append(want, uint32((int(list[i-1].Counter)+k)%8))
			}
			t.Logf("PREDICTION  slot %d : %d emission(s) manquante(s) entre @%dus (c%d) et @%dus (c%d) — compteur(s) attendu(s) %v",
				sl, step-1, list[i-1].TS, list[i-1].Counter, list[i].TS, list[i].Counter, want)
		}
	}
}

// i48mCandLog ventile les candidats relaches : profils de gardes (denominateurs), detail
// des candidats du slot cible non vus par le strict.
func i48mCandLog(t *testing.T, cands []i48mCand, strictSeen map[string]bool, slot uint32) {
	t.Helper()
	profiles := map[string]int{}
	news := 0
	for _, c := range cands {
		key := c.Guards
		if key == "" {
			key = "(conforme production)"
		}
		profiles[key]++
		if strictSeen[c.i48mKey()] {
			continue
		}
		news++
		if slot != 0 && c.Slot != slot && !c.Walked {
			continue // les MARCHE OK des autres slots restent visibles : ils sont rares
		}
		state := fmt.Sprintf("marche arretee apres i%d", c.StopID)
		if c.Walked {
			state = fmt.Sprintf("MARCHE OK : c%d rang %s", c.Counter, eqRankLabel(c.Rank))
		}
		t.Logf("CANDIDAT  slot %d @%dus (chunk %d pkt %d off %d) %s gardes[%s] masque %v — %s",
			c.Slot, c.TS, c.Chunk, c.Pkt, c.Off, c.Variant, c.Guards, c.Idx, state)
	}
	keys := make([]string, 0, len(profiles))
	for k := range profiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("PROFIL %-40s : %d candidats", k, profiles[k])
	}
	t.Logf("CANDIDATS NON VUS PAR LE STRICT (tous slots) : %d", news)
}
