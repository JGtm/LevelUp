package replaydiff

// empreinte_durees_test.go — LA MUTATION QUE L'AXE « DUREES » EXISTE POUR ATTRAPER : un
// intervalle ROGNE, meme d'une seule frame, sur un calque dont le NOMBRE d'elements ne bouge
// pas. C'est exactement le defaut mesure sur l'Oddball du 2026-09-06 (rognage de fenetre de
// grappin, 91,2 s de duree perdue contre 32,6 s pour l'equivalent en rejets purs) : le
// comparateur d'AVANT cet axe ne voyait que les rejets (le compte baisse), jamais le rognage.

import (
	"strings"
	"testing"
)

// TestIntervalleRogneEstUnePerte — LE TEST DE MUTATION DEMANDE : `t1` recule d'UNE frame sur
// un span, le NOMBRE d'elements du calque reste identique (3 -> 3). Sans cet axe, aucun
// ecart ne serait rapporte ; avec lui, la somme des durees baisse et c'est une perte.
func TestIntervalleRogneEstUnePerte(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":38,"matchId":"a","grappleLines":[
		{"t0":100,"t1":149,"slot":7},
		{"t0":200,"t1":209,"slot":7},
		{"t0":300,"t1":319,"slot":9}]}`)
	// Le premier intervalle perd sa DERNIERE frame (149 -> 148) : 1 frame sur 50 + 10 + 20 = 80.
	rogne := doc(t, `{"schemaVersion":39,"matchId":"a","grappleLines":[
		{"t0":100,"t1":148,"slot":7},
		{"t0":200,"t1":209,"slot":7},
		{"t0":300,"t1":319,"slot":9}]}`)

	// Le compte d'elements NE BOUGE PAS : c'est precisement le cas que le comptage seul rate.
	rCompte := Comparer(Empreindre(ancien), Empreindre(rogne))
	if got := sensDe(rCompte, "equipement", "grappleLines/n"); got != "" {
		t.Fatalf("le nombre d'elements ne doit pas bouger (3 -> 3), sens %q", got)
	}

	if got := sensDe(rCompte, "equipement", "grappleLines/duree-totale"); got != SensPerte {
		t.Fatalf("un intervalle rogne de 1 frame doit etre une PERTE sur la duree totale, sens %q (rapport : %+v)",
			got, rCompte.Differences)
	}
	// La ventilation par slot doit voir la MEME perte, sur le slot touche seulement.
	if got := sensDe(rCompte, "equipement", "grappleLines/duree-totale/par-slot/7"); got != SensPerte {
		t.Fatalf("le slot rogne (7) doit porter la perte, sens %q", got)
	}
	if got := sensDe(rCompte, "equipement", "grappleLines/duree-totale/par-slot/9"); got != "" {
		t.Fatalf("le slot non touche (9) ne doit porter aucun ecart, sens %q", got)
	}

	// Verification chiffree : 80 - 1 = 79.
	empA, empB := Empreindre(ancien), Empreindre(rogne)
	ma, oka := empA.Mesures["equipement/grappleLines/duree-totale"]
	mb, okb := empB.Mesures["equipement/grappleLines/duree-totale"]
	if !oka || !okb || ma.Num != 80 || mb.Num != 79 {
		t.Fatalf("durees totales attendues 80 -> 79, obtenu %v(%v) -> %v(%v)", ma.Num, oka, mb.Num, okb)
	}
}

// TestDureeIdentiqueQuandRienNeBouge — le cas neutre : deux documents identiques ne portent
// aucun ecart sur l'axe des durees.
func TestDureeIdentiqueQuandRienNeBouge(t *testing.T) {
	texte := `{"schemaVersion":39,"matchId":"a","flagCarries":[
		{"team":0,"spans":[{"t0":10,"t1":19,"xuid":"111"}]}]}`
	r := Comparer(Empreindre(doc(t, texte)), Empreindre(doc(t, texte)))
	if got := sensDe(r, "ports", "flagCarries.spans/duree-totale"); got != "" {
		t.Fatalf("deux documents identiques ne doivent porter aucun ecart de duree, sens %q (rapport : %+v)",
			got, r.Differences)
	}
}

// TestDureeVentileeParXuidQuandLeSpanEnPorteUn — FlagSpan/BombCarry/SkullCarry/VipPeriod
// portent un xuid DIRECT : la ventilation individuelle doit suivre cette cle-la, pas un slot.
func TestDureeVentileeParXuidQuandLeSpanEnPorteUn(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":38,"matchId":"a","bombCarries":[
		{"t0":0,"t1":9,"xuid":"555"}]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"a","bombCarries":[
		{"t0":0,"t1":4,"xuid":"555"}]}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "ports", "bombCarries/duree-totale/par-xuid/555"); got != SensPerte {
		t.Fatalf("la ventilation par xuid doit voir la perte (10 -> 5 frames), sens %q (rapport : %+v)",
			got, r.Differences)
	}
	// Aucune ventilation par slot ne doit apparaitre : le span ne porte pas ce champ.
	empB := Empreindre(nouveau)
	for k := range empB.Mesures {
		if k == "ports/bombCarries/duree-totale/par-slot/0" {
			t.Fatalf("un span avec xuid ne doit pas ventiler par slot : %s", k)
		}
	}
}

// TestSpanSansIdentiteNeVentilePasIndividuellement — ZoneSpan ne porte ni xuid ni slot (son
// identite est `owner`, une EQUIPE) : seule la somme du calque doit exister, aucune clef
// individuelle fantome.
func TestSpanSansIdentiteNeVentilePasIndividuellement(t *testing.T) {
	texte := `{"schemaVersion":39,"matchId":"a","zoneStates":[
		{"zoneRef":0,"spans":[{"t0":0,"t1":9,"owner":1,"active":false}]}]}`
	e := Empreindre(doc(t, texte))
	if _, ok := e.Mesures["objets-objectif/zoneStates.spans/duree-totale"]; !ok {
		t.Fatalf("la somme du calque doit exister meme sans identite individuelle : %+v", e.Mesures)
	}
	const prefixeIndividuel = "objets-objectif/zoneStates.spans/duree-totale/par-"
	for k := range e.Mesures {
		if strings.HasPrefix(k, prefixeIndividuel) {
			t.Fatalf("un span sans xuid ni slot ne doit ventiler aucune identite individuelle : %s", k)
		}
	}
}

// TestDureeT1Inclus — la convention documentee sur chaque span du document (« T1 est
// INCLUS ») : [10,19] dure 10 frames, pas 9.
func TestDureeT1Inclus(t *testing.T) {
	e := Empreindre(doc(t, `{"schemaVersion":39,"matchId":"a","grappleLines":[{"t0":10,"t1":19,"slot":1}]}`))
	m, ok := e.Mesures["equipement/grappleLines/duree-totale"]
	if !ok || m.Num != 10 {
		t.Fatalf("duree de [10,19] attendue 10 (T1 inclus), obtenu %v (present=%v)", m.Num, ok)
	}
}

// TestObjetSansT0T1NEstPasUnSpan — un objet qui n'a pas les DEUX champs numeriques ne doit
// jamais etre compte comme une duree, meme s'il porte un `t1` isole ou un `t0` textuel.
func TestObjetSansT0T1NEstPasUnSpan(t *testing.T) {
	cas := []string{
		`{"schemaVersion":39,"matchId":"a","autresTruc":[{"t1":19,"slot":1}]}`, // t0 absent
		`{"schemaVersion":39,"matchId":"a","autresTruc":[{"t0":10,"slot":1}]}`, // t1 absent
		`{"schemaVersion":39,"matchId":"a","autresTruc":[{"t0":"x","t1":19}]}`, // t0 non numerique
		`{"schemaVersion":39,"matchId":"a","autresTruc":[{"t0":19,"t1":10}]}`,  // t1 < t0
	}
	for _, texte := range cas {
		e := Empreindre(doc(t, texte))
		for k := range e.Mesures {
			if k == "autres:autresTruc/autresTruc/duree-totale" {
				t.Fatalf("texte %q : un objet sans intervalle valide ne doit poser aucune duree (%s)", texte, k)
			}
		}
	}
}

// TestDureeImbriqueeDeuxNiveaux — les occupations de vehicule (`vehicles[].rides[]`) vivent a
// DEUX niveaux sous leur calque de premier niveau : la recursion doit les atteindre.
func TestDureeImbriqueeDeuxNiveaux(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":38,"matchId":"a","vehicles":[
		{"slot":771,"family":"warthog","t0":0,"t1":999,
		 "rides":[{"t0":10,"t1":59,"slot":5,"xuid":"42"}]}]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"a","vehicles":[
		{"slot":771,"family":"warthog","t0":0,"t1":999,
		 "rides":[{"t0":10,"t1":39,"slot":5,"xuid":"42"}]}]}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "vehicules", "vehicles.rides/duree-totale/par-xuid/42"); got != SensPerte {
		t.Fatalf("l'occupation rognee (50 -> 30 frames) doit etre vue a deux niveaux de profondeur, sens %q (rapport : %+v)",
			got, r.Differences)
	}
	// La vie du vehicule elle-meme (t0/t1 au premier niveau du calque) n'a pas bouge.
	if got := sensDe(r, "vehicules", "vehicles/duree-totale/par-slot/771"); got != "" {
		t.Fatalf("la duree de la vie du vehicule ne doit pas bouger, sens %q", got)
	}
}

// TestSensInverseEstUnGain — miroir de TestBaisseDeCompteEstUnePerte (comparaison.go) : sans
// cette assertion, un axe qui nommerait « perte » toute variation passerait les cas ci-dessus.
func TestDureesSensInverseEstUnGain(t *testing.T) {
	court := doc(t, `{"schemaVersion":38,"matchId":"a","grappleLines":[{"t0":0,"t1":9,"slot":1}]}`)
	long := doc(t, `{"schemaVersion":39,"matchId":"a","grappleLines":[{"t0":0,"t1":19,"slot":1}]}`)
	r := Comparer(Empreindre(court), Empreindre(long))
	if got := sensDe(r, "equipement", "grappleLines/duree-totale"); got != SensGain {
		t.Fatalf("10 -> 20 frames doit etre un gain, pas %q", got)
	}
}
