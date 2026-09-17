package grammar

// observateur.go — L OBSERVATEUR : TOUT CE QUI REGARDE LE DECODAGE SANS LE CHANGER
// (lot 2.2.f du PLAN_DECODEUR_FILM).
//
// # CE QUE CE FICHIER REMPLACE
//
// VINGT-NEUF variables de paquet — un crochet par famille de composants, plus la capture de
// position, le masque de record et la sonde de references d unite — et HUIT compteurs
// d observation de l inference de chaine. Chacune vivait a cote du deserialiseur qu elle
// observait, chacune avait son reglage, et chacune avait sa propre duree de vie a gerer a la
// main : trente-sept etats de processus, tous nuls en production, qu un verrou de paquet
// devait serialiser parce qu on ne pouvait rien dire de leur ensemble.
//
// # CE QUE L OBSERVATEUR EST, ET CE QU IL N EST PAS
//
// UN OBSERVATEUR NE CHANGE AUCUNE CONSOMMATION DE BITS. C est la propriete qui le distingue du
// PROFIL : le profil DECIDE des largeurs, l observateur ne fait que recevoir ce que le
// deserialiseur a deja lu. Un champ de cette structure qui changerait un compte de bits serait
// une valeur de profil mal rangee, pas une sonde.
//
// TOUS SES CHAMPS SONT NULS EN PRODUCTION. Les balayages de `filmdec` en posent un le temps
// d une marche et le retirent au retour ; rien hors du paquet n en installe.
//
// # IL N EST PLUS UN ETAT DE PROCESSUS (lot 2.3)
//
// Le lecteur de bits le PORTE ([Lecteur.obs], `nil` en production) et les portes a
// [FrameConfig] le posent ([FrameConfig.Obs], par [Lecteur.poserCadre]). Les VINGT-NEUF
// crochets de deserialiseur ne partagent plus un observateur de processus : chaque balayage
// construit le SIEN ([NouvelleObservation]), l installe sur le cadre ou sur la grammaire de sa
// marche, et le lecteur le recoit d un seul geste avec le profil.
//
// LA VARIABLE `observateur` ET SON RESTAURATEUR `poserObservateur` ONT DISPARU. C etait la
// DERNIERE variable de paquet ECRITE de `filmdec` — et l une des deux raisons pour lesquelles
// tout decodage passait.

// Observation porte tout ce qui regarde un decodage sans le changer. Ses champs de fonction
// sont NULS en production.
type Observation struct {
	// (depuis `ability_energy.go`)
	// AbilityEnergyHook, si non nil, reçoit d'i56 le masque R(3) et les trois valeurs 7 bits
	// (AbilityEnergyUnarmed pour un emplacement non armé). Le déser reste inchangé bit pour bit.
	AbilityEnergyHook func(mask uint32, ch [AbilityEnergyCharges]int)

	// (depuis `ability_state_hooks.go`)
	// CamoStateHook, si non nil, reçoit CHAQUE lecture d'i28 par le déser de production.
	CamoStateHook func(st CamoState)

	// (depuis `ability_state_hooks.go`)
	// MobilityActionHook, si non nil, reçoit les deux drapeaux de tête de CHAQUE lecture d'i54
	// (flag1 est le gate du corps — cf. consumeBipedMobilityAction, « une action est transmise
	// à cet instant » quand il vaut 1).
	MobilityActionHook func(flag1, flag2 bool)

	// (depuis `ability_state_hooks.go`)
	// SpartanAbilityHook, si non nil, reçoit CHAQUE lecture d'i57 : le tag R(2), et — sur la
	// SEULE branche tag==1, la seule qui paie une charge utile — le R(2) interne
	// (FUN_142f25d78) et le R(24) (FUN_14076dc04, 0x18). hasRef est faux sur les autres
	// branches : Sub et Ref n'y existent pas dans le flux.
	SpartanAbilityHook func(tag, sub, ref uint64, hasRef bool)

	// (depuis `ability_state_hooks.go`)
	// AbilityNonPredictedHook, si non nil, reçoit CHAQUE lecture d'i59 : le tag externe R(2)
	// et — depuis le port du corps tag==3 (2026-08-16, plan PLAN_GRAPPIN_LIGNE) — la lecture
	// complète du bloc FUN_142f25e90 (tag interne, ids, les trois vecteurs quantifiés, queue).
	// Cf. AbilityNonPredictedState (components_biped_anchor.go) : BodyWalked/BodyOK disent si
	// le corps a été parcouru et s'il est allé au bout.
	AbilityNonPredictedHook func(st AbilityNonPredictedState)

	// (depuis `components_biped_ability.go`)
	// GrenadeSetHook, si non nil, reçoit d'i47 : le masque R(6) des types portés et la sélection
	// R(3) — GrenadeSetNoSelection quand aucun type n'est désigné. Le déser reste inchangé bit
	// pour bit.
	GrenadeSetHook func(mask uint32, sel int)

	// (depuis `components_biped_ability.go`)
	// AbilitySetHook, si non nil, reçoit d'i48 : la valeur R(3) (compteur de rotation), le RANG
	// de palette R(6) — ou AbilitySetNoRank quand la porte est fermée — et la largeur totale
	// consommée. Le déser reste inchangé bit pour bit : le hook ne fait que publier.
	AbilitySetHook func(counter uint64, rank int, width int)

	// (depuis `components_game_engine.go`)
	// GameEngineHook, si non nil, recoit CHAQUE lecture d'un des cinq composants.
	//
	// CONTRAT DE `values` ET DE `present`, commun aux quatre hooks de ce lot :
	//
	//	values   les champs lus par le deser, DANS L'ORDRE DU FLUX, bits de porte compris.
	//	         Jamais de dequantification, jamais de mise a l'echelle : c'est le lot qui
	//	         mesure qui decide du sens, pas la plomberie.
	//	present  faux quand la porte de TETE du composant s'est fermee et qu'aucun champ n'a
	//	         suivi. Une porte fermee n'est PAS une valeur nulle, et les confondre
	//	         fabriquerait des transitions qui n'existent pas.
	//
	GameEngineHook func(f GameEngineField, values []uint64, present bool)

	// (depuis `components_managed_object.go`)
	// ManagedObjectHook, si non nil, recoit chaque lecture d'un champ de ti=10.
	//
	// PAS DE `present` ICI : aucun de ces composants n'a de porte de tete.
	ManagedObjectHook func(f ManagedObjectField, values []uint64)

	// (depuis `components_managed_object.go`)
	// NavpointHook, si non nil, recoit chaque lecture d'un champ de ti=12. Pas de `present` : le
	NavpointHook func(f NavpointField, values []uint64)

	// (depuis `components_managed_objective.go`)
	// ObjectiveHook, si non nil, recoit chaque lecture d'un champ publie de ti=11.
	//
	// PAS DE `present` ICI : aucun de ces composants n'a de porte de tete — leur presence est le bit
	// de MASQUE, que l'appelant connait deja.
	ObjectiveHook func(f ObjectiveField, values []uint64)

	// (depuis `components_managed_property.go`)
	// ManagedPropertyHook, si non nil, recoit chaque lecture d'un champ de ti=13.
	//
	// PAS DE `present` ICI : aucun des deux composants n'a de porte de tete — le tag EST la valeur de
	//
	// FORME DES VALEURS : `values[0]` est toujours le TAG ; `values[1]`, present seulement quand la
	// branche lit, est le quantum BRUT. Une branche muette publie donc un seul element — et c'est une
	// information, pas un manque : elle dit que la propriete existe et que ce record n'en porte pas
	// la valeur.
	ManagedPropertyHook func(f ManagedPropertyField, values []uint64)

	// (depuis `components_object.go`)
	// consumeWeaponStateTypeInfoVariant (i43..46 = HELD WEAPON) mirrors FUN_1407f06bc.
	// Returns the variant-name string-id (the WEAPON) and whether the slot is present.
	//
	// Gate (FUN_14080d69c): R(1). If 0 -> slot absent (field = 0xFFFFFFFF), no more
	// reads. If 1 -> FUN_14080d6f0 reads R(32) (the optional local-handle id) THEN:
	//
	//	variant   = R(32)   FUN_14080dec4 "variant-name"   <-- THE WEAPON
	//	R(12)               inline (comp+0x7c)
	//	FUN_140e9fadc       R(7)
	//	gate2 = R(1); if 1 -> FUN_141001f50 (R(?) shot-id; ~2 bytes XOR-decoded)
	//	R(1)                (comp+0x82 low bit)
	//	FUN_1407f2494       R(1); if 1 -> R(4) count + count*(R(1)+[R(32)])
	//	FUN_1407f0550       FUN_1404d343c + R(1)+[R(32)] + 3*(R(8)+R(1)+[dequant]+R(1)+[dequant])
	//	FUN_1407f2058       R(1); if 0 -> R(5)
	//	FUN_140e958c4       (handle lookup, 0 bits)
	//
	// Then unconditionally (both gate branches):
	//
	//	FUN_1407f08bc       R(1); if 1 -> R(8)
	//	FUN_1406d01fc       R(3) + 2*(R(1);if 0 -> R(2))
	//
	// The variant string-id is read at a FIXED position: gateR1 + R(32 handle) +
	// R(32 variant). Downstream sub-reads are data-dependent (loops); only the
	// variant itself is load-bearing for weapon attribution.
	// Le drapeau « présent » n'est pas rendu : noVariant (0xFFFFFFFF) EST le témoin
	// d'absence, et c'est déjà celui que le reste du paquet teste.
	// HeldWeaponHook, si non nil, reçoit CHAQUE lecture d'i43..i46 (l'arme portée), y compris
	// les lectures d'emplacement ABSENT (variant == noVariant) : c'est la transition
	// présent/absent qui porte le lâcher, la retirer rendrait le signal borgne. Même contrat que
	// les autres sondes (SetAbilitySetHook, SetObjectParentStateHook, SetGrenadeCountsHook).
	HeldWeaponHook func(idHigh, idLow uint32)

	// (depuis `components_object_state.go`)
	// ObjectParentStateHook, si non nil, reçoit CHAQUE lecture d'i10.
	ObjectParentStateHook func(ObjectParentState)

	// (depuis `components_player.go`)
	// PlayerStateHook, si non nil, recoit CHAQUE lecture d'un des onze composants. Meme contrat de
	// `values` / `present` que `GameEngineHook` (cf. son commentaire).
	PlayerStateHook func(f PlayerStateField, values []uint64, present bool)

	// (depuis `components_probe.go`)
	// ProbeHook, si non nil, recoit les valeurs des composants sondes.
	//
	// PAS DE `present` ICI, a la difference des trois autres hooks : aucun des quatre composants
	// n'a de porte de tete. Ajouter un booleen toujours vrai serait un champ qui mentirait le jour
	// ou l'un d'eux en gagnerait une.
	ProbeHook func(ti uint32, comp ProbeComponent, values []uint64)

	// (depuis `default_state.go`)
	// MppHook, si non nil, reçoit chaque lecture d'un champ du bloc. `present` est faux quand la
	// porte s'est fermée sans transmettre de valeur — une porte fermée n'est pas une valeur nulle.
	MppHook func(f MPPField, value uint64, present bool)

	// (depuis `emp_timer.go`)
	// EmpTimerHook, si non nil, reçoit le quantum R(8) de CHAQUE lecture d'i51 par le déser de
	// production. Champ de l'observation d'UN balayage, jamais du processus (lot 2.3).
	EmpTimerHook func(quant uint32)

	// (depuis `equipment_creation.go`)
	// EquipmentCreationHook, si non nil, reçoit CHAQUE lecture des deux champs du default-state de
	// ti=37 : le champ, la valeur, et `present` — faux quand la porte s'est fermée sans transmettre
	// de valeur. Une porte fermée n'est PAS une valeur nulle.
	EquipmentCreationHook func(f EquipmentCreationField, value uint64, present bool)

	// (depuis `equipment_state.go`)
	// EquipmentStateHook, si non nil, reçoit CHAQUE lecture d'un des quatre composants : le
	// champ, la valeur, et `present` — faux quand la porte du composant s'est fermée sans
	// transmettre de valeur. Une porte fermée n'est PAS une valeur nulle, et les confondre
	// fabriquerait des transitions qui n'existent pas.
	EquipmentStateHook func(f EquipmentField, value uint64, present bool)

	// (depuis `offline_aim.go`)
	// RecordMaskHook (DEBUG) reçoit, pour chaque record biped ÉMIS, la liste des index de
	// composants de son masque, le payload du paquet et le bit qui suit i0. Sert au
	// diagnostic « quel champ, à quel offset, porte la direction » ; nil en production (même
	// convention que `UnitRefHook`). Les appels sont dans le MÊME ordre que les
	// positions renvoyées (filtres de post-traitement mis à part).
	RecordMaskHook func(idx []int, payload []byte, afterI0 int)

	// (depuis `position_capture.go`)
	// PosCaptureHook, when non-nil, receives every i0 position payload the deser decodes.
	// Nil by default (no-op, no behaviour change). Not safe for concurrent use across
	// goroutines (single-frame decode is sequential).
	//
	// INSTALLÉ DEPUIS LE PAQUET, PLUS DE L'EXTÉRIEUR : le seul installateur est
	// `scanForTargetDelta` (frame_records.go), qui capture la position i0 de chaque record
	// d'essai. Le réglage public `SetPositionCaptureHook` a été supprimé le 2026-09-05
	// (lot E, item E.2) : il n'avait aucun appelant.
	PosCaptureHook func(PositionSample)

	// (depuis `unit_ref_probe.go`)
	// UnitRefHook, si non nil, reçoit CHAQUE lecture de champ de référence.
	UnitRefHook func(UnitRefRead)

	// (depuis `unit_weaponstate.go`)
	// GrenadeCountsHook, si non nil, reçoit le compteur et les valeurs lues par i22.
	// Sonde de mesure uniquement (cmd/tmp_i22check) : le déser est inchangé.
	GrenadeCountsHook func(count uint64, values []uint64)

	// (depuis `unit_weaponstate.go`)
	// UnitEquipmentHook, si non nil, reçoit CHAQUE lecture d'i26. Même contrat
	// que les autres sondes (SetAbilitySetHook, SetObjectParentStateHook).
	UnitEquipmentHook func(UnitEquipmentRead)

	// (depuis `unit_weaponstate.go`)
	// WeaponAmmoHook, si non nil, reçoit CHAQUE lecture d'un `weapon-state-ammo` : le chargeur
	// R(8) et sa présence (porte ACTIVE-BAS), puis le quantum R(12) de fraction et sa présence.
	//
	// LE HOOK NE SAIT PAS DE QUEL EMPLACEMENT IL PARLE : le déser est le même pour les quatre
	// occurrences (i30/i33/i36/i39). C'est l'APPELANT qui associe la publication à l'emplacement,
	// depuis l'index de composant que la marche vient de consommer (cf. inventory_delta.go).
	WeaponAmmoHook func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32)

	// (depuis `unit_weaponstate.go`)
	// WeaponRoundsHook, si non nil, reçoit CHAQUE lecture d'un `weapon-state-rounds-inventory`.
	// Même remarque que WeaponAmmoHook : l'emplacement vient de l'appelant, pas du déser.
	WeaponRoundsHook func(rounds uint32)

	// (depuis `unit_weaponstate.go`)
	// consumeBipedDesiredWeaponSet mirrors FUN_1406d01fc (the thunk at 0x14109d298
	// adjusts the pointer to recordState[0x10]+0x13d8 then tail-jumps here):
	//
	//	FUN_1406d0f20 = R(3).
	//	FUN_1406d00ec = R(1)+optR(2).
	//	FUN_1406d00ec = R(1)+optR(2).
	//
	// DesiredWeaponSetHook, si non nil, reçoit CHAQUE lecture d'i42 avec la valeur du R(3) de
	// tête (l'emplacement d'arme désiré). Même règle que les autres sondes : l'observateur est
	// PORTÉ par le lecteur, et deux films se décodent donc en parallèle
	// (`TestDeuxFilmsEnParallele`).
	DesiredWeaponSetHook func(sel uint32)

	// (depuis `unit_weaponstate.go`)
	// GroundWeaponAmmoHook, si non nil, reçoit chaque lecture d'i20 sur l'archétype ARME AU SOL.
	// autres sondes.
	GroundWeaponAmmoHook func(a, b, c uint32)
	// CompWidths enregistre, par nom de composant, les largeurs de bouchon qui ont produit la
	// reconstruction gagnante — l histogramme dont un portage lit la largeur a porter.
	//
	// C EST « LA TABLE SANS VERROU » QUE L EN-TETE DU VERROU DE DECODAGE NOMMAIT : elle etait
	// ecrite pendant un balayage sans qu aucun verrou ne la protege, et c etait l une des deux
	// raisons pour lesquelles tout le decodage passait par un verrou de processus. Elle est
	// desormais un CHAMP de l observateur, donc une chose qu un appelant possede (item 2.2.f) —
	// et le verrou, prive de ses deux raisons, est parti au lot 2.3.
	CompWidths map[string]map[int]int
	// ChaineReparees compte les records sauves par l inference de largeur de composant.
	ChaineReparees int
	// Les cinq issues de l inference de chaine : immediate, profonde, ambigue, aucune, budget.
	ChaineImmediat, ChaineProfond, ChaineAmbigu, ChaineAucun, ChaineBudget int
	// ResyncValides compte les reprises par resynchronisation validee (diagnostic).
	ResyncValides int
	// IndexAbsolus : histogramme des index de plage rencontres sur les chemins ABSOLUS de i0
	// (7ter.54 axe 3). Purement observationnel — incremente sur l axe 0 de chaque lecture, ne
	// change AUCUNE consommation de bits. C est la mesure qui dit si l index dominant est 0
	// (bornes de la carte) ou pas, donc quelle ligne de la table de largeurs pese reellement.
	// C etait la variable de paquet `absIdxHist` jusqu au lot 2.3.
	IndexAbsolus map[int]int
}

// compterIndexAbsolu incremente l histogramme des index de plage absolus.
func (o *Observation) compterIndexAbsolu(idx int) {
	if o == nil {
		return
	}
	if o.IndexAbsolus == nil {
		o.IndexAbsolus = map[int]int{}
	}
	o.IndexAbsolus[idx]++
}

// prendreIndexAbsolus rend l histogramme et le remet a zero.
func (o *Observation) prendreIndexAbsolus() map[int]int {
	if o == nil {
		return map[int]int{}
	}
	out := make(map[int]int, len(o.IndexAbsolus))
	for k, v := range o.IndexAbsolus {
		out[k] = v
	}
	o.IndexAbsolus = map[int]int{}
	return out
}

// NouvelleObservation rend un observateur vide, pret a recevoir des crochets. C est la forme
// qu un instrument passe par [FrameConfig.Obs] au lieu d ecrire dans le processus.
func NouvelleObservation() *Observation {
	return &Observation{CompWidths: map[string]map[int]int{}}
}

// neutraliserCaptures met a nil les DEUX crochets de capture (position, reference d unite) et
// rend leur restauration.
//
// POURQUOI CES DEUX-LA, ET POURQUOI TEMPORAIREMENT. Les chemins d INFERENCE essaient une lecture
// sur des bits qu ils abandonneront peut-etre ; une lecture speculative n est pas une lecture, et
// sans cette neutralisation les tentatives abandonnees deposeraient des echantillons a des
// positions que la traversee retenue ne lit jamais. Les COMPTEURS, eux, restent partages : c est
// le meme observateur, seuls deux champs sont eteints.
func (o *Observation) neutraliserCaptures() func() {
	if o == nil {
		return func() {}
	}
	pos, ref := o.PosCaptureHook, o.UnitRefHook
	o.PosCaptureHook, o.UnitRefHook = nil, nil
	return func() { o.PosCaptureHook, o.UnitRefHook = pos, ref }
}

// neutraliserCapturePosition met a nil le seul crochet de position et rend sa restauration.
func (o *Observation) neutraliserCapturePosition() func() {
	if o == nil {
		return func() {}
	}
	pos := o.PosCaptureHook
	o.PosCaptureHook = nil
	return func() { o.PosCaptureHook = pos }
}

// compterResyncValide compte une reprise par resynchronisation validee (diagnostic).
func (o *Observation) compterResyncValide() {
	if o != nil {
		o.ResyncValides++
	}
}

// compterReparation compte un record sauve par l inference de largeur de composant, et range les
// largeurs de bouchon gagnantes dans l histogramme du composant.
func (o *Observation) compterReparation(nom string, largeurs []int) {
	if o == nil {
		return
	}
	o.ChaineReparees++
	if o.CompWidths == nil {
		o.CompWidths = map[string]map[int]int{}
	}
	if o.CompWidths[nom] == nil {
		o.CompWidths[nom] = map[int]int{}
	}
	for _, w := range largeurs {
		o.CompWidths[nom][w]++
	}
}

// compterIssueDeChaine compte une resolution : immediate (le record suivant confirme) ou
// PROFONDE (la marche recursive a traverse une suite de transitoires).
func (o *Observation) compterIssueDeChaine(immediate bool) {
	switch {
	case o == nil:
	case immediate:
		o.ChaineImmediat++
	default:
		o.ChaineProfond++
	}
}

// compterEchecDeChaine compte un echec : budget epuise, ou aucun alignement confirme.
func (o *Observation) compterEchecDeChaine(budgetEpuise bool) {
	switch {
	case o == nil:
	case budgetEpuise:
		o.ChaineBudget++
	default:
		o.ChaineAucun++
	}
}

// compterAmbiguiteDeChaine compte un alignement AMBIGU — plusieurs candidats survivent, et on ne
// choisit pas.
func (o *Observation) compterAmbiguiteDeChaine() {
	if o != nil {
		o.ChaineAmbigu++
	}
}
