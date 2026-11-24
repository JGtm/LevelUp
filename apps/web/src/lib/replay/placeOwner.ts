/**
 * placeOwner.ts — QUI EST LE JOUEUR QUE LE FILM DÉSIGNE PAR UNE PLACE, à une image. La table d'identité est le roster
 * du document : `roster[].filmIndex` est l'index de joueur du film, le même espace que `place` dans `reloads[]`,
 * `gestures[]`, et dans les éliminations et actions de score d'un bot (qui n'a pas d'identifiant). Aucune seconde clé.
 *
 * UN INDEX DÉSIGNE UN JOUEUR quand une seule identité du roster le porte (`rosterEntryKey` : le xuid, ou `bot:<nom>`).
 * Quand plusieurs identités le portent (un joueur parti puis son remplaçant, des bots qui partagent un index), l'index
 * ne suffit pas : la place revient à l'identité dont la PRÉSENCE publiée couvre l'image (bornes `from` à `toMax`, à
 * défaut `to`), à elle seule. Zéro ou plusieurs identités présentes : elle ne revient à personne, jamais à tous. Un
 * index que le roster ne porte pas ne revient à personne non plus : on ne devine pas un joueur.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import type { ReplayRosterEntry } from '@/lib/api/types'

import { rosterEntryKey } from './rosterLogic'

/** Une entrée du roster, vue par ce module : son index, son identité et les images où elle est présente. */
export type PlaceRosterEntry = Pick<ReplayRosterEntry, 'filmIndex' | 'xuid' | 'bot' | 'name'> & {
  presence: readonly { from: number; to: number; toMax?: number }[]
}

/** La présence publiée d'une entrée couvre-t-elle l'image ? La borne haute est `toMax`, à défaut `to`. */
function presentAt(entry: PlaceRosterEntry, frame: number): boolean {
  return entry.presence.some((p) => frame >= p.from && frame <= (p.toMax ?? p.to))
}

/** La clé de roster de l'unique identité d'un index, ou à défaut de l'unique identité présente à l'image ; `null` sinon. */
function ownerKeyOf(candidates: readonly PlaceRosterEntry[] | undefined, frame: number): string | null {
  if (candidates === undefined) return null
  const keys = new Set<string>()
  for (const entry of candidates) {
    const key = rosterEntryKey(entry)
    if (key) keys.add(key)
  }
  if (keys.size === 1) return [...keys][0]
  const present = new Set<string>()
  for (const entry of candidates) {
    const key = rosterEntryKey(entry)
    if (key && presentAt(entry, frame)) present.add(key)
  }
  return present.size === 1 ? [...present][0] : null
}

/**
 * placeOwnerResolver range le roster par index, une fois, et rend le résolveur `(place, frame) => clé de roster | null`.
 * La clé est celle de `ReplayPlayer.xuid` : le xuid d'un humain, `bot:<nom>` pour un bot.
 */
export function placeOwnerResolver(roster: readonly PlaceRosterEntry[]): (place: number, frame: number) => string | null {
  const byPlace = new Map<number, PlaceRosterEntry[]>()
  for (const entry of roster) byPlace.set(entry.filmIndex, [...(byPlace.get(entry.filmIndex) ?? []), entry])
  return (place, frame) => ownerKeyOf(byPlace.get(place), frame)
}
