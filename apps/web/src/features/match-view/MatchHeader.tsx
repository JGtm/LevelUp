/**
 * MatchHeader — composants d'en-tête de la page match (Breadcrumb + Navigation).
 *
 * P8.4 (revue 2026-04-29) : extraits de MatchViewPage.tsx (~120L).
 */
import { useEffect, useCallback } from 'react'
import { Link, useNavigate, useRouter } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useMatchNeighbors } from './queries'

interface MatchBreadcrumbProps {
  playerSlug: string
  matchLabel: string
}

export function MatchBreadcrumb({ playerSlug, matchLabel }: MatchBreadcrumbProps) {
  const router = useRouter()

  function handleBack() {
    const canGoBack = router.history.length > 1
    if (canGoBack) {
      router.history.back()
    } else {
      void router.navigate({
        to: '/players/$playerSlug/explorer',
        params: { playerSlug },
      })
    }
  }

  return (
    <div className="flex items-center gap-2 px-6 pt-4 pb-2 text-sm text-muted-foreground">
      <button
        type="button"
        onClick={handleBack}
        className="flex items-center gap-1 hover:text-foreground transition-colors"
        aria-label="Retour"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="h-4 w-4"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M17 10a.75.75 0 01-.75.75H5.612l4.158 3.96a.75.75 0 11-1.04 1.08l-5.5-5.25a.75.75 0 010-1.08l5.5-5.25a.75.75 0 111.04 1.08L5.612 9.25H16.25A.75.75 0 0117 10z"
            clipRule="evenodd"
          />
        </svg>
        Retour
      </button>
      <span aria-hidden="true">·</span>
      <Link
        to="/players/$playerSlug/home"
        params={{ playerSlug }}
        className="hover:text-foreground transition-colors truncate max-w-[12rem]"
      >
        {playerSlug}
      </Link>
      <span aria-hidden="true">›</span>
      <span className="text-foreground truncate">{matchLabel}</span>
    </div>
  )
}

interface MatchNavigationProps {
  playerSlug: string
  matchId: string
}

export function MatchNavigation({ playerSlug, matchId }: MatchNavigationProps) {
  const navigate = useNavigate()
  const { data: neighbors } = useMatchNeighbors(playerSlug, matchId)

  const goTo = useCallback(
    (targetMatchId: string | null | undefined) => {
      if (!targetMatchId) return
      void navigate({
        to: '/players/$playerSlug/matches/$matchId',
        params: { playerSlug, matchId: targetMatchId },
      })
    },
    [navigate, playerSlug],
  )

  // Raccourcis clavier ← / →
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      if (e.key === 'ArrowLeft') goTo(neighbors?.previous_match_id)
      if (e.key === 'ArrowRight') goTo(neighbors?.next_match_id)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [goTo, neighbors])

  if (!neighbors) return null

  return (
    <div className="flex items-center gap-2">
      <Button
        variant="ghost"
        size="sm"
        disabled={!neighbors.previous_match_id}
        onClick={() => goTo(neighbors.previous_match_id)}
        title="Match précédent (←)"
        aria-label="Match précédent"
      >
        ◀
      </Button>
      <span className="text-xs text-muted-foreground tabular-nums">
        {neighbors.current_index + 1} / {neighbors.total_matches}
      </span>
      <Button
        variant="ghost"
        size="sm"
        disabled={!neighbors.next_match_id}
        onClick={() => goTo(neighbors.next_match_id)}
        title="Match suivant (→)"
        aria-label="Match suivant"
      >
        ▶
      </Button>
    </div>
  )
}
