/**
 * useSoloFilterStore — store des filtres pour le contexte Stats Solo.
 *
 * Consommé par : SoloFiltersBar/FilterOmnibar (timeseries, history), TimeseriesPage,
 * ExplorerMatchesTable (pour le filterSpec de nav vers match-view).
 *
 * Clé localStorage : `levelup-solo-filter-v1`.
 * URL share-link : `?f=...` (rétrocompat avec les shares existants).
 */

import { createFilterStore } from '@/stores/createFilterStore'
import { getApiLocale, getApiTitleSlug } from '@/lib/api/client'

export const useSoloFilterStore = createFilterStore({
  name: 'levelup-solo-filter-v1',
  urlEnabled: true,
  urlParam: 'f',
  // Estampille le titre actif dans le deep-link `?f=` : un share-link ne se
  // réapplique qu'au titre pour lequel il a été généré (cf. reconcileActiveTitle).
  getActiveTitleSlug: getApiTitleSlug,
  // Langue du partageur dans le lien : l'aperçu du lien sort dans la langue de
  // l'interface qui l'a produit (le robot d'aperçu n'en annonce aucune).
  // Lecture différée (et non la référence) : la locale est lue au clic, et un module qui
  // importe ce store sans jamais partager de lien ne touche pas l'accesseur.
  getLocale: () => getApiLocale(),
})
