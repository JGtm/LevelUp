package filmdec

// dispatch_object.go — TETE DE LA CHAINE DE DISPATCH DES COMPOSANTS.
//
// # POURQUOI UNE CHAINE, ET PAS UN SEUL `switch`
//
// `consumeByName` etait un unique `switch` de 815 lignes et 194 arms dans `traverse.go`
// (1 388 lignes). Le lot 2.7 l'a coupe en SEPT maillons, par DEPLACEMENT PUR : chaque arm est
// recopie au caractere pres, seule la branche `default` change — elle passe la main au maillon
// suivant au lieu de rendre `ported=false`. L'equivalence est exacte : un arm qui rend
// `ported=false` rend TOUJOURS depuis son propre maillon (il ne tombe pas dans `default`), et
// le dernier maillon rend le `default` d'origine, mot pour mot.
//
// L'ORDRE DES ARMS EST CONSERVE, y compris entre maillons : il est celui des lots de portage
// successifs (les commentaires « Batch8 », les dates, les numeros de FUN_ s'y lisent en
// sequence), et c'est la seule trace ecrite de l'ordre dans lequel la grammaire a ete portee.
// Un regroupement par famille l'aurait detruit.
//
// La chaine, dans l'ordre :
//
//	consumeByName                            objet (i0..i17), unite-acteur, world-object
//	consumeItemAndTacmapComponent            equipement, objet pose, projectile, tacmap
//	consumePlayerAndSceneComponent           joueur (ti=5), scene, statborg, physique
//	consumeCrewFlockAndMusicComponent        equipage, nuee, musique, effets, moteur de partie
//	consumePlayerTailAndGameEngineComponent  queue joueur, joueur gere (ti=9), moteur de partie
//	consumeCaptureAndBipedComponent          composants CAPTES (obje, arme tenue, vitalites,
//	                                         etat de mort), arme, bipede, etat de simulation
//	consumeManagedAndObjectiveComponent      objet gere (ti=10/12/13), objectif (ti=11), unite
//
// # EXEMPTION DE LONGUEUR (seuil de 80 lignes, CLAUDE.md regle 5)
//
// Chaque maillon depasse 80 lignes et le restera : un maillon est une TABLE, pas un
// algorithme — un arm par composant ECS du jeu, deux a quatre lignes chacun, sans branche
// partagee a factoriser. Sa longueur est le NOMBRE DE COMPOSANTS PORTES. La seule
// « extraction » possible serait de couper la table plus fin, ce qui multiplierait les bornes
// arbitraires sans rendre un maillon plus lisible ni plus sur. Le decoupage retenu suit donc
// les familles dominantes et les frontieres de lots de portage, pas un quota de lignes.
// Cette exemption vaut pour les SEPT maillons de la chaine (fichiers `dispatch_*.go`).

// consumeByName dispatches a component to its ported bit-consumer. It returns the
// variant-name for variant-bearing components (obje, held-weapon), the captured
// object-dead-state (non-nil only for the dead-state component), and ported=false
// for components whose deser is not yet bit-exact.
//
// Premier maillon : composants d'OBJET (i0 a i17), unite-acteur, et le chemin WORLD-OBJECT.
func consumeByName(br *BitReader, name string, typeIndex uint32, level uint32) (variant uint32, dead *DeadState, ported bool) {
	variant = noVariant
	switch name {
	case "object-position-dynamic-precision-component": // i0
		consumeObjectPositionDynamicPrecisionD(br, TraversalPrecision)
		return variant, nil, true
	case "object-translational-velocity-dynamic-precision-component": // i1
		consumeObjectTranslationalVelocity(br)
		return variant, nil, true
	case "object-angular-velocity-dynamic-precision-component": // i3 de ti=40 UNIQUEMENT
		// Déser propre, résolu STATIQUEMENT le 2026-09-03 : FUN_140d87740 (et non
		// FUN_140d70998, qui est celui du composant SANS « dynamic-precision »). Chaîne de
		// preuve et validation 6/6 contre la table live : components_dynprec_orientation.go.
		// Le regroupement des deux noms dans une seule branche était la moitié de la cause
		// racine du blocage d'i4 sur ti=40 (l'autre moitié étant i2).
		consumeObjectAngularVelocityDynPrec(br)
		return variant, nil, true
	case "object-angular-velocity-component": // i3 du bipède (ti=35) et de la plupart des archétypes
		// VRAI deser biped i3 (table ECS live) = FUN_140d70998 = FUN_14076d528 = un SEUL gate
		// dynamic-precision : R(1) present ; si 0 -> R(19) dir + R(8) magnitude. L'ancien routage
		// (consumeObjectAngularVelocity=FUN_140d87740) ajoutait un gate EXTERNE parasite + R(96)
		// keep -> sur-lecture de 96 bits qui engloutissait les records des joueurs suivants
		// (oracle: 1 seul biped-delta/frame au lieu de ~8). Validé par tmp_framedump (biped-deltas
		// par frame).
		//
		// LA BASCULE `useLegacyAngularVel` A DISPARU le 2026-09-05 (lot E, item E.2) : son
		// drapeau valait `false` et n'avait aucun installateur (le setter n'avait pas
		// d'appelant), donc la branche « ancien comportement » était inatteignable. Le
		// désérialiseur qu'elle appelait, `consumeObjectAngularVelocity`, RESTE : il est le
		// déser CORRECT d'i3 pour ti=40, via `consumeObjectAngularVelocityDynPrec`.
		consumeDynPrecVec3(br, angularMagBits, angularScaleBits)
		return variant, nil, true
	case "object-region-state-component": // i6
		consumeObjectRegionState(br)
		return variant, nil, true
	case "object-damage-sections-component": // i7
		consumeObjectDamageSections(br)
		return variant, nil, true
	case "object-constraint-component": // i8
		consumeObjectConstraint(br)
		return variant, nil, true
	case "object-parent-state-component": // i10 (1st desync on typeIndex=40)
		consumeObjectParentState(br, paramForComponent(name), typeIndex)
		return variant, nil, true
	case "object-scale-component": // i12
		consumeObjectScale(br)
		return variant, nil, true
	case "object-maximum-vitalities-component": // i13
		consumeObjectMaximumVitalities(br)
		return variant, nil, true
	case compObjectDissolver: // i14
		consumeObjectDissolver(br)
		return variant, nil, true
	case "object-low-frequency-component": // i15 = FUN_1407ef088 (validé, matche la table live)
		consumeObjectLowFrequency(br)
		return variant, nil, true
	case "object-physics-flags-component": // i16
		consumeObjectPhysicsFlags(br)
		return variant, nil, true
	case "object-frame-configuration-component": // i17
		consumeObjectFrameConfiguration(br)
		return variant, nil, true
	case "unit-actor-control-component":
		consumeUnitActorControl(br, paramForComponent(name))
		return variant, nil, true
	case "unit-actor-state-component":
		consumeUnitActorState(br, paramForComponent(name))
		return variant, nil, true
	case "unit-malleable-property-component":
		consumeUnitMalleableProperty(br, paramForComponent(name))
		return variant, nil, true
	case "biped-spartan-ability-malleable-property-component": // i58 (FUN_140fea4c0)
		consumeBipedSpartanAbilityMalleableProperty(br)
		return variant, nil, true
	case compObjectPosition: // world-object i0 (FUN_14076e29c)
		// Structure RE bit-exacte (FUN_14076e420 precHigh R(1) -> FUN_14076e524 index-sel+index+3axes
		// / FUN_141f85880 AABB ; FUN_14076e3e4 handle-tail gated precHigh ; FUN_14076e304 R(2) finite).
		//
		// DÉCOUPAGE CORRIGÉ le 2026-07-26 — et c'est le bug le plus retors rencontré ici.
		// L'ancien portage lisait `R(2)` d'index puis 13/13/13. Le vrai découpage est `R(1)`
		// d'index puis 13/13/**14**. LE TOTAL EST LE MÊME (45 bits) : aucun test de longueur,
		// aucune mesure de désynchronisation ne pouvait le voir. Seule une mesure de CONTENU le
		// révèle — ici le profil de bascule bit à bit sur 6 794 paires de records consécutifs
		// d'une même entité, qui place les frontières à 16 / 29 / 43.
		//
		// L'ancien commentaire affirmait « largeurs world-object UNIVERSELLES, PAS map-specific ».
		// C'est RÉFUTÉ : ce sont les largeurs DE LA CARTE, les mêmes que celles du bipède
		// (13/13/14 sur Cliffhanger), et les bornes de déquantification sont le même AABB de BSP.
		// Seule la PORTE diffère du bipède : 2 + largeur d'index de région ici (precHigh + index-sel + index, soit 3 bits sur 78 cartes et 4 sur Live Fire — lot BB.2, 2026-09-12)
		// contre 5 là-bas. C'est ce qui explique la contradiction « i0 45 vs 47 bits » qui
		// traînait dans les notes : ce ne sont pas deux mesures du même champ, ce sont deux
		// archétypes différents.
		if br.ReadBit() { // precHigh (FUN_14076e420 R(1))
			br.ReadBits(59) // precHigh=1 : FUN_141f85880 AABB + handle-tail + R(2) (total 60 mesuré)
		} else {
			if !br.ReadBit() { // FUN_14076e524 index-sel ; si 0 -> lit l'index de région
				br.ReadBits(WorldObjectPrecision.IndexW)
			}
			for a := 0; a < 3; a++ {
				br.ReadBits(WorldObjectPrecision.AxisW[a]) // FUN_140cc5128 axe a
			}
			br.ReadBits(2) // FUN_14076e304 R(2) finite (handle-tail = 0 bit quand precHigh=0)
		}
		return variant, nil, true
	case "object-translational-velocity-component": // world-object i1 (FUN_14076e228)
		consume14076d528(br) // R(1)[+R(19)+R(10)]
		return variant, nil, true
	case "object-forward-and-up-dynamic-precision-component": // i2 de ti=38/39/40/43
		// N'EST PAS le déser du bipède. Résolu statiquement le 2026-09-03 :
		// FUN_140c5f7ec (le bipède porte `object-forward-and-up-component` ->
		// FUN_14076e278). Le « reuse biped i2 deser » qui tenait ici était une
		// réutilisation héritée de ti=38, jamais mesurée (CADRAGE_VEHICULES § 2), et
		// elle amputait i2 de son ou ses bits de tête sur TOUS les records ti=40.
		return variant, nil, consumeObjectForwardAndUpDynPrec(br, paramForComponent(name))
	default:
		return consumeItemAndTacmapComponent(br, name, typeIndex, level)
	}
}
