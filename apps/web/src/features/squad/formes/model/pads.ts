/**
 * pads.ts — LE CONTRÔLE DES ARMES SPÉCIALES (artefact 2ec1b8eb, bloc 2).
 *
 * LE VOCABULAIRE, ET IL COMPTE. Un SOCLE est l'emplacement fixe d'une carte où
 * une arme de puissance réapparaît. Une PRISE DE SOCLE est un ramassage sur cet
 * emplacement, lu dans l'événement natif du film : daté à la milliseconde, il
 * porte son ramasseur. Une OCCUPATION est un socle vidé — avec ou sans
 * ramasseur nommé.
 *
 * LE DÉNOMINATEUR N'EST PAS LE NOMBRE DE SOCLES : c'est le nombre de prises
 * NOMMÉES. Les occupations sans ramasseur n'entrent dans aucun camp, et la
 * réserve reste à l'écran (`unnamedOccupations`) — sans elle, la barre se lirait
 * comme la totalité des socles.
 *
 * LES TROIS FAMILLES D'ARME (lourde / précision / autre) arrivent DÉJÀ RANGÉES
 * du serveur, qui les dérive du registre canonique d'armes. Rien n'est classé
 * ici : une table d'armes écrite côté web serait une seconde vérité.
 */
import type { SquadFormesBlock, SquadFormesMatch, SquadFormesWeapon } from '@/lib/api/types'

import { isMySide, lobbyOf, matchSizes, measuredMatches, parityOf, sharePct, average } from './access'

/** Les trois familles d'arme de socle, dans l'ordre d'affichage de l'artefact. */
export const WEAPON_CLASSES = ['heavy', 'precision', 'other'] as const

export type WeaponClass = (typeof WEAPON_CLASSES)[number]

/** Clé de famille d'arme -> ce que le titre en sait. */
export function weaponIndex(block: SquadFormesBlock): Record<string, SquadFormesWeapon> {
  const out: Record<string, SquadFormesWeapon> = {}
  for (const w of block.weapons ?? []) out[w.key] = w
  return out
}

/** Les occupations d'un socle par arme, sur un match. */
function padsOf(match: SquadFormesMatch): { weapon: string; occupations: number; named: number }[] {
  return match.weapon_pads ?? []
}

/** Le total d'occupations d'une arme sur les matchs mesurés. */
export function weaponOccupations(block: SquadFormesBlock, key: string): number {
  let n = 0
  for (const m of measuredMatches(block)) {
    for (const p of padsOf(m)) if (p.weapon === key) n += p.occupations
  }
  return n
}

/** Le total de prises NOMMÉES d'une arme (tous joueurs). */
export function weaponNamed(block: SquadFormesBlock, key: string): number {
  let n = 0
  for (const m of measuredMatches(block)) {
    for (const p of padsOf(m)) if (p.weapon === key) n += p.named
  }
  return n
}

/** Le nombre de matchs mesurés où l'arme avait au moins un socle. */
export function matchesWithWeapon(block: SquadFormesBlock, key: string): number {
  return measuredMatches(block).filter((m) => padsOf(m).some((p) => p.weapon === key)).length
}

/** Les prises d'une arme par un joueur, sur les matchs mesurés. */
export function weaponPickupsByPlayer(
  block: SquadFormesBlock,
  key: string,
  xuid: string,
): number {
  let n = 0
  for (const m of measuredMatches(block)) {
    for (const p of lobbyOf(m)) {
      if (p.xuid === xuid) n += p.pads_by_weapon?.[key] ?? 0
    }
  }
  return n
}

/**
 * Les armes du scope, TRIÉES PAR VOLUME DE PRISES NOMMÉES (décroissant), la clé
 * départageant à volume égal — un ordre d'affichage stable d'une requête à
 * l'autre.
 */
export function weaponsByVolume(block: SquadFormesBlock): SquadFormesWeapon[] {
  const named = new Map<string, number>()
  for (const w of block.weapons ?? []) named.set(w.key, weaponNamed(block, w.key))
  return [...(block.weapons ?? [])].sort((a, b) => {
    const d = (named.get(b.key) ?? 0) - (named.get(a.key) ?? 0)
    return d !== 0 ? d : a.key.localeCompare(b.key)
  })
}

/** Les occupations de socle SANS ramasseur nommé, sur les matchs mesurés. */
export function unnamedOccupations(block: SquadFormesBlock): number {
  return measuredMatches(block).reduce((a, m) => a + (m.pad_unnamed ?? 0), 0)
}

/** Les prises NOMMÉES du scope — le dénominateur de toutes les parts du bloc 2. */
export function namedPickups(block: SquadFormesBlock): number {
  return measuredMatches(block).reduce((a, m) => a + (m.pad_named ?? 0), 0)
}

/** L'agrégat d'une famille d'arme : ma part et celle de mon camp dans le lobby. */
export interface WeaponClassAggregate {
  me: number
  team: number
  lobby: number
  myShareOfLobbyPct: number | null
  teamShareOfLobbyPct: number | null
  teamParity: number | null
  lobbyParity: number | null
}

/**
 * agregeFamille — les prises de socle d'une famille d'arme, sur les matchs
 * mesurés. La parité du lobby est celle des prises de socle en général (« 1
 * joueur sur 8,4 ») : une famille n'a pas sa propre parité, elle a la mienne.
 */
export function aggregateWeaponClass(
  block: SquadFormesBlock,
  weapons: Record<string, SquadFormesWeapon>,
  weaponClass: WeaponClass,
): WeaponClassAggregate {
  const main = block.main_xuid ?? ''
  let me = 0
  let team = 0
  let lobby = 0
  const teamSizes: number[] = []
  const lobbySizes: number[] = []
  for (const match of measuredMatches(block)) {
    const sizes = matchSizes(match)
    teamSizes.push(sizes.team)
    lobbySizes.push(sizes.lobby)
    for (const p of lobbyOf(match)) {
      let v = 0
      for (const [key, count] of Object.entries(p.pads_by_weapon ?? {})) {
        if ((weapons[key]?.class ?? 'other') === weaponClass) v += count
      }
      lobby += v
      if (isMySide(match, p)) team += v
      if (p.xuid === main) me += v
    }
  }
  return {
    me,
    team,
    lobby,
    myShareOfLobbyPct: sharePct(me, lobby),
    teamShareOfLobbyPct: sharePct(team, lobby),
    teamParity: parityOf(average(teamSizes) ?? 0),
    lobbyParity: parityOf(average(lobbySizes) ?? 0),
  }
}
