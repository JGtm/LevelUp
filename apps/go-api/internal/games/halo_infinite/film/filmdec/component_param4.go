package filmdec

// component_param4.go — LE param_4 DU MOTEUR, PAR COMPOSANT.
//
// Sorti de `traverse.go` par deplacement pur au lot 2.7 (scission des fichiers de plus de
// 500 lignes) : aucune ligne de logique n'a change. Ce fichier porte la seule chose dont il
// parle — la propriete externe `param_4` que le descripteur d'un composant rend a
// FUN_14076cb60, et que quatre desers (i10, i19, i20, i23) lisent comme une largeur.

// recordStateParam is the engine's per-component param_4 (the actor-tick /
// weapon-set count returned by the descriptor's vtable[0] in FUN_14076cb60). It is
// NOT read from the bitstream: it is an external descriptor property computed per
// component before its deser runs. Three biped components branch on it:
//
//	unit-actor-control  (i19): gates the optional 2nd slot id (param_4 > 1).
//	unit-actor-state    (i20): selects the width-table read (1->8,2->10,3->11,>=4->12,
//	                           default 12 when param_4 == 0).
//	unit-malleable-property (i23): gates a few R(1) flags (param_4 > 2, > 3).
//
// The default (0) is the conservative shape: actor-state reads width 12, and the
// optional slot / flag reads are absent. When a real descriptor count is known it
// must be supplied (it cannot be recovered from the bits alone).
//
// Exposed as a package var (not const) so a calibration harness can sweep {0,1,2,3}
// to find the value that re-synchronises the component AFTER a recordStateParam-
// dependent one (i10 object-parent-state, i19/i20/i23 unit-*). Verdict from the
// deser-fix workflow: i10's keyframe desync is THIS parameter, not a code bug.
var recordStateParam uint32 = 0

// SetRecordStateParam lets a harness sweep the runtime actor-tick/weapon-set count
// (param_4) that varies the bit width of i10/i19/i20/i23. See recordStateParam.
func SetRecordStateParam(v uint32) { recordStateParam, recordStateParamOverride = v, true }

// paramByComponent porte le VRAI param_4, par composant, tel que la capture live le
// mesure (colonne `param4` de .ai/V7.5/dumps/ce_capture_delta.csv). Extraction sur
// l'archétype bipède (ti=35) : la valeur est CONSTANTE pour un composant donné —
// aucun composant n'y présente deux valeurs sur 464 010 mesures.
//
//	i13 object-maximum-vitalities  -> 3       i23 unit-malleable-property     -> 4
//	i15 object-low-frequency       -> 2       i43..i46 weapon-state-type-info -> 2
//	i17 object-frame-configuration -> 0       i53 biped-malleable-property    -> 2
//	i18 unit-control               -> 2       i57/i59 biped-spartan-ability*  -> 2
//	tout le reste                  -> 1
//
// AVANT ce correctif, une SEULE variable globale valant 0 servait tous les composants —
// c'est-à-dire la valeur juste pour le seul i17, et fausse pour tous les autres.
// La table ne change QUE les composants dont le déser branche sur param_4 (i10, i19,
// i20, i23 et flock-destination) ; elle est neutre partout ailleurs.
var paramByComponent = map[string]uint32{
	"object-maximum-vitalities-component":  3,
	"object-low-frequency-component":       2,
	"object-frame-configuration-component": 0,
	"unit-control-component":               2,
	"unit-malleable-property-component":    4,
	compWeaponStateTypeInfo:                2,
	"biped-malleable-property":             2,
	"biped-malleable-property-component":   2,
	abilityPredictedNameAlt:                2,
	abilityPredictedName:                   2,
	// i2 dyn.-prec. (ti=38/39/40/43) : arg5 de FUN_140c5f7ec. >= 2 fait lire un bit de
	// porte C supplémentaire qui, posé, bascule la charge utile sur FUN_142e29bac
	// (R(1)[+R(30)] + R(30)) au lieu de FUN_140c5fa84. MESURÉ le 2026-09-03 sur la bande
	// ti=40 de `0d76e8f1` et `fccc61cd` : avec param=1 l'histogramme des quanta i4 qui
	// suit reste étalé (27,7 % / 39,3 % de quanta au-dessus de 192) ; avec param=2 il se
	// concentre au plein — 93,6 % / 98,5 %, au-dessus même du témoin bipède (82,9 %
	// / 86,2 %), et sans perdre un seul record (1249/1249 et 201/201 atteignent i4).
	// Même nature de mesure que la clé i59 ci-dessous.
	compForwardUpDynPrec: 2,
	// i59 manquait à la table (2026-08-16, plan PLAN_GRAPPIN_LIGNE) : le commentaire
	// ci-dessus disait « i57/i59 -> 2 » mais seules les clés d'i57 existaient, donc la
	// queue R(3) d'i59 (FUN_140fc147c, param_4>1) n'était JAMAIS lue offline. Mesure :
	// chaque record i59 finissait à 3 bits exactement du record suivant (écarts
	// p10=p50=p90=3, n=988, TestI59AnchorWalkProof) ; avec la clé, l'écart tombe à 0.
	// Les deux étiquettes viennent des constantes de `grapple_state.go` (une seule source par
	// littéral de registre) : le lecteur d'i59 et cette table ne peuvent plus diverger.
	grappleComponentNameAlt: 2,
	grappleComponentName:    2,
}

// paramForComponent rend le param_4 du composant `name`. Défaut 1 : c'est la valeur
// mesurée pour l'écrasante majorité des composants (0 était un choix « conservateur »
// jamais mesuré, et faux).
func paramForComponent(name string) uint32 {
	if v, ok := paramByComponent[name]; ok {
		return v
	}
	if recordStateParamOverride {
		return recordStateParam // un harnais de balayage a forcé la valeur
	}
	return 1
}

// recordStateParamOverride passe à true dès qu'un harnais appelle SetRecordStateParam :
// le balayage manuel garde alors la main sur les composants hors table.
var recordStateParamOverride = false
