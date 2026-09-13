/**
 * shared.ts — LES TROIS ASSEMBLAGES QUE PLUSIEURS CARTES PARTAGENT : les
 * colonnes de la bande (un match = une heure et une carte), une ligne de piste
 * du lobby (escouade + reste du camp + adversaire), et les segments de
 * l'escouade seule.
 *
 * Factorisés dès la deuxième copie : la piste du lobby est employée par trois
 * cartes (équipement, socles, objectifs) et la moindre divergence sur l'ordre
 * des segments ferait trois lectures différentes du même camp.
 */
import { TEAM_REST_INK } from '../colors'
import { lobbyOf } from '../model/access'
import type { PisteRow, PisteSegment } from '../forms/Piste100Form'
import type { LobbyParts } from '../model/aggregates'
import type { FormesViewModel } from '../viewModel'

/** Les colonnes de la bande : un match, son heure, sa carte. */
export function matchColumns(vm: FormesViewModel): { key: string; time: string; map: string }[] {
  return vm.matches.map((m) => ({
    key: m.match_id,
    time: vm.matchTime(m),
    map: vm.matchMap(m),
  }))
}

/**
 * Les segments de l'ESCOUADE SEULE, dans l'ordre d'affichage (le joueur de la
 * page en tête). Sert aux pistes dont le dénominateur n'est pas le lobby.
 */
export function squadSegments(
  vm: FormesViewModel,
  value: (xuid: string) => number,
): PisteSegment[] {
  return vm.squad.map((s) => ({ key: s.xuid, label: s.label, value: value(s.xuid), ink: s.ink }))
}

/**
 * Une ligne de PISTE DU LOBBY : l'escouade nommée, le reste de mon camp, puis
 * l'adversaire — compté, jamais nommé, toujours en dernier (la barre se lit de
 * gauche à droite du plus proche au plus lointain).
 */
export function lobbyTrackRow(
  vm: FormesViewModel,
  key: string,
  label: string,
  parts: LobbyParts,
  sublabel?: string,
): PisteRow {
  const segments: PisteSegment[] = vm.squad.map((s) => ({
    key: `${key}-${s.xuid}`,
    label: s.label,
    value: parts.bySquad[s.xuid] ?? 0,
    ink: s.ink,
  }))
  segments.push({
    key: `${key}-rest`,
    label: vm.t.common.teamRest,
    value: parts.teamRest,
    ink: TEAM_REST_INK,
  })
  segments.push({
    key: `${key}-enemy`,
    label: vm.t.common.enemyTeam,
    value: parts.opponents,
    hatch: true,
  })
  return { key, label, sublabel, segments }
}

/**
 * Le nombre de JOUEURS DISTINCTS mesurés sur la période — le dénominateur du
 * repère « lobbies observés ». Un joueur croisé sur trois matchs compte une
 * fois : c'est une population, pas un nombre de places.
 */
export function measuredPlayersCount(vm: FormesViewModel): number {
  const seen = new Set<string>()
  for (const m of vm.measured) for (const p of lobbyOf(m)) seen.add(p.xuid)
  return seen.size
}
