package replay

// filmfacts_equipement_porte.go — L EQUIPEMENT PORTE DANS LE FICHIER DE FAITS (schema des faits 13, vies suivies au 14, leurs creations au 15) :
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
		w.u(uint64(len(l.Tracked)))
		for _, s := range l.Tracked {
			w.u(uint64(s.Slot))
			w.u(uint64(s.Gen))
		}
		encodeObjetsEngendres(w, l.Spawned)
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
		l.Tracked = decodeVies(r)
		l.Spawned = decodeObjetsEngendres(r)
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
		&st.Activations, &st.Signals, &st.HandleReads, &st.TrackedEntries, &st.TrackedEnds, &st.SpawnedBirths, &st.SpawnedPositioned}
}

// decodeVies relit une liste de vies (slot, generation) ; une liste vide se relit nil.
func decodeVies(r *greader) []types.LifeKey {
	n := r.compte(2)
	if n == 0 {
		return nil
	}
	out := make([]types.LifeKey, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		out = append(out, types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}) //nolint:gosec // ecrits depuis des uint32
	}
	return out
}

// encodeObjetsEngendres / decodeObjetsEngendres : les creations lues des objets engendres (schema des faits 15).
func encodeObjetsEngendres(w *gwriter, ns []types.SpawnedObject) {
	w.u(uint64(len(ns)))
	for _, n := range ns {
		w.u(uint64(n.Life.Slot))
		w.u(uint64(n.Life.Gen))
		w.u(n.TimestampUS)
		w.u(uint64(n.TypeIndex))
		w.bool8(n.HasPosition)
		w.f32(n.X)
		w.f32(n.Y)
		w.f32(n.Z)
	}
}

func decodeObjetsEngendres(r *greader) []types.SpawnedObject {
	n := r.compte(16)
	if n == 0 {
		return nil
	}
	out := make([]types.SpawnedObject, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		s := types.SpawnedObject{Life: types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}} //nolint:gosec // ecrits depuis des uint32
		s.TimestampUS = r.u()
		s.TypeIndex = uint32(r.u()) //nolint:gosec // idem
		s.HasPosition = r.bool8()
		s.X, s.Y, s.Z = r.f32(), r.f32(), r.f32()
		out = append(out, s)
	}
	return out
}
