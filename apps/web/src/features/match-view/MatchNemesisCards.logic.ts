/**
 * MatchNemesisCards.logic — némésis et souffre-douleur du joueur de la page du match.
 *
 * Calculés par la fonction commune `lib/antagonists` (celle du rejeu) sur les éliminations du
 * fil `combat_tab.highlight_events`, au dernier instant : la dernière image du rejeu et les
 * cartes de la page ne peuvent pas diverger. Le fil peut porter une même élimination en
 * double : la fonction commune écarte les doublons (même tueur, même victime, même instant).
 *
 * Règles de lecture du fil :
 *  - seule une élimination (`kill`) qui nomme son tueur ET sa victime par un xuid est comptée ;
 *  - un bot n'a pas de xuid : il n'est jamais un antagoniste (ni némésis ni souffre-douleur) ;
 *  - est adversaire tout autre joueur que le joueur de la page qui n'est pas de son camp
 *    (`team_side` du tableau de score) ; sans camp connu pour le joueur, tout autre joueur ;
 *    un joueur absent du tableau de score n'est pas du camp du joueur.
 */
import { buildAntagonists, type AntagonistKill } from '@/lib/antagonists/antagonists'
import type { MatchHighlightEvent, MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

export interface MatchAntagonist {
  xuid: string
  gamertag: string
  count: number
}

export interface MatchAntagonistsReading {
  nemesis: MatchAntagonist | null
  bully: MatchAntagonist | null
}

/** Éliminations du fil dont le tueur et la victime ont un xuid, avec le nom que porte le fil. */
function collectKills(events: readonly MatchHighlightEvent[]): {
  kills: AntagonistKill[]
  nameByXuid: Map<string, string>
} {
  const kills: AntagonistKill[] = []
  const nameByXuid = new Map<string, string>()
  for (const e of events) {
    if ((e.event_type ?? '').toLowerCase() !== 'kill') continue
    if (!e.actor_xuid || !e.victim_xuid || e.event_time_ms == null) continue
    kills.push({ killer: e.actor_xuid, victim: e.victim_xuid, t: e.event_time_ms })
    if (e.actor_gamertag && !nameByXuid.has(e.actor_xuid)) nameByXuid.set(e.actor_xuid, e.actor_gamertag)
    if (e.victim_gamertag && !nameByXuid.has(e.victim_xuid)) nameByXuid.set(e.victim_xuid, e.victim_gamertag)
  }
  return { kills, nameByXuid }
}

export function readMatchAntagonists(
  events: readonly MatchHighlightEvent[],
  scoreboard: readonly MatchScoreboardRow[],
  meXUID: string | null,
): MatchAntagonistsReading {
  if (!meXUID) return { nemesis: null, bully: null }

  const sideByXuid = new Map<string, string | null>()
  const nameByXuid = new Map<string, string>()
  for (const r of scoreboard) {
    sideByXuid.set(r.xuid, r.team_side ?? null)
    if (r.gamertag) nameByXuid.set(r.xuid, r.gamertag)
  }
  const allyTeam = sideByXuid.get(meXUID) ?? null
  const isFoe = (xuid: string): boolean => xuid !== meXUID && (allyTeam == null || sideByXuid.get(xuid) !== allyTeam)

  const fil = collectKills(events)
  const at = buildAntagonists(fil.kills, meXUID, isFoe).at(Number.POSITIVE_INFINITY)
  const named = (xuid: string, count: number): MatchAntagonist => ({
    xuid,
    gamertag: displayPlayerName(nameByXuid.get(xuid) ?? fil.nameByXuid.get(xuid), xuid),
    count,
  })
  return {
    nemesis: at.nemesis ? named(at.nemesis.holder, at.nemesis.count) : null,
    bully: at.victim ? named(at.victim.holder, at.victim.count) : null,
  }
}
