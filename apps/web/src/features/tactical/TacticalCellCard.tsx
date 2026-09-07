/**
 * TacticalCellCard — la carte « Cellule sélectionnée » de la vue d'analyse (item 5.6).
 *
 * SEULEMENT CE QUE LE RASTER PUBLIE : la valeur et son unité, et le nombre de matchs
 * CONTRIBUTEURS quand la cellule le porte (`CelluleTactique.matchs`). La liste des
 * matchs eux-mêmes et le lien `?frame=` vers le rejeu restent HORS PÉRIMÈTRE : le
 * contrat ne publie aucun identifiant de match par cellule (seulement un compte), et le
 * lien dépend du lecteur (`playbackStore`), reporté au lot D avec l'item 5.5 — cf. plan
 * §Phase 5.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import type { CelluleTactique } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { unitForQuestion, type TacticalQuestion } from './tacticalView.logic'

export interface TacticalCellCardProps {
  t: TacticalText
  locale: Locale
  question: TacticalQuestion
  cellule: CelluleTactique | null
}

export function TacticalCellCard({ t, locale, question, cellule }: TacticalCellCardProps) {
  const unite = unitForQuestion(t, question)
  const numFmt = new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 })

  return (
    <SectionCard title={t.cellTitle} label={t.cellTitle}>
      <div className="p-3" data-testid="tactical-cell-card">
        {!cellule ? (
          <EmptyStateNotice title={t.cellPlaceholder} description={t.cellPlaceholderDescription} />
        ) : (
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
        )}
      </div>
    </SectionCard>
  )
}
