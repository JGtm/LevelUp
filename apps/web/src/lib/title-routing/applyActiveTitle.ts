import { api, setApiTitleSlug } from '@/lib/api/client'
import { queryClient } from '@/app/queryClient'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import type { BootstrapResponse } from '@/lib/api/types'

/**
 * applyActiveTitle — UNIQUE fonction effectful du module title-routing (D-10),
 * extraite de `appShellStore.switchTitle`. Applique un nouveau titre actif :
 * POST /session/context → client API + store → reset filtres → purge cache →
 * re-bootstrap + hydrate → filet joueur.
 *
 * Réutilisée par le bouton (switchTitle wrapper) ET, en Phase 2+, par le layout
 * `t/$titleSlug` sur divergence segment↔store (D-6). THROW en cas d'échec, SANS
 * rollback interne : le chemin d'erreur appartient à l'APPELANT (D-6 — le layout
 * navigue en arrière, le bouton fait un rollback store-only). Gère
 * `isTitleSwitching`. No-op si le slug est déjà courant.
 *
 * NB imports circulaires (store ↔ module) : ce fichier importe le store (pas
 * l'inverse — appShellStore importe applyActiveTitle). Le cycle est runtime-safe
 * (usages différés dans les corps de fonction, jamais à l'init des modules).
 */
export async function applyActiveTitle(slug: string): Promise<void> {
  const store = useAppShellStore
  if (slug === store.getState().currentTitleSlug) return

  store.setState({ isTitleSwitching: true })
  try {
    // 1. Commit côté serveur. Le header X-LevelUp-Title affirme le titre sur CHAQUE
    // requête (getTitleHeader) → la résolution per-requête ne dépend plus de la
    // session : ce POST n'est PLUS sur le chemin critique de l'ordre des refetch.
    // Il reste requis pour la PERSISTANCE/REPRISE (F5) : /bootstrap dérive
    // current_title_slug de la SESSION, pas du header.
    await api.post('/session/context', { title_slug: slug })
    // 2. Basculer client API + store AVANT tout refetch.
    setApiTitleSlug(slug)
    store.setState({ currentTitleSlug: slug })
    // 2bis. Reset des filtres contextuels (state solo/squad lié à l'ancien titre :
    // picked_sessions, cascade modes/maps/playlists). resetFilters réécrit aussi
    // l'URL (?f=) et le localStorage — synchrone ici.
    useSoloFilterStore.getState().resetFilters()
    useSquadFilterStore.getState().resetFilters()
    // 3. Annuler les requêtes en vol PUIS purger tout le cache (après le commit du
    // titre : aucune donnée de l'ancien titre ne survit ni ne se re-peuple).
    await queryClient.cancelQueries()
    queryClient.clear()
    // 4. Re-bootstrap → données du nouveau titre + réhydratation.
    const bootstrap = await api.get<BootstrapResponse>('/bootstrap')
    store.getState().hydrateFromBootstrap(bootstrap)
    // 5. Filet joueur : si le bootstrap n'a pas désigné de joueur courant alors que
    // des joueurs existent, sélectionner le premier (sinon la NavL1 reste vide).
    const after = store.getState()
    if (after.currentPlayer == null && after.availablePlayers.length > 0) {
      after.setCurrentPlayer(after.availablePlayers[0])
    }
  } finally {
    store.setState({ isTitleSwitching: false })
  }
}
