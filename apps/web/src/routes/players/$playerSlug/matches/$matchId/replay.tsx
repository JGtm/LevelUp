/**
 * Route /players/$playerSlug/matches/$matchId/rejeu — Rejouer le match en 2D.
 *
 * Stub en attente du projet externe de replay 2D.
 * Activer via `REJEU_2D_ENABLED` dans `src/lib/feature-flags.ts`.
 */
import { createFileRoute, useParams, Link } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'

export const Route = createFileRoute(
  '/players/$playerSlug/matches/$matchId/replay',
)({
  component: RejeuPage,
})

function RejeuPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }

  return (
    <div className="p-6">
      <Card>
        <CardContent className="py-12 text-center space-y-3">
          <p className="text-muted-foreground text-sm">
            Le rejouer le match en 2D est en cours de développement.
          </p>
          <Link
            to="/players/$playerSlug/matches/$matchId"
            params={{ playerSlug, matchId }}
            className="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-3 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            ← Retour au match
          </Link>
        </CardContent>
      </Card>
    </div>
  )
}
