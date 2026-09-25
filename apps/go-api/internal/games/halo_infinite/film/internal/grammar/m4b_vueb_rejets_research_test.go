//go:build research

package grammar

// m4b_vueb_rejets_research_test.go — LOT M4b : POURQUOI LA VUE C N EST PAS LUE. Mesure seule.
//
// La vue C n est lue que si la vue B clot sa liste ; et une vue B « close » par le REJET d un
// en-tete (slot absent de la table de datums) finit peut-etre sur un record legitime que le monde
// hors ligne ne connait pas — la vue C lue ensuite ne ferme alors pas le paquet. L instrument
// rejoue la marche de production (monde, liaisons d image-cle, table anticipee, localisateur
// strict) et, pour chaque paquet, relit l en-tete sur lequel la vue B s est arretee :
//
//	terminateur (type 0)  -> fin de vue ecrite par le jeu
//	rejet                 -> le slot, son archetype d image-cle ULTERIEURE s il en a un, et la
//	                         vue C qui suit : fermee ou non.
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go` ; M4B_FENETRES=a-b,c-d
// restreint le detail des rejets aux trames listees.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// m4bFin est la fin de vue B d un paquet, relue.
type m4bFin struct {
	trame      int
	rejet      bool
	typ        int
	slot, gen  uint32
	vueC       LectureVueC
	nRecs      int
	dernierTI  uint32
	nonLocalis bool
}

// TestM4bRejetsDeVueB publie, par cause, la fin de la vue B et le verdict de la vue C.
func TestM4bRejetsDeVueB(t *testing.T) {
	cad := s3LireCadre(t)
	var fenetres [][2]int
	for _, s := range strings.Split(os.Getenv("M4B_FENETRES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			fenetres = append(fenetres, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	fins := m4bMarcher(t, tc, cad)
	type agg struct{ n, fermes int }
	par := map[string]*agg{}
	slots := map[string]int{}
	for _, f := range fins {
		k := "terminateur"
		switch {
		case f.nonLocalis:
			k = "liste non localisee"
		case f.rejet:
			k = "rejet"
		case !f.vueC.Atteinte:
			k = "vue B ouverte (desync)"
		}
		a := par[k]
		if a == nil {
			a = &agg{}
			par[k] = a
		}
		a.n++
		a.fermes += s3B(f.vueC.Fermee)
		if f.rejet && !f.vueC.Fermee && m4bDans(f.trame, fenetres) {
			slots[fmt.Sprintf("slot %5d gen %d (dernier ti %d)", f.slot, f.gen, f.dernierTI)]++
		}
	}
	for k, a := range par {
		t.Logf("== fin de vue B : %-24s %6d paquets · vue C fermee %6d", k, a.n, a.fermes)
	}
	cles := make([]string, 0, len(slots))
	for k := range slots {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return slots[cles[i]] > slots[cles[j]] })
	t.Logf("== REJETS SANS FERMETURE dans les fenetres %v : %d slots distincts", fenetres, len(cles))
	for i, k := range cles {
		if i == 40 {
			break
		}
		t.Logf("   %s x %d", k, slots[k])
	}
}

// m4bDans dit si une trame tombe dans une des fenetres (toutes quand la liste est vide).
func m4bDans(tr int, fs [][2]int) bool {
	if len(fs) == 0 {
		return true
	}
	for _, f := range fs {
		if tr >= f[0] && tr <= f[1] {
			return true
		}
	}
	return false
}

// m4bDebut rend le debut de liste de PRODUCTION ([debutDeLaListe]).
func m4bDebut(pay []byte, w *World, cfg FrameConfig) int {
	d, _ := debutDeLaListe(pay, w, cfg)
	return d
}

// m4bMarcher rejoue la marche de production (`ScanMarcheDesTrames`) et rend, par paquet delta,
// la fin de sa vue B relue.
func m4bMarcher(t *testing.T, tc t516Temoin, cad s3Cadre) []m4bFin {
	t.Helper()
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	var derniere LectureVueC
	obs.VueControleHook = func(l LectureVueC) { derniere = l }
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	var out []m4bFin
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			f := m4bFin{trame: cad.trame(pk.TimestampUS)}
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					f.nonLocalis = true
					out = append(out, f)
					continue
				}
			}
			derniere = LectureVueC{}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			f.vueC, f.nRecs = derniere, len(recs)
			m4bRelireFin(pay, recs, debut, cfg, &f)
			out = append(out, f)
		}
	}
	return out
}

// m4bRelireFin relit l en-tete qui suit le dernier record rendu de la vue B.
func m4bRelireFin(pay []byte, recs []FrameRecord, debut int, cfg FrameConfig, f *m4bFin) {
	at := debut
	if n := len(recs); n > 0 {
		at = recs[n-1].Trace.EndBit
		f.dernierTI = recs[n-1].TypeIndex
	}
	if at <= 0 || at >= len(pay)*8 {
		return
	}
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(at)
	if cfg.HasExtraFields {
		br.Skip(32)
	}
	f.typ = readRecordType(br)
	if f.typ == recEnd {
		return
	}
	id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
	f.rejet, f.slot, f.gen = f.vueC.Atteinte, id&0x3fffffff, id>>30
}

// TestM4bSlotsRejetes suit les slots rejetes nommes par M4B_SLOTS : leurs declarations d image-cle
// (instant, generation, archetype), et les records NEW / DEL / DELTA que la marche en lit.
func TestM4bSlotsRejetes(t *testing.T) {
	cad := s3LireCadre(t)
	suivis := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("M4B_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
			suivis[v] = true
		}
	}
	tc := t516Cadre(t)
	marche := tc.fc.MarcheDImageCle()
	lignes := map[uint32][]string{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			mp := marche.Marcher(pk.Payload(data))
			for _, r := range mp.Records {
				if suivis[uint32(r.Slot)] { //nolint:gosec // slot borne
					lignes[uint32(r.Slot)] = append(lignes[uint32(r.Slot)], //nolint:gosec // idem
						fmt.Sprintf("image-cle c%d t%d gen %d ti %d", c, cad.trame(pk.TimestampUS), r.Gen, r.TI))
				}
			}
			table, _ := TableDeDatums(pk.Payload(data))
			for s, ti := range table {
				if suivis[s] {
					lignes[s] = append(lignes[s], fmt.Sprintf("datum c%d t%d ti %d", c,
						cad.trame(pk.TimestampUS), ti))
				}
			}
		}
	}
	for s := range suivis {
		t.Logf("== slot %d : %d declarations", s, len(lignes[s]))
		for i, l := range lignes[s] {
			if i == 12 {
				t.Logf("   ...")
				break
			}
			t.Logf("   %s", l)
		}
	}
}

// TestM4bRecordsDesSlots compte, pour les slots de M4B_SLOTS, les records que la marche de
// production LIT (NEW, DEL, DELTA) et les rejets ou ils tombent en en-tete, avec les trames.
func TestM4bRecordsDesSlots(t *testing.T) {
	cad := s3LireCadre(t)
	suivis := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("M4B_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
			suivis[v] = true
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	vus := map[string]int{}
	premiers := map[string]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			for _, r := range recs {
				if !suivis[r.Slot] {
					continue
				}
				k := fmt.Sprintf("slot %d type %d ti %d desync %v", r.Slot, r.Type, r.TypeIndex, r.DesyncAt >= 0)
				vus[k]++
				if _, deja := premiers[k]; !deja {
					premiers[k] = cad.trame(pk.TimestampUS)
				}
			}
		}
	}
	for k, n := range vus {
		t.Logf("   %s x %d (premiere trame %d)", k, n, premiers[k])
	}
}

// TestM4bJournalDesPaquets journalise, pour les trames de M4B_FENETRES, la marche de chaque
// paquet : records rendus (type, slot, archetype, desync), fin de la vue B et verdict de la vue C.
func TestM4bJournalDesPaquets(t *testing.T) {
	cad := s3LireCadre(t)
	var fenetres [][2]int
	for _, s := range strings.Split(os.Getenv("M4B_FENETRES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			fenetres = append(fenetres, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	var derniere LectureVueC
	obs.VueControleHook = func(l LectureVueC) { derniere = l }
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			tr := cad.trame(pk.TimestampUS)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					if m4bDans(tr, fenetres) {
						t.Logf("t%d c%d p%d : liste non localisee (%d bits)", tr, c, pk.Index, len(pay)*8)
					}
					continue
				}
			}
			derniere = LectureVueC{}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			if !m4bDans(tr, fenetres) {
				continue
			}
			var f m4bFin
			f.vueC = derniere
			m4bRelireFin(pay, recs, debut, cfg, &f)
			var sb strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&sb, "%s%d/%d%s ", map[int]string{1: "N", 2: "D", 3: ""}[r.Type], r.Slot,
					r.TypeIndex, map[bool]string{true: "!", false: ""}[r.DesyncAt >= 0])
				if r.DesyncAt >= 0 {
					for _, cr := range r.Trace.Comps {
						fmt.Fprintf(&sb, "[i%d %s porte=%v] ", cr.Index, cr.Name, cr.Ported)
					}
					fmt.Fprintf(&sb, "desyncAt=%d masque=%x ", r.DesyncAt, r.Trace.Mask)
				}
			}
			fin := "fin"
			if f.rejet {
				fin = fmt.Sprintf("REJET %d/g%d", f.slot, f.gen)
			} else if !f.vueC.Atteinte {
				fin = "B ouverte"
			}
			t.Logf("t%d c%d p%d %d bits : %s| %s | C atteinte %v fermee %v entrees %d", tr, c, pk.Index,
				len(pay)*8, sb.String(), fin, f.vueC.Atteinte, f.vueC.Fermee, len(f.vueC.Entrees))
		}
	}
}

// TestM4bCausesDesTrous ventile les paquets NON LUS (vue C non fermee) du film entier par leur
// cause racine : liste non localisee ; vue B ouverte par la desynchronisation d un record (le
// composant NON PORTE qui l arrete, par archetype) ; vue B close sur un REJET d en-tete (l archetype
// du record qui le precede) ; vue C lue sans fermer apres un terminateur.
func TestM4bCausesDesTrous(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	var derniere LectureVueC
	obs.VueControleHook = func(l LectureVueC) { derniere = l }
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	causes := map[string]int{}
	total := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			total++
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					causes["liste non localisee"]++
					continue
				}
			}
			derniere = LectureVueC{}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			if derniere.Fermee {
				continue
			}
			causes[m4bCauseDuTrou(pay, recs, debut, cfg, derniere)]++
		}
	}
	cles := make([]string, 0, len(causes))
	for k := range causes {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return causes[cles[i]] > causes[cles[j]] })
	t.Logf("== CAUSES DES TROUS (%d paquets delta) :", total)
	for i, k := range cles {
		if i == 30 {
			break
		}
		t.Logf("   %6d  %s", causes[k], k)
	}
}

// m4bCauseDuTrou nomme la cause d un paquet non lu dont la marche a rendu `recs`.
func m4bCauseDuTrou(pay []byte, recs []FrameRecord, debut int, cfg FrameConfig, l LectureVueC) string {
	if !l.Atteinte {
		for _, r := range recs {
			if r.DesyncAt < 0 {
				continue
			}
			for _, cr := range r.Trace.Comps {
				if !cr.Ported {
					return fmt.Sprintf("vue B ouverte : ti=%d %s i%d %s", r.TypeIndex,
						map[int]string{1: "NEW", 3: "DELTA"}[r.Type], cr.Index, cr.Name)
				}
			}
			return fmt.Sprintf("vue B ouverte : ti=%d desync sans composant non porte", r.TypeIndex)
		}
		return "vue B ouverte : aucun record desynchronise rendu"
	}
	var f m4bFin
	f.vueC = l
	m4bRelireFin(pay, recs, debut, cfg, &f)
	if f.rejet {
		return fmt.Sprintf("rejet d en-tete apres ti=%d", f.dernierTI)
	}
	return fmt.Sprintf("vue C lue sans fermer (arret %d) apres un terminateur", l.Arret)
}
