package filmdec

// ability_impulses_test.go — les pièces PURES du balayage des impulsions de capacité : la
// résolution du composant par son nom, le filtre de tag, et l'ordre total de la sortie. Le
// balayage lui-même se juge sur pièces (golden d'assemblage du paquet `replay`, et test
// d'acceptation contre le relevé Theater).

import "testing"

func TestComponentIndexOfAnyResoutParNomExact(t *testing.T) {
	// LA RESOLUTION EST EXACTE, jamais par prefixe : « biped-spartan-ability » est un prefixe
	// de « biped-spartan-ability-non-predicted-state », et un appariement par prefixe
	// confondrait i57 avec i59 — donc le tag predit avec le non predit.
	// Les noms viennent des CONSTANTES de production, jamais d'une copie : un test qui
	// réécrirait les étiquettes cesserait de contrôler celles que le film porte.
	comps := make([]string, 60)
	comps[48] = "biped-desired-ability-set-component"
	comps[57] = abilityPredictedNameAlt
	comps[59] = grappleComponentNameAlt
	arch := Archetype{Components: comps}
	if got := componentIndexOfAny(arch, abilityPredictedName, abilityPredictedNameAlt); got != 57 {
		t.Fatalf("i57 resolu a %d, attendu 57 (etiquette alternative)", got)
	}
	if got := componentIndexOfAny(arch, grappleComponentName, grappleComponentNameAlt); got != 59 {
		t.Fatalf("i59 resolu a %d, attendu 59", got)
	}
	if got := componentIndexOfAny(Archetype{Components: []string{"autre"}},
		abilityPredictedName, abilityPredictedNameAlt); got != -1 {
		t.Fatalf("composant absent resolu a %d, attendu -1", got)
	}
}

// TestAbilityImpulseScannerEpingleLaValeurDuTag — LE TAG EST ÉPINGLÉ EN DUR, ET C'EST LE POINT
// DU TEST.
//
// La constante de production n'est JAMAIS lue ici. Un test qui injecterait
// `abilityImpulseTag` et vérifierait qu'une valeur sur quatre sort passerait pour N'IMPORTE
// QUELLE valeur de la constante — y compris 3, LE TAG DU GRAPPIN (grapple_state.go) : le
// document publierait alors les corps d'accroche de grappin sous `family:"thruster"`, et rien
// dans la suite ne bougerait. Les quatre valeurs sont donc écrites à la main, avec le verdict
// attendu de chacune.
func TestAbilityImpulseScannerEpingleLaValeurDuTag(t *testing.T) {
	cas := []struct {
		tag     uint64
		publiee bool
		quoi    string
	}{
		{0, false, "etat de repos (1 572 lectures sur 00ba2e1c, pic au niveau du hasard)"},
		{1, true, "L IMPULSION — la seule que ce canal publie"},
		{2, false, "etat de repos (1 565 lectures, idem)"},
		{3, false, "LE GRAPPIN — il a son propre calque (grappleLines), jamais celui-ci"},
	}
	for _, c := range cas {
		var st AbilityImpulseStats
		sc := &abilityImpulseScanner{st: &st}
		sc.publish(true, c.tag, true, 512, 3, FilmPacket{Index: 7, TimestampUS: 1_234_000})
		if got := len(sc.out) == 1; got != c.publiee {
			t.Fatalf("tag %d (%s) : publiee=%t, attendu %t", c.tag, c.quoi, got, c.publiee)
		}
		if st.Read != 1 {
			t.Fatalf("tag %d : lues=%d, attendu 1 — une lecture aboutie compte quel que soit son tag",
				c.tag, st.Read)
		}
		if st.Tag1 != boolToInt(c.publiee) {
			t.Fatalf("tag %d : tag1=%d, attendu %d", c.tag, st.Tag1, boolToInt(c.publiee))
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestAbilityImpulsePublieLaLocalisationEtLeTemoinDeComposant(t *testing.T) {
	var st AbilityImpulseStats
	sc := &abilityImpulseScanner{st: &st}
	pk := FilmPacket{Index: 7, TimestampUS: 1_234_000}
	sc.publish(true, 1, true, 512, 3, pk)
	// Une lecture NON aboutie ne compte ni comme lue ni comme impulsion.
	sc.publish(false, 1, false, 512, 3, pk)
	if st.Read != 1 || len(sc.out) != 1 {
		t.Fatalf("lues=%d sorties=%d, attendu 1/1 — une lecture non aboutie a ete comptee",
			st.Read, len(sc.out))
	}
	got := sc.out[0]
	if got.Slot != 512 || got.Chunk != 3 || got.PacketIndex != 7 || got.TimestampUS != 1_234_000 {
		t.Fatalf("impulsion %+v : la localisation du paquet ne voyage pas", got)
	}
	if !got.Predicted {
		t.Fatal("le temoin i57/i59 ne voyage pas : la couverture ne saurait plus lequel a parle")
	}
}

// TestAbilityImpulseEmetLesDeuxComposants — i57 ET i59 CONTRIBUENT, chacun pour sa lecture.
// N'en publier qu'un ferait tomber `coverage.abilityImpulses.reads` de moitié dans le document
// servi (86 -> 43 sur le film de référence) sans qu'aucun compteur ne le dise.
func TestAbilityImpulseEmetLesDeuxComposants(t *testing.T) {
	var st AbilityImpulseStats
	sc := &abilityImpulseScanner{st: &st}
	sc.tag57, sc.got57 = 1, true
	sc.tag59, sc.got59 = 1, true
	sc.emit(true, true, 512, 3, FilmPacket{Index: 7, TimestampUS: 1_234_000})
	if st.Read != 2 || st.Tag1 != 2 || len(sc.out) != 2 {
		t.Fatalf("lues=%d tag1=%d sorties=%d, attendu 2/2/2 — un composant ne contribue plus",
			st.Read, st.Tag1, len(sc.out))
	}
	if !sc.out[0].Predicted {
		t.Fatal("la premiere sortie doit venir du composant PREDIT (i57)")
	}
	if sc.out[1].Predicted {
		t.Fatal("la seconde sortie doit venir du composant NON PREDIT (i59)")
	}
	// Un composant annoncé dont la lecture n'a pas abouti ne publie rien ET compte au
	// dénominateur des perdues : les deux voies sont indépendantes.
	st, sc.got59 = AbilityImpulseStats{}, false
	sc.st, sc.out = &st, nil
	sc.emit(true, true, 512, 3, FilmPacket{Index: 8})
	if st.Read != 1 || st.Unread != 1 || len(sc.out) != 1 {
		t.Fatalf("lues=%d illisibles=%d sorties=%d, attendu 1/1/1", st.Read, st.Unread, len(sc.out))
	}
}

func TestImputeUnreadCompteChaqueComposantAnnonceEtNonAtteint(t *testing.T) {
	// UN COMPOSANT ANNONCE ET NON LU N'EST PAS UNE ABSENCE D'IMPULSION : c'est une lecture
	// perdue, et le denominateur doit le dire. Les deux composants comptent separement.
	var st AbilityImpulseStats
	sc := &abilityImpulseScanner{st: &st}
	sc.got57, sc.got59 = false, false
	sc.imputeUnread(true, true)
	if st.Unread != 2 {
		t.Fatalf("illisibles=%d, attendu 2 (i57 et i59 annonces, aucun atteint)", st.Unread)
	}
	sc.got57 = true
	sc.imputeUnread(true, true)
	if st.Unread != 3 {
		t.Fatalf("illisibles=%d, attendu 3 (i57 lu, i59 non)", st.Unread)
	}
	sc.imputeUnread(false, false)
	if st.Unread != 3 {
		t.Fatalf("illisibles=%d : un composant NON annonce a ete compte comme perdu", st.Unread)
	}
}

func TestSortAbilityImpulsesOrdreTotal(t *testing.T) {
	// L'ORDRE EST TOTAL — instant, puis slot, puis composant : sans le troisieme critere, deux
	// lectures co-transmises du meme slot au meme instant sortiraient dans l'ordre du parcours,
	// et l'artefact dependrait de rien de mesurable.
	out := []AbilityImpulse{
		{Slot: 20, TimestampUS: 5, Predicted: false},
		{Slot: 10, TimestampUS: 5, Predicted: false},
		{Slot: 10, TimestampUS: 5, Predicted: true},
		{Slot: 10, TimestampUS: 1, Predicted: false},
	}
	sortAbilityImpulses(out)
	want := []AbilityImpulse{
		{Slot: 10, TimestampUS: 1, Predicted: false},
		{Slot: 10, TimestampUS: 5, Predicted: true},
		{Slot: 10, TimestampUS: 5, Predicted: false},
		{Slot: 20, TimestampUS: 5, Predicted: false},
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("position %d = %+v, attendu %+v (ordre %+v)", i, out[i], want[i], out)
		}
	}
}
