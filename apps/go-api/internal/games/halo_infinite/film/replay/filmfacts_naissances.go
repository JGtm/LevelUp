package replay

// filmfacts_naissances.go — CE QUE LE FILM DIT DES ARMES A L INSTANT, DANS LES FAITS PERSISTES
// (lots M3.1 et M3.2 de la campagne « retours rejeu », 2026-09-23, blob v26).
//
// LA SANTE DE LA MARCHE D IMAGE-CLE VOYAGE AVEC LES FAITS pour la meme raison que les stats des
// canaux delta : le document la publie (`coverage.keyframes`), et un rejeu depuis les faits doit
// rendre le MEME document qu un decodage du film (equivalence S8 du lot 4.1). Une couverture
// recalculee a la relecture n existerait pas : elle se mesure pendant la marche.
//
// LES DOTATIONS DE NAISSANCE (lot M3.2) VOYAGENT AVEC LEURS REFUS, pour la meme raison : une
// liste vide sans ses compteurs ne distinguerait pas « aucune naissance » de « aucune ne se
// ferme ». Chaque emplacement porte son RANG — la cle que la premiere emission de la vie partage
// avec lui — et sa famille, sentinelle d emplacement vide comprise.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// encodeMarcheImageCle ecrit la couverture de la marche d image-cle, champ par champ.
func encodeMarcheImageCle(w *gwriter, c grammar.KeyframeWalkCoverage) {
	for _, v := range []int{c.Payloads, c.Records, c.Bipedes, c.Voisins, c.Sauts, c.Recalages,
		c.Elections, c.Refutations, c.Glissements, c.BipedesAbsentsEncadres} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

// decodeMarcheImageCle relit ce que [encodeMarcheImageCle] a ecrit, dans le meme ordre.
func decodeMarcheImageCle(r *greader) grammar.KeyframeWalkCoverage {
	var c grammar.KeyframeWalkCoverage
	for _, p := range []*int{&c.Payloads, &c.Records, &c.Bipedes, &c.Voisins, &c.Sauts, &c.Recalages,
		&c.Elections, &c.Refutations, &c.Glissements, &c.BipedesAbsentsEncadres} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return c
}

// encodeNaissances ecrit les dotations de naissance puis leurs compteurs.
func encodeNaissances(w *gwriter, births []types.BirthLoadout, st types.BirthLoadoutStats) {
	w.u(uint64(len(births)))
	var lastTS uint64
	for _, b := range births {
		w.u(b.TimestampUS - lastTS) // le balayage rend les naissances dans l ordre du film
		lastTS = b.TimestampUS
		w.u(uint64(b.Slot))
		w.u(uint64(b.Generation))
		w.u(uint64(len(b.Weapons)))
		for _, a := range b.Weapons {
			w.u(uint64(a.Emplacement)) //nolint:gosec // rang d emplacement, positif
			w.u(uint64(a.Family))
			w.u(uint64(a.Low))
		}
	}
	for _, v := range []int{st.Creations, st.Read, st.Desync, st.Overflow, st.Unconfirmed,
		st.NoWeaponComponent, st.ClosedByDelta, st.ClosedByBoundNew, st.ClosedByAnticipatedNew} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

// decodeNaissances relit ce que [encodeNaissances] a ecrit, dans le meme ordre.
func decodeNaissances(r *greader) ([]types.BirthLoadout, types.BirthLoadoutStats) {
	n := int(r.u()) //nolint:gosec // compte ecrit positif
	var out []types.BirthLoadout
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		b := types.BirthLoadout{TimestampUS: lastTS}
		b.Slot = uint32(r.u())       //nolint:gosec // ecrit depuis un uint32
		b.Generation = uint32(r.u()) //nolint:gosec // ecrit depuis un uint32
		na := int(r.u())             //nolint:gosec // compte ecrit positif
		for j := 0; j < na && r.err == nil; j++ {
			var a types.BirthWeapon
			a.Emplacement = int(r.u()) //nolint:gosec // rang ecrit positif
			a.Family = uint32(r.u())   //nolint:gosec // ecrit depuis un uint32
			a.Low = uint32(r.u())      //nolint:gosec // ecrit depuis un uint32
			b.Weapons = append(b.Weapons, a)
		}
		out = append(out, b)
	}
	var st types.BirthLoadoutStats
	for _, p := range []*int{&st.Creations, &st.Read, &st.Desync, &st.Overflow, &st.Unconfirmed,
		&st.NoWeaponComponent, &st.ClosedByDelta, &st.ClosedByBoundNew, &st.ClosedByAnticipatedNew} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return out, st
}
