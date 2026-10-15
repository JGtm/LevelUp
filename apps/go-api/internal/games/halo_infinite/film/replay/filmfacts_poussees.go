package replay

// filmfacts_poussees.go — LES POUSSEES D EQUIPEMENT DANS LE FICHIER DE FAITS (schema des faits 12) :
// la liste des paires 105 -> 104 et les denominateurs de leur lecture, au complement de la section 1
// ([encodeComplementDesEntrees]), avant le verdict du fil des morts.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// encodePoussees ecrit les poussees (triees par instant, d ou le delta) puis leurs denominateurs.
func encodePoussees(w *gwriter, ks []types.Knockback, st types.KnockbackStats) {
	w.u(uint64(len(ks)))
	var last uint64
	for _, k := range ks {
		w.u(k.TimestampUS - last)
		last = k.TimestampUS
		w.u(uint64(k.Pushed.Slot))
		w.u(uint64(k.Pushed.Gen))
		w.bool8(k.HasAuthor)
		w.u(uint64(k.Author.Slot))
		w.u(uint64(k.Author.Gen))
		w.bool8(k.HasImpulse)
		for a := range 3 {
			w.f32(k.Impulse[a])
		}
	}
	for _, v := range [...]int{st.Read104, st.Read105, st.Paired, st.Unpaired, st.WithoutAuthor, st.WithoutImpulse} {
		w.u(uint64(v))
	}
}

// decodePoussees relit ce qu [encodePoussees] ecrit ; une liste vide se relit nil.
func decodePoussees(r *greader) ([]types.Knockback, types.KnockbackStats) {
	n := r.compte(10)
	ks := make([]types.Knockback, 0, n)
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		kb := types.Knockback{TimestampUS: last}
		kb.Pushed = types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		kb.HasAuthor = r.bool8()
		kb.Author = types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		kb.HasImpulse = r.bool8()
		for a := range 3 {
			kb.Impulse[a] = r.f32()
		}
		ks = append(ks, kb)
	}
	if len(ks) == 0 {
		ks = nil
	}
	var st types.KnockbackStats
	for _, p := range [...]*int{&st.Read104, &st.Read105, &st.Paired, &st.Unpaired, &st.WithoutAuthor, &st.WithoutImpulse} {
		*p = int(r.u())
	}
	return ks, st
}
