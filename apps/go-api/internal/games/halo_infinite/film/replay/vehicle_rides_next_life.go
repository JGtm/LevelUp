package replay

// vehicle_rides_next_life.go — UN JOUEUR N EST JAMAIS A DEUX ENDROITS : un episode d occupation
// s arrete quand une AUTRE vie publiee du meme joueur commence (reprise du lot M7b des retours du
// rejeu, revue adverse RR-M7b-01, 2026-09-24).
//
// LE DEFAUT. Un episode d occupation est borne par ses propres signaux (sortie lue, reapparition du
// MEME slot, fin de vie du vehicule). Or un joueur qui meurt a bord REAPPARAIT SOUS UN AUTRE SLOT de
// bipede : rien, dans ces signaux, ne ferme alors son episode, qui peut courir jusqu a la fin de vie
// du vehicule. A l ecran, son nom reste ecrit sur le vehicule pendant que sa nouvelle vie marche
// ailleurs — le meme joueur deux fois sur la carte, ce que la regle des places de l utilisateur
// (2026-09-23) interdit. MESURE du 2026-09-24 (107 documents rejoues des faits) : 18 episodes dans
// 5 documents, TOUTES familles et toutes sources (lus comme de repli), dont un episode lu de
// Warthog (`c259789d`) qui recouvrait TROIS vies suivantes du meme joueur ; 2 sont ecartes en amont
// par la garde de montee a bord (`vehicle_turrets_boarding.go`), 16 sont coupes ici.
//
// LA REGLE, TITRE-AGNOSTIQUE ET SANS SEUIL : la naissance d une autre vie publiee du meme joueur
// (meme xuid, ou meme nom de bot) PROUVE que l episode precedent est fini. L episode est coupe a la
// frame qui la precede, sa visee avec lui. Rien n est efface : la montee a bord reste publiee.
// AUCUNE CONSTANTE, AUCUN SEUIL : c est la naissance que le film ecrit qui borne l episode. Mais le
// film ecrit aussi la VRAIE fin (la mort a bord), que le calque ne lit pas pour cela : la borne est
// donc un repli nomme, `repli_episode_borne_par_la_vie_suivante`, compte a chaque coupe
// (`coverage.fallbacks`), et qui se retire quand la fin lue le rend muet.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// cutRidesAtNextLife coupe les episodes de `vehicles` a la naissance de la vie suivante du meme
// joueur parmi `players` (les pistes PUBLIEES, deja nommees), et rend le nombre de coupes.
func cutRidesAtNextLife(vehicles []VehicleTrack, players []Track, fb *fallback.Compteur) int {
	births := playerBirthsByIdentity(players)
	if len(births) == 0 {
		return 0
	}
	cuts := 0
	for i := range vehicles {
		for j := range vehicles[i].Rides {
			r := &vehicles[i].Rides[j]
			next, ok := nextBirthAfter(births[rideIdentity(*r, players)], r.T0)
			if !ok || next > r.T1 {
				continue
			}
			r.T1 = next - 1
			r.Aim = clampVehicleRideAim(r.Aim, r.T0, r.T1)
			fb.Declenche(fallback.NomEpisodeBorneParLaVieSuivante)
			cuts++
		}
	}
	return cuts
}

// playerBirthsByIdentity rend, par identite de joueur, les frames de naissance de ses vies
// publiees, triees.
func playerBirthsByIdentity(players []Track) map[string][]int {
	out := map[string][]int{}
	for _, t := range players {
		if id := trackIdentity(t); id != "" {
			out[id] = append(out[id], t.StartFrame)
		}
	}
	for id := range out {
		sort.Ints(out[id])
	}
	return out
}

// trackIdentity : le xuid d une vie, ou le nom de son bot (un bot n a pas de xuid). Vide : vie
// sans nom, qui ne prouve rien sur personne.
func trackIdentity(t Track) string {
	switch {
	case t.XUID != "":
		return "xuid:" + t.XUID
	case t.Bot != "":
		return "bot:" + t.Bot
	}
	return ""
}

// rideIdentity : l identite de l occupant d un episode — son xuid publie, sinon celle de la vie
// de bipede d ou il est monte (la derniere piste de son slot nee au plus tard a son debut).
func rideIdentity(r VehicleRide, players []Track) string {
	if r.XUID != "" {
		return "xuid:" + r.XUID
	}
	best, found := Track{}, false
	for _, t := range players {
		if t.Slot == r.Slot && t.StartFrame <= r.T0 && (!found || t.StartFrame > best.StartFrame) {
			best, found = t, true
		}
	}
	if !found {
		return ""
	}
	return trackIdentity(best)
}

// nextBirthAfter rend la premiere naissance STRICTEMENT posterieure a `t0` dans une liste triee.
func nextBirthAfter(births []int, t0 int) (int, bool) {
	i := sort.SearchInts(births, t0+1)
	if i >= len(births) {
		return 0, false
	}
	return births[i], true
}

// recountVehicleRides recompte les compteurs d EPISODES de la couverture sur les vies telles
// qu elles sont publiees — apres une coupe, la duree couverte, la visee et les chevauchements ont
// change. Les autres compteurs du calque ne dependent pas des episodes et restent.
func recountVehicleRides(tracks []VehicleTrack, cov *VehicleCoverage) {
	cov.VehiclesRidden, cov.Rides, cov.RidesNamed, cov.RidesWithSeat = 0, 0, 0, 0
	cov.AimRideFrames, cov.RidesWithAim, cov.AimSamples = 0, 0, 0
	cov.RidesRead, cov.RidesProximity, cov.Ambiguous = 0, 0, 0
	for _, tr := range tracks {
		tallyVehicleRides(tr.Rides, cov)
	}
}
