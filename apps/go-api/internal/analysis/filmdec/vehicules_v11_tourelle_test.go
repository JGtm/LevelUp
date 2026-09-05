package filmdec

// vehicules_v11_tourelle_test.go — INSTRUMENT DU LOT V11 : « la TOURELLE replique-t-elle
// quelque chose, et QUOI ? » (2026-09-03).
//
// LA QUESTION. Le lot V8 a etabli que la tourelle d'artilleur d'un warthog / falcon est une
// entite `ti=40` A PART, recensee aux images-cles au slot `chassis - 1`, MUETTE au sens du
// balayage de positions (zero echantillon accepte, zero record de creation). « Muette » n'y
// veut dire qu'une chose : aucun record avec un `i0` ABSOLU de la region attendue. Le balayage
// `ScanBipedRecords` exige en effet un `i0` absolu ET un masque dont le premier index vaut 0 :
// un record qui ne replique QUE `i41 vehicle-seats-override-pitch` / `i42 ...-yaw` lui est
// STRUCTURELLEMENT invisible. La question posee ici est donc l'autre : dans la MARCHE
// SEQUENTIELLE des records (celle qui lit tous les masques, i0 present ou non), le slot de la
// tourelle emet-il des records, et quels composants portent-ils ?
//
// DEUX POPULATIONS, ET C'EST LA MESURE :
//
//	CHASSIS  slot `ti=40` avec au moins un echantillon de position accepte.
//	MUET     slot `ti=40` recense aux images-cles, zero echantillon accepte. Les candidats
//	         tourelle du lot V8 en sont un sous-ensemble (ceux dont `slot+1` est un chassis
//	         recense a la MEME fenetre).
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS=0d76e8f1,fccc61cd \
//	  go test ./internal/analysis/filmdec/ -run TestV11 -v -timeout 120m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	v11RootEnv  = "V11_ROOT"  // racine du cache (contient film_chunks/)
	v11FilmsEnv = "V11_FILMS" // short8 separes par des virgules
)

// v11VehiculeTI est l'archetype vehicule.
const v11VehiculeTI = 40

// v11VuesParPaquet : nombre de vues decodees par paquet delta (meme valeur que l'instrument
// V5 — un paquet porte jusqu'a 3 vues plus une marge).
const v11VuesParPaquet = 4

// v11Films rend les repertoires de chunks demandes.
func v11Films(t *testing.T) []string {
	t.Helper()
	root, films := os.Getenv(v11RootEnv), os.Getenv(v11FilmsEnv)
	if root == "" || films == "" {
		t.Skipf("mesure non demandee : %s ou %s vide", v11RootEnv, v11FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(films, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, root+"/film_chunks/"+s)
		}
	}
	return out
}

// v11SlotStat porte ce que la marche sequentielle a vu d'UN slot.
type v11SlotStat struct {
	Records  int
	Desync   int
	Masque   map[int]int // index de composant -> nombre de records qui le portent
	Formes   map[string]int
	Premier  uint64
	Dernier  uint64
	Nonvides int
}

func newV11SlotStat() *v11SlotStat {
	return &v11SlotStat{Masque: map[int]int{}, Formes: map[string]int{}}
}

// v11MasqueIdx rend les index de composants d'un masque 64 bits, en ordre croissant.
func v11MasqueIdx(m uint64) []int {
	var out []int
	for i := 0; i < 64; i++ {
		if m&(uint64(1)<<uint(i)) != 0 {
			out = append(out, i)
		}
	}
	return out
}

// v11FormeMasque rend la signature textuelle d'un masque (« i0,i1,i2 »).
func v11FormeMasque(idx []int) string {
	if len(idx) == 0 {
		return "(vide)"
	}
	parts := make([]string, 0, len(idx))
	for _, i := range idx {
		parts = append(parts, fmt.Sprintf("i%d", i))
	}
	return strings.Join(parts, ",")
}

// v11Marche decode tout le film en marche sequentielle stateful et rend, par slot `ti=40`, le
// releve des records vus. Renvoie aussi les compteurs globaux de qualite de la marche.
func v11Marche(dir string) (map[uint32]*v11SlotStat, map[string]int, error) {
	brut, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("chunk_00 illisible : %w", err)
	}
	reg, err := ParseRegistryChunk(brut)
	if err != nil {
		return nil, nil, fmt.Errorf("registre illisible : %w", err)
	}
	stat := map[string]int{}
	parSlot := map[uint32]*v11SlotStat{}
	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			if p.Type == PacketTypeKeyframe {
				w = WorldFromKeyframe(reg, pay)
				stat["keyframes"]++
				continue
			}
			if p.Type != PacketTypeDelta {
				continue
			}
			stat["paquets"]++
			recs, _ := DecodeFrameViews(pay, w, cfg, v11VuesParPaquet, cfg.PacketPreambleBits)
			v11Collecte(recs, parSlot, stat, p.TimestampUS)
		}
	}
	return parSlot, stat, nil
}

// v11Collecte range les records d'un paquet par slot, pour le seul archetype vehicule.
func v11Collecte(recs []FrameRecord, parSlot map[uint32]*v11SlotStat, stat map[string]int, ts uint64) {
	for _, r := range recs {
		stat["records"]++
		stat[fmt.Sprintf("records_ti%d", r.TypeIndex)]++
		if r.DesyncAt != -1 {
			stat["desync"]++
		}
		if r.TypeIndex != v11VehiculeTI {
			continue
		}
		s := parSlot[r.Slot]
		if s == nil {
			s = newV11SlotStat()
			parSlot[r.Slot] = s
			s.Premier = ts
		}
		s.Records++
		s.Dernier = ts
		if r.DesyncAt != -1 {
			s.Desync++
		}
		idx := v11MasqueIdx(r.Trace.Mask)
		if len(idx) > 0 {
			s.Nonvides++
		}
		for _, i := range idx {
			s.Masque[i]++
		}
		s.Formes[v11FormeMasque(idx)]++
	}
}

// TestV11TourelleRecords — LA MESURE : le slot d'une tourelle emet-il des records, et
// lesquels ? Elle croise trois sources qui ne se recouvrent pas :
//
//	le RECENSEMENT des images-cles      quelles vies `ti=40` existent (slot, gen, fenetre) ;
//	le BALAYAGE de positions            quels slots emettent un record a `i0` absolu ;
//	la MARCHE SEQUENTIELLE des records  quels slots emettent un record TOUT COURT, et quel
//	                                    masque il porte.
//
// Un slot recense, sans position, mais qui emet des records dans la marche, est le seul cas
// qui ouvrirait la piste « la tourelle replique son orientation ».
func TestV11TourelleRecords(t *testing.T) {
	for _, dir := range v11Films(t) {
		v11TourelleUnFilm(t, dir)
	}
}

func v11TourelleUnFilm(t *testing.T, dir string) {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		t.Logf("V11 %s : film absent — saute", dir)
		return
	}
	release := LockProcessDecode()
	defer release()
	kf := ScanFilmWorldObjectKeyframes(dir, v11VehiculeTI)
	if len(kf.Band) == 0 {
		t.Logf("V11 %s : bande ti=40 vide", dir)
		return
	}
	avecPos := v11SlotsAvecPosition(t, dir, kf.Band)
	parSlot, stat, err := v11Marche(dir)
	if err != nil {
		t.Logf("V11 %s : %v", dir, err)
		return
	}
	t.Logf("V11 MARCHE %s — paquets=%d records=%d dont desync=%d · records ti=40=%d ti=35=%d "+
		"| bande ti=40 = %d slots, vies recensees = %d, slots avec position = %d",
		dir, stat["paquets"], stat["records"], stat["desync"], stat["records_ti40"],
		stat["records_ti35"], len(kf.Band), len(kf.SeenUS), len(avecPos))

	rec, muets := v11ClasseSlots(kf, avecPos)
	t.Logf("V11 CLASSES %s — slots recenses=%d dont CHASSIS (avec position)=%d MUETS=%d",
		dir, len(rec), len(rec)-len(muets), len(muets))
	v11PublieClasse(t, dir, "CHASSIS", v11Sous(parSlot, rec, muets, false))
	v11PublieClasse(t, dir, "MUETS", v11Sous(parSlot, rec, muets, true))
	v11PublieMuetsDetail(t, dir, kf, muets, parSlot)
}

// v11SlotsAvecPosition rend les slots de la bande qui emettent au moins un record a `i0`
// absolu accepte. Flux BRUT : aucun post-filtre ne peut avoir efface un slot.
func v11SlotsAvecPosition(t *testing.T, dir string, band map[uint32]bool) map[uint32]int {
	t.Helper()
	opt := ScanFilmOptions{RequireTag1: false, DropSaturated: true, QuantaOnly: true}
	pos, err := ScanFilmBipedPositionsForBand(dir, NewSlotBand(band), opt)
	if err != nil {
		t.Logf("V11 %s : balayage de positions : %v", dir, err)
		return map[uint32]int{}
	}
	out := map[uint32]int{}
	for _, p := range pos {
		out[p.Slot]++
	}
	return out
}

// v11ClasseSlots rend l'ensemble des slots RECENSES aux images-cles et le sous-ensemble MUET.
func v11ClasseSlots(kf WorldObjectKeyframes, avecPos map[uint32]int) (rec, muets map[uint32]bool) {
	rec, muets = map[uint32]bool{}, map[uint32]bool{}
	for k := range kf.SeenUS {
		s := uint32(k.Slot)
		rec[s] = true
		if avecPos[s] == 0 {
			muets[s] = true
		}
	}
	return rec, muets
}

// v11Sous rend le releve agrege d'une classe de slots.
func v11Sous(parSlot map[uint32]*v11SlotStat, rec, muets map[uint32]bool, muet bool) *v11SlotStat {
	out := newV11SlotStat()
	for s := range rec {
		if muets[s] != muet {
			continue
		}
		st := parSlot[s]
		if st == nil {
			continue
		}
		out.Records += st.Records
		out.Desync += st.Desync
		out.Nonvides += st.Nonvides
		for i, n := range st.Masque {
			out.Masque[i] += n
		}
		for f, n := range st.Formes {
			out.Formes[f] += n
		}
	}
	return out
}

// v11PublieClasse imprime le releve d'une classe : records vus, histogramme des composants du
// masque, et les formes de masque les plus frequentes.
func v11PublieClasse(t *testing.T, dir, nom string, s *v11SlotStat) {
	t.Helper()
	if s.Records == 0 {
		t.Logf("V11 %s [%s] — AUCUN record vu par la marche sequentielle", dir, nom)
		return
	}
	idx := make([]int, 0, len(s.Masque))
	for i := range s.Masque {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var b strings.Builder
	for _, i := range idx {
		fmt.Fprintf(&b, "i%d=%.1f%% ", i, 100*float64(s.Masque[i])/float64(s.Records))
	}
	t.Logf("V11 %s [%s] — %d records (desync %d, masque non vide %d)\n    masque : %s",
		dir, nom, s.Records, s.Desync, s.Nonvides, b.String())
	t.Logf("    formes les plus frequentes : %s", v11TopFormes(s.Formes, 6))
}

// v11TopFormes rend les n formes de masque les plus frequentes.
func v11TopFormes(f map[string]int, n int) string {
	type kv struct {
		K string
		N int
	}
	all := make([]kv, 0, len(f))
	for k, v := range f {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].N != all[j].N {
			return all[i].N > all[j].N
		}
		return all[i].K < all[j].K
	})
	if len(all) > n {
		all = all[:n]
	}
	parts := make([]string, 0, len(all))
	for _, e := range all {
		parts = append(parts, fmt.Sprintf("%s (%d)", e.K, e.N))
	}
	return strings.Join(parts, " · ")
}

// v11PublieMuetsDetail imprime, slot par slot, ce que la marche a vu des slots MUETS et de
// leur voisin `slot+1` — le motif de la tourelle du lot V8.
func v11PublieMuetsDetail(t *testing.T, dir string, kf WorldObjectKeyframes, muets map[uint32]bool,
	parSlot map[uint32]*v11SlotStat) {
	t.Helper()
	slots := make([]uint32, 0, len(muets))
	for s := range muets {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		st, vois := parSlot[s], parSlot[s+1]
		nRec, nVois := 0, 0
		if st != nil {
			nRec = st.Records
		}
		if vois != nil {
			nVois = vois.Records
		}
		forme := "-"
		if st != nil {
			forme = v11TopFormes(st.Formes, 3)
		}
		t.Logf("V11 %s MUET slot=%d — records marche=%d · voisin slot+1 records=%d · formes : %s",
			dir, s, nRec, nVois, forme)
	}
	_ = kf
}
