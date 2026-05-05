import type { MatchMedal, MatchCitationSnippet } from '@/lib/api/types'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import type { MatchViewText } from './i18n'

// ---------------------------------------------------------------------------
// Médailles
// ---------------------------------------------------------------------------

interface MatchMedalsSectionProps {
  medals: MatchMedal[]
  t: MatchViewText
}

export function MatchMedalsSection({ medals, t }: MatchMedalsSectionProps) {
  if (medals.length === 0) return null

  return (
    <div>
      <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-3">
        {t.sectionMedals}
      </h3>
      <div className="flex flex-wrap gap-4">
        {medals.map((medal) => {
          const glow = dropShadowForDifficulty(medal.difficulty ?? undefined)
          return (
            <div
              key={medal.medal_name_id}
              title={medal.description ?? medal.name}
              className="flex flex-col items-center gap-1 cursor-default"
            >
              {medal.image_url ? (
                <img
                  src={medal.image_url}
                  alt={medal.name}
                  className="w-10 h-10 object-contain"
                  style={glow ? { filter: glow } : undefined}
                  onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                />
              ) : (
                <div className="w-10 h-10 rounded bg-muted flex items-center justify-center">
                  <span className="text-[9px] text-muted-foreground">{medal.medal_name_id}</span>
                </div>
              )}
              <span className="text-[9px] text-muted-foreground/80 leading-tight text-center max-w-[48px] truncate">
                {medal.name}
              </span>
              <span className="text-[10px] font-semibold text-foreground/70 leading-none">
                ×{medal.count}
              </span>
            </div>
          )
        })}
      </div>
    </div>
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
  if (citations.length === 0) return null

  return (
    <div>
      <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-3">
        {t.sectionCitations}
      </h3>
      <div className="flex flex-wrap gap-4">
        {citations.map((cit) => (
          <div
            key={cit.key}
            title={cit.description ?? cit.name}
            className="flex flex-col items-center gap-1 cursor-default"
          >
            <CitationProgressRing
              pct={cit.progress_pct}
              imageUrl={cit.image_url ?? undefined}
              isNewlyMastered={cit.is_newly_mastered}
              size={44}
            />
            <span className="text-[9px] text-muted-foreground/80 leading-tight text-center max-w-[52px] truncate">
              {cit.name}
            </span>
            <span className="text-[9px] font-semibold text-info leading-none">
              +{cit.delta}
            </span>
            {cit.is_newly_mastered && (
              <span className="text-[8px] font-bold text-warning leading-none">
                {t.newlyMastered}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
