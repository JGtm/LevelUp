package filmdec

// r11_entite_research_test.go — L'OBJET EN MAIN, ET SES CHARGES.
//
// POURQUOI CE SECOND GISEMENT. La mesure d'i56 (`r11_charges_research_test.go`) etablit que
// le canal de charge du BIPEDE ne s'arme que pour le grappin et le propulseur, jamais pour le
// repulseur. Restent les charges portees par l'ENTITE d'equipement elle-meme (ti=37,
// `equipment-charges-remaining` i27, R(8)) — le report 2 du registre R9.
//
// CE QUE L'ANCRE CHANGE. R9 a juge ce canal sur une jointure d'identite globale (13 % par les
// poses, 3,2 % par les handles d'i26) et sur un recensement de masque qui le laissait a 4x le
// plancher de bruit. Ici on ne demande plus une jointure de corpus : on demande UNE entite —
// celle que JGtm tient a 2:47 sur `72b0a25e` — et on regarde ses charges. Le handle vient
// d'i26 `unit-equipment-component`, le canal cote PORTEUR (70,2 % des prises d'equipement
// gagnent une entree i26 dans la seconde, mesure du 2026-08-30).
//
// TROIS SORTIES, dans cet ordre :
//  1. les emissions d'i26 du joueur suivi, avec leur en-tete R(3) et leurs entrees ;
//  2. l'histogramme des valeurs d'i27 sur tout le film — c'est lui qui dit si ce champ porte
//     des CHARGES (des petits entiers) ou du bruit de reconnaissance (R9 par. 3.3 : 247, 251,
//     244, des valeurs impossibles pour une charge d'equipement) ;
//  3. la serie des charges des entites que le joueur suivi a referencees.
//
// GARDES : `R9_FILMS`, `R9_ARTIFACTS`, `R8_BOUNDS`, `R11_IDS` (obligatoire), `R11_XUID`.
// Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R9_FILMS=<repo>/data/cache/film_chunks \
//	  R9_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  R8_BOUNDS=<wt>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R11_IDS=72b0a25e go test ./internal/analysis/filmdec/ \
//	  -run '^TestR11Entite$' -count=1 -timeout 120m -v

import (
	"fmt"
	"sort"
	"testing"
)

func TestR11Entite(t *testing.T) {
	for _, dir := range r11FilmDirs(t) {
		r11EntiteOneFilm(t, dir)
	}
}

func r11EntiteOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r11Prepare(t, dir)
	xuid := r11XUID()

	emis, err := ScanFilmUnitEquipment(dir)
	if err != nil {
		t.Fatalf("%s : i26 illisible : %v", s.id, err)
	}
	t.Logf("%s : %d emissions i26 (canal cote PORTEUR)", s.id, len(emis))
	wanted := r11LogI26(t, s, emis, xuid)

	samples, st, err := ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("%s : etat ti=37 illisible : %v", s.id, err)
	}
	t.Logf("  ti=37 : records %d, masque∋charges %d, charges lues %d, marches interrompues %d, "+
		"slots %d", st.Records, st.WithField[EquipCharges], st.Read[EquipCharges],
		st.Broken, st.Slots)
	r11LogChargeHist(t, samples)
	r11LogWanted(t, s, samples, wanted)
	r11LogCreations(t, s, dir, wanted)
}

// r11LogCreations nomme les entites tenues par leur record de CREATION : le mot de 32 bits du
// bloc `object-multiplayer-properties` est le GlobalID du tag `eqip` de l'objet — son
// identite telle que le jeu la porte. C'est ce qui CONFIRME, ou non, que le handle vu dans
// i26 a l'instant de l'ancre est bien un repulseur.
func r11LogCreations(t *testing.T, s r11Setup, dir string, wanted map[uint32]bool) {
	t.Helper()
	if len(wanted) == 0 {
		return
	}
	wr := r8MapEntry(t, dir).Range()
	cre, _, err := ScanFilmEquipmentCreations(dir, &wr)
	if err != nil {
		t.Logf("  creations ti=37 illisibles : %v", err)
		return
	}
	t.Logf("  %d records de creation ti=37 ; identite des entites tenues :", len(cre))
	for _, c := range cre {
		if !wanted[c.Slot] {
			continue
		}
		id := "(non transmis)"
		if c.MPPPresent[MPPWord32] {
			id = fmt.Sprintf("0x%08x", uint32(c.MPPVal[MPPWord32]))
		}
		t.Logf("    %-7s entite %d/%d cree — GlobalID eqip %s", r9MMSS(s.ms(c.TimestampUS)),
			c.Slot, c.Gen, id)
	}
}

// r11LogI26 publie les emissions d'i26 du joueur suivi et rend les valeurs d'entree vues —
// ce sont les handles des objets qu'il a tenus.
func r11LogI26(t *testing.T, s r11Setup, emis []UnitEquipmentEmission, xuid string) map[uint32]bool {
	t.Helper()
	wanted := map[uint32]bool{}
	n := 0
	for _, e := range emis {
		ms := s.ms(e.TimestampUS)
		if !r11IsTarget(s.art, e.Slot, ms, xuid) {
			continue
		}
		n++
		var parts []string
		for _, en := range e.Read.Entries {
			parts = append(parts, fmt.Sprintf("val=%d/gen=%d/porte=%v", en.Val, en.Tail, en.Present))
			if en.Present {
				wanted[en.Val] = true
			}
		}
		t.Logf("    %-7s slot=%-4d i26 tete=%d entrees=%d %v",
			r9MMSS(ms), e.Slot, e.Read.Head, len(e.Read.Entries), parts)
	}
	t.Logf("  -> %d emissions i26 du joueur suivi, %d handles distincts", n, len(wanted))
	return wanted
}

// r11LogChargeHist publie l'histogramme des valeurs d'i27. UNE CHARGE D'EQUIPEMENT VAUT 0 A
// QUELQUES UNITES : si la masse des valeurs est ailleurs, le champ ne porte pas de charges
// (ou la marche atterrit a cote), et c'est ce que R9 par. 3.3 soupconnait.
func r11LogChargeHist(t *testing.T, samples []EquipmentStateSample) {
	t.Helper()
	hist := map[uint64]int{}
	small, total := 0, 0
	for _, sm := range samples {
		if !sm.Present[EquipCharges] {
			continue
		}
		v := sm.Val[EquipCharges]
		hist[v]++
		total++
		if v <= 8 {
			small++
		}
	}
	var vals []uint64
	for v := range hist {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(a, b int) bool { return hist[vals[a]] > hist[vals[b]] })
	var top []string
	for i, v := range vals {
		if i >= 12 {
			break
		}
		top = append(top, fmt.Sprintf("%d:%d", v, hist[v]))
	}
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(small) / float64(total)
	}
	t.Logf("  i27 charges-remaining : %d valeurs lues, %d distinctes, %d dans 0..8 (%.1f %%)",
		total, len(vals), small, pct)
	t.Logf("    valeurs les plus frequentes : %v", top)
}

// r11LogWanted publie la serie de charges des entites referencees par le joueur suivi.
func r11LogWanted(t *testing.T, s r11Setup, samples []EquipmentStateSample, wanted map[uint32]bool) {
	t.Helper()
	if len(wanted) == 0 {
		t.Logf("  aucun handle a suivre : i26 n'a rien transmis pour ce joueur")
		return
	}
	n := 0
	for _, sm := range samples {
		if !wanted[sm.Slot] {
			continue
		}
		var parts []string
		for f := 0; f < EquipmentFieldCount; f++ {
			if sm.Present[f] {
				parts = append(parts, fmt.Sprintf("%s=%d", EquipmentField(f), sm.Val[f]))
			}
		}
		if len(parts) == 0 {
			continue
		}
		n++
		t.Logf("    %-7s entite %d/%d : %v", r9MMSS(s.ms(sm.TimestampUS)), sm.Slot, sm.Gen, parts)
	}
	t.Logf("  -> %d records d'etat pour les %d entites tenues par le joueur suivi", n, len(wanted))
}
