/**
 * ExperienceDropdown — single-select dropdown (Tous / Classé / Non classé).
 *
 * Composant partagé extrait de CareerHighlightMatchesSection. Réutilisé par
 * la page Synthesis (filtres cascade locaux). Visuellement aligné sur
 * MultiSelectFilter (pill button + popover) mais avec boutons radio.
 *
 * Les labels sont passés en props pour rester i18n-agnostique : chaque
 * consommateur fournit son propre manifest (career / synthesis / autre).
 */
import { useRef, useState } from 'react'
import { useDismissable } from '@/lib/hooks/useDismissable'

export type Experience = 'all' | 'ranked' | 'unranked'

export interface ExperienceCount {
  value: Experience
  count: number
}

interface ExperienceDropdownProps {
  value: Experience
  onChange: (next: Experience) => void
  counts: ExperienceCount[]
  labels: { placeholder: string; all: string; ranked: string; unranked: string }
  /** Taille de police compacte (text-xs) du trigger — pour aligner sur les pills text-xs. */
  dense?: boolean
}

export function ExperienceDropdown({ value, onChange, counts, labels, dense }: ExperienceDropdownProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)

  // Échap et un clic hors du menu le ferment ; Échap rend le focus au bouton, le clic dehors le laisse où l'on a cliqué.
  const ref = useDismissable(open, (reason) => {
    setOpen(false)
    if (reason === 'escape') triggerRef.current?.focus()
  })

  const countMap = new Map<string, number>()
  for (const c of counts) countMap.set(c.value, c.count)

  const currentLabel = value === 'ranked' ? labels.ranked : value === 'unranked' ? labels.unranked : labels.all
  const isActive = value !== 'all'

  const options: { value: Experience; label: string }[] = [
    { value: 'all', label: labels.all },
    { value: 'ranked', label: labels.ranked },
    { value: 'unranked', label: labels.unranked },
  ]

  return (
    <div ref={ref} className="relative">
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className={`rounded border px-2 py-1 ${dense ? 'text-xs' : 'text-sm'} bg-background flex items-center gap-1 whitespace-nowrap transition-colors ${
          isActive
            ? 'border-primary text-primary'
            : 'border-input text-muted-foreground hover:border-foreground'
        }`}
      >
        {labels.placeholder} : {currentLabel}
        <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2 4l4 4 4-4" stroke="currentColor" strokeWidth="1.5" fill="none" />
        </svg>
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-1 min-w-[16rem] rounded border border-border bg-popover p-1 shadow-md">
          {options.map((opt) => {
            const c = countMap.get(opt.value) ?? 0
            const selected = opt.value === value
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => {
                  onChange(opt.value)
                  setOpen(false)
                }}
                className={`w-full px-2 py-1 text-left text-sm rounded flex items-center justify-between transition-colors ${
                  selected ? 'bg-accent/60 font-semibold' : 'hover:bg-accent/40'
                }`}
              >
                <span>{opt.label}</span>
                <span className="font-mono text-xs text-muted-foreground">{c}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
