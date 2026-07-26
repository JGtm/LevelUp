/**
 * MatchHeaderCard — layout asymétrique image map + contenu (mock C 2026-05-05).
 *
 * Découpé depuis MatchHeader.tsx (audit #6 god-file split).
 * Sous-sections internes : MapImage, ActionButtons, OutcomeRow, PerfRankRow.
 */
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { AlertDialog } from '@/components/ui/alert-dialog'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'
import { asDominance } from '@/components/charts/outcomeSequence'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'
import {
  OVERTIME_COLOR_TOKEN,
  overtimeLabel,
  overtimeTooltip,
} from '@/lib/narrative/overtime'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar } from '@/lib/accessibility'
import { useToggleMatchFavorite } from './queries'
import { useSetMatchExclusion } from '@/features/match-history/queries'
import {
  MATCH_VIEW_TEXT,
  type MatchViewLocale,
} from './i18n'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import type { MatchViewHeader as MatchViewHeaderData, MatchViewRank } from '@/lib/api/types'
import { formatDuration } from './MatchHeader.utils'
import { PerfRankRow } from './MatchHeader.perfRank'

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
  const excludeMutation = useSetMatchExclusion(playerSlug)
  const favoriteMutation = useToggleMatchFavorite(playerSlug, matchId)
  const [copied, setCopied] = useState(false)
  const [exclusionDialogOpen, setExclusionDialogOpen] = useState(false)

  // Un match classé (CSR officiel) ne peut pas être exclu — defense-in-depth :
  // le backend renvoie 422 si l'appel passe quand même, on intercepte côté UI
  // pour griser le bouton et expliquer la raison via tooltip.
  const excludeBlockedByRanked = header.is_ranked && !header.is_excluded

  const outcomeColor = header.outcome_color_token
    ? tokenCssVar(header.outcome_color_token as SemanticToken)
    : header.outcome_color
  const perfColor = header.performance_color_token
    ? tokenCssVar(header.performance_color_token as SemanticToken)
    : (header.performance_color ?? 'inherit')

  function handleCopyId() {
    void navigator.clipboard.writeText(matchId).then(() => {
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    })
  }

  function errorCode(err: unknown): string | null {
    if (err && typeof err === 'object' && 'code' in err) {
      const c = (err as { code?: unknown }).code
      if (typeof c === 'string') return c
    }
    return null
  }

  function openExclusionDialog() {
    if (excludeBlockedByRanked) return
    setExclusionDialogOpen(true)
  }

  function handleConfirmExclusion() {
    excludeMutation.mutate(
      { matchId, excluded: !header.is_excluded },
      {
        // Invalidations gérées au niveau de la mutation (matchHistory + matchView).
        onSuccess: () => setExclusionDialogOpen(false),
        onError: (err: unknown) => {
          setExclusionDialogOpen(false)
          const code = errorCode(err)
          if (code === 'ranked_not_excludable') {
            toast.error(t.excludeErrorRanked)
          } else {
            toast.error(t.excludeErrorGeneric)
          }
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
          <MapImageSection
            header={header}
            favoriteIsPending={favoriteMutation.isPending}
            mapUnknownLabel={t.mapUnknown}
            addFavoriteLabel={t.addFavorite}
            removeFavoriteLabel={t.removeFavorite}
            onToggleFavorite={handleToggleFavorite}
          />

          {/* Colonne droite : titre + outcome + actions + perf/rang */}
          <div className="flex flex-1 flex-col gap-3 px-6 py-4">
            <TitleAndActionsRow
              header={header}
              matchTitle={matchTitle}
              copied={copied}
              excludePending={excludeMutation.isPending}
              excludeBlockedByRanked={excludeBlockedByRanked}
              locale={locale}
              onCopyId={handleCopyId}
              onOpenExclusionDialog={openExclusionDialog}
            />

            <OutcomeRow header={header} outcomeColor={outcomeColor} locale={locale} />

            <PerfRankRow
              header={header}
              rank={rank}
              perfColor={perfColor}
              locale={locale}
            />
          </div>
        </div>
      </div>
      <AlertDialog
        open={exclusionDialogOpen}
        onOpenChange={setExclusionDialogOpen}
        title={header.is_excluded ? t.reactivateConfirmTitle : t.excludeConfirmTitle}
        description={header.is_excluded ? t.reactivateConfirmBody : t.excludeConfirmBody}
        confirmLabel={header.is_excluded ? t.reactivate : t.excludeShort}
        cancelLabel={t.cancelAction}
        destructive={!header.is_excluded}
        busy={excludeMutation.isPending}
        onConfirm={handleConfirmExclusion}
      />
    </div>
  )
}

// ── MapImageSection ────────────────────────────────────────────────────────

interface MapImageSectionProps {
  header: MatchViewHeaderData
  favoriteIsPending: boolean
  mapUnknownLabel: string
  addFavoriteLabel: string
  removeFavoriteLabel: string
  onToggleFavorite: () => void
}

function MapImageSection({
  header,
  favoriteIsPending,
  mapUnknownLabel,
  addFavoriteLabel,
  removeFavoriteLabel,
  onToggleFavorite,
}: MapImageSectionProps) {
  return (
    <div className="relative h-[230px] w-full flex-shrink-0 overflow-hidden bg-muted md:w-[400px]">
      {header.map_image_url ? (
        <img
          src={header.map_image_url}
          alt={header.map_ui || mapUnknownLabel}
          className="h-full w-full object-cover"
          loading="lazy"
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center text-sm text-muted-foreground">
          {header.map_ui || mapUnknownLabel}
        </div>
      )}
      {/* Étoile favori en overlay */}
      <button
        type="button"
        onClick={onToggleFavorite}
        disabled={favoriteIsPending}
        className="absolute left-3 top-3 rounded-full bg-black/40 p-2 transition-colors hover:bg-black/60 disabled:opacity-40"
        aria-label={header.is_favorite ? removeFavoriteLabel : addFavoriteLabel}
        title={header.is_favorite ? removeFavoriteLabel : addFavoriteLabel}
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
  )
}

// ── TitleAndActionsRow ─────────────────────────────────────────────────────

interface TitleAndActionsRowProps {
  header: MatchViewHeaderData
  matchTitle: string
  copied: boolean
  excludePending: boolean
  excludeBlockedByRanked: boolean
  locale: MatchViewLocale
  onCopyId: () => void
  onOpenExclusionDialog: () => void
}

function TitleAndActionsRow({
  header,
  matchTitle,
  copied,
  excludePending,
  excludeBlockedByRanked,
  locale,
  onCopyId,
  onOpenExclusionDialog,
}: TitleAndActionsRowProps) {
  const t = MATCH_VIEW_TEXT[locale]
  return (
    <div className="flex items-start justify-between gap-3">
      <div className="min-w-0">
        <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
          {header.is_ranked && <RankedIcon className="h-5 w-5 shrink-0 sm:h-6 sm:w-6" />}
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
          onClick={onCopyId}
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
          onClick={onOpenExclusionDialog}
          disabled={excludePending || excludeBlockedByRanked}
          title={
            excludeBlockedByRanked
              ? t.excludeRankedDenied
              : header.is_excluded
                ? t.reactivateTooltip
                : t.excludeTooltip
          }
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
  )
}

// ── OutcomeRow ─────────────────────────────────────────────────────────────

interface OutcomeRowProps {
  header: MatchViewHeaderData
  outcomeColor: string | null | undefined
  locale: MatchViewLocale
}

function OutcomeRow({ header, outcomeColor, locale }: OutcomeRowProps) {
  return (
    <div className="flex flex-wrap items-baseline gap-2">
      <span className="text-2xl font-bold" style={{ color: outcomeColor ?? undefined }}>
        {header.outcome_label}
      </span>
      {header.score_label && (
        <>
          <span className="text-2xl font-bold select-none text-muted-foreground">
            ·
          </span>
          <span
            className="text-2xl font-bold tabular-nums"
            style={{ color: outcomeColor ?? undefined }}
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
            flag={header.dominance_badge.flag}
            locale={locale}
          />
        </>
      )}
      {/* Prolongation : marqueur INDÉPENDANT du badge dominance — les deux
          peuvent coexister sur un même match (ex. remontada en prolongation). */}
      {header.is_overtime && (
        <NarrativeBadge
          className="self-center"
          label={overtimeLabel(locale)}
          title={overtimeTooltip(locale, header.overtime_seconds)}
          colorVar={tokenVar(OVERTIME_COLOR_TOKEN)}
          size="sm"
        />
      )}
    </div>
  )
}

// ── DominanceBadgeInline ───────────────────────────────────────────────────

// Le color_token backend ("narrative.dominance.loss.strong") ne correspond pas
// aux SemanticTokens frontend — on mappe par flag numérique via la table CANONIQUE
// partagée avec ExplorerMatchesTable et la bande de résultats (lib/narrative/dominance,
// D-10 V721-14a — ex-copie locale DOMINANCE_TOKENS supprimée).

interface DominanceBadgeInlineProps {
  labelKey: string
  flag: number
  locale: MatchViewLocale
}

// ── RankedIcon ─────────────────────────────────────────────────────────────
// Bouclier SVG (≠ étoile favori) indiquant un match classé officiel (CSR).

export function RankedIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      {/* color-allow: currentColor uniquement — couleur hérité du contexte */}
      <path
        fillRule="evenodd"
        d="M12 1.5a.75.75 0 0 1 .493.186l7.5 6.75A.75.75 0 0 1 20.25 9v6a6.75 6.75 0 0 1-13.5 0V9a.75.75 0 0 1 .257-.564l7.5-6.75A.75.75 0 0 1 12 1.5Zm0 1.652L5.25 9.283V15a5.25 5.25 0 0 0 10.5 0V9.283L12 3.152Z"
        clipRule="evenodd"
      />
      <path d="m11.47 10.22-1.5 1.5a.75.75 0 0 0 1.06 1.06l.97-.97.97.97a.75.75 0 1 0 1.06-1.06l-1.5-1.5a.75.75 0 0 0-1.06 0Z" />
    </svg>
  )
}

export function DominanceBadgeInline({ labelKey, flag, locale }: DominanceBadgeInlineProps) {
  const entry = matchViewManifest[labelKey as MatchViewManifestKey]
  const label = entry ? entry[locale] : labelKey
  const dominanceValue = asDominance(flag)
  const token = dominanceValue ? DOMINANCE_COLOR_TOKENS[dominanceValue] : undefined
  if (!token) return null
  const bgColor = tokenCssVar(token)
  const textColor = tokenCssVar(`${token}-text` as SemanticToken)
  return (
    <span
      className="self-center rounded px-2.5 py-0.5 text-sm font-bold uppercase tracking-wide"
      style={{ backgroundColor: bgColor, color: textColor }}
      data-testid="match-header-dominance-badge"
    >
      {label}
    </span>
  )
}
