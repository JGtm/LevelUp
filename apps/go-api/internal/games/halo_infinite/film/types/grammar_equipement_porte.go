package types

// grammar_equipement_porte.go — LES TYPES DE CONTRAT DE L EQUIPEMENT PORTE (couche `grammar`,
// lot « equipement porte » du plan d ajustements du 2026-10-14).
//
// Un objet d equipement PORTE est une entite ti=37 que le film cree (record NEW) attachee a un
// bipede : son `object-parent-state` (i10) ouvre la porte « attache » et nomme le parent. Tant
// qu il est porte, ses records DELTA disent son etat : l energie (i24), l activation (i21), le
// signal de controle (i22), la pile des poignees suivies (i28). La grammaire les rend BRUTS ; le
// sens (pose de faille, consommation) se lit au rejeu.

// CarriedEquipmentRead est UNE lecture d un objet d equipement porte : sa creation, un delta ou sa
// suppression. Un champ n a de sens que si son drapeau `Has*` est vrai.
type CarriedEquipmentRead struct {
	// TimestampUS est l horodatage du paquet — MEME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Object est la vie de l objet (slot, generation).
	Object LifeKey
	// Carrier est le slot du bipede porteur, lu a la creation (i10).
	Carrier uint32
	// Kind : "new", "delta" ou "del".
	Kind CarriedReadKind
	// Energy est la valeur brute de l energie (i24, R(14) sur [0, 1]).
	HasEnergy bool
	Energy    uint32
	// Activation : la porte d i21 s est ouverte sur une poignee (`FUN_1408f0ac4`, porte a 1) ; une
	// lecture a porte fermee (R(3)) n est pas une activation.
	HasActivation bool
	// Signal est le compteur R(4) du signal de controle (i22).
	HasSignal bool
	Signal    uint32
	// Handles est le nombre d entrees de la pile des poignees suivies (i28).
	HasHandles bool
	Handles    uint32
}

// CarriedReadKind qualifie une lecture d objet porte.
type CarriedReadKind string

const (
	// CarriedNew : le record de creation, attache a un bipede.
	CarriedNew CarriedReadKind = "new"
	// CarriedDelta : un record delta pendant le port.
	CarriedDelta CarriedReadKind = "delta"
	// CarriedDel : la suppression de l objet.
	CarriedDel CarriedReadKind = "del"
)

// CarriedEquipmentStats compte ce que la lecture a rencontre.
type CarriedEquipmentStats struct {
	// Created : records NEW ti=37 lus ; Attached : ceux dont i10 ouvre la porte « attache » sur un
	// bipede (les objets portes suivis).
	Created, Attached int
	// Deltas : records delta des objets suivis ; Deleted : suppressions lues.
	Deltas, Deleted int
	// Detached : objets suivis dont un delta i10 ferme la porte « attache » — suivi arrete.
	Detached int
	// EnergyReads, Activations, Signals, HandleReads : lectures des quatre composants.
	EnergyReads, Activations, Signals, HandleReads int
}
