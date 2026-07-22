/**
 * Route index — redirige vers la page d'accueil du joueur actif.
 */
import { createFileRoute, Navigate } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { useAppShellStore } from '@/stores/appShellStore'
import { resolveIndexRedirect } from '@/components/shell/shellNavigation'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export const Route = createFileRoute('/')({
  component: IndexPage,
})

function IndexPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const authMode = useAppShellStore((s) => s.authMode)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useTitleSlug()
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const redirect = resolveIndexRedirect({
    isBootstrapped,
    authMode,
    currentUsername,
    currentPlayerSlug: currentPlayer?.player_slug,
    firstAvailablePlayerSlug: availablePlayers[0]?.player_slug,
  })

  // Anonyme en mode auth (typiquement juste après « Se déconnecter », qui recharge
  // sur '/') : redirection DÉCLARATIVE vers /login. Sans elle, l'anonyme xbox
  // (available_players vide, filtrage ownership) tombait sur le fallback « setup »
  // rendu via le <Outlet/> nu de __root — donc SANS NavL1 — et la redirection
  // impérative de __root (navigate en useEffect) pouvait se perdre dans la course
  // avec le settle initial du routeur au rechargement plein. Un <Navigate> déclaratif
  // est traité par le routeur pendant le rendu de '/', immunisé contre cette course.
  if (redirect.kind === 'login') {
    return <Navigate to="/login" replace />
  }

  if (redirect.kind === 'player') {
    return (
      <Navigate
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
        params={{ titleSlug, playerSlug: redirect.slug }}
        replace
      />
    )
  }

  if (redirect.kind === 'setup') {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
        {t('common.index.no_player_configured')}{' '}
        <a href="/setup" className="ml-1 text-primary underline">
          {t('common.index.configure_app')}
        </a>
      </div>
    )
  }

  // redirect.kind === 'wait' — bootstrap pas encore hydraté : ne rien rendre (évite
  // un flash du fallback sans shell). __root affiche déjà l'écran de chargement.
  return null
}
