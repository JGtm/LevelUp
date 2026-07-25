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
 * Appelée par le layout `t/$titleSlug` sur divergence segment↔store (D-6, chemin
 * principal — inversion de contrôle depuis Phase 4a), ET par le fallback
 * sans-joueur du TitleSwitcher (aucune route joueur à cibler par l'URL). THROW en
 * cas d'échec, SANS rollback interne : le chemin d'erreur appartient à l'APPELANT
 * (D-6 — le layout toaste + navigue vers le titre courant, Phase 4b). Gère
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
    // SÉQUENCE ANTI-FUITE CROSS-TITRE (V72-29). L'ordre est load-bearing : header,
    // session et clé de cache doivent bouger ENSEMBLE, et aucune requête de l'ancien
    // titre ne doit survivre dans la fenêtre du nouveau. `isTitleSwitching` reste vrai
    // sur toute la fenêtre → le layout `t/$titleSlug` rend `null` (sous-arbre joueur
    // démonté) et le refetch window-focus du bootstrap est neutralisé (cf. __root).
    //
    // 1. Annuler les requêtes EN VOL d'abord — avant tout changement d'état. Une
    //    réponse de l'ancien titre partie avant la bascule ne doit pas atterrir dans
    //    le cache après le clear (source de la fuite : donnée H5 sous la clé Infinite).
    await queryClient.cancelQueries()
    // 2. Commit serveur CONFIRMÉ. Le header n'est pas encore basculé : session et
    //    header restent cohérents (ancien titre) pendant l'attente — aucune fenêtre où
    //    le header dit un titre et la session un autre. /bootstrap dérive
    //    current_title_slug de la SESSION → ce commit doit précéder le re-bootstrap.
    await api.post('/session/context', { title_slug: slug })
    // 3. Basculer client API + store SEULEMENT après confirmation du POST. Header et
    //    clé de cache (via le store) passent au nouveau titre de façon synchrone.
    setApiTitleSlug(slug)
    store.setState({ currentTitleSlug: slug })
    // 4. Reset des filtres contextuels (state solo/squad lié à l'ancien titre :
    //    picked_sessions, cascade modes/maps/playlists). resetFilters réécrit aussi
    //    l'URL (?f=) et le localStorage — synchrone ici, estampillé au NOUVEAU titre.
    useSoloFilterStore.getState().resetFilters()
    useSquadFilterStore.getState().resetFilters()
    // 5. Purger tout le cache APRÈS la bascule du titre : aucune donnée de l'ancien
    //    titre ne survit ni ne se re-peuple sous une clé du nouveau.
    queryClient.clear()
    // 6. Re-bootstrap → données du nouveau titre + réhydratation (header = nouveau).
    const bootstrap = await api.get<BootstrapResponse>('/bootstrap')
    store.getState().hydrateFromBootstrap(bootstrap)
    // 7. Filet joueur : si le bootstrap n'a pas désigné de joueur courant alors que
    //    des joueurs existent, sélectionner le premier (sinon la NavL1 reste vide).
    const after = store.getState()
    if (after.currentPlayer == null && after.availablePlayers.length > 0) {
      after.setCurrentPlayer(after.availablePlayers[0])
    }
  } finally {
    store.setState({ isTitleSwitching: false })
  }
}
