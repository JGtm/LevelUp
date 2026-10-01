package replay

// vehicle_takes.go — LA RESSOURCE « VEHICULES » DE L EMPRISE : prises par camp et par joueur, et
// temps a bord, projetes depuis le calque vehicules du document (plan
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.1, decisions D2, D4, D6, D8).
//
// PUR : aucune I/O, aucune lecture de film, aucune base. L entree est le document deja assemble
// (`vehicles[].rides[]`, `family`, `part`, `carrier`, `roster[].team`, `vehicleScenery`), et la
// sortie ne depend que de lui. Ce fichier LIT le calque, il ne le produit pas : les fichiers qui
// l ecrivent (`vehicle_rides*.go`, `build_vehicles.go`, `document_vehicles.go`) ne sont pas touches.
//
// # UNE PRISE (D2)
//
// Une prise est UNE VIE DE VEHICULE QUI PASSE A UN CAMP : dans une vie, le premier episode d un
// occupant de ce camp, tout siege confondu, une fois que la vie appartenait a l autre camp (ou a
// personne). Un retour au meme camp apres un passage adverse est une NOUVELLE prise ; un joueur
// qui sort puis remonte pendant que l adversaire n est pas monte n en fait pas une seconde. Le
// joueur credite est l occupant de cet episode — le conducteur (siege 0) d abord (`vehicleSeatRank`) quand deux
// episodes commencent a la meme image, puis les sieges connus, puis les sieges inconnus (un
// artilleur de tourelle reportee sur son porteur n a pas de siege). Les episodes `film` et
// `proximity` comptent tous : le calque a deja ecarte les replis contredits. Leur part est
// publiee (`ProximityEpisodes`), jamais cachee.
//
// # LE TEMPS A BORD (D4)
//
// La somme des episodes, (t1 − t0) × intervalle d image, par (camp, joueur, famille). C est la
// barre « exposition » de l Emprise, et le denominateur du rendement par minute a bord.
//
// # LE PERIMETRE (D6)
//
// Toute vie que le calque publie, tourelles fixes comprises. TROIS REGLES, chacune tenue par un
// test :
//   - LE DECOR : une vie que le verdict publie (`doc.VehicleScenery.Hidden`) est ecartee et
//     comptee. La regle de decor elle-meme (`service.decideVehicleScenery`, cinq conditions de
//     pose puis hors de la zone jouable) se decide A LA REQUETE, avec la zone jouable de la
//     carte, que l artefact ne connait pas : elle n est ni recopiee ici, ni appelable d ici. Sa
//     condition « aucun occupant » garantit que le decor ne porte JAMAIS d episode, donc jamais
//     de prise ; le verdict, quand il est pose, est honore en plus.
//   - LA PIECE MONTEE (`part = turret`) APPARTIENT A SON PORTEUR : ses episodes restes sur elle
//     (la couche de tourelles ne reporte pas tous les episodes, cf. `moveTurretRides`) sont
//     rattaches a la vie du porteur, sauf si le MEME occupant y est deja a bord pendant le meme
//     intervalle (compte, pas ajoute : le temps ne se double pas). Une piece sans porteur trouve
//     reste sa propre vie (une tourelle fixe), comptee comme orpheline.
//   - LA FAMILLE INCONNUE est NOMMEE (`VehicleFamilyUnknown`), jamais rangee dans une famille
//     voisine ni ecartee.
//
// # LE CAMP
//
// Celui que le FILM ecrit pour l occupant (`roster[].team`, 0 a 8). Un occupant sans xuid (un
// bot, ou un slot que le pont n a pas nomme), sans ligne de roster, ou dans un mode « aucune
// equipe » (-1) n a pas de camp : son episode est compte a part (`EpisodesNoXUID`,
// `EpisodesNoCamp`) et ne fait ni prise ni temps.
//
// # D8 : « NON MESURE », JAMAIS ZERO
//
// Un artefact sans occupation lue (schema < 67, ou calque vehicules non balaye) rend
// `Measured = false` et aucune ligne : l appelant affiche « non mesure », pas « 0 prise ».

import (
	"cmp"
	"slices"
)

// VehicleFamilyUnknown nomme la famille d une vie dont le chassis n est pas dans la table
// (`VehicleTrack.Family` vide). Identifiant STABLE ; le libelle « Vehicule inconnu » se resout
// cote affichage.
const VehicleFamilyUnknown = "unknown"

// vehicleTakesMinSchema : le premier schema dont les episodes portent leur provenance
// `film` / `proximity` et le siege lu (cf. `VehicleRideSrcFilm`). En deca, l occupation n est pas
// lue comme ce calcul la consomme.
const vehicleTakesMinSchema = 67

// vehicleTakesSrcProximity : valeur publiee de `VehicleRideSrcProximity`, liee par le test
// `TestVehicleTakes_ConstantesAlignees`.
const vehicleTakesSrcProximity = "proximity"

// Raisons d un calcul « non mesure » (`VehicleTakesReport.Reason`). Identifiants STABLES.
const (
	VehicleTakesUnmeasuredSchema     = "schema_before_67"
	VehicleTakesUnmeasuredNotScanned = "vehicles_not_scanned"
	VehicleTakesUnmeasuredNoInterval = "frame_interval_missing"
)

// VehicleUsageRow : les prises et le temps a bord d UN occupant sur UNE famille, dans UN camp.
type VehicleUsageRow struct {
	// Camp est le designateur d equipe du film (0 a 8).
	Camp int `json:"camp"`
	// XUID est l occupant, en decimal. Jamais vide : un occupant sans xuid n a pas de ligne.
	XUID string `json:"xuid"`
	// Family est la famille de chassis, ou `VehicleFamilyUnknown`.
	Family string `json:"family"`
	// Takes : prises credites a cet occupant sur cette famille (D2).
	Takes int `json:"takes"`
	// AboardMS : temps a bord, en ms (D4).
	AboardMS int64 `json:"aboardMs"`
	// Episodes et ProximityEpisodes : les episodes comptes dans cette ligne, dont ceux de repli.
	Episodes          int `json:"episodes"`
	ProximityEpisodes int `json:"proximityEpisodes"`
	// Frags : frags de classe engin de cet occupant tombes PENDANT un de ses episodes de cette famille
	// (D9, `PairVehicleFrags`). Zero tant que l appariement n a pas ete joue.
	Frags int `json:"frags"`
}

// VehicleTakesCoverage dit ce que le calcul a lu, rattache et ecarte. Ce sont les compteurs qui
// distinguent un « zero » d un « non lu ».
type VehicleTakesCoverage struct {
	// Lives : vies du calque lues ; LivesWithRides : celles qui portent au moins un episode compte.
	Lives          int `json:"lives"`
	LivesWithRides int `json:"livesWithRides"`
	// Episodes : episodes lus (avant tout ecart) ; ProximityEpisodes : ceux de repli ;
	// ProximityMS : leur duree.
	Episodes          int   `json:"episodes"`
	ProximityEpisodes int   `json:"proximityEpisodes"`
	ProximityMS       int64 `json:"proximityMs"`
	// EpisodesNoXUID : occupant non nomme ; EpisodesNoCamp : occupant nomme mais sans camp.
	EpisodesNoXUID int `json:"episodesNoXuid"`
	EpisodesNoCamp int `json:"episodesNoCamp"`
	// HiddenLives : vies de decor ecartees sur le verdict publie.
	HiddenLives int `json:"hiddenLives"`
	// PartRides : episodes d une piece montee rattaches a son porteur ; PartRidesDuplicate :
	// ceux ecartes parce que le meme occupant etait deja a bord du porteur ; OrphanParts : pieces
	// dont le porteur n est pas dans le document (gardees comme vies propres).
	PartRides          int `json:"partRides"`
	PartRidesDuplicate int `json:"partRidesDuplicate"`
	OrphanParts        int `json:"orphanParts"`
}

// VehicleTakesReport est la projection d un document.
type VehicleTakesReport struct {
	// Measured : faux quand l artefact n a pas d occupation lue (D8). Aucune ligne alors.
	Measured bool `json:"measured"`
	// Reason nomme pourquoi `Measured` est faux ; vide sinon.
	Reason string `json:"reason,omitempty"`
	// Rows : triees par (camp, xuid, famille).
	Rows []VehicleUsageRow `json:"rows,omitempty"`
	// Rides : les episodes COMPTES (ceux qui ont un camp et un xuid), dans l ordre de montee de
	// chaque vie. C est la matiere de l appariement des frags (`PairVehicleFrags`).
	Rides    []VehicleAccountedRide `json:"rides,omitempty"`
	Coverage VehicleTakesCoverage   `json:"coverage"`
}

// vehicleLifeKey est la cle d une vie de vehicule.
type vehicleLifeKey struct{ slot, gen uint32 }

// vehicleLifeRide : un episode rattache a la vie qui le porte, avec la famille de cette vie.
type vehicleLifeRide struct {
	ride   VehicleRide
	family string
}

// ProjectVehicleTakes projette les prises et le temps a bord d un document (cf. l en-tete).
// `doc` nil rend un rapport non mesure.
func ProjectVehicleTakes(doc *ReplayDocument) VehicleTakesReport {
	if reason := vehicleTakesUnmeasured(doc); reason != "" {
		return VehicleTakesReport{Reason: reason}
	}
	rep := VehicleTakesReport{Measured: true}
	rides := vehicleRidesByLife(doc, &rep.Coverage)
	camps := vehicleCampsByXUID(doc)
	acc := newVehicleAccum()
	step := int64(doc.FrameIntervalMS)
	for _, key := range sortedLifeKeys(rides) {
		if vehicleAccountLife(rides[key], camps, step, acc, &rep.Coverage) {
			rep.Coverage.LivesWithRides++
		}
	}
	rep.Rows = vehicleUsageRows(acc.rows)
	rep.Rides = acc.rides
	return rep
}

// vehicleTakesUnmeasured rend la raison pour laquelle le document n a pas d occupation lue (D8),
// ou vide quand il en a une.
func vehicleTakesUnmeasured(doc *ReplayDocument) string {
	switch {
	case doc == nil || doc.SchemaVersion < vehicleTakesMinSchema:
		return VehicleTakesUnmeasuredSchema
	case doc.Coverage == nil || doc.Coverage.Vehicles == nil || !doc.Coverage.Vehicles.Scanned:
		return VehicleTakesUnmeasuredNotScanned
	case doc.FrameIntervalMS <= 0:
		return VehicleTakesUnmeasuredNoInterval
	}
	return ""
}

// vehicleRidesByLife range les episodes par vie PORTEUSE : la piece montee rend les siens a son
// porteur. Decor ecarte, doublons de piece comptes.
func vehicleRidesByLife(doc *ReplayDocument, cov *VehicleTakesCoverage) map[vehicleLifeKey][]vehicleLifeRide {
	hidden := map[vehicleLifeKey]bool{}
	if doc.VehicleScenery != nil {
		for _, h := range doc.VehicleScenery.Hidden {
			hidden[vehicleLifeKey{h.Slot, h.Gen}] = true
		}
	}
	byKey := map[vehicleLifeKey]*VehicleTrack{}
	for i := range doc.Vehicles {
		byKey[vehicleLifeKey{doc.Vehicles[i].Slot, doc.Vehicles[i].Gen}] = &doc.Vehicles[i]
	}
	out := map[vehicleLifeKey][]vehicleLifeRide{}
	// Les vies qui portent d abord, les pieces ensuite : le test de doublon d une piece lit les
	// episodes DEJA poses sur son porteur, quel que soit l ordre du document.
	for _, pass := range []bool{false, true} {
		for i := range doc.Vehicles {
			v := &doc.Vehicles[i]
			if (v.Part == VehiclePartTurret) != pass {
				continue
			}
			cov.Lives++
			if hidden[vehicleLifeKey{v.Slot, v.Gen}] {
				cov.HiddenLives++
				continue
			}
			vehicleAddLifeRides(out, byKey, v, cov)
		}
	}
	return out
}

// vehicleAddLifeRides pose les episodes d une vie sur sa vie porteuse.
func vehicleAddLifeRides(out map[vehicleLifeKey][]vehicleLifeRide, byKey map[vehicleLifeKey]*VehicleTrack,
	v *VehicleTrack, cov *VehicleTakesCoverage) {
	owner, family := vehicleLifeKey{v.Slot, v.Gen}, vehicleFamilyName(v.Family)
	isPart := v.Part == VehiclePartTurret
	if isPart {
		if c, ok := vehicleCarrierOf(v, byKey); ok {
			owner, family = vehicleLifeKey{c.Slot, c.Gen}, vehicleFamilyName(c.Family)
		} else {
			cov.OrphanParts++
			isPart = false // une tourelle sans porteur reste sa propre vie
		}
	}
	for _, r := range v.Rides {
		cov.Episodes++
		if isPart && vehicleRideAlreadyAboard(out[owner], r) {
			cov.PartRidesDuplicate++
			continue
		}
		if isPart {
			cov.PartRides++
		}
		out[owner] = append(out[owner], vehicleLifeRide{ride: r, family: family})
	}
}

// vehicleCarrierOf rend le porteur d une piece montee quand le document le contient.
func vehicleCarrierOf(v *VehicleTrack, byKey map[vehicleLifeKey]*VehicleTrack) (*VehicleTrack, bool) {
	if v.Carrier == nil {
		return nil, false
	}
	c, ok := byKey[vehicleLifeKey{v.Carrier.Slot, v.Carrier.Gen}]
	return c, ok
}

// vehicleRideAlreadyAboard dit si le meme occupant est deja a bord du porteur pendant une part de
// l intervalle de `r` (l episode de la piece n ajoute alors rien).
func vehicleRideAlreadyAboard(have []vehicleLifeRide, r VehicleRide) bool {
	for _, h := range have {
		if h.ride.XUID == r.XUID && h.ride.T0 <= r.T1 && r.T0 <= h.ride.T1 {
			return true
		}
	}
	return false
}

// vehicleFamilyName rend la famille publiee, ou la famille inconnue nommee.
func vehicleFamilyName(family string) string {
	if family == "" {
		return VehicleFamilyUnknown
	}
	return family
}

// vehicleCampsByXUID indexe le camp que le film ecrit, par joueur. Une ligne sans xuid (un bot)
// ou sans camp (`nil`, `-1`) n a pas de camp.
func vehicleCampsByXUID(doc *ReplayDocument) map[string]int {
	out := map[string]int{}
	for _, r := range doc.Roster {
		if r.XUID != "" && r.Team != nil && *r.Team >= 0 {
			out[r.XUID] = *r.Team
		}
	}
	return out
}

// vehicleUsageKey : la cle d une ligne de sortie.
type vehicleUsageKey struct {
	camp   int
	xuid   string
	family string
}

// vehicleAccountLife compte une vie : le temps de tous ses episodes, les prises dans l ordre de
// montee. Rend vrai quand au moins un episode a ete compte.
func vehicleAccountLife(rides []vehicleLifeRide, camps map[string]int, stepMS int64,
	acc *vehicleAccum, cov *VehicleTakesCoverage) bool {
	// TRI STABLE, CLE RESIDUELLE = LE RANG D ENTREE (DT-9, cliquet `archlint/film_tri_total_test.go`,
	// converti a la fusion J11.6 du 2026-10-01 sans changer l ordre) : `vehicleRidesByLife` pose les
	// episodes dans l ordre du document (vies porteuses puis pieces, episodes dans l ordre publie).
	slices.SortStableFunc(rides, func(x, y vehicleLifeRide) int {
		a, b := x.ride, y.ride
		return cmp.Or(cmp.Compare(a.T0, b.T0),
			cmp.Compare(vehicleSeatRank(a.Seat), vehicleSeatRank(b.Seat)),
			cmp.Compare(a.XUID, b.XUID))
	})
	counted, current := false, -1 // aucun camp ne vaut -1 (ecarte par vehicleCampsByXUID)
	for _, e := range rides {
		r := e.ride
		duration := int64(r.T1-r.T0) * stepMS
		prox := r.Src == vehicleTakesSrcProximity
		if prox {
			cov.ProximityEpisodes++
			cov.ProximityMS += duration
		}
		camp, ok := vehicleRideCamp(r, camps, cov)
		if !ok {
			continue
		}
		row := vehicleUsageRowOf(acc.rows, vehicleUsageKey{camp, r.XUID, e.family})
		acc.rides = append(acc.rides, VehicleAccountedRide{
			Camp: camp, XUID: r.XUID, Family: e.family, T0: r.T0, T1: r.T1, Proximity: prox})
		row.AboardMS += duration
		row.Episodes++
		if prox {
			row.ProximityEpisodes++
		}
		if camp != current {
			row.Takes++
			current = camp
		}
		counted = true
	}
	return counted
}

// vehicleRideCamp rend le camp de l occupant d un episode, ou compte pourquoi il n en a pas.
func vehicleRideCamp(r VehicleRide, camps map[string]int, cov *VehicleTakesCoverage) (int, bool) {
	if r.XUID == "" {
		cov.EpisodesNoXUID++
		return 0, false
	}
	camp, ok := camps[r.XUID]
	if !ok {
		cov.EpisodesNoCamp++
	}
	return camp, ok
}

func vehicleUsageRowOf(acc map[vehicleUsageKey]*VehicleUsageRow, k vehicleUsageKey) *VehicleUsageRow {
	if row, ok := acc[k]; ok {
		return row
	}
	row := &VehicleUsageRow{Camp: k.camp, XUID: k.xuid, Family: k.family}
	acc[k] = row
	return row
}

// vehicleUsageRows rend les lignes triees par (camp, xuid, famille).
func vehicleUsageRows(acc map[vehicleUsageKey]*VehicleUsageRow) []VehicleUsageRow {
	out := make([]VehicleUsageRow, 0, len(acc))
	for _, r := range acc {
		out = append(out, *r)
	}
	// (camp, xuid, famille) est la cle de la carte : unique, le comparateur est total.
	slices.SortFunc(out, func(a, b VehicleUsageRow) int {
		return cmp.Or(cmp.Compare(a.Camp, b.Camp), cmp.Compare(a.XUID, b.XUID), cmp.Compare(a.Family, b.Family))
	})
	return out
}

// sortedLifeKeys rend les vies dans un ordre stable : la carte n en a pas, et une sortie qui
// change d ordre fait clignoter les comparaisons.
func sortedLifeKeys(m map[vehicleLifeKey][]vehicleLifeRide) []vehicleLifeKey {
	keys := make([]vehicleLifeKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// (slot, generation) est la cle de la carte : unique, le comparateur est total.
	slices.SortFunc(keys, func(a, b vehicleLifeKey) int {
		return cmp.Or(cmp.Compare(a.slot, b.slot), cmp.Compare(a.gen, b.gen))
	})
	return keys
}

// VehicleAccountedRide : un episode COMPTE dans une ligne de sortie, avec son intervalle en
// frames du document (bornes incluses). `Family` est celle de la vie porteuse (une piece montee a
// celle de son porteur).
type VehicleAccountedRide struct {
	Camp      int
	XUID      string
	Family    string
	T0, T1    int
	Proximity bool
}

// vehicleAccum : ce que le calcul cumule — les lignes, et les episodes qui les ont nourries.
type vehicleAccum struct {
	rows  map[vehicleUsageKey]*VehicleUsageRow
	rides []VehicleAccountedRide
}

func newVehicleAccum() *vehicleAccum {
	return &vehicleAccum{rows: map[vehicleUsageKey]*VehicleUsageRow{}}
}
