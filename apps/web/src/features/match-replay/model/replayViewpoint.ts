/**
 * replayViewpoint — DE QUEL JOUEUR REGARDE-T-ON CE MATCH ? Un seul endroit répond.
 *
 * # POURQUOI CE MODULE EXISTE (2026-09-06, lot L2b du plan « frise, point de vue »)
 *
 * La page de rejeu a toujours eu un point de vue : « le joueur de la page », c'est-à-dire la
 * ligne `is_me` du tableau de score. Mais cette notion n'était écrite NULLE PART — elle était
 * REDÉCOUVERTE à cinq endroits qui relisaient `is_me` chacun de leur côté (les marques
 * d'identité, le camp de référence des calques, la lecture de fin de match, et deux sections
 * de la page match). Tant que la réponse est « toujours la même », cinq lectures parallèles ne
 * divergent pas. Dès qu'on veut REGARDER LE MATCH PAR LES YEUX D'UN AUTRE (le menu de joueurs
 * du lot suivant), elles divergent toutes en silence : la carte suivrait le joueur choisi
 * pendant que les couleurs des calques resteraient sur le joueur de la page, et rien — ni type,
 * ni test — ne s'en apercevrait.
 *
 * Le point de vue est donc désormais UNE VALEUR : un xuid, résolu ici, passé en paramètre à
 * tout ce qui en dépend. Le garde-rail `test/noIsMeOutsideViewpoint.guard.test.ts` interdit
 * qu'un sixième lecteur de `is_me` réapparaisse.
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne choisit pas : l'état de sélection vit dans `hooks/useReplayViewpoint.ts` (le React est
 * là-bas, la règle est ici). Il ne connaît ni le temps, ni la lecture — changer de point de vue
 * ne déplace jamais le curseur (décision 1 du plan).
 *
 * LE REPLI N'EST PAS RECOPIÉ : « le joueur de la page » se lit par `meXUIDOf`, le foyer que le
 * dépôt a déjà pour cette question. Ce module n'introduit donc AUCUNE nouvelle lecture de
 * `is_me` — c'était l'objet même du lot.
 */
import { meXUIDOf } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Ce que la résolution demande d'une ligne de tableau de score, et rien de plus. */
export type ViewpointRow = Pick<MatchScoreboardRow, 'xuid' | 'is_me'>

/**
 * resolveViewpoint rend le xuid par les yeux duquel la page se regarde.
 *
 * TROIS RÉPONSES, DANS CET ORDRE :
 *   1. le joueur SÉLECTIONNÉ, s'il a bien une ligne au tableau de score ;
 *   2. sinon le joueur DE LA PAGE (`meXUIDOf`) — le défaut, à chaque montage (décision 8) ;
 *   3. sinon `null`, et les lecteurs se taisent plutôt que de deviner un camp.
 *
 * POURQUOI LA SÉLECTION EST VÉRIFIÉE CONTRE LE TABLEAU DE SCORE. Un xuid choisi puis disparu
 * (le tableau se recharge, la sélection survit au re-rendu) ne désigne plus personne : sans
 * cette vérification, le point de vue vaudrait un joueur inexistant et TOUT deviendrait neutre
 * — aucun camp allié, aucune marque, aucun écran de fin — sans qu'aucune erreur ne le dise. Le
 * repli sur le joueur de la page est la seule dégradation qui reste lisible.
 */
export function resolveViewpoint(
  scoreboard: readonly ViewpointRow[] | null | undefined,
  selected: string | null,
): string | null {
  const sb = scoreboard ?? []
  if (selected && sb.some((r) => r.xuid === selected)) return selected
  return meXUIDOf(sb)
}
