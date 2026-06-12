package prestige

// squad_progress.go — attribution match→escouade, règle « no-overlap ».
//
// Cœur algorithmique de la progression d'un défi d'escouade (Phase C,
// PLAN_COACH_V3_GENERATION § Identité d'escouade). Pure logique, sans accès DB :
// les ensembles (roster, coéquipiers connus, participants) sont fournis par
// l'orchestration (qui lit shared.match_participants + ListSquadsForUser).
//
// Règle « session » validée produit : un match compte pour une escouade S ssi
//   (1) tout le roster de S est présent dans le match, ET
//   (2) AUCUN autre coéquipier connu — membre d'une AUTRE escouade du joueur —
//       n'est présent, ET
//   (3) les randoms (xuid hors de tout roster connu) sont ignorés (ils comblent
//       les trous en 4v4 / 16v16).
//
// Conséquence voulue : un match du trio {A,B,C} ne crédite QUE le trio ; il ne
// crédite pas un duo {A,B} défini (C, coéquipier connu, est présent). Un membre
// qui revient en session ne voit donc jamais l'objectif « avoir bougé sans lui ».

// SquadMatchParticipants décrit les participants (xuids) d'un match, pour
// l'attribution no-overlap. MatchID identifie le match.
type SquadMatchParticipants struct {
	MatchID string
	Xuids   []string
}

// MatchCountsForSquad applique la règle no-overlap pour UN match.
//
// roster              : membres de l'escouade évaluée (xuids).
// otherKnownTeammates : coéquipiers connus HORS roster (union des autres
//
//	escouades du joueur, moins le roster) — cf. OtherKnownTeammates.
//
// participants        : xuids présents dans le match (roster + autres + randoms).
//
// Un roster vide ne crédite aucun match (pas d'escouade sans membre).
func MatchCountsForSquad(roster, otherKnownTeammates map[string]struct{}, participants []string) bool {
	if len(roster) == 0 {
		return false
	}
	present := make(map[string]struct{}, len(participants))
	for _, p := range participants {
		present[p] = struct{}{}
	}
	// (1) tout le roster doit être présent.
	for x := range roster {
		if _, ok := present[x]; !ok {
			return false
		}
	}
	// (2) aucun coéquipier connu hors roster ne doit être présent.
	for _, p := range participants {
		if _, isOther := otherKnownTeammates[p]; !isOther {
			continue
		}
		if _, inRoster := roster[p]; !inRoster {
			return false
		}
	}
	return true
}

// OtherKnownTeammates calcule l'ensemble des coéquipiers connus HORS roster
// courant : union des membres de TOUTES les escouades du joueur, privée des
// membres du roster courant. C'est l'ensemble qui « disqualifie » un match au
// titre de la règle (2).
func OtherKnownTeammates(currentRoster []string, allSquadRosters [][]string) map[string]struct{} {
	inRoster := toXUIDSet(currentRoster)
	other := make(map[string]struct{})
	for _, r := range allSquadRosters {
		for _, x := range r {
			if _, ok := inRoster[x]; ok {
				continue
			}
			other[x] = struct{}{}
		}
	}
	return other
}

// FilterSquadMatches retourne les MatchID des matchs qui comptent pour
// l'escouade définie par roster, sachant otherKnownTeammates. L'ordre d'entrée
// est préservé.
func FilterSquadMatches(roster []string, otherKnownTeammates map[string]struct{}, matches []SquadMatchParticipants) []string {
	rs := toXUIDSet(roster)
	var out []string
	for _, m := range matches {
		if MatchCountsForSquad(rs, otherKnownTeammates, m.Xuids) {
			out = append(out, m.MatchID)
		}
	}
	return out
}

// SquadMatchMetric décrit un match candidat pour l'évaluation d'un défi
// d'escouade : ses participants (pour la règle no-overlap) + la valeur de la
// métrique du défi par membre du roster sur ce match.
//
// Values est indexé par xuid (membres du roster) ; un membre absent de la map
// pour un match n'a pas contribué (ne compte pas dans son cumul).
type SquadMatchMetric struct {
	MatchID string
	Xuids   []string
	Values  map[string]float64
}

// SquadParticipantProgress est la progression cumulée d'un membre du roster sur
// un défi d'escouade.
type SquadParticipantProgress struct {
	Xuid      string  `json:"xuid"`
	Value     float64 `json:"value"`     // métrique cumulée sur les matchs comptés
	Matches   int     `json:"matches"`   // nb de matchs comptés où le membre a contribué
	Completed bool    `json:"completed"` // Value >= targetPerMember (mode cumulatif)
}

// AggregateSquadProgress calcule la progression de chaque membre du roster sur
// un défi d'escouade en **mode cumulatif** (collectif) : on ne garde que les
// matchs qui comptent pour l'escouade (règle no-overlap, cf. MatchCountsForSquad)
// puis on somme la métrique par membre. Completed si la somme atteint
// targetPerMember (> 0). Le résultat suit l'ordre de `roster`.
//
// Le mode « threshold » (cible atteinte sur un match plutôt que cumulée) n'est
// pas couvert en V1 — à ajouter quand un défi d'escouade threshold sera proposé.
func AggregateSquadProgress(roster []string, otherKnownTeammates map[string]struct{}, matches []SquadMatchMetric, targetPerMember float64) []SquadParticipantProgress {
	rs := toXUIDSet(roster)
	sums := make(map[string]float64, len(roster))
	counts := make(map[string]int, len(roster))
	for _, m := range matches {
		if !MatchCountsForSquad(rs, otherKnownTeammates, m.Xuids) {
			continue
		}
		for x := range rs {
			if v, ok := m.Values[x]; ok {
				sums[x] += v
				counts[x]++
			}
		}
	}
	out := make([]SquadParticipantProgress, 0, len(roster))
	for _, x := range roster {
		v := sums[x]
		out = append(out, SquadParticipantProgress{
			Xuid:      x,
			Value:     v,
			Matches:   counts[x],
			Completed: targetPerMember > 0 && v >= targetPerMember,
		})
	}
	return out
}

// toXUIDSet construit un set depuis une liste de xuids.
func toXUIDSet(xs []string) map[string]struct{} {
	s := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		s[x] = struct{}{}
	}
	return s
}
