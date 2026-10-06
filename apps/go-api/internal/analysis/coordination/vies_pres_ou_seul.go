package coordination

// vies_pres_ou_seul.go — « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL » (Séries temporelles ›
// Usages ; type publié : domain.TimeseriesLivesNearTeammate).
//
// # LA VIE, SA FENÊTRE, SA MORT, SES FRAGS
//
//	vie       une vie du joueur (`match_lives`) close par une mort (`end_cause = death`) ;
//	fenêtre   [son début, le début de sa vie suivante dans le même match) — la règle de
//	          rattachement du décodeur (`replay/placement_des_vies.go`) : un frag posthume
//	          (grenade, échange) est celui de la vie qui vient de finir. La fin de la vie au film
//	          n'est pas l'instant de la mort au journal, d'où une fenêtre et non une clé ;
//	mort      la ligne de contexte (`match_death_context`) du joueur dans la fenêtre ;
//	rangement à portée du radar du match (borne incluse, la même comparaison que l'Isolement)
//	          ou au-delà ; sans distance (aucun coéquipier visible) ou sans contexte : écartée ;
//	          match sans portée : écartée ; les deux comptées ;
//	journal   un match dont la dernière passe du journal des morts n'est pas publiable n'a aucun
//	          frag lu : ses vies sont écartées et comptées AVANT toute autre cause (portée
//	          comprise), jamais rangées avec zéro frag — la règle de la lecture Tactique et du
//	          placement des vies du décodeur ;
//	frags     publiables, du joueur, sur un AUTRE camp ; trahison ou camp inconnu : écartés et
//	          comptés au bilan (l'appelant les journalise).
//
// Pur : aucune base, aucun fichier, aucune horloge.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// CauseVieMort — la cause de fin de vie qui dit que le joueur est mort (`match_lives.end_cause`).
const CauseVieMort = "death"

// ViesPresOuSeul range les vies terminées par une mort. `rayonParMatch` : la portée du radar de
// chaque match (absent = sans portée). Second retour : les frags écartés (trahison, camp inconnu).
func ViesPresOuSeul(l domain.ViesLues, rayonParMatch map[string]float64) (domain.TimeseriesLivesNearTeammate, int) {
	var out domain.TimeseriesLivesNearTeammate
	fragsEcartes := 0
	vies := viesParMatch(l.Vies)
	morts := mortsParMatch(l.Morts)
	frags := map[string][]int64{}
	for _, f := range l.Frags {
		if f.CampTueur == nil || f.CampVictime == nil || *f.CampTueur == *f.CampVictime {
			fragsEcartes++
			continue
		}
		frags[f.MatchID] = append(frags[f.MatchID], f.TimeMS)
	}
	for matchID, vs := range vies {
		out.MatchesRead++
		rayon, aUnRayon := rayonParMatch[matchID]
		if !aUnRayon {
			out.MatchesWithoutRadar++
		}
		for i, v := range vs {
			if v.EndCause != CauseVieMort {
				continue
			}
			fin := int64(-1) // -1 : pas de vie suivante, la fenêtre court jusqu'au bout du match
			if i+1 < len(vs) {
				fin = vs[i+1].StartMS
			}
			if l.JournalNonPubliable[matchID] {
				out.ExcludedUnpublishable++
				continue
			}
			rangerVie(&out, aUnRayon, rayon, mortDansFenetre(morts[matchID], v.StartMS, fin),
				compterDansFenetre(frags[matchID], v.StartMS, fin))
		}
	}
	return out, fragsEcartes
}

// rangerVie verse une vie terminée par une mort (d'un match au journal publiable) dans son côté,
// ou dans le compte de son écart.
func rangerVie(out *domain.TimeseriesLivesNearTeammate, aUnRayon bool, rayon float64, mort *domain.MortSituee, kills int) {
	switch {
	case !aUnRayon:
		out.ExcludedNoRadar++
	case mort == nil || mort.PlusProcheM == nil:
		out.ExcludedUnlocated++
	case aPortee(mort.PlusProcheM, rayon):
		out.Near.Lives++
		out.Near.Kills += kills
	default:
		out.Alone.Lives++
		out.Alone.Kills += kills
	}
}

// dansFenetre : début ≤ t < fin (fin = -1 : sans borne).
func dansFenetre(t, debut, fin int64) bool {
	return t >= debut && (fin < 0 || t < fin)
}

// mortDansFenetre rend la première mort de la fenêtre (nil : aucune).
func mortDansFenetre(morts []domain.MortSituee, debut, fin int64) *domain.MortSituee {
	for i := range morts {
		if dansFenetre(morts[i].TimeMS, debut, fin) {
			return &morts[i]
		}
	}
	return nil
}

func compterDansFenetre(instants []int64, debut, fin int64) int {
	n := 0
	for _, t := range instants {
		if dansFenetre(t, debut, fin) {
			n++
		}
	}
	return n
}

// viesParMatch range les vies par match, débuts croissants.
func viesParMatch(vies []domain.VieLue) map[string][]domain.VieLue {
	out := map[string][]domain.VieLue{}
	for _, v := range vies {
		out[v.MatchID] = append(out[v.MatchID], v)
	}
	for _, vs := range out {
		sort.SliceStable(vs, func(a, b int) bool { return vs[a].StartMS < vs[b].StartMS })
	}
	return out
}

// mortsParMatch range les morts par match, instants croissants.
func mortsParMatch(morts []domain.MortSituee) map[string][]domain.MortSituee {
	out := map[string][]domain.MortSituee{}
	for _, m := range morts {
		out[m.MatchID] = append(out[m.MatchID], m)
	}
	for _, ms := range out {
		sort.SliceStable(ms, func(a, b int) bool { return ms[a].TimeMS < ms[b].TimeMS })
	}
	return out
}

// aPortee : une distance mesurée, à la portée ou en deçà (borne incluse). Une distance absente
// n'est jamais à portée.
func aPortee(d *float64, rayon float64) bool {
	return d != nil && *d <= rayon
}
