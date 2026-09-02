package replay

// vehicle_tracks.go — L ASSEMBLAGE PUR du calque des vehicules : de ce que le film rend
// (recensement, creations, nuage de positions) a la VIE publiee.
//
// L OBJET PUBLIE EST LA VIE, PAS LE SLOT, et c est une lecon du chantier : le pool de slots
// reboucle et la generation ne fait que 2 bits, donc `(slot, gen)` est la seule cle. Le NUAGE
// de positions, lui, ne porte PAS de generation (`filmdec.BipedPosition`) : les vies d un meme
// slot y sont fondues. C est le RECENSEMENT qui les separe, et la fenetre de chaque vie qui
// decoupe le nuage — limite structurelle deja ecrite au rapport V1 (item 1), reprise telle
// quelle ici plutot que contournee.
//
// CE QUI EST REPRIS DE LA MESURE, SANS RIEN REGLER A NOUVEAU : la tolerance de recensement
// (~20 s, mediane d intervalle d image-cle mesuree a 20,00 s sur huit films) et le seuil de
// vitesse au-dela duquel la direction d `i1` vaut un cap (5 m/s, oracle V1a.3).

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehicleCensusTolUS est la TOLERANCE de la fenetre d une vie, de part et d autre de son
// recensement. Le recensement BORNE, il ne DATE pas : les images-cles sont espacees de ~20 s
// (mediane d intervalle mesuree 20,00 s, p90 20,00 a 20,02 s sur huit films — cf.
// `filmdec/world_object_census.go`). Une vie recensee de `t0` a `t1` a donc pu naitre jusqu a
// 20 s avant `t0` et durer jusqu a 20 s apres `t1`.
const vehicleCensusTolUS = uint64(20_000_000)

// vehicleSampleStrideFrames est le PAS d echantillonnage d une trajectoire de vehicule, EN
// FRAMES du document (une frame = `FrameIntervalMS`, 100 ms par defaut).
//
// POURQUOI LA GRILLE DU DOCUMENT, ET PAS PLUS GROSSIER. Le film replique a ~60 Hz par entite ;
// la publier telle quelle multiplierait par six le poids du calque pour un rendu que le client
// n affiche pas plus finement — c est exactement le raisonnement de `DefaultFrameIntervalMS`
// pour les trajectoires JOUEURS (`build.go`), et le calque des vehicules se pose sur le MEME
// axe. Un pas plus grossier a ete ecarte sur des chiffres : la vitesse `i1` d un vehicule monte
// a 26,1 m/s (V1a.3, maxima par film 12,2 a 26,1), soit 2,6 m parcourus en une frame et 5,2 m
// en deux — sur Behemoth, dont l emprise de jeu mesuree fait 46 m par 54 m (V2 § 1.2), deux
// frames feraient glisser le sprite de 10 % de la carte entre deux echantillons.
//
// LA REGLE EST « LE PREMIER OBSERVE GAGNE », celle de `decimateTracks` : un point par vie et par
// pas. Elle est deterministe parce que le nuage entre trie par instant.
const vehicleSampleStrideFrames = 1

// vehicleRelayRadiusM est la distance EN PLAN sous laquelle la NAISSANCE d une vie et la
// DERNIERE POSITION CONNUE d une autre designent LE MEME vehicule, pas deux.
//
// LE BUG QU IL CORRIGE, VU PAR L UTILISATEUR EN VISIONNAGE : quand un vehicule est pris, un
// DOUBLE reste quelques secondes a l ancienne place. Le film ne deplace pas l objet, il le
// RE-CREE sous un NOUVEAU SLOT : la vie garee (naissance seule, aucun echantillon, aucun
// occupant) cesse d etre recensee, et une vie du meme chassis demarre AU MEME POINT. Les deux
// sont publiees, donc deux sprites — l ancien restant visible jusqu a `t1max`, c est-a-dire
// jusqu a ~20 s (l intervalle d image-cle).
//
// LA MESURE (2026-09-02, sur les deux artefacts de demonstration) : 10 paires sur `0d76e8f1`,
// 3 sur `fccc61cd`, ecart de position 0,00 m dans 12 cas et 0,01 m dans le dernier, TEMOIN NUL
// (0 paire quand on exige des chassis DIFFERENTS aux memes criteres). Le rayon est fixe a 0,5 m
// — cinquante fois la dispersion mesuree, et tres en dessous de l espacement des emplacements
// d apparition (rayon d amas 0,00 m, 6 emplacements distincts sur Behemoth, V2 § 1).
const vehicleRelayRadiusM = 0.5

// vehicleRelayMarginFrames est la marge accordee au-dela de `T1Max` pour accueillir un relais.
// ELLE VAUT ZERO, et c est une mesure : les 13 relais observes demarrent tous DANS
// [`T1` .. `T1Max`], jamais apres. Une marge n ajouterait aucun relais reel et n elargirait que
// la surface de faux positifs. Elle est nommee pour que le jour ou un film la demande, le
// changement soit une decision datee et pas un chiffre glisse dans une condition.
const vehicleRelayMarginFrames = 0

// vehicleMinSpeedMPS est la vitesse au-dela de laquelle la direction de la velocite `i1` vaut un
// CAP. C est le seuil de l oracle V1a.3 (rapport `V1A_RAPPORT_2026-08-31.md` § 3.1), sous lequel
// la mesure a valide `i1` : ecart median au deplacement de 1,7 a 2,1 deg sur quatre films
// (R = 0,992 a 0,997), contre 51 a 88 deg pour le temoin par melange deterministe.
//
// SOUS CE SEUIL, AUCUN CAP N EST CALCULE : la direction d un vecteur quasi nul est du bruit. Le
// cap du dernier echantillon mobile est alors REPORTE — un vehicule a l arret garde le cap sous
// lequel il s est arrete, ce qui est la decision de cadrage du plan (et la seule honnete : `i2`
// est REFUTE et `i21` est ABSENT de `ti=40`, cf. V1_CONDUCTEUR_VISEE_2026-09-01).
const vehicleMinSpeedMPS = 5.0

// vehicleLife est une vie de vehicule telle que le recensement la borne, decoupee de sa voisine
// du meme slot.
type vehicleLife struct {
	key filmdec.EquipmentLifeKey
	// firstUS / lastUS : premiere et derniere image-cle qui RECENSE la vie.
	firstUS, lastUS uint64
	// goneByUS est la premiere image-cle qui ne la recense PLUS : la premiere preuve d absence.
	// Zero = aucune (la vie est encore la a la derniere image-cle du film, ou nee apres elle).
	goneByUS uint64
	// loUS / hiUS bornent la fenetre dans laquelle le nuage de positions appartient a CETTE vie.
	loUS, hiUS uint64
	census     int
}

// buildVehicleTracks assemble les vies publiables et leur couverture. PUR.
func buildVehicleTracks(
	scan VehicleScan, bipeds []filmdec.BipedPosition, own OwnerReport, clock replayClock,
) ([]VehicleTrack, VehicleCoverage) {
	cov := VehicleCoverage{Scanned: scan.Scanned, UnknownChassis: map[string]int{}}
	if !scan.Scanned || clock.step == 0 {
		return nil, cov
	}
	lives := vehicleLives(scan.Keyframes)
	cov.Lives = len(lives)
	spawns := vehicleSpawnsByLife(scan.Creations)
	bySlot := vehiclePositionsBySlot(scan.Positions)
	rides := buildVehicleRides(vehicleRideInputs{
		vehBySlot: bySlot, bipeds: bipeds, events: scan.Events,
		own: own, lives: lives, clock: clock,
	})
	out := make([]VehicleTrack, 0, len(lives))
	for _, l := range lives {
		tr, ok := vehicleTrackOf(l, spawns[l.key], bySlot[l.key.Slot], rides[l.key], clock)
		if !ok {
			cov.NoPosition++
			continue
		}
		out = append(out, tr)
	}
	sortVehicleTracks(out)
	// LES RELAIS SE FUSIONNENT AVANT LE COMPTAGE : la couverture doit decrire ce qui est PUBLIE,
	// pas ce qui a ete assemble. `Published` baisse donc exactement de `Merged`.
	out, cov.Merged = mergeVehicleRelays(out)
	tallyVehicleCoverage(out, &cov)
	return out, cov
}

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

// vehicleLives construit les vies bornees a partir du recensement, et DECOUPE les vies
// successives d un meme slot : sans ce decoupage, la fenetre de tolerance de l une mordrait sur
// l autre et le nuage de positions serait attribue deux fois.
func vehicleLives(kf filmdec.WorldObjectKeyframes) []vehicleLife {
	out := make([]vehicleLife, 0, len(kf.SeenUS))
	for key, seen := range kf.SeenUS {
		if len(seen) == 0 {
			continue
		}
		l := vehicleLife{key: key, firstUS: seen[0], lastUS: seen[len(seen)-1], census: len(seen)}
		l.goneByUS = firstTimeAfter(kf.TimesUS, l.lastUS)
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].key.Slot != out[j].key.Slot {
			return out[i].key.Slot < out[j].key.Slot
		}
		return out[i].firstUS < out[j].firstUS
	})
	assignVehicleWindows(out)
	return out
}

// assignVehicleWindows pose `loUS` / `hiUS` sur des vies DEJA triees par (slot, premier
// recensement). Deux vies consecutives d un meme slot se partagent la frontiere : la fenetre de
// l une s arrete ou celle de l autre commence.
func assignVehicleWindows(lives []vehicleLife) {
	for i := range lives {
		l := &lives[i]
		l.loUS = subUS(l.firstUS, vehicleCensusTolUS)
		l.hiUS = l.goneByUS
		if l.hiUS == 0 {
			l.hiUS = l.lastUS + vehicleCensusTolUS
		}
		if i > 0 && lives[i-1].key.Slot == l.key.Slot && lives[i-1].hiUS > l.loUS {
			l.loUS = lives[i-1].hiUS
		}
		if i+1 < len(lives) && lives[i+1].key.Slot == l.key.Slot && l.hiUS > lives[i+1].firstUS {
			l.hiUS = lives[i+1].firstUS
		}
	}
}

// vehicleSpawnsByLife retient, par vie, le record de creation le PLUS PRECOCE : c est la
// naissance. Les records suivants d une meme vie sont des re-annonces, et le mot d identite y est
// constant (gate 1 de V1.5 : 100 % de constance par vie sur les deux films mesures).
func vehicleSpawnsByLife(cre []filmdec.EquipmentCreation) map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation {
	out := map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation{}
	for _, c := range cre {
		k := filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}
		if prev, ok := out[k]; ok && prev.TimestampUS <= c.TimestampUS {
			continue
		}
		out[k] = c
	}
	return out
}

// vehiclePositionsBySlot indexe le nuage par slot et TRIE chaque liste par instant : le
// decoupage par fenetre de vie et la regle « le premier observe gagne » en dependent.
func vehiclePositionsBySlot(pos []filmdec.BipedPosition) map[uint32][]filmdec.BipedPosition {
	out := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		out[p.Slot] = append(out[p.Slot], p)
	}
	for s := range out {
		v := out[s]
		sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
	}
	return out
}

// vehicleTrackOf assemble UNE vie. Rend faux quand elle n a NI naissance lue NI echantillon :
// une vie sans la moindre position n a rien a dessiner, et lui inventer un point serait pire que
// de la taire.
func vehicleTrackOf(
	l vehicleLife, spawn filmdec.EquipmentCreation, pos []filmdec.BipedPosition,
	rides []VehicleRide, clock replayClock,
) (VehicleTrack, bool) {
	samples, lastSeenUS := vehicleSamplesOf(pos, l, clock)
	hasSpawn := spawn.TimestampUS > 0
	if !hasSpawn && len(samples) == 0 {
		return VehicleTrack{}, false
	}
	tr := VehicleTrack{Slot: l.key.Slot, Gen: l.key.Gen, End: VehicleEndUnknown}
	if hasSpawn {
		tr.Spawn = &VehicleSpawn{X: round2(spawn.X), Y: round2(spawn.Y), Z: round2(spawn.Z)}
		if spawn.MPPPresent[filmdec.MPPWord32] {
			tr.Chassis = formatChassisID(uint32(spawn.MPPVal[filmdec.MPPWord32]))
			tr.Family = vehicleFamilyOf(uint32(spawn.MPPVal[filmdec.MPPWord32]))
		}
	}
	tr.Samples = samples
	tr.T0, tr.T1, tr.T1Max = vehicleBounds(l, spawn, lastSeenUS, clock)
	tr.Rides = clampVehicleRides(rides, tr.T0, tr.T1Max)
	return tr, true
}

// clampVehicleRides ramene chaque episode d occupation dans la fenetre d affichage de la vie
// [t0, t1max], et ECARTE ceux qui lui sont entierement exterieurs.
//
// SANS ce clamp, un joueur devenait INVISIBLE : pour une vie sans record de creation, la fenetre
// de tolerance du nuage de positions commence ~20 s AVANT la premiere image-cle (`loUS`), un trou
// de position peut donc dater un episode avant `t0`. Sur ces images, le calque ne dessine pas
// encore le vehicule mais le pion de l occupant est deja supprime — ni l un ni l autre a l ecran
// (revue adversariale 2026-09-02, MAJEUR n° 3).
func clampVehicleRides(rides []VehicleRide, t0, t1max int) []VehicleRide {
	if len(rides) == 0 {
		return rides
	}
	out := make([]VehicleRide, 0, len(rides))
	for _, r := range rides {
		if r.T1 < t0 || r.T0 > t1max {
			continue
		}
		if r.T0 < t0 {
			r.T0 = t0
		}
		if r.T1 > t1max {
			r.T1 = t1max
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// vehicleBounds pose les trois bornes d affichage, MEME DOCTRINE que les armes au sol
// (`document_ground_weapon_items.go`) : `T1` est la DERNIERE PREUVE de presence, `T1Max` la
// PREMIERE PREUVE d absence. Entre les deux, le film ne dit rien — c est un intervalle de ~20 s
// que le client rend comme il l entend, mais dans du MESURE.
//
// `T0` prefere l instant du record de CREATION quand il existe (date a la milliseconde) au
// premier recensement (borne a ~20 s pres).
func vehicleBounds(
	l vehicleLife, spawn filmdec.EquipmentCreation, lastSeenUS uint64, clock replayClock,
) (t0, t1, t1max int) {
	bornUS := l.firstUS
	if spawn.TimestampUS > 0 && spawn.TimestampUS < bornUS {
		bornUS = spawn.TimestampUS
	}
	lastProofUS := l.lastUS
	if lastSeenUS > lastProofUS {
		lastProofUS = lastSeenUS
	}
	t0 = clock.frame(bornUS)
	t1 = clock.frame(lastProofUS)
	if t1 < t0 {
		t1 = t0
	}
	t1max = clock.frames - 1
	if l.goneByUS > 0 {
		t1max = clock.frame(l.goneByUS)
	}
	if t1max < t1 {
		t1max = t1
	}
	return t0, t1, t1max
}

// vehicleSamplesOf projette le nuage d un slot sur l axe de frames, DANS la fenetre de la vie.
// Rend aussi l instant du dernier echantillon retenu — la derniere preuve de presence que le
// flux de position apporte, souvent posterieure au dernier recensement.
func vehicleSamplesOf(
	pos []filmdec.BipedPosition, l vehicleLife, clock replayClock,
) ([]VehicleSample, uint64) {
	var (
		out      []VehicleSample
		lastFr   = -1
		lastSeen uint64
		heading  float32
		hasHead  bool
	)
	for _, p := range pos {
		if p.TimestampUS < l.loUS || p.TimestampUS > l.hiUS {
			continue
		}
		if h, ok := vehicleHeadingOf(p); ok {
			heading, hasHead = h, true
		}
		lastSeen = p.TimestampUS
		fr := clock.frame(p.TimestampUS)
		if lastFr >= 0 && fr-lastFr < vehicleSampleStrideFrames {
			continue
		}
		lastFr = fr
		s := VehicleSample{T: fr, X: round2(p.X), Y: round2(p.Y), Z: round2(p.Z)}
		if hasHead {
			s.H = headingForJSON(heading)
		}
		out = append(out, s)
	}
	return out, lastSeen
}

// vehicleHeadingOf rend le CAP en degres [0,360[ d un echantillon, quand sa velocite `i1` depasse
// le seuil de l oracle. Meme origine et meme sens que `atan2(Y, X)` des positions dequantifiees,
// donc la MEME convention que `Point.H` — le client n a qu une regle d orientation a connaitre.
func vehicleHeadingOf(p filmdec.BipedPosition) (float32, bool) {
	v, ok := p.VelocityVector()
	if !ok {
		return 0, false
	}
	speed := math.Hypot(float64(v[0]), float64(v[1]))
	if speed < vehicleMinSpeedMPS {
		return 0, false
	}
	deg := math.Atan2(float64(v[1]), float64(v[0])) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return float32(deg), true
}

// sortVehicleTracks fige l ordre publie : par instant d apparition, puis par vie. Un ordre stable
// est ce qui rend l artefact comparable d une cuisson a l autre (les vies sortent d une map).
func sortVehicleTracks(tracks []VehicleTrack) {
	sort.Slice(tracks, func(i, j int) bool {
		switch {
		case tracks[i].T0 != tracks[j].T0:
			return tracks[i].T0 < tracks[j].T0
		case tracks[i].Slot != tracks[j].Slot:
			return tracks[i].Slot < tracks[j].Slot
		default:
			return tracks[i].Gen < tracks[j].Gen
		}
	})
}

// firstTimeAfter rend le premier instant de `times` (TRIE) strictement posterieur a `at`, ou zero.
func firstTimeAfter(times []uint64, at uint64) uint64 {
	i := sort.Search(len(times), func(k int) bool { return times[k] > at })
	if i >= len(times) {
		return 0
	}
	return times[i]
}

// subUS soustrait sans passer sous zero (les instants sont des uint64).
func subUS(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}
