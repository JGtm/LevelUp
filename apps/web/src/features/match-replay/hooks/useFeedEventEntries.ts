/**
 * useFeedEventEntries — les lignes d'ÉVÉNEMENT du fil (objectifs, prises d'armes et de bonus, armes
 * ramassées au sol, montées et destructions de véhicule) tirées du document, UNE fois par document
 * et par langue. Elles ne rejoignent que la liste AFFICHÉE (`withEventLines`) : `model.feed` n'en
 * porte aucune.
 *
 * Les armes ramassées au sol se bâtissent APRÈS les prises de socle, contre lesquelles elles
 * écartent leurs doublons.
 *
 * Sans document (tests, chargement), aucune ligne : le fil reste celui des éliminations.
 */
import { useMemo } from 'react'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { padNameFor } from '../layers/useReplayWeaponPads'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { groundPickupFeedEntries } from '../model/groundPickupFeedLogic'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { objectiveFeedEntries } from '../model/objectiveFeedLogic'
import { pickupFeedEntries } from '../model/pickupFeedLogic'
import { vehicleFeedEntries } from '../model/vehicleFeedLogic'

export function useFeedEventEntries(
  doc: ReplayDocumentReady | null | undefined,
  locale: ReplayLocale,
): ReplayFeedEntry[] {
  return useMemo(() => {
    if (!doc) return []
    const t = REPLAY_TEXT[locale]
    const nameOf = (weapon: string) => padNameFor(weapon, doc.weaponLabels, t, locale)
    const padPickups = pickupFeedEntries(doc, nameOf)
    return [
      ...objectiveFeedEntries(doc).entries,
      ...padPickups,
      ...groundPickupFeedEntries(doc, nameOf, padPickups),
      ...vehicleFeedEntries(doc, locale),
    ]
  }, [doc, locale])
}
