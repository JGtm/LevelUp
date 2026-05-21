/**
 * Route index — redirige vers la page d'accueil du joueur actif.
 */
import { createFileRoute, Navigate } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export const Route = createFileRoute('/')({
  component: IndexPage,
})

function IndexPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  if (!currentPlayer && availablePlayers.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
        {t('common.index.no_player_configured')}{' '}
        <a href="/setup" className="ml-1 text-primary underline">
          {t('common.index.configure_app')}
        </a>
      </div>
    )
  }

  const slug = currentPlayer?.player_slug ?? availablePlayers[0]?.player_slug

  if (slug) {
    return (
      <Navigate
        to="/players/$playerSlug/home"
        params={{ playerSlug: slug }}
        replace
      />
    )
  }

  return null
}
