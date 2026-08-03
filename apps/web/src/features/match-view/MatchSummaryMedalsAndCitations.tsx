import type { ReactNode } from 'react'
import type { MatchMedal, MatchCitationSnippet, MatchNativeCommendation } from '@/lib/api/types'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { citationMastery } from '@/lib/citations/mastery'
import { MedalIcon } from '@/components/ui/MedalIcon'
import type { MatchViewText } from './i18n'

/**
 * Plancher de hauteur du CORPS de carte (hors bandeau de titre). Remplace
 * l'ancien plancher fixe de 280 px qui réservait la hauteur de 3 rangées de
 * vignettes même pour un match à 2 médailles (v7.3 lot 2, item 2.1).
 *
 * L'égalisation des deux cartes d'une même rangée est laissée à la grille CSS
 * du parent (`MatchViewPage`, `align-items: stretch` par défaut) : la carte
 * riche impose sa hauteur à sa voisine, et quand les DEUX sont pauvres la
 * rangée entière se compacte. Le plancher ne sert plus qu'à garder un état
 * vide (« Aucune médaille ») lisible et une carte à une seule rangée aérée.
 */
export const MEDALS_CARD_MIN_BODY_HEIGHT = 96

function buildTooltip(name: string, description?: string | null): string {
  if (description && description.trim() !== '') {
    return `${name} : ${description}`
  }
  return name
}

// Mimétique de ChartCard (apps/web/src/components/charts/ChartCard.tsx) :
// même bordure / padding / titre avec border-b — pour s'aligner visuellement
// avec les cartes de la même rangée (dont MatchFragCard).
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
  // `flex flex-col` + corps `flex-1` : la carte remplit la cellule de grille que
  // sa voisine étire, sans jamais réserver de hauteur quand rien ne l'étire.
  return (
    <div className="relative flex flex-col rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{title}</div>
      <div
        className={`flex-1 p-3${isEmpty ? ' flex items-center justify-center' : ''}`}
        style={{ minHeight: MEDALS_CARD_MIN_BODY_HEIGHT }}
        data-testid="medals-pane-body"
      >
        {isEmpty ? (
          <span className="text-sm text-muted-foreground">{emptyMessage}</span>
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
            {medal.image_url || medal.sprite_sheet ? (
              <MedalIcon
                imageUrl={medal.image_url}
                spriteSheet={medal.sprite_sheet}
                spriteLeft={medal.sprite_left}
                spriteTop={medal.sprite_top}
                spriteWidth={medal.sprite_width}
                spriteHeight={medal.sprite_height}
                label={medal.name}
                size={50}
                className="object-contain"
                style={glow ? { filter: glow } : undefined}
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
// Commendations NATIVES (Halo 5)
// ---------------------------------------------------------------------------

interface MatchNativeCommendationsSectionProps {
  commendations: MatchNativeCommendation[]
  t: MatchViewText
}

// MatchNativeCommendationsSection rend les commendations NATIVES (Halo 5) gagnées
// sur le match. Parité EXACTE avec les citations dérivées d'Infinite
// (MatchCitationsSection) : on réutilise CitationProgressRing — anneau de
// progression (pct = progress_pct), doré + check si is_newly_mastered, et
// l'indicateur de palier tier_index/tier_count (X/Y). Les commendations déjà
// masterisées AVANT le match sont filtrées côté backend (non émises).
// Sans définition (name vide), on dégrade sur un préfixe d'ID court.
export function MatchNativeCommendationsSection({
  commendations,
  t,
}: MatchNativeCommendationsSectionProps) {
  return (
    <PaneCard
      title={t.sectionNativeCommendations}
      isEmpty={commendations.length === 0}
      emptyMessage={t.noNativeCommendations}
    >
      {commendations.map((comm) => {
        const label =
          comm.name && comm.name.trim() !== '' ? comm.name : `#${comm.id.slice(0, 8)}`
        const tierCount = comm.tier_count ?? 0
        const tierIndex = comm.tier_index ?? 0
        const cumulative = comm.cumulative ?? 0
        const nextTarget = comm.next_tier_target ?? 0
        const showTier = tierCount > 0
        const isMastered = citationMastery(comm)
        return (
          <div
            key={comm.id}
            title={label}
            className="flex flex-col items-center gap-1 cursor-default w-[81px]"
          >
            <CitationProgressRing
              pct={comm.progress_pct}
              imageUrl={comm.icon_url ?? undefined}
              isMastered={isMastered}
              isNewlyMastered={comm.is_newly_mastered}
              size={56}
            />
            <span className="text-[12px] text-muted-foreground leading-tight text-center w-full truncate">
              {label}
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
                ×{comm.count}
              </span>
            </div>
            {comm.is_newly_mastered && (
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
        const isMastered = citationMastery(cit)
        return (
          <div
            key={cit.key}
            title={tooltip}
            className="flex flex-col items-center gap-1 cursor-default w-[81px]"
          >
            <CitationProgressRing
              pct={cit.progress_pct}
              imageUrl={cit.image_url ?? undefined}
              isMastered={isMastered}
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
