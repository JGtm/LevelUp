/**
 * playerMarks.ts — QUI EST « MOI », QUI EST UN AMI, sur la page de rejeu. Pur, testable.
 *
 * DEUX MARQUES, UNE GRAMMAIRE SUR LES TROIS PANNEAUX (carte, fiches, fil — décision D5 du
 * plan d'habillage, 2026-08-16) :
 *   - `me`     : le joueur DE LA PAGE (`is_me` du scoreboard) — pas le compte connecté ;
 *   - `friend` : un gamertag de `settings.friend_gamertags` du COMPTE CONNECTÉ, apparié
 *                par la même clé que les charts de la Match View (`normalizeGamertagKey`).
 * Le joueur de la page n'est jamais marqué ami de lui-même. Un ami ADVERSE est marqué
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
 */
export function buildPlayerMarks(
  scoreboard: readonly MatchScoreboardRow[],
  friendGamertags: readonly string[],
): ReadonlyMap<string, PlayerMarkKind> {
  const friends = new Set<string>()
  for (const gt of friendGamertags) {
    const key = normalizeGamertagKey(gt)
    if (key) friends.add(key)
  }
  const marks = new Map<string, PlayerMarkKind>()
  for (const row of scoreboard) {
    if (row.is_me) {
      marks.set(row.xuid, 'me')
      continue
    }
    if (friends.has(normalizeGamertagKey(row.gamertag))) marks.set(row.xuid, 'friend')
  }
  return marks
}
