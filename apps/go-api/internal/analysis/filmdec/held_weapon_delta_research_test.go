package filmdec

// held_weapon_delta_research_test.go — INSTRUMENT DE MESURE (pas de production).
//
// CE QU'IL CHERCHE. Deux canaux du bipede portent le changement d'armement, et ils sont
// complementaires :
//
//   - weapon-state-type-info (i43..i46) dit QUELLE arme occupe un emplacement. En delta il
//     n'entre au masque que lorsque cette identite CHANGE : chaque emission est une prise,
//     un lacher ou un echange, datee a la ms du paquet. Mesure : 0 repetition sur 31.
//   - biped-desired-weapon-set (i42) dit QUEL emplacement est desire, sans nommer l'arme.
//     Il emet beaucoup plus souvent : c'est le candidat pour combler le rappel d'i43..i46.
//
// CE QU'IL NE FAIT PAS. Il ne relie aucune prise a un objet du monde (l'arme au sol qui a
// disparu), et il ne publie rien dans le document de rejeu. Mesure d'abord.
//
// GARDE : HW_FILM porte le repertoire d'UN film (celui qui contient chunk_00). Sans elle le
// test se saute — les films ne sont pas versionnes.
//
//	CGO_ENABLED=0 HW_FILM=<depot>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run HeldWeapon -v -timeout 30m

import (
	"os"
	"sort"
	"testing"
)

const (
	hwFilmEnv = "HW_FILM"
	// compBipedDesiredWeaponSet est l'etiquette de registre d'i42.
	compBipedDesiredWeaponSet = "biped-desired-weapon-set"
	// hwKindIdentity / hwKindSelect distinguent les deux canaux dans un meme flux d'evenements.
	hwKindIdentity = "identite"
	hwKindSelect   = "selection"
)

// hwEvent est UNE emission rattachee a son record.
type hwEvent struct {
	Slot        uint32
	Chunk       int
	TimestampUS uint64
	Kind        string
	// CompIndex : l'emplacement d'arme pour hwKindIdentity ; l'index d'i42 sinon.
	CompIndex int
	// IDHigh / Variant : les deux moities de l'id 64 bits (hwKindIdentity seulement).
	// IDHigh est la FAMILLE, celle que le catalogue de production nomme.
	IDHigh  uint32
	Variant uint32
	// Sel : le R(3) de tete d'i42 (hwKindSelect seulement).
	Sel uint32
}

// hwSetup porte la configuration resolue une fois pour un film.
type hwSetup struct {
	dir       string
	chunks    []int
	slots     map[uint32]bool
	lay       I0Layout
	arch      Archetype
	weaponIdx map[int]bool
	selIdx    int
	minRecord int
}

// hwResolve resout la configuration du film. Les index viennent des NOMS du registre, jamais
// de constantes : un index de composant est un numero de build.
func hwResolve(t *testing.T, dir string) hwSetup {
	t.Helper()
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
		t.Fatalf("aucun slot biped dans les keyframes de %s", dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible : %v", err)
	}
	arch, err := bipedArchetype(dir)
	if err != nil {
		t.Fatalf("archetype biped illisible : %v", err)
	}
	widx, sel := map[int]bool{}, -1
	for id := 0; id < 64; id++ {
		switch arch.component(id) {
		case compWeaponStateTypeInfo:
			widx[id] = true
		case compBipedDesiredWeaponSet:
			sel = id
		}
	}
	if len(widx) == 0 {
		t.Fatalf("aucun %s dans l archetype biped du registre", compWeaponStateTypeInfo)
	}
	return hwSetup{
		dir: dir, chunks: chunks, slots: slots, lay: lay, arch: arch,
		weaponIdx: widx, selIdx: sel,
		minRecord: bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits(),
	}
}

// hwCapture porte ce que les sondes deposent pendant la marche d'UN record.
type hwCapture struct {
	high, low uint32
	gotWeapon bool
	sel       uint32
	gotSel    bool
}

// hwInstall pose les deux sondes et rend la fonction de restauration.
func hwInstall(c *hwCapture) func() {
	prevW, prevS := heldWeaponHook, desiredWeaponSetHook
	SetHeldWeaponHook(func(h, l uint32) { c.high, c.low, c.gotWeapon = h, l, true })
	SetDesiredWeaponSetHook(func(s uint32) { c.sel, c.gotSel = s, true })
	return func() {
		SetHeldWeaponHook(prevW)
		SetDesiredWeaponSetHook(prevS)
	}
}

// hwScan balaye les paquets delta et rend toutes les emissions des deux canaux.
func hwScan(s hwSetup) ([]hwEvent, int, int) {
	var out []hwEvent
	records, withComp := 0, 0
	var cap hwCapture
	defer hwInstall(&cap)()

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
			for p := 0; p+s.minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				records++
				if hwMaskHasTarget(idx, s.weaponIdx, s.selIdx) {
					withComp++
					out = append(out, hwWalk(pay, i0, total, idx, s, &cap, c, slot, pk.TimestampUS)...)
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return out, records, withComp
}

// hwWalk marche UN record et rend les emissions qu'il porte.
func hwWalk(
	pay []byte, i0, total int, idx []int, s hwSetup, cap *hwCapture,
	chunk int, slot uint32, ts uint64,
) []hwEvent {
	var out []hwEvent
	walkRecordComponents(pay, i0, total, idx, s.lay, s.arch, func(id int) bool {
		switch {
		case s.weaponIdx[id] && cap.gotWeapon:
			out = append(out, hwEvent{
				Slot: slot, Chunk: chunk, TimestampUS: ts, Kind: hwKindIdentity,
				CompIndex: id, IDHigh: cap.high, Variant: cap.low,
			})
		case id == s.selIdx && cap.gotSel:
			out = append(out, hwEvent{
				Slot: slot, Chunk: chunk, TimestampUS: ts, Kind: hwKindSelect,
				CompIndex: id, Sel: cap.sel,
			})
		}
		cap.gotWeapon, cap.gotSel = false, false
		return true
	})
	return out
}

func hwMaskHasTarget(idx []int, widx map[int]bool, sel int) bool {
	for _, id := range idx {
		if widx[id] || id == sel {
			return true
		}
	}
	return false
}

// TestHeldWeaponDeltaCensus mesure le volume des deux canaux.
func TestHeldWeaponDeltaCensus(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := hwResolve(t, dir)
	t.Logf("film=%s chunks=%d slots_biped=%d emplacements_arme=%d index_i42=%d",
		dir, len(s.chunks), len(s.slots), len(s.weaponIdx), s.selIdx)

	ev, records, withComp := hwScan(s)
	ident, sel := 0, 0
	for _, e := range ev {
		if e.Kind == hwKindIdentity {
			ident++
		} else {
			sel++
		}
	}
	t.Logf("records biped delta ancres=%d ; masque portant une cible=%d", records, withComp)
	t.Logf("emissions : identite (i43..i46)=%d ; selection (i42)=%d", ident, sel)

	if ident == 0 {
		t.Log("VERDICT : aucune emission d identite. Rien ne peut etre conclu sur ce film.")
		return
	}
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].TimestampUS < ev[j].TimestampUS })
	type key struct {
		slot uint32
		comp int
	}
	prev := map[key]uint32{}
	seen := map[key]bool{}
	var repeats int
	for _, e := range ev {
		if e.Kind != hwKindIdentity {
			continue
		}
		k := key{e.Slot, e.CompIndex}
		if seen[k] && e.IDHigh == prev[k] {
			repeats++
		}
		seen[k], prev[k] = true, e.IDHigh
	}
	t.Logf("couples (slot, emplacement) vus=%d ; emissions d identite REPETEES=%d", len(seen), repeats)
	t.Log("LECTURE : 0 repetition = la grammaire reserve i43..i46 au CHANGEMENT, donc chaque " +
		"emission est une prise, un lacher ou un echange date a la ms du paquet.")
}

// TestHeldWeaponChangesProduction confronte le balayage de PRODUCTION
// (ScanFilmHeldWeaponChanges) a l'instrument de mesure : ils doivent voir la meme chose.
// Sans cette confrontation, le portage en production pourrait deriver en silence.
func TestHeldWeaponChangesProduction(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := hwResolve(t, dir)
	ref := hwKeyframeRef(t, dir)
	want := hwIdentities(hwScanEvents(s))

	got, st, err := ScanFilmHeldWeaponChanges(dir, ref.setAt)
	if err != nil {
		t.Fatalf("balayage de production : %v", err)
	}
	t.Logf("PRODUCTION : records=%d masquesPorteurs=%d emissions=%d repetitions=%d",
		st.Records, st.WithComponent, st.Emissions, st.Repeats)
	// LA PROPRIETE N'EST PAS ABSOLUE, ET C'EST MESURE. On a d'abord observe « 0 repetition »
	// sur un film de 31 emissions et on en a fait une propriete du format. Sur des films plus
	// fournis il en reste UNE par film : 1/229 et 1/100, soit 0,4 % et 1,0 %. La liste des
	// prises reste donc une liste de prises a ~99 %, mais l'affirmation « zero repetition »
	// etait une generalisation depuis un petit echantillon. Le seuil ci-dessous est un
	// RATCHET : il casse si le taux se degrade, pas si l'exception connue subsiste.
	if pct := 100 * float64(st.Repeats) / float64(max(st.Emissions, 1)); pct > 2.0 {
		t.Errorf("REPETITIONS = %d sur %d emissions (%.1f %%) : au-dela de 2 %%, le composant "+
			"n'entre plus au masque QUE sur changement et la liste cesse d'etre une liste de "+
			"prises.", st.Repeats, st.Emissions, pct)
	} else {
		t.Logf("repetitions = %d sur %d emissions (%.1f %%) — sous le ratchet de 2 %%",
			st.Repeats, st.Emissions, pct)
	}
	if len(got) != len(want) {
		t.Fatalf("production=%d emissions, instrument=%d : les deux chemins divergent",
			len(got), len(want))
	}
	byKind := map[HeldWeaponChangeKind]int{}
	for i, c := range got {
		if c.TimestampUS != want[i].TimestampUS || c.Slot != want[i].Slot ||
			c.SlotIndex != want[i].CompIndex || c.Family != want[i].IDHigh {
			t.Fatalf("divergence a l'index %d : production=%+v instrument=%+v", i, c, want[i])
		}
		byKind[c.Kind]++
	}
	t.Logf("PAR NATURE : %v", byKind)
}
