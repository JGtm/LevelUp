/**
 * TacticalCellCard — la carte « Cellule sélectionnée » de la vue d'analyse (item 5.6,
 * complétée par le lot M1 — Tactique S.1, « voir dans le rejeu »).
 *
 * DEUX BLOCS : la valeur agrégée du raster (toujours servie avec la cellule), PUIS les
 * CONTRIBUTIONS individuelles (`useTacticalCellule`, lancée uniquement quand une cellule
 * est sélectionnée) — chacune un lien vers le rejeu du match, à l'instant concerné.
 *
 * OWNERSHIP (ADR 0029) : les contributions reçues sont déjà filtrées côté serveur — cette
 * carte ne fait qu'afficher `matchs_non_ouvrables` en pied de liste (0 → rien, jamais un
 * zéro qui inviterait à le lire comme une absence de restriction).
 *
 * `?frame=` EST UNE APPROXIMATION POUR QUATRE QUESTIONS SUR SIX — voir la doc de
 * `instantToFrame` (tacticalView.logic.ts) : le lien ouvre TOUJOURS le bon match, à un
 * instant exact pour « temps »/« routes », approché pour les quatre autres (décalage
 * d'horloge par match non publié — découverte consignée, lot M1).
 */
import { useRouter } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { formatClock } from '@/lib/replay/replayLogic'
import { useTitleSlug } from '@/lib/title-routing'

import type { TacticalText } from './i18n'
import { instantToFrame, unitForQuestion, type TacticalQuestion } from './tacticalView.logic'

export interface TacticalCellCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  question: TacticalQuestion
  cellule: CelluleTactique | null
  /** Contributions de la cellule sélectionnée. `null` = aucune cellule choisie (la requête
   *  n'est alors pas lancée, cf. `useTacticalCellule`) OU chargement en cours. */
  contributions: readonly TacticalContribution[] | null
  contributionsLoading: boolean
  matchsNonOuvrables: number
}

export function TacticalCellCard({
  t,
  locale,
  playerSlug,
  question,
  cellule,
  contributions,
  contributionsLoading,
  matchsNonOuvrables,
}: TacticalCellCardProps) {
  const unite = unitForQuestion(t, question)
  const numFmt = new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 })

  return (
    <SectionCard title={t.cellTitle} label={t.cellTitle}>
      <div className="flex flex-col gap-3 p-3" data-testid="tactical-cell-card">
        {!cellule ? (
          <EmptyStateNotice title={t.cellPlaceholder} description={t.cellPlaceholderDescription} />
        ) : (
          <>
            <div className="flex flex-col gap-1">
              <p className="flex items-baseline gap-2" data-testid="tactical-cell-value">
                <span className="text-2xl font-semibold text-foreground">
                  {numFmt.format(cellule.valeur)}
                </span>
                <span className="text-sm text-muted-foreground">{unite}</span>
              </p>
              {cellule.matchs > 0 && (
                <p className="text-xs text-muted-foreground">{t.cellMatches(cellule.matchs)}</p>
              )}
            </div>
            <TacticalCellContributions
              t={t}
              locale={locale}
              playerSlug={playerSlug}
              contributions={contributions}
              loading={contributionsLoading}
              matchsNonOuvrables={matchsNonOuvrables}
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}

interface TacticalCellContributionsProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  contributions: readonly TacticalContribution[] | null
  loading: boolean
  matchsNonOuvrables: number
}

function TacticalCellContributions({
  t,
  locale,
  playerSlug,
  contributions,
  loading,
  matchsNonOuvrables,
}: TacticalCellContributionsProps) {
  const titleSlug = useTitleSlug()
  const router = useRouter()
  const dateFmt = new Intl.DateTimeFormat(intlLocale(locale), { dateStyle: 'medium' })

  return (
    <div className="flex flex-col gap-2 border-t border-border pt-2" data-testid="tactical-cell-contributions">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t.cellContributionsTitle}
      </h3>
      {loading && <p className="text-xs text-muted-foreground">{t.cellContributionsLoading}</p>}
      {!loading && (contributions === null || contributions.length === 0) && (
        <p className="text-xs text-muted-foreground">{t.cellContributionsEmpty}</p>
      )}
      {!loading && contributions !== null && contributions.length > 0 && (
        <ul className="flex flex-col gap-1">
          {contributions.map((c, i) => {
            const date = dateFmt.format(new Date(c.match_started_at))
            const instant = formatClock(c.instant_ms)
            const frame = instantToFrame(c.instant_ms)
            const base = router.buildLocation({
              to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
              params: { titleSlug, playerSlug, matchId: c.match_id },
            }).href
            const href = `${base}${base.includes('?') ? '&' : '?'}frame=${frame}`
            return (
              <li key={`${c.match_id}-${c.instant_ms}-${i}`}>
                <a
                  href={href}
                  className="text-xs text-primary hover:underline"
                  data-testid="tactical-cell-contribution-link"
                >
                  {t.cellContributionLabel(date, instant)}
                </a>
              </li>
            )
          })}
        </ul>
      )}
      {matchsNonOuvrables > 0 && (
        <p className="text-2xs text-muted-foreground" data-testid="tactical-cell-not-openable">
          {t.cellFooterNotOpenable(matchsNonOuvrables)}
        </p>
      )}
    </div>
  )
}
