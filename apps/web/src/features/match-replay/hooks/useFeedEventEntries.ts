/**
 * useFeedEventEntries — les lignes d'ÉVÉNEMENT du fil (objectifs, fins de manche, portages du crâne,
 * débuts de tenue de la colline, prises d'armes et de bonus, armes ramassées au sol, montées et
 * destructions de véhicule) tirées du document, UNE fois par document et par langue. Elles ne
 * rejoignent que la liste AFFICHÉE (`withEventLines`) : `model.feed` n'en porte aucune.
 *
 * `campIds` : les camps du film, dont la ligne de fin de manche écrit le score de chacun.
 *
 * Les armes ramassées au sol se bâtissent APRÈS les prises de socle, contre lesquelles elles
 * écartent leurs doublons.
 *
 * Le pictogramme de la cause d'une destruction de véhicule se retrouve par weapon_key dans le
 * référentiel des pictogrammes d'arme du titre (`icons`, cf. `useKillSourceIcons`) ; vide pendant le
 * chargement ou en cas d'erreur : les lignes s'affichent alors sans pictogramme.
 *
 * Sans document (tests, chargement), aucune ligne : le fil reste celui des éliminations.
 */
import { useMemo } from 'react'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { padNameFor } from '../layers/useReplayWeaponPads'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { groundPickupFeedEntries } from '../model/groundPickupFeedLogic'
import { hillFeedEntries } from '../model/hillFeedLogic'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { objectiveFeedEntries } from '../model/objectiveFeedLogic'
import { pickupFeedEntries } from '../model/pickupFeedLogic'
import { roundEndFeedEntries } from '../model/roundEndFeedLogic'
import { skullFeedEntries } from '../model/skullFeedLogic'
import { vehicleFeedEntries } from '../model/vehicleFeedLogic'
import { vehicleDestructionsOf } from '../model/vehicleDestructionSource'
import type { WeaponIconsByKey } from '../model/killSourceIcons'

export function useFeedEventEntries(
  doc: ReplayDocumentReady | null | undefined,
  locale: ReplayLocale,
  icons: WeaponIconsByKey,
  campIds: readonly number[],
): ReplayFeedEntry[] {
  return useMemo(() => {
    if (!doc) return []
    const t = REPLAY_TEXT[locale]
    const nameOf = (weapon: string) => padNameFor(weapon, doc.weaponLabels, t, locale, doc.abilityLabels)
    const padPickups = pickupFeedEntries(doc, nameOf)
    return [
      ...objectiveFeedEntries(doc).entries,
      ...roundEndFeedEntries(doc, campIds),
      ...skullFeedEntries(doc),
      ...hillFeedEntries(doc),
      ...padPickups,
      ...groundPickupFeedEntries(doc, nameOf, padPickups),
      ...vehicleFeedEntries(doc, locale, vehicleDestructionsOf(doc), icons),
    ]
  }, [doc, locale, icons, campIds])
}
