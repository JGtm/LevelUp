package replay

// vehicle_tracks.go — L ASSEMBLAGE PUR du calque des vehicules : de ce que le film rend
// (recensement, creations, nuage de positions) a la VIE publiee.
//
// L OBJET PUBLIE EST LA VIE, PAS LE SLOT, et c est une lecon du chantier : le pool de slots
// reboucle et la generation ne fait que 2 bits, donc `(slot, gen)` est la seule cle. Le NUAGE
// de positions, lui, ne porte PAS de generation (`grammar.BipedPosition`) : les vies d un meme
// slot y sont fondues. C est le RECENSEMENT qui les separe, et la fenetre de chaque vie qui
// decoupe le nuage — limite structurelle deja ecrite au rapport V1 (item 1), reprise telle
// quelle ici plutot que contournee.
//
// CE QUI EST REPRIS DE LA MESURE, SANS RIEN REGLER A NOUVEAU : la tolerance de recensement
// (~20 s, mediane d intervalle d image-cle mesuree a 20,00 s sur huit films) et le seuil de
// vitesse au-dela duquel la direction d `i1` vaut un cap (5 m/s, oracle V1a.3).
//
// LA FUSION DES VIES EN RELAIS vit dans vehicle_relays.go — un DEPLACEMENT du 2026-09-05, sans
// une ligne de logique changee, pour repasser sous le seuil de 500 lignes. Elle ne lit rien du
// film : elle travaille sur les vies deja assemblees ici.
//
// LA FIN DE VIE, ELLE, VIT DANS vehicle_end.go (lot 1.9.10) : le recensement BORNE la vie, le
// composant `object-dead-state` la DATE. Ce fichier ne fait que poser la fenetre et appeler.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
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
	key types.EquipmentLifeKey
	// firstUS / lastUS : premiere et derniere image-cle qui RECENSE la vie.
	firstUS, lastUS uint64
	// goneByUS est la premiere image-cle qui ne la recense PLUS : la premiere preuve d absence.
	// Zero = aucune (la vie est encore la a la derniere image-cle du film, ou nee apres elle).
	goneByUS uint64
	// loUS / hiUS bornent la fenetre dans laquelle le nuage de positions appartient a CETTE vie.
	loUS, hiUS uint64
	census     int
	// deathUS est l instant de la MORT QUE LE FILM ECRIT pour cette vie (composant
	// `object-dead-state` de `ti=40`), zero quand il n en ecrit pas. C est lui, et lui seul, qui
	// DATE la fin — cf. vehicle_end.go. `deathTailDesync` retient la QUALITE du record qui l a
	// rendu (rupture apres le dead-state), pour que la couverture ne melange pas les deux.
	deathUS         uint64
	deathTailDesync bool
}

// buildVehicleTracks assemble les vies publiables, leur couverture et le bilan de rattachement.
func buildVehicleTracks(
	scan VehicleScan, bipeds []grammar.BipedPosition, reg IdentityRegistry, clock replayClock,
) ([]VehicleTrack, VehicleCoverage, vehicleRideStats) {
	// `AimReads` compte ce que le FILM a rendu, pas ce que les episodes en retiennent : c est lui
	// qui distingue « aucun occupant ne visait » de « le decodeur n a rien lu ».
	cov := VehicleCoverage{Scanned: scan.Scanned, UnknownChassis: map[string]int{}, AimReads: len(scan.Aims)}
	if !scan.Scanned || clock.step == 0 {
		return nil, cov, vehicleRideStats{}
	}
	// REPLI NOMME ET COMPTE (D14) : le cadre de la marche n a pas ete confirme par le balayage
	// (profil plat), la lecture des morts a donc tourne sur la largeur par defaut. Le compte
	// voyage avec l artefact — il dit que ce calque repose sur un cadre non confirme.
	if scan.Scanned && scan.DeathStats.CadreParDefaut {
		clock.fb.Declenche(fallback.NomCadreDeMarcheParDefautConserve)
	}
	lives, deathTally := vehicleLives(scan.Keyframes, scan.Deaths)
	cov.Lives = len(lives)
	cov.DeathsRead, cov.DeathsMatched = deathTally.read, deathTally.matched
	cov.DeathsUnmatched, cov.DeathsTailDesync = deathTally.unmatched, deathTally.tailDesync
	spawns := vehicleSpawnsByLife(scan.Creations)
	bySlot := vehiclePositionsBySlot(scan.Positions)
	rides, st := buildVehicleRides(vehicleRideInputs{
		vehBySlot: bySlot, bipeds: bipeds, events: scan.Events, reg: reg, lives: lives,
		aimBySlot: vehicleAimBySlot(scan.Aims),
		drawable:  vehicleDrawableLives(lives, spawns, bySlot), clock: clock,
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
	tallyVehicleCoverage(out, &cov, clock.fb)
	tallyVehicleEnds(out, &cov)
	return out, cov, st
}

// vehicleLives construit les vies bornees a partir du recensement, DECOUPE les vies successives
// d un meme slot — sans ce decoupage, la fenetre de tolerance de l une mordrait sur l autre et
// le nuage de positions serait attribue deux fois — puis pose sur chacune la MORT QUE LE FILM
// ECRIT (cf. vehicle_end.go). L ordre est celui de D14 (b) : les fenetres d abord, la lecture
// ensuite, parce que c est la fenetre qui departage deux vies de meme `(slot, gen)`.
func vehicleLives(
	kf grammar.WorldObjectKeyframes, deaths []types.ObjectDeath,
) ([]vehicleLife, vehicleDeathTally) {
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
	return out, assignVehicleDeaths(out, deaths)
}

// assignVehicleWindows pose `loUS` / `hiUS` sur des vies DEJA triees par (slot, premier
// recensement). Deux vies consecutives d un meme slot se partagent la frontiere : la fenetre de
// l une s arrete ou celle de l autre commence.
//
// UNE VIE QUE LE RECENSEMENT NE FERME JAMAIS N A PAS DE BORNE HAUTE, et c est le correctif du
// lot 1.9.10. `goneByUS == 0` veut dire EXACTEMENT une chose : la derniere image-cle du film
// recense encore cette vie — elle finit donc AVEC le film. L ancienne borne « dernier
// recensement + 20 s » etait un repli qui COUPAIT le nuage de positions d un vehicule vivant,
// jusqu a 20 s avant la fin du film alors que rien ne le justifiait ; le seul decoupage
// legitime est celui de la vie SUIVANTE du meme slot, applique juste en dessous.
func assignVehicleWindows(lives []vehicleLife) {
	for i := range lives {
		l := &lives[i]
		l.loUS = subUS(l.firstUS, vehicleCensusTolUS)
		l.hiUS = l.goneByUS
		if l.hiUS == 0 {
			l.hiUS = ^uint64(0)
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
func vehicleSpawnsByLife(cre []types.EquipmentCreation) map[types.EquipmentLifeKey]types.EquipmentCreation {
	out := map[types.EquipmentLifeKey]types.EquipmentCreation{}
	for _, c := range cre {
		k := types.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}
		if prev, ok := out[k]; ok && prev.TimestampUS <= c.TimestampUS {
			continue
		}
		out[k] = c
	}
	return out
}

// vehiclePositionsBySlot indexe le nuage par slot et TRIE chaque liste par instant : le
// decoupage par fenetre de vie et la regle « le premier observe gagne » en dependent.
func vehiclePositionsBySlot(pos []grammar.BipedPosition) map[uint32][]grammar.BipedPosition {
	out := map[uint32][]grammar.BipedPosition{}
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
	l vehicleLife, spawn types.EquipmentCreation, pos []grammar.BipedPosition,
	rides []VehicleRide, clock replayClock,
) (VehicleTrack, bool) {
	samples, lastSeenUS := vehicleSamplesOf(pos, l, clock)
	hasSpawn := spawn.TimestampUS > 0
	if !hasSpawn && len(samples) == 0 {
		return VehicleTrack{}, false
	}
	tr := VehicleTrack{Slot: l.key.Slot, Gen: l.key.Gen}
	tr.End, tr.TEnd = vehicleEndOf(l, clock)
	if hasSpawn {
		tr.Spawn = &VehicleSpawn{X: round2(spawn.X), Y: round2(spawn.Y), Z: round2(spawn.Z)}
		if spawn.MPPPresent[grammar.MPPWord32] {
			tr.Chassis = formatChassisID(uint32(spawn.MPPVal[grammar.MPPWord32]))
			tr.Family = vehicleFamilyOf(uint32(spawn.MPPVal[grammar.MPPWord32]))
		}
	}
	tr.Samples = samples
	tr.T0, tr.T1, tr.T1Max = vehicleBounds(l, spawn, lastSeenUS, clock)
	if !vehicleFamilyIsRideable(tr.Family) {
		// AUCUN EPISODE SUR UN VEHICULE NON PILOTABLE. La vie reste publiee (sa trajectoire est
		// vraie) mais elle ne porte pas d occupant : un episode accroche a un decor escamoterait
		// le pion d un joueur reel passe a proximite. Vu par l utilisateur en visionnage
		// (2026-09-02) sur `fccc61cd`, ou un prop de la famille `falcon` — quasi immobile, vivant
		// tout le match — s etait vu attribuer un trajet. Le calque web filtre deja ces familles
		// a l affichage ; la garde est ici AUSSI pour que le document ne l affirme pas.
		tr.Rides = nil
		return tr, true
	}
	tr.Rides = clampVehicleRides(rides, tr.T0, tr.T1Max)
	return tr, true
}

// vehicleFamillesNonPilotables sont les familles que le multijoueur Halo Infinite ne laisse pas
// conduire : transports scriptes et decor. DECISION UTILISATEUR (2026-09-02) : « les pelicans ne
// sont pas jouables en multiplayer a ce jour (sauf parties custom locales, mais on ne les gere
// pas dans l app, par decision) ».
//
// ELLES RESTENT PUBLIEES : leur vie est vraie, et le client choisit de ne pas les dessiner. Ce
// qui est interdit ici, c est de leur attribuer un OCCUPANT — une affirmation, elle, qui serait
// fausse. La meme liste vit cote web (`vehiclesLayer.FAMILLES_NON_JOUABLES`) ; les deux se
// justifient : le document refuse de l affirmer, le calque refuse de le dessiner.
var vehicleFamillesNonPilotables = map[string]bool{
	familleFalcon:  true,
	famillePelican: true,
	famillePhantom: true,
	familleSkiff:   true,
	// LA TOURELLE AUTOMATIQUE BANNIE (lot 1.9.9, decision utilisateur du 2026-09-14) : un
	// ELEMENT DE CARTE, immobile, que personne ne conduit. Elle est ici pour la meme raison que
	// les transports ci-dessus — refuser l OCCUPANT, qui serait une affirmation fausse — mais
	// PAS pour la meme consequence cote client : le decor n est pas dessine, elle SI (pictogramme
	// dedie, cf. `replay_labels.toml` et `vehiclesLayer.VEHICLE_MAP_ELEMENT_RENDER`).
	familleTourelleAutoBannie: true,
}

// vehicleFamilyIsRideable dit si une famille peut porter un episode d occupation.
//
// SEULES LES FAMILLES EXPLICITEMENT NON PILOTABLES le refusent. Une famille VIDE — un chassis que
// `vehicleFamilyByChassis` ne nomme pas ENCORE — n en fait PAS partie : c est une IGNORANCE, pas
// une propriete du vehicule, et la confondre avec « non pilotable » supprime l occupant d un
// vehicule parfaitement reel.
//
// CORRIGE LE 2026-09-19, ET LA MESURE EST LA RAISON. La regle disait « une famille vide ne le peut
// pas non plus : on ne sait pas ce que c est, donc on n affirme rien de qui serait a bord ». Elle
// etait tenable tant que les chassis inconnus etaient rares. Le cablage de l etat par defaut de
// `ti=40` (lot 5.1.7-b) rend 18 naissances de plus sur `11de8353` — `withSpawn` 37 -> 55 — et
// SEIZE de ces chassis sont inconnus de la table : `familyUnknown` passe de 5 a 21. Autant de vies
// qui, sous l ancienne regle, ne pouvaient plus porter aucun occupant.
//
// CE QUE CELA COUTAIT, NOMME : sur `11de8353`, l episode de l occupant `585` dans le Warthog
// `773/1` (frames 1 987..2 043, xuid 2533274898781893, siege 0) disparaissait, et avec lui les
// SEPT tirs qu il portait (frames 2 005 a 2 039, `shots.v = 773`) — `shots/n` 1 690 -> 1 683 et
// `coverage.shots.noSlot` 278 -> 285 au corpus gate. La vie `773/1` etait pourtant publiee a
// l identique des deux cotes, famille `warthog` comprise : c est le rattachement geometrique qui,
// avec 18 vehicules de plus comme candidats, elisait un voisin au chassis inconnu — lequel jetait
// ensuite l episode.
//
// L IGNORANCE SE DIT, ELLE NE SE PROPAGE PAS. C est deja la regle du chassis a cote de la famille
// (« un chassis absent de la table garde son hexadecimal a l ecran, et n emprunte pas le sprite
// d un voisin ») : le client dessine un marqueur neutre et le document reste vrai.
func vehicleFamilyIsRideable(family string) bool {
	return !vehicleFamillesNonPilotables[family]
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
		// LA SERIE DE VISEE SUIT SON EPISODE (cf. clampVehicleRideAim).
		r.Aim = clampVehicleRideAim(r.Aim, r.T0, r.T1)
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
	l vehicleLife, spawn types.EquipmentCreation, lastSeenUS uint64, clock replayClock,
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
	// LA MORT ECRITE EST UNE PREUVE D ABSENCE PLUS SERREE que l image-cle qui ne recense plus :
	// elle est datee a la milliseconde la ou le recensement borne a ~20 s. Elle ne remplace pas
	// `T1` (la derniere preuve de PRESENCE) et ne coupe pas la trajectoire : un echantillon
	// posterieur reste publie, et la contradiction est comptee (`SamplesAfterEnd`).
	if l.deathUS > 0 {
		if d := clock.frame(l.deathUS); d < t1max {
			t1max = d
		}
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
	pos []grammar.BipedPosition, l vehicleLife, clock replayClock,
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
func vehicleHeadingOf(p grammar.BipedPosition) (float32, bool) {
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
