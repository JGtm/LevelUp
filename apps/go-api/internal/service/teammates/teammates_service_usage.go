// Package teammates — teammates_service_usage.go : LES BLOCS D'USAGE DE LA PAGE TEAMMATES —
// « formes retenues » (lot D2), l'historique d'objectif (lot L3 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) et l'Emprise (lot L4).
//
// Le bloc « servi ou gâché » de l'équipement (`equipment_usage`, étape E6.1bis du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09) n'est PLUS publié ici depuis le lot L5.4 du même plan :
// son seul lecteur était l'ancien onglet Usages de l'Escouade, remplacé par l'onglet Emprise.
// Il reste servi au solo (Séries temporelles, Synthèse) par son producteur
// (squadagg.BuildEquipmentUsageBlock).
//
// PÉRIMÈTRE (décision D2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, qui remplace
// celle d'E6.1bis) : les matchs de la COMPOSITION EXACTE intersectés avec les matchs filtrés
// — allSquadRows (population escouade, option composition exacte comprise :
// filterExactComposition s'y applique dans GetPage) ∩ filteredMatches (période, cascade,
// sessions). Sans coéquipier sélectionné, filteredMatches seul : la page reste utile en solo.
// Avant D2, le périmètre était filteredMatches même avec une escouade : les cartes d'usage
// comptaient des matchs joués sans les coéquipiers affichés. Les « amis » des blocs sont les
// coéquipiers SÉLECTIONNÉS (req.SelectedGamertags), pas les amis configurés (app_settings).
package teammates

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithUsageSummary injecte la source du résumé d'usage (vues _latest) — le MÊME repo que les
// pages Sessions et Synthèse : lectures communes des blocs d'usage et de l'Emprise. Câblé gated
// par film.usage_summary (registry, jamais slug==) ; nil ⇒ l'Emprise publie la seule feuille de
// match (film_unsupported), réponse partielle propre.
func (s *TeammatesService) WithUsageSummary(repo port.SessionUsageRepository) *TeammatesService {
	s.sessionUsageRepo = repo
	return s
}

// porteeUsage — ce que GetPage donne aux blocs d'usage : les deux populations dont le
// périmètre D2 est l'intersection, l'historique de la composition et son camp par match
// (historique d'objectif), l'historique de matchs de la page (libellés) et les sessions de la
// composition (compte de matchs d'une soirée, ADR 0033 — habitude de l'Emprise).
type porteeUsage struct {
	filtered        []legacymatch.SynthesisMatchRow
	squadRows       []domain.SquadMatchRow
	timelineRows    []domain.SquadMatchRow
	mainTeamByMatch map[string]map[string]struct{}
	history         []domain.SquadMatchHistoryRow
	// pairNames : match_id -> pair_name BRUT, lu sur les lignes canoniques du joueur. LA source
	// unique du mode écarté (drapeau neutre, D6) pour le fil de la session ET l'historique : la
	// fin du fil doit tomber sur le point « ce soir ».
	pairNames           map[string]string
	compositionSessions []domain.CompositionSessionEntry
}

// blocsUsage — les trois blocs publiés.
type blocsUsage struct {
	formes   *domain.SquadFormesBlock
	objectif *domain.SquadObjectiveHistory
	emprise  *domain.SquadEmpriseBlock
}

// loadUsageBlocks publie les blocs du résumé d'usage — « formes retenues » et l'Emprise — sur
// le MÊME périmètre D2 : leurs trois lectures communes (films, joueurs, participants) sont
// faites une fois (D2.6, lot perf L2). Puis l'historique d'objectif, sur le même périmètre
// comme soirée affichée. Chaque étape est sautée dès que la requête est annulée (D2.7) :
// GetPage rend alors l'erreur.
func (s *TeammatesService) loadUsageBlocks(
	ctx context.Context, playerXUID string, p porteeUsage, req domain.TeammatesQueryRequest,
) blocsUsage {
	selection := len(req.SelectedGamertags) > 0
	scope := perimetreEscouade(p.filtered, p.squadRows, selection)
	var lectures *squadagg.LecturesUsage
	var out blocsUsage
	siVivante(ctx, func() { lectures = s.lireUsagePartage(ctx, playerXUID, scope) })
	siVivante(ctx, func() { out.formes = s.loadSquadFormes(ctx, playerXUID, scope, p, req, lectures) })
	if selection {
		siVivante(ctx, func() {
			out.objectif = s.loadObjectiveHistory(ctx, lignesDuPerimetre(p.squadRows, scope), p.timelineRows, p.mainTeamByMatch, p.pairNames)
		})
	}
	// L'Emprise (lot L4) : ses propres lectures, le même périmètre (teammates_service_emprise.go).
	siVivante(ctx, func() { out.emprise = s.loadEmprise(ctx, playerXUID, p, req, perimetreLu{scope, lectures}) })
	// « Groupés ou isolés » (plan Emprise vies, lot V3) : le même périmètre, la composition des
	// fiches de l'Emprise ; sa propre lecture et sa propre section (teammates_service_emprise_placement.go).
	siVivante(ctx, func() { s.attacherPlacement(ctx, out.emprise, scope) })
	return out
}

// perimetreEscouade — le périmètre D2 : filteredMatches restreint aux matchs de la population
// escouade (allSquadRows, déjà passée par filterExactComposition sous l'option), dans l'ordre
// de filteredMatches. Sans sélection : filteredMatches tel quel.
func perimetreEscouade(
	filtered []legacymatch.SynthesisMatchRow, squadRows []domain.SquadMatchRow, selection bool,
) []legacymatch.SynthesisMatchRow {
	if !selection {
		return filtered
	}
	garde := make(map[string]struct{}, len(squadRows))
	for _, r := range squadRows {
		garde[r.MatchID] = struct{}{}
	}
	out := make([]legacymatch.SynthesisMatchRow, 0, len(squadRows))
	for _, m := range filtered {
		if _, ok := garde[m.MatchID]; ok {
			out = append(out, m)
		}
	}
	return out
}

// lignesDuPerimetre — les lignes escouade des matchs du périmètre (une par match).
func lignesDuPerimetre(squadRows []domain.SquadMatchRow, scope []legacymatch.SynthesisMatchRow) []domain.SquadMatchRow {
	dans := make(map[string]bool, len(scope))
	for _, m := range scope {
		dans[m.MatchID] = true
	}
	out := make([]domain.SquadMatchRow, 0, len(scope))
	for _, r := range squadRows {
		if dans[r.MatchID] {
			dans[r.MatchID] = false
			out = append(out, r)
		}
	}
	return out
}

// lireUsagePartage fait les trois lectures communes, sous la section `usage_shared`, quand au
// moins un des deux blocs les ferait : scope non vide, joueur connu, un résumé d'usage câblé.
// Sinon nil, et chaque bloc garde sa dégradation (scope vide ⇒ nil, source absente ⇒
// indisponible). Les deux sources sont le MÊME lecteur (duckdb.SessionUsageRepo, câblé deux
// fois sur la base du joueur) : la lecture passe par celle du résumé d'usage.
func (s *TeammatesService) lireUsagePartage(
	ctx context.Context, playerXUID string, filteredMatches []legacymatch.SynthesisMatchRow,
) *squadagg.LecturesUsage {
	repo := s.sessionUsageRepo
	if repo == nil && s.formesUsageRepo != nil {
		repo = s.formesUsageRepo
	}
	if repo == nil || playerXUID == "" || len(filteredMatches) == 0 {
		return nil
	}
	defer timing.FromContext(ctx).Section("usage_shared")()
	return squadagg.LireUsage(ctx, repo, teammatesMatchIDs(filteredMatches))
}

// pairNamesOf — match_id -> pair_name brut : celui des lignes canoniques du joueur d'abord,
// celui des lignes escouade (même colonne du registre) pour les matchs qui n'en portent pas.
func pairNamesOf(rows []canonical.PlayerMatchRow, squadRows []domain.SquadMatchRow) map[string]string {
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Enrichment.PairName != nil && *r.Enrichment.PairName != "" {
			out[r.Summary.MatchID] = *r.Enrichment.PairName
		}
	}
	for _, r := range squadRows {
		if _, ok := out[r.MatchID]; !ok && r.PairName != "" {
			out[r.MatchID] = r.PairName
		}
	}
	return out
}

// teammatesMatchIDs — les identifiants d'un scope de SynthesisMatchRow, dans
// l'ordre. Miroir de synthesisMatchIDs (service/synthesis_service_usage.go) :
// packages disjoints (K3b), même littéral de boucle qu'un seul champ — pas de
// troisième copie à ce jour côté teammates (règle CLAUDE.md n°6, 2 copies).
func teammatesMatchIDs(rows []legacymatch.SynthesisMatchRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.MatchID)
	}
	return out
}
