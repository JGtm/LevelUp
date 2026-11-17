/**
 * useFeedZones — LE NOM DE LA ZONE DE LA VICTIME, ligne de frag par ligne de frag.
 *
 * La règle est celle de `model/feedZone.ts` (jumeau du Go) ; ce hook ne fait que la calculer UNE
 * fois par document, fil et carte (aucune recherche pendant la lecture) et rendre le résultat
 * dans la langue de la page. Un hook à part, parce qu'un hook exporté depuis un fichier de
 * composant coûte un avertissement `react-refresh`.
 *
 * LA JOINTURE DU MATCH (`players`, la feuille jointe au film) est ce qui relie un BOT du fil (xuid de base `bid(N.0)`) à ses
 * vies du film : les bots comptent, ils ont donc leur zone comme toute victime.
 *
 * `undefined` = la carte n'a AUCUNE zone nommée (Forge, par construction) : le fil ne réserve
 * alors pas de gouttière. Sinon la fonction rend le nom de la zone de la victime de la ligne, ou la
 * chaîne VIDE quand elle n'est pas établie — la gouttière reste, vide, pour que les colonnes
 * des lignes de frag s'alignent.
 */
import { useMemo } from 'react'

import { calloutLabel, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayLocale } from '../i18n/i18n'
import { victimZones } from '../model/feedZone'
import type { ReplayFeedEntry } from '../model/killFeedLogic'

/** Le nom de la zone de la victime d'une ligne du fil, désignée par sa clé (`ReplayFeedEntry.key`). */
export type FeedZoneOf = (entryKey: string) => string

export function useFeedZones(
  entries: readonly ReplayFeedEntry[],
  doc: ReplayDocumentReady | null | undefined,
  zones: readonly CalloutZoneReady[],
  locale: ReplayLocale,
  players?: readonly ReplayPlayer[],
): FeedZoneOf | undefined {
  // La jointure du match joint les bots du fil (xuid de base) à leurs vies : sans elle, un bot n'a pas de zone.
  const byKey = useMemo(
    () => (doc ? victimZones(entries, doc, zones, players) : null),
    [entries, doc, zones, players],
  )
  return useMemo(() => {
    if (zones.length === 0) return undefined
    return (entryKey: string) => {
      const zone = byKey?.get(entryKey)
      return zone ? calloutLabel(zone, locale) : ''
    }
  }, [zones, byKey, locale])
}
