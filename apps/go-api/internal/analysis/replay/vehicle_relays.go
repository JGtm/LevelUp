package replay

// vehicle_relays.go — LA FUSION DES VIES EN RELAIS du calque des vehicules.
//
// POURQUOI CE FICHIER EXISTE : DEPLACEMENT, PAS REECRITURE. vehicle_tracks.go arrivait a 538
// lignes, au-dela du seuil de 500 (CLAUDE.md n° 5). Le sujet le plus autonome de l assemblage
// en sort tel quel, sans une ligne de logique changee : reconnaitre qu une vie de vehicule en
// RELAIE une autre (meme chassis, meme point, la seconde nait dans l intervalle non observe de
// la premiere), et les fondre en une seule vie publiee.
//
// LA FRONTIERE : vehicle_tracks.go ASSEMBLE une vie a partir du film (recensement, creations,
// nuage de positions) ; ce fichier-ci decide, une fois les vies assemblees, LESQUELLES n en font
// qu une. Il ne lit rien du film — il ne travaille que sur des `VehicleTrack` deja construites.
// Le point d appel unique est `buildVehicleTracks`, juste apres `sortVehicleTracks`.

import "sort"

// mergeVehicleRelays fond les vies EN RELAIS : deux vies consecutives du MEME chassis, au MEME
// point, dont la seconde commence dans l intervalle non observe de la premiere. Rend la tranche
// fusionnee et le NOMBRE de fusions.
//
// POURQUOI ICI ET PAS AU RECENSEMENT. Le recensement voit des SLOTS, et deux slots differents y
// sont deux objets — il a raison, c est ce que le film dit. C est au niveau de la VIE PUBLIEE,
// une fois la naissance, la trajectoire et les bornes connues, que le relais se reconnait : il
// faut la position, le chassis ET les deux bornes d affichage pour trancher.
//
// LES CHAINES SONT COUVERTES (A -> B -> C) : la boucle repart tant qu une fusion a eu lieu, et
// la vie fusionnee porte la fin de B, donc C se presente ensuite comme le relais de A+B. L ordre
// est celui de `sortVehicleTracks` (T0, slot, generation) : la fusion est deterministe.
func mergeVehicleRelays(tracks []VehicleTrack) ([]VehicleTrack, int) {
	merged := 0
	for {
		i, j, ok := findVehicleRelay(tracks)
		if !ok {
			return tracks, merged
		}
		tracks[i] = mergeVehicleRelay(tracks[i], tracks[j])
		tracks = append(tracks[:j], tracks[j+1:]...)
		merged++
	}
}

// findVehicleRelay rend le PREMIER couple (vie finissante, vie relais) de la tranche.
func findVehicleRelay(tracks []VehicleTrack) (int, int, bool) {
	for i := range tracks {
		for j := range tracks {
			if i != j && isVehicleRelay(tracks[i], tracks[j]) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

// isVehicleRelay dit si `b` est la RE-CREATION de `a`. Les trois conditions sont cumulatives et
// aucune n est negociable : un chassis different est un autre vehicule, un point different est un
// autre emplacement, et un debut hors de l intervalle non observe est une coexistence reelle.
func isVehicleRelay(a, b VehicleTrack) bool {
	if a.Chassis == "" || a.Chassis != b.Chassis {
		return false
	}
	if b.T0 < a.T1 || b.T0 > a.T1Max+vehicleRelayMarginFrames {
		return false
	}
	ax, ay, aok := vehicleTrackLastPos(a)
	bx, by, bok := vehicleTrackFirstPos(b)
	return aok && bok && planDist(ax, ay, bx, by) <= vehicleRelayRadiusM
}

// mergeVehicleRelay fond `b` dans `a` : la NAISSANCE et le debut de `a`, la TRAJECTOIRE et les
// bornes de fin de `b`, les episodes des deux — reclampes a la fenetre resultante par le meme
// `clampVehicleRides` que l assemblage.
//
// L IDENTITE PUBLIEE EST CELLE DE `a`, la vie qui commence : c est elle qui porte le record de
// creation (position exacte de naissance) et le `t0` date a la milliseconde. Le slot de `b` ne
// disparait pas d une information utile — `(slot, gen)` n a de sens qu a l interieur du film.
func mergeVehicleRelay(a, b VehicleTrack) VehicleTrack {
	out := a
	out.Samples = appendVehicleSamples(a.Samples, b.Samples)
	out.T1, out.T1Max, out.End = b.T1, b.T1Max, b.End
	if out.T1Max < out.T1 {
		out.T1Max = out.T1
	}
	rides := append(append([]VehicleRide(nil), a.Rides...), b.Rides...)
	sort.SliceStable(rides, func(i, j int) bool {
		if rides[i].T0 != rides[j].T0 {
			return rides[i].T0 < rides[j].T0
		}
		return rides[i].Slot < rides[j].Slot
	})
	out.Rides = clampVehicleRides(rides, out.T0, out.T1Max)
	if out.Spawn == nil {
		out.Spawn = b.Spawn
	}
	return out
}

// appendVehicleSamples enchaine deux trajectoires en gardant l axe de frames STRICTEMENT
// croissant : un echantillon du relais anterieur au dernier point de la vie precedente serait un
// retour en arriere, et le client interpole entre points consecutifs.
func appendVehicleSamples(a, b []VehicleSample) []VehicleSample {
	if len(a) == 0 {
		return b
	}
	out := append([]VehicleSample(nil), a...)
	lastT := a[len(a)-1].T
	for _, s := range b {
		if s.T <= lastT {
			continue
		}
		out = append(out, s)
		lastT = s.T
	}
	return out
}

// vehicleTrackFirstPos rend la PREMIERE position connue d une vie : son premier echantillon, ou
// sa naissance quand elle n a jamais bouge.
func vehicleTrackFirstPos(tr VehicleTrack) (x, y float32, ok bool) {
	if len(tr.Samples) > 0 {
		return tr.Samples[0].X, tr.Samples[0].Y, true
	}
	if tr.Spawn != nil {
		return tr.Spawn.X, tr.Spawn.Y, true
	}
	return 0, 0, false
}

// vehicleTrackLastPos rend la DERNIERE position connue d une vie, meme repli.
func vehicleTrackLastPos(tr VehicleTrack) (x, y float32, ok bool) {
	if n := len(tr.Samples); n > 0 {
		return tr.Samples[n-1].X, tr.Samples[n-1].Y, true
	}
	if tr.Spawn != nil {
		return tr.Spawn.X, tr.Spawn.Y, true
	}
	return 0, 0, false
}
