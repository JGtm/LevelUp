package squademprise

// match.go — UN MATCH : ses prises par camp et par objet, ses issues et son effet des bonus,
// ses frags aux armes spéciales. La soirée et l'habitude ne sont que des sommes de matchs.

import (
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// index — les lectures rangées par match, et le camp de chaque participant.
type index struct {
	films      map[string]sessionusage.FilmRow
	players    map[string][]sessionusage.PlayerRow
	tiers      map[string][]sessionusage.PadTierRow
	tc         sessionusage.TeamContext
	powerKills map[string][]PowerKillRow
	squad      map[string]bool
	playerXUID string
	// veh : la ressource véhicules, nil quand elle n'est pas lue (vehicles.go).
	veh *vehicleIndex
}

func newIndex(in *Input) *index {
	ix := &index{
		players: map[string][]sessionusage.PlayerRow{}, tiers: map[string][]sessionusage.PadTierRow{},
		powerKills: map[string][]PowerKillRow{}, squad: map[string]bool{}, playerXUID: in.PlayerXUID,
	}
	for _, p := range in.Players {
		ix.squad[p.XUID] = true
	}
	for _, r := range in.PowerKills {
		ix.powerKills[r.MatchID] = append(ix.powerKills[r.MatchID], r)
	}
	ix.veh = newVehicleIndex(in.Vehicles)
	if in.Film == nil {
		return ix
	}
	ix.films = in.Film.Films
	for _, p := range in.Film.Players {
		ix.players[p.MatchID] = append(ix.players[p.MatchID], p)
	}
	for _, r := range in.Film.PadTiers {
		ix.tiers[r.MatchID] = append(ix.tiers[r.MatchID], r)
	}
	ix.tc = sessionusage.BuildTeamContext(in.PlayerXUID, in.Film.Participants)
	return ix
}

// camp — notre camp dans un match.
type camp struct {
	ours   int
	teamOf map[string]int
	squad  map[string]bool
}

// side dit si ce xuid est de notre camp et, s'il l'est, sous quelle entrée il compte : son xuid
// s'il est un joueur des fiches, "" (reste du camp) sinon.
func (c camp) side(xuid string) (us bool, who string) {
	if t, ok := c.teamOf[xuid]; !ok || t != c.ours {
		return false, ""
	}
	if c.squad[xuid] {
		return true, xuid
	}
	return true, ""
}

// sideIndex : 0 = nous, 1 = eux.
func sideIndex(us bool) int {
	if us {
		return 0
	}
	return 1
}

// matchTally — ce qu'un match apporte.
type matchTally struct {
	hasFilm, teamKnown bool
	// filmTeam : le camp du joueur est connu des participants lus avec le film.
	filmTeam    bool
	tiers       string
	obj         *objets
	outcomes    [2]sessionusage.OutcomeCounts
	effectMS    [2]int64
	effectKills [2]int
	pwk         *domain.SquadEmpriseCount
	// unclassified : les prises sur un emplacement non identifié ; nil sans ligne `non_classe`.
	unclassified *domain.SquadEmpriseCount
	// veh : les véhicules du match (vehicles.go), indépendants du film.
	veh vehicleTally
}

// bonusMeasured : les prises et issues des bonus se lisent sur ce match.
func (t matchTally) bonusMeasured() bool { return t.hasFilm && t.filmTeam }

// tiersMeasured : armes spéciales et râteliers se lisent sur ce match.
func (t matchTally) tiersMeasured() bool {
	return t.bonusMeasured() && t.tiers == domain.EmpriseTiersMeasured
}

// tallyMatch compte un match.
func tallyMatch(id string, ix *index) matchTally {
	t := matchTally{obj: newObjets(), pwk: powerKillsOf(ix.powerKills[id], ix.playerXUID)}
	tallyVehicles(&t, id, ix)
	ours, known := ix.tc.PlayerTeam[id]
	t.filmTeam, t.teamKnown = known, known || t.pwk != nil
	film, ok := ix.films[id]
	if !ok {
		return t
	}
	t.hasFilm = true
	t.tiers = tiersState(ix.tiers[id])
	if !known {
		return t
	}
	c := camp{ours: ours, teamOf: ix.tc.TeamOf[id], squad: ix.squad}
	tallyBonus(&t, c, film, ix.players[id])
	t.unclassified = unclassifiedOf(c, ix.tiers[id])
	if t.tiers == domain.EmpriseTiersMeasured {
		tallyTiers(&t, c, ix.tiers[id])
	}
	return t
}

// timeScaled dit si le film d'un match a une échelle de temps. Sans elle (`duration_ms` nul,
// artefact d'un ancien schéma : `frame_interval_ms = 0`), les temps d'effet valent 0 et le
// schéma interdit de les croire (steps_shared_usage_summary.go) : le match sort du rendement
// des bonus, frags ET temps d'effet, comme il sort des cadences de Sessions
// (countMeasuredWithoutDuration). Ses prises et issues restent comptées.
func timeScaled(film sessionusage.FilmRow) bool { return film.DurationMS > 0 }

// WithoutTimeScale compte les matchs filmés du périmètre sans échelle de temps : le service le
// journalise, comme Sessions (une exclusion ne se fait jamais en silence).
func WithoutTimeScale(in *Input) int {
	if in.Film == nil {
		return 0
	}
	n := 0
	for _, m := range in.Current {
		if f, ok := in.Film.Films[m.MatchID]; ok && !timeScaled(f) {
			n++
		}
	}
	return n
}

// tallyBonus compte les deux bonus : prises attribuées, issues, effet (si le film a une échelle
// de temps), socles vidés.
func tallyBonus(t *matchTally, c camp, film sessionusage.FilmRow, players []sessionusage.PlayerRow) {
	families := sessionusage.PowerupFamilies()
	scaled := timeScaled(film)
	for i := range players {
		p := &players[i]
		us, who := c.side(p.XUID)
		s := sideIndex(us)
		for _, f := range families {
			oc := sessionusage.PlayerOutcomeCounts(p, []string{f})
			t.obj.get(domain.EmpriseResourcePowerup, f).add(us, who, oc.Taken, oc.Kept, oc.Dropped)
			t.outcomes[s].Add(oc)
			if !scaled {
				continue
			}
			ms, k := sessionusage.PowerupEffect(p, f)
			t.effectMS[s] += ms
			t.effectKills[s] += k
		}
	}
	for _, f := range families {
		if n, ok := film.PowerupPickups[f]; ok {
			o := t.obj.get(domain.EmpriseResourcePowerup, f)
			o.padsEmptied += n
			o.hasPads = true
		}
	}
}

// tallyTiers compte les prises sur les socles de puissance et les râteliers.
func tallyTiers(t *matchTally, c camp, rows []sessionusage.PadTierRow) {
	for _, r := range rows {
		res := tierResource(r.Tier)
		if res == "" || r.Pickups <= 0 {
			continue
		}
		us, who := c.side(r.XUID)
		t.obj.get(res, r.WeaponFamily).add(us, who, r.Pickups, 0, 0)
	}
}

// unclassifiedOf compte les prises sur un emplacement non identifié (niveau `non_classe`), équipe
// contre adversaire, quel que soit l'état des niveaux du match ; nil sans aucune.
func unclassifiedOf(c camp, rows []sessionusage.PadTierRow) *domain.SquadEmpriseCount {
	var n domain.SquadEmpriseCount
	for _, r := range rows {
		if r.Tier != domain.PadTierUnclassified || r.Pickups <= 0 {
			continue
		}
		if us, _ := c.side(r.XUID); us {
			n.Us += r.Pickups
		} else {
			n.Them += r.Pickups
		}
	}
	if n.Us+n.Them == 0 {
		return nil
	}
	return &n
}

// tiersState — l'état des niveaux de socle d'un match. Les valeurs de match (socles vus,
// socles confirmés) sont identiques sur toutes ses lignes.
func tiersState(rows []sessionusage.PadTierRow) string {
	if len(rows) == 0 {
		return domain.EmpriseTiersNotMeasured
	}
	if rows[0].PadsTotal > 0 && rows[0].PadsConfirmed == 0 {
		return domain.EmpriseTiersUnestablished
	}
	return domain.EmpriseTiersMeasured
}

// powerKillsOf — les frags aux armes spéciales de chaque camp. Nil quand le camp du joueur est
// inconnu ou qu'aucune ligne ne porte la grandeur.
func powerKillsOf(rows []PowerKillRow, playerXUID string) *domain.SquadEmpriseCount {
	ours, known := -1, false
	for _, r := range rows {
		if r.XUID == playerXUID && r.TeamID != nil {
			ours, known = *r.TeamID, true
		}
	}
	if !known {
		return nil
	}
	var c domain.SquadEmpriseCount
	lu := false
	for _, r := range rows {
		if r.Kills == nil {
			continue
		}
		lu = true
		if r.TeamID != nil && *r.TeamID == ours {
			c.Us += *r.Kills
		} else {
			c.Them += *r.Kills
		}
	}
	if !lu {
		return nil
	}
	return &c
}

// publierMatch projette un match au contrat.
func publierMatch(m Match, t matchTally, in *Input) domain.SquadEmpriseMatch {
	pm := domain.SquadEmpriseMatch{
		MatchID: m.MatchID, HasFilm: t.hasFilm, TeamKnown: t.teamKnown,
		PowerWeaponKills: t.pwk, Resources: []domain.SquadEmpriseMatchResource{},
		Vehicles: t.veh.state, VehiclesReason: t.veh.reason, UnclassifiedPickups: t.unclassified,
	}
	if t.hasFilm {
		pm.Tiers = t.tiers
	}
	for _, res := range resourceOrder {
		if !t.obj.has(res) {
			continue
		}
		pm.Resources = append(pm.Resources, domain.SquadEmpriseMatchResource{
			Resource: res, Taken: t.obj.total(res), Objects: t.obj.publier(res, in.Players, in),
		})
	}
	return pm
}
