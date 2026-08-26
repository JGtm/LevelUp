package replay

// zone_states_hill_designator_test.go — LE DESIGNATEUR DE COLLINE (lot C-ter volet 1), sur des
// enregistrements CONSTRUITS : les periodes sont FERMEES A LA BASCULE du tag 5, pas au sommet de
// la jauge ; la colline VIDE est visible ; un designateur non chaine ou sans proprietaire voisin
// n'est pas elu (repli par les rampes).
//
// LE TEST DISCRIMINANT : les rampes de la jauge sont posees loin des bascules du designateur. Si
// `active` revenait a la jauge, les bornes des intervalles seraient celles des rampes (96 -> 295,
// 296 -> 599) et non celles du designateur (50 -> 199, 200 -> 399, 400 -> 599) — le test echoue.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// zoneChainedReadAt fabrique une lecture dont le record CHAINE — la seule que le designateur lit.
func zoneChainedReadAt(slot uint32, frame, tag int, value uint64) filmdec.ManagedPropertyRead {
	r := zoneReadAt(slot, frame, tag, value)
	r.Chained = true
	return r
}

// hillDesignatorCase monte l'objet de mode : slot 40 = designateur (bascules a 200 et 400),
// slot 41 = proprietaire (premier contact a 50), slot 43 = jauge (rampes culminant a 100 dans la
// zone 1 et a 300 dans la zone 0). La derniere periode (400 -> 599) n'a AUCUNE rampe : c'est la
// colline vide, que seule la grappe sur toute la periode localise (zone 1).
func hillDesignatorCase(chained bool) (ZoneInput, zoneCtx) {
	mk := zoneReadAt
	if chained {
		mk = zoneChainedReadAt
	}
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, mk(40, 200, filmdec.ManagedPropertyTagStringID, 0x78F81557))
	reads = append(reads, mk(40, 400, filmdec.ManagedPropertyTagStringID, 0x8727C0FF))
	reads = append(reads, zoneReadAt(41, 50, filmdec.ManagedPropertyTagU32, zoneNeutralOwner))
	reads = append(reads, zoneReadAt(41, 120, filmdec.ManagedPropertyTagU32, 0))
	reads = append(reads, zoneRampAt(43, 100, 900_000)...)
	reads = append(reads, zoneRampAt(43, 300, 910_000)...)
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0)) // zone 1 pendant la premiere rampe
	}
	for f := 296; f <= 300; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0)) // zone 0 pendant la seconde
	}
	for f := 450; f <= 460; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0)) // zone 1 pendant la colline VIDE
	}
	return in, zoneTestCtx(nil, []Track{track("2533", pts...)})
}

// TestZoneStatesCollineDesignateurFermeLesPeriodesALaBascule — les intervalles suivent le
// designateur, la colline vide est publiee, la methode le dit.
func TestZoneStatesCollineDesignateurFermeLesPeriodesALaBascule(t *testing.T) {
	in, c := hillDesignatorCase(true)
	states, cov := buildZoneStates(in, c)
	if cov.Method != ZoneMethodDesignator {
		t.Fatalf("methode %q, attendu %q", cov.Method, ZoneMethodDesignator)
	}
	if cov.HillPeriods != 3 || cov.Unpaired != 0 || cov.Paired != 2 {
		t.Fatalf("periodes %d / non localisees %d / zones %d, attendu 3 / 0 / 2 : %+v",
			cov.HillPeriods, cov.Unpaired, cov.Paired, states)
	}
	// LES BORNES SONT CELLES DU DESIGNATEUR — c'est le sujet de ce cas, et il tient sur l'UNION
	// des intervalles d'une zone. Depuis le 2026-08-26 une periode se SUBDIVISE quand la colline
	// change de main (cf. hillStatesOf) : le proprietaire est teste a part
	// (zone_states_hill_owner_test.go), ici on verifie que le decoupage ne DEBORDE pas de la
	// periode et ne laisse pas de trou dedans.
	want := map[int][][2]int{0: {{200, 399}}, 1: {{50, 199}, {400, 599}}}
	for _, st := range states {
		got := fusionneSpansContigus(st.Spans)
		spans := want[st.ZoneRef]
		if len(got) != len(spans) {
			t.Fatalf("zone %d : %d periode(s) apres fusion, attendu %d : %+v", st.ZoneRef,
				len(got), len(spans), st.Spans)
		}
		for i, sp := range got {
			if sp[0] != spans[i][0] || sp[1] != spans[i][1] {
				t.Errorf("zone %d periode %d : [%d ; %d], attendu [%d ; %d] — les bornes doivent"+
					" etre celles du DESIGNATEUR, pas de la jauge", st.ZoneRef, i, sp[0], sp[1],
					spans[i][0], spans[i][1])
			}
		}
		for _, sp := range st.Spans {
			if !sp.Active {
				t.Errorf("zone %d : intervalle [%d ; %d] non marque ACTIF", st.ZoneRef, sp.T0, sp.T1)
			}
		}
	}
	// LA PROGRESSION NE SE DUPLIQUE PAS SUR UNE PERIODE SUBDIVISEE (cf. hillSpansOf) : le sommet
	// est celui de la PERIODE, il n'est la propriete d'aucun de ses morceaux. Seule la periode
	// [200 ; 399] sort d'un seul tenant ET porte une rampe.
	for _, st := range states {
		for _, sp := range st.Spans {
			veut := sp.T0 == 200
			if (sp.Progress != nil) != veut {
				t.Errorf("zone %d [%d ; %d] : progression %v, attendu presente=%v", st.ZoneRef,
					sp.T0, sp.T1, sp.Progress, veut)
			}
		}
	}
}

// fusionneSpansContigus rend les bornes des suites d'intervalles qui se touchent — la PERIODE
// telle que le designateur l'a fermee, quels que soient les changements de main dedans.
func fusionneSpansContigus(spans []ZoneSpan) [][2]int {
	var out [][2]int
	for _, sp := range spans {
		if n := len(out); n > 0 && out[n-1][1]+1 == sp.T0 {
			out[n-1][1] = sp.T1
			continue
		}
		out = append(out, [2]int{sp.T0, sp.T1})
	}
	return out
}

// TestZoneStatesCollineDesignateurNonChaineRetombeSurLesRampes — un tag 5 dont le record ne
// chaine pas est de la contamination : il n'est pas elu, et le repli par les rampes reprend la
// main (bornes des rampes, methode « positions »).
func TestZoneStatesCollineDesignateurNonChaineRetombeSurLesRampes(t *testing.T) {
	in, c := hillDesignatorCase(false)
	states, cov := buildZoneStates(in, c)
	if cov.Method != ZoneMethodPositions {
		t.Fatalf("methode %q, attendu %q (repli)", cov.Method, ZoneMethodPositions)
	}
	if len(states) == 0 {
		t.Fatalf("aucun etat publie par le repli")
	}
	for _, st := range states {
		for _, sp := range st.Spans {
			if sp.T0 == 200 || sp.T0 == 400 {
				t.Errorf("zone %d : intervalle ouvert a %d — une borne du designateur alors qu'il"+
					" n'est pas chaine", st.ZoneRef, sp.T0)
			}
		}
	}
}

// TestZoneStatesCollineDesignateurExigeUnProprietaireVoisin — trois string-ids sur trois slots
// consecutifs a la fin du match (le trio de fin de match des 4 films) ne font pas un
// designateur : aucun proprietaire ne parle sur leur slot suivant.
func TestZoneStatesCollineDesignateurExigeUnProprietaireVoisin(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	for i, v := range []uint64{0x6050ABD7, 0x3327C7DA, 0xF2F9EB27} {
		reads = append(reads, zoneChainedReadAt(uint32(60+i), 590, filmdec.ManagedPropertyTagStringID, v))
	}
	reads = append(reads, zoneRampAt(43, 100, 900_000)...)
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0))
	}
	_, cov := buildZoneStates(in, zoneTestCtx(nil, []Track{track("2533", pts...)}))
	if cov.Method != ZoneMethodPositions {
		t.Fatalf("methode %q, attendu %q : le trio n'est pas un designateur", cov.Method,
			ZoneMethodPositions)
	}
}
