package replay

// vehicules_v4_tirs_test.go — INSTRUMENT DE MESURE (lot V4, etages 2 et 3) : LES TIRS DES
// JOUEURS EMBARQUES, et la CO-MOBILITE qui dirait ce que les episodes ratent. LECTURE SEULE,
// garde par V4_ROOT / V4_FILMS. Les DEUX etages partagent le decodage d un film — il coute
// ~3 min par film, le dedoubler doublerait la mesure sans rien apprendre.
//
// LE CONSTAT QUI OUVRE L ETAGE 2. Sur `0d76e8f1`, 1 166 tirs sont publies, 12 episodes
// d occupation aussi — et PAS UN SEUL tir ne tombe pendant un episode. Dans le meme temps
// 203 evenements de tir sont ecartes « sans slot ». Ce n est pas une coincidence : un joueur
// embarque cesse de repliquer la position de son bipede (V1a.4), donc `slotFor` ne trouve
// AUCUNE position a moins de 120 ms de son tir, et le tir tombe.
//
// CE QUE L EVENEMENT PORTE, ET CE QU IL NE PORTE PAS. Le record 105 porte le TIREUR
// (`FilmIndex`, ecrit dans le film, ni devine ni vote), l ARME, et l INSTANT. Il ne porte AUCUNE
// position monde — la position publiee d un tir a toujours ete celle du BIPEDE a cet instant
// (`shots.go`), et c est justement ce qui manque a un joueur embarque. La geometrie ne peut donc
// pas servir de critere de rattachement : il n y a rien a comparer a la position du vehicule.
// LE CRITERE EST L IDENTITE — le tireur est nomme, l episode nomme son occupant, l instant les
// recoupe. Son temoin est donc TEMPOREL.
//
// LE SECOND TEMOIN, INDEPENDANT ET DECISIF : L ARME. Les identifiants d arme du film se lisent
// en deux moities de 32 bits ; les armes PERSONNELLES portent toutes la meme moitie basse
// (`0x42C9679F` sur les 1 166 tirs publies de `0d76e8f1`, sans exception). Une moitie basse
// NULLE designe une autre famille — celle qu on ne peut pas porter a pied. Un tir dont l arme
// est de cette famille prouve, SANS AUCUNE GEOMETRIE, que son tireur etait a bord.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v4TemoinsFrames sont les decalages du temoin des tirs, en frames (100 ms) : +/-30, +/-60 et
// +/-120 s. UN SEUL decalage ne mesure rien de fiable — il peut tomber sur un creux ou sur une
// autre periode d orphelins du meme joueur. Le temoin publie est la MOYENNE.
var v4TemoinsFrames = []int{-1200, -600, -300, 300, 600, 1200}

// v4ArmePersoBasse est la moitie basse commune a TOUTES les armes personnelles observees.
// Mesure du 2026-09-02 : 1 166 / 1 166 tirs publies de `0d76e8f1` la portent, et les 19 familles
// distinctes du film aussi. Elle sert de SEPARATEUR, jamais de table d armes.
const v4ArmePersoBasse = uint32(0x42C9679F)

// v4TirAgg agrege l etage 2 pour UN film.
type v4TirAgg struct {
	dispo, attaches, sansSlot, ambigus, horsFenetre int
	// pendantEpisode : tirs ecartes qui tombent dans UN episode de leur tireur.
	pendantEpisode int
	// plusieursEpisodes : tirs ecartes couverts par DEUX vehicules distincts au meme instant.
	plusieursEpisodes int
	// temoin : somme des rattachements aux MEMES episodes decales (cf. v4TemoinsFrames).
	temoin int
	// armeVehicule* ventilent selon la moitie basse NULLE de l identifiant d arme.
	armeVehiculeTotal, armeVehiculeRejete, armeVehiculeEnEpisode int
	// armePersoEnEpisode : les tirs d arme PERSONNELLE qui tombent quand meme dans un episode
	// (passager qui tire au fusil : legitime, mais il faut le compter a part).
	armePersoEnEpisode int
	parArme            map[string]int
}

// TestV4Diagnostic — ETAGES 2 (tirs) et 3 (co-mobilite), un decodage par film.
func TestV4Diagnostic(t *testing.T) {
	root := v4Root(t)
	for _, f := range v4Corpus(t) {
		v4DiagnosticUnFilm(t, root, f)
	}
}

func v4DiagnosticUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	ctx, ok := v4Decode(t, root, f)
	if !ok {
		return
	}
	tracks, cov, _ := buildVehicleTracks(ctx.scan, ctx.bip, ctx.own, ctx.clock)
	ag := v4MesureTirs(ctx, tracks)
	t.Logf("V4-TIRS %s (%s) — episodes=%d nommes=%d | tirs dispo=%d rattaches=%d sansSlot=%d"+
		" ambigus=%d horsFenetre=%d",
		f.ID, f.Carte, cov.Rides, cov.RidesNamed, ag.dispo, ag.attaches, ag.sansSlot,
		ag.ambigus, ag.horsFenetre)
	t.Logf("V4-TIRS %s ORPHELINS PENDANT UN EPISODE : %d (deux vehicules : %d) — TEMOIN moyen"+
		" sur %d decalages : %.1f", f.ID, ag.pendantEpisode, ag.plusieursEpisodes,
		len(v4TemoinsFrames), float64(ag.temoin)/float64(len(v4TemoinsFrames)))
	t.Logf("V4-TIRS %s ARME DE VEHICULE (moitie basse nulle) : %d evenements, dont %d ecartes"+
		" par la porte du bipede, dont %d dans un episode | arme PERSONNELLE dans un episode : %d",
		f.ID, ag.armeVehiculeTotal, ag.armeVehiculeRejete, ag.armeVehiculeEnEpisode,
		ag.armePersoEnEpisode)
	t.Logf("V4-TIRS %s armes des orphelins en episode : %s", f.ID, v4ArmesLigne(ag.parArme))
	v4OracleMesureFilm(t, ctx, tracks)
	v4CoMesureFilm(t, ctx)
}

// v4RideRef designe un episode ET la vie de vehicule qui le porte.
type v4RideRef struct {
	vehSlot uint32
	ride    VehicleRide
}

// v4RidesByOccupant indexe les episodes publies par slot d occupant.
func v4RidesByOccupant(tracks []VehicleTrack) map[uint32][]v4RideRef {
	out := map[uint32][]v4RideRef{}
	for _, tr := range tracks {
		for _, r := range tr.Rides {
			out[r.Slot] = append(out[r.Slot], v4RideRef{vehSlot: tr.Slot, ride: r})
		}
	}
	return out
}

// v4MesureTirs rejoue la porte de production sur chaque evenement de tir, puis interroge les
// episodes pour ceux qu elle ecarte.
func v4MesureTirs(ctx v4Ctx, tracks []VehicleTrack) v4TirAgg {
	ag := v4TirAgg{dispo: len(ctx.fire), parArme: map[string]int{}}
	slotTracks := indexBySlot(ctx.bip)
	rides := v4RidesByOccupant(tracks)
	slotsOf := v4SlotsParJoueur(ctx.own.Owner)
	for _, e := range ctx.fire {
		vehArme := e.WeaponID != 0 && uint32(e.WeaponID) != v4ArmePersoBasse
		if vehArme {
			ag.armeVehiculeTotal++
		}
		if v4TirRattache(ctx, slotTracks, e, &ag) {
			continue
		}
		if vehArme {
			ag.armeVehiculeRejete++
		}
		v4TirOrphelin(ctx, rides, slotsOf[e.FilmIndex], v4TirArg{ev: e, vehArme: vehArme}, &ag)
	}
	return ag
}

// v4TirArg porte les entrees d UN orphelin (regle des 5 parametres du depot).
type v4TirArg struct {
	ev      filmdec.FireEvent
	vehArme bool
}

// v4TirRattache rejoue la PREMIERE porte (celle du bipede) et compte sa cause de rejet. Rend
// vrai quand le tir est rattache — l orphelin n existe pas.
func v4TirRattache(
	ctx v4Ctx, slotTracks map[uint32]slotTrack, e filmdec.FireEvent, ag *v4TirAgg,
) bool {
	slot, reason := slotFor(slotTracks, ctx.own.Owner, e.FilmIndex, e.TimestampUS)
	if reason == reasonAttached {
		if p, d := slotTracks[slot].at(e.TimestampUS); d <= shotPosToleranceUS && p.HasWorld {
			ag.attaches++
			return true
		}
		ag.horsFenetre++
		return false
	}
	switch reason {
	case reasonNoSlot:
		ag.sansSlot++
	case reasonAmbiguous:
		ag.ambigus++
	}
	return false
}

// v4TirOrphelin interroge les episodes du tireur, au reel puis a chaque decalage temoin.
func v4TirOrphelin(
	ctx v4Ctx, rides map[uint32][]v4RideRef, slots []uint32, a v4TirArg, ag *v4TirAgg,
) {
	fr := ctx.clock.frame(a.ev.TimestampUS)
	if n := v4CompteEpisodes(rides, slots, fr, 0); n > 0 {
		ag.pendantEpisode++
		if n > 1 {
			ag.plusieursEpisodes++
		}
		if a.vehArme {
			ag.armeVehiculeEnEpisode++
		} else {
			ag.armePersoEnEpisode++
		}
		if a.ev.WeaponID != 0 {
			ag.parArme[formatWeaponID(a.ev.WeaponID)]++
		}
	}
	for _, d := range v4TemoinsFrames {
		if v4CompteEpisodes(rides, slots, fr, d) > 0 {
			ag.temoin++
		}
	}
}

// v4SlotsParJoueur inverse le pont slot -> index de joueur.
func v4SlotsParJoueur(owner map[uint32]int) map[int][]uint32 {
	out := map[int][]uint32{}
	for s, pi := range owner {
		out[pi] = append(out[pi], s)
	}
	for pi := range out {
		sort.Slice(out[pi], func(i, j int) bool { return out[pi][i] < out[pi][j] })
	}
	return out
}

// v4CompteEpisodes compte les episodes d un joueur qui couvrent une frame, les bornes des
// episodes etant DECALEES de `decal` frames (les temoins).
func v4CompteEpisodes(
	rides map[uint32][]v4RideRef, slots []uint32, frame, decal int,
) int {
	n := 0
	for _, s := range slots {
		for _, rr := range rides[s] {
			if frame >= rr.ride.T0+decal && frame <= rr.ride.T1+decal {
				n++
			}
		}
	}
	return n
}

// v4ArmesLigne rend la ventilation par arme, triee, sur une ligne.
func v4ArmesLigne(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%s=%d ", k, m[k])
	}
	return s
}
