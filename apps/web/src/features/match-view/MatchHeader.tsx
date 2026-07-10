/**
 * MatchHeader — composants d'en-tête de la page match.
 *
 * Refonte 2026-05-05 (mock C) :
 *   - MatchBreadcrumb (existant) : breadcrumb retour + slug + label
 *   - MatchNavigationBar (nouveau) : barre nav full-width avec labels textuels
 *     "← Match précédent · Match X/Y · Match suivant →"
 *   - MatchHeaderCard (déplacé dans MatchHeader.card.tsx) : layout asymétrique
 *     image map (gauche) + contenu (droite : titre + outcome + actions + perf/rang).
 *
 * MatchNavigation (legacy ◀ Y/X ▶) reste exporté en alias pour compat
 * (utilisé par MatchViewPage avant migration).
 *
 * Découpage 2026-05 (audit #6 god-file split) :
 *   - MatchHeader.utils.ts : formatDuration() + nextTierLabel()
 *   - MatchHeader.card.tsx : MatchHeaderCard + sous-sections (Map, Title, Outcome, PerfRank)
 */
import { useCallback, useEffect } from 'react'
import { Link, useRouter } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useMatchNeighborsResolved } from '@/lib/match-nav/useMatchNeighborsResolved'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { clearNavContext } from '@/lib/match-nav/navContext'
import {
  MATCH_VIEW_TEXT,
  buildContextLabel,
  buildDescriptorLabel,
  type MatchViewLocale,
} from './i18n'

// Re-export pour préserver l'API publique (tests + MatchViewPage)
export { MatchHeaderCard, DominanceBadgeInline } from './MatchHeader.card'

// ────────────────────────────────────────────────────────────────────────────
// Breadcrumb (inchangé)
// ────────────────────────────────────────────────────────────────────────────

interface MatchBreadcrumbProps {
  playerSlug: string
  matchLabel: string
  locale: MatchViewLocale
}

export function MatchBreadcrumb({ playerSlug, matchLabel, locale }: MatchBreadcrumbProps) {
  const router = useRouter()
  const t = MATCH_VIEW_TEXT[locale]

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
        aria-label={t.back}
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
        {t.back}
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

// ────────────────────────────────────────────────────────────────────────────
// Navigation bar — Match précédent · Match X/Y · Match suivant
// ────────────────────────────────────────────────────────────────────────────

interface MatchNavigationBarProps {
  playerSlug: string
  matchId: string
  locale: MatchViewLocale
}

export function MatchNavigationBar({ playerSlug, matchId, locale }: MatchNavigationBarProps) {
  const t = MATCH_VIEW_TEXT[locale]
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const { data: neighbors, source, contextLabel, contextDescriptor, navContext } =
    useMatchNeighborsResolved(playerSlug, matchId)

  const goTo = useCallback(
    (targetMatchId: string | null | undefined) => {
      if (!targetMatchId) return
      // Propage le contexte courant lors d'un prev/next : la liste matchIds
      // reste valable, on continue dans le même périmètre filtré.
      navigateToMatch(targetMatchId, navContext)
    },
    [navigateToMatch, navContext],
  )

  const exitContext = useCallback(() => {
    // Purge sessionStorage + state + query params URL pour ce matchId.
    // Phase 2b : si l'URL contenait playlist/mode/from/..., on les retire
    // pour que le hook cascade retombe sur l'API globale propre.
    clearNavContext(matchId)
    const url = new URL(window.location.href)
    for (const k of ['playlist', 'mode', 'from', 'to', 'session', 'outcome', 'with_player']) {
      url.searchParams.delete(k)
    }
    window.history.replaceState({}, '', url.toString())
    window.dispatchEvent(new PopStateEvent('popstate'))
  }, [matchId])

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

  // Cascade des labels (Phase 2c) :
  //   1. ContextDescriptor typé (préféré, depuis state/sessionStorage)
  //   2. filtersLabel pré-localisé legacy (Phase 2a)
  //   3. buildContextLabel(filterSpec) côté API (Phase 2b, URL params)
  // Si descriptor → format intégré "Matchs <ctx> X/Y" (matchCounterCtxFmt).
  // Sinon → "Match X/Y" classique + chip contexte à droite.
  const descriptorFragment = buildDescriptorLabel(contextDescriptor, locale)
  const apiContextLabel =
    source === 'api' && navContext?.filterSpec
      ? buildContextLabel(navContext.filterSpec, locale)
      : null
  const fallbackLabel = contextLabel ?? apiContextLabel ?? null
  const counter = descriptorFragment
    ? t.matchCounterCtxFmt(descriptorFragment, neighbors.current_index + 1, neighbors.total_matches)
    : t.matchCounter(neighbors.current_index + 1, neighbors.total_matches)
  const showExit = !!descriptorFragment || !!fallbackLabel

  return (
    <div className="flex items-center justify-between gap-2 px-6 py-2">
      <Button
        variant="ghost"
        size="sm"
        disabled={!neighbors.previous_match_id}
        onClick={() => goTo(neighbors.previous_match_id)}
        title={`${t.prevMatch} (←)`}
        aria-label={t.prevMatch}
        className="text-sm"
      >
        ← {t.prevMatch}
      </Button>
      <div className="flex min-w-0 flex-wrap items-center justify-center gap-x-2 gap-y-0.5 text-xs text-muted-foreground">
        <span className="font-medium tabular-nums truncate">{counter}</span>
        {!descriptorFragment && fallbackLabel && (
          <>
            <span aria-hidden="true">·</span>
            <span className="truncate">{fallbackLabel}</span>
          </>
        )}
        {showExit && (
          <>
            <span aria-hidden="true">·</span>
            <button
              type="button"
              onClick={exitContext}
              className="underline-offset-2 hover:text-foreground hover:underline"
              title={t.exitContext}
            >
              {t.exitContext} ↩
            </button>
          </>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        disabled={!neighbors.next_match_id}
        onClick={() => goTo(neighbors.next_match_id)}
        title={`${t.nextMatch} (→)`}
        aria-label={t.nextMatch}
        className="text-sm"
      >
        {t.nextMatch} →
      </Button>
    </div>
  )
}
