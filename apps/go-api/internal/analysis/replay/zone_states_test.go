package replay

// zone_states_test.go — LE CALQUE DE L'ETAT DES ZONES, sur des enregistrements CONSTRUITS.
//
// AUCUN FILM ICI, ET C'EST LE POINT : la regle d'appariement, la lecture du proprietaire et les
// degradations se testent sur des lectures fabriquees, donc a la milliseconde pres et sans
// dependre d'un fichier de 300 Mo. Les temoins reels (deux Bastion, un KOTH) sont cuits a part
// et publies au journal du lot ; ce fichier verrouille la REGLE.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// zoneReadAt fabrique une lecture scalaire de `ti=13` posee sur une frame de la grille.
func zoneReadAt(slot uint32, frame, tag int, value uint64) filmdec.ManagedPropertyRead {
	return filmdec.ManagedPropertyRead{
		Slot: slot, TimestampUS: uint64(frame) * 100_000, Field: filmdec.ManagedPropertyScalar,
		PlayerIndex: -1, Tag: tag, Value: value, HasValue: true,
	}
}

// zoneRampAt fabrique une rampe de jauge culminant a `peak` : trois emissions croissantes et une
// amplitude tres au-dessus du seuil (4 096 quanta).
func zoneRampAt(slot uint32, peak int, top uint64) []filmdec.ManagedPropertyRead {
	return []filmdec.ManagedPropertyRead{
		zoneReadAt(slot, peak-4, filmdec.ManagedPropertyTagQuant, 1_000),
		zoneReadAt(slot, peak-2, filmdec.ManagedPropertyTagQuant, 200_000),
		zoneReadAt(slot, peak, filmdec.ManagedPropertyTagQuant, top),
	}
}

// zoneTestCtx est l'axe de temps commun aux cas : grille de 100 ms, 600 frames.
func zoneTestCtx(actions []ObjectiveAction, tracks []Track) zoneCtx {
	return zoneCtx{origin: 0, step: 100_000, frames: 600, intervalMS: 100,
		tracks: tracks, actions: actions, matchID: "test"}
}

// zoneTestInput monte l'entree d'un cas a deux zones.
func zoneTestInput(reads []filmdec.ManagedPropertyRead) ZoneInput {
	return ZoneInput{
		Scanned:    true,
		Reads:      reads,
		Zones:      []Zone{zoneAt(0, 101, -20, 0, 0), zoneAt(1, 102, 20, 0, 0)},
		Roles:      "strongholds_zone",
		TeamByXUID: map[string]int{"2533": 0, "2535": 1},
	}
}

// bastionCase monte le cas nominal : deux zones, deux jauges, deux canaux de proprietaire,
// QUATRE captures nommees posees DANS leur zone — deux par zone.
//
// DEUX CAPTURES PAR ZONE, ET C'EST LE SEUIL QUI L'EXIGE (revue R1) : un canal n'est elu
// proprietaire qu'a partir de deux concordances avec le roster (zoneOwnerMinAgreements). Une
// zone prise une seule fois dans tout le match n'a pas de proprietaire publie — c'est la regle,
// et le cas nominal doit la franchir plutot que de vivre dessous.
func bastionCase() (ZoneInput, zoneCtx) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(10, 100, 900_000)...) // zone 101, prise par l'equipe 0
	reads = append(reads, zoneRampAt(10, 300, 950_000)...) // zone 101, reprise par l'equipe 1
	reads = append(reads, zoneRampAt(20, 200, 800_000)...) // zone 102, prise par l'equipe 1
	reads = append(reads, zoneRampAt(20, 400, 820_000)...) // zone 102, reprise par l'equipe 0
	reads = append(reads,
		zoneReadAt(11, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(11, 101, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(11, 301, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(21, 201, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 401, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(10, 5, filmdec.ManagedPropertyTagStringID, 0x67F43AC3),
	)
	actions := []ObjectiveAction{action("2533", 100), action("2535", 200), action("2535", 300),
		action("2533", 400)}
	tracks := []Track{
		track("2533", pointAt(100, -19.5, 0, 0), pointAt(400, 20.5, 0, 0)),
		track("2535", pointAt(200, 20.5, 0, 0), pointAt(300, -19.5, 0, 0)),
	}
	return zoneTestInput(reads), zoneTestCtx(actions, tracks)
}

// TestZoneStatesPublieUnEtatParZoneAppariee est le cas nominal : chaque zone que les captures
// rattachent a un slot de jauge sort avec ses intervalles de propriete.
func TestZoneStatesPublieUnEtatParZoneAppariee(t *testing.T) {
	in, c := bastionCase()
	states, cov := buildZoneStates(in, c)
	if len(states) != 2 {
		t.Fatalf("%d zone(s) publiee(s), attendu 2 : %+v", len(states), states)
	}
	if states[0].ZoneRef != 0 || states[1].ZoneRef != 1 {
		t.Errorf("zoneRef publies %d et %d, attendu 0 et 1 — l'index doit etre celui du catalogue",
			states[0].ZoneRef, states[1].ZoneRef)
	}
	if states[0].Key != 0x67F43AC3 {
		t.Errorf("cle du slot de jauge %#x, attendu 0x67F43AC3", states[0].Key)
	}
	if cov.Method != ZoneMethodCaptures {
		t.Errorf("methode %q, attendu %q", cov.Method, ZoneMethodCaptures)
	}
	if cov.Paired != 2 || cov.Unpaired != 0 {
		t.Errorf("apparies %d / non apparies %d, attendu 2 / 0", cov.Paired, cov.Unpaired)
	}
	if cov.Captures != 4 || cov.Attributed != 4 {
		t.Errorf("captures %d, attribuees %d — attendu 4 et 4", cov.Captures, cov.Attributed)
	}
	if cov.OwnerUnpaired != 0 {
		t.Errorf("zones sans proprietaire elu %d, attendu 0", cov.OwnerUnpaired)
	}
}

// TestZoneStatesIntervallesSuiventLesBascules : le proprietaire change QUAND le canal change, et
// l'intervalle court jusqu'a la bascule suivante — le dernier jusqu'a la fin de l'axe.
func TestZoneStatesIntervallesSuiventLesBascules(t *testing.T) {
	in, c := bastionCase()
	states, _ := buildZoneStates(in, c)
	spans := states[0].Spans
	if len(spans) != 3 {
		t.Fatalf("%d intervalle(s) sur la zone 0, attendu 3 : %+v", len(spans), spans)
	}
	want := []struct {
		t0, t1 int
		owner  *int
	}{{0, 100, nil}, {101, 300, intPtr(0)}, {301, 599, intPtr(1)}}
	for i, w := range want {
		got := spans[i]
		if got.T0 != w.t0 || got.T1 != w.t1 {
			t.Errorf("intervalle %d : [%d ; %d], attendu [%d ; %d]", i, got.T0, got.T1, w.t0, w.t1)
		}
		switch {
		case w.owner == nil && got.Owner != nil:
			t.Errorf("intervalle %d : proprietaire %d, attendu « personne »", i, *got.Owner)
		case w.owner != nil && (got.Owner == nil || *got.Owner != *w.owner):
			t.Errorf("intervalle %d : proprietaire %v, attendu %d", i, got.Owner, *w.owner)
		}
		if got.Active {
			t.Errorf("intervalle %d marque ACTIF : reserve aux modes a colline", i)
		}
	}
}

// TestZoneStatesProgressionEstLeSommetDeLaJauge : la progression publiee est le sommet atteint
// DANS l'intervalle, ramene sur l'EXCURSION MESUREE de la jauge de cette zone.
//
// L'ECHELLE EST CELLE DU MATCH, ET C'EST UNE MESURE QUI L'IMPOSE : la plage declaree du deser
// ([-100, +100]) est mille fois plus large que ce que la jauge parcourt reellement — ramenee
// dessus, toute valeur vaut 0,50, soit un arc a moitie plein en permanence.
func TestZoneStatesProgressionEstLeSommetDeLaJauge(t *testing.T) {
	in, c := bastionCase()
	states, _ := buildZoneStates(in, c)
	spans := states[0].Spans
	// L'intervalle qui contient la rampe LA PLUS HAUTE du slot (950 000) touche le sommet de
	// l'excursion : sa progression vaut exactement 1.
	if spans[1].Progress == nil || *spans[1].Progress != 1 {
		t.Errorf("progression %v sur l'intervalle du sommet, attendu 1", spans[1].Progress)
	}
	// Celui de la rampe plus basse (900 000) reste en dessous, sans jamais sortir de [0, 1].
	if spans[0].Progress == nil {
		t.Fatalf("aucune progression sur l'intervalle qui contient la premiere rampe")
	}
	if p := *spans[0].Progress; p <= 0 || p >= 1 {
		t.Errorf("progression %v de la rampe basse, attendue dans ]0, 1[", p)
	}
	// Un intervalle sans aucune emission de jauge n'a pas de progression a montrer.
	if spans[2].Progress != nil {
		t.Errorf("progression %v sur un intervalle sans emission de jauge", *spans[2].Progress)
	}
}

// TestZoneStatesProgressionSansExcursionEstAbsente : une jauge qui ne bouge pas n'a pas de
// progression a montrer — publier 0 ou 1 affirmerait une capture qui n'a pas eu lieu.
func TestZoneStatesProgressionSansExcursionEstAbsente(t *testing.T) {
	var plate zoneGauge
	if p := plate.progressOf(42); p != nil {
		t.Errorf("progression %v sur une jauge jamais vue, attendu absente", *p)
	}
	fige := zoneGaugeOf([]zoneSample{{t: 0, v: 7}, {t: 1, v: 7}})
	if p := fige.progressOf(7); p != nil {
		t.Errorf("progression %v sur une jauge plate, attendu absente", *p)
	}
}

// TestZoneStatesControleDuProprietaire : la valeur du canal juste apres une capture est
// confrontee a l'equipe du capteur, et les deux comptes sont publies.
func TestZoneStatesControleDuProprietaire(t *testing.T) {
	in, c := bastionCase()
	_, cov := buildZoneStates(in, c)
	if cov.OwnerChecked != 4 || cov.OwnerAgreed != 4 {
		t.Errorf("controle du proprietaire %d/%d, attendu 4/4", cov.OwnerAgreed, cov.OwnerChecked)
	}
}

// TestZoneStatesCanalConfirmeUneSeuleFoisNEstPasElu — LE SEUIL D'ACCORD (revue R1). Un canal
// qui ne concorde qu'UNE fois avec le roster n'est pas elu : la zone n'a pas de proprietaire
// publie, elle n'entre pas dans `zoneStates`, et `coverage.zones.ownerUnpaired` le dit.
//
// SANS LE SEUIL, une seule coincidence suffisait : le canal elu teintait alors la zone sur toute
// la duree du match, ce qui a exactement l'apparence d'une mesure.
func TestZoneStatesCanalConfirmeUneSeuleFoisNEstPasElu(t *testing.T) {
	in, c := bastionCase()
	// La reprise de la zone 1 disparait : il ne reste qu'UNE capture concordante sur son canal.
	in.Reads = zoneReadsWithout(in.Reads, 21, 401)
	c.actions = c.actions[:3]
	states, cov := buildZoneStates(in, c)
	if len(states) != 1 || states[0].ZoneRef != 0 {
		t.Fatalf("%d zone(s) publiee(s) : %+v — seule la zone 0 a deux concordances", len(states), states)
	}
	if cov.Paired != 2 {
		t.Errorf("jauges appariees %d, attendu 2 — la zone 1 reste appariee par sa jauge", cov.Paired)
	}
	if cov.OwnerUnpaired != 1 {
		t.Errorf("zones sans proprietaire elu %d, attendu 1", cov.OwnerUnpaired)
	}
}

// TestZoneStatesUnCanalNEstProprietaireQueDUneZone — L'UNICITE (revue R1). Un canal qui est
// l'argmax de DEUX zones n'en garde qu'une : celle du plus grand accord. L'autre reste sans
// proprietaire plutot que de recevoir les MEMES intervalles.
func TestZoneStatesUnCanalNEstProprietaireQueDUneZone(t *testing.T) {
	in, c := bastionCase()
	// Le canal de la zone 1 disparait : le canal de la zone 0 (slot 11) devient l'argmax des
	// deux zones — il concorde deux fois sur la zone 0 et deux fois sur la zone 1.
	in.Reads = zoneReadsWithoutSlot(in.Reads, 21)
	in.Reads = append(in.Reads,
		zoneReadAt(11, 201, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(11, 401, filmdec.ManagedPropertyTagU32, 0),
	)
	states, cov := buildZoneStates(in, c)
	if len(states) != 1 {
		t.Fatalf("%d zone(s) publiee(s), attendu 1 : un canal ne tient qu'une zone — %+v",
			len(states), states)
	}
	if cov.OwnerUnpaired != 1 {
		t.Errorf("zones sans proprietaire elu %d, attendu 1", cov.OwnerUnpaired)
	}
}

// zoneReadsWithout retire la lecture d'un slot posee sur une frame donnee.
func zoneReadsWithout(reads []filmdec.ManagedPropertyRead, slot uint32,
	frame int,
) []filmdec.ManagedPropertyRead {
	out := make([]filmdec.ManagedPropertyRead, 0, len(reads))
	for _, r := range reads {
		if r.Slot == slot && r.TimestampUS == uint64(frame)*100_000 {
			continue
		}
		out = append(out, r)
	}
	return out
}

// zoneReadsWithoutSlot retire toutes les lectures d'un slot.
func zoneReadsWithoutSlot(reads []filmdec.ManagedPropertyRead,
	slot uint32,
) []filmdec.ManagedPropertyRead {
	out := make([]filmdec.ManagedPropertyRead, 0, len(reads))
	for _, r := range reads {
		if r.Slot == slot {
			continue
		}
		out = append(out, r)
	}
	return out
}

// TestZoneStatesSlotNonApparieNEstPasPublie : un slot de jauge qu'aucune capture ne rattache
// n'invente pas de zone — il se compte, et rien de plus.
func TestZoneStatesSlotNonApparieNEstPasPublie(t *testing.T) {
	in, c := bastionCase()
	in.Reads = append(in.Reads, zoneRampAt(30, 450, 700_000)...) // rampe loin de toute capture
	states, cov := buildZoneStates(in, c)
	if len(states) != 2 {
		t.Errorf("%d zone(s) publiee(s), attendu 2 — le slot orphelin ne doit rien publier",
			len(states))
	}
	if cov.Unpaired != 1 {
		t.Errorf("slots non apparies %d, attendu 1", cov.Unpaired)
	}
}

// TestZoneStatesValeurInconnueNOuvreAucunIntervalle : une valeur de canal qui n'est pas un camp
// du roster ne devient PAS un proprietaire, et le refus se compte.
func TestZoneStatesValeurInconnueNOuvreAucunIntervalle(t *testing.T) {
	in, c := bastionCase()
	in.Reads = append(in.Reads, zoneReadAt(11, 400, filmdec.ManagedPropertyTagU32, 7))
	states, cov := buildZoneStates(in, c)
	if cov.UnknownOwner != 1 {
		t.Fatalf("valeurs inconnues %d, attendu 1", cov.UnknownOwner)
	}
	for _, s := range states[0].Spans {
		if s.Owner != nil && *s.Owner == 7 {
			t.Errorf("l'equipe 7 a ete publiee alors qu'aucun joueur ne l'occupe")
		}
	}
}

// TestZoneStatesSansRosterAccepteLesDeuxCampsMesures : hors ligne (aucun fait de match), seules
// les deux valeurs mesurees du canal sont acceptees comme camps.
func TestZoneStatesSansRosterAccepteLesDeuxCampsMesures(t *testing.T) {
	in, c := bastionCase()
	in.TeamByXUID = nil
	states, cov := buildZoneStates(in, c)
	if len(states) == 0 {
		t.Fatalf("aucun etat publie sans roster : les camps 0 et 1 restent lisibles")
	}
	if cov.OwnerChecked != 0 {
		t.Errorf("controle du proprietaire %d, attendu 0 sans roster", cov.OwnerChecked)
	}
}

// TestZoneStatesSansCatalogueNePublieRien : sans zone de carte, aucun intervalle n'a de cible.
// La couverture, elle, est publiee — c'est elle qui distingue les silences.
func TestZoneStatesSansCatalogueNePublieRien(t *testing.T) {
	in, c := bastionCase()
	in.Zones = nil
	states, cov := buildZoneStates(in, c)
	if states != nil {
		t.Errorf("%d etat(s) publie(s) sans catalogue de zones", len(states))
	}
	if cov == nil || cov.Catalog != 0 {
		t.Fatalf("couverture attendue avec un catalogue vide, obtenu %+v", cov)
	}
}

// TestZoneStatesNonBalayeNePublieAucuneCouverture : « rien n'a ete lu » et « rien n'existait »
// ne sont pas la meme chose, et l'absence de couverture porte la premiere.
func TestZoneStatesNonBalayeNePublieAucuneCouverture(t *testing.T) {
	in, c := bastionCase()
	in.Scanned = false
	states, cov := buildZoneStates(in, c)
	if states != nil || cov != nil {
		t.Errorf("balayage absent : attendu (nil, nil), obtenu (%d etats, %+v)", len(states), cov)
	}
}

// TestZoneStatesCollineActivePeriodes : sans oracle nomme, la zone active se lit dans la GRAPPE
// des positions, et les intervalles sortent marques ACTIFS et sans proprietaire.
func TestZoneStatesCollineActivePeriodes(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900_000)...)
	reads = append(reads, zoneRampAt(40, 400, 900_000)...)
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

// TestZoneStatesCollineSansGrappeNePublieRien : une montee de jauge sans position pour la
// localiser ne pose aucune colline — on refuse plutot que de choisir la zone la plus proche.
func TestZoneStatesCollineSansGrappeNePublieRien(t *testing.T) {
	in := zoneTestInput(zoneRampAt(40, 100, 900_000))
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
	reads := zoneRampAt(40, 100, 900_000)
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

func intPtr(v int) *int { return &v }

// TestZoneStatesTientLeVolumeDUnVraiFilm — LE COUT DU CALQUE, sur les ordres de grandeur
// MESURES d'un vrai match de Bastion (`7344d24f`, 2026-08-18) : 26 slots dont 5 de jauge a
// ~1 000 emissions et 10 de propriete, 246 captures nommees, 8 joueurs, ~5 000 frames.
//
// POURQUOI CE TEST EXISTE. Le calque tourne DANS l'assemblage d'artefact, derriere une dizaine
// de balayages de film : une regle quadratique y serait invisible en unitaire et couterait des
// minutes en production. Le volume est donc reproduit ici, sans film — si une boucle devient
// quadratique, ce test ne rend plus la main.
func TestZoneStatesTientLeVolumeDUnVraiFilm(t *testing.T) {
	const frames, captures = 5000, 246
	var reads []filmdec.ManagedPropertyRead
	for slot := uint32(10); slot < 15; slot++ { // 5 slots de jauge, ~1 000 emissions chacun
		for i := 0; i < 330; i++ {
			reads = append(reads, zoneRampAt(slot, 8+i*15, uint64(500_000+i))...)
		}
	}
	for slot := uint32(30); slot < 40; slot++ { // 10 slots de propriete
		for i := 0; i < 40; i++ {
			v := uint64(i % 2)
			if i%7 == 0 {
				v = zoneNeutralOwner
			}
			reads = append(reads, zoneReadAt(slot, 20+i*120, filmdec.ManagedPropertyTagU32, v))
		}
	}
	in := zoneTestInput(reads)
	actions := make([]ObjectiveAction, 0, captures)
	pts := make([]Point, 0, frames)
	for f := 0; f < frames; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0))
	}
	for i := 0; i < captures; i++ {
		actions = append(actions, action("2533", 8+i*20))
	}
	c := zoneCtx{origin: 0, step: 100_000, frames: frames, intervalMS: 100,
		tracks: []Track{track("2533", pts...)}, actions: actions, matchID: "volume"}
	states, cov := buildZoneStates(in, c)
	if cov == nil {
		t.Fatalf("aucune couverture rendue sur le cas de volume")
	}
	t.Logf("volume : %d lectures, %d slots, %d captures -> %d zone(s), %d intervalle(s)",
		len(reads), cov.Slots, cov.Captures, len(states), cov.Spans)
}
