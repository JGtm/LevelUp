package replay

// zone_states_hill_test.go — LE VOLET COLLINE, sur des enregistrements CONSTRUITS : les periodes
// de garde lues dans la grappe des positions, l'invariant « une seule zone active a la fois »,
// le compte des rampes non localisees, et la garde `Hill` qui ferme le repli par les positions
// hors des modes a colline.
//
// Les fabriques partagees vivent dans `zone_states_test.go` ; `zoneGaugeSamplesAt` est propre a
// ce volet.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// TestZoneStatesCollineActivePeriodes : sans oracle nomme, la zone active se lit dans la GRAPPE
// des positions, et les intervalles sortent marques ACTIFS et sans proprietaire.
func TestZoneStatesCollineActivePeriodes(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900)...)
	reads = append(reads, zoneRampAt(40, 400, 900)...)
	in := zoneTestInput(reads)
	in.Roles = "strongholds_zone,extraction_zone"
	in.Hill = true // le mode du match est un mode a COLLINE : c'est ce qui ouvre le repli
	// Aucune capture nommee (KOTH n'en a pas) : la grappe seule parle. Les joueurs se tiennent
	// dans la zone 102 pendant la premiere montee, dans la 101 pendant la seconde.
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0))
	}
	for f := 396; f <= 400; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0))
	}
	c := zoneTestCtx(nil, []Track{track("2533", pts...)})
	states, cov := buildZoneStates(in, c)
	if cov.Method != ZoneMethodPositions {
		t.Fatalf("methode %q, attendu %q", cov.Method, ZoneMethodPositions)
	}
	if cov.HillPeriods != 2 || len(states) != 2 {
		t.Fatalf("%d periode(s) et %d zone(s), attendu 2 et 2 : %+v", cov.HillPeriods, len(states), states)
	}
	for _, s := range states {
		for _, sp := range s.Spans {
			if !sp.Active {
				t.Errorf("zone %d : intervalle [%d ; %d] non marque ACTIF", s.ZoneRef, sp.T0, sp.T1)
			}
			if sp.Owner != nil {
				t.Errorf("zone %d : proprietaire publie en mode a colline (%d)", s.ZoneRef, *sp.Owner)
			}
		}
	}
}

// zoneGaugeSamplesAt fabrique une rampe de jauge SUR MESURE : trois emissions croissantes aux
// frames demandees. `zoneRampAt` en pose une de 4 frames ; ici la duree est le sujet du test.
func zoneGaugeSamplesAt(slot uint32, t0, tMid, tPeak int, topMilli uint64) []filmdec.ManagedPropertyRead {
	return []filmdec.ManagedPropertyRead{
		zoneReadAt(slot, t0, filmdec.ManagedPropertyTagQuant, gaugeQ(1)),
		zoneReadAt(slot, tMid, filmdec.ManagedPropertyTagQuant, gaugeQ(200)),
		zoneReadAt(slot, tPeak, filmdec.ManagedPropertyTagQuant, gaugeQ(topMilli)),
	}
}

// TestZoneStatesCollineUneSeuleZoneActiveALaFois — L'INVARIANT DU MODE (revue R1). Deux gardes
// qui se RECOUVRENT dans le temps, sur deux slots et deux zones, ne peuvent pas laisser deux
// zones marquees `active` au meme instant : la precedente est fermee a l'instant ou la suivante
// commence.
//
// SANS LA FERMETURE SYSTEMATIQUE, la periode precedente n'etait tronquee que si un TROU la
// separait de la suivante — deux recouvrantes sortaient donc toutes les deux actives, et le
// rendu montrait deux collines.
func TestZoneStatesCollineUneSeuleZoneActiveALaFois(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneGaugeSamplesAt(40, 100, 150, 200, 900)...) // garde longue
	reads = append(reads, zoneGaugeSamplesAt(50, 150, 170, 190, 910)...) // garde DEDANS
	in := zoneTestInput(reads)
	in.Hill = true
	// La grappe est dans la zone 1 sauf entre 150 et 190, ou elle passe dans la zone 0 : la
	// premiere rampe designe la zone 1, la seconde la zone 0, et les deux se recouvrent.
	var pts []Point
	for f := 100; f <= 200; f++ {
		x := float32(20.5)
		if f >= 150 && f <= 190 {
			x = -19.5
		}
		pts = append(pts, pointAt(f, x, 0, 0))
	}
	states, cov := buildZoneStates(in, zoneTestCtx(nil, []Track{track("2533", pts...)}))
	if cov.HillPeriods != 2 || len(states) != 2 {
		t.Fatalf("%d periode(s) et %d zone(s), attendu 2 et 2 : %+v", cov.HillPeriods,
			len(states), states)
	}
	type span struct {
		ref    int
		t0, t1 int
	}
	var actifs []span
	for _, st := range states {
		for _, sp := range st.Spans {
			if sp.Active {
				actifs = append(actifs, span{ref: st.ZoneRef, t0: sp.T0, t1: sp.T1})
			}
		}
	}
	for i := range actifs {
		for j := i + 1; j < len(actifs); j++ {
			a, b := actifs[i], actifs[j]
			if a.t0 <= b.t1 && b.t0 <= a.t1 {
				t.Errorf("deux zones ACTIVES au meme instant : zone %d [%d ; %d] et zone %d"+
					" [%d ; %d]", a.ref, a.t0, a.t1, b.ref, b.t0, b.t1)
			}
		}
	}
}

// TestZoneStatesCollineCompteLesRampesNonLocalisees — LE DENOMINATEUR DE LA METHODE PAR
// POSITIONS (revue R1). Une montee de jauge que la grappe ne sait pas localiser est une garde
// REELLE dont on ignore le lieu : elle est ecartee, et elle se compte dans `unpaired`.
//
// SANS CE COMPTE, `unpaired` restait a zero quoi qu'il arrive en methode `positions+geometry` :
// un appariement partiel se lisait exactement comme un appariement complet.
func TestZoneStatesCollineCompteLesRampesNonLocalisees(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900)...) // gardee DANS la zone 1
	reads = append(reads, zoneRampAt(40, 400, 910)...) // gardee loin de toute zone
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0))
	}
	for f := 396; f <= 400; f++ {
		pts = append(pts, pointAt(f, 500, 500, 0))
	}
	states, cov := buildZoneStates(in, zoneTestCtx(nil, []Track{track("2533", pts...)}))
	if cov.Unpaired != 1 {
		t.Errorf("rampes non localisees %d, attendu 1", cov.Unpaired)
	}
	if cov.HillPeriods != 1 || cov.Paired != 1 {
		t.Errorf("periodes %d / zones appariees %d, attendu 1 et 1", cov.HillPeriods, cov.Paired)
	}
	if len(states) != 1 || states[0].ZoneRef != 1 {
		t.Errorf("%d zone(s) publiee(s) : %+v — seule la garde localisee sort", len(states), states)
	}
}

// TestZoneStatesCollineSansGrappeNePublieRien : une montee de jauge sans position pour la
// localiser ne pose aucune colline — on refuse plutot que de choisir la zone la plus proche.
func TestZoneStatesCollineSansGrappeNePublieRien(t *testing.T) {
	in := zoneTestInput(zoneRampAt(40, 100, 900))
	in.Hill = true
	c := zoneTestCtx(nil, []Track{track("2533", pointAt(100, 500, 500, 0))})
	states, cov := buildZoneStates(in, c)
	if len(states) != 0 || cov.HillPeriods != 0 {
		t.Errorf("%d etat(s) et %d periode(s) publies sans grappe", len(states), cov.HillPeriods)
	}
}

// TestZoneStatesHorsCollineNeReplieJamaisSurLesPositions — LE VERROU DE LA REVUE R1 : un mode
// SANS capture de zone joue sur une carte qui en DECLARE (un CTF sur une carte a zones de
// livraison : 18 cartes du catalogue) ne doit publier AUCUN intervalle actif.
//
// SANS LA GARDE, LE REPLI S'OUVRAIT SUR L'ABSENCE DE CAPTURE — c'est-a-dire sur le cas nominal
// de tous les modes qui n'en ont pas — et posait des collines sur des zones de livraison. Les
// memes lectures, avec `Hill` a vrai, publient bien des periodes : c'est le mode qui tranche, pas
// le silence de l'oracle.
func TestZoneStatesHorsCollineNeReplieJamaisSurLesPositions(t *testing.T) {
	reads := zoneRampAt(40, 100, 900)
	pts := make([]Point, 0, 5)
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0))
	}
	c := zoneTestCtx(nil, []Track{track("2533", pts...)}) // aucune capture nommee : un CTF

	ctf := zoneTestInput(reads) // Hill reste FAUX : le mode n'est pas un mode a colline
	states, cov := buildZoneStates(ctf, c)
	if len(states) != 0 {
		t.Errorf("%d etat(s) publie(s) hors mode a colline : %+v", len(states), states)
	}
	if cov == nil {
		t.Fatalf("aucune couverture publiee : le silence doit rester explicite")
	}
	if cov.Catalog != 2 || cov.HillPeriods != 0 || cov.Spans != 0 {
		t.Errorf("couverture %+v : attendu catalogue 2, 0 periode, 0 intervalle", cov)
	}
	if cov.Method != ZoneMethodCaptures {
		t.Errorf("methode %q, attendu %q — la methode par positions n'a pas ete tentee",
			cov.Method, ZoneMethodCaptures)
	}

	koth := zoneTestInput(reads)
	koth.Hill = true
	kothStates, kothCov := buildZoneStates(koth, c)
	if len(kothStates) == 0 || kothCov.HillPeriods == 0 {
		t.Fatalf("le MEME film en mode a colline ne publie rien (%d etats, %d periodes) :"+
			" la garde doit trancher sur le mode, pas sur la lecture",
			len(kothStates), kothCov.HillPeriods)
	}
	for _, s := range kothStates {
		for _, sp := range s.Spans {
			if !sp.Active {
				t.Errorf("zone %d : intervalle [%d ; %d] non ACTIF en mode a colline",
					s.ZoneRef, sp.T0, sp.T1)
			}
		}
	}
}
