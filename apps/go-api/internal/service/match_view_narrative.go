// Package service — match_view_narrative.go : helpers Phase 1 méta-plan
// § 6.1.3 pour brancher les fondations `analysis/narrative` sur le service
// MatchView (chunk MV2).
//
// Couvre :
//   - Cadence intra-match (kills par phase de 60s) via narrative.ComputeCadenceProfiles
//   - Impact 8 rôles via narrative.IdentifyImpactRoles
//
// Les helpers consomment les `domain.EventRaw` chargés par le repo
// existant (Q21) en les convertissant à la volée vers `canonical.HighlightEvent`.
// Pas de modification du repo nécessaire — le contrat raw reste valide.
package service

import (
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// MatchCadencePhaseSeconds est la phase par défaut de la cadence MatchView.
// Aligné sur TugOfWarBinSize (30 000 ms = 30s) pour que les tranches des deux
// graphes "Cadence des frags" et "Dominance par tranche" soient identiques.
// 60 secondes pour aligner avec le réglage UX Squad V2 (chunk S6).
const MatchCadencePhaseSeconds = 30

// convertEventsRawToCanonical convertit les events bruts (Q21) en
// canonical.HighlightEvent pour pouvoir les passer aux fonctions narrative.
//
// Les fonctions narrative (cadence, impact_roles) attendent des
// canonical.HighlightEvent avec MatchID, EventType, TimeMS, KillerXUID/
// VictimXUID/PlayerXUID selon le sens sémantique du type.
//
// EventRaw stocke un seul `xuid` dont le sens dépend du EventType (cf.
// HighlightEventsRepo) :
//
//	kill, first_kill, finisher, clutch -> xuid = tueur (KillerXUID)
//	death, first_death                 -> xuid = victime (VictimXUID)
//	medal, assist                      -> xuid = joueur (PlayerXUID)
func convertEventsRawToCanonical(events []domain.EventRaw, matchID string) []canonical.HighlightEvent {
	out := make([]canonical.HighlightEvent, 0, len(events))
	for _, ev := range events {
		var timeMS int64
		if ev.TimeMS != nil {
			timeMS = *ev.TimeMS
		}
		he := canonical.HighlightEvent{
			MatchID:   matchID,
			EventType: ev.EventType,
			TimeMS:    timeMS,
		}
		if ev.XUID != nil {
			xuid := *ev.XUID
			he.XUID = xuid
			switch canonical.HighlightEventType(ev.EventType) {
			case canonical.EventKill, canonical.EventFirstKill,
				canonical.EventFinisher, canonical.EventClutch:
				he.KillerXUID = &xuid
			case canonical.EventDeath, canonical.EventFirstDeath:
				he.VictimXUID = &xuid
			case canonical.EventMedal, canonical.EventAssist:
				he.PlayerXUID = &xuid
			}
		}
		out = append(out, he)
	}
	return out
}

// extractMatchSquadXUIDs extrait l'ensemble des xuids des joueurs présents
// au scoreboard du match. Sert de "squad" pour narrative.ComputeCadenceProfiles
// (cadence MatchView = tous les joueurs du match, pas seulement le main).
func extractMatchSquadXUIDs(scoreboard []domain.ScoreboardRaw) []string {
	out := make([]string, 0, len(scoreboard))
	seen := make(map[string]bool, len(scoreboard))
	for _, row := range scoreboard {
		if row.XUID == "" || seen[row.XUID] {
			continue
		}
		seen[row.XUID] = true
		out = append(out, row.XUID)
	}
	return out
}

// extractTeamOutcomesFromScoreboard mappe xuid -> Outcome depuis le scoreboard.
// Utilisé par narrative.IdentifyImpactRoles pour déterminer les rôles "win-only"
// (clutch_finisher, silent_hero) et "loss-only" (false_brother).
//
// OutcomeCode == 0 est traité comme indéfini (xuid omis du résultat).
func extractTeamOutcomesFromScoreboard(scoreboard []domain.ScoreboardRaw) map[string]canonical.Outcome {
	out := make(map[string]canonical.Outcome, len(scoreboard))
	for _, row := range scoreboard {
		if row.XUID == "" || row.OutcomeCode == 0 {
			continue
		}
		switch row.OutcomeCode {
		case 2:
			out[row.XUID] = canonical.OutcomeWin
		case 3:
			out[row.XUID] = canonical.OutcomeLoss
		case 1:
			out[row.XUID] = canonical.OutcomeTie
		case 4:
			out[row.XUID] = canonical.OutcomeDNF
		}
	}
	return out
}

// BuildMatchCadenceChart construit la cadence intra-match (kills par phase
// de 60s, 1 série par joueur du match). Output au format ChartPointStacked
// pour rendu via le wrapper `<BarStacked>` (S10) côté front.
//
// Si events est vide, retourne nil (pas de cadence à afficher).
//
// Variant principal pour les callers ayant des EventRaw (chunk MV2 legacy).
// Pour les callers ayant déjà des canonical.HighlightEvent (chunk MV4.A
// loader unifié), utiliser BuildMatchCadenceChartFromCanonical.
func BuildMatchCadenceChart(
	events []domain.EventRaw,
	scoreboard []domain.ScoreboardRaw,
	matchID string,
) *domain.ChartSeries[domain.ChartPointStacked] {
	if len(events) == 0 || len(scoreboard) == 0 {
		return nil
	}
	canonicalEvents := convertEventsRawToCanonical(events, matchID)
	return BuildMatchCadenceChartFromCanonical(canonicalEvents, scoreboard)
}

// BuildMatchCadenceChartFromCanonical : variante consommant directement des
// canonical.HighlightEvent (chunk MV4.A — loader unifié). Pas de conversion.
func BuildMatchCadenceChartFromCanonical(
	canonicalEvents []canonical.HighlightEvent,
	scoreboard []domain.ScoreboardRaw,
) *domain.ChartSeries[domain.ChartPointStacked] {
	if len(canonicalEvents) == 0 || len(scoreboard) == 0 {
		return nil
	}
	squadXUIDs := extractMatchSquadXUIDs(scoreboard)

	profiles := narrative.ComputeCadenceProfiles(canonicalEvents, squadXUIDs, MatchCadencePhaseSeconds)
	if len(profiles) == 0 {
		return nil
	}

	// Trouver le K max (longest match en buckets).
	maxBuckets := 0
	for _, p := range profiles {
		if len(p.Buckets) > maxBuckets {
			maxBuckets = len(p.Buckets)
		}
	}

	// Aggréger : 1 datapoint par phase, components = xuid -> kills.
	bucketAgg := make([]map[string]float64, maxBuckets)
	for i := range bucketAgg {
		bucketAgg[i] = make(map[string]float64)
		for _, x := range squadXUIDs {
			bucketAgg[i][x] = 0
		}
	}
	for _, p := range profiles {
		for i, count := range p.Buckets {
			if i >= maxBuckets {
				break
			}
			bucketAgg[i][p.XUID] += float64(count)
		}
	}

	dps := make([]domain.ChartPointStacked, 0, maxBuckets)
	for i, comp := range bucketAgg {
		dps = append(dps, domain.ChartPointStacked{
			Category:   matchPhaseCategoryLabel(i),
			Components: comp,
		})
	}
	return &domain.ChartSeries[domain.ChartPointStacked]{
		Key:        "match_view.combat.cadence",
		LabelKey:   "match_view.combat.cadence_title",
		Datapoints: dps,
		Meta: map[string]any{
			chartMetaPhaseSeconds: MatchCadencePhaseSeconds,
			"bucket_count":        maxBuckets,
		},
	}
}

// matchPhaseCategoryLabel formate un index de bucket en catégorie stable
// "phase_NN" zero-padded (cf. squad cadence pattern).
func matchPhaseCategoryLabel(idx int) string {
	const labels = "0123456789"
	if idx < 10 {
		return "phase_0" + string(labels[idx])
	}
	tens := idx / 10
	units := idx % 10
	if tens < 10 {
		return "phase_" + string(labels[tens]) + string(labels[units])
	}
	// idx >= 100 (très rare, match >100 phases = 100 minutes)
	return "phase_99+"
}

// BuildMatchImpactRoles8 construit la liste des 8 rôles narratifs attribués
// sur ce match via narrative.IdentifyImpactRoles. Renvoie 1 entrée par
// (joueur × rôle) attribué.
//
// Le squad ici est l'ensemble des xuids du scoreboard (vue MatchView = tous
// les joueurs du match, pas seulement le squad utilisateur).
//
// Variant principal pour les callers ayant des EventRaw (chunk MV2 legacy).
// Pour les callers ayant déjà des canonical.HighlightEvent, utiliser
// BuildMatchImpactRoles8FromCanonical.
func BuildMatchImpactRoles8(
	events []domain.EventRaw,
	scoreboard []domain.ScoreboardRaw,
	matchID string,
) []domain.MatchViewImpactRole {
	if len(events) == 0 || len(scoreboard) == 0 {
		return nil
	}
	canonicalEvents := convertEventsRawToCanonical(events, matchID)
	return BuildMatchImpactRoles8FromCanonical(canonicalEvents, scoreboard)
}

// BuildMatchImpactRoles8FromCanonical : variante consommant directement des
// canonical.HighlightEvent (chunk MV4.A — loader unifié).
func BuildMatchImpactRoles8FromCanonical(
	canonicalEvents []canonical.HighlightEvent,
	scoreboard []domain.ScoreboardRaw,
) []domain.MatchViewImpactRole {
	if len(canonicalEvents) == 0 || len(scoreboard) == 0 {
		return nil
	}
	teamOutcomes := extractTeamOutcomesFromScoreboard(scoreboard)
	squadXUIDs := extractMatchSquadXUIDs(scoreboard)

	assignments := narrative.IdentifyImpactRoles(canonicalEvents, teamOutcomes, squadXUIDs)
	if len(assignments) == 0 {
		return nil
	}
	out := make([]domain.MatchViewImpactRole, 0, len(assignments))
	for _, a := range assignments {
		out = append(out, domain.MatchViewImpactRole{
			XUID:       a.XUID,
			RoleKey:    string(a.Role),
			LabelKey:   a.LabelKey,
			ColorToken: a.ColorToken,
			Inverted:   a.Inverted,
		})
	}
	return out
}
