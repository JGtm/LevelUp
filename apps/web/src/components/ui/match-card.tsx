/**
 * MatchCard — tuile de match (Sprint 56).
 *
 * Affiche :
 *  - Image de la map (h-48, object-cover)
 *  - Titre centré `mode sur carte`
 *  - Playlist en sous-titre
 *  - Section score colorée selon le résultat avec badges narratifs
 */
import type { RecentMatchItem } from '@/lib/api/types'
import { getMatchCardOutcomeStyle, getMatchNarrativeBadgeMeta } from './match-card-presentation'

export interface MatchCardProps {
  match: RecentMatchItem
  locale?: 'fr' | 'en'
  onClick?: () => void
  onToggleFavorite?: () => void
  favoriteDisabled?: boolean
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function normalizeModeLabel(modeLabel: string | null | undefined, mapLabel: string | null | undefined): string | null {
  if (!modeLabel) {
    return null
  }

  let normalized = modeLabel.trim()
  if (!normalized) {
    return null
  }

  const spacedSeparatorIndex = normalized.indexOf(' : ')
  if (spacedSeparatorIndex > 0) {
    normalized = normalized.slice(0, spacedSeparatorIndex).trim()
  } else {
    const separatorIndex = normalized.lastIndexOf(':')
    if (separatorIndex >= 0 && separatorIndex < normalized.length - 1) {
      normalized = normalized.slice(separatorIndex + 1).trim()
    }
  }

  if (mapLabel?.trim()) {
    const escapedMap = escapeRegExp(mapLabel.trim())
    normalized = normalized.replace(new RegExp(`\\s+(?:on|sur)\\s+${escapedMap}$`, 'i'), '')
  } else {
    normalized = normalized.replace(/\s+(?:on|sur)\s+.+$/i, '')
  }

  normalized = normalized.replace(/\s*-\s*(?:Forge|Ranked)\b/gi, '').trim()
  return normalized || modeLabel.trim()
}
function buildMatchHeading(match: RecentMatchItem, locale: 'fr' | 'en'): string {
  const normalizedMode = normalizeModeLabel(match.mode_ui, match.map_ui)
  const connector = locale === 'en' ? 'on' : 'sur'

  if (normalizedMode && match.map_ui) {
    return `${normalizedMode} ${connector} ${match.map_ui}`
  }

  return normalizedMode ?? match.map_ui ?? match.title
}

export function MatchCard({ match: m, locale = 'fr', onClick, onToggleFavorite, favoriteDisabled }: MatchCardProps) {
  const heading = buildMatchHeading(m, locale)
  const outcomeStyle = getMatchCardOutcomeStyle(m.outcome_tone)
  const scoreLabel = m.score_label?.trim() ?? ''
  const narrativeBadges = m.narrative_badges ?? []

  return (
    <div
      className="rounded-xl overflow-hidden border border-border bg-[#1d2328] flex flex-col cursor-default hover:border-border transition-colors"
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Image de la map */}
      <div className="relative h-48 bg-muted overflow-hidden flex-shrink-0">
        {m.map_image_url ? (
          <img
            src={m.map_image_url}
            alt={m.map_ui ?? m.title}
            className="w-full h-full object-cover"
            loading="lazy"
            onError={(e) => {
              e.currentTarget.style.display = 'none'
              e.currentTarget.nextElementSibling?.removeAttribute('style')
            }}
          />
        ) : null}
        <div
          className="w-full h-full flex items-center justify-center text-muted-foreground text-xs"
          style={m.map_image_url ? { display: 'none' } : undefined}
        >
          {m.map_ui ?? 'Map inconnue'}
        </div>

        {/* Bouton favori en overlay coin supérieur gauche */}
        {onToggleFavorite && (
          <button
            type="button"
            aria-label={m.is_favorite ? 'Retirer des favoris' : 'Ajouter aux favoris'}
            disabled={favoriteDisabled}
            onClick={(e) => {
              e.stopPropagation()
              onToggleFavorite()
            }}
            className="absolute top-2 left-2 rounded-full p-1 bg-black/40 hover:bg-black/60 transition-colors disabled:opacity-40"
          >
            {m.is_favorite ? (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="#f59e0b" className="h-4 w-4" aria-hidden="true">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="#f59e0b" className="h-4 w-4" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.562.562 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
              </svg>
            )}
          </button>
        )}
      </div>

      {/* Corps */}
      <div className="flex flex-1 flex-col gap-3 px-3 py-3">
        <div className="space-y-1 text-center">
          <p className="text-sm font-semibold text-foreground leading-tight">
            {heading}
          </p>
          {m.playlist_ui && (
            <p className="text-xs text-muted-foreground leading-tight">
              {m.playlist_ui}
            </p>
          )}
        </div>

        <div
          data-testid="match-card-stats-panel"
          className="mt-auto flex h-28 flex-col items-center justify-center rounded-lg border px-3 py-3 text-center"
          style={{
            backgroundColor: outcomeStyle.panelBackground,
            borderColor: outcomeStyle.panelBorder,
          }}
        >
          <p
            data-testid="match-card-score"
            className="text-4xl font-black leading-none tracking-tight"
            style={{ color: outcomeStyle.scoreColor }}
          >
            {scoreLabel}
          </p>
          <div
            data-testid="match-card-badges-row"
            className="mt-3 flex min-h-6 flex-wrap items-center justify-center gap-2"
          >
            {narrativeBadges.map((badgeType) => {
              const badgeMeta = getMatchNarrativeBadgeMeta(badgeType)
              if (!badgeMeta) {
                return null
              }
              return (
                <span
                  key={badgeType}
                  className="rounded px-2 py-1 text-[10px] font-black uppercase tracking-[0.12em]"
                  style={{
                    backgroundColor: badgeMeta.color,
                    color: badgeMeta.textColor,
                  }}
                >
                  {badgeMeta.label}
                </span>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}
