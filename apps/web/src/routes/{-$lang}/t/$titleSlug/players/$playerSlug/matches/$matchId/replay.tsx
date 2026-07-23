/**
 * Route /players/$playerSlug/matches/$matchId/rejeu — Rejouer le match en 2D.
 *
 * Stub en attente du projet externe de replay 2D.
 * Activer via `REJEU_2D_ENABLED` dans `src/lib/feature-flags.ts`.
 */
import { createFileRoute, Link } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
)({
  // Sous-route du détail match : sans capability `matchmaking`, le rejeu n'a pas
  // de sens (le match n'existe pas pour ce titre). NO-OP halo_infinite.
  component: () => (
    <RouteCapabilityGate capability="matchmaking">
      <RejeuPage />
    </RouteCapabilityGate>
  ),
})

function RejeuPage() {
  // Route.useParams() porte lang?/titleSlug/playerSlug/matchId (typés) — même forme
  // que la cible matches/$matchId, réutilisable tel quel (préserve titre ET langue).
  const params = Route.useParams()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  return (
    <div className="p-6">
      <Card>
        <CardContent className="py-12 text-center space-y-3">
          <p className="text-muted-foreground text-sm">
            {t('common.replay.in_development')}
          </p>
          <Link
            to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId"
            params={params}
            className="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-3 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {t('common.replay.back_to_match')}
          </Link>
        </CardContent>
      </Card>
    </div>
  )
}
