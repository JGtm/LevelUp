package grammar

// component_param4.go — LE param_4 DU MOTEUR : LE FILM L ECRIT, ON LE LIT.
//
// # CE QUE C EST
//
// `param_4` est la propriete externe que le descripteur d un composant rend a `FUN_14076cb60`
// avant que son deserialiseur ne tourne. Huit desers du depot branchent dessus, et la valeur
// decide alors d une LARGEUR :
//
//	i2  object-forward-and-up-dynamic-precision  >= 2 : un bit de porte C de plus, qui bascule
//	                                             la charge utile sur FUN_142e29bac
//	i10 object-parent-state                      < 2 : la lecture libre ; > 2 : la queue R(3)
//	i19 unit-actor-control                       > 1 : l identifiant de second slot
//	i20 unit-actor-state                         la table de largeur (1->8, 2->10, 3->11, >=4->12)
//	i23 unit-malleable-property                  > 2 et > 3 : quelques R(1)
//	i53 biped-malleable-property                 > 1 : un drapeau
//	i62 biped-slide                              >= 1 : un second R(8) dequantifie
//	i59 biped-spartan-ability-non-predicted      > 1 : la queue R(3)
//	ti=12 i2..i6 les cinq filtres du navpoint    decide DEUX largeurs du bloc de filtres
//	ti=21 flock-destination                      > 1 : un R(2)
//
// # SA SOURCE, ETABLIE LE 2026-09-18 (lot 5.1.7) : LA COLONNE `level` DU REGISTRE DU FILM
//
// `param_4` EST le niveau que l entree de registre du composant porte en `entree + 0x100` —
// celui que `FUN_142e2c690` passe au deserialiseur, et que [Archetype.Level] rend deja. Le
// traverseur le descend jusqu ici sous le nom `level` : aucun appelant n a besoin d une table,
// d un balayage ni d un defaut.
//
// TROIS SOURCES INDEPENDANTES LE DISENT, ET ELLES CONCORDENT :
//
//  1. LA CAPTURE LIVE. Les vingt entrees que cette table portait etaient mesurees sur la colonne
//     `param4` de `.ai/V7.5/dumps/ce_capture_delta.csv` (464 010 mesures sur `ti=35`). Elles
//     valent TOUTES le `level` de la meme ligne de `testdata/ecs_table.tsv`.
//  2. L ECRIVAIN. Le lot 5.1.1 a LU la valeur des cinq filtres de `ti=12` au slot `+0x10` du
//     descripteur — `MOV EAX,0x3 ; RET` a `0x14117e0e0` pour `i2`, `MOV EAX,0x2 ; RET` a
//     `0x141179610` pour `i3..i6`. Ce sont exactement leurs `level`.
//  3. LA TABLE ECS. Un nom de composant n a qu UN `level` sur tous les archetypes du registre
//     (mesure du 2026-09-18 sur les 48 lignes concernees), ce qu une propriete de descripteur
//     doit avoir et qu une valeur par archetype n aurait pas.
//
// # CE QUI A DISPARU LE 2026-09-18, ET POURQUOI
//
// `paramForComponent(br, name)` consultait d abord CETTE table, puis — hors table — le `param_4`
// qu un harnais avait FORCE sur le profil du lecteur, sinon 1. Le harnais etait
// `killsource.calibrateRSP` : un balayage de 0 a 5 qui retenait la valeur maximisant la
// CROISSANCE DES SLOTS sur les records de BIPEDE, et que `replaybuild` passait ensuite a la
// cuisson du rejeu. C etait le repli nomme `repli_parametre_etat_record_infere`, dont la cible
// de retrait etait ecrite d avance : « lot qui trouvera la source LUE de `param_4` (registre ECS
// par composant, ou table du build) ». C est ce lot ; la valeur est LUE ; le repli est retire.
//
// TROIS DESERS N AVAIENT AUCUNE ENTREE et prenaient donc la valeur BALAYEE : `i10
// object-parent-state` (vrai `level` 3), `i19 unit-actor-control` (2), `i20 unit-actor-state`
// (4). Sur `4f77afc1` et `a349fea8` le balayage retenait 4, qui se comporte comme 3 / 2 / 4 pour
// les seuls tests que ces desers font (`< 2`, `> 1`, `> 2`, `>= 4`) — mesure du 2026-09-18 :
// zero difference d octet. La faute etait LATENTE, pas active : un film dont le balayage aurait
// retenu 0 ou 1 aurait lu les trois a la mauvaise largeur.

// paramByComponent N EST PLUS UNE SOURCE : c est un RATCHET DE COHERENCE, et le seul appelant
// qui la lit encore est celui qui n a pas d archetype sous la main (cf. [paramMesureDuComposant]).
//
// DEUX garde-rails la tiennent (`param4_par_build_ratchet_test.go`, 2026-09-18) :
// `TestParam4TableEgaleLExecutable` compare chaque entree a la constante que `vtable[0]` du
// descripteur rend dans l executable COURANT, et `TestParam4RegistreParBuild` compare le registre
// de SEPT mini-bobines — une par cle de profil — a cette meme constante, aux ecarts connus et
// dates pres. Une valeur ecrite a la main ne peut donc diverger ni de l executable, ni d un build.
//
// Provenance des vingt valeurs : capture live `param4` sur `ti=35` (464 010 mesures, aucune
// valeur double pour un composant donne) pour les quinze premieres ; slot `+0x10` du descripteur
// (lot 5.1.1) pour les cinq filtres de `ti=12`.
var paramByComponent = map[string]uint32{
	compObjectMaximumVitalities:          3,
	compObjectLowFrequency:               2,
	compObjectFrameConfiguration:         0,
	"unit-control-component":             2,
	"unit-malleable-property-component":  4,
	compObjectParentState:                3,
	"unit-actor-control-component":       2,
	"unit-actor-state-component":         4,
	compWeaponStateTypeInfo:              2,
	"biped-malleable-property":           2,
	"biped-malleable-property-component": 2,
	abilityPredictedNameAlt:              2,
	abilityPredictedName:                 2,
	compForwardUpDynPrec:                 2,
	grappleComponentNameAlt:              2,
	grappleComponentName:                 2,
	compNavpointDistanceFilters:          3,
	compNavpointOffscreenFilters:         2,
	compNavpointOccludedFilters:          2,
	compNavpointVisibilityFilter:         2,
	compNavpointDockingFilter:            2,
}

// paramMesureDuComposant rend le `param_4` d un composant PAR SON NOM.
//
// UN SEUL APPELANT, ET C EST SA RAISON D ETRE : la grammaire d orientation des archetypes
// `ti=38/39/40/43` (`offline_aim.go`) compose sa grammaire AVANT d ouvrir le moindre lecteur et
// sans registre, donc sans `Archetype.Level`. Partout ailleurs la valeur descend du FILM, par le
// parametre `level` du traverseur — c est la seule source.
//
// Le nom qu il passe (`object-forward-and-up-dynamic-precision-component`) a le meme `level` (2)
// sur les quatre archetypes qui le portent ; le ratchet ci-dessus le tient.
//
// DEFAUT 1 pour un nom hors table : c est la valeur du `level` de l ecrasante majorite des
// composants du registre. Il n est atteignable par aucun appelant d aujourd hui.
func paramMesureDuComposant(name string) uint32 {
	if v, ok := paramByComponent[name]; ok {
		return v
	}
	return 1
}
