/**
 * Route /terms — conditions d'utilisation.
 *
 * Chemin ANONYME : `routes/__root.tsx` l'exclut de la redirection vers /login (cf.
 * `ANONYMOUS_PATHS` de `components/shell/shellNavigation.ts`), sans quoi le lien du pied de
 * page de l'écran de connexion serait mort.
 */
import { createFileRoute } from '@tanstack/react-router'
import { TermsPage } from '@/features/legal/TermsPage'

export const Route = createFileRoute('/terms')({
  component: TermsPage,
})
