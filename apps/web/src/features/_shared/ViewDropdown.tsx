/**
 * ViewDropdown — sélecteur single-select de la vue match_context (Tous / Solo /
 * Escouade), aligné visuellement sur ExperienceDropdown (pill + popover radio).
 *
 * Extrait de useLocalFilterBar.tsx : ce dernier exporte un hook, donc ne peut pas
 * héberger un composant sans casser le fast refresh (react-refresh).
 */
import { useEffect, useRef, useState } from 'react'
import type { LocalFilterBarViewLabels, MatchView } from './useLocalFilterBar'

export function ViewDropdown({
  value,
  onChange,
  labels,
}: {
  value: MatchView
  onChange: (next: MatchView) => void
  labels: LocalFilterBarViewLabels
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const options: { value: MatchView; label: string }[] = [
    { value: 'all', label: labels.viewAll },
    { value: 'solo', label: labels.viewSolo },
    { value: 'squad', label: labels.viewSquad },
  ]
  const currentLabel = options.find((o) => o.value === value)?.label ?? labels.viewAll
  const isActive = value !== 'all'

  return (
    <div ref={ref} className="relative" data-testid="relations-view-dropdown">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`rounded border px-2 py-1 text-xs bg-background flex items-center gap-1 whitespace-nowrap transition-colors ${
          isActive ? 'border-primary text-primary' : 'border-input text-muted-foreground hover:border-foreground'
        }`}
      >
        {labels.view} : {currentLabel}
        <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2 4l4 4 4-4" stroke="currentColor" strokeWidth="1.5" fill="none" />
        </svg>
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-1 min-w-[12rem] rounded border border-border bg-popover p-1 shadow-md">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => {
                onChange(opt.value)
                setOpen(false)
              }}
              className={`w-full px-2 py-1 text-left text-sm rounded transition-colors ${
                opt.value === value ? 'bg-accent/60 font-semibold' : 'hover:bg-accent/40'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
