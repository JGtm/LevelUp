/**
 * MatchHeader — composants d'en-tête de la page match.
 *
 * Refonte 2026-05-05 (mock C) :
 *   - MatchBreadcrumb (existant) : breadcrumb retour + slug + label
 *   - MatchNavigationBar (nouveau) : barre nav full-width avec labels textuels
 *     "← Match précédent · Match X/Y · Match suivant →"
 *   - MatchHeaderCard (nouveau) : layout asymétrique image map (gauche) +
 *     contenu (droite : titre + outcome + actions + perf/rang).
 *
 * MatchNavigation (legacy ◀ Y/X ▶) reste exporté en alias pour compat
 * (utilisé par MatchViewPage avant migration).
 */
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useRouter } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'
import { useMatchNeighbors, useToggleMatchFavorite } from './queries'
import { useSetMatchExclusion } from '@/features/match-history/queries'
import { queryKeys } from '@/lib/query/keys'
import { MATCH_VIEW_TEXT, type MatchViewLocale } from './i18n'
import type { MatchViewHeader as MatchViewHeaderData, MatchViewRank } from '@/lib/api/types'

// ────────────────────────────────────────────────────────────────────────────
// Breadcrumb (inchangé)
// ────────────────────────────────────────────────────────────────────────────

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

// ────────────────────────────────────────────────────────────────────────────
// Navigation bar — Match précédent · Match X/Y · Match suivant
// ────────────────────────────────────────────────────────────────────────────

interface MatchNavigationBarProps {
  playerSlug: string
  matchId: string
  locale: MatchViewLocale
}

export function MatchNavigationBar({ playerSlug, matchId, locale }: MatchNavigationBarProps) {
  const navigate = useNavigate()
  const t = MATCH_VIEW_TEXT[locale]
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

  const counter = t.matchCounter(neighbors.current_index + 1, neighbors.total_matches)

  return (
    <div className="flex items-center justify-between border-b bg-card px-6 py-2">
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
      <span className="text-xs font-medium tabular-nums text-muted-foreground">
        {counter}
      </span>
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

// Alias rétrocompat pour l'ancien composant ◀ X/Y ▶ (utilisé jusqu'à la
// migration complète vers MatchNavigationBar).
export const MatchNavigation = MatchNavigationBar

// ────────────────────────────────────────────────────────────────────────────
// Header card — image + outcome + actions + perf/rang (mock C)
// ────────────────────────────────────────────────────────────────────────────

interface MatchHeaderCardProps {
  header: MatchViewHeaderData
  rank: MatchViewRank
  matchId: string
  playerSlug: string
  matchTitle: string
  locale: MatchViewLocale
}

export function MatchHeaderCard({
  header,
  rank,
  matchId,
  playerSlug,
  matchTitle,
  locale,
}: MatchHeaderCardProps) {
  const t = MATCH_VIEW_TEXT[locale]
  const queryClient = useQueryClient()
  const excludeMutation = useSetMatchExclusion(playerSlug)
  const favoriteMutation = useToggleMatchFavorite(playerSlug, matchId)
  const [copied, setCopied] = useState(false)

  const outcomeColor = header.outcome_color_token
    ? tokenCssVar(header.outcome_color_token as SemanticToken)
    : header.outcome_color
  const perfColor = header.performance_color_token
    ? tokenCssVar(header.performance_color_token as SemanticToken)
    : (header.performance_color ?? 'inherit')
  const deltaColor =
    rank.delta_value != null
      ? tokenCssVar(skillDeltaScale(rank.delta_value))
      : 'inherit'

  function handleCopyId() {
    void navigator.clipboard.writeText(matchId).then(() => {
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    })
  }

  function handleToggleExclusion() {
    excludeMutation.mutate(
      { matchId, excluded: !header.is_excluded },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: queryKeys.matchView(playerSlug, matchId),
          })
        },
      },
    )
  }

  function handleToggleFavorite() {
    favoriteMutation.mutate(!header.is_favorite)
  }

  return (
    <div className="border-b bg-card">
      <div className="flex flex-col gap-0 md:flex-row">
        {/* Colonne image map (400×230 max selon les assets 560/320) */}
        <div className="relative h-[230px] w-full flex-shrink-0 overflow-hidden bg-muted md:w-[400px]">
          {header.map_image_url ? (
            <img
              src={header.map_image_url}
              alt={header.map_ui || t.mapUnknown}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-sm text-muted-foreground">
              {header.map_ui || t.mapUnknown}
            </div>
          )}
          {/* Étoile favori en overlay */}
          <button
            type="button"
            onClick={handleToggleFavorite}
            disabled={favoriteMutation.isPending}
            className="absolute left-3 top-3 rounded-full bg-black/40 p-2 transition-colors hover:bg-black/60 disabled:opacity-40"
            aria-label={header.is_favorite ? t.removeFavorite : t.addFavorite}
            title={header.is_favorite ? t.removeFavorite : t.addFavorite}
          >
            {header.is_favorite ? (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="#f59e0b" className="h-5 w-5" aria-hidden="true"> {/* color-allow: amber gold pour étoile favori — CLAUDE.md §20 (warning/amber UI générique) */}
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="#f59e0b" className="h-5 w-5" aria-hidden="true"> {/* color-allow: amber gold pour étoile favori (outline) — CLAUDE.md §20 */}
                <path strokeLinecap="round" strokeLinejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.562.562 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
              </svg>
            )}
          </button>
        </div>

        {/* Colonne droite : titre + outcome + actions + perf/rang */}
        <div className="flex flex-1 flex-col gap-3 px-6 py-4">
          {/* Titre + date */}
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
              {matchTitle}
            </h1>
            {header.start_time_label && (
              <p className="mt-1 text-sm text-muted-foreground">
                {header.start_time_label}
              </p>
            )}
          </div>

          {/* Outcome row : Victoire 87-62 + Playlist tag */}
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-2xl font-bold" style={{ color: outcomeColor }}>
              {header.outcome_label}
            </span>
            {header.score_label && (
              <span className="text-sm text-muted-foreground tabular-nums">
                {header.score_label}
              </span>
            )}
            {header.playlist_label && (
              <Badge variant="outline" className="text-xs">
                {header.playlist_label}
              </Badge>
            )}
          </div>

          {/* Actions bar : copier ID · marquer non pertinent */}
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <button
              type="button"
              onClick={handleCopyId}
              className="rounded px-2 py-1 hover:bg-muted hover:text-foreground transition-colors"
              title={t.copyMatchId}
            >
              {copied ? `✓ ${t.copied}` : t.copyMatchId}
            </button>
            <span aria-hidden="true">·</span>
            <button
              type="button"
              onClick={handleToggleExclusion}
              disabled={excludeMutation.isPending}
              className={
                header.is_excluded
                  ? 'rounded px-2 py-1 hover:bg-muted hover:text-foreground transition-colors'
                  : 'rounded px-2 py-1 hover:bg-destructive/10 hover:text-destructive transition-colors'
              }
            >
              {header.is_excluded ? `↩ ${t.reactivate}` : `${t.markIrrelevant} ⊘`}
            </button>
          </div>

          {/* Perf + rang row */}
          <div className="mt-auto flex flex-wrap items-end gap-x-8 gap-y-3 border-t pt-3">
            {header.performance_display && header.performance_display !== '-' && (
              <div className="flex flex-col">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {t.performance}
                </span>
                <span
                  className="text-4xl font-black leading-none tabular-nums"
                  style={{ color: perfColor }}
                >
                  {header.performance_display}
                </span>
              </div>
            )}

            {rank.rating_type !== 'none' && (
              <div className="flex flex-col">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {t.rank}
                </span>
                <div className="flex items-center gap-3">
                  {rank.icon_url && (
                    <img
                      src={rank.icon_url}
                      alt={rank.tier_label ?? rank.rating_type}
                      className="h-12 w-12 object-contain"
                      loading="lazy"
                    />
                  )}
                  <div className="flex flex-col gap-0.5">
                    {rank.tier_label && (
                      <span className="text-base font-bold text-foreground">
                        {rank.tier_label}
                      </span>
                    )}
                    <div className="flex items-center gap-2 text-xs">
                      {rank.numeric_value != null && (
                        <span className="tabular-nums text-muted-foreground">
                          {rank.rating_type} {rank.numeric_value.toFixed(0)}
                        </span>
                      )}
                      {rank.delta_value != null && (
                        <span
                          className="font-bold tabular-nums"
                          style={{ color: deltaColor }}
                        >
                          {rank.delta_value >= 0
                            ? `▲ +${rank.delta_value.toFixed(0)}`
                            : `▼ ${rank.delta_value.toFixed(0)}`}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
