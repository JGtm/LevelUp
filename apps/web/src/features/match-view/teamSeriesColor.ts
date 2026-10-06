/**
 * teamSeriesColor.ts — L'ENCRE D'UNE SÉRIE D'ÉQUIPE DANS UN GRAPHE, et il n'y en a qu'une.
 *
 * UNE SEULE FAMILLE DE COULEUR D'ÉQUIPE : les TOKENS `team-ally` / `team-enemy`, donc la
 * palette d'accessibilité que l'utilisateur a réglée (décision D1 du plan d'habillage) — en
 * graphe (`teamSeriesColor`, par `resolveToken`) comme en DOM (`teamTokenCssVar`) : tableau des
 * scores, en-têtes de camp, fil des éliminations, bandeau, frise et carte du rejeu. La couleur
 * OFFICIELLE du jeu par `team_id` (Eagle bleu, Cobra rouge) ne peint aucun camp : sur Halo
 * Infinite le `team_id` est TOUJOURS présent, et une cascade qui la plaçait avant le token
 * n'atteignait jamais le réglage de l'utilisateur — un joueur qui a réglé ses camps en vert et
 * orange voyait du bleu et du rouge.
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
import { resolveToken, tokenCssVar } from '@/lib/accessibility'

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

/**
 * L'ENCRE NEUTRE D'UN CAMP INCONNU EN DOM. Variable de thème, jamais un jeton d'équipe :
 * affirmer un camp qu'on n'a pas mesuré serait pire que de ne rien dire (même règle que le
 * repli `tc.axisLabel` de `teamSeriesColor`).
 */
const NEUTRAL_TEAM_COLOR = 'var(--muted-foreground)'

/**
 * teamTokenCssVar — LE PENDANT DOM de `teamSeriesColor`, et il obéit à la MÊME frontière.
 *
 * Les graphes en DOM/CSS (barres de la grille des usages, bâtons du contrôle des socles,
 * face-à-face des objectifs) ne passent pas par `resolveToken` : ils écrivent la VARIABLE CSS
 * du jeton, qui suit la palette sans re-rendu. La règle de choix, elle, est la même — jetons
 * `team-ally` / `team-enemy`, donc la palette d'accessibilité réglée par l'utilisateur, jamais
 * la cascade d'IDENTITÉ de `teamColor.ts` (qui place la couleur officielle du jeu devant et
 * n'atteindrait jamais le réglage sur Halo Infinite, où `team_id` est toujours présent).
 *
 * `ally === null` = camp INCONNU (aucun `is_me` au tableau des scores, ou joueur du film sans
 * ligne de scoreboard) : encre neutre du thème.
 */
export function teamTokenCssVar(ally: boolean | null): string {
  if (ally === null) return NEUTRAL_TEAM_COLOR
  return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
}
