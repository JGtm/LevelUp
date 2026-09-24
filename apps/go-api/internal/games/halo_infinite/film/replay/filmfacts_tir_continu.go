package replay

// filmfacts_tir_continu.go — LE TIR CONTINU DANS LES FAITS PERSISTES (lot M4b de la campagne
// « retours rejeu », 2026-09-24, blob v27).
//
// LES RAFALES VOYAGENT AVEC LEURS TROUS ET LEURS COMPTEURS, pour la raison des autres canaux : le
// document publie leur couverture (`coverage.continuousFire`), et un rejeu depuis les faits doit
// rendre le MEME document qu un decodage du film (equivalence S8). Une rafale sans ses trous
// rendrait sonore un passage que la lecture n a pas atteint ; des rafales sans compteurs ne
// diraient pas si « aucune rafale » est un film sans tir continu ou un film dont la vue de
// controle n est jamais lue.
//
// LES TIRS PORTENT LEUR UNITE ET LEUR NUMERO (lot M4b.3) : l unite tireuse (reference 0) pose un
// tir de vehicule sur SON vehicule et rattache un tir sans indice de tireur a son corps ; le numero
// de tir est le controle du tir continu (coups simules contre sauts du compteur).

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// encodeTirContinu ecrit les rafales (TRIEES par debut : delta d horodatage) et les compteurs.
func encodeTirContinu(w *gwriter, rs []types.ContinuousFireBurst, st types.ContinuousFireStats) {
	w.u(uint64(len(rs)))
	var lastTS uint64
	for _, r := range rs {
		w.u(r.StartUS - lastTS)
		lastTS = r.StartUS
		w.u(r.EndUS - r.StartUS)
		w.u(uint64(r.FilmIndex)) //nolint:gosec // index de controle, 0..31
		w.u(uint64(r.Hand))      //nolint:gosec // main 0 ou 1
		w.bool8(r.Barrel)
		w.u(uint64(r.Input)) //nolint:gosec // rang 0..2
		w.i(int64(r.Weapon))
		w.str(r.StartBound)
		w.str(r.EndBound)
		w.u(uint64(r.Entries)) //nolint:gosec // compte positif
		w.u(uint64(len(r.Holes)))
		for _, h := range r.Holes {
			w.u(h.StartUS - r.StartUS)
			w.u(h.EndUS - h.StartUS)
		}
	}
	w.bool8(st.Scanned)
	for _, v := range compteursTirContinu(&st) {
		w.u(uint64(*v)) //nolint:gosec // compteurs positifs
	}
	w.i(st.HeldHoleMS)
}

// decodeTirContinu relit ce que [encodeTirContinu] a ecrit, dans le meme ordre.
func decodeTirContinu(r *greader) ([]types.ContinuousFireBurst, types.ContinuousFireStats) {
	n := int(r.u()) //nolint:gosec // compte ecrit positif
	out := make([]types.ContinuousFireBurst, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		b := types.ContinuousFireBurst{StartUS: lastTS}
		b.EndUS = b.StartUS + r.u()
		b.FilmIndex = int(r.u()) //nolint:gosec // ecrit depuis un index 0..31
		b.Hand = int(r.u())      //nolint:gosec // ecrit depuis 0 ou 1
		b.Barrel = r.bool8()
		b.Input = int(r.u()) //nolint:gosec // ecrit depuis un rang 0..2
		b.Weapon = int(r.i())
		b.StartBound, b.EndBound = r.str(), r.str()
		b.Entries = int(r.u()) //nolint:gosec // compte ecrit positif
		nh := int(r.u())       //nolint:gosec // compte ecrit positif
		for j := 0; j < nh && r.err == nil; j++ {
			h := types.ContinuousFireHole{StartUS: b.StartUS + r.u()}
			h.EndUS = h.StartUS + r.u()
			b.Holes = append(b.Holes, h)
		}
		out = append(out, b)
	}
	var st types.ContinuousFireStats
	st.Scanned = r.bool8()
	for _, v := range compteursTirContinu(&st) {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	st.HeldHoleMS = r.i()
	return out, st
}

// compteursTirContinu rend les compteurs entiers des stats, dans l ORDRE DU FORMAT : ajouter un
// compteur, c est l ajouter ICI et monter `filmFactsMagic`.
func compteursTirContinu(st *types.ContinuousFireStats) []*int {
	return []*int{&st.Packets, &st.Reached, &st.Closed, &st.Holes, &st.HoleRuns, &st.Unlocated,
		&st.OpenViewB, &st.StopOverflow, &st.StopKind, &st.StopBlockBC, &st.StopCap, &st.NotClosing,
		&st.Entries, &st.WithAction, &st.Firing, &st.Bursts, &st.BurstsWithHole, &st.InnerHoles}
}
