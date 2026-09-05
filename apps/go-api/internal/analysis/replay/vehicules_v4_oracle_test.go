package replay

// vehicules_v4_oracle_test.go — INSTRUMENT DE MESURE (lot V4, etage 4) : L ORACLE DE
// L EMBARQUEMENT, et ce que les episodes publies en ratent. LECTURE SEULE, appele par
// `TestV4Diagnostic` (qui detient deja le decodage du film).
//
// L ORACLE, ET POURQUOI C EN EST UN. Les identifiants d arme du film se lisent en deux moities
// de 32 bits. Sur `0d76e8f1`, les 1 166 tirs PUBLIES portent tous la meme moitie basse
// (`0x42C9679F`) et couvrent les 19 familles d armes personnelles du match — sans exception. Les
// 23 evenements dont la moitie basse est NULLE, eux, sont TOUS ecartes par la porte du bipede
// (23/23). Une arme qu on ne peut pas porter a pied, tiree par un joueur dont le bipede ne
// replique plus : c est un TIR DEPUIS UN VEHICULE, etabli sans aucune geometrie et sans aucun
// episode. C est donc le denominateur honnete du calque d occupation — la seule population dont
// on sait, independamment, qu elle DEVRAIT etre couverte.
//
// CE QUE CET ETAGE MESURE : pour chaque tir de l oracle, l etat du flux de son tireur. Un tir
// non couvert par un episode designe une CAUSE precise et une seule — silence trop court pour
// ouvrir un trou, silence TERMINAL (aucun point apres : le trou n existe pas), vehicule trop
// loin du dernier point, ou instant hors de la fenetre d une vie. Sans cette ventilation, « il
// manque des episodes » ne designe aucun chantier.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v4OracleCause nomme l etat du flux du tireur au moment d un tir de l oracle.
type v4OracleCause int

const (
	v4OracleCouvert          v4OracleCause = iota // un episode publie couvre le tir
	v4OracleSilenceCourte                         // silence ferme, mais sous vehicleGapMinMS
	v4OracleSilenceTerminale                      // aucun point apres : le trou n a pas de bord droit
	v4OracleTropLoin                              // trou assez long, mais aucun vehicule sous le rayon
	v4OracleHorsVie                               // vehicule trouve, mais l instant sort de sa fenetre
	v4OracleSansSilence                           // le bipede repliquait : le tir aurait du se rattacher
	v4OracleSansSlot                              // aucun slot connu pour ce joueur
	v4OracleNbCauses
)

var v4OracleNoms = [v4OracleNbCauses]string{
	"COUVERT", "silence<3s", "silence TERMINAL", "vehicule trop loin",
	"hors fenetre de vie", "bipede repliquait", "aucun slot",
}

// v4OracleAgg agrege l etage 4.
type v4OracleAgg struct {
	total     int
	parCause  [v4OracleNbCauses]int
	distances []float64
	silences  []float64
}

// v4OracleMesureFilm publie l etage 4 pour UN film.
func v4OracleMesureFilm(t *testing.T, ctx v4Ctx, tracks []VehicleTrack) {
	t.Helper()
	slotTracks := indexBySlot(ctx.bip)
	rides := v4RidesByOccupant(tracks)
	slotsOf := v4SlotsParJoueur(ctx.own.Owner)
	var ag v4OracleAgg
	for _, e := range ctx.fire {
		if e.WeaponID == 0 || uint32(e.WeaponID) == v4ArmePersoBasse {
			continue
		}
		ag.total++
		v4OracleUnTir(ctx, slotTracks, rides, slotsOf[e.FilmIndex], e, &ag)
	}
	t.Logf("V4-ORACLE %s (%s) — tirs d arme de vehicule : %d · %s",
		ctx.film.ID, ctx.film.Carte, ag.total, v4OracleLigne(ag))
	if len(ag.distances) > 0 {
		p := v4Percentiles(ag.distances, 0.5, 0.9)
		q := v4Percentiles(ag.silences, 0.5, 0.9)
		t.Logf("V4-ORACLE %s NON COUVERTS — distance dernier point -> vehicule : d50=%.1f d90=%.1f"+
			" · duree du silence : d50=%.1fs d90=%.1fs",
			ctx.film.ID, p[0], p[1], q[0], q[1])
	}
}

// v4OracleUnTir classe UN tir de l oracle.
func v4OracleUnTir(
	ctx v4Ctx, slotTracks map[uint32]slotTrack, rides map[uint32][]v4RideRef,
	slots []uint32, e filmdec.FireEvent, ag *v4OracleAgg,
) {
	if v4CompteEpisodes(rides, slots, ctx.clock.frame(e.TimestampUS), 0) > 0 {
		ag.parCause[v4OracleCouvert]++
		return
	}
	if len(slots) == 0 {
		ag.parCause[v4OracleSansSlot]++
		return
	}
	cause, dist, silence := v4OracleEtat(ctx, slotTracks, slots, e.TimestampUS)
	ag.parCause[cause]++
	if cause != v4OracleSansSilence {
		ag.distances = append(ag.distances, dist)
		ag.silences = append(ag.silences, silence)
	}
}

// v4OracleEtat rend la cause la plus favorable parmi les slots du tireur : celle du slot dont le
// silence contient l instant. Rend aussi la distance au vehicule le plus proche du dernier point
// avant le silence, et la duree du silence en secondes.
func v4OracleEtat(
	ctx v4Ctx, slotTracks map[uint32]slotTrack, slots []uint32, atUS uint64,
) (v4OracleCause, float64, float64) {
	best, bestDist, bestSil := v4OracleSansSlot, 0.0, 0.0
	for _, s := range slots {
		pts := slotTracks[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > atUS })
		if i == 0 {
			continue // le flux de ce slot commence apres le tir : ce n est pas le bon slot
		}
		last := pts[i-1]
		if gapUS(last.TimestampUS, atUS) <= shotPosToleranceUS {
			return v4OracleSansSilence, 0, 0
		}
		terminal := i >= len(pts)
		sil := float64(atUS-last.TimestampUS) / 1e6
		if !terminal {
			sil = float64(pts[i].TimestampUS-last.TimestampUS) / 1e6
		}
		cause, dist := v4OracleClasse(ctx, last, terminal, sil)
		if cause < best || best == v4OracleSansSlot {
			best, bestDist, bestSil = cause, dist, sil
		}
	}
	return best, bestDist, bestSil
}

// v4OracleClasse dit ce qui manque a UN silence pour devenir un episode.
func v4OracleClasse(
	ctx v4Ctx, last filmdec.BipedPosition, terminal bool, silenceS float64,
) (v4OracleCause, float64) {
	slot, dist, _, ok := v4NearestHeld(ctx, last)
	switch {
	case terminal:
		return v4OracleSilenceTerminale, dist
	case silenceS*1000 < float64(vehicleGapMinMS):
		return v4OracleSilenceCourte, dist
	case !ok || dist > vehicleBoardRadiusM:
		return v4OracleTropLoin, dist
	}
	if _, alive := vehicleLifeAt(ctx.lives, slot, last.TimestampUS); !alive {
		return v4OracleHorsVie, dist
	}
	return v4OracleTropLoin, dist
}

// v4OracleLigne rend la ventilation par cause.
func v4OracleLigne(ag v4OracleAgg) string {
	s := ""
	for i, n := range ag.parCause {
		if n == 0 {
			continue
		}
		s += fmt.Sprintf("%s=%d ", v4OracleNoms[i], n)
	}
	return s
}
