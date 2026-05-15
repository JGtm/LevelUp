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

export const useSoloFilterStore = createFilterStore({
  name: 'levelup-solo-filter-v1',
  urlEnabled: true,
  urlParam: 'f',
})
