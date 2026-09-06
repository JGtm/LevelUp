/**
 * matchSides — QUI EST DE QUEL CÔTÉ, lu une seule fois dans le tableau de score.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, K2). Deux lectures du tableau de score
 * étaient réécrites d'un calque à l'autre : « quelle est MON équipe » (trois copies, plus la
 * canonique qui vivait dans le module des SONS d'objectif — un foyer que personne ne va
 * chercher pour peindre une onde) et « l'équipe de chaque xuid » (deux copies byte-identiques).
 * Une divergence entre deux de ces lectures ne se voit pas : elle donne la couleur de l'ennemi
 * à un allié, sur un calque, sans qu'aucun type ni aucun test ne s'en aperçoive.
 *
 * LES DEUX RÈGLES, ET CE QU'ELLES REFUSENT DE DEVINER :
 *
 *   - MON équipe est celle de la ligne marquée `is_me`. Pas de ligne « moi », ou un `team_side`
 *     qui ne se parse pas : `null`. Les appelants s'en servent pour SE TAIRE (encre neutre,
 *     son muet), jamais pour prendre l'équipe 0 par défaut.
 *   - L'ÉQUIPE D'UN XUID vient de sa propre ligne. Un joueur absent du tableau — un remplaçant
 *     que le pont n'a pas nommé, un acteur hors scoreboard — n'entre PAS dans la table : le
 *     lecteur reçoit `undefined` et rend le neutre, jamais une équipe devinée.
 *
 * Pas de React : deux fonctions pures, mémoïsées par leurs appelants sur `scoreboard`.
 */
import { parseTeamSideID } from '@/lib/halo/teamNames'

/** Ce que ces lectures demandent d'une ligne de tableau de score, et rien de plus. */
export interface ScoreboardSide {
  xuid: string
  team_side?: string | null
  is_me?: boolean | null
}

/**
 * allyTeamFromScoreboard — L'IDENTIFIANT D'ÉQUIPE ALLIÉE, ou `null` s'il n'est pas résolu.
 *
 * `null` N'EST PAS « ÉQUIPE 0 » : c'est l'absence de camp de référence. Un calque qui le
 * reçoit doit se taire (encre neutre), pas choisir.
 */
export function allyTeamFromScoreboard(
  scoreboard: readonly ScoreboardSide[] | null | undefined,
): number | null {
  return parseTeamSideID(scoreboard?.find((r) => r.is_me)?.team_side ?? null)
}

/**
 * teamOfXuidFromScoreboard — la table `xuid -> équipe`, pour les calques qui colorent un ACTEUR.
 *
 * LES LIGNES SANS CAMP RÉSOLU N'Y ENTRENT PAS : une entrée `undefined` et une entrée absente
 * disent la même chose au lecteur (« je ne sais pas »), et n'en garder qu'une seule forme
 * évite d'avoir à tester les deux.
 */
export function teamOfXuidFromScoreboard(
  scoreboard: readonly ScoreboardSide[] | null | undefined,
): Map<string, number> {
  const map = new Map<string, number>()
  for (const r of scoreboard ?? []) {
    const team = parseTeamSideID(r.team_side ?? null)
    if (team !== null) map.set(r.xuid, team)
  }
  return map
}
