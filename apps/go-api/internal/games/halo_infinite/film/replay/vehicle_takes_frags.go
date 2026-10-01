package replay

// vehicle_takes_frags.go — L'APPARIEMENT DES FRAGS DE CLASSE ENGIN AUX EPISODES PUBLIES (plan
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, decision D9).
//
// # POURQUOI
//
// Sur tout le lobby, 62 % seulement des frags de classe engin tombent dans un episode publie
// (81 / 131 au releve L7.0) : le calque n'attribue qu'une part des vies de vehicule. Diviser TOUS
// les frags par le seul temps a bord publie gonflerait le rendement. Le rendement (frags par minute
// a bord) ne compte donc au numerateur que les frags tombes PENDANT un episode publie de leur
// tueur — meme population que le denominateur. Les autres sont comptes, pas caches.
//
// # LA JONCTION D'HORLOGE
//
// `frameOfFilmMS` et `frameInWindow` (equipment_episode_kills.go) : LES MEMES que la jointure des
// episodes d'equipement, jamais recopiees. Le frag est date sur l'horloge « debut du film »
// (`killsource.Kill.TimeMS`, persistee telle quelle dans `match_kill_events.time_ms`), l'episode
// sur l'axe de frames du document ; `OriginMs` les accorde. Sans origine, rien n'est apparie :
// `Read` est faux, et c'est dit (jamais un zero).
//
// PUR : l'appelant a deja resolu les frags (classe, tueur) ; ce fichier ne lit ni film ni base.

// Raisons d un appariement non lu (`VehicleFragsCoverage.Reason`). Identifiants STABLES.
const (
	VehicleFragsNoOrigin = "origin_missing"
	VehicleFragsNoSource = "no_kill_source"
)

// VehicleFragRef : un frag de classe engin, deja resolu en identite (xuid decimal du tueur) et
// date sur l'horloge « debut du film ».
type VehicleFragRef struct {
	XUID   string
	TimeMS int
}

// VehicleFragsCoverage dit ce que l'appariement a lu. Read faux = non mesure (pas de zero).
type VehicleFragsCoverage struct {
	Read      bool   `json:"read"`
	Reason    string `json:"reason,omitempty"`
	Total     int    `json:"total"`
	Matched   int    `json:"matched"`
	Unmatched int    `json:"unmatched"`
}

// PairVehicleFrags pose `Frags` sur les lignes du rapport et rend la couverture. Un frag est
// apparie a UN episode compte de son tueur dont la fenetre [T0, T1] le couvre (bornes incluses) ;
// si plusieurs le couvrent (deux familles qui se chevauchent), le premier dans l'ordre de montee
// l'emporte — un frag ne compte jamais deux fois.
//
// Un rapport non mesure, un document sans origine ou sans pas d'image : Read faux, rien n'est
// modifie. `kills` vide avec `killsRead` vrai est un ZERO mesure (aucun frag d'engin) ; l'appelant
// distingue « aucun evenement de mort lu » et passe alors `killsRead` faux.
func PairVehicleFrags(doc *ReplayDocument, rep *VehicleTakesReport, kills []VehicleFragRef, killsRead bool) VehicleFragsCoverage {
	if !killsRead {
		return VehicleFragsCoverage{Reason: VehicleFragsNoSource}
	}
	if doc == nil || rep == nil || !rep.Measured || doc.OriginMs == nil || doc.FrameIntervalMS <= 0 {
		return VehicleFragsCoverage{Reason: VehicleFragsNoOrigin}
	}
	cov := VehicleFragsCoverage{Read: true, Total: len(kills)}
	rowOf := map[vehicleUsageKey]int{}
	for i, r := range rep.Rows {
		rowOf[vehicleUsageKey{r.Camp, r.XUID, r.Family}] = i
	}
	byXUID := map[string][]int{}
	for i, r := range rep.Rides {
		byXUID[r.XUID] = append(byXUID[r.XUID], i)
	}
	for _, k := range kills {
		frame := frameOfFilmMS(k.TimeMS, *doc.OriginMs, doc.FrameIntervalMS)
		hit := -1
		for _, i := range byXUID[k.XUID] {
			if frameInWindow(frame, rep.Rides[i].T0, rep.Rides[i].T1) {
				hit = i
				break
			}
		}
		if hit < 0 {
			cov.Unmatched++
			continue
		}
		r := rep.Rides[hit]
		rep.Rows[rowOf[vehicleUsageKey{r.Camp, r.XUID, r.Family}]].Frags++
		cov.Matched++
	}
	return cov
}
