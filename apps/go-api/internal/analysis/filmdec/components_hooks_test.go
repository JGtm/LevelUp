package filmdec

// components_hooks_test.go — LE GARDE-RAIL DE LA PLOMBERIE DE PUBLICATION (lot 0 item 0.6).
//
// CE QUE LA PLOMBERIE PROMET, ET CE QUE CE FICHIER VERIFIE. Vingt-trois `case` ont quitte le
// `switch` de `consumeByName` pour des fonctions nommees qui PUBLIENT par un hook de paquet.
// La promesse est double, et chaque moitie a son test :
//
//	1. SANS HOOK, COMPORTEMENT IDENTIQUE BIT A BIT — `TestHooksConsumeSameBitsWithoutHook`
//	   compare, sur 500 tampons aleatoires par composant, la position du lecteur et le drapeau
//	   `ported` avec et sans hook installe. Meme esprit et meme raison d'etre que
//	   `TestCaptureConsumesSameBitsAsDispatch` : une publication qui deplacerait un bit
//	   desynchroniserait tout le reste du record, et ce serait le pire bug de ce decodeur.
//	2. LE HOOK RECOIT EXACTEMENT LES BITS LUS — les tests par famille ci-dessous construisent
//	   des flux dont on maitrise chaque champ et exigent la valeur ET le compte de bits.
//
// LE BALAYAGE EST PILOTE PAR UNE LISTE (`hookedNames`) : deplacer un `case` de plus sans
// l'ajouter a cette liste ne le protege de rien, et l'oubli se voit — le test de completude
// `TestHookedNamesCoversMovedCases` confronte la liste au dispatch reel.

import (
	"math/rand"
	"reflect"
	"testing"
)

// hookedNames liste TOUS les composants dont le `case` a ete deplace au lot 0 item 0.6.
// L'ordre est celui des familles : ti=0, ti=5, ti=37, ti=10, sondes.
var hookedNames = []string{
	// ti=0 — moteur de jeu
	compGameEngineCurrentState, compGameEngineCurrentRound, compGameEngineSuddenDeath,
	compGameEngineGracePeriod, compGameEngineRoundConditionFlags,
	// ti=5 — entite joueur
	compPlayerSoftKillTimer, compPlayerTargetTracking, compPlayerDesiredRespawnPlayer,
	compPlayerEngineLoadout, compPlayerDesiredRespawnLoc, compPlayerLivesRemaining,
	compPlayerLastBetrayer, compPlayerControlAiming, compPlayerActiveInGame,
	compPlayerPendingJoinInProgress, compPlayerMalleableProperties,
	// ti=37 — equipement (deux canaux ajoutes au hook existant)
	compEquipmentEnergyDelay, compEquipmentCharges,
	// ti=10 — objet scripte du mode
	compManagedObjectBoundaryVisibility,
	// sondes
	compSplashMessageStatic, compSplashMessageDynamic, compHighFrequency,
	compManagedObjectPropName,
}

// clearAllHooks retire les quatre hooks et le hook d'equipement, et les restaure a la sortie.
func clearAllHooks(t *testing.T) {
	t.Helper()
	ge, ps, mo, pr, eq := gameEngineHook, playerStateHook, managedObjectHook, probeHook, equipmentStateHook
	t.Cleanup(func() {
		gameEngineHook, playerStateHook, managedObjectHook = ge, ps, mo
		probeHook, equipmentStateHook = pr, eq
	})
	gameEngineHook, playerStateHook, managedObjectHook, probeHook, equipmentStateHook = nil, nil, nil, nil, nil
}

// installCountingHooks installe des hooks qui comptent leurs appels, sur les cinq familles.
func installCountingHooks(appels *int) {
	gameEngineHook = func(GameEngineField, []uint64, bool) { *appels++ }
	playerStateHook = func(PlayerStateField, []uint64, bool) { *appels++ }
	managedObjectHook = func(ManagedObjectField, []uint64) { *appels++ }
	probeHook = func(uint32, ProbeComponent, []uint64) { *appels++ }
	equipmentStateHook = func(EquipmentField, uint64, bool) { *appels++ }
}

// TestHooksConsumeSameBitsWithoutHook — LA MOITIE « aucun bit ne bouge » DE LA PROMESSE.
func TestHooksConsumeSameBitsWithoutHook(t *testing.T) {
	rng := rand.New(rand.NewSource(20260817))
	buf := make([]byte, 128)
	for _, name := range hookedNames {
		for iter := 0; iter < 500; iter++ {
			for i := range buf {
				buf[i] = byte(rng.Intn(256))
			}
			// Le niveau entre dans la largeur du vec3 de ti=5 i12 : on le fait varier.
			level := uint32(iter % 12)

			clearAllHooks(t)
			sans := NewBitReader(buf)
			_, _, portedSans := consumeByName(sans, name, BipedTypeIndex, level)

			appels := 0
			installCountingHooks(&appels)
			avec := NewBitReader(buf)
			_, _, portedAvec := consumeByName(avec, name, BipedTypeIndex, level)

			if sans.BitPos() != avec.BitPos() {
				t.Fatalf("%s (iteration %d, niveau %d) : %d bits sans hook, %d avec — la "+
					"publication a deplace le lecteur", name, iter, level, sans.BitPos(), avec.BitPos())
			}
			if portedSans != portedAvec {
				t.Fatalf("%s : `ported` diverge (sans=%v avec=%v)", name, portedSans, portedAvec)
			}
			if appels == 0 {
				t.Fatalf("%s : le hook n'a PAS ete appele — le `case` ne publie rien", name)
			}
		}
	}
}

// TestHookedNamesCoversMovedCases — LA COMPLETUDE DE LA LISTE.
//
// Un `case` deplace mais absent de `hookedNames` ne serait garde par rien. Le test interroge le
// DISPATCH REEL : chaque nom de la liste doit y etre traite, et le compte doit valoir celui que
// le plan a ferme (23 composants).
func TestHookedNamesCoversMovedCases(t *testing.T) {
	const attendus = 23
	if len(hookedNames) != attendus {
		t.Fatalf("hookedNames porte %d noms, %d attendus — le perimetre de l'item 0.6 est FERME "+
			"(5 de ti=0, 11 de ti=5, 2 de ti=37, 1 de ti=10, 4 sondes)", len(hookedNames), attendus)
	}
	vus := map[string]bool{}
	for _, n := range hookedNames {
		if vus[n] {
			t.Errorf("%s apparait deux fois dans hookedNames", n)
		}
		vus[n] = true
	}
	// LA COUCHE DE CAPTURE N'EST PAS DOUBLEE PAR UN HOOK, et c'est une regle, pas un hasard :
	// ti=0 i5 et ti=5 i1 sont deja rendus par `consumeByNameCapturing` sous forme typee. Un
	// hook sur eux ferait une TROISIEME copie de la meme grammaire, ce que le depot interdit.
	// Ce test le verrouille pour les lots suivants, qui liront ces hooks sans lire ce fichier.
	for _, n := range captureNames {
		if vus[n] {
			t.Errorf("%s est A LA FOIS capture (capture.go) et publie par un hook — troisieme "+
				"copie de la meme grammaire", n)
		}
	}
	clearAllHooks(t)
	for _, n := range hookedNames {
		appels := 0
		installCountingHooks(&appels)
		br := NewBitReader(make([]byte, 64))
		if _, _, ported := consumeByName(br, n, BipedTypeIndex, 0); !ported {
			t.Errorf("%s : le dispatch rend ported=false sur un tampon nul", n)
		}
		if appels == 0 {
			t.Errorf("%s : aucun hook appele — le nom est dans la liste mais ne publie pas", n)
		}
	}
}

// hookCase decrit un flux construit, la valeur attendue du hook et le cout en bits.
type hookCase struct {
	nom     string
	comp    string
	level   uint32
	ecr     func(w *bitw)
	present bool
	valeurs []uint64
	bits    int
}

// runHookCases joue une table de cas et verifie valeur, presence et compte de bits.
func runHookCases(t *testing.T, cas []hookCase, capture func(*[]uint64, *bool, *int)) {
	t.Helper()
	for _, c := range cas {
		clearAllHooks(t)
		var got []uint64
		var present bool
		var appels int
		capture(&got, &present, &appels)

		w := &bitw{}
		c.ecr(w)
		br := NewBitReader(append(w.buf, make([]byte, 32)...))
		if _, _, ported := consumeByName(br, c.comp, BipedTypeIndex, c.level); !ported {
			t.Errorf("%s : ported=false", c.nom)
		}
		if appels != 1 {
			t.Errorf("%s : %d appel(s) de hook, 1 attendu", c.nom, appels)
			continue
		}
		if br.BitPos() != c.bits {
			t.Errorf("%s : %d bits consommes, %d attendus", c.nom, br.BitPos(), c.bits)
		}
		if present != c.present {
			t.Errorf("%s : present=%v, %v attendu", c.nom, present, c.present)
		}
		if !reflect.DeepEqual(got, c.valeurs) && !(len(got) == 0 && len(c.valeurs) == 0) {
			t.Errorf("%s : valeurs %v, %v attendues", c.nom, got, c.valeurs)
		}
	}
}

// TestGameEngineHookValues — ti=0 : les cinq champs, valeur par valeur.
func TestGameEngineHookValues(t *testing.T) {
	cas := []hookCase{
		{
			nom: "i2 etat de partie", comp: compGameEngineCurrentState,
			ecr: func(w *bitw) { w.put(5, 3) }, present: true, valeurs: []uint64{5}, bits: 3,
		},
		{
			nom: "i4 manche, porte OUVERTE (bit 0)", comp: compGameEngineCurrentRound,
			ecr:     func(w *bitw) { w.put(0, 1); w.put(19, 5) },
			present: true, valeurs: []uint64{19}, bits: 6,
		},
		{
			nom: "i4 manche, porte FERMEE (bit 1)", comp: compGameEngineCurrentRound,
			ecr:     func(w *bitw) { w.put(1, 1); w.put(31, 5) },
			present: false, valeurs: nil, bits: 1,
		},
		{
			nom: "i6 mort subite", comp: compGameEngineSuddenDeath,
			ecr:     func(w *bitw) { w.put(0xbeef, 16); w.put(0x1234, 16); w.put(21, 5) },
			present: true, valeurs: []uint64{0xbeef, 0x1234, 21}, bits: 37,
		},
		{
			nom: "i7 periode de grace", comp: compGameEngineGracePeriod,
			ecr:     func(w *bitw) { w.put(1, 16); w.put(2, 16); w.put(3, 5) },
			present: true, valeurs: []uint64{1, 2, 3}, bits: 37,
		},
		{
			nom: "i8 conditions de fin de manche", comp: compGameEngineRoundConditionFlags,
			ecr: func(w *bitw) { w.put(0x2a5, 10) }, present: true, valeurs: []uint64{0x2a5}, bits: 10,
		},
	}
	runHookCases(t, cas, func(got *[]uint64, present *bool, appels *int) {
		gameEngineHook = func(_ GameEngineField, v []uint64, p bool) {
			*got, *present, *appels = v, p, *appels+1
		}
	})
}

// TestPlayerStateHookValues — ti=5 : les onze champs, valeur par valeur.
func TestPlayerStateHookValues(t *testing.T) {
	cas := []hookCase{
		{
			nom: "i2 minuteur de mort douce", comp: compPlayerSoftKillTimer,
			ecr:     func(w *bitw) { w.put(3, 5); w.put(17, 5); w.put(31, 5) },
			present: true, valeurs: []uint64{3, 17, 31}, bits: 15,
		},
		{
			nom: "i3 detection de cible", comp: compPlayerTargetTracking,
			ecr: func(w *bitw) { w.put(1, 1); w.put(0, 1) }, present: true,
			valeurs: []uint64{1, 0}, bits: 2,
		},
		{
			nom: "i6 joueur de reapparition desire", comp: compPlayerDesiredRespawnPlayer,
			ecr: func(w *bitw) { w.put(0xcafe, 16) }, present: true, valeurs: []uint64{0xcafe}, bits: 16,
		},
		{
			nom: "i11 chargement de depart (8 octets)", comp: compPlayerEngineLoadout,
			ecr: func(w *bitw) {
				for _, b := range []uint64{1, 2, 3, 4, 250, 251, 252, 253} {
					w.put(b, 8)
				}
			},
			present: true, valeurs: []uint64{1, 2, 3, 4, 250, 251, 252, 253}, bits: 64,
		},
		{
			nom: "i14 vies restantes", comp: compPlayerLivesRemaining,
			ecr: func(w *bitw) { w.put(77, 7) }, present: true, valeurs: []uint64{77}, bits: 7,
		},
		{
			nom: "i15 dernier traitre", comp: compPlayerLastBetrayer,
			ecr: func(w *bitw) { w.put(41, 6) }, present: true, valeurs: []uint64{41}, bits: 6,
		},
		{
			nom: "i17 direction de visee", comp: compPlayerControlAiming,
			ecr: func(w *bitw) { w.put(0x5aaaa, 19) }, present: true, valeurs: []uint64{0x5aaaa}, bits: 19,
		},
		{
			nom: "i18 present en partie", comp: compPlayerActiveInGame,
			ecr: func(w *bitw) { w.put(1, 1) }, present: true, valeurs: []uint64{1}, bits: 1,
		},
		{
			nom: "i19 arrivee en cours de partie", comp: compPlayerPendingJoinInProgress,
			ecr: func(w *bitw) { w.put(0, 1) }, present: true, valeurs: []uint64{0}, bits: 1,
		},
	}
	runHookCases(t, cas, func(got *[]uint64, present *bool, appels *int) {
		playerStateHook = func(_ PlayerStateField, v []uint64, p bool) {
			*got, *present, *appels = v, p, *appels+1
		}
	})
}

// TestPlayerDesiredRespawnLocationHook — ti=5 i12, LES TROIS BRANCHES.
//
// Le composant a deux portes imbriquees, et leurs trois issues ne se confondent pas : porte de
// tete fermee (aucun champ), `precHigh` leve (l'identifiant seul, le vecteur par defaut ne
// coutant aucun bit), et le cas complet. Publier une position a l'origine dans l'un des deux
// premiers cas serait fabriquer une donnee.
func TestPlayerDesiredRespawnLocationHook(t *testing.T) {
	const niveau = 4 // largeur d'axe = 6 + 4 = 10 bits
	cas := []hookCase{
		{
			nom: "porte de tete FERMEE", comp: compPlayerDesiredRespawnLoc, level: niveau,
			ecr: func(w *bitw) { w.put(0, 1) }, present: false, valeurs: nil, bits: 1,
		},
		{
			nom:  "precHigh LEVE : vecteur par defaut, l'identifiant seul",
			comp: compPlayerDesiredRespawnLoc, level: niveau,
			ecr: func(w *bitw) {
				w.put(1, 1)        // porte de tete
				w.put(1, 1)        // precHigh == 1 -> vecteur par defaut, 0 bit
				w.put(0x3ffff, 19) // identifiant de reapparition
			},
			present: false, valeurs: []uint64{0x3ffff}, bits: 1 + 1 + 19,
		},
		{
			nom:  "cas complet : index absent, trois quanta, identifiant",
			comp: compPlayerDesiredRespawnLoc, level: niveau,
			ecr: func(w *bitw) {
				w.put(1, 1)    // porte de tete
				w.put(0, 1)    // precHigh == 0
				w.put(1, 1)    // index-present select == 1 -> pas d'index lu
				w.put(100, 10) // qx
				w.put(200, 10) // qy
				w.put(300, 10) // qz
				w.put(12345, 19)
			},
			present: true, valeurs: []uint64{100, 200, 300, 12345, niveau},
			bits: 1 + 1 + 1 + 30 + 19,
		},
		{
			nom:  "cas complet avec index lu (select == 0 -> R(1))",
			comp: compPlayerDesiredRespawnLoc, level: niveau,
			ecr: func(w *bitw) {
				w.put(1, 1)
				w.put(0, 1)
				w.put(0, 1) // index-present select == 0 -> R(1) d'index
				w.put(1, 1) // l'index
				w.put(7, 10)
				w.put(8, 10)
				w.put(9, 10)
				w.put(1, 19)
			},
			present: true, valeurs: []uint64{7, 8, 9, 1, niveau},
			bits: 1 + 1 + 1 + 1 + 30 + 19,
		},
	}
	runHookCases(t, cas, func(got *[]uint64, present *bool, appels *int) {
		playerStateHook = func(_ PlayerStateField, v []uint64, p bool) {
			*got, *present, *appels = v, p, *appels+1
		}
	})
}

// TestPlayerMalleablePropertiesHook — ti=5 i20 : 24 entrees, bits de porte compris.
//
// C'EST LA FORME QUI IMPORTE ICI : sans les bits de porte, on ne saurait pas LAQUELLE des six
// proprietes a transmis sa valeur. Le test met une porte a 1 et les cinq autres a 0.
func TestPlayerMalleablePropertiesHook(t *testing.T) {
	clearAllHooks(t)
	var got []uint64
	var appels int
	playerStateHook = func(_ PlayerStateField, v []uint64, _ bool) { got, appels = v, appels+1 }

	w := &bitw{}
	w.put(1, 1)
	w.put(0, 1)
	w.put(1, 1) // les trois drapeaux de tete
	for i := 0; i < 6; i++ {
		if i == 2 {
			w.put(1, 1)
			w.put(0xabc, 12)
			continue
		}
		w.put(0, 1)
	}
	for i := 0; i < 9; i++ {
		w.put(uint64(i%2), 1)
	}
	br := NewBitReader(append(w.buf, make([]byte, 8)...))
	consumeByName(br, compPlayerMalleableProperties, BipedTypeIndex, 0)

	if appels != 1 {
		t.Fatalf("%d appel(s) de hook, 1 attendu", appels)
	}
	if n := 3 + 1 + 12 + 5 + 9; br.BitPos() != n {
		t.Fatalf("%d bits consommes, %d attendus", br.BitPos(), n)
	}
	want := []uint64{
		1, 0, 1, // drapeaux de tete
		0, 0, 0, 0, 1, 0xabc, 0, 0, 0, 0, 0, 0, // six couples (porte, valeur)
		0, 1, 0, 1, 0, 1, 0, 1, 0, // neuf drapeaux de queue
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("valeurs %v\nattendues %v", got, want)
	}
}

// TestEquipmentHookNewFields — ti=37 i26 et i27, les deux canaux ajoutes.
func TestEquipmentHookNewFields(t *testing.T) {
	cas := []struct {
		nom   string
		comp  string
		champ EquipmentField
		val   uint64
		bits  int
	}{
		{"i26 delai de tics avant recharge", compEquipmentEnergyDelay, EquipEnergyDelay, 0x3a5, 10},
		{"i27 charges restantes", compEquipmentCharges, EquipCharges, 0xc7, 8},
	}
	for _, c := range cas {
		clearAllHooks(t)
		var gotF EquipmentField
		var gotV uint64
		var gotP bool
		appels := 0
		equipmentStateHook = func(f EquipmentField, v uint64, p bool) {
			gotF, gotV, gotP, appels = f, v, p, appels+1
		}
		w := &bitw{}
		w.put(c.val, c.bits)
		br := NewBitReader(append(w.buf, make([]byte, 8)...))
		consumeByName(br, c.comp, EquipmentTypeIndex, 0)
		switch {
		case appels != 1:
			t.Errorf("%s : %d appel(s), 1 attendu", c.nom, appels)
		case gotF != c.champ:
			t.Errorf("%s : champ %v, %v attendu", c.nom, gotF, c.champ)
		case gotV != c.val:
			t.Errorf("%s : valeur %#x, %#x attendue", c.nom, gotV, c.val)
		case !gotP:
			t.Errorf("%s : present=false alors que le composant n'a pas de porte", c.nom)
		case br.BitPos() != c.bits:
			t.Errorf("%s : %d bits, %d attendus", c.nom, br.BitPos(), c.bits)
		}
	}
	// L'etiquette de registre doit suivre le champ : un `String()` oublie se voit ici.
	if EquipEnergyDelay.String() != compEquipmentEnergyDelay || EquipCharges.String() != compEquipmentCharges {
		t.Errorf("String() ne nomme pas les deux nouveaux champs : %q / %q",
			EquipEnergyDelay.String(), EquipCharges.String())
	}
	if EquipmentFieldCount != 6 {
		t.Errorf("EquipmentFieldCount vaut %d, 6 attendu", EquipmentFieldCount)
	}
}

// TestManagedObjectHookFlagOrder — ti=10 i0 : LE RANG DES 32 DRAPEAUX.
//
// Le jeu lit un bit par iteration et pose l'iteration i au RANG i. `ReadBits(32)` poserait
// l'iteration 0 au rang 31 : ce test distingue les deux, et c'est pour cela que la boucle du
// deser n'est pas raccourcie.
func TestManagedObjectHookFlagOrder(t *testing.T) {
	clearAllHooks(t)
	var got []uint64
	appels := 0
	managedObjectHook = func(_ ManagedObjectField, v []uint64) { got, appels = v, appels+1 }

	w := &bitw{}
	w.put(1, 1) // iteration 0 -> rang 0
	for i := 1; i < 31; i++ {
		w.put(0, 1)
	}
	w.put(1, 1) // iteration 31 -> rang 31
	br := NewBitReader(append(w.buf, make([]byte, 8)...))
	consumeByName(br, compManagedObjectBoundaryVisibility, 10, 0)

	if appels != 1 {
		t.Fatalf("%d appel(s) de hook, 1 attendu", appels)
	}
	if br.BitPos() != 32 {
		t.Fatalf("%d bits consommes, 32 attendus", br.BitPos())
	}
	want := uint64(1) | uint64(1)<<31
	if len(got) != 1 || got[0] != want {
		t.Fatalf("drapeaux %v, [%#x] attendu — l'iteration i doit poser son bit au RANG i", got, want)
	}
}

// TestProbeHookPassesRegistryTypeIndex — LES SONDES, ET LE `ti` QUI VIENT DU REGISTRE.
//
// Le hook doit rendre le `typeIndex` que la traversee lui passe, JAMAIS une constante : deux
// registres differents ont ete mesures sur le corpus (item 0.3), donc cabler un numero
// d'archetype serait faux par avance. Le test appelle le meme composant sous deux `ti`.
func TestProbeHookPassesRegistryTypeIndex(t *testing.T) {
	cas := []struct {
		nom  string
		comp string
		want ProbeComponent
		val  uint64
		bits int
	}{
		{"ti=47 i1 message dynamique", compSplashMessageDynamic, ProbeSplashDynamic, 0xabcdef, 24},
		{"ti=4 i0 haute frequence", compHighFrequency, ProbeHighFrequency, 0x5a, 8},
		{"ti=13 i0 nom de propriete", compManagedObjectPropName, ProbeManagedObjectPropertyName, 0xdeadbeef, 32},
	}
	for _, c := range cas {
		for _, ti := range []uint32{4, 47, 13, 99} {
			clearAllHooks(t)
			var gotTI uint32
			var gotComp ProbeComponent
			var gotVals []uint64
			appels := 0
			probeHook = func(ti uint32, comp ProbeComponent, v []uint64) {
				gotTI, gotComp, gotVals, appels = ti, comp, v, appels+1
			}
			w := &bitw{}
			w.put(c.val, c.bits)
			br := NewBitReader(append(w.buf, make([]byte, 8)...))
			consumeByName(br, c.comp, ti, 0)
			switch {
			case appels != 1:
				t.Errorf("%s (ti=%d) : %d appel(s), 1 attendu", c.nom, ti, appels)
			case gotTI != ti:
				t.Errorf("%s : la sonde recoit ti=%d au lieu de %d — un numero d'archetype est cable",
					c.nom, gotTI, ti)
			case gotComp != c.want:
				t.Errorf("%s : composant %v, %v attendu", c.nom, gotComp, c.want)
			case len(gotVals) != 1 || gotVals[0] != c.val:
				t.Errorf("%s : valeurs %v, [%#x] attendu", c.nom, gotVals, c.val)
			case br.BitPos() != c.bits:
				t.Errorf("%s : %d bits, %d attendus", c.nom, br.BitPos(), c.bits)
			}
		}
	}
}

// TestProbeSplashStaticPublishesUnconditionalField — ti=47 i0 : le R(24) inconditionnel.
//
// Le composant lit un ensemble de references sous porte, PUIS un R(24) toujours present, PUIS
// un corps garde. C'est ce R(24) que la sonde publie ; le test le prouve en le placant derriere
// une porte de references FERMEE, puis derriere une porte OUVERTE portant deux elements — la
// valeur publiee doit etre la meme dans les deux cas.
func TestProbeSplashStaticPublishesUnconditionalField(t *testing.T) {
	const attendu = 0x0f1e2d
	flux := []struct {
		nom string
		ecr func(w *bitw)
	}{
		{"porte de references FERMEE", func(w *bitw) {
			w.put(0, 1)        // pas d'ensemble de references
			w.put(attendu, 24) // le champ inconditionnel
			w.put(0, 1)        // pas de corps
		}},
		{"porte de references OUVERTE, deux elements", func(w *bitw) {
			w.put(1, 1)  // ensemble de references present
			w.put(7, 32) // identifiant
			w.put(2, 3)  // deux elements
			w.put(3, 3)  // element de tag 3 -> R(32)
			w.put(0, 32)
			w.put(0, 3)        // element de tag 0 -> rien
			w.put(attendu, 24) // le champ inconditionnel
			w.put(0, 1)        // pas de corps
		}},
	}
	for _, f := range flux {
		clearAllHooks(t)
		var gotVals []uint64
		appels := 0
		probeHook = func(_ uint32, comp ProbeComponent, v []uint64) {
			if comp != ProbeSplashStatic {
				t.Errorf("%s : composant %v au lieu de ProbeSplashStatic", f.nom, comp)
			}
			gotVals, appels = v, appels+1
		}
		w := &bitw{}
		f.ecr(w)
		br := NewBitReader(append(w.buf, make([]byte, 16)...))
		consumeByName(br, compSplashMessageStatic, 47, 0)
		if appels != 1 {
			t.Errorf("%s : %d appel(s), 1 attendu", f.nom, appels)
			continue
		}
		if len(gotVals) != 1 || gotVals[0] != attendu {
			t.Errorf("%s : valeurs %v, [%#x] attendu", f.nom, gotVals, attendu)
		}
	}
}

// TestHookFieldStringsAreRegistryLabels — LES ETIQUETTES.
//
// `String()` doit rendre l'etiquette de REGISTRE de chaque champ, pour toutes les valeurs de
// chaque enumeration : c'est ce qui relie une valeur publiee au nom que le film lui donne. Une
// valeur oubliee dans un `switch` rendrait « champ inconnu » et le test l'attrape.
func TestHookFieldStringsAreRegistryLabels(t *testing.T) {
	for f := GameEngineField(0); f < GameEngineFieldCount; f++ {
		if s := f.String(); s == "" || s[0] != 'g' {
			t.Errorf("GameEngineField(%d).String() = %q — attendu une etiquette `game-engine-...`", f, s)
		}
	}
	for f := PlayerStateField(0); f < PlayerStateFieldCount; f++ {
		if s := f.String(); s == "" || s[0] != 'p' {
			t.Errorf("PlayerStateField(%d).String() = %q — attendu une etiquette `player-...`", f, s)
		}
	}
	for p := ProbeComponent(0); p < ProbeComponentCount; p++ {
		if s := p.String(); s == "" || s[0] < 'a' || s[0] > 'z' {
			t.Errorf("ProbeComponent(%d).String() = %q", p, s)
		}
	}
	if s := ManagedObjectBoundaryVisibility.String(); s != compManagedObjectBoundaryVisibility {
		t.Errorf("ManagedObjectField.String() = %q", s)
	}
	// Une valeur hors domaine doit se NOMMER comme telle, pas rendre une etiquette au hasard.
	if s := GameEngineField(99).String(); s == compGameEngineCurrentState {
		t.Error("GameEngineField(99) rend l'etiquette du premier champ")
	}
	if s := PlayerStateField(99).String(); s == compPlayerSoftKillTimer {
		t.Error("PlayerStateField(99) rend l'etiquette du premier champ")
	}
	if s := ProbeComponent(99).String(); s == compSplashMessageStatic {
		t.Error("ProbeComponent(99) rend l'etiquette du premier champ")
	}
}
