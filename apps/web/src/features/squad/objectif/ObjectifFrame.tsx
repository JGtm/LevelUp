/**
 * ObjectifFrame — le cadre commun des cartes d'objectif rendues en DOM (lot L3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : même chrome que `ChartCard` (titre et aide ⓘ,
 * contenu centré verticalement, légende en pied de carte, centrée — spec S2), pour que les
 * cartes en DOM et les cartes ECharts de la section se lisent comme une seule famille.
 *
 * ObjectifLegend — la légende de la maquette C3EW : pastilles carrées, paire victoire /
 * défaite, trait 50 % en pointillé vertical, médiane en pointillé horizontal.
 * ObjectifNote — l'encadré en pointillé de la maquette (début en gras, suite en clair).
 */
import type { ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import type { ObjectifNote as NoteText } from './objectifStrings'

export function ObjectifFrame({
  title,
  info,
  legend,
  children,
  testId,
}: {
  title: string
  info: string
  legend?: ReactNode
  children: ReactNode
  testId?: string
}) {
  return (
    <div className="flex h-full min-w-0 flex-col rounded-lg border border-border bg-card" data-testid={testId}>
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        <span className="flex items-center gap-1.5">
          {title}
          <InfoTooltip content={info} />
        </span>
      </div>
      <div className="flex flex-1 flex-col p-3">
        <div className="my-auto flex flex-col gap-3">{children}</div>
      </div>
      {legend != null && (
        <div className="flex-none border-t border-border px-3 py-2" data-testid="objectif-legend">
          {legend}
        </div>
      )}
    </div>
  )
}

export type ObjectifLegendItem =
  | { kind: 'square'; label: string; color: string }
  | { kind: 'pair'; label: string; colors: [string, string] }
  | { kind: 'parity'; label: string; color: string }
  | { kind: 'median'; label: string }

export function ObjectifLegend({ items, ariaLabel }: { items: ObjectifLegendItem[]; ariaLabel: string }) {
  return (
    <ul
      aria-label={ariaLabel}
      className="flex flex-wrap items-center justify-center gap-x-3.5 gap-y-1.5 text-[11.5px] text-muted-foreground"
    >
      {items.map((it) => (
        <li key={it.label} className="inline-flex items-center gap-1.5">
          <LegendMark item={it} />
          <span>{it.label}</span>
        </li>
      ))}
    </ul>
  )
}

function LegendMark({ item }: { item: ObjectifLegendItem }) {
  switch (item.kind) {
    case 'square':
      return <span className="inline-block h-[11px] w-[11px] rounded-[2px]" style={{ backgroundColor: item.color }} aria-hidden />
    case 'pair':
      return (
        <span className="inline-flex" aria-hidden>
          <span className="inline-block h-[11px] w-[11px] rounded-[2px]" style={{ backgroundColor: item.colors[0] }} />
          <span className="-ml-[3px] inline-block h-[11px] w-[11px] rounded-[2px]" style={{ backgroundColor: item.colors[1] }} />
        </span>
      )
    case 'parity':
      return (
        <span
          className="inline-block h-3 w-0.5"
          style={{ backgroundImage: `repeating-linear-gradient(to bottom, ${item.color} 0 3px, transparent 3px 5px)` }}
          aria-hidden
        />
      )
    case 'median':
      return (
        <span
          className="inline-block h-0.5 w-[11px]"
          style={{
            backgroundImage:
              'repeating-linear-gradient(to right, var(--muted-foreground) 0 3px, transparent 3px 5px)', // color-allow: repère neutre de la légende (maquette C3EW)
          }}
          aria-hidden
        />
      )
  }
}

/** L'encadré de note de la maquette (bordure pointillée `info`). */
export function ObjectifNote({ note, testId }: { note: NoteText; testId?: string }) {
  return (
    <div
      className="rounded-lg border border-dashed px-3 py-2 text-xs text-muted-foreground"
      style={{
        borderColor: tokenCssVar('info'),
        backgroundColor: `color-mix(in oklab, ${tokenCssVar('info')} 5%, transparent)`,
      }}
      data-testid={testId}
    >
      {note.lead && <span className="font-medium text-foreground">{note.lead}</span>}
      {note.rest}
    </div>
  )
}
