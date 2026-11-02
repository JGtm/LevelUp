/**
 * Route /telecharger — page « Télécharger pour Windows ».
 *
 * Chemin ANONYME : `components/shell/shellNavigation.ts` (ANONYMOUS_PATHS) l'exclut de la redirection
 * vers /login, un visiteur doit pouvoir installer l'application avant d'avoir un compte. La page
 * elle-même se retire quand l'instance ne propose pas l'application (capacité du profil).
 */
import { createFileRoute } from '@tanstack/react-router'
import { DownloadPage } from '@/features/download/DownloadPage'

export const Route = createFileRoute('/telecharger')({
  component: DownloadPage,
})
