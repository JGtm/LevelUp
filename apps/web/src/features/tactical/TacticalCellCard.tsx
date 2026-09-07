/**
 * TacticalCellCard.tsx — Carte de section pour la cellule sélectionnée.
 *
 * Affiche : valeur de la cellule, nombre de matchs contributeurs, liste de drilldown.
 */
import { useMemo } from 'react'
import { getTacticalText, type TacticalLocale } from './i18n'
import type { QuestionType } from './tacticalView.logic'
import { getQuestionUnit } from './tacticalView.logic'

export interface CellSample {
  date: string
  outcome: 'victoire' | 'défaite'
  time: string
  frame?: number
}

export interface TacticalCellCardProps {
  locale: TacticalLocale
  question: QuestionType
  selectedCell?: {
    value: number
    matchCount: number
    samples: CellSample[]
  } | null
  onOpenFrame?: (frame: number) => void
}

export function TacticalCellCard({
  locale,
  question,
  selectedCell,
  onOpenFrame,
}: TacticalCellCardProps) {
  const text = getTacticalText(locale)
  const unit = useMemo(() => getQuestionUnit(question, locale), [question, locale])

  const displayValue = useMemo(() => {
    if (!selectedCell) return '—'
    return `${Math.abs(selectedCell.value).toFixed(2)} ${unit}`.trim()
  }, [selectedCell, unit])

  const subtitle = useMemo(() => {
    if (!selectedCell) return text.cellNoSelection
    const count = selectedCell.matchCount
    const word = locale === 'fr' ? 'matchs distincts' : 'distinct matches'
    return `${count} ${word} ont alimenté cette cellule.`
  }, [selectedCell, text, locale])

  const moreCount = useMemo(() => {
    if (!selectedCell) return 0
    return Math.max(0, selectedCell.matchCount - (selectedCell.samples?.length ?? 0))
  }, [selectedCell])

  return (
    <div className="rounded-md border border-border bg-card">
      <h3 className="border-b border-border px-3 py-2 text-sm font-medium">{text.selectedCellTitle}</h3>

      <div className="p-3">
        <p className="font-mono text-xl font-semibold">{displayValue}</p>
        <p className="text-xs text-muted-foreground mt-2">{subtitle}</p>

        {selectedCell && selectedCell.samples && selectedCell.samples.length > 0 && (
          <div className="mt-4 space-y-2">
            {selectedCell.samples.map((sample, i) => (
              <div
                key={i}
                className="flex items-center justify-between gap-2 text-xs p-2 rounded bg-muted/30 border border-border/50"
              >
                <span className="text-muted-foreground">{sample.date}</span>
                <span className={sample.outcome === 'victoire' ? 'text-outcome-win' : 'text-outcome-loss'}>
                  {locale === 'fr' ? (sample.outcome === 'victoire' ? 'Victoire' : 'Défaite') : (sample.outcome === 'victoire' ? 'Win' : 'Loss')}
                </span>
                <span className="text-muted-foreground">{sample.time}</span>
                {sample.frame !== undefined && onOpenFrame && (
                  <button
                    onClick={() => onOpenFrame(sample.frame!)}
                    className="text-chart-1 hover:underline text-xs"
                  >
                    {locale === 'fr' ? 'Ouvrir' : 'Open'}
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        {moreCount > 0 && (
          <p className="mt-3 text-xs text-muted-foreground">
            {locale === 'fr'
              ? `${moreCount} autres matchs comptent dans la cellule mais ne te sont pas ouverts : le calque est anonyme, la liste ne l'est pas.`
              : `${moreCount} other matches count in the cell but are not opened for you: the layer is anonymous, the list is not.`}
          </p>
        )}
      </div>
    </div>
  )
}
