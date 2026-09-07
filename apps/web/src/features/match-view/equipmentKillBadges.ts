/**
 * equipmentKillBadges — LE BADGE « TEMPS FORT » : N frags dans le MÊME épisode d'état actif
 * (camouflage, surbouclier).
 *
 * PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F, sous-lot F.3. Décision utilisateur 8a/8b (DEC-7
 * révisée) : GO à petite population — camo 35,2 % (25/71 épisodes en lecture STRICTE),
 * surbouclier 55,6 % (10/18), voir F.0/F.1 pour le détail des marges.
 *
 * POURQUOI LE MEILLEUR ÉPISODE, ET NON LA SOMME DU JOUEUR. `equipmentUsageLogic.ts` cumule les
 * frags sur TOUS les épisodes d'un joueur (le tableau des usages) ; ce badge-ci cherche un
 * INSTANT : un même épisode continu portant beaucoup de frags. Trois frags répartis sur quatre
 * épisodes d'une minute chacun ne sont pas le même fait qu'un triple sous UN SEUL passage de
 * camouflage — seul le second mérite un badge narratif.
 *
 * LE SEUIL EST ÉCRIT D'AVANCE (mission F.3) et NE DOIT PAS ÊTRE AJUSTÉ À LA SORTIE : le maximum
 * observé sur le corpus F.0 (n=149 épisodes) est 10, les exemples réels commencent à 3.
 *
 * UN BADGE PAR FAMILLE AU PLUS, LE MEILLEUR ÉPISODE GAGNE : deux joueurs à 3 frags sous camo
 * n'ouvrent pas deux badges — la barre reste un résumé, pas un journal.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'
import {
  EPISODE_FAMILIES,
  type EquipmentEpisodeFamily,
} from '@/features/match-replay/model/equipmentUsageLogic'
import { buildPlayers, indexBySlot, playerName } from '@/lib/replay/rosterLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

/**
 * EQUIPMENT_KILL_BADGE_THRESHOLD — le seuil ÉCRIT D'AVANCE. NE PAS L'AJUSTER À LA SORTIE : la
 * mesure F.0 (corpus de 149 épisodes) fixe ce nombre avant tout regard sur le résultat produit.
 */
export const EQUIPMENT_KILL_BADGE_THRESHOLD = 3

/** Un badge « temps fort » : la famille, le nombre RÉEL de frags du meilleur épisode, et qui. */
export interface EquipmentKillBadge {
  family: EquipmentEpisodeFamily
  kills: number
  xuid: string
  playerName: string
}

/**
 * computeEquipmentKillBadges — le meilleur épisode par famille, s'il franchit le seuil.
 *
 * PUR : aucun réseau, aucun React. `doc` est l'artefact déjà normalisé (comme
 * `buildEquipmentUsage`) ; `scoreboard` peut manquer, auquel cas le joueur garde son nom de
 * film (même cascade que `equipmentUsageLogic`, via `displayPlayerName`).
 */
export function computeEquipmentKillBadges(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[] | undefined,
): EquipmentKillBadge[] {
  const players = buildPlayers(doc, scoreboard ?? [])
  const ownerOfSlot = indexBySlot(players, (p) => p)

  const best = new Map<EquipmentEpisodeFamily, { kills: number; slot: number }>()
  for (const e of doc.equipmentEpisodes) {
    if (!(EPISODE_FAMILIES as readonly string[]).includes(e.fam)) continue
    const fam = e.fam as EquipmentEpisodeFamily
    const kills = e.k ?? 0
    const cur = best.get(fam)
    if (!cur || kills > cur.kills) best.set(fam, { kills, slot: e.slot })
  }

  const out: EquipmentKillBadge[] = []
  for (const fam of EPISODE_FAMILIES) {
    const candidate = best.get(fam)
    if (!candidate || candidate.kills < EQUIPMENT_KILL_BADGE_THRESHOLD) continue
    const owner = ownerOfSlot.get(candidate.slot)
    // Aucun propriétaire mesuré pour ce slot (pont slot -> xuid incomplet) : pas de nom à
    // porter, pas de badge — même règle que le reste du rejeu, on ne devine jamais un joueur.
    if (!owner) continue
    out.push({
      family: fam,
      kills: candidate.kills,
      xuid: owner.xuid,
      playerName: displayPlayerName(playerName(owner), owner.xuid),
    })
  }
  return out
}
