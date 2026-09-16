package grammar

// harnais_observation_test.go — L OBSERVATEUR DES INSTRUMENTS (lot 2.3).
//
// # CE QUE CE FICHIER REMPLACE
//
// Jusqu au lot 2.3, `filmdec` portait UN observateur de processus (`observateur`) et
// VINGT-HUIT reglages publics (`SetXxxHook`) qui ecrivaient dedans. C etait la DERNIERE
// variable de paquet ecrite du decodeur, et l une des deux raisons pour lesquelles tout
// decodage passait sous un verrou.
//
// LA PRODUCTION N EN A PLUS : chaque balayage construit le SIEN ([NouvelleObservation]) et le
// pose sur ses lecteurs avec son profil ([ContexteDeLecture]). Les INSTRUMENTS, eux, installent
// un crochet puis appellent un deserialiseur de bas niveau sur un lecteur qu ils ont construit :
// ils partagent donc un observateur de HARNAIS, borne a ce fichier `_test.go` — hors du
// perimetre du ratchet, qui ne compte que les fichiers non-test.
var observateur = NouvelleObservation()

// lecteurDInstrument construit un lecteur portant le profil ET l observateur du harnais. C est
// ce que `LecteurSur` faisait implicitement avant le lot 2.3, quand les deux etaient des
// variables de paquet.
func lecteurDInstrument(buf []byte) *Lecteur {
	br := LecteurSur(buf)
	br.PoserContexte(ContexteDeLecture{Profil: profilDInstrument, Obs: observateur})
	return br
}

// poserObservateur installe l observateur du harnais et rend le precedent.
func poserObservateur(o *Observation) *Observation {
	prev := observateur
	observateur = o
	return prev
}

// Les vingt-huit reglages de crochet du harnais. Ils ecrivaient dans la variable de paquet de
// `filmdec` jusqu au lot 2.3 ; ils ecrivent desormais dans l observateur de CE fichier.
func SetAbilityEnergyHook(h func(mask uint32, ch [AbilityEnergyCharges]int)) {
	observateur.AbilityEnergyHook = h
}
func SetCamoStateHook(h func(st CamoState))              { observateur.CamoStateHook = h }
func SetMobilityActionHook(h func(flag1, flag2 bool))    { observateur.MobilityActionHook = h }
func SetGrenadeSetHook(h func(mask uint32, sel int))     { observateur.GrenadeSetHook = h }
func SetEmpTimerHook(h func(quant uint32))               { observateur.EmpTimerHook = h }
func SetUnitRefHook(h func(UnitRefRead))                 { observateur.UnitRefHook = h }
func SetWeaponRoundsHook(h func(rounds uint32))          { observateur.WeaponRoundsHook = h }
func SetDesiredWeaponSetHook(h func(sel uint32))         { observateur.DesiredWeaponSetHook = h }
func SetGroundWeaponAmmoHook(h func(a, b, c uint32))     { observateur.GroundWeaponAmmoHook = h }
func SetHeldWeaponHook(h func(idHigh, idLow uint32))     { observateur.HeldWeaponHook = h }
func SetObjectParentStateHook(h func(ObjectParentState)) { observateur.ObjectParentStateHook = h }
func SetSpartanAbilityHook(h func(tag, sub, ref uint64, hasRef bool)) {
	observateur.SpartanAbilityHook = h
}
func SetAbilityNonPredictedHook(h func(st AbilityNonPredictedState)) {
	observateur.AbilityNonPredictedHook = h
}
func SetAbilitySetHook(h func(counter uint64, rank int, width int)) {
	observateur.AbilitySetHook = h
}
func SetGameEngineHook(h func(f GameEngineField, values []uint64, present bool)) {
	observateur.GameEngineHook = h
}
func SetManagedObjectHook(h func(f ManagedObjectField, values []uint64)) {
	observateur.ManagedObjectHook = h
}
func SetNavpointHook(h func(f NavpointField, values []uint64)) { observateur.NavpointHook = h }
func SetObjectiveHook(h func(f ObjectiveField, values []uint64)) {
	observateur.ObjectiveHook = h
}
func SetManagedPropertyHook(h func(f ManagedPropertyField, values []uint64)) {
	observateur.ManagedPropertyHook = h
}
func SetPlayerStateHook(h func(f PlayerStateField, values []uint64, present bool)) {
	observateur.PlayerStateHook = h
}
func SetProbeHook(h func(ti uint32, comp ProbeComponent, values []uint64)) {
	observateur.ProbeHook = h
}
func SetMultiplayerPropertiesHook(h func(f MPPField, value uint64, present bool)) {
	observateur.MppHook = h
}
func SetEquipmentCreationHook(h func(f EquipmentCreationField, value uint64, present bool)) {
	observateur.EquipmentCreationHook = h
}
func SetEquipmentStateHook(h func(f EquipmentField, value uint64, present bool)) {
	observateur.EquipmentStateHook = h
}
func SetRecordMaskHook(h func(idx []int, payload []byte, afterI0 int)) {
	observateur.RecordMaskHook = h
}
func SetGrenadeCountsHook(h func(count uint64, values []uint64)) {
	observateur.GrenadeCountsHook = h
}
func SetUnitEquipmentHook(h func(UnitEquipmentRead)) { observateur.UnitEquipmentHook = h }
func SetWeaponAmmoHook(h func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32)) {
	observateur.WeaponAmmoHook = h
}

// AbsIndexHistogram rend (et remet a zero) l histogramme des index de plage absolus du harnais.
func AbsIndexHistogram() map[int]int { return observateur.prendreIndexAbsolus() }

// contexteDInstrument rend le profil ET l observateur du harnais, sous la forme que les entrees
// de marche recoivent.
func contexteDInstrument() ContexteDeLecture {
	return ContexteDeLecture{Profil: profilDInstrument, Obs: observateur}
}
