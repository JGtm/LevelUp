package grammar

// lectures_bipedes_test.go — LE CONTRAT DES LECTURES BIPEDES : le rejeu suit celui de la marche de
// record, chaque publication de la marche va au composant qui la porte, l ancrage ne passe que
// derriere elle, et une lecture que les deux sources portent a la meme valeur.

import (
	"fmt"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// traceur rend une observation dont les onze crochets des lecteurs bipedes ecrivent chaque appel,
// valeurs comprises, dans la trace rendue.
func traceur() (*Observation, *[]string) {
	var trace []string
	noter := func(nom string, v ...any) { trace = append(trace, nom+fmt.Sprint(v...)) }
	obs := NouvelleObservation()
	obs.AbilityEnergyHook = func(m uint32, ch [AbilityEnergyCharges]int) { noter("charges", m, ch) }
	obs.SpartanAbilityHook = func(tag, sub, ref uint64, a bool) { noter("spartan", tag, sub, ref, a) }
	obs.AbilityNonPredictedHook = func(st AbilityNonPredictedState) { noter("nonPredite", st) }
	obs.AbilitySetHook = func(n uint64, rang, w int) { noter("rang", n, rang, w) }
	obs.CamoStateHook = func(st CamoState) { noter("camo", st) }
	obs.GrenadeSetHook = func(m uint32, sel int) { noter("choix", m, sel) }
	obs.GrenadeCountsHook = func(n uint64, v []uint64) { noter("comptes", n, v) }
	obs.HeldWeaponHook = func(h, l uint32) { noter("arme", h, l) }
	obs.WeaponAmmoHook = func(a bool, m uint32, b bool, f uint32) { noter("munitions", a, m, b, f) }
	obs.WeaponRoundsHook = func(n uint32) { noter("cartouches", n) }
	obs.UnitEquipmentHook = func(u UnitEquipmentRead) { noter("equipement", u) }
	return obs, &trace
}

// appelTrace rend une publication qui ecrit `etiquette` dans la trace du crochet des cartouches.
func appelTrace(composant int, etiquette uint32) appelDeComposant {
	return appelDeComposant{composant: composant, rejouer: func(o *Observation) { o.WeaponRoundsHook(etiquette) }}
}

// TestLeRejeuSuitLeContratDeLaMarcheDeRecord : les publications d un composant se rejouent AVANT
// sa visite, dans l ordre des index ; une visite qui rend faux arrete ; le composant d arret
// publie sans etre visite, et seulement si le parcours va jusqu a lui.
func TestLeRejeuSuitLeContratDeLaMarcheDeRecord(t *testing.T) {
	r := &recordBipedeLu{masque: masqueDesIndex([]int{0, 21, 30, 47, 56}), atteints: masqueDesIndex([]int{21, 30, 47}),
		arret: 56, appels: []appelDeComposant{appelTrace(21, 1), appelTrace(30, 2), appelTrace(30, 3), appelTrace(56, 4)}}
	obs, trace := traceur()
	var ordre []string
	obs.WeaponRoundsHook = func(n uint32) { ordre = append(ordre, fmt.Sprint("appel", n)) }
	r.parcourir(obs, func(id int) bool {
		ordre = append(ordre, fmt.Sprint("visite", id))
		return true
	})
	attendu := []string{"appel1", "visite21", "appel2", "appel3", "visite30", "visite47", "appel4"}
	if !slices.Equal(ordre, attendu) {
		t.Fatalf("ordre du rejeu %v, attendu %v", ordre, attendu)
	}
	if len(*trace) != 0 {
		t.Fatalf("une publication est partie sur un autre crochet : %v", *trace)
	}

	ordre = nil
	if !r.parcourirJusqua(obs, 30) || !slices.Equal(ordre, []string{"appel1", "appel2", "appel3"}) {
		t.Fatalf("jusqu a i30 : %v — la cible doit etre atteinte, ni i47 ni l arret rejoues", ordre)
	}
	ordre = nil
	if r.parcourirJusqua(obs, 56) || !slices.Contains(ordre, "appel4") {
		t.Fatalf("jusqu a l arret : %v — l arret publie sans etre atteint", ordre)
	}
	if !r.annonce(56) || r.annonce(22) || r.annonce(-1) || r.annonce(64) {
		t.Fatal("annonce ne lit pas le masque")
	}
}

// TestChaquePublicationVaAuComposantQuiLaPorte : une publication tombe dans un composant APRES son
// premier bit et jusqu au depart du suivant compris ; ce qui precede le record, ou suit son
// dernier composant, n est a personne ; le composant sur lequel la lecture s arrete garde ce qu il
// a publie jusqu a la fin du record.
func TestChaquePublicationVaAuComposantQuiLaPorte(t *testing.T) {
	r := &FrameRecord{Slot: 520, ID: 2<<30 | 520, Type: recDelta, TypeIndex: BipedTypeIndex, HeaderBit: 90, FinBit: 400,
		Trace: EntityTrace{Mask: masqueDesIndex([]int{0, 21, 30, 56}), Comps: []CompResult{
			{Index: 0, Ported: true, StartBit: 100}, {Index: 21, Ported: true, StartBit: 160},
			{Index: 30, Ported: true, StartBit: 200}, {Index: 56, Ported: false, StartBit: 300},
		}}}
	rb := recordDeLaMarche(&lecture.Paquet{Chunk: 3, Index: 7, TS: 11}, r)
	if rb.I0 != 100 || rb.Gen != 2 || rb.arret != 56 || rb.atteints != masqueDesIndex([]int{21, 30}) ||
		rb.Chunk != 3 || rb.Packet.Index != 7 || rb.Packet.TimestampUS != 11 {
		t.Fatalf("record %+v : i0 100, generation 2, arret i56, atteints i21 et i30 attendus", rb)
	}
	var appels []appelEnAttente
	for _, b := range []int{50, 160, 161, 200, 201, 300, 350, 400, 401} {
		appels = append(appels, appelEnAttente{bit: b, rejouer: func(*Observation) {}})
	}
	hors := 0
	k := attribuerLesAppels(&rb, r, appels, 0, &hors)
	var vu []int
	for _, a := range rb.appels {
		vu = append(vu, a.composant)
	}
	// 50 precede le record ; 160 est la fin d i0 (rien) ; 161 et 200 sont d i21 ; 201 et 300 d i30 ;
	// 350 et 400 d i56, sur lequel la lecture s arrete ; 401 suit le record.
	if want := []int{0, 21, 21, 30, 30, 56, 56}; !slices.Equal(vu, want) {
		t.Fatalf("composants attribues %v, attendu %v", vu, want)
	}
	if hors != 1 || k != len(appels)-1 {
		t.Fatalf("hors record %d (attendu 1), rang rendu %d (attendu %d)", hors, k, len(appels)-1)
	}
}

// TestLAncrageNePasseQueDerriereLaMarcheSurLaMiniBobine : sur des octets reels, un record recupere
// par l ancrage n est jamais dans ce que la fermeture d une trame prouve, ni d un slot que la marche
// a lu dans son paquet ; aucun record, d une source ou de l autre, n est d un corps que la garde des
// generations vivantes datees refuse ; et quand l ancrage ancre un record que la marche a lu, ils
// publient les memes valeurs, composant par composant.
func TestLAncrageNePasseQueDerriereLaMarcheSurLaMiniBobine(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	fc := NewFilmContext(film)
	canal := nouveauCanalDesLecturesBipedes(fc)
	if err := Distribuer(fc, canal); err != nil {
		t.Fatal(err)
	}
	lu := fc.recup.lectures
	if lu == nil || lu.recuperes == 0 || lu.recuperes == len(lu.records) {
		t.Fatalf("la bobine doit porter des records lus ET recuperes : %+v", lu)
	}
	if got := fc.ComptesDesReplis().AncragesBipedesApresLaMarche; got != lu.recuperes {
		t.Fatalf("rapport des replis %d, records recuperes %d", got, lu.recuperes)
	}
	parLaMarche := map[cleDeRecord]*recordBipedeLu{}
	for i := range lu.records {
		r := &lu.records[i]
		k := cleDeRecord{paquetDuFlux{r.Chunk, r.Packet.Index}, r.Slot, r.I0}
		if !fc.GenerationsVivantesA(r.Packet.TimestampUS).Accepte(types.LifeKey{Slot: r.Slot, Gen: r.Gen}) {
			t.Fatalf("record %+v (recupere %v) d un corps que la garde des generations vivantes refuse", k, r.Recupere)
		}
		if !r.Recupere {
			parLaMarche[k] = r
			continue
		}
		tr := canal.trames[k.paquet]
		if int64(r.I0) >= int64(tr.prouveeDes) || slices.Contains(tr.slots, r.Slot) {
			t.Fatalf("record recupere %+v dans ce que la fermeture prouve (des le bit %d) ou d un slot que la marche a lu",
				k, tr.prouveeDes)
		}
	}
	// Un corps mort n agit plus : aucun record d une vie a l instant de son dead-state ou apres.
	if lu.corpsMorts == 0 {
		t.Fatal("la bobine porte des morts : des records de corps morts doivent avoir ete ecartes")
	}
	for i := range lu.records {
		r := &lu.records[i]
		if mort, connue := canal.mortA[types.LifeKey{Slot: r.Slot, Gen: r.Gen}]; connue && r.Packet.TimestampUS >= mort {
			t.Fatalf("record du slot %d a %d, apres le dead-state de sa vie (%d)", r.Slot, r.Packet.TimestampUS, mort)
		}
	}
	comparerLesSources(t, fc, parLaMarche)
}

// cleDeRecord designe un record bipede : son paquet, son slot, le bit de son i0.
type cleDeRecord struct {
	paquet paquetDuFlux
	slot   uint32
	i0     int
}

// comparerLesSources compare, sur chaque record que l ancrage et la marche lisent tous deux au meme
// bit, ce qu ils publient composant par composant.
func comparerLesSources(t *testing.T, fc *FilmContext, parLaMarche map[cleDeRecord]*recordBipedeLu) {
	t.Helper()
	lay, _ := fc.I0Layout()
	arch, _ := fc.bipedArchetype()
	obsA, traceA := traceur()
	obsM, traceM := traceur()
	g := grammaireRecord{lay: lay, arch: arch, prof: fc.ProfilDeBalayage(), obs: obsA}
	communs := 0
	fc.parcourirLesAncresBipedes(func(a deltaBipedRecord) {
		m := parLaMarche[cleDeRecord{paquetDuFlux{a.Chunk, a.Packet.Index}, a.Slot, a.I0}]
		if m == nil {
			return
		}
		communs++
		*traceA, *traceM = (*traceA)[:0], (*traceM)[:0]
		walkRecordComponents(a.Payload, a.I0, a.Total, a.Mask, g, func(int) bool { return true })
		m.parcourir(obsM, func(int) bool { return true })
		if !slices.Equal(*traceA, *traceM) {
			t.Fatalf("chunk %d paquet %d slot %d : l ancrage publie %v, la marche %v", a.Chunk, a.Packet.Index,
				a.Slot, *traceA, *traceM)
		}
	})
	if communs == 0 {
		t.Fatal("aucun record commun aux deux sources : la comparaison ne garde rien")
	}
}

// TestLAncrageNeRendQueCeQueLaMarcheNaPasLu : la regle de l ancrage derriere la marche, cas par cas,
// pour un record du slot 520 dont l i0 commence au bit 300.
func TestLAncrageNeRendQueCeQueLaMarcheNaPasLu(t *testing.T) {
	cas := []struct {
		nom   string
		trame trameDuCanal
		vu    bool
		rend  bool
	}{
		{"trame que la marche n a pas rendue", trameDuCanal{}, false, true},
		{"trame fermee depuis la tete", trameDuCanal{prouveeDes: 0}, true, false},
		{"trame fermee depuis la tete, sans le slot", trameDuCanal{slots: []uint32{600}}, true, false},
		{"debut localise apres le record", trameDuCanal{prouveeDes: 400, slots: []uint32{600}}, true, true},
		{"debut localise avant le record", trameDuCanal{prouveeDes: 200, slots: []uint32{600}}, true, false},
		{"debut localise apres le record, slot lu", trameDuCanal{prouveeDes: 400, slots: []uint32{520}}, true, false},
		{"trame non fermee ou la marche a lu le slot", trameDuCanal{prouveeDes: rienDeProuve, slots: []uint32{520}}, true, false},
		{"trame non fermee sans le slot", trameDuCanal{prouveeDes: rienDeProuve, slots: []uint32{600}}, true, true},
		{"trame non fermee sans record lu", trameDuCanal{prouveeDes: rienDeProuve}, true, true},
	}
	for _, c := range cas {
		if got := rendParLAncrage(c.trame, c.vu, 520, 300); got != c.rend {
			t.Errorf("%s : rend %v, attendu %v", c.nom, got, c.rend)
		}
	}
}

// TestLaMarcheNeDonneQueLesCorpsVivants : un record d une generation que la garde des generations
// vivantes refuse ne va a aucun lecteur et se compte ; le record qui porte le dead-state d une vie
// et ceux de ce corps qui le suivent sont ecartes et comptes, jusqu au record NEW qui recree la
// generation ; un record qui n annonce aucun composant des lecteurs n est pas retenu.
func TestLaMarcheNeDonneQueLesCorpsVivants(t *testing.T) {
	c := &canalDesLecturesBipedes{trames: map[paquetDuFlux]trameDuCanal{}, mortA: map[types.LifeKey]uint64{},
		utiles: masqueDesIndex([]int{21})}
	gens := NouvellesGenerationsVivantes([]types.LifeKey{{Slot: 520, Gen: 1}, {Slot: 521, Gen: 1}})
	record := func(typ int, slot, gen uint32, mort bool, idx ...int) FrameRecord {
		r := FrameRecord{Slot: slot, ID: gen<<30 | slot, Type: typ, TypeIndex: BipedTypeIndex, HeaderBit: 10, FinBit: 90,
			Trace: EntityTrace{Mask: masqueDesIndex(idx)}}
		for j, id := range idx {
			r.Trace.Comps = append(r.Trace.Comps, CompResult{Index: id, Ported: true, StartBit: 20 + 10*j})
		}
		if mort {
			r.Trace.Dead = &types.DeadState{}
		}
		return r
	}
	paquet := func(ts uint64) *lecture.Paquet {
		return &lecture.Paquet{Chunk: 1, Index: int(ts), TS: ts, Fermeture: lecture.Fermeture{Verdict: lecture.VerdictFerme}}
	}
	c.recueillir(paquet(1), []FrameRecord{
		record(recDelta, 520, 1, false, 0, 21), // retenu
		record(recDelta, 520, 0, false, 0, 21), // generation inconnue du slot : refuse
		record(recDelta, 521, 1, false, 0, 5),  // n annonce rien des lecteurs
	}, nil, gens)
	c.recueillir(paquet(2), []FrameRecord{record(recDelta, 520, 1, true, 0, 21)}, nil, gens)  // dead-state
	c.recueillir(paquet(3), []FrameRecord{record(recDelta, 520, 1, false, 0, 21)}, nil, gens) // corps mort
	c.recueillir(paquet(4), []FrameRecord{record(recNew, 520, 1, false, 0, 21), record(recDelta, 520, 1, false, 0, 21)}, nil, gens)
	var retenus []uint64
	for _, r := range c.lu.records {
		retenus = append(retenus, r.Packet.TimestampUS)
	}
	if !slices.Equal(retenus, []uint64{1, 4}) || c.lu.generationsRefusees != 1 || c.lu.corpsMorts != 2 || c.lu.examines != 6 {
		t.Fatalf("retenus aux instants %v (attendu [1 4]), generations refusees %d (1), corps morts %d (2), examines %d (6)",
			retenus, c.lu.generationsRefusees, c.lu.corpsMorts, c.lu.examines)
	}
	if got := c.trames[paquetDuFlux{1, 1}]; got.prouveeDes != 0 || !slices.Equal(got.slots, []uint32{520, 520, 521}) {
		t.Fatalf("trame retenue %+v : prouvee des 0, slots lus 520, 520, 521 attendus", got)
	}
}

// TestCeQueLaFermetureProuve : une trame fermee partie de la tete prouve le paquet entier ; une trame
// fermee dont le debut de vue B a ete localise, sa liste depuis ce debut ; une trame non fermee, rien.
func TestCeQueLaFermetureProuve(t *testing.T) {
	cas := []struct {
		verdict lecture.Verdict
		debut   lecture.DebutDeVueB
		prouve  uint32
	}{
		{lecture.VerdictFerme, lecture.DebutEnTete, 0},
		{lecture.VerdictFerme, lecture.DebutParVueA, 0},
		{lecture.VerdictFerme, lecture.DebutParSignature, 250},
		{lecture.VerdictFerme, lecture.DebutParChaine, 250},
		{lecture.VerdictFerme, lecture.DebutParFermeture, 250},
		{lecture.VerdictRefuse, lecture.DebutEnTete, rienDeProuve},
		{lecture.VerdictRefuse, lecture.DebutParFermetureAuBit, rienDeProuve},
		{lecture.VerdictQueueOpaque, lecture.DebutEnTete, rienDeProuve},
		{lecture.VerdictQueueOpaque, lecture.DebutNonLocalise, rienDeProuve},
	}
	for _, c := range cas {
		p := &lecture.Paquet{Debut: c.debut, VueB: lecture.VueB{Debut: 250}, Fermeture: lecture.Fermeture{Verdict: c.verdict}}
		if got := preuveDeLaTrame(p); got != c.prouve {
			t.Errorf("verdict %d, debut %d : prouve des %d, attendu %d", c.verdict, c.debut, got, c.prouve)
		}
	}
}
