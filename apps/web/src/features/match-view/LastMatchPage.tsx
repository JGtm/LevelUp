/**
 * LastMatchPage — résolution du dernier match dans le scope filtré.
 * Redirige automatiquement vers MatchViewPage une fois le match résolu.
 */
import { useEffect, useState } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useLastMatchResolve } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { LastMatchResolveResponse } from '@/lib/api/types'

export function LastMatchPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const filterContext = useGlobalFilterStore((s) => s.filterContext)

  const [resolved, setResolved] = useState<LastMatchResolveResponse | null>(null)
  const [currentIndex, setCurrentIndex] = useState<number | null>(null)
  const [error, setError] = useState(false)

  const mutation = useLastMatchResolve(playerSlug)

  function resolve(index?: number) {
    setError(false)
    mutation.mutate(
      { filters: filterContext, current_index: index ?? null },
      {
        onSuccess: (res) => {
          setResolved(res)
          setCurrentIndex(res.current_index)
        },
        onError: () => setError(true),
      },
    )
  }

  useEffect(() => {
    resolve()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playerSlug])

  function goToMatch(matchId: string) {
    navigate({
      to: '/players/$playerSlug/explorer/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  if (mutation.isPending) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Résolution du dernier match…" />
      </div>
    )
  }

  if (error || (mutation.isError && !resolved)) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Aucun match dans le scope actuel.</p>
            <button onClick={() => resolve()} className="mt-2 text-sm text-primary underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!resolved) return null

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Dernier match"
        subtitle={`${resolved.current_index + 1} / ${resolved.total_matches_in_scope} match${resolved.total_matches_in_scope > 1 ? 's' : ''} dans le scope`}
      />

      <div className="p-6 space-y-4">
        {/* Navigation prev / next */}
        <div className="flex items-center justify-between">
          <Button
            variant="outline"
            size="sm"
            disabled={resolved.previous_match_id == null}
            onClick={() => {
              if (resolved.previous_match_id) {
                resolve((currentIndex ?? 0) + 1)
              }
            }}
          >
            ← Précédent
          </Button>

          <Button
            variant="default"
            onClick={() => goToMatch(resolved.current_match_id)}
          >
            Voir le match →
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={resolved.next_match_id == null}
            onClick={() => {
              if (resolved.next_match_id) {
                resolve((currentIndex ?? 0) - 1)
              }
            }}
          >
            Suivant →
          </Button>
        </div>

        <Card>
          <CardContent className="py-4 text-sm text-muted-foreground space-y-1">
            <p>
              <span className="font-medium text-foreground">Match ID : </span>
              {resolved.current_match_id}
            </p>
            <p>
              <span className="font-medium text-foreground">Position : </span>
              {resolved.current_index + 1} / {resolved.total_matches_in_scope}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
