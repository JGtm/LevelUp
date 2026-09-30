package squademprise

// vehicles.go — LA RESSOURCE « VEHICULES » : lecture (types), index, décompte d'un match, somme
// d'une soirée, production et couverture (plan `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot
// L7.3 ; type publié : domain/squad_emprise_vehicles.go, décisions D2 à D10).
//
// Les prises, le temps à bord et les frags appariés sont ÉCRITS par la dérivation de l'artefact
// (`match_vehicle_takes_latest`) ; ce fichier ne recalcule aucune règle de la projection (prise,
// décor, pièce montée : L7.1). Il fait ce que la dérivation ne peut pas faire : le camp de NOTRE
// côté, les joueurs de l'escouade, la somme sur un périmètre, les frags de tout le lobby par camp.
//
// # LES DEUX PÉRIMÈTRES, ET POURQUOI IL N'Y EN A PAS TROIS
//
//	mesuré    passe lue (D8) ET notre camp connu : ses prises, son temps à bord, ses objets ;
//	rendement mesuré ET frags appariés (la dérivation a lu les frags d'engin) ET frags par camp
//	          lus de la base : les frags (barre épaisse), le temps à bord (barre fine) et le
//	          rendement se lisent sur ces matchs-là, et eux seuls (D9).
//
// # LES CAMPS
//
// Notre camp est celui du joueur de la page dans `match_participants` (`VehicleRead.PlayerTeam`) ;
// le camp d'une prise est celui du film (`roster[].team`), identique sur les huit matchs témoins.
// Un tueur sans camp connu dans un match à camp connu compte pour l'adversaire (règle du paquet).

import "levelup/go-api/internal/domain"

// VehicleRow — une ligne de prise de `match_vehicle_takes_latest` : un joueur, une famille, un
// camp, sur un match.
type VehicleRow struct {
	MatchID           string
	Camp              int
	XUID, Family      string
	Takes             int
	AboardMS          int64
	Episodes          int
	ProximityEpisodes int
	// Frags : frags de classe véhicule tombés PENDANT un épisode de ce joueur sur cette famille (D9).
	Frags int
}

// VehiclePass — la ligne `match` d'une passe : la couverture que la dérivation a écrite.
type VehiclePass struct {
	MatchID string
	// Measured faux : véhicules non mesurés (D8), Reason dit pourquoi.
	Measured  bool
	Reason    string
	DocSchema int
	// EpisodesRead / EpisodesUnnamed / EpisodesNoCamp : D10.
	EpisodesRead, EpisodesUnnamed, EpisodesNoCamp int
	// FragsRead faux : les frags d'engin n'ont pas pu être appariés (FragsReason dit pourquoi).
	FragsRead   bool
	FragsReason string
	// FragsTotal : frags d'engin du match ; FragsUnmatched : ceux qu'aucun épisode de leur tueur ne couvre.
	FragsTotal, FragsUnmatched int
}

// VehicleFragRow — les frags de classe véhicule d'un match obtenus par les tueurs d'un camp
// (D5 : tout le lobby, camp du tueur par `match_participants`). TeamID nil = tueur sans camp.
type VehicleFragRow struct {
	MatchID string
	TeamID  *int
	Frags   int
}

// VehicleRead — la lecture bornée des véhicules : un chargement par requête.
type VehicleRead struct {
	Passes []VehiclePass
	Rows   []VehicleRow
	// Frags : lignes par (match, camp) des frags d'engin, POUR LES SEULS matchs de EventsRead.
	Frags []VehicleFragRow
	// EventsRead : les matchs dont les événements de mort ont été lus et classés. Un match dont la
	// passe compte des frags d'engin et qui n'y figure pas est hors du périmètre du rendement.
	EventsRead map[string]bool
	// PlayerTeam : le camp du joueur de la page dans chaque match (`match_participants`).
	PlayerTeam map[string]int
}

// vehicleIndex — la lecture rangée par match.
type vehicleIndex struct {
	passes map[string]VehiclePass
	rows   map[string][]VehicleRow
	frags  map[string][]VehicleFragRow
	read   map[string]bool
	team   map[string]int
}

func newVehicleIndex(v *VehicleRead) *vehicleIndex {
	if v == nil {
		return nil
	}
	ix := &vehicleIndex{
		passes: make(map[string]VehiclePass, len(v.Passes)), rows: map[string][]VehicleRow{},
		frags: map[string][]VehicleFragRow{}, read: v.EventsRead, team: v.PlayerTeam,
	}
	for _, p := range v.Passes {
		ix.passes[p.MatchID] = p
	}
	for _, r := range v.Rows {
		ix.rows[r.MatchID] = append(ix.rows[r.MatchID], r)
	}
	for _, f := range v.Frags {
		ix.frags[f.MatchID] = append(ix.frags[f.MatchID], f)
	}
	return ix
}

// vehicleTally — ce qu'un match apporte de la ressource.
type vehicleTally struct {
	// state : "" (la ressource n'est pas lue), EmpriseVehiclesMeasured ou EmpriseVehiclesNotMeasured.
	state, reason string
	// inYield : le match est dans le périmètre commun des frags, du temps à bord et du rendement.
	inYield bool
	// aboard / paired / kills (0 = nous, 1 = eux) : sur les matchs du rendement seulement.
	aboard [2]int64
	paired [2]int
	kills  [2]int
	// Couverture : lue sur les matchs mesurés.
	episodesRead, episodesUnnamed, episodesNoCamp, proximity int
	fragsTotal, fragsUnmatched                               int
}

// tallyVehicles compte les véhicules d'un match dans t.veh, et ses prises par famille dans t.obj.
// Indépendant du film : la ressource vient de l'artefact, pas du résumé d'usage.
func tallyVehicles(t *matchTally, id string, ix *index) {
	v := ix.veh
	if v == nil {
		return
	}
	pass, hasPass := v.passes[id]
	ours, known := v.team[id]
	switch {
	case !hasPass:
		t.veh.state, t.veh.reason = domain.EmpriseVehiclesNotMeasured, domain.EmpriseVehiclesNoPass
		return
	case !pass.Measured:
		t.veh.state, t.veh.reason = domain.EmpriseVehiclesNotMeasured, pass.Reason
		return
	case !known:
		t.veh.state, t.veh.reason = domain.EmpriseVehiclesNotMeasured, domain.EmpriseVehiclesTeamUnknown
		return
	}
	t.veh.state = domain.EmpriseVehiclesMeasured
	t.veh.episodesRead, t.veh.episodesUnnamed, t.veh.episodesNoCamp = pass.EpisodesRead, pass.EpisodesUnnamed, pass.EpisodesNoCamp
	t.veh.inYield = pass.FragsRead && (pass.FragsTotal == 0 || v.read[id])
	if t.veh.inYield {
		t.veh.fragsTotal, t.veh.fragsUnmatched = pass.FragsTotal, pass.FragsUnmatched
		t.veh.kills = killsParCamp(v.frags[id], ours)
	}
	for _, r := range v.rows[id] {
		us := r.Camp == ours
		who := ""
		if us && ix.squad[r.XUID] {
			who = r.XUID
		}
		o := t.obj.get(domain.EmpriseResourceVehicle, r.Family)
		o.add(us, who, r.Takes, 0, 0)
		o.addAboard(us, who, r.AboardMS)
		t.veh.proximity += r.ProximityEpisodes
		if t.veh.inYield {
			s := sideIndex(us)
			t.veh.aboard[s] += r.AboardMS
			t.veh.paired[s] += r.Frags
		}
	}
}

// killsParCamp — les frags d'engin de chaque camp : le nôtre, puis tout le reste (un tueur sans
// camp compte pour l'adversaire).
func killsParCamp(rows []VehicleFragRow, ours int) [2]int {
	var k [2]int
	for _, r := range rows {
		if r.TeamID != nil && *r.TeamID == ours {
			k[0] += r.Frags
		} else {
			k[1] += r.Frags
		}
	}
	return k
}

// vehicleSum — les sommes d'un ensemble de matchs.
type vehicleSum struct {
	measured, notMeasured, yield int
	aboard                       [2]int64
	paired, kills                [2]int
	cov                          vehicleTally // couverture : les compteurs de t, cumulés
}

func (s *vehicleSum) add(t vehicleTally) {
	switch t.state {
	case domain.EmpriseVehiclesMeasured:
		s.measured++
	case domain.EmpriseVehiclesNotMeasured:
		s.notMeasured++
		return
	default:
		return
	}
	s.cov.episodesRead += t.episodesRead
	s.cov.episodesUnnamed += t.episodesUnnamed
	s.cov.episodesNoCamp += t.episodesNoCamp
	s.cov.proximity += t.proximity
	if !t.inYield {
		return
	}
	s.yield++
	s.cov.fragsTotal += t.fragsTotal
	s.cov.fragsUnmatched += t.fragsUnmatched
	for i := 0; i < 2; i++ {
		s.aboard[i] += t.aboard[i]
		s.paired[i] += t.paired[i]
		s.kills[i] += t.kills[i]
	}
}

// productionVehicules : frags depuis un véhicule (D5), temps à bord (D4), rendement par minute à
// bord sur les frags appariés (D9). Absente sans match du rendement ni grandeur.
func productionVehicules(s *soiree) (domain.SquadEmpriseProduction, bool) {
	v := &s.veh
	if v.yield == 0 || (v.aboard[0]+v.aboard[1] == 0 && v.kills[0]+v.kills[1] == 0) {
		return domain.SquadEmpriseProduction{}, false
	}
	kills := domain.SquadEmpriseCount{Us: v.kills[0], Them: v.kills[1]}
	paired := domain.SquadEmpriseCount{Us: v.paired[0], Them: v.paired[1]}
	p := domain.SquadEmpriseProduction{
		Resource: domain.EmpriseResourceVehicle,
		Kills:    kills,
		Exposure: &domain.SquadEmpriseExposure{
			Kind:        domain.EmpriseExposureAboardMS,
			Value:       domain.SquadEmpriseCount{Us: int(v.aboard[0]), Them: int(v.aboard[1])},
			Kills:       kills,
			PairedKills: &paired,
		},
		YieldUs:   parMinute(v.paired[0], v.aboard[0]),
		YieldThem: parMinute(v.paired[1], v.aboard[1]),
	}
	p.RelativeGap = ecartRelatif(p.YieldUs, p.YieldThem)
	return p, true
}

// couvertureVehicules — la couverture publiée. Nil quand la ressource n'est pas lue et n'a pas
// échoué (titre sans la capability) ; `unavailable` dit l'échec d'une lecture.
func couvertureVehicules(s *soiree, lue bool, unavailable string) *domain.SquadEmpriseVehicles {
	if unavailable != "" {
		return &domain.SquadEmpriseVehicles{Unavailable: unavailable}
	}
	if !lue {
		return nil
	}
	v := &s.veh
	c := &domain.SquadEmpriseVehicles{
		MatchesMeasured: v.measured, MatchesNotMeasured: v.notMeasured,
		EpisodesRead: v.cov.episodesRead, EpisodesUnnamed: v.cov.episodesUnnamed,
		EpisodesNoCamp: v.cov.episodesNoCamp, ProximityEpisodes: v.cov.proximity,
		FragsMatches: v.yield, FragsTotal: v.cov.fragsTotal,
		FragsPaired: v.cov.fragsTotal - v.cov.fragsUnmatched,
	}
	if c.FragsTotal > 0 {
		share := float64(c.FragsPaired) / float64(c.FragsTotal)
		c.PairedShare = &share
	}
	return c
}
