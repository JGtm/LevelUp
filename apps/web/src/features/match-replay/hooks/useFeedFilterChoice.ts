/**
 * useFeedFilterChoice — LE FILTRE DU FIL tel que le panneau « Vue » le règle : les filtres que CE match propose (avec
 * leur nombre de lignes), le filtre appliqué, et le geste qui le change. C'est la même lecture que la colonne du fil
 * (`useFeedView`) — même liste complète, même filtre mémorisé, même repli sur Tout — mais sans l'instant lu : la
 * rubrique « Fil » règle le filtre aussi au Cinéma et au Plein cadre, où la colonne du fil est cachée.
 *
 * LE FILTRE EST UNE PRÉFÉRENCE PARTAGÉE (`useFeedFilter`) : le choix fait ici est celui que la colonne applique, et
 * l'inverse, sans qu'aucun des deux ne connaisse l'autre.
 */
import { useMemo } from 'react'

import type { FilmAllegiance } from '../../../lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayLocale } from '../i18n/i18n'
import { withEventLines } from '../model/feedEventLines'
import {
  availableFeedFilters,
  effectiveFeedFilter,
  type FeedFilterId,
  type FeedFilterOption,
} from '../model/feedFilterLogic'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { NO_WEAPON_ICONS } from '../model/killSourceIcons'
import { useFeedFilter } from '../settings/useFeedFilter'
import { useChatterFeedLines } from './useChatterLines'
import { useFeedEventEntries } from './useFeedEventEntries'

export interface FeedFilterChoice {
  options: readonly FeedFilterOption[]
  value: FeedFilterId
  /** Le filtre RETENU par le lecteur, même s'il n'est pas proposé pour ce match (`value` retombe alors sur Tout). */
  retained: FeedFilterId
  choose: (id: FeedFilterId) => void
}

export function useFeedFilterChoice(
  doc: ReplayDocumentReady | null | undefined,
  locale: ReplayLocale,
  allegiance: FilmAllegiance,
  entries: readonly ReplayFeedEntry[],
): FeedFilterChoice {
  const campIds = useMemo(() => allegiance.camps.map((c) => c.team), [allegiance])
  // Les pictogrammes d'arme ne changent aucun compte : les lignes de véhicule se comptent avec ou sans.
  const chatter = useChatterFeedLines(doc, locale)
  const events = useFeedEventEntries(doc, locale, NO_WEAPON_ICONS, campIds, chatter)
  const options = useMemo(() => availableFeedFilters(withEventLines(entries, events)), [entries, events])
  const [stored, choose] = useFeedFilter()
  return { options, value: effectiveFeedFilter(stored, options), retained: stored, choose }
}
