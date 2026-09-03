/**
 * teamSeriesColor.ts — L'ENCRE D'UNE SÉRIE D'ÉQUIPE DANS UN GRAPHE, et il n'y en a qu'une.
 *
 * DEUX FAMILLES DE COULEUR D'ÉQUIPE COEXISTENT DANS LE DÉPÔT, ET LEUR DIFFÉRENCE EST DU SENS.
 *
 *   IDENTITÉ (`teamColor.ts`, `teamColorResolver`) — le tableau des scores, l'en-tête d'un
 *   camp, la section Objectifs, le fil des éliminations. La cascade y place la couleur
 *   OFFICIELLE du jeu par `team_id` (Eagle bleu, Cobra rouge) AVANT le token sémantique :
 *   c'est le camp tel que le JEU le peint, et c'est ce qu'on veut là où l'écran nomme les
 *   équipes.
 *
 *   GRAPHE (ce fichier) — les courbes et les barres. Elles prennent les TOKENS
 *   `team-ally` / `team-enemy`, donc la palette d'accessibilité que l'utilisateur a réglée
 *   (décision D1 du plan d'habillage, suivie par `ReplayScoreBanner`, `ReplayTeamHeader`,
 *   `ReplayTimelineTracks`, `ReplayCanvas` et `ExplorerMatchesTable`). Sur Halo Infinite le
 *   `team_id` est TOUJOURS présent : la cascade d'identité n'atteindrait jamais le token, et
 *   un joueur qui a réglé ses camps en vert et orange verrait quand même du bleu et du rouge
 *   sur ses graphes.
 *
 * POURQUOI CE FICHIER PLUTÔT QU'UNE FONCTION PRIVÉE. Elle l'était — dans
 * `MatchScoreCurveChart.tsx` — jusqu'à ce que le graphe des points marqués ait besoin de la
 * MÊME encre (2026-09-03). Deux camps peints par deux fonctions, ce sont deux couleurs pour
 * la même équipe sur deux blocs du même onglet (règle CLAUDE.md n°6, ≤ 2 copies).
 *
 * AUCUNE VALEUR EN DUR : `resolveToken` lit la palette courante, et l'appel doit se faire
 * DEPUIS `buildOption` — c'est le rebuild sur changement de palette qui rafraîchit la teinte.
 */
import type { EChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'

/**
 * teamSeriesColor rend l'encre d'une série d'équipe.
 *
 * `ally === null` = camp INCONNU (aucun joueur du scoreboard ne le rattache au joueur de la
 * page) : encre neutre du thème, jamais l'une des deux couleurs par défaut — affirmer un
 * camp qu'on n'a pas mesuré serait pire que de ne rien dire.
 */
export function teamSeriesColor(ally: boolean | null, tc: EChartsThemeColors): string {
  if (ally === null) return tc.axisLabel
  return resolveToken(ally ? 'team-ally' : 'team-enemy')
}
