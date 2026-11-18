/**
 * useFullWidthPage — LA PAGE COURANTE DEMANDE TOUTE LA LARGEUR DE LA FENÊTRE.
 *
 * La coquille borne le contenu des pages à `app-shell-width` (1320 px). Une page dont une vue est dessinée sur
 * toute la fenêtre (le Cinéma du rejeu) appelle ce hook avec `true` tant que cette vue est affichée : le drapeau
 * du store de la coquille est posé et la coquille retire le plafond, sans remonter la page. Il est retiré quand
 * `active` redevient faux et au démontage ; la mesure se fait avant la peinture, pour que la page ne montre
 * jamais une image à la mauvaise largeur.
 */
import { useLayoutEffect } from 'react'

import { useAppShellStore } from '@/stores/appShellStore'

export function useFullWidthPage(active: boolean): void {
  const setFullWidthPage = useAppShellStore((s) => s.setFullWidthPage)
  useLayoutEffect(() => {
    if (!active) return
    setFullWidthPage(true)
    return () => setFullWidthPage(false)
  }, [active, setFullWidthPage])
}
