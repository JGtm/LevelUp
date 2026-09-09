// Package teammates - teammates_service_composition_sessions.go : construction
// de domain.CompositionSessionEntry (ADR 0033). Décrit, PAR SESSION de la
// composition sélectionnée : le compte POST-filtre (MatchCount, source unique —
// déjà publié par buildCompositionSessionLabels), le compte AVANT le filtre
// composition exacte (MatchCountRoster) et les matchs écartés par ce filtre avec
// le(s) coéquipier(s) connu(s) responsables, nommés
// (ExcludedByExactComposition).
//
// Décomposé de teammates_service.go (limite 500 lignes, CLAUDE.md règle 5).
package teammates

import "levelup/go-api/internal/domain"

// buildCompositionSessionEntries assemble les CompositionSessionEntry publiées
// par GetPage. kept/roster/excluded partagent tous la population
// allSquadRowsForTimeline (historique complet de la composition, non filtré par
// session ni période) à des étapes différentes du filtre composition exacte :
//   - kept    : après filtre (ou intersection brute si l'option est OFF) —
//     détermine QUELLES sessions apparaissent (une session à 0 match gardé
//     disparaît, comportement historique inchangé) ;
//   - roster  : avant filtre — fournit MatchCountRoster pour les sessions qui
//     apparaissent ;
//   - excluded : matchs écartés par le filtre — regroupés par session pour
//     ExcludedByExactComposition.
//
// teamByMatch/extraPool/topRows ne servent qu'à nommer les responsables d'un
// match écarté (extraPresentOn + résolution gamertag) ; nil/vide => aucun écart
// à publier (option OFF ou dégradation gracieuse du chargement de l'équipe).
func buildCompositionSessionEntries(
	kept, roster, excluded []domain.SquadMatchRow,
	teamByMatch map[string]map[string]struct{},
	extraPool map[string]struct{},
	topRows []domain.TopTeammateRow,
) []domain.CompositionSessionEntry {
	keptSessions := buildCompositionSessionLabels(kept)
	rosterCounts := sessionMatchCounts(roster)
	excludedBySession := groupExcludedBySession(excluded, teamByMatch, extraPool, topRows)

	out := make([]domain.CompositionSessionEntry, 0, len(keptSessions))
	for _, session := range keptSessions {
		out = append(out, domain.CompositionSessionEntry{
			SessionLabelEntry:          session,
			MatchCountRoster:           rosterCounts[session.Label],
			ExcludedByExactComposition: excludedBySession[session.Label],
		})
	}
	return out
}

// wrapSessionLabelsAsComposition adapte SessionLabelsList.Squad (aucun
// coéquipier sélectionné, pas de notion de composition exacte) au type
// CompositionSessionEntry : MatchCountRoster reprend MatchCount (rien à
// filtrer), aucun exclu.
func wrapSessionLabelsAsComposition(entries []domain.SessionLabelEntry) []domain.CompositionSessionEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]domain.CompositionSessionEntry, len(entries))
	for i, e := range entries {
		out[i] = domain.CompositionSessionEntry{SessionLabelEntry: e, MatchCountRoster: e.MatchCount}
	}
	return out
}

// sessionMatchCounts réutilise buildCompositionSessionLabels (dédup + regroupement
// par SessionLabel déjà correct) pour n'en extraire que le compte par label —
// jamais une seconde règle de dédup/regroupement (CLAUDE.md règle 6).
func sessionMatchCounts(rows []domain.SquadMatchRow) map[string]int {
	if len(rows) == 0 {
		return nil
	}
	out := make(map[string]int, len(rows))
	for _, e := range buildCompositionSessionLabels(rows) {
		out[e.Label] = e.MatchCount
	}
	return out
}

// groupExcludedBySession construit, pour chaque session, la liste des matchs
// écartés par le filtre composition exacte avec le(s) coéquipier(s) responsable(s)
// nommés (jamais un xuid nu, jamais une liste vide sous exclusion — repli "Joueur
// <4 derniers>", même convention que Q32b).
func groupExcludedBySession(
	excluded []domain.SquadMatchRow,
	teamByMatch map[string]map[string]struct{},
	extraPool map[string]struct{},
	topRows []domain.TopTeammateRow,
) map[string][]domain.CompositionExcludedMatch {
	if len(excluded) == 0 {
		return nil
	}
	gtByXUID := topGamertagsByXUID(topRows)
	out := make(map[string][]domain.CompositionExcludedMatch)
	for _, m := range excluded {
		if m.SessionLabel == nil || *m.SessionLabel == "" {
			continue
		}
		culprits := extraPresentOn(teamByMatch[m.MatchID], extraPool)
		gamertags := make([]string, 0, len(culprits))
		for _, xuid := range culprits {
			gamertags = append(gamertags, resolveGamertagFallback(xuid, gtByXUID))
		}
		label := *m.SessionLabel
		out[label] = append(out[label], domain.CompositionExcludedMatch{
			MatchID:        m.MatchID,
			StartTime:      m.StartTime,
			MapUI:          m.MapUI,
			ExtraGamertags: gamertags,
		})
	}
	return out
}

// countDistinctCulpritXUIDs dénombre les coéquipiers connus DISTINCTS
// responsables d'au moins une exclusion — dénominateur du log
// teammates.exact_composition_gap (A1.6).
func countDistinctCulpritXUIDs(
	excluded []domain.SquadMatchRow,
	teamByMatch map[string]map[string]struct{},
	extraPool map[string]struct{},
) int {
	seen := make(map[string]struct{})
	for _, m := range excluded {
		for _, xuid := range extraPresentOn(teamByMatch[m.MatchID], extraPool) {
			seen[xuid] = struct{}{}
		}
	}
	return len(seen)
}

// topGamertagsByXUID indexe topRows (Q29, déjà chargé) par xuid -> gamertag.
// Les gamertags vides (non résolus côté requête) ne sont pas indexés : le
// repli resolveGamertagFallback s'applique alors.
func topGamertagsByXUID(topRows []domain.TopTeammateRow) map[string]string {
	out := make(map[string]string, len(topRows))
	for _, r := range topRows {
		if r.XUID != "" && r.Gamertag != "" {
			out[r.XUID] = r.Gamertag
		}
	}
	return out
}

// resolveGamertagFallback résout un xuid en gamertag via la table topRows,
// avec le même repli que la requête SQL Q32b (Q32bMainTeamParticipantsTemplate,
// queries_squad.go) : "Joueur <4 derniers>" — jamais une chaîne vide.
func resolveGamertagFallback(xuid string, byXUID map[string]string) string {
	if gt, ok := byXUID[xuid]; ok && gt != "" {
		return gt
	}
	last4 := xuid
	if len(xuid) > 4 {
		last4 = xuid[len(xuid)-4:]
	}
	return "Joueur " + last4
}
