/**
 * playerMarks.ts — QUI EST « MOI », QUI EST UN AMI, sur la page de rejeu. Pur, testable.
 *
 * DEUX MARQUES, UNE GRAMMAIRE SUR LES TROIS PANNEAUX (carte, fiches, fil — décision D5 du
 * plan d'habillage, 2026-08-16) :
 *   - `me`     : LE POINT DE VUE de la page de rejeu, qui vaut le joueur de la page par
 *                défaut (décision 13 du plan « frise, point de vue », 2026-09-06) — pas le
 *                compte connecté. Le disque cerclé de la carte dit « celui qu'on regarde » ;
 *                le jour où l'on regarde par les yeux d'un autre, il le suit ;
 *   - `friend` : un gamertag de `settings.friend_gamertags` du COMPTE CONNECTÉ, apparié
 *                par la même clé que les charts de la Match View (`normalizeGamertagKey`).
 * Celui qu'on regarde n'est jamais marqué ami de lui-même. Un ami ADVERSE est marqué
 * aussi : la marque dit l'identité, pas le camp — le camp, c'est la couleur.
 *
 * Aucune marque pour un xuid absent du scoreboard : sans ligne, on ne sait rien de lui.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { normalizeGamertagKey } from '@/lib/players/displayName'

export type PlayerMarkKind = 'me' | 'friend'

/** Référence STABLE pour « aucune marque » : évite un `new Map()` à chaque rendu. */
export const NO_MARKS: ReadonlyMap<string, PlayerMarkKind> = new Map()

/**
 * buildPlayerMarks rend, par xuid, la marque à porter. `friendGamertags` = la liste des
 * réglages, telle quelle (la normalisation est faite ici).
 *
 * `meXUID` (2026-09-06) désigne QUI porte la marque `me` : le point de vue de la page de
 * rejeu. Absent, la marque retombe sur la ligne `is_me` du tableau de score — le comportement
 * d'avant le point de vue, que la caractérisation L2a fixe et que les appelants qui ne
 * connaissent pas de point de vue continuent d'obtenir. Un `meXUID` qui n'a pas de ligne au
 * tableau de score ne fait porter la marque à PERSONNE : pas de repli sur `is_me`, sinon le
 * disque cerclé désignerait quelqu'un d'autre que ce qu'on regarde.
 */
export function buildPlayerMarks(
  scoreboard: readonly MatchScoreboardRow[],
  friendGamertags: readonly string[],
  meXUID?: string | null,
): ReadonlyMap<string, PlayerMarkKind> {
  const friends = new Set<string>()
  for (const gt of friendGamertags) {
    const key = normalizeGamertagKey(gt)
    if (key) friends.add(key)
  }
  const marks = new Map<string, PlayerMarkKind>()
  for (const row of scoreboard) {
    if (meXUID == null ? row.is_me : row.xuid === meXUID) {
      marks.set(row.xuid, 'me')
      continue
    }
    if (friends.has(normalizeGamertagKey(row.gamertag))) marks.set(row.xuid, 'friend')
  }
  return marks
}
