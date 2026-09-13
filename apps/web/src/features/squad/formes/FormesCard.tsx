/**
 * FormesCard.tsx — LE GABARIT D'UNE CARTE du bloc : titre, corps qui défile
 * horizontalement dans SON conteneur, légende centrée, note de pied.
 *
 * TROIS PARTIES, TOUJOURS DANS CET ORDRE (artefact 2ec1b8eb) : la forme, puis ce
 * que veulent dire ses couleurs, puis ce qu'elle dit et ce qu'elle abandonne.
 * Une carte sans note est une carte qu'on croit avoir comprise.
 *
 * Le chrome vient de `SectionCard` — le gabarit unique des cartes de section du
 * dépôt, jamais un cadre réécrit ici (CLAUDE.md n°6).
 */
import type { ReactNode } from 'react'

import { SectionCard } from '@/components/ui/section-card'

import { ENEMY_HATCH, PARITY_INK, UNMEASURED_HATCH } from './colors'
import { RichText } from './forms/RichText'

/** Une entrée de légende : une pastille (aplat, hachure ou trait) et son nom. */
export interface FormesLegendEntry {
  label: string
  /** L'encre de l'aplat. Absente avec `hatch` ou `line`. */
  ink?: string
  /** Hachure de l'adversaire (compté, jamais nommé). */
  hatch?: boolean
  /** Hachure du non mesuré (pas de film décodé). */
  unmeasured?: boolean
  /** Le trait de parité — un repère, pas une donnée. */
  line?: boolean
}

export interface FormesCardProps {
  title: string
  children: ReactNode
  legend?: FormesLegendEntry[]
  /** La note de pied, avec ses passages en gras (`**ainsi**`). */
  note?: string
}

function LegendChip({ entry }: { entry: FormesLegendEntry }) {
  if (entry.line) {
    return (
      <i
        className="inline-block h-3.5 w-[3px] flex-none"
        style={{ backgroundColor: PARITY_INK }}
        aria-hidden="true"
      />
    )
  }
  if (entry.hatch || entry.unmeasured) {
    return (
      <i
        className="inline-block h-2.5 w-2.5 flex-none border border-muted-foreground"
        style={entry.hatch ? ENEMY_HATCH : UNMEASURED_HATCH}
        aria-hidden="true"
      />
    )
  }
  return (
    <i
      className="inline-block h-2.5 w-2.5 flex-none"
      style={{ backgroundColor: entry.ink }}
      aria-hidden="true"
    />
  )
}

export function FormesCard({ title, children, legend, note }: FormesCardProps) {
  return (
    <SectionCard title={title} label={title}>
      <div className="overflow-x-auto px-3 pb-3 pt-4">{children}</div>
      {legend != null && legend.length > 0 && (
        <div className="flex flex-wrap justify-center gap-x-5 gap-y-1.5 border-t border-border px-3 py-2 text-3xs text-muted-foreground">
          {legend.map((entry) => (
            <span key={entry.label} className="inline-flex items-center gap-1.5">
              <LegendChip entry={entry} />
              {entry.label}
            </span>
          ))}
        </div>
      )}
      {note != null && note !== '' && (
        <div className="border-t border-dashed border-border px-3 pb-2.5 pt-2 text-3xs text-muted-foreground">
          <RichText text={note} />
        </div>
      )}
    </SectionCard>
  )
}

/** Le sous-titre d'une sous-forme, quand une carte en porte deux. */
export function FormesSubtitle({ children }: { children: string }) {
  return (
    <div className="mb-2 mt-5 text-3xs font-semibold uppercase tracking-wider text-muted-foreground first:mt-0">
      {children}
    </div>
  )
}
