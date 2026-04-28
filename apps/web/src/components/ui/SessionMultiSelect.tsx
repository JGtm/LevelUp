/**
 * SessionMultiSelect — sélecteur multi-sessions avec fuzzy search et mini-filtre date.
 *
 * Composant contrôlé partagé Squad + Stats. La sélection est propagée au parent
 * via onChange uniquement sur "Valider" (validation différée).
 * Le filtre de date interne filtre UNIQUEMENT la liste visible — pas les matchs analysés.
 */
import { useRef, useState, useEffect } from 'react'
import type { SessionLabelEntry } from '@/lib/api/types'

// ─── Types ───────────────────────────────────────────────────────────────────

interface Texts {
  all: string
  count: (n: number) => string
  filterList: string
  from: string
  to: string
  validate: string
  selectAll: string
  deselectAll: string
  search: string
  empty: string
}

function getTexts(locale: string): Texts {
  const isFr = locale === 'fr'
  return {
    all:        isFr ? 'Toutes les sessions'    : 'All sessions',
    count:      (n) => isFr ? `${n} session${n > 1 ? 's' : ''}` : `${n} session${n !== 1 ? 's' : ''}`,
    filterList: isFr ? 'Filtrer la liste'       : 'Filter list',
    from:       isFr ? 'Du'                     : 'From',
    to:         isFr ? 'Au'                     : 'To',
    validate:   isFr ? 'Valider'                : 'Apply',
    selectAll:  isFr ? 'Tout sélectionner'      : 'Select all',
    deselectAll:isFr ? 'Tout désélectionner'    : 'Deselect all',
    search:     isFr ? 'Rechercher…'            : 'Search…',
    empty:      isFr ? 'Aucune session'         : 'No sessions',
  }
}

// ─── Props ───────────────────────────────────────────────────────────────────

export interface SessionMultiSelectProps {
  sessions: SessionLabelEntry[]
  selected: string[]
  onChange: (labels: string[]) => void
  locale: string
  placeholder?: string
}

// ─── Composant ───────────────────────────────────────────────────────────────

export function SessionMultiSelect({
  sessions,
  selected,
  onChange,
  locale,
  placeholder,
}: SessionMultiSelectProps) {
  const t = getTexts(locale)
  const intlLocale = locale === 'fr' ? 'fr-FR' : 'en-US'

  const [isOpen, setIsOpen]     = useState(false)
  const [query, setQuery]       = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo]     = useState('')
  const [pending, setPending]   = useState<string[]>(selected)
  const containerRef = useRef<HTMLDivElement>(null)

  // Sync pending ← selected quand le panel est fermé ou que selected change.
  useEffect(() => {
    if (!isOpen) setPending(selected)
  }, [selected, isOpen])

  // Fermeture click-outside.
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleMouseDown)
    return () => document.removeEventListener('mousedown', handleMouseDown)
  }, [])

  // Sessions filtrées par recherche + plage de dates (filtre la liste uniquement).
  const filtered = sessions.filter((s) => {
    if (query && !s.label.toLowerCase().includes(query.toLowerCase())) return false
    if (dateFrom && new Date(s.ended_at) < new Date(dateFrom)) return false
    if (dateTo   && new Date(s.started_at) > new Date(dateTo))  return false
    return true
  })

  const allFilteredSelected =
    filtered.length > 0 && filtered.every((s) => pending.includes(s.label))

  function togglePending(label: string) {
    setPending((prev) =>
      prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label],
    )
  }

  function toggleAllFiltered() {
    if (allFilteredSelected) {
      setPending((prev) => prev.filter((l) => !filtered.some((s) => s.label === l)))
    } else {
      const toAdd = filtered.map((s) => s.label).filter((l) => !pending.includes(l))
      setPending((prev) => [...prev, ...toAdd])
    }
  }

  function handleOpen() {
    setPending(selected)
    setIsOpen(true)
  }

  function handleValidate() {
    onChange(pending)
    setIsOpen(false)
  }

  const summaryLabel =
    selected.length === 0 ? (placeholder ?? t.all) : t.count(selected.length)

  return (
    <div ref={containerRef} className="relative">
      {/* Bouton déclencheur */}
      <button
        type="button"
        onClick={handleOpen}
        className="flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-1.5 text-sm hover:bg-accent whitespace-nowrap"
      >
        <span>{summaryLabel}</span>
        <span className="text-muted-foreground text-xs">▾</span>
      </button>

      {/* Panneau dropdown */}
      {isOpen && (
        <div className="absolute z-50 mt-1 w-80 rounded-md border border-border bg-popover shadow-md">
          {/* Recherche textuelle */}
          <div className="p-2 border-b border-border/50">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t.search}
              className="w-full rounded-md border border-input bg-background px-2 py-1.5 text-sm outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          {/* Mini filtre date (ne filtre QUE la liste visible) */}
          <div className="p-2 border-b border-border/50">
            <div className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">
              {t.filterList}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground shrink-0">{t.from}</span>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="flex-1 min-w-0 rounded border border-input bg-background px-1.5 py-1 text-xs outline-none focus:ring-1 focus:ring-ring"
              />
              <span className="text-xs text-muted-foreground shrink-0">{t.to}</span>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="flex-1 min-w-0 rounded border border-input bg-background px-1.5 py-1 text-xs outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          </div>

          {/* Tout sélectionner / désélectionner */}
          {filtered.length > 0 && (
            <div className="px-3 py-1.5 border-b border-border/50">
              <button
                type="button"
                onClick={toggleAllFiltered}
                className="text-xs text-primary hover:underline"
              >
                {allFilteredSelected ? t.deselectAll : t.selectAll}
              </button>
            </div>
          )}

          {/* Liste des sessions */}
          <div className="max-h-52 overflow-y-auto">
            {filtered.length === 0 && (
              <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                {t.empty}
              </div>
            )}
            {filtered.map((s) => {
              const isChecked = pending.includes(s.label)
              const dateLabel = new Date(s.started_at).toLocaleDateString(intlLocale)
              return (
                <label
                  key={s.label}
                  className="flex items-start gap-2.5 px-3 py-2 hover:bg-accent cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={isChecked}
                    onChange={() => togglePending(s.label)}
                    className="mt-0.5 shrink-0"
                  />
                  <div className="min-w-0">
                    <div className="text-sm truncate">{s.label}</div>
                    <div className="text-xs text-muted-foreground">{dateLabel}</div>
                  </div>
                </label>
              )
            })}
          </div>

          {/* Bouton Valider */}
          <div className="p-2 border-t border-border/50">
            <button
              type="button"
              onClick={handleValidate}
              className="w-full rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              {t.validate}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
