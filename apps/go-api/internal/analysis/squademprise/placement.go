package squademprise

// placement.go — LE BLOC « GROUPÉS OU ISOLÉS » (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot
// V3.2 ; type publié : domain.SquadEmprisePlacement).
//
// # L'UNIVERS D'UNE VIE
//
// Une ligne de la lecture n'entre que si elle est d'un joueur de la composition ET d'un match du
// périmètre (défense : le dépôt borne déjà les deux). Puis, match par match :
//
//	portée courante absente   la variante n'a plus (ou pas) de portée de radar : le match SORT de
//	                          l'univers (MatchesWithoutRange), aucune de ses vies n'est retenue ;
//	portée écrite ≠ courante  la table des portées a changé depuis le sync (ou la ligne n'a pas de
//	                          portée alors que la variante en a une) : la vie est ÉCARTÉE et
//	                          comptée (StaleLives), le service le journalise — un rattrapage est dû ;
//	sinon                     la vie est RETENUE : comptée, ses cumuls versés à la couverture.
//
// Une vie retenue est MESURÉE quand sa médiane existe (au moins 2 000 ms mesurées au sync) : elle
// est alors tracée et classée dans un quart ; sinon comptée « non mesurée ».
//
// Pur : aucune ouverture de base, aucune horloge. La portée courante est résolue par le service
// (`mappings.PorteeDuRadar`, source unique).

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// ordreDesQuarts — l'ordre de publication des quarts (domain.EmprisePlacement*).
var ordreDesQuarts = []string{
	domain.EmprisePlacementInRangeProductive, domain.EmprisePlacementIsolatedProductive,
	domain.EmprisePlacementInRangeCostly, domain.EmprisePlacementIsolatedCostly,
}

// PlacementInput — ce que le calcul demande.
type PlacementInput struct {
	// Players : la composition, dans l'ordre des fiches de l'Emprise.
	Players []domain.SessionUsageSquadPlayer
	// Scope : les matchs du périmètre D2, dans l'ordre chronologique.
	Scope []string
	Read  PlacementRead
	// CurrentRadar : match -> portée COURANTE du radar de sa variante, en mètres (> 0). Absent =
	// la variante n'a pas de portée connue.
	CurrentRadar map[string]float64
}

// PlacementBilan — ce que le service journalise ; jamais publié.
type PlacementBilan struct {
	// StaleMatches : les matchs dont au moins une vie est écartée pour portée périmée, dans
	// l'ordre du périmètre.
	StaleMatches []string
	// IgnoredRows : lignes hors composition ou hors périmètre (le dépôt n'en rend pas).
	IgnoredRows int
}

// Placement rend le bloc, ou nil quand aucun match du périmètre ne porte de vie écrite pour la
// composition.
func Placement(in PlacementInput) (*domain.SquadEmprisePlacement, PlacementBilan) {
	var bilan PlacementBilan
	ordre := make(map[string]int, len(in.Scope))
	for i, id := range in.Scope {
		ordre[id] = i
	}
	joueurs := make(map[string]*joueurPlacement, len(in.Players))
	for _, p := range in.Players {
		joueurs[p.XUID] = &joueurPlacement{}
	}
	parMatch := map[string][]PlacementRow{}
	for _, r := range in.Read.Rows {
		_, dansScope := ordre[r.MatchID]
		if _, membre := joueurs[r.XUID]; !dansScope || !membre {
			bilan.IgnoredRows++
			continue
		}
		parMatch[r.MatchID] = append(parMatch[r.MatchID], r)
	}
	if len(parMatch) == 0 {
		return nil, bilan
	}
	out := &domain.SquadEmprisePlacement{
		IsolatedFromRatio:   domain.EmprisePlacementIsolatedFromRatio,
		ProductiveFromKills: domain.EmprisePlacementProductiveFromKills,
		Coverage:            domain.SquadEmprisePlacementCoverage{MatchesTotal: len(in.Scope), MatchesWithPlacement: len(parMatch)},
	}
	for _, id := range in.Scope {
		lignes, ok := parMatch[id]
		if !ok {
			continue
		}
		rayon, connu := in.CurrentRadar[id]
		if !connu {
			out.Coverage.MatchesWithoutRange++
			continue
		}
		if retenirLeMatch(lignes, rayon, joueurs, &out.Coverage) {
			bilan.StaleMatches = append(bilan.StaleMatches, id)
		}
	}
	for _, p := range in.Players {
		trierLesVies(joueurs[p.XUID].vies, ordre)
		out.Players = append(out.Players, joueurs[p.XUID].publier(p))
	}
	return out, bilan
}

// joueurPlacement — les vies retenues d'un joueur, pendant le calcul.
type joueurPlacement struct {
	total int
	vies  []domain.SquadEmprisePlacementLife
}

// retenirLeMatch verse les vies d'un match à portée courante connue ; rend vrai si au moins une
// vie a été écartée pour portée périmée. Les lignes arrivent dans l'ordre du dépôt (joueur, début).
func retenirLeMatch(
	lignes []PlacementRow, rayon float64, joueurs map[string]*joueurPlacement, cov *domain.SquadEmprisePlacementCoverage,
) bool {
	perime := false
	for _, r := range lignes {
		if r.RadarM == nil || r.BeyondMS == nil || *r.RadarM != rayon {
			cov.StaleLives++
			perime = true
			continue
		}
		cov.LivesTotal++
		cov.MeasuredMS += r.MeasuredMS
		cov.CarrierMS += r.CarrierMS
		cov.TeamDownMS += r.TeamDownMS
		cov.UnplacedMS += r.UnplacedMS
		cov.TeammateUnplacedMS += r.TeammateUnplacedMS
		j := joueurs[r.XUID]
		j.total++
		if r.MedianM == nil {
			cov.LivesUnmeasured++
			continue
		}
		cov.LivesMeasured++
		j.vies = append(j.vies, vieTracee(r, rayon))
	}
	return perime
}

// vieTracee projette une vie mesurée : ses deux coordonnées et son quart.
func vieTracee(r PlacementRow, rayon float64) domain.SquadEmprisePlacementLife {
	v := domain.SquadEmprisePlacementLife{
		MatchID: r.MatchID, StartMS: r.StartMS, DurationMS: r.DurationMS,
		RadarRatio: *r.MedianM / rayon, Kills: r.Kills,
	}
	if r.MeasuredMS > 0 {
		v.OutOfRadarShare = float64(*r.BeyondMS) / float64(r.MeasuredMS)
	}
	v.Quadrant = quadrantDe(v.RadarRatio, v.Kills)
	return v
}

// quadrantDe — LE classement d'une vie (décision V4) : isolé = ratio ≥ 1,0 ; rentable = au moins
// un frag. Les deux bornes sont comprises.
func quadrantDe(radarRatio float64, kills int) string {
	isole := radarRatio >= domain.EmprisePlacementIsolatedFromRatio
	rentable := kills >= domain.EmprisePlacementProductiveFromKills
	switch {
	case rentable && !isole:
		return domain.EmprisePlacementInRangeProductive
	case rentable:
		return domain.EmprisePlacementIsolatedProductive
	case !isole:
		return domain.EmprisePlacementInRangeCostly
	default:
		return domain.EmprisePlacementIsolatedCostly
	}
}

// publier rend l'entrée d'un joueur : médianes et quarts sur ses vies mesurées.
func (j *joueurPlacement) publier(p domain.SessionUsageSquadPlayer) domain.SquadEmprisePlacementPlayer {
	out := domain.SquadEmprisePlacementPlayer{
		XUID: p.XUID, Gamertag: p.Gamertag, LivesTotal: j.total, LivesMeasured: len(j.vies),
		Lives: j.vies,
	}
	if out.Lives == nil {
		out.Lives = []domain.SquadEmprisePlacementLife{}
	}
	comptes := map[string]int{}
	ratios := make([]float64, 0, len(j.vies))
	frags := make([]float64, 0, len(j.vies))
	for _, v := range j.vies {
		comptes[v.Quadrant]++
		ratios = append(ratios, v.RadarRatio)
		frags = append(frags, float64(v.Kills))
	}
	if len(j.vies) > 0 {
		mr, mk := analysis.MedianFloat(ratios), analysis.MedianFloat(frags)
		out.MedianRadarRatio, out.MedianKills = &mr, &mk
	}
	for _, q := range ordreDesQuarts {
		quart := domain.SquadEmprisePlacementQuadrant{Quadrant: q, Lives: comptes[q]}
		if len(j.vies) > 0 {
			part := float64(comptes[q]) / float64(len(j.vies))
			quart.Share = &part
		}
		out.Quadrants = append(out.Quadrants, quart)
	}
	return out
}

// trierLesVies — garde d'ordre : chronologique dans le périmètre (rang du match, puis début).
func trierLesVies(vies []domain.SquadEmprisePlacementLife, ordre map[string]int) {
	sort.SliceStable(vies, func(a, b int) bool {
		if ordre[vies[a].MatchID] != ordre[vies[b].MatchID] {
			return ordre[vies[a].MatchID] < ordre[vies[b].MatchID]
		}
		return vies[a].StartMS < vies[b].StartMS
	})
}
