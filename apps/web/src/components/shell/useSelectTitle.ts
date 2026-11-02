import { useNavigate } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { applyActiveTitle } from '@/lib/title-routing/applyActiveTitle'
import { log } from '@/components/shell/_logger'

/**
 * Bascule vers un titre : ce qu'un clic sur une entrée de sélecteur de jeu déclenche (barre haute,
 * barre latérale dépliée ou repliée). Une entrée désactivée, déjà courante, ou une bascule en cours ne
 * fait rien.
 */
export function useSelectTitle(onSwitched?: () => void) {
  const navigate = useNavigate()
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)

  const onSelect = (slug: string, disabled: boolean, isCurrent: boolean) => {
    if (disabled || isCurrent || isTitleSwitching) return
    // NAVIGATE-FIRST (D-6, inversion de contrôle finale — Phase 4a). On ne bascule
    // PLUS ici : on navigue vers le SEGMENT du titre cible, et le layout
    // `t/$titleSlug` détecte la divergence segment↔store puis exécute
    // applyActiveTitle. Le playerSlug visé est celui du titre COURANT (joueur
    // courant, sinon 1er disponible) : VOULU — si ce joueur n'existe pas sur le
    // titre cible, le filet resolvePlayerFallback (PlayerLayout) re-cible au
    // fresh-load post-bascule.
    const state = useAppShellStore.getState()
    const playerSlug = state.currentPlayer?.player_slug ?? state.availablePlayers[0]?.player_slug
    if (playerSlug) {
      void navigate({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
        params: { titleSlug: slug, playerSlug },
      })
      onSwitched?.()
      return
    }
    // Cas limite : AUCUN joueur (availablePlayers vide, aucun joueur courant) → il
    // n'existe pas de route de titre nue (segment sans sous-arbre joueur) à cibler
    // par l'URL. Fallback DOCUMENTÉ :
    // basculer DIRECTEMENT via applyActiveTitle (hors chemin URL, faute de
    // destination joueur) puis renvoyer à l'index (qui gère l'onboarding et
    // re-résout le titre actif). Le menu se ferme quoi qu'il arrive.
    void (async () => {
      try {
        await applyActiveTitle(slug)
      } catch (err) {
        log.warn(
          'title_switch:direct_fallback_failed',
          'bascule directe (aucun joueur) échouée',
          err,
        )
      } finally {
        onSwitched?.()
        void navigate({ to: '/{-$lang}' })
      }
    })()
  }
  return onSelect
}
