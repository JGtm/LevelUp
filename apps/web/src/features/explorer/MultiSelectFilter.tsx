/**
 * MultiSelectFilter — dropdown bouton + checkboxes pour sélection multiple.
 *
 * Utilisé dans ExplorerPage pour tous les filtres multi-valeurs (exp type,
 * playlist, mode, carte, perf-tier, outcome, skill-tier).
 */
import { useEffect, useRef, useState } from 'react'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest } from '@/lib/i18n/generated/explorer'
import { useAppShellStore } from '@/stores/appShellStore'

export interface MultiSelectOption {
  value: string
  label: string
  /** CSS color (token var) optionnelle pour pastille devant le label. */
  swatch?: string
  /** Si true, l'option est désactivée (ex: skill tier sans contexte ranked). */
  disabled?: boolean
}

interface Props {
  options: MultiSelectOption[]
  selected: Set<string>
  toggle: (value: string) => void
  placeholder: string
  /** Si true, affiche le composant même si options vide (utile en chargement). */
  alwaysShow?: boolean
  /** Tooltip optionnel sur le bouton (ex: explication désactivation). */
  title?: string
  /** Si true, désactive le bouton entier. */
  disabled?: boolean
}

export function MultiSelectFilter({
  options,
  selected,
  toggle,
  placeholder,
  alwaysShow,
  title,
  disabled,
}: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const fmtCount = (n: number) =>
    formatMessage(explorerManifest, 'explorer.filters.selected_count', locale, { n })

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  if (options.length === 0 && !alwaysShow) return null

  const label = selected.size === 0 ? placeholder : `${placeholder} · ${fmtCount(selected.size)}`

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => !disabled && setOpen((o) => !o)}
        disabled={disabled}
        title={title}
        className={`rounded border px-2 py-1 text-sm bg-background flex items-center gap-1 whitespace-nowrap transition-colors ${
          disabled
            ? 'cursor-not-allowed opacity-40 border-border text-muted-foreground'
            : selected.size > 0
              ? 'border-primary text-primary'
              : 'border-input text-muted-foreground hover:border-foreground'
        }`}
      >
        {label}
        <span className="text-xs opacity-60">▾</span>
      </button>
      {open && !disabled && (
        <div className="absolute z-50 mt-1 w-max min-w-full rounded-md border border-border bg-background shadow-lg max-h-60 overflow-y-auto">
          {options.map((opt) => (
            <label
              key={opt.value}
              className={`flex items-center gap-2 px-3 py-1.5 text-sm ${
                opt.disabled
                  ? 'cursor-not-allowed opacity-40'
                  : 'cursor-pointer hover:bg-primary/10'
              }`}
            >
              <input
                type="checkbox"
                checked={selected.has(opt.value)}
                onChange={() => !opt.disabled && toggle(opt.value)}
                disabled={opt.disabled}
                className="rounded accent-primary"
              />
              {opt.swatch && (
                <span
                  className="inline-block h-2 w-2 rounded-full"
                  style={{ backgroundColor: opt.swatch }}
                  aria-hidden
                />
              )}
              <span>{opt.label}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
