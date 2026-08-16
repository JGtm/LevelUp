package filmdec

// ability_state_hooks.go — SONDES des composants d'ÉTAT DE CAPACITÉ/ÉQUIPEMENT du bipède.
//
// QUATRE désérialiseurs consommaient leurs bits pour rester alignés et JETAIENT les
// valeurs — exactement le défaut d'i48 (corrigé le 2026-08-14, ability_rank.go) et des
// quatre champs de ti=37 (corrigé le 2026-08-15, equipment_state.go) :
//
//	i28 unit-active-camo-state            (consumeUnitActiveCamoState, unit_weaponstate.go)
//	i54 biped-mobility-action             (consumeBipedMobilityAction, components_biped_ability.go)
//	i57 biped-spartan-ability             (consumeBipedSpartanAbility, idem)
//	i59 biped-spartan-ability-non-predicted-state (consumeBipedSpartanAbilityNonPredictedState, idem)
//
// LA RÈGLE QUI GOUVERNE (décision n°4 du plan PLAN_ETAT_ACTIF_EQUIPEMENT) : c'est le
// DÉSERIALISEUR qui publie, jamais un second lecteur posé à côté de lui. Aucune largeur ne
// change : les hooks ne font que publier ce que les désers lisaient déjà. Les hooks sont des
// globaux de paquet : un seul décodage filmdec par process (cf. decode_gate.go).

// CamoState est UNE lecture du composant i28 `unit-active-camo-state` (FUN_142ed3ae0).
// La grammaire, dans l'ordre du flux : R(3) ; R(1) flag0 ; si flag0==0 : R(1) flag1 ; si
// flag1==0 : R(12) déquantifié (FUN_1406d84b4, W=12) ; puis 6 x (R(1) porte ; si 1 : R(12)).
type CamoState struct {
	// C3 est le champ R(3) de tête (comp+0x7d7). Sémantique non établie.
	C3 uint8
	// Flag0 est la première porte. Flag1 n'est LU que si Flag0 vaut 0 (Flag1Read le dit) —
	// confondre « non lu » et « faux » fabriquerait des transitions qui n'existent pas.
	Flag0, Flag1, Flag1Read bool
	// HasFrac / FracQ : le quantum R(12) principal, présent seulement si flag0==0 et
	// flag1==0. Les bornes de déquantification de FUN_1406d84b4 ne sont PAS établies pour ce
	// composant : le quantum BRUT est publié, à déquantifier explicitement côté instrument.
	HasFrac bool
	FracQ   uint16
	// SubPresent / SubQ : les six champs optionnels de queue (FUN_1431fc0cc = 6 x
	// FUN_1411b1ac0, porte à 1 = valeur présente).
	SubPresent [6]bool
	SubQ       [6]uint16
}

// camoStateHook, si non nil, reçoit CHAQUE lecture d'i28 par le déser de production.
var camoStateHook func(st CamoState)

// SetCamoStateHook installe (ou retire, avec nil) la sonde d'i28.
func SetCamoStateHook(h func(st CamoState)) { camoStateHook = h }

// mobilityActionHook, si non nil, reçoit les deux drapeaux de tête de CHAQUE lecture d'i54
// (flag1 est le gate du corps — cf. consumeBipedMobilityAction, « une action est transmise
// à cet instant » quand il vaut 1).
var mobilityActionHook func(flag1, flag2 bool)

// SetMobilityActionHook installe (ou retire, avec nil) la sonde d'i54.
func SetMobilityActionHook(h func(flag1, flag2 bool)) { mobilityActionHook = h }

// spartanAbilityHook, si non nil, reçoit CHAQUE lecture d'i57 : le tag R(2), et — sur la
// SEULE branche tag==1, la seule qui paie une charge utile — le R(2) interne
// (FUN_142f25d78) et le R(24) (FUN_14076dc04, 0x18). hasRef est faux sur les autres
// branches : Sub et Ref n'y existent pas dans le flux.
var spartanAbilityHook func(tag, sub, ref uint64, hasRef bool)

// SetSpartanAbilityHook installe (ou retire, avec nil) la sonde d'i57.
func SetSpartanAbilityHook(h func(tag, sub, ref uint64, hasRef bool)) { spartanAbilityHook = h }

// abilityNonPredictedHook, si non nil, reçoit le tag R(2) de CHAQUE lecture d'i59. Le corps
// de la branche tag==3 (FUN_142f25e90) n'est PAS porté : le tag est publié, la suite du
// record reste ce qu'elle était (cf. le CAVEAT du déser).
var abilityNonPredictedHook func(tag uint64)

// SetAbilityNonPredictedHook installe (ou retire, avec nil) la sonde d'i59.
func SetAbilityNonPredictedHook(h func(tag uint64)) { abilityNonPredictedHook = h }
