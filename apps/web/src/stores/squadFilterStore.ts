/**
 * useSquadFilterStore — store des filtres pour le contexte Escouade.
 *
 * Consommé par : SquadLayout (qui rend sa propre barre de filtres),
 * SquadV2RouteHost (lit period + cascade), SquadMatchHistoryTable (filterSpec
 * de nav vers match-view).
 *
 * Clé localStorage : `levelup-squad-filter-v1`. Pas d'URL share-link
 * (les shares squad sont rares en pratique, on garde `?f=` pour le solo).
 */

import { createFilterStore } from '@/stores/createFilterStore'

export const useSquadFilterStore = createFilterStore({
  name: 'levelup-squad-filter-v1',
  urlEnabled: false,
})
