package sessionusage

// objectives.go — le bloc 3 de la session : OBJECTIFS PAR RÔLE (prendre /
// défendre / tenir) et par famille de mode, agrégés depuis
// `match_objective_stats_latest` (les deux camps y sont déjà : RIEN n'est
// produit, tout est lu — §5/S2 du handoff).
//
// # LA TABLE DES RÔLES EST LA SOURCE UNIQUE, ET ELLE VIT DANS narrative
//
// La classification colonne → rôle est narrative.ObjectiveRoleColumns
// (objective_roles.go) : elle est assise sur les constantes de colonnes
// d'objective_participation.go et verrouillée par le garde-rail de partition
// TestObjectiveRoles_PartitionDesColonnesDAction. Ce package n'en recopie RIEN :
// la couche repo GÉNÈRE ses sommes par rôle depuis cette table, et la présence
// d'un rôle pour une famille se DÉRIVE ici de l'intersection rôle ∩ vocabulaire
// de la famille (ObjectiveFamilyActionWeights + ObjectiveFamilyHoldColumns) —
// aucune liste parallèle.
//
// Conséquence de la classification narrative, à connaître en lisant le §7 du
// handoff : `flag_grabs` est HORS rôle (écartée des tables de poids), donc les
// prises de drapeau brutes ne comptent pas dans « prendre » — seuls captures,
// assists de capture, vols et retourneurs abattus y comptent.

import (
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// ObjectiveRow — une ligne (match, joueur) déjà projetée par rôle (repo :
// ObjectiveStatsRepo.LoadObjectiveRoleRows, sommes générées depuis
// narrative.ObjectiveRoleColumns).
type ObjectiveRow struct {
	MatchID     string
	XUID        string
	Family      narrative.ObjectiveFamily
	Take        float64
	Defend      float64
	HoldSeconds float64
}

// ObjectivesInput — le scope objectifs de la session : les lignes, et le
// contexte de camp/effectif PAR MATCH (mêmes participants que le bloc usage).
type ObjectivesInput struct {
	PlayerXUID string
	// SquadXUIDs : coéquipiers suivis (contexte escouade) — une ligne Squad par
	// rôle et par xuid, dans cet ordre. Vide = contexte solo.
	SquadXUIDs []string
	Rows       []ObjectiveRow
	// PlayerTeam : matchID -> camp du joueur suivi (clé absente = inconnu).
	PlayerTeam map[string]int
	// TeamOf : matchID -> (xuid -> camp).
	TeamOf map[string]map[string]int
	// TeamSize / LobbySize : matchID -> effectif présent à la fin.
	TeamSize  map[string]int
	LobbySize map[string]int
}

// ComputeObjectives agrège le bloc objectifs. nil si aucune ligne (session sans
// mode à objectif : le bloc est OMIS, pas servi à zéro).
func ComputeObjectives(in ObjectivesInput) *domain.SessionObjectivesBlock {
	if len(in.Rows) == 0 {
		return nil
	}
	matchIDs := distinctMatchIDs(in.Rows)
	out := &domain.SessionObjectivesBlock{MatchesWithObjectives: len(matchIDs)}
	out.TeamSizeAvg, out.TeamParityPct = averageSizeOf(matchIDs, in.TeamSize)
	out.LobbySizeAvg, out.LobbyParityPct = averageSizeOf(matchIDs, in.LobbySize)
	out.Roles = objectiveRoleMetrics(in, in.Rows)

	for _, fam := range narrative.AllObjectiveFamilies() {
		var famRows []ObjectiveRow
		for _, r := range in.Rows {
			if r.Family == fam {
				famRows = append(famRows, r)
			}
		}
		if len(famRows) == 0 {
			continue
		}
		out.Families = append(out.Families, domain.SessionObjectiveFamilyBlock{
			Family:  string(fam),
			Matches: len(distinctMatchIDs(famRows)),
			Roles:   filterRolesForFamily(fam, objectiveRoleMetrics(in, famRows)),
		})
	}
	return out
}

// objectiveRoleMetrics — les trois rôles (prendre, défendre, tenir) sur un
// sous-ensemble de lignes, chacun en triplet joueur/camp/lobby + parts, et une
// ligne Squad par coéquipier suivi.
//
// MÊME RÈGLE DE SCOPE que computeMetric (usage.go) : les grandeurs d'équipe se
// calculent sur le sous-ensemble des matchs du scope à camp connu — numérateurs
// ET dénominateurs (sinon player_share_of_team dépasse 100 % dès que le scope
// mêle matchs à camp connu et inconnu) ; sous-ensemble vide = nil (un scope
// entièrement à camp inconnu ne publie AUCUNE part d'équipe, jamais un 0).
func objectiveRoleMetrics(in ObjectivesInput, rows []ObjectiveRow) []domain.SessionObjectiveRoleMetric {
	roles := narrative.AllObjectiveRoles()
	out := make([]domain.SessionObjectiveRoleMetric, 0, len(roles))
	for _, role := range roles {
		m := domain.SessionObjectiveRoleMetric{
			Role:       string(role),
			IsDuration: role == narrative.ObjectiveRoleHold,
		}
		var teamSum, playerTeamScope, lobbyTeamScope float64 // scope camp connu
		teamKnown := false
		squadTotals := make(map[string]float64, len(in.SquadXUIDs))
		squadTeamScope := make(map[string]float64, len(in.SquadXUIDs))
		for i := range rows {
			r := &rows[i]
			v := objectiveRoleValue(role, r)
			m.LobbyTotal += v
			if r.XUID == in.PlayerXUID {
				m.PlayerTotal += v
			}
			if playerTeam, ok := in.PlayerTeam[r.MatchID]; ok {
				teamKnown = true
				lobbyTeamScope += v
				if r.XUID == in.PlayerXUID {
					playerTeamScope += v
				}
				if teamID, known := in.TeamOf[r.MatchID][r.XUID]; known && teamID == playerTeam {
					teamSum += v
				}
				squadTeamScope[r.XUID] += v
			}
			squadTotals[r.XUID] += v
		}
		if teamKnown {
			team := teamSum
			m.TeamTotal = &team
			m.TeamShareOfLobbyPct = sharePct(teamSum, lobbyTeamScope)
			m.PlayerShareOfTeamPct = sharePct(playerTeamScope, teamSum)
		}
		m.PlayerShareOfLobbyPct = sharePct(m.PlayerTotal, m.LobbyTotal)
		for _, x := range in.SquadXUIDs {
			m.Squad = append(m.Squad, domain.SessionUsageSquadShare{
				XUID:            x,
				Total:           squadTotals[x],
				ShareOfTeamPct:  sharePct(squadTeamScope[x], teamSum),
				ShareOfLobbyPct: sharePct(squadTotals[x], m.LobbyTotal),
			})
		}
		out = append(out, m)
	}
	return out
}

// objectiveRoleValue — la valeur d'un rôle sur une ligne projetée.
func objectiveRoleValue(role narrative.ObjectiveRole, r *ObjectiveRow) float64 {
	switch role {
	case narrative.ObjectiveRoleTake:
		return r.Take
	case narrative.ObjectiveRoleDefend:
		return r.Defend
	case narrative.ObjectiveRoleHold:
		return r.HoldSeconds
	}
	return 0
}

// filterRolesForFamily retire d'un bloc de famille les rôles SANS colonne pour
// cette famille (extraction n'a pas de « tenir ») : un zéro structurel n'est pas
// une mesure. La présence se dérive de narrative (rôle ∩ vocabulaire de la
// famille), jamais d'une liste locale.
func filterRolesForFamily(fam narrative.ObjectiveFamily, roles []domain.SessionObjectiveRoleMetric) []domain.SessionObjectiveRoleMetric {
	out := make([]domain.SessionObjectiveRoleMetric, 0, len(roles))
	for _, r := range roles {
		if familyHasRole(fam, narrative.ObjectiveRole(r.Role)) {
			out = append(out, r)
		}
	}
	return out
}

// familyHasRole — la famille possède-t-elle au moins une colonne du rôle ?
// Vocabulaire de la famille : clés d'ObjectiveFamilyActionWeights + colonnes
// d'ObjectiveFamilyHoldColumns (les deux sources uniques de narrative).
func familyHasRole(fam narrative.ObjectiveFamily, role narrative.ObjectiveRole) bool {
	vocab := map[string]bool{}
	for col := range narrative.ObjectiveFamilyActionWeights[fam] {
		vocab[col] = true
	}
	for _, col := range narrative.ObjectiveFamilyHoldColumns[fam] {
		vocab[col] = true
	}
	for _, col := range narrative.ObjectiveRoleColumns(role) {
		if vocab[col] {
			return true
		}
	}
	return false
}

func distinctMatchIDs(rows []ObjectiveRow) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		if !seen[r.MatchID] {
			seen[r.MatchID] = true
			out = append(out, r.MatchID)
		}
	}
	return out
}

// averageSizeOf — effectif moyen sur les matchs du scope où il est connu (>0),
// et la parité 100/moyenne. (0, nil) quand aucun match ne le porte. Tri des ids
// inutile au calcul mais gardé déterministe par construction (ordre des rows).
func averageSizeOf(matchIDs []string, sizes map[string]int) (float64, *float64) {
	sum, n := 0, 0
	for _, id := range matchIDs {
		if s := sizes[id]; s > 0 {
			sum += s
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	avg := float64(sum) / float64(n)
	p := 100 / avg
	return avg, &p
}
