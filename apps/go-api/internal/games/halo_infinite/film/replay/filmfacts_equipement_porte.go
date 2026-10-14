package replay

// filmfacts_equipement_porte.go — L EQUIPEMENT PORTE DANS LE FICHIER DE FAITS (schema des faits 13) :
// les lectures des objets ti=37 crees attaches et leurs denominateurs, au complement de la section 1
// ([encodeComplementDesEntrees]), apres les poussees.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// encodeEquipementPorte ecrit les lectures (triees dans l ordre du flux, d ou le delta d instant) puis
// les denominateurs.
func encodeEquipementPorte(w *gwriter, ls []types.CarriedEquipmentRead, st types.CarriedEquipmentStats) {
	w.u(uint64(len(ls)))
	var last uint64
	for _, l := range ls {
		w.u(l.TimestampUS - last)
		last = l.TimestampUS
		w.u(uint64(l.Object.Slot))
		w.u(uint64(l.Object.Gen))
		w.u(uint64(l.Carrier))
		w.str(string(l.Kind))
		w.bool8(l.HasEnergy)
		w.u(uint64(l.Energy))
		w.bool8(l.HasActivation)
		w.bool8(l.HasSignal)
		w.u(uint64(l.Signal))
		w.bool8(l.HasHandles)
		w.u(uint64(l.Handles))
	}
	for _, v := range comptesDeLEquipementPorte(&st) {
		w.u(uint64(*v))
	}
}

// decodeEquipementPorte relit ce qu [encodeEquipementPorte] ecrit ; une liste vide se relit nil.
func decodeEquipementPorte(r *greader) ([]types.CarriedEquipmentRead, types.CarriedEquipmentStats) {
	n := r.compte(12)
	ls := make([]types.CarriedEquipmentRead, 0, n)
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		l := types.CarriedEquipmentRead{TimestampUS: last}
		l.Object = types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())} //nolint:gosec // ecrits depuis des uint32
		l.Carrier = uint32(r.u())                                         //nolint:gosec // idem
		l.Kind = types.CarriedReadKind(r.str())
		l.HasEnergy = r.bool8()
		l.Energy = uint32(r.u()) //nolint:gosec // R(14)
		l.HasActivation = r.bool8()
		l.HasSignal = r.bool8()
		l.Signal = uint32(r.u()) //nolint:gosec // R(4)
		l.HasHandles = r.bool8()
		l.Handles = uint32(r.u()) //nolint:gosec // R(4)
		ls = append(ls, l)
	}
	if len(ls) == 0 {
		ls = nil
	}
	var st types.CarriedEquipmentStats
	for _, p := range comptesDeLEquipementPorte(&st) {
		*p = int(r.u()) //nolint:gosec // compteurs
	}
	return ls, st
}

// comptesDeLEquipementPorte rend les denominateurs, dans l ordre du fichier.
func comptesDeLEquipementPorte(st *types.CarriedEquipmentStats) []*int {
	return []*int{&st.Created, &st.Attached, &st.Deltas, &st.Deleted, &st.Detached, &st.EnergyReads,
		&st.Activations, &st.Signals, &st.HandleReads}
}
