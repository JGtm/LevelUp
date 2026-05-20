import type { ReactNode } from 'react'
import type { MatchMedal, MatchCitationSnippet } from '@/lib/api/types'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import type { MatchViewText } from './i18n'

const CARD_HEIGHT = 280

function buildTooltip(name: string, description?: string | null): string {
  if (description && description.trim() !== '') {
    return `${name} : ${description}`
  }
  return name
}

// Mimétique de ChartCard (apps/web/src/components/charts/ChartCard.tsx) :
// même bordure / padding / titre avec border-b — pour s'aligner visuellement
// avec MatchWeaponPieChart sur la même rangée.
function PaneCard({
  title,
  children,
  isEmpty,
  emptyMessage,
}: {
  title: string
  children: ReactNode
  isEmpty: boolean
  emptyMessage: string
}) {
  return (
    <div className="relative rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">{title}</div>
      <div className="p-3" style={{ minHeight: CARD_HEIGHT }}>
        {isEmpty ? (
          <div
            className="flex items-center justify-center text-sm text-muted-foreground"
            style={{ minHeight: CARD_HEIGHT }}
          >
            {emptyMessage}
          </div>
        ) : (
          <div className="flex flex-wrap gap-x-5 gap-y-4">{children}</div>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Médailles
// ---------------------------------------------------------------------------

interface MatchMedalsSectionProps {
  medals: MatchMedal[]
  t: MatchViewText
}

export function MatchMedalsSection({ medals, t }: MatchMedalsSectionProps) {
  return (
    <PaneCard title={t.sectionMedals} isEmpty={medals.length === 0} emptyMessage={t.noMedals}>
      {medals.map((medal) => {
        const glow = dropShadowForDifficulty(medal.difficulty ?? undefined)
        const tooltip = buildTooltip(medal.name, medal.description)
        return (
          <div
            key={medal.medal_name_id}
            title={tooltip}
            className="flex flex-col items-center gap-1 cursor-default w-[74px]"
          >
            {medal.image_url ? (
              <img
                src={medal.image_url}
                alt={medal.name}
                className="w-[50px] h-[50px] object-contain"
                style={glow ? { filter: glow } : undefined}
                onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
              />
            ) : (
              <div className="w-[50px] h-[50px] rounded bg-muted flex items-center justify-center">
                <span className="text-2xs text-muted-foreground">{medal.medal_name_id}</span>
              </div>
            )}
            <span className="text-3xs text-muted-foreground leading-tight text-center w-full truncate">
              {medal.name}
            </span>
            <span className="text-3xs font-semibold text-foreground/80 leading-none">
              ×{medal.count}
            </span>
          </div>
        )
      })}
    </PaneCard>
  )
}

// ---------------------------------------------------------------------------
// Citations
// ---------------------------------------------------------------------------

interface MatchCitationsSectionProps {
  citations: MatchCitationSnippet[]
  t: MatchViewText
}

export function MatchCitationsSection({ citations, t }: MatchCitationsSectionProps) {
  return (
    <PaneCard title={t.sectionCitations} isEmpty={citations.length === 0} emptyMessage={t.noCitations}>
      {citations.map((cit) => {
        const tooltip = buildTooltip(cit.name, cit.description)
        const tierCount = cit.tier_count ?? 0
        const tierIndex = cit.tier_index ?? 0
        const cumulative = cit.cumulative ?? 0
        const nextTarget = cit.next_tier_target ?? 0
        const showTier = tierCount > 0
        const isMastered = showTier && tierIndex >= tierCount
        return (
          <div
            key={cit.key}
            title={tooltip}
            className="flex flex-col items-center gap-1 cursor-default w-[81px]"
          >
            <CitationProgressRing
              pct={cit.progress_pct}
              imageUrl={cit.image_url ?? undefined}
              isNewlyMastered={cit.is_newly_mastered}
              size={56}
            />
            <span className="text-[12px] text-muted-foreground leading-tight text-center w-full truncate">
              {cit.name}
            </span>
            {showTier && (
              <span className="text-[12px] font-semibold text-foreground/80 leading-none">
                {tierIndex}/{tierCount}
              </span>
            )}
            <div className="flex items-baseline justify-center gap-1.5 leading-none">
              {showTier && !isMastered && (
                <span className="text-3xs text-muted-foreground">
                  {cumulative}/{nextTarget}
                </span>
              )}
              <span className="text-[12px] font-semibold text-info">
                +{cit.delta}
              </span>
            </div>
            {cit.is_newly_mastered && (
              <span className="text-3xs font-bold text-warning leading-none">
                {t.newlyMastered}
              </span>
            )}
          </div>
        )
      })}
    </PaneCard>
  )
}
