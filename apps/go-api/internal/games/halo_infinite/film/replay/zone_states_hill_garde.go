package replay

// zone_states_hill_garde.go — OU EST LA COLLINE D UNE PERIODE : LA OU SE TIENT LE CAMP QUE LE FILM
// DIT PROPRIETAIRE.
//
// LA REGLE. Une colline se tient en s y tenant : tant que le canal de propriete de l objet de mode
// nomme un camp, au moins un joueur de ce camp est dans la colline (une colline que son camp quitte
// se vide puis repasse au neutre, et la jauge de l objet de mode le montre). La colline
// d une periode est donc la zone du catalogue ou le camp proprietaire est PRESENT pendant les frames
// ou il la tient : deux lectures du film (le proprietaire, l equipe de chaque vie) et la geometrie
// des pistes, sans dependre de la jauge.
//
// LE DECOMPTE est en FRAMES, pas en positions : une frame tenue compte une fois pour chaque zone
// ou se trouve au moins un joueur du camp proprietaire. Le denominateur est le nombre de frames
// tenues ou au moins une position de ce camp est publiee (une frame sans aucune piste du camp ne
// dit rien de l endroit).
//
// TROIS ISSUES, et la troisieme n est pas un echec :
//
//	placee     la zone en tete est NETTE (`clearModalZone`) et couvre au moins
//	           `hillGardeMinPresence` des frames observees ;
//	ecartee    la garde est lisible mais aucune zone ne la couvre assez : la colline de cette
//	           periode n est pas au catalogue de la carte, et la poser sur la plus proche serait
//	           afficher une colline ou il n y en a pas (la periode se compte dans `unpaired`) ;
//	illisible  aucune frame tenue observee (canal de propriete muet, ou equipes des vies non
//	           lues) : l appelant revient aux votes par la jauge, repli nomme
//	           `repli_colline_votes_sans_garde`.

// hillGardeMinPresence est la part minimale des frames tenues observees ou le camp proprietaire doit
// etre dans la zone retenue. Sur les films a collines du parc, la colline vraie couvre 64 a 100 %
// de ces frames et une periode dont la colline manque au catalogue 25 a 27 % : le seuil tombe dans
// l ecart (journal du plan `.ai/PLAN_KOTH_REJEU_2026-10-09.md`).
const hillGardeMinPresence = 0.5

// hillTeamPoint est une position publiee et le camp de la vie qui la porte.
type hillTeamPoint struct {
	team int
	pt   Point
}

// hillTeamPointsByFrame indexe les positions des vies A CAMP CONNU par frame (une vie sans equipe
// lue ne peut temoigner d aucune garde).
func hillTeamPointsByFrame(tracks []Track) map[int][]hillTeamPoint {
	out := map[int][]hillTeamPoint{}
	for _, tr := range tracks {
		if tr.Team < 0 {
			continue
		}
		for _, p := range tr.Points {
			out[p.T] = append(out[p.T], hillTeamPoint{team: tr.Team, pt: p})
		}
	}
	return out
}

// hillGarde est le decompte de garde d une periode.
type hillGarde struct {
	// votes : par zone du catalogue, les frames tenues ou le camp proprietaire y a un joueur.
	votes map[int]int
	// observed : les frames tenues ou au moins une position du camp proprietaire est publiee.
	observed int
}

// hillGardeOf compte la garde de la periode [t0, t1] sur le canal de propriete `owner`.
func hillGardeOf(zones []Zone, pts map[int][]hillTeamPoint, owner []zoneSample, t0, t1 int) hillGarde {
	g := hillGarde{votes: map[int]int{}}
	cur, k := uint64(zoneNeutralOwner), 0
	for f := t0; f <= t1; f++ {
		for k < len(owner) && owner[k].t <= f {
			cur = owner[k].v
			k++
		}
		if cur == zoneNeutralOwner {
			continue
		}
		hillGardeFrame(zones, pts[f], int(cur), &g) //nolint:gosec // valeur de camp bornee par le canal
	}
	return g
}

// hillGardeFrame ajoute une frame tenue par `team` au decompte.
func hillGardeFrame(zones []Zone, pts []hillTeamPoint, team int, g *hillGarde) {
	seen, vus := false, map[int]bool{}
	for _, x := range pts {
		if x.team != team {
			continue
		}
		seen = true
		if best, hits := nearestZones(zones, x.pt); len(hits) == 1 && best <= zoneCaptureDistanceM {
			vus[hits[0].SpatialRank] = true
		}
	}
	if !seen {
		return
	}
	g.observed++
	for ref := range vus {
		g.votes[ref]++
	}
}

// hillPlacement est l issue de la garde d une periode.
type hillPlacement int

const (
	hillGardeIllisible hillPlacement = iota
	hillGardePlacee
	hillGardeEcartee
)

// place rend la zone de la periode et l issue (cf. l en-tete).
func (g hillGarde) place() (int, hillPlacement) {
	if g.observed == 0 {
		return 0, hillGardeIllisible
	}
	ref, ok := clearModalZone(g.votes)
	if !ok || float64(g.votes[ref]) < hillGardeMinPresence*float64(g.observed) {
		return 0, hillGardeEcartee
	}
	return ref, hillGardePlacee
}

// hillPeriodTop retient le sommet de jauge le plus haut des montees qui tombent dans la periode : la
// progression publiee, quelle que soit la voie qui a place la periode.
func hillPeriodTop(ramps []zoneRamp, p *hillPeriod) {
	for _, r := range ramps {
		if r.tPeak < p.t0 || r.t0 > p.t1 {
			continue
		}
		if !p.hasTop || r.top > p.top {
			p.top, p.hasTop = r.top, true
		}
	}
}
