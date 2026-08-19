package replay

// zone_states_owner_test.go — LE VOLET « QUI TIENT LA ZONE », sur des enregistrements
// CONSTRUITS : l'appariement des jauges, l'election du proprietaire (seuil, unicite), les
// intervalles de propriete et leur progression, le controle de la valeur du canal contre le
// roster.
//
// Les fabriques partagees (`zoneReadAt`, `zoneRampAt`, `bastionCase`, ...) vivent dans
// `zone_states_test.go` ; seules celles propres a ce volet sont posees ici.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

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
// DANS l'intervalle, sur l'echelle du JEU (0 = jauge au repos, 1 = pleine).
//
// L'ECHELLE EST CELLE DU JEU, ET C'EST UNE MESURE QUI L'IMPOSE (lot C-ter volet 3) : la plage
// declaree du deser ([-100, +100]) est cent fois plus large que ce que la jauge parcourt (elle vit
// sur [0, +1]), et l'excursion mesuree du match, adoptee un temps a la place, se faussait sur une
// seule emission aberrante sous zero (cf. gaugeProgressOf).
func TestZoneStatesProgressionEstLeSommetDeLaJauge(t *testing.T) {
	in, c := bastionCase()
	states, _ := buildZoneStates(in, c)
	spans := states[0].Spans
	// L'intervalle qui contient la rampe LA PLUS HAUTE du slot (0,95) publie ce sommet, sur
	// l'echelle du jeu.
	if want := gaugeProgressOf(gaugeQ(950)); spans[1].Progress == nil || *spans[1].Progress != want {
		t.Errorf("progression %v sur l'intervalle du sommet, attendu %v", spans[1].Progress, want)
	}
	// Celui de la rampe plus basse (0,9) reste en dessous, sans jamais sortir de [0, 1].
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
	in.Reads = append(in.Reads, zoneRampAt(30, 450, 700)...) // rampe loin de toute capture
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
