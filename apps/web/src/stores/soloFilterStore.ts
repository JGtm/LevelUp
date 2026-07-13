/**
 * useSoloFilterStore — store des filtres pour le contexte Stats Solo.
 *
 * Consommé par : NavL2/FilterOmnibar (timeseries, history), TimeseriesPage,
 * ExplorerMatchesTable (pour le filterSpec de nav vers match-view).
 *
 * Clé localStorage : `levelup-solo-filter-v1`.
 * URL share-link : `?f=...` (rétrocompat avec les shares existants).
 */

import { createFilterStore } from '@/stores/createFilterStore'
import { getApiTitleSlug } from '@/lib/api/client'

export const useSoloFilterStore = createFilterStore({
  name: 'levelup-solo-filter-v1',
  urlEnabled: true,
  urlParam: 'f',
  // Estampille le titre actif dans le deep-link `?f=` : un share-link ne se
  // réapplique qu'au titre pour lequel il a été généré (cf. reconcileActiveTitle).
  getActiveTitleSlug: getApiTitleSlug,
})
