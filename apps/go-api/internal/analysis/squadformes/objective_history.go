package squadformes

// objective_history.go — LE RAPPORT DE FORCE À L'OBJECTIF, SOIRÉE APRÈS SOIRÉE (lot L3 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décisions D6 et D7). Voir le type publié,
// domain.SquadObjectiveHistory, pour la définition de la mesure.
//
// # POURQUOI DANS CE PAQUET
//
// La part d'un rôle se lit sur les colonnes de la famille de mode QUI APPARTIENNENT à ce rôle :
// c'est familyColumnsOfRole, déjà écrit ici pour les grilles d'objectif. Un calcul ailleurs en
// aurait fait une troisième copie (règle n°6 de CLAUDE.md : sessionusage.familyHasRole en porte
// déjà une deuxième).
//
// # CE QUI N'ENTRE PAS DANS UN RÔLE
//
// Les grandeurs FACULTATIVES (prises nettes de drapeau, lues du film) : elles ne sont mesurées
// que sur les matchs dont l'artefact a été lu, et les verser dans un rôle ferait varier son
// dénominateur d'une soirée à l'autre sans que rien ne le dise (D7).
//
// # CE QUE LE TITRE DÉCIDE EN AMONT
//
// Le mode écarté (le drapeau neutre, D6 : aucun retour possible, donc une part mécanique) est
// un savoir du titre : l'appelant le marque sur HistoryMatch.Excluded. Ce paquet n'importe
// aucun paquet de titre (ADR 0012 / 0025).
//
// Pur : aucune ouverture de base, aucune horloge, aucune chaîne de langue.

import (
	"sort"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// HistoryMatch — un match de la composition, tel que la page le connaît.
type HistoryMatch struct {
	MatchID      string
	SessionLabel string
	StartTime    time.Time
	// Won : notre camp a gagné le match.
	Won bool
	// Excluded : mode écarté de la mesure par le titre (drapeau neutre, D6).
	Excluded bool
}

// HistoryInput — tout ce que le calcul demande.
type HistoryInput struct {
	// Current : les matchs de la soirée affichée (le périmètre D2 de la page).
	Current []HistoryMatch
	// Timeline : l'historique complet de la composition (toutes ses soirées).
	Timeline []HistoryMatch
	// Rows : les lignes d'objectif (les deux camps) de Current ∪ Timeline.
	Rows []ObjectiveColumnRow
	// Camp : par match, les xuids de notre camp (le joueur de la page compris). Un match
	// absent n'a pas de camp connu : il ne compte dans aucune soirée.
	Camp map[string]map[string]struct{}
}

// objectiveRoles — les trois rôles, dans l'ordre de publication.
var objectiveRoles = narrative.AllObjectiveRoles()

// matchRoleShares — les parts d'un match, par rôle. ok[i] faux = le lobby n'a rien fait sur ce
// rôle (ou la famille n'en a pas) : le match ne pèse pas dans la moyenne de ce rôle.
type matchRoleShares struct {
	family narrative.ObjectiveFamily
	share  [3]float64
	ok     [3]bool
}

// BuildObjectiveHistory rend le bloc publié.
func BuildObjectiveHistory(in HistoryInput) domain.SquadObjectiveHistory {
	parts := sharesByMatch(in.Rows, in.Camp)
	current := buildEvening(in.Current, parts)
	current.SessionLabel = commonSessionLabel(in.Current)

	inCurrent := make(map[string]bool, len(in.Current))
	firstCurrent := time.Time{}
	for _, m := range in.Current {
		inCurrent[m.MatchID] = true
		if firstCurrent.IsZero() || m.StartTime.Before(firstCurrent) {
			firstCurrent = m.StartTime
		}
	}

	out := domain.SquadObjectiveHistory{
		Current:             current,
		Previous:            []domain.SquadObjectiveEvening{},
		MinObjectiveMatches: domain.SquadObjectiveMinMatches,
	}
	for _, s := range groupSessions(in.Timeline) {
		ev := buildEvening(s.matches, parts)
		ev.SessionLabel = s.label
		if ev.ObjectiveMatches > 0 {
			out.EveningsWithObjective++
			if ev.ObjectiveMatches < domain.SquadObjectiveMinMatches {
				out.EveningsBelowMinimum++
			}
		}
		if ev.ObjectiveMatches < domain.SquadObjectiveMinMatches || touches(s.matches, inCurrent) {
			continue
		}
		// UNE SOIRÉE « PRÉCÉDENTE » EST JOUÉE AVANT LA SOIRÉE AFFICHÉE : un périmètre filtré sur
		// une soirée ancienne ne se compare pas à celles qui l'ont suivie.
		if !firstCurrent.IsZero() && !s.first.Before(firstCurrent) {
			continue
		}
		out.Previous = append(out.Previous, ev)
	}
	if n := len(out.Previous); n > domain.SquadObjectiveHistoryMaxPrevious {
		out.Previous = out.Previous[n-domain.SquadObjectiveHistoryMaxPrevious:]
	}
	return out
}

// sharesByMatch — les parts de chaque match qui a une feuille d'objectif ET un camp connu.
func sharesByMatch(rows []ObjectiveColumnRow, camp map[string]map[string]struct{}) map[string]matchRoleShares {
	byMatch := map[string][]ObjectiveColumnRow{}
	for _, r := range rows {
		byMatch[r.MatchID] = append(byMatch[r.MatchID], r)
	}
	out := make(map[string]matchRoleShares, len(byMatch))
	for id, list := range byMatch {
		ours, known := camp[id]
		if !known || len(list) == 0 {
			continue
		}
		fam := list[0].Family
		var s matchRoleShares
		s.family = fam
		for i, role := range objectiveRoles {
			cols := roleColumns(fam, role)
			var us, lobby float64
			for _, r := range list {
				v := 0.0
				for _, c := range cols {
					v += r.Values[c]
				}
				lobby += v
				if _, in := ours[r.XUID]; in {
					us += v
				}
			}
			if lobby > 0 {
				s.share[i], s.ok[i] = us/lobby, true
			}
		}
		out[id] = s
	}
	return out
}

// roleColumns — les colonnes d'un rôle pour une famille, SANS les grandeurs facultatives (D7).
func roleColumns(fam narrative.ObjectiveFamily, role narrative.ObjectiveRole) []string {
	var out []string
	for _, c := range familyColumnsOfRole(fam, role) {
		if _, optional := narrative.ObjectiveExtraGrandeurFamily(c); optional {
			continue
		}
		out = append(out, c)
	}
	return out
}

// buildEvening — une soirée : ses matchs à objectif (dédoublonnés, modes écartés retirés), ses
// victoires, son mélange de modes et la moyenne des parts de chaque rôle (D7).
func buildEvening(matches []HistoryMatch, parts map[string]matchRoleShares) domain.SquadObjectiveEvening {
	var ev domain.SquadObjectiveEvening
	seen := map[string]bool{}
	families := map[string]int{}
	var sum [3]float64
	var n [3]int
	var first time.Time
	for _, m := range matches {
		if seen[m.MatchID] || m.Excluded {
			continue
		}
		seen[m.MatchID] = true
		p, ok := parts[m.MatchID]
		if !ok {
			continue
		}
		ev.ObjectiveMatches++
		if m.Won {
			ev.Wins++
		}
		families[string(p.family)]++
		if first.IsZero() || m.StartTime.Before(first) {
			first = m.StartTime
		}
		for i := range objectiveRoles {
			if p.ok[i] {
				sum[i] += p.share[i]
				n[i]++
			}
		}
	}
	if !first.IsZero() {
		ev.StartTime = first.UTC().Format(time.RFC3339)
	}
	ev.Families = familyCounts(families)
	ev.Take, ev.Defend, ev.Hold = mean(sum[0], n[0]), mean(sum[1], n[1]), mean(sum[2], n[2])
	return ev
}

func mean(sum float64, n int) *float64 {
	if n == 0 {
		return nil
	}
	v := sum / float64(n)
	return &v
}

// familyCounts — le mélange de modes, du plus joué au moins joué (clé en départage).
func familyCounts(counts map[string]int) []domain.SquadObjectiveFamilyCount {
	out := make([]domain.SquadObjectiveFamilyCount, 0, len(counts))
	for f, c := range counts {
		out = append(out, domain.SquadObjectiveFamilyCount{Family: f, Matches: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Matches != out[j].Matches {
			return out[i].Matches > out[j].Matches
		}
		return out[i].Family < out[j].Family
	})
	return out
}

// session — une soirée de l'historique : son libellé, ses matchs, son premier instant.
type session struct {
	label   string
	matches []HistoryMatch
	first   time.Time
}

// groupSessions groupe l'historique par libellé de session, dans l'ordre chronologique du
// premier match. Un match sans session n'appartient à aucune soirée : la mesure se lit par
// soirée, jamais par match isolé.
func groupSessions(matches []HistoryMatch) []session {
	byLabel := map[string]*session{}
	var order []*session
	for _, m := range matches {
		if m.SessionLabel == "" {
			continue
		}
		s := byLabel[m.SessionLabel]
		if s == nil {
			s = &session{label: m.SessionLabel, first: m.StartTime}
			byLabel[m.SessionLabel] = s
			order = append(order, s)
		}
		s.matches = append(s.matches, m)
		if m.StartTime.Before(s.first) {
			s.first = m.StartTime
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].first.Before(order[j].first) })
	out := make([]session, 0, len(order))
	for _, s := range order {
		out = append(out, *s)
	}
	return out
}

// touches dit si la soirée contient un match de la soirée affichée.
func touches(matches []HistoryMatch, current map[string]bool) bool {
	for _, m := range matches {
		if current[m.MatchID] {
			return true
		}
	}
	return false
}

// commonSessionLabel — le libellé de session partagé par tous les matchs, vide sinon.
func commonSessionLabel(matches []HistoryMatch) string {
	label := ""
	for _, m := range matches {
		if m.SessionLabel == "" || (label != "" && m.SessionLabel != label) {
			return ""
		}
		label = m.SessionLabel
	}
	return label
}
