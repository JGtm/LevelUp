package grammar

// default_state_arch.go — DEFAULT-STATE PAR ARCHETYPE (vtable[0x60]), porte 100% STATIQUEMENT.
//
// CHAINE DE RESOLUTION (verifiee, reproductible, aucune capture live) :
//
//	registrar d'archetypes FUN_140e453b4
//	  -> FUN_140e45fc4(world, ti, &descripteur)   (ti 0x00..0x31, COUNT = 0x32 = 50)
//	  -> le descripteur est un objet .data dont la vtable est installee au RUNTIME par un
//	     constructeur : la trouver via l'unique xref [WRITE] sur l'adresse du descripteur,
//	     puis lire le LEA qui precede (`48 8d 05 disp32` -> vtable)
//	  -> deserialiseur du default-state = *(vtable + 0x60)
//	     (c'est l'appel `(**(code **)(*plVar2 + 0x60))(plVar2, size, dst, bitreader, 1)`
//	      de FUN_1408f1aa4, le lecteur de record NEW)
//
// CONTROLE DE LA CHAINE : ti35 (biped) -> descripteur 1446e14f0 -> vtable 143737178
// -> *(143737178+0x60) = 140f44c38 = FUN_140F44C38, le deser biped deja valide bit-exact
// en live (cf. default_state.go). La chaine est donc juste sur le seul cas verifiable.
//
// STUB = FUN_1408d8220 (`return 1;`, 0 bit). Les archetypes qui le portent consomment
// reellement 0 bit : pour eux le decodeur avait deja raison, et leurs deraillements de
// record NEW viennent des COMPOSANTS, pas du default-state.
// Stubs mesures : ti0, ti1, ti2, ti4, ti7, ti15, ti16, ti18, ti19, ti22, ti25, ti26, ti27,
// ti30, ti31, ti32, ti33, ti34, ti45, ti46.
//
// TI14 N'EST PAS UN STUB — correction du 2026-09-14 (lot 1.3). Cette liste portait
// « + ti14 = FUN_140467a20, un `return;` partage », et c'etait FAUX de deux facons :
//
//	(a) `vtable[0x60]` de ti14 vaut `0x140FED6F4` (KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md,
//	    classe REAL), et ce descripteur decompile en `V ; R(5)` — 6 bits, pas 0 ;
//	(b) `FUN_140467a20` n'est pas le deserialiseur de ti14 : c'est un `return;` PARTAGE par
//	    treize symboles exportes sans rapport (`AK::MemoryMgr::GetCategoryStats`,
//	    `ManagedDebug_LogError`, `Variant_InitializeStaticScriptComponents`, ...). Une
//	    resolution qui atterrit sur ce thunk n'a pas trouve un stub : elle s'est trompee
//	    de table.
//
// Les deux chaines de la mesure (releve B.2 de `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`) le
// confirment : `n2` devient constant (28) a 6 bits, et la fermeture passe de 0 a 5 024 records
// sur 5 024, sur trois builds. Relu chez l'ecrivain le 2026-09-14 (Ghidra, base 0x140000000).
//
// PREFIXE COMMUN « version » (note V ci-dessous), present en tete de la quasi-totalite des
// deserialiseurs : `R(1) gate ; si 1 -> R(8)`. C'est exactement le prologue du biped
// (`uVar10 = 13 ; if R(1) then uVar10 = R(8)`).
//
// Chaque grammaire ci-dessous est bit-exacte : toute feuille fait `*(reader+0x2c) += N`.
// Un archetype dont UNE largeur de feuille n'est pas etablie statiquement n'est PAS inscrit
// dans la table (on ne devine pas : un skip faux vaut mieux mesure qu'un skip invente).

// consumeVersionPrefix porte le prologue « version » commun : R(1) gate ; si 1 -> R(8).
func consumeVersionPrefix(br *Lecteur) {
	if br.ReadBit() {
		br.ReadBits(8)
	}
}

// defaultStateDeserByTI : deserialiseur du default-state par typeIndex d'archetype.
// ti35 (biped) est traite a part dans TraverseEntity (consumeBipedDefaultState).
//
// ABSENT = 0 BIT, ET C'EST UN REPLI NOMME (D14 du PLAN_DECODEUR_FILM, ADR 0034 D-10).
// Deux populations tres differentes tombent dans la meme branche « pas d'entree » :
//
//	(a) les STUBS mesures (liste en tete de fichier) : leur `vtable[0x60]` est
//	    `FUN_1408d8220`, qui consomme reellement 0 bit. Ce n'est PAS un repli, c'est la
//	    grammaire du jeu ;
//	(b) les archetypes dont la grammaire n'est pas entierement resolue — au 2026-09-14 :
//	    ti23 (`0x142EEA440`), ti40 (`0x1410A5A74`, cf. default_state_ti40.go), ti41
//	    (`0x1408EFB58`), ti44 (`0x142EEA020`). Pour eux, 0 bit est un REPLI : il decale tout
//	    ce qui suit dans le record, et il ne se defend que parce qu'un skip faux se mesure
//	    mieux qu'un skip invente (cf. « on ne devine pas », en tete de ce fichier).
//
// POSE : 2026-07-31 (`3f0ec70b3`, premiere version du fichier — la phrase « absent = 0 bit »
// y est deja). CRITERE DE RETRAIT : un archetype sort du repli des que son `vtable[0x60]` est
// relu chez l'ecrivain — la table n'admet pas une largeur devinee. COMPTAGE : la fermeture par
// archetype (`testdata/keyframe_closure.golden`, lot 0.A.3) dit, bobine par bobine, ce que
// chaque repli coute ; le repli est retire quand la grammaire est lue, pas quand le compte est
// bas.
var defaultStateDeserByTI = map[uint32]func(*Lecteur){
	3:  consumeDefaultStateTI3,
	5:  consumeDefaultStateTI5,
	6:  consumeDefaultStateTI6,
	8:  consumeDefaultStateTI8,
	9:  consumeDefaultStateTI9,
	10: consumeDefaultStateTI10,
	11: consumeVersionPrefix, // FUN_14110d4d8 : V seul
	12: consumeVersionPrefix, // FUN_1410ed0e8 : V seul
	13: consumeDefaultStateTI13,
	14: consumeDefaultStateTI14, // FUN_140fed6f4 : V ; R(5) — relu 2026-09-14 (lot 1.3)
	17: consumeDefaultStateTI17, // FUN_14101a0a4 : V ; R(7) — relu 2026-09-14 (lot 1.3)
	20: consumeVersionPrefix,    // FUN_142eea600 : V seul
	21: consumeDefaultStateTI21, // FUN_141133c24 : R(18) SEC — relu 2026-09-14 (lot 1.3)
	24: consumeDefaultStateTI24,
	28: consumeDefaultStateTI28,
	29: consumeVersionPrefix, // FUN_14116f514 : V seul — relu 2026-09-14 (lot 1.3)
	36: consumeDefaultStateTI36,
	37: consumeDefaultStateTI37,
	38: consumeDefaultStateTI38,
	39: consumeDefaultStateTI38, // meme deser FUN_1408f0b48 que ti38
	42: consumeDefaultStateTI42, // FUN_1407f0c68 (default_state_ti42.go) — VALIDE PAR ORACLE
	43: consumeDefaultStateTI36, // FUN_140fe7630 : meme forme V + MPP que ti36
	47: consumeDefaultStateTI14, // FUN_1410f44f8 : meme forme V + R(5) que ti14 (lot 1.3)
	48: consumeDefaultStateTI48,
	49: consumeVersionPrefix, // FUN_141fd39c0 : V seul
}

// consumeDefaultStateTI3 porte FUN_142eea28c (archetype 3, « low-frequency ») :
//
//	V ; FUN_1408f0ac4(dst, br, 0) ; FUN_1407f08bc [R(1) ; si 1 R(8)] ;
//	FUN_142af28d8 [R(1)] ; R(8)
func consumeDefaultStateTI3(br *Lecteur) {
	consumeVersionPrefix(br)
	consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0) @142eea359
	consumeGateR(br, 8)     // FUN_1407f08bc -> FUN_1407f08f8 = R(8)
	br.ReadBit()            // FUN_142af28d8 = R(1)
	br.ReadBits(8)          // R(8) terminal -> dst+0xb
}

// consumeDefaultStateTI5 porte FUN_140fed600 (archetype 5, « player-waypoint ») :
// V ; R(6) (index de joueur, valide < 0x20 par la valeur de retour).
func consumeDefaultStateTI5(br *Lecteur) {
	consumeVersionPrefix(br)
	br.ReadBits(6)
}

// consumeDefaultStateTI6 porte FUN_140ff7f44 (archetype 6, « statborg ») : R(6) SEC,
// sans prefixe de version (valide < 0x30 par la valeur de retour).
func consumeDefaultStateTI6(br *Lecteur) { br.ReadBits(6) }

// consumeDefaultStateTI8 porte FUN_142f14688 (archetype 8, sans composant) :
//
//	g = R(1)
//	si g : v = R(8) ; si v > 1 -> R(16) et FIN
//	sinon (ou v <= 1) : R(8)
func consumeDefaultStateTI8(br *Lecteur) {
	if br.ReadBit() {
		if v := br.ReadBits(8); v > 1 {
			br.ReadBits(16)
			return
		}
	}
	br.ReadBits(8)
}

// consumeDefaultStateTI9 porte FUN_1410d7540 (archetype 9, « managed-player ») :
// V ; R(6) ; R(6) ; R(1). Elle JETTE ce que [readManagedPlayerDefaultState] publie : la table
// `defaultStateDeserByTI` n'a qu'une signature, et le seul lecteur qui ait besoin de la valeur
// (l'equipe, lot 1.7) appelle l'autre.
func consumeDefaultStateTI9(br *Lecteur) { readManagedPlayerDefaultState(br) }

// readManagedPlayerDefaultState porte la MEME grammaire et REND le premier `R(6)`.
//
// CE PREMIER CHAMP EST L'INDEX DE JOUEUR DE L'ENTITE, et c'est MESURE, pas suppose (lot 1.7,
// 2026-09-14, 18 films et 7 builds) : il vaut exactement le rang du siege de la table de
// `chunk_00` pour chaque entite presente au premier paquet d'image-cle (8/8, 23/23, 24/24 selon
// le film), il est CONSTANT sur toute la vie de l'entite, et il continue au-dela des sieges pour
// les joueurs arrives en cours de partie. Le controle qui interdit d'y lire un simple ORDINAL :
// sur `50247b26` la suite lue est `0 1 3 4 ... 22 24` — trouee, donc pas un rang de parcours.
// Meme forme que le `R(6)` de ti=5 (`player-waypoint`), que l'executable borne a `< 0x20`.
//
// La largeur du champ n'est pas devinee : elle vient de `FUN_1410d7540`, comme les deux autres.
func readManagedPlayerDefaultState(br *Lecteur) (playerIndex int) {
	consumeVersionPrefix(br)
	playerIndex = int(br.ReadBits(6))
	br.ReadBits(6)
	br.ReadBit()
	return playerIndex
}

// consumeDefaultStateTI10 porte FUN_141020244 (archetype 10, « managed-object ») :
// V ; FUN_1408f0ac4(dst, br, 0).
func consumeDefaultStateTI10(br *Lecteur) {
	consumeVersionPrefix(br)
	consume1408f0ac4(br, 0)
}

// consumeDefaultStateTI13 porte FUN_140ce55e8 (archetype 13, « managed-object-property-name ») :
//
//	V ; FUN_14080dec4 "propertyName" = R(32) ;
//	g = R(1) ; si g == 0 -> 1 x FUN_140ce59bc [R(4)] ; sinon 32 x FUN_140ce59bc.
func consumeDefaultStateTI13(br *Lecteur) {
	consumeVersionPrefix(br)
	br.ReadBits(32) // FUN_14080dec4 "propertyName"
	n := 1
	if br.ReadBit() {
		n = 32
	}
	for i := 0; i < n; i++ {
		br.ReadBits(4) // FUN_140ce59bc = R(4)
	}
}

// consumeDefaultStateTI14 porte FUN_140fed6f4 (archetype 14, `crew-order-component` en i0) :
// V ; R(5). Le prefixe de version y est le MEME appel qu'ailleurs (`FUN_1406cf008`, relu le
// 2026-09-14 : R(1) sec), et la feuille terminale fait `*(reader+0x2c) += 5`, validee
// `< 0x20` par la valeur de retour — exactement la forme de ti5, a la largeur pres.
//
// FUN_1410f44f8 (ti47) a la meme forme, au bit pres, et partage donc ce porteur.
func consumeDefaultStateTI14(br *Lecteur) {
	consumeVersionPrefix(br)
	br.ReadBits(5)
}

// consumeDefaultStateTI17 porte FUN_14101a0a4 (archetype 17) : V ; R(7).
// La feuille fait `*(reader+0x2c) += 7` et la fonction rend 1 sans borner la valeur.
func consumeDefaultStateTI17(br *Lecteur) {
	consumeVersionPrefix(br)
	br.ReadBits(7)
}

// consumeDefaultStateTI21 porte FUN_141133c24 (archetype 21, `flock-*-component`) :
// un unique `R(0x12)` = R(18), SANS prefixe de version. C'est le seul des cinq etats du lot
// 1.3 a ne pas commencer par `FUN_1406cf008` : le decompile n'appelle rien avant sa feuille.
func consumeDefaultStateTI21(br *Lecteur) { br.ReadBits(18) }

// consumeDefaultStateTI24 porte FUN_142eea5b4 (archetype 24, « state-checksum ») :
// FUN_1406d00ec [R(1) ; si 0 -> R(2)] ; FUN_140c1e31c [R(3)]. Pas de prefixe de version.
func consumeDefaultStateTI24(br *Lecteur) {
	consumeID2(br)
	br.ReadBits(3) // FUN_140c1e31c = R(3)
}

// consumeDefaultStateTI28 porte FUN_142eea4d8 (archetype 28, « narrative-moment ») : R(32) sec.
func consumeDefaultStateTI28(br *Lecteur) { br.ReadBits(32) }

// consumeDefaultStateTI36 porte FUN_1407f2224 (archetype 36, « object-position ») :
// V ; FUN_14080cfe8 (bloc object-multiplayer-properties, deja porte bit-exact).
// FUN_140fe7630 (ti43) a exactement la meme forme.
func consumeDefaultStateTI36(br *Lecteur) {
	consumeVersionPrefix(br)
	consumeMultiplayerPropertiesBlock(br)
}

// consumeDefaultStateTI37 porte FUN_1407f105c (archetype 37, « equipment ») :
//
//	V ; FUN_1407f2224 (= le deser de ti36, avec SON propre prefixe V) ;
//	ECS_ReadEntityRefIndex5 (FUN_1407f2058 = R(1) ; si 0 -> R(5)) ;
//	g = R(1) ; si g -> R(32) "ability-enabled-id".
//
// La sortie anticipee de FUN_1407f105c depend de la valeur de retour de FUN_1407f2224,
// qui ne vaut 0 qu'en depassement de buffer : le chemin nominal lit toujours la suite.
//
// LES DEUX DERNIERES FEUILLES PUBLIENT leur valeur (equipment_creation.go) au lieu de la
// jeter — meme correction qu'i48 le 2026-08-14 et que les quatre champs d'equipment_state.go
// le 2026-08-16. Les largeurs sont INCHANGEES : `consumeGate0R(br, 5)` et
// `consumeGateR(br, 32)` sont deroules a l'identique, porte comprise.
func consumeDefaultStateTI37(br *Lecteur) {
	consumeVersionPrefix(br)
	consumeDefaultStateTI36(br)
	if !br.ReadBit() { // ECS_ReadEntityRefIndex5 = consumeGate0R(br, 5) : porte INVERSEE
		br.obs.publishEquipmentCreation(EquipCreationRef, br.ReadBits(5), true)
	} else {
		br.obs.publishEquipmentCreation(EquipCreationRef, 0, false)
	}
	if br.ReadBit() { // FUN_14080dec4 "ability-enabled-id" = consumeGateR(br, 32)
		br.obs.publishEquipmentCreation(EquipCreationAbilityID, br.ReadBits(32), true)
	} else {
		br.obs.publishEquipmentCreation(EquipCreationAbilityID, 0, false)
	}
}

// consumeDefaultStateTI38 porte FUN_1408f0b48 (archetypes 38 ET 39, « object-position ») :
// V ; FUN_14080cfe8 (MPP) ; FUN_1408f0ac4(dst+0x60, br, 0).
func consumeDefaultStateTI38(br *Lecteur) {
	consumeVersionPrefix(br)
	consumeMultiplayerPropertiesBlock(br)
	consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0) @1408f0bae
}

// consumeDefaultStateTI48 porte FUN_142f14668 (archetype 48, « forge-player-data ») :
// ECS_ReadEntityRefIndex5 seul (FUN_1407f2058 = R(1) ; si 0 -> R(5)).
func consumeDefaultStateTI48(br *Lecteur) { consumeGate0R(br, 5) }

// EtatParDefautPorte dit si la grammaire du depot porte le deserialiseur d etat par defaut de
// l archetype `ti`. Faux = 0 bit consomme, et c est soit un STUB du jeu, soit un REPLI
// (cf. l en-tete de ce fichier, qui les distingue population par population).
//
// C ETAIT UNE METHODE DE `profile.KeyframeProfile` JUSQU AU LOT 2.5.b, et elle ne lisait jamais son
// recepteur : elle interroge la table ci-dessus, qui dit ce que CE DEPOT sait decoder — de la
// grammaire, pas une valeur de profil. Le type est descendu en `profile`, la question est restee.
func EtatParDefautPorte(ti uint32) bool {
	if ti == BipedTypeIndex {
		return true // traite a part par TraverseEntity (consumeBipedDefaultState)
	}
	_, ok := defaultStateDeserByTI[ti]
	return ok
}
