package analysis

// comeback_frag_contrast.go — SABORDAGE (6) / ABNÉGATION (7) : le score et les
// frags racontent deux matchs opposés, dans un mode à objectifs.
//
// Les deux badges sont les deux faces d'un même match : si l'équipe qui perd au
// score a dominé aux frags (son SABORDAGE), l'équipe qui gagne au score a été
// dominée aux frags (son ABNÉGATION).
//
// Critère mixte, mesuré le 2026-09-16 sur 349 matchs à objectifs Halo Infinite
// dotés d'une timeline de frags (base locale au 30/08) et choisi par
// l'utilisateur (« ni trop fréquents, mais de temps en temps ») :
//
//   - l'équipe dominante aux frags a MENÉ aux frags ≥ 75 % du temps de match ;
//   - elle finit avec les frags adverses ≤ 85 % des siens (soit ≥ 17,6 % de frags de plus) ;
//   - elle totalise au moins 10 frags (garde contre les matchs abrégés).
//
// Résultat de la mesure : 10 matchs sur 349 (≈ 1 sur 35). Les critères écartés :
// l'écart FINAL seul (perdant ≤ 40 % des frags → 0 match sur 349, les écarts de
// frags en mode à objectifs sont serrés) et l'écart ABSOLU en frags (en BTB, 7
// frags d'écart sur 160 ne sont pas une domination).

// FragContrastMinLeadShare : part minimale du temps de match passée en tête aux frags.
const FragContrastMinLeadShare = 0.75

// FragContrastMaxEnemyRatio : frags de l'équipe dominée / frags de l'équipe
// dominante, au plus (0.85 : le dominé fait au plus 85 % du dominant, soit au moins
// 17,6 % de frags de plus pour le dominant — ce n est PAS « 15 % de plus », mesure 2026-09-17).
const FragContrastMaxEnemyRatio = 0.85

// FragContrastMinWinnerFrags : volume minimal de frags de l'équipe dominante.
const FragContrastMinWinnerFrags = 10

// ComputeFragContrastDominance détecte SABORDAGE / ABNÉGATION du point de vue
// du joueur, depuis la timeline de frags DÉDOUBLONNÉE (triée par TimeMS ASC).
//
//   - matchEndMS    : fin du match en ms (durée) ; la dernière frag fait foi si
//     elle est postérieure ;
//   - playerTeamID  : 0 ou 1 ;
//   - playerOutcome : code Halo (2=Win, 3=Loss) ; Tie/DNF → 0.
//
// L'appelant garantit un mode à objectifs (en Slayer frags et score se confondent)
// et n'applique ce badge que si aucun autre ne s'applique.
func ComputeFragContrastDominance(events []KillEvent, matchEndMS int64, playerTeamID, playerOutcome int) int {
	if playerOutcome != OutcomeWin && playerOutcome != OutcomeLoss {
		return DominanceFlagNone
	}
	if len(events) == 0 || (playerTeamID != 0 && playerTeamID != 1) {
		return DominanceFlagNone
	}
	myFrags, enemyFrags, myLeadMS, enemyLeadMS, totalMS := fragTimeline(events, matchEndMS, playerTeamID)
	if totalMS <= 0 {
		return DominanceFlagNone
	}
	switch playerOutcome {
	case OutcomeLoss:
		if fragDominates(myFrags, enemyFrags, myLeadMS, totalMS) {
			return DominanceFlagSabordage
		}
	case OutcomeWin:
		if fragDominates(enemyFrags, myFrags, enemyLeadMS, totalMS) {
			return DominanceFlagAbnegation
		}
	}
	return DominanceFlagNone
}

// fragDominates : l'équipe « dom » a dominé l'équipe « sub » aux frags au sens
// du critère mixte (temps en tête, écart relatif final, volume).
func fragDominates(domFrags, subFrags int, domLeadMS, totalMS int64) bool {
	if domFrags < FragContrastMinWinnerFrags {
		return false
	}
	if float64(subFrags) > FragContrastMaxEnemyRatio*float64(domFrags) {
		return false
	}
	return float64(domLeadMS) >= FragContrastMinLeadShare*float64(totalMS)
}

// fragTimeline cumule les frags de chaque camp et le temps passé en tête par
// chacun. L'écart après une frag vaut jusqu'à la frag suivante (ou la fin du
// match) ; avant la première frag, égalité.
func fragTimeline(events []KillEvent, matchEndMS int64, playerTeamID int) (myFrags, enemyFrags int, myLeadMS, enemyLeadMS, totalMS int64) {
	totalMS = max(matchEndMS, events[len(events)-1].TimeMS)
	for i, e := range events {
		if e.TeamID == playerTeamID {
			myFrags++
		} else {
			enemyFrags++
		}
		until := totalMS
		if i+1 < len(events) {
			until = events[i+1].TimeMS
		}
		span := max(until-e.TimeMS, 0)
		switch {
		case myFrags > enemyFrags:
			myLeadMS += span
		case enemyFrags > myFrags:
			enemyLeadMS += span
		}
	}
	return myFrags, enemyFrags, myLeadMS, enemyLeadMS, totalMS
}
