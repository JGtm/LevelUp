package tactical

// previsions_reapparition.go — LA PREVISION DE REAPPARITION PENDANT LE COMPTE A REBOURS : pour
// chaque mort d'un match suivie d'une reapparition, la probabilite de chaque zone nommee, une fois
// par seconde, a partir de la position des autres joueurs A CET INSTANT.
//
// C'est le modele de reapparition.go (memes sept variables, memes poids) applique a la place du
// rejeu : a l'instant d'un echantillon, les joueurs vivants se lisent dans l'index des vies du
// match (IndexVies.Vivants, la meme lecture que la mesure M-4 a l'instant de la mort), les
// candidats sont les points de la carte, et la probabilite d'une ZONE est la SOMME des
// probabilites de ses points candidats (decision D-3 : le texte parle en zones nommees).
//
// # UNE PREVISION NE VOIT PAS LA REPONSE
//
// Les candidats se deduisent des reapparitions des AUTRES matchs de la carte : les evenements du
// match prevu sont ecartes avant toute construction (evenementsHors), si bien qu'un point vu dans
// ce seul match n'est pas candidat et que la frequence d'un point ne compte jamais ce match. C'est
// la regle de la validation croisee (CandidatPour) appliquee ici par construction.
//
// # CE QUI N'EST PAS PREVU
//
// Une mort sans reapparition (derniere vie, fin de match) n'a pas de compte a rebours. Une mort
// dont l'equipe est inconnue, dont les positions ne sont pas finies, ou qui n'a aucun adversaire
// vivant a aucun instant du compte a rebours n'est pas prevue : elle se COMPTE (MortsEcartees),
// elle ne disparait pas en silence.
//
// PUR : aucune I/O.

import (
	"math"
	"sort"

	"levelup/go-api/internal/domain"
)

// Statuts d'une prevision de match.
const (
	// PrevisionPrete : au moins une mort a une prevision.
	PrevisionPrete = "ready"
	// PrevisionCartePauvre : la carte n'a pas assez de matchs, de reapparitions ou de points candidats
	// hors le match prevu pour qu'une prevision ait un sens.
	PrevisionCartePauvre = "not_enough_matches"
	// PrevisionSansZone : la carte n'a aucune zone nommee ; sans elles la prevision ne parle a
	// personne.
	PrevisionSansZone = "no_zone"
	// PrevisionAucuneReapparition : le match n'a aucune mort suivie d'une reapparition prevue.
	PrevisionAucuneReapparition = "no_respawn"
)

// Reglages de la prevision.
const (
	// PasPrevisionMs : une prevision par seconde de compte a rebours.
	PasPrevisionMs = 1000
	// MatchsMinCartePrevision : matchs minimum de la famille sur la carte, le match prevu compris. C'est
	// le plancher de matchs de la lecture Tactique (domain.ReapparitionsAdversesPlancherMatchs, une seule
	// valeur) et le seuil de l'ajustement des poids (levelup spawn-model fit).
	MatchsMinCartePrevision = domain.ReapparitionsAdversesPlancherMatchs
	// ReapparitionsMinCartePrevision : reapparitions minimum des AUTRES matchs de la famille sur la
	// carte (le match prevu n'y compte pas : une prevision ne voit pas la reponse). C'est le plancher de
	// reapparitions de la lecture Tactique (domain.ReapparitionsAdversesPlancherReapparitions, une seule
	// valeur) : le meme scope de donnees decide des deux lectures, aucun mode n'est ecarte a la main.
	ReapparitionsMinCartePrevision = domain.ReapparitionsAdversesPlancherReapparitions
	// PrecisionProbabilite : les probabilites se publient au millieme, une zone dont la
	// probabilite arrondie est nulle ne se publie pas.
	PrecisionProbabilite = 1000.0
	// CandidatsMinPrevision : en dessous, un choix « parmi les candidats » n'en est pas un (meme
	// seuil que la construction des echantillons).
	CandidatsMinPrevision = 2
)

// EntreePrevision est tout ce que la prevision d'un match demande.
type EntreePrevision struct {
	// Match est le match prevu : ses pistes (positions des joueurs) et les debuts de compte a
	// rebours que le film porte.
	Match EntreeOccupation
	// MatchID est l'identifiant sous lequel les evenements de ce match figurent dans Evenements.
	MatchID string
	// Evenements sont les reapparitions de la carte et de la famille de modes : seuls Reel et
	// MatchID servent (les candidats). Ceux du match prevu sont ecartes.
	Evenements []EvenementReapparition
	// Poids est le jeu de poids de la famille (PoidsPourFamille).
	Poids [NbVariables]float64
	// Zones sont les zones nommees de la carte.
	Zones []ZoneNommee
}

// ProbaZone est la probabilite qu'une zone porte la reapparition. Zone est l'indice de volume de
// la zone dans son catalogue (ZoneNommee.VolumeIndex).
type ProbaZone struct {
	Zone int
	P    float64
}

// EchantillonPrevision est la prevision a un instant du compte a rebours : les zones de la plus
// probable a la moins probable (ex aequo : indice de volume croissant), et la part des candidats
// qu'aucune zone nommee ne porte.
type EchantillonPrevision struct {
	Frame    int
	Zones    []ProbaZone
	HorsZone float64
}

// PrevisionDeMort est la prevision d'une mort suivie d'une reapparition.
type PrevisionDeMort struct {
	// XUID est le joueur mort : la prevision lui appartient.
	XUID string
	// Mort est l'endroit et l'instant ou sa vie s'est arretee ; FrameReapparition l'instant ou
	// la suivante commence, Reel le point ou elle commence.
	Mort              PointMort
	FrameReapparition int
	Reel              Point3
	// ZoneReelle est la zone qui nomme le point reel, -1 si aucune.
	ZoneReelle   int
	Echantillons []EchantillonPrevision
}

// PrevisionsDuMatch est la prevision de toutes les reapparitions d'un match.
type PrevisionsDuMatch struct {
	Statut string
	// MatchsCarte : matchs de la famille sur la carte, le match prevu compris ; Candidats : les
	// points candidats hors le match prevu.
	MatchsCarte int
	Candidats   int
	Morts       []PrevisionDeMort
	// MortsEcartees : reapparitions que la prevision ne peut pas servir (cf. l'en-tete).
	MortsEcartees int
}

// previseur porte ce que les echantillons d'un match partagent.
type previseur struct {
	index *IndexVies
	// pasFrames est la duree d'une seconde en frames du document.
	pasFrames int
	cands     []Candidat
	// zonesCand : la zone de chaque candidat (indice de volume, -1 aucune).
	zonesCand []int
	poids     [NbVariables]float64
	zones     []ZoneNommee
}

// PrevoirLesReapparitions prevoit, pour chaque mort suivie d'une reapparition, la zone de la
// reapparition a chaque seconde du compte a rebours. Les morts sont rendues par instant de mort
// croissant, puis par xuid.
func PrevoirLesReapparitions(e EntreePrevision) PrevisionsDuMatch {
	autres := evenementsHors(e.Evenements, e.MatchID)
	cands := PointsCandidats(autres)
	res := PrevisionsDuMatch{MatchsCarte: matchsDistincts(autres) + 1, Candidats: len(cands)}
	switch {
	case len(e.Zones) == 0:
		res.Statut = PrevisionSansZone
		return res
	case res.MatchsCarte < MatchsMinCartePrevision || len(autres) < ReapparitionsMinCartePrevision ||
		len(cands) < CandidatsMinPrevision:
		res.Statut = PrevisionCartePauvre
		return res
	case e.Match.IntervalleFrameMs <= 0:
		res.Statut = PrevisionAucuneReapparition
		return res
	}
	p := &previseur{
		index: NouvelIndexVies(e.Match), cands: cands, poids: e.Poids, zones: e.Zones,
		pasFrames: max(1, int(math.Round(float64(PasPrevisionMs)/float64(e.Match.IntervalleFrameMs)))),
	}
	p.zonesCand = make([]int, len(cands))
	for i, c := range cands {
		p.zonesCand[i] = IndiceDeZone(e.Zones, c.Pos)
	}
	p.prevoirLesMorts(e.Match, &res)
	if len(res.Morts) == 0 {
		res.Statut = PrevisionAucuneReapparition
		return res
	}
	res.Statut = PrevisionPrete
	return res
}

// prevoirLesMorts parcourt les pistes nommees dans l'ordre du document et prevoit chaque
// reapparition apres mort.
func (p *previseur) prevoirLesMorts(entree EntreeOccupation, res *PrevisionsDuMatch) {
	spawns := p.index.Spawns()
	for i, piste := range entree.Pistes {
		s, ok := spawns[i]
		if !ok || !s.Reapparition || s.Mort == nil {
			continue
		}
		mort, ok := p.prevoirUneMort(piste, s)
		if !ok {
			res.MortsEcartees++
			continue
		}
		res.Morts = append(res.Morts, mort)
	}
	sort.SliceStable(res.Morts, func(a, b int) bool {
		x, y := res.Morts[a], res.Morts[b]
		if x.Mort.Frame != y.Mort.Frame {
			return x.Mort.Frame < y.Mort.Frame
		}
		return x.XUID < y.XUID
	})
}

// prevoirUneMort prevoit la reapparition `s` du joueur de la piste. Faux : la mort n'est pas
// previsible (equipe inconnue, positions non finies, aucun adversaire vivant du compte a rebours).
func (p *previseur) prevoirUneMort(piste Piste, s SpawnPiste) (PrevisionDeMort, bool) {
	equipe := piste.Equipe
	if equipe < 0 {
		equipe = p.index.EquipeDuJoueur(piste.XUID)
	}
	reel := Point3{X: s.X, Y: s.Y, Z: s.Z}
	mort := Point3{X: s.Mort.X, Y: s.Mort.Y, Z: s.Mort.Z}
	if equipe < 0 || !reel.Fini() || !mort.Fini() {
		return PrevisionDeMort{}, false
	}
	out := PrevisionDeMort{
		XUID: piste.XUID, Mort: *s.Mort, FrameReapparition: s.Frame, Reel: reel,
		ZoneReelle: IndiceDeZone(p.zones, reel),
	}
	for f := s.Mort.Frame; f < s.Frame; f += p.pasFrames {
		ev := evenementAuInstant(p.index.Vivants(piste.XUID, f), equipe, mort)
		if len(ev.Ennemis) == 0 {
			continue
		}
		out.Echantillons = append(out.Echantillons, p.echantillon(f, ev))
	}
	return out, len(out.Echantillons) > 0
}

// echantillon note les candidats pour l'etat du jeu `ev` et somme les probabilites par zone.
func (p *previseur) echantillon(frame int, ev EvenementReapparition) EchantillonPrevision {
	scores := make([]float64, len(p.cands))
	for i, c := range p.cands {
		scores[i] = Score(Variables(ev, c, c.Total, nil), p.poids)
	}
	proba := Probabilites(scores)
	somme := map[int]float64{}
	horsZone := 0.0
	for i, pr := range proba {
		if z := p.zonesCand[i]; z >= 0 {
			somme[z] += pr
		} else {
			horsZone += pr
		}
	}
	out := EchantillonPrevision{Frame: frame, HorsZone: arrondiProbabilite(horsZone)}
	for z, v := range somme {
		if r := arrondiProbabilite(v); r > 0 {
			out.Zones = append(out.Zones, ProbaZone{Zone: z, P: r})
		}
	}
	sort.Slice(out.Zones, func(a, b int) bool {
		if out.Zones[a].P != out.Zones[b].P {
			return out.Zones[a].P > out.Zones[b].P
		}
		return out.Zones[a].Zone < out.Zones[b].Zone
	})
	return out
}

func arrondiProbabilite(v float64) float64 {
	return math.Round(v*PrecisionProbabilite) / PrecisionProbabilite
}

// IndiceDeZone rend l'indice de volume de la zone qui nomme le point (regle de NommerZone, avec la
// hauteur du point), -1 si aucune zone ne le nomme.
func IndiceDeZone(zones []ZoneNommee, p Point3) int {
	r, ok := NommerZone(p.X, p.Y, []float64{p.Z}, zones)
	if !ok {
		return -1
	}
	return r.Zone.VolumeIndex
}

// evenementsHors rend les evenements qui ne sont pas du match donne : la prevision ne voit pas
// la reponse.
func evenementsHors(evts []EvenementReapparition, matchID string) []EvenementReapparition {
	out := make([]EvenementReapparition, 0, len(evts))
	for _, e := range evts {
		if e.MatchID != matchID {
			out = append(out, e)
		}
	}
	return out
}

func matchsDistincts(evts []EvenementReapparition) int {
	vus := map[string]struct{}{}
	for _, e := range evts {
		vus[e.MatchID] = struct{}{}
	}
	return len(vus)
}

// evenementAuInstant separe les joueurs vivants en equipiers et adversaires du point de vue de
// `equipe` ; un joueur dont l'equipe est inconnue ou la position non finie n'est compte dans
// aucun des deux. `mort` est le lieu de la mort du joueur prevu.
func evenementAuInstant(vivants []Vivant, equipe int, mort Point3) EvenementReapparition {
	e := EvenementReapparition{Mort: mort}
	for _, v := range vivants {
		pos := Point3{X: v.X, Y: v.Y, Z: v.Z}
		if !pos.Fini() || v.Equipe < 0 {
			continue
		}
		if v.Equipe == equipe {
			e.Equipiers = append(e.Equipiers, pos)
		} else {
			e.Ennemis = append(e.Ennemis, pos)
		}
	}
	return e
}

// Fini dit si les trois coordonnees sont des nombres finis (ni NaN ni infini).
func (p Point3) Fini() bool {
	for _, v := range [3]float64{p.X, p.Y, p.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
