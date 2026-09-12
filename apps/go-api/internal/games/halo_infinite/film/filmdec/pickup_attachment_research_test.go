package filmdec

// pickup_attachment_research_test.go — LE RAMASSAGE COMME ATTACHEMENT.
//
// L'HYPOTHESE, ecrite avant la mesure. Ramasser un objet ne le DETRUIT pas : il devient
// ENFANT du bipede qui le porte. La mesure des cycles de vie l'a montre par la negative — le
// lacher coincide avec une NAISSANCE d'arme au sol au bit du meme paquet (+0 ms, 4 fois sur
// 4), alors que la prise ne coincide avec AUCUNE mort (etalement de -11 a +11 s). Le canal du
// ramassage est donc `object-parent-state` (i10), dont Ghidra a etabli le sens le meme jour :
// le champ lu par FUN_1406d3140 vaut 0xFFFFFFFF quand l'objet est libre et porte un handle
// quand il est attache (branche detachee de FUN_140c1e4d0, sentinelles +0x274/+0x278/+0x27a).
//
// PORTEE : armes au sol (ti=42) ET equipements (ti=37) — les deux archetypes portent i10, et
// rien ne laisse penser que le moteur traite les deux differemment. Le test le VERIFIE au lieu
// de le supposer.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"os"
	"sort"
	"testing"
)

// paTransition est un passage d'etat d'attachement d'une entite du monde.
type paTransition struct {
	TimestampUS uint64
	Slot        uint32
	TypeIndex   uint32
	// Attached : l'etat APRES la transition. true = l'objet vient d'etre pris en charge.
	Attached bool
}

// paScanAttachments rend les passages detache <-> attache des archetypes demandes.
//
// Le rattachement d'une lecture i10 a son record est EXACT : la sonde publie la position de
// bit du debut de la lecture, la trace du record publie la meme position pour ce composant,
// et les deux se joignent par EGALITE — jamais par voisinage.
func paScanAttachments(t *testing.T, dir string, want map[uint32]bool) []paTransition {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("registre (chunk_00) illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	vues := map[int]ObjectParentState{}
	prev := objectParentStateHook
	SetObjectParentStateHook(func(s ObjectParentState) { vues[s.StartBit] = s })
	defer SetObjectParentStateHook(prev)

	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	state := map[uint32]bool{}
	seen := map[uint32]bool{}
	var out []paTransition
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			switch p.Type {
			case PacketTypeKeyframe:
				w = WorldFromKeyframe(reg, pay)
				continue
			case PacketTypeDelta:
			default:
				continue
			}
			for k := range vues {
				delete(vues, k)
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, gwViewsPerPacket, cfg.PacketPreambleBits)
			out = append(out, paCollect(recs, vues, want, state, seen, p.TimestampUS)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// paCollect rattache les lectures d'UN paquet a leurs records et rend les transitions.
func paCollect(
	recs []FrameRecord, vues map[int]ObjectParentState, want map[uint32]bool,
	state, seen map[uint32]bool, ts uint64,
) []paTransition {
	var out []paTransition
	for _, r := range recs {
		if !want[r.TypeIndex] || r.DesyncAt != -1 {
			continue
		}
		for _, comp := range r.Trace.Comps {
			if comp.Name != "object-parent-state-component" {
				continue
			}
			st, ok := vues[comp.StartBit]
			if !ok {
				continue
			}
			if seen[r.Slot] && state[r.Slot] == st.Attached {
				continue // pas de changement : rien a signaler
			}
			if seen[r.Slot] {
				out = append(out, paTransition{
					TimestampUS: ts, Slot: r.Slot, TypeIndex: r.TypeIndex, Attached: st.Attached,
				})
			}
			seen[r.Slot], state[r.Slot] = true, st.Attached
		}
	}
	return out
}

// paNear dit s'il existe une transition du sens demande, sur un des archetypes voulus, a
// moins de gwWindowUS de at.
func paNear(tr []paTransition, at uint64, attached bool, ti uint32) bool {
	for _, e := range tr {
		if e.Attached != attached || e.TypeIndex != ti {
			continue
		}
		if gwAbs(int64(e.TimestampUS)-int64(at)) <= gwWindowUS {
			return true
		}
	}
	return false
}

// TestPickupIsAttachment confronte les prises et les lachers lus sur le BIPEDE aux passages
// d'attachement des objets du monde — armes au sol ET equipements.
func TestPickupIsAttachment(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : une PRISE doit coincider, a moins de 0,5 s, avec " +
		"une entite du monde passant de DETACHEE a ATTACHEE ; un LACHER avec le passage " +
		"inverse. Seuil >= 70 %, temoin decale de 30 s <= 15 %. Sous ces valeurs le mecanisme " +
		"d'attachement n'est pas etabli et rien ne doit etre publie.")

	want := map[uint32]bool{GroundWeaponTypeIndex: true, paEquipmentTypeIndex: true}
	tr := paScanAttachments(t, dir, want)
	byKind := map[string]int{}
	for _, e := range tr {
		k := "ti=42"
		if e.TypeIndex == paEquipmentTypeIndex {
			k = "ti=37"
		}
		if e.Attached {
			byKind[k+" attache"]++
		} else {
			byKind[k+" detache"]++
		}
	}
	t.Logf("MONDE : transitions d'attachement = %v (total=%d)", byKind, len(tr))
	if len(tr) == 0 {
		t.Log("VERDICT : aucune transition d'attachement lue. Le mecanisme ne peut pas etre " +
			"teste sur ce film.")
		return
	}

	prises, lachers := paBipedEvents(t, dir)
	t.Logf("BIPEDE : prises=%d lachers=%d", len(prises), len(lachers))

	score := func(times []uint64, attached bool, ti uint32, shift uint64) (int, int) {
		hit := 0
		for _, at := range times {
			if paNear(tr, at+shift, attached, ti) {
				hit++
			}
		}
		return hit, len(times)
	}
	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}
	ph, pn := score(prises, true, GroundWeaponTypeIndex, 0)
	phW, _ := score(prises, true, GroundWeaponTypeIndex, gwWitnessShiftUS)
	lh, ln := score(lachers, false, GroundWeaponTypeIndex, 0)
	lhW, _ := score(lachers, false, GroundWeaponTypeIndex, gwWitnessShiftUS)

	t.Logf("PRISE  -> ATTACHEMENT (ti=42) : %d/%d (%.1f %%) ; temoin : %d/%d (%.1f %%)",
		ph, pn, pct(ph, pn), phW, pn, pct(phW, pn))
	t.Logf("LACHER -> DETACHEMENT (ti=42) : %d/%d (%.1f %%) ; temoin : %d/%d (%.1f %%)",
		lh, ln, pct(lh, ln), lhW, ln, pct(lhW, ln))
	t.Log("EQUIPEMENT (ti=37) : le canal cote BIPEDE est i26 unit-equipment-component, qui " +
		"n'est pas decode en valeurs — le volume de transitions ti=37 ci-dessus dit seulement " +
		"si le meme mecanisme existe pour les equipements, pas s'il s'apparie a un porteur.")
}

// paEquipmentTypeIndex est l'archetype des equipements / objets du monde.
const paEquipmentTypeIndex = 37

// paBipedEvents rend les instants des prises et des lachers lus sur le bipede.
func paBipedEvents(t *testing.T, dir string) (prises, lachers []uint64) {
	t.Helper()
	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	kfRef := hwKeyframeRef(t, dir)
	type key struct {
		slot uint32
		comp int
	}
	prev, seen := map[key]uint32{}, map[key]bool{}
	for _, e := range ev {
		k := key{e.Slot, e.CompIndex}
		switch {
		case e.IDHigh == noVariant:
			lachers = append(lachers, e.TimestampUS)
		case seen[k] && prev[k] == noVariant:
			prises = append(prises, e.TimestampUS)
		case !seen[k]:
			if ref, ok := kfRef.setAt(e.Slot, e.TimestampUS); ok && !ref[e.IDHigh] {
				prises = append(prises, e.TimestampUS)
			}
		}
		seen[k], prev[k] = true, e.IDHigh
	}
	return prises, lachers
}
