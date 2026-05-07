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
import { Link, useRouter } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'
import { useToggleMatchFavorite } from './queries'
import { useMatchNeighborsResolved } from '@/lib/match-nav/useMatchNeighborsResolved'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { clearNavContext } from '@/lib/match-nav/navContext'
import { useSetMatchExclusion } from '@/features/match-history/queries'
import { queryKeys } from '@/lib/query/keys'
import {
  MATCH_VIEW_TEXT,
  buildContextLabel,
  buildDescriptorLabel,
  type MatchViewLocale,
} from './i18n'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
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
              ↩ {t.exitContext}
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

// Alias rétrocompat pour l'ancien composant ◀ X/Y ▶ (utilisé jusqu'à la
// migration complète vers MatchNavigationBar).
export const MatchNavigation = MatchNavigationBar

// ────────────────────────────────────────────────────────────────────────────
// Header card — image + outcome + actions + perf/rang (mock C)
// ────────────────────────────────────────────────────────────────────────────

function formatDuration(secs: number): string {
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return s > 0 ? `${m}m ${s}s` : `${m}m`
}

const TIER_ORDER_EN = [
  'Bronze 1', 'Bronze 2', 'Bronze 3', 'Bronze 4', 'Bronze 5', 'Bronze 6',
  'Silver 1', 'Silver 2', 'Silver 3', 'Silver 4', 'Silver 5', 'Silver 6',
  'Gold 1', 'Gold 2', 'Gold 3', 'Gold 4', 'Gold 5', 'Gold 6',
  'Platinum 1', 'Platinum 2', 'Platinum 3', 'Platinum 4', 'Platinum 5', 'Platinum 6',
  'Diamond 1', 'Diamond 2', 'Diamond 3', 'Diamond 4', 'Diamond 5', 'Diamond 6',
  'Onyx',
] as const

// Chiffres romains I–VI dans l'ordre décroissant de longueur pour éviter
// qu'un préfixe court matche avant le suffixe complet (ex: " V" avant " VI").
const ROMAN_NEXT: Record<string, string> = {
  'VI': '',   // tier boundary — pas de "sous-tier 7"
  'V': 'VI', 'IV': 'V', 'III': 'IV', 'II': 'III', 'I': 'II',
}

function nextTierLabel(current: string | null | undefined): string | null {
  if (!current) return null

  // Lookup direct sur les labels anglais (ex: "Gold 3")
  const idx = TIER_ORDER_EN.indexOf(current as typeof TIER_ORDER_EN[number])
  if (idx !== -1) return idx < TIER_ORDER_EN.length - 1 ? TIER_ORDER_EN[idx + 1] : null

  // Fallback générique : "[nom] [chiffre romain I-VI]" (ex: "Or III" → "Or IV")
  // Vérifié du plus long au plus court pour éviter les faux-positifs.
  for (const rom of ['VI', 'V', 'IV', 'III', 'II', 'I'] as const) {
    if (current.endsWith(' ' + rom)) {
      const next = ROMAN_NEXT[rom]
      if (!next) return null  // sub-tier VI : tier boundary, nom inconnu
      return current.slice(0, -(rom.length + 1)) + ' ' + next
    }
  }

  // Fallback digit "[nom] [1-5]" (ex: "Gold 3" non anglais canonique)
  const m = current.match(/^(.+)\s+([1-5])$/)
  if (m) return `${m[1]} ${parseInt(m[2]) + 1}`

  return null
}

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

  // Barre composite de progression dans le tier.
  // tierSize = 50 pts (même constante que le backend buildRankBlock).
  const TIER_SIZE = 50
  const rankDeltaPct = rank.delta_value != null ? rank.delta_value / TIER_SIZE : 0
  const rankCurrentFill = rank.progress_pct ?? null
  // Position avant ce match (clampée à [0, 1] si le delta a changé de tier)
  const rankBeforeFill =
    rankCurrentFill != null
      ? Math.max(0, Math.min(1, rankCurrentFill - rankDeltaPct))
      : null
  // Portion stable = la plus petite des deux positions
  const rankBaseFill = rankCurrentFill != null && rankBeforeFill != null
    ? Math.min(rankCurrentFill, rankBeforeFill)
    : null
  // Segment delta : début et largeur en %
  const rankDeltaStart =
    rankCurrentFill != null && rankBeforeFill != null
      ? Math.min(rankCurrentFill, rankBeforeFill) * 100
      : 0
  const rankDeltaWidth =
    rankCurrentFill != null && rankBeforeFill != null
      ? Math.abs(rankCurrentFill - rankBeforeFill) * 100
      : 0
  const rankDeltaColor =
    rankDeltaPct > 0
      ? tokenCssVar('divergent-pos')
      : rankDeltaPct < 0
        ? tokenCssVar('divergent-neg')
        : tokenCssVar('divergent-neutral')

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
    <div className="px-6 pt-4">
      <div className="overflow-hidden rounded-lg border border-border bg-card text-card-foreground shadow-sm">
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
          {/* Titre + date + playlist · actions haut-droite */}
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
                {matchTitle}
              </h1>
              {(header.start_time_label || header.playlist_label || header.playable_duration_seconds) && (
                <div className="mt-1 flex items-center gap-2">
                  {header.start_time_label && (
                    <p className="text-sm text-muted-foreground">
                      {header.start_time_label}
                    </p>
                  )}
                  {header.playable_duration_seconds != null && (
                    <>
                      <span className="text-muted-foreground/50 select-none" aria-hidden="true">·</span>
                      <span
                        className="text-sm tabular-nums text-muted-foreground"
                        aria-label={`${t.duration} : ${formatDuration(header.playable_duration_seconds)}`}
                      >
                        {formatDuration(header.playable_duration_seconds)}
                      </span>
                    </>
                  )}
                  {header.playlist_label && (
                    <Badge variant="outline" className="text-xs">
                      {header.playlist_label}
                    </Badge>
                  )}
                </div>
              )}
            </div>

            {/* Boutons d'action haut-droite */}
            <div className="flex shrink-0 items-center gap-1.5">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopyId}
                title={t.copyTooltip}
                className="h-8 gap-1.5 text-xs"
              >
                {copied ? (
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2.5" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                ) : (
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                )}
                {copied ? t.copied : t.copyShort}
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={handleToggleExclusion}
                disabled={excludeMutation.isPending}
                title={header.is_excluded ? t.reactivateTooltip : t.excludeTooltip}
                className={
                  header.is_excluded
                    ? 'h-8 gap-1.5 text-xs'
                    : 'h-8 gap-1.5 text-xs text-destructive border-destructive/50 hover:bg-destructive/10 hover:text-destructive'
                }
              >
                {header.is_excluded ? (
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 15L3 9m0 0l6-6M3 9h12a6 6 0 010 12h-3" />
                  </svg>
                ) : (
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                  </svg>
                )}
                {header.is_excluded ? t.reactivate : t.excludeShort}
              </Button>
            </div>
          </div>

          {/* Outcome row : Victoire · 87-62 · Domination */}
          <div className="flex flex-wrap items-baseline gap-2">
            <span className="text-2xl font-bold" style={{ color: outcomeColor }}>
              {header.outcome_label}
            </span>
            {header.score_label && (
              <>
                <span className="text-2xl font-bold select-none text-muted-foreground">
                  ·
                </span>
                <span
                  className="text-2xl font-bold tabular-nums"
                  style={{ color: outcomeColor }}
                >
                  {header.score_label}
                </span>
              </>
            )}
            {header.dominance_badge && (
              <>
                <span className="text-2xl font-bold select-none text-muted-foreground">
                  ·
                </span>
                <DominanceBadgeInline
                  labelKey={header.dominance_badge.label_key}
                  colorToken={header.dominance_badge.color_token}
                  locale={locale}
                />
              </>
            )}
          </div>

          {/* Perf + rang row */}
          <div className="mt-auto flex flex-wrap items-start gap-y-3 border-t pt-3">
            {header.performance_display && header.performance_display !== '-' && (
              <div className="flex flex-col items-center">
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

            {header.performance_display && header.performance_display !== '-' && rank.rating_type !== 'none' && (
              <div className="mx-6 w-px self-stretch bg-border" />
            )}

            {rank.rating_type !== 'none' && (
              <div className="flex flex-1 items-center gap-3 min-w-0">
                {rank.icon_url && (
                  <img
                    src={rank.icon_url}
                    alt={rank.tier_label ?? rank.rating_type}
                    className="h-[44px] w-[44px] shrink-0 object-contain"
                    loading="lazy"
                  />
                )}
                <div className="flex flex-col gap-0.5 shrink-0">
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {t.rank}
                  </span>
                  {rank.tier_label && (
                    <span className="text-base font-bold text-foreground leading-none">
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

                {rankCurrentFill != null && rankBaseFill != null && (
                  // Conteneur relatif — les labels sont en absolute top-full pour
                  // ne pas gonfler la hauteur et décentrer la barre.
                  <div className="relative flex flex-1 items-center min-w-[80px]">
                    <div className="relative h-2 w-full overflow-hidden rounded-sm bg-muted">
                      {/* Portion stable (position avant le match) */}
                      <div
                        className="absolute inset-y-0 left-0"
                        style={{
                          width: `${(rankBaseFill * 100).toFixed(1)}%`,
                          backgroundColor: tokenCssVar('divergent-neutral'),
                        }}
                      />
                      {/* Segment delta (gain ou perte ce match) */}
                      {rankDeltaWidth > 0.1 && (
                        <div
                          className="absolute inset-y-0"
                          style={{
                            left: `${rankDeltaStart.toFixed(1)}%`,
                            width: `${rankDeltaWidth.toFixed(1)}%`,
                            backgroundColor: rankDeltaColor,
                          }}
                        />
                      )}
                    </div>
                    <div className="absolute inset-x-0 top-full mt-1 flex justify-between text-[10px] text-muted-foreground tabular-nums">
                      <span>{rank.tier_label ?? ''}</span>
                      <span>{nextTierLabel(rank.tier_label)}</span>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        </div>
      </div>
    </div>
  )
}

interface DominanceBadgeInlineProps {
  labelKey: string
  colorToken: string
  locale: MatchViewLocale
}

function DominanceBadgeInline({ labelKey, colorToken, locale }: DominanceBadgeInlineProps) {
  const entry = matchViewManifest[labelKey as MatchViewManifestKey]
  const label = entry ? entry[locale] : labelKey
  const color = tokenCssVar(colorToken as SemanticToken)
  return (
    <span
      className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-sm font-semibold uppercase tracking-wide"
      style={{
        backgroundColor: `color-mix(in oklab, ${color} 18%, transparent)`,
        borderColor: `color-mix(in oklab, ${color} 55%, transparent)`,
        color,
      }}
      data-testid="match-header-dominance-badge"
    >
      {label}
    </span>
  )
}
