// Package service — squad_service_v2_impact.go : helpers Impact 8 roles pour
// l'onglet Synergies de la page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase P5).
//
// Utilise narrative.IdentifyImpactRoles (Phase 0 chunk 5) pour attribuer les
// 8 roles narratifs (first_blood, clutch_finisher, last_casualty,
// last_group_kill, first_group_death, silent_hero, false_brother, top_killer)
// par match × joueur.
//
// Cote rendu :
//   - ImpactRolesMatrix : heatmap match × joueur, cellules avec emojis
//   - ImpactRanking     : 8 colonnes (1 par role), tri par count desc
package service

import (
	"sort"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildImpactRolesMatrix construit la matrice match × joueur des roles
// d'impact attribues.
//
// Parametres :
//
//	events       : highlight_events filmes des matchs partages (chargees via
//	               port.HighlightEventsRepository, capability match.detail.events).
//	squadOrder   : ordre stable des gamertags (main + coequipiers tels que la
//	               page les affiche). Sert a peupler ImpactRolesMatrix.SquadGamertags.
//	squadXUIDs   : map gamertag -> xuid pour resoudre les RoleAssignment.XUID.
//	               Le service amont fournit cette correspondance.
//	sharedMatches : pour hydrater MainOutcome + StartedAt par match.
func BuildImpactRolesMatrix(
	events []canonical.HighlightEvent,
	squadOrder []string,
	squadXUIDs map[string]string,
	sharedMatches []domain.SquadSharedMatch,
) domain.ImpactRolesMatrix {
	xuidToGT := reverseMap(squadXUIDs)
	xuids := make([]string, 0, len(squadXUIDs))
	for _, xuid := range squadXUIDs {
		xuids = append(xuids, xuid)
	}
	teamOutcomes := buildTeamOutcomes(sharedMatches, squadXUIDs)

	roleAssignments := narrative.IdentifyImpactRoles(events, teamOutcomes, xuids)
	byMatch := groupAssignmentsByMatch(roleAssignments, xuidToGT)

	rows := make([]domain.ImpactRolesMatchRow, 0, len(sharedMatches))
	for _, sm := range sharedMatches {
		row := domain.ImpactRolesMatchRow{
			MatchID:       sm.MatchID,
			StartedAt:     sm.StartedAt,
			MainOutcome:   sm.Outcome,
			RolesByPlayer: byMatch[sm.MatchID],
		}
		if row.RolesByPlayer == nil {
			row.RolesByPlayer = make(map[string][]domain.ImpactRoleCell)
		}
		rows = append(rows, row)
	}
	return domain.ImpactRolesMatrix{
		MatchRows:      rows,
		SquadGamertags: append([]string{}, squadOrder...),
	}
}

// reverseMap inverse une map gamertag -> xuid en xuid -> gamertag.
func reverseMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[v] = k
	}
	return out
}

// buildTeamOutcomes construit la map xuid -> Outcome pour chaque membre du squad.
//
// L'outcome d'un xuid est resolu depuis le premier SharedMatch ou le joueur
// est present (les Outcomes sont identiques pour les membres d'une meme equipe
// sur un match donne, mais on prend la decision UX de propager le MainOutcome
// de chaque match aux roles, pour matcher la coloration cellule = outcome).
//
// Pour les xuids non presents dans sharedMatches (cas degrade), on omet -
// IdentifyImpactRoles tolere les xuid manquants dans teamOutcomes (default zero).
func buildTeamOutcomes(
	sharedMatches []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) map[string]canonical.Outcome {
	out := make(map[string]canonical.Outcome, len(squadXUIDs))
	for _, sm := range sharedMatches {
		for gt, xuid := range squadXUIDs {
			if _, ok := out[xuid]; ok {
				continue
			}
			if row, present := sm.Players[gt]; present {
				out[xuid] = row.Self.Outcome
			}
		}
		if len(out) == len(squadXUIDs) {
			break
		}
	}
	return out
}

// groupAssignmentsByMatch groupe les RoleAssignment par match_id puis par
// gamertag (resolu via xuidToGT). Les xuids hors du squad sont ignores.
func groupAssignmentsByMatch(
	assignments []narrative.RoleAssignment,
	xuidToGT map[string]string,
) map[string]map[string][]domain.ImpactRoleCell {
	out := make(map[string]map[string][]domain.ImpactRoleCell)
	for _, a := range assignments {
		gt, ok := xuidToGT[a.XUID]
		if !ok {
			continue
		}
		if out[a.MatchID] == nil {
			out[a.MatchID] = make(map[string][]domain.ImpactRoleCell)
		}
		out[a.MatchID][gt] = append(out[a.MatchID][gt], domain.ImpactRoleCell{
			RoleKey:    string(a.Role),
			LabelKey:   a.LabelKey,
			ColorToken: a.ColorToken,
			Inverted:   a.Inverted,
		})
	}
	return out
}

// invertedRoles est l'ensemble des roles "negatifs" : le ranking utilise
// un gradient couleur inverse pour ces roles (Boulet plutot que MVP).
var invertedRoles = map[narrative.ImpactRole]bool{
	narrative.RoleLastCasualty:    true,
	narrative.RoleLastGroupKill:   true, // partiellement inverted (cf. assignations)
	narrative.RoleFirstGroupDeath: true,
	narrative.RoleFalseBrother:    true,
}

// allRolesOrder est l'ordre stable d'affichage des colonnes du ranking.
var allRolesOrder = []narrative.ImpactRole{
	narrative.RoleFirstBlood,
	narrative.RoleClutchFinisher,
	narrative.RoleTopKiller,
	narrative.RoleSilentHero,
	narrative.RoleLastCasualty,
	narrative.RoleLastGroupKill,
	narrative.RoleFirstGroupDeath,
	narrative.RoleFalseBrother,
}

// BuildImpactRanking construit le tableau MVP/Boulet : 1 colonne par role
// avec le classement des joueurs du squad par count desc sur ce role.
//
// Les roles sont retournes dans l'ordre allRolesOrder (positifs en premier,
// negatifs ensuite). Chaque entry est triee par count desc, fallback
// gamertag asc en cas d'egalite.
//
// Tous les xuids du squad apparaissent dans les entries (count=0 inclus)
// pour permettre au front de rendre une ligne par joueur dans chaque colonne.
func BuildImpactRanking(matrix domain.ImpactRolesMatrix) []domain.ImpactRanking {
	if len(matrix.SquadGamertags) == 0 {
		return nil
	}

	// Compter les roles par (gamertag, role).
	counts := make(map[narrative.ImpactRole]map[string]int, len(allRolesOrder))
	for _, role := range allRolesOrder {
		counts[role] = make(map[string]int, len(matrix.SquadGamertags))
		for _, gt := range matrix.SquadGamertags {
			counts[role][gt] = 0 // initialiser tous les joueurs
		}
	}
	for _, row := range matrix.MatchRows {
		for gt, cells := range row.RolesByPlayer {
			for _, cell := range cells {
				role := narrative.ImpactRole(cell.RoleKey)
				if _, known := counts[role]; !known {
					continue
				}
				counts[role][gt]++
			}
		}
	}

	out := make([]domain.ImpactRanking, 0, len(allRolesOrder))
	for _, role := range allRolesOrder {
		entries := make([]domain.ImpactRankingEntry, 0, len(matrix.SquadGamertags))
		for _, gt := range matrix.SquadGamertags {
			entries = append(entries, domain.ImpactRankingEntry{
				Gamertag: gt,
				Count:    counts[role][gt],
			})
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Count != entries[j].Count {
				return entries[i].Count > entries[j].Count
			}
			return entries[i].Gamertag < entries[j].Gamertag
		})
		out = append(out, domain.ImpactRanking{
			RoleKey:  string(role),
			LabelKey: "narrative.role." + string(role),
			Inverted: invertedRoles[role],
			Entries:  entries,
		})
	}
	return out
}
