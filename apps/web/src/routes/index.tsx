/**
 * Route index — redirige vers la page d'accueil du joueur actif.
 */
import { createFileRoute } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'

export const Route = createFileRoute('/')({
  component: IndexPage,
})

function IndexPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)

  if (!currentPlayer && availablePlayers.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-500">
        Aucun joueur configuré.{' '}
        <a href="/setup" className="ml-1 text-purple-600 underline">
          Configurer l'application
        </a>
      </div>
    )
  }

  const slug = currentPlayer?.player_slug ?? availablePlayers[0]?.player_slug

  if (slug) {
    // Redirection vers la page d'accueil du joueur
    return (
      <meta
        httpEquiv="refresh"
        content={`0; url=/players/${slug}/home`}
      />
    )
  }

  return null
}
