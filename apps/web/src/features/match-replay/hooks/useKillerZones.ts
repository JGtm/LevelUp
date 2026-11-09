/**
 * useKillerZones — LE NOM DE LA ZONE DU TUEUR, ligne de frag par ligne de frag.
 *
 * La règle est celle de `model/killerZone.ts` (jumeau du Go) ; ce hook ne fait que la calculer UNE
 * fois par document, fil et carte (aucune recherche pendant la lecture) et rendre le résultat
 * dans la langue de la page. Un hook à part, parce qu'un hook exporté depuis un fichier de
 * composant coûte un avertissement `react-refresh`.
 *
 * `undefined` = la carte n'a AUCUNE zone nommée (Forge, par construction) : le fil ne réserve
 * alors pas de gouttière. Sinon la fonction rend le nom de la zone du tueur de la ligne, ou la
 * chaîne VIDE quand elle n'est pas établie — la gouttière reste, vide, pour que les colonnes
 * des lignes de frag s'alignent.
 */
import { useMemo } from 'react'

import { calloutLabel, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { killerZones } from '../model/killerZone'

/** Le nom de la zone du tueur d'une ligne du fil, désignée par sa clé (`ReplayFeedEntry.key`). */
export type KillerZoneOf = (entryKey: string) => string

export function useKillerZones(
  entries: readonly ReplayFeedEntry[],
  doc: ReplayDocumentReady | null | undefined,
  zones: readonly CalloutZoneReady[],
  locale: ReplayLocale,
): KillerZoneOf | undefined {
  const byKey = useMemo(() => (doc ? killerZones(entries, doc, zones) : null), [entries, doc, zones])
  return useMemo(() => {
    if (zones.length === 0) return undefined
    return (entryKey: string) => {
      const zone = byKey?.get(entryKey)
      return zone ? calloutLabel(zone, locale) : ''
    }
  }, [zones, byKey, locale])
}
