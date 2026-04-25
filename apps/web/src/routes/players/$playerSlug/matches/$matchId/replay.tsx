/**
 * Route /players/$playerSlug/matches/$matchId/rejeu — Rejouer le match en 2D.
 *
 * Stub en attente du projet externe de replay 2D.
 * Activer via `REJEU_2D_ENABLED` dans `src/lib/feature-flags.ts`.
 */
import { createFileRoute, useParams, Link } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

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
          <Button variant="outline" size="sm" asChild>
            <Link
              to="/players/$playerSlug/matches/$matchId"
              params={{ playerSlug, matchId }}
            >
              ← Retour au match
            </Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
