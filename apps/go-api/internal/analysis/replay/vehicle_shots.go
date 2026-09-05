package replay

// vehicle_shots.go — LA SECONDE PORTE DES TIRS : celle des joueurs EMBARQUÉS.
//
// LE CONSTAT MESURÉ QUI OUVRE CE FICHIER (2026-09-02, instrument `vehicules_v4_tirs_test.go`).
// Sur `0d76e8f1`, le document publiait 1 166 tirs et 12 épisodes d'occupation — et PAS UN SEUL
// tir ne tombait pendant un épisode, alors que 203 événements de tir étaient écartés « sans
// slot ». Ce n'est pas une coïncidence : un occupant attaché cesse de répliquer la position de
// son bipède (primitive V1a.4), donc `slotFor` ne trouve AUCUNE position à moins de 120 ms de
// son tir et le tir tombe. Le calque des tirs était muet exactement là où le joueur est le plus
// visible — au volant.
//
// CE QUE L'ÉVÉNEMENT PORTE, ET CE QU'IL NE PORTE PAS — c'est ce qui dicte le critère.
// Le record de tir (type 105) porte son TIREUR (`FilmIndex`, ÉCRIT dans le film : ni deviné, ni
// voté), son ARME et son INSTANT. Il ne porte AUCUNE position monde : la position publiée d'un
// tir a toujours été celle du BIPÈDE à cet instant (`shots.go`), et c'est précisément ce qui
// manque à un embarqué. UN CRITÈRE GÉOMÉTRIQUE EST DONC IMPOSSIBLE ICI — il n'y a rien à
// comparer à la position du véhicule. Le critère est l'IDENTITÉ, et il est plus fort qu'un
// rayon : le film nomme le tireur, l'épisode d'occupation nomme son occupant, et l'instant les
// recoupe. La géométrie n'intervient qu'APRÈS, pour POSER le tir — jamais pour le rattacher.
//
// LE TÉMOIN QUI VA AVEC est donc TEMPOREL, pas spatial : les mêmes tirs contre les mêmes
// épisodes décalés de 60 s. Mesure du 2026-09-02, écrite avant la bascule et rejouée après.
//
// CE QUE CETTE PORTE NE REPREND PAS : les tirs AMBIGUS (`reasonAmbiguous`). L'ambiguïté naît de
// DEUX slots du même joueur qui répliquent tous deux une position à cet instant — donc d'un
// joueur qui n'est PAS embarqué. Les reprendre mélangerait un défaut du pont slot -> joueur avec
// un embarquement, et gonflerait le compte sans rien mesurer.
//
// PUR : aucune I/O. Appelé par `BuildFromPositions` APRÈS `attachVehicles` — il lui faut les
// épisodes publiés et les trajectoires sur lesquelles poser les tirs.

import (
	"log/slog"
	"sort"
)

// vehicleWeaponLowHalf est la moitié BASSE de 32 bits que portent TOUTES les armes personnelles
// observées. Elle sert de TÉMOIN, jamais de porte : aucune décision de ce fichier n'en dépend.
//
// MESURE DU 2026-09-02 (`vehicules_v4_tirs_test.go`, films `0d76e8f1` et `fccc61cd`). Les 1 166
// tirs publiés de `0d76e8f1` la portent tous, sur les 19 familles d'armes personnelles du match
// — sans exception. Les 23 événements dont la moitié basse est NULLE, eux, sont TOUS écartés par
// la porte du bipède (23/23) : une arme qu'on ne porte pas à pied, tirée par un joueur qui ne
// réplique plus. 17 de ces 23 tombent dans un épisode publié (73,9 %), contre 6 des 229 orphelins
// d'arme personnelle (2,6 %) — enrichissement x28, et c'est le témoin le plus fort du calque
// parce qu'il n'emprunte RIEN au critère de rattachement (ni instant, ni identité, ni géométrie).
//
// SI ELLE CHANGEAIT (nouveau build, nouveau titre), SEUL UN COMPTEUR SERAIT FAUX : la publication
// ne la lit pas. C'est délibéré — un séparateur empirique ne doit jamais commander une sortie.
const vehicleWeaponLowHalf = uint32(0x42C9679F)

// vehicleShotRide désigne UN épisode d'occupation ET la vie de véhicule qui le porte.
type vehicleShotRide struct {
	// track est le rang de la vie dans `doc.Vehicles` : la trajectoire où lire la position.
	track int
	ride  VehicleRide
}

// attachVehicleShots publie, dans `doc.Shots`, les tirs des joueurs embarqués et met la
// couverture à jour. Rien à faire sans épisode ni orphelin — le document sort inchangé.
func attachVehicleShots(
	doc *ReplayDocument, orphans []orphanShot, own OwnerReport, clock replayClock,
) {
	cov := doc.Coverage
	if len(orphans) == 0 || len(doc.Vehicles) == 0 || cov == nil {
		return
	}
	rides := vehicleRidesByOccupant(doc.Vehicles)
	if len(rides) == 0 {
		return
	}
	slotsOf := vehicleSlotsByPlayer(own.Owner)
	published := publishedSlots(doc.Tracks)
	var added []Shot
	for _, o := range orphans {
		s, verdict := vehicleShotOf(o, rides, slotsOf[o.ev.FilmIndex], doc.Vehicles, clock)
		vehicleShotTally(doc.Coverage.Vehicles, verdict)
		if verdict != vehicleShotPlaced {
			continue
		}
		// MÊME PORTE QUE LES TIRS À PIED : un tir dont le tireur n'a aucune trajectoire publiée
		// ne rencontrerait jamais de piste à décorer côté client (règle de
		// `keepShotsOfPublishedTracks`). Il change alors de compteur, il ne disparaît pas.
		if !published[s.Slot] {
			vehicleShotMove(&cov.Shots, o.reason, false)
			continue
		}
		vehicleShotMove(&cov.Shots, o.reason, true)
		if cov.Vehicles != nil && o.ev.WeaponID != 0 &&
			uint32(o.ev.WeaponID) != vehicleWeaponLowHalf {
			cov.Vehicles.ShotsVehicleWeapon++
		}
		added = append(added, s)
	}
	if len(added) == 0 {
		return
	}
	doc.Shots = append(doc.Shots, added...)
	sort.SliceStable(doc.Shots, func(i, j int) bool { return doc.Shots[i].T < doc.Shots[j].T })
	cov.Verdict["shots"] = verdictOf(cov.Shots)
	logVehicleShots(doc.Coverage.Vehicles, len(added))
}

// Verdicts d'un orphelin passé devant la porte du véhicule.
type vehicleShotVerdict int

const (
	// vehicleShotNoRide : aucun épisode de ce tireur ne couvre l'instant. C'est le cas nominal
	// d'un tir à pied que le pont n'a pas su placer — il n'a rien à voir avec un véhicule.
	vehicleShotNoRide vehicleShotVerdict = iota
	// vehicleShotAmbiguous : DEUX véhicules distincts portent un épisode de ce tireur au même
	// instant. Physiquement impossible : c'est un artefact du pont ou du découpage des vies. On
	// ne tranche pas, on compte.
	vehicleShotAmbiguous
	// vehicleShotUnplaced : l'épisode existe mais la vie de véhicule n'a NI échantillon NI
	// naissance à cet instant — rien où poser le tir.
	vehicleShotUnplaced
	// vehicleShotPlaced : tir posé sur le véhicule.
	vehicleShotPlaced
)

// vehicleShotOf décide du sort d'UN orphelin.
func vehicleShotOf(
	o orphanShot, rides map[uint32][]vehicleShotRide, slots []uint32,
	tracks []VehicleTrack, clock replayClock,
) (Shot, vehicleShotVerdict) {
	fr := clock.frame(o.ev.TimestampUS)
	cand := vehicleShotCandidates(rides, slots, fr)
	if len(cand) == 0 {
		return Shot{}, vehicleShotNoRide
	}
	pick := cand[0]
	for _, c := range cand[1:] {
		if c.track != pick.track {
			return Shot{}, vehicleShotAmbiguous
		}
	}
	x, y, ok := vehiclePosAt(tracks[pick.track], fr)
	if !ok {
		return Shot{}, vehicleShotUnplaced
	}
	v := tracks[pick.track].Slot
	s := Shot{T: fr, Slot: pick.ride.Slot, X: round2(x), Y: round2(y), Vehicle: &v}
	if h, ok := o.ev.AimHeadingDeg(); ok {
		s.H = headingForJSON(float32(h))
	}
	if o.ev.WeaponID != 0 {
		s.Weapon = formatWeaponID(o.ev.WeaponID)
	}
	return s, vehicleShotPlaced
}

// vehicleShotCandidates rend les épisodes d'un tireur qui couvrent une frame. L'ordre est
// déterministe : siège croissant (le CONDUCTEUR d'abord — décision de cadrage du calque), puis
// rang de la vie, puis slot d'occupant.
func vehicleShotCandidates(
	rides map[uint32][]vehicleShotRide, slots []uint32, frame int,
) []vehicleShotRide {
	var out []vehicleShotRide
	for _, s := range slots {
		for _, rr := range rides[s] {
			if frame >= rr.ride.T0 && frame <= rr.ride.T1 {
				out = append(out, rr)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := vehicleSeatRank(out[i].ride.Seat), vehicleSeatRank(out[j].ride.Seat)
		switch {
		case si != sj:
			return si < sj
		case out[i].track != out[j].track:
			return out[i].track < out[j].track
		default:
			return out[i].ride.Slot < out[j].ride.Slot
		}
	})
	return out
}

// vehicleSeatRank ordonne les sièges, un siège NON LU passant après tous les autres.
func vehicleSeatRank(seat *int) int {
	if seat == nil {
		return 1 << 30
	}
	return *seat
}

// vehiclePosAt rend la position INTERPOLÉE d'une vie de véhicule à une frame.
//
// L'INTERPOLATION EST CELLE DU CLIENT, et c'est la seule justesse qui compte : le tir doit
// sortir de l'endroit où le sprite EST dessiné, pas d'un échantillon voisin. Hors de la plage
// des échantillons, la position est TENUE (premier / dernier échantillon) plutôt qu'extrapolée —
// extrapoler la vitesse d'un véhicule ferait sortir le tir de la carte. Sans aucun échantillon,
// la NAISSANCE : un véhicule que personne n'a bougé est là où il est né.
func vehiclePosAt(tr VehicleTrack, frame int) (x, y float32, ok bool) {
	n := len(tr.Samples)
	if n == 0 {
		if tr.Spawn == nil {
			return 0, 0, false
		}
		return tr.Spawn.X, tr.Spawn.Y, true
	}
	i := sort.Search(n, func(k int) bool { return tr.Samples[k].T >= frame })
	switch {
	case i == 0:
		return tr.Samples[0].X, tr.Samples[0].Y, true
	case i >= n:
		return tr.Samples[n-1].X, tr.Samples[n-1].Y, true
	}
	a, b := tr.Samples[i-1], tr.Samples[i]
	if b.T == a.T {
		return b.X, b.Y, true
	}
	f := float32(frame-a.T) / float32(b.T-a.T)
	return a.X + (b.X-a.X)*f, a.Y + (b.Y-a.Y)*f, true
}

// vehicleRidesByOccupant indexe les épisodes publiés par slot d'occupant.
func vehicleRidesByOccupant(tracks []VehicleTrack) map[uint32][]vehicleShotRide {
	out := map[uint32][]vehicleShotRide{}
	for i, tr := range tracks {
		for _, r := range tr.Rides {
			out[r.Slot] = append(out[r.Slot], vehicleShotRide{track: i, ride: r})
		}
	}
	return out
}

// vehicleSlotsByPlayer inverse le pont slot -> index de joueur du film. Un joueur porte
// plusieurs slots dans un match (un par réapparition) : ils sont TRIÉS, pour que la décision
// d'ambiguïté ne dépende pas de l'ordre d'itération d'une map.
func vehicleSlotsByPlayer(owner map[uint32]int) map[int][]uint32 {
	out := map[int][]uint32{}
	for s, pi := range owner {
		out[pi] = append(out[pi], s)
	}
	for pi := range out {
		sort.Slice(out[pi], func(i, j int) bool { return out[pi][i] < out[pi][j] })
	}
	return out
}

// vehicleShotMove déplace un orphelin de son compteur de rejet vers `Attached` (publié) ou
// `Unpublished` (rattaché, mais sans trajectoire publiée). L'INVARIANT `Balanced()` tient :
// un événement change de case, il n'en gagne ni n'en perd aucune.
func vehicleShotMove(c *LayerCoverage, reason rejectReason, publishedTrack bool) {
	switch reason {
	case reasonNoSlot:
		c.NoSlot--
	case reasonOutOfWindow:
		c.OutOfWindow--
	case reasonAttached, reasonAmbiguous:
		return // jamais orphelins : cf. l'en-tête de `buildShots`
	}
	if publishedTrack {
		c.Attached++
		return
	}
	c.Unpublished++
}

// vehicleShotTally compte le verdict dans la couverture du calque des véhicules.
func vehicleShotTally(cov *VehicleCoverage, v vehicleShotVerdict) {
	if cov == nil {
		return
	}
	switch v {
	case vehicleShotPlaced:
		cov.Shots++
	case vehicleShotAmbiguous:
		cov.ShotsAmbiguous++
	case vehicleShotUnplaced:
		cov.ShotsUnplaced++
	case vehicleShotNoRide:
		cov.ShotsNoRide++
	}
}

// logVehicleShots journalise la seconde porte avec ses dénominateurs.
func logVehicleShots(cov *VehicleCoverage, published int) {
	if cov == nil {
		return
	}
	slog.Info("rejeu : tirs en vehicule",
		"publies", published, "poses", cov.Shots, "ambigus", cov.ShotsAmbiguous,
		"sansPosition", cov.ShotsUnplaced, "orphelinsHorsEpisode", cov.ShotsNoRide)
}
