/**
 * Route /privacy — politique de confidentialité.
 *
 * Chemin ANONYME : `routes/__root.tsx` l'exclut de la redirection vers /login,
 * sans quoi le lien du pied de page de l'écran de connexion serait mort.
 */
import { createFileRoute } from '@tanstack/react-router'
import { PrivacyPage } from '@/features/legal/PrivacyPage'

export const Route = createFileRoute('/privacy')({
  component: PrivacyPage,
})
