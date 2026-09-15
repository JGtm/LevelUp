package filmdec

// keyframe_fullstate_loop.go — LA BOUCLE D'ETAT COMPLET DU JEU, PORTEE, ET DEPUIS LE LOT 1.4
// (2026-09-14) LA SEULE LECTURE DU CORPS D'UN RECORD D'IMAGE-CLE EN PRODUCTION.
//
// D'OU ELLE VIENT. Le lot R7-d a trouve, en deroulant la piste du lot R7-c, la chaine que le
// jeu emprunte pour lire un ETAT COMPLET (et non un delta) :
//
//	FUN_142e2bfd0   en-tete PAR ENTITE, puis l'etat par defaut et la boucle
//	  -> FUN_1428e2b68   recupere le descripteur d'archetype et la TABLE de 64 entrees
//	    -> FUN_142e2c690  LA BOUCLE : 64 entrees nommees, AUCUN masque de presence
//
// Detail dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section 6.
//
// CE QUI EST PORTE, ET SA PREUVE (adresses relues le 2026-08-17, Ghidra lecture seule) :
//
//	FUN_142e2bfd0 : R(32) id · R(32) typeIndex · R(32) · R(4) (FUN_142e29cf8) · R(8)
//	                = 108 bits d'en-tete, puis R(32) n1 (> 0 -> etat par defaut vtable[0x60],
//	                suivi d'un R(32) de controle si le drapeau film est mis), puis R(32) n2
//	                (> 0 -> vtable[0x88], 0 bit, puis la boucle de composants).
//	FUN_142e2c690 : pour k de 0 a 63, si l'entree k de la table porte un NOM, deserialiser
//	                via vtable[0x28] avec le niveau lu en `entree + 0x100`, puis, si le
//	                drapeau film est mis, R(1) et, si ce bit est mis, R(32).
//
// IL N'Y A PLUS D'OPTION DE CADRE (lot 1.4, decision D8 du PLAN_DECODEUR_FILM). `WalkKeyframeFullState`
// prend le payload, le bit du record et le registre, et rien d'autre : l'en-tete de 108 bits, les
// deux mots de taille et l'etat par defaut sont CE QUE LE JEU LIT, pas des reglages. Une bascule
// qui ne peut plus qu'introduire un cadre faux n'est pas une variable, c'est un piege — meme
// arbitrage qu'au lot 1.2 pour `shiftArchetypeLevels`.
//
// HORS LIGNE — jamais depuis un chemin de requete. Aucune ecriture, aucun schema.

// keyframeFullStateHeaderBits est l'en-tete PAR ENTITE d'un etat complet tel que
// `FUN_142e2bfd0` le lit : `R(32)` id, `R(32)` typeIndex, `R(32)`, `R(4)`, `R(8)`.
//
// Il ne CONTREDIT pas l'en-tete de 64 bits `[id:32][field:26][ti:6]` valide par R3/R5 : le
// balayeur oracle (`kfValidAnchor`) n'accepte une ancre que si le mot de 32 bits a `q+32`
// vaut moins de 50, ce qui veut dire `field26 == 0` sous une lecture et `typeIndex < 50`
// sous l'autre. Les deux sont INDISCERNABLES sur les ancres acceptees, et les 6 bits de
// `typeIndex` lus en `+58` valent la meme chose dans les deux cas.
const keyframeFullStateHeaderBits = 108

// keyframeFullStateSizeBits est la largeur des deux mots de taille que `FUN_142e2bfd0` lit
// autour de l'etat par defaut (`n1` avant, `n2` apres) : ce sont des comptes testes `> 0`,
// pas des longueurs de saut.
const keyframeFullStateSizeBits = 32

// keyframeFullStateTemoin est LE BOUTON DES DEUX TEMOINS NEGATIFS, ET RIEN D'AUTRE.
//
// Il n'est PAS une option de la lecture : il est NON EXPORTE, il n'a aucun appelant hors des
// instruments de mesure (garde-rail : `keyframe_fullstate_loop_guard_test.go`), et la lecture
// de production passe par `WalkKeyframeFullState`, qui ne l'expose pas. Il existe parce que la
// regle 4 de `METHODE_RETRO_INGENIERIE_FILM.md` INTERDIT de calculer un plancher de faux
// positifs sur ce flux : il faut le MESURER, donc pouvoir jouer la meme lecture a cote de la
// bonne. Les deux temoins qui s'en servent sont nommes :
//
//	TEMOIN DE HASARD (`EnTeteBits = 109`) — la meme lecture, l'en-tete decale d'UN bit.
//	    Il vaut 391 fermetures sur 62 686 records (0,6 %) contre 19 337 (30,8 %) pour la bonne
//	    forme, mesure du 2026-09-14 : c'est ce chiffre qui rend le taux de fermeture lisible.
//	    Instrument : `imagecle_fermeture_research_test.go` (colonne `etat-complet+1bit`).
//	ORACLE `n2` (`SansEtatParDefaut`, `EnTeteBits = 108 + w`) — l'etat par defaut REMPLACE par
//	    un decalage de `w` bits, pour MESURER la largeur d'un etat par defaut qu'aucun
//	    deserialiseur ne porte encore. Instrument : `imagecle_oracle_n2_research_test.go`.
//	    C'est la chaine qui a donne les cinq largeurs du lot 1.3.
type keyframeFullStateTemoin struct {
	// EnTeteBits remplace la largeur d'en-tete par entite. 0 = celle du jeu (108).
	EnTeteBits int
	// SansEtatParDefaut saute le deserialiseur d'etat par defaut de l'archetype ; le decalage
	// qui le remplace est porte par `EnTeteBits`.
	SansEtatParDefaut bool
}

// WalkKeyframeFullState rejoue le corps d'un record d'image-cle par la boucle d'ETAT COMPLET
// du jeu, en partant du premier bit du record (`recBit`). C'est LA lecture de production des
// records d'image-cle depuis le lot 1.4.
//
// Il REUTILISE la boucle de composants de production (`traverseComponentLoop`) et les
// deserialiseurs d'etat par defaut : rien n'est recopie, seul le CADRE change par rapport au
// record NEW du chemin delta.
//
// Les bascules globales de grammaire (`filmComponentCorruptionCheck`, `simStateComplete`,
// `keyframeWriterI0Grammar`, ...) sont celles du process : l'appelant les regle et detient
// `LockProcessDecode`.
func WalkKeyframeFullState(pay []byte, recBit int, reg *Registry) EntityTrace {
	return walkKeyframeFullState(pay, recBit, reg, keyframeFullStateTemoin{})
}

// walkKeyframeFullState est la marche, avec le bouton des temoins. Les instruments l'appellent
// avec un temoin nomme ; la production passe par `WalkKeyframeFullState`, donc par le temoin nul.
func walkKeyframeFullState(pay []byte, recBit int, reg *Registry, tem keyframeFullStateTemoin) EntityTrace {
	hdr := tem.EnTeteBits
	if hdr <= 0 {
		hdr = keyframeFullStateHeaderBits
	}
	t := EntityTrace{DesyncAt: -1}
	// Le typeIndex se lit aux 6 bits de queue du deuxieme mot de 32 bits, position
	// commune aux deux lectures d'en-tete (cf. `keyframeFullStateHeaderBits`).
	t.TypeIndex = uint32(kfReadBits(pay, recBit+keyframeRecordTIBit, 6))
	br := NewBitReader(pay)
	br.SetBitPos(recBit + hdr)
	if t.TypeIndex >= objectArchetypeCount {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	if !consumeFullStateDefaultBlock(br, t.TypeIndex, tem.SansEtatParDefaut) {
		// `n2 == 0` : le jeu NE LANCE PAS la boucle de composants (FUN_142e2bfd0, le
		// `if (0 < (int)uVar7)` qui garde `vtable[0x88]` et `FUN_1428e2b68`). Le record
		// s'arrete donc sur son second mot de taille.
		t.EndBit = br.BitPos()
		return t
	}
	t.Mask = ^uint64(0) // etat complet : aucun masque de presence, tous les composants presents
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// consumeFullStateDefaultBlock joue ce que `FUN_142e2bfd0` lit ENTRE l'en-tete par entite et la
// boucle de composants : `R(32) n1`, l'etat par defaut, le mot de controle du mode film, puis
// `R(32) n2`. Il rend VRAI quand la boucle de composants doit suivre.
//
// LES DEUX MOTS DE TAILLE SONT DES GARDES, PAS DE SIMPLES COMPTES — RELU LE 2026-09-15
// (lot 1.9.1 bis, pas 2 quater). Le decompile de `FUN_142e2bfd0` porte DEUX fois le meme motif
// `if (0 < (int)uVar7)` : le premier garde l'appel a `vtable[0x60]` (l'etat par defaut), le
// second garde `vtable[0x88]` PUIS `FUN_1428e2b68`, qui mene a la boucle de composants
// (`FUN_142e2c690`). Autrement dit : `n1 == 0` -> AUCUN etat par defaut n'est ecrit, et
// `n2 == 0` -> AUCUN composant ne l'est.
//
// LE DEPOT LISAIT LES DEUX INCONDITIONNELLEMENT, et cela se voyait ailleurs sans etre compris :
// `default_state_n2_constant_test.go` ecarte « trois a quinze records par groupe dont le `n1`
// s'ecarte du modal (en pratique 0) » en les appelant des ANCRES FORTUITES. L'ecrivain dit
// qu'un `n1` nul est un record LEGITIME sans etat par defaut.
func consumeFullStateDefaultBlock(br *BitReader, ti uint32, sansEtatParDefaut bool) bool {
	n1 := int32(br.ReadBits(keyframeFullStateSizeBits)) //nolint:gosec // 32 bits lus, compares SIGNES
	if !sansEtatParDefaut && n1 > 0 {                   // FUN_142e2bfd0 : `if (0 < (int)uVar7)`, comparaison SIGNEE
		consumeKeyframeDefaultState(br, ti)
		if filmComponentCorruptionCheck {
			// FUN_142e2bfd0 : mot de controle INCONDITIONNEL (pas de R(1) de garde ici,
			// contrairement au controle PAR COMPOSANT de FUN_142e2c690).
			br.ReadBits(keyframeFullStateSizeBits)
		}
	}
	// n2 : meme comparaison SIGNEE ; > 0 => vtable[0x88] puis la boucle de composants.
	return int32(br.ReadBits(keyframeFullStateSizeBits)) > 0 //nolint:gosec // idem
}
