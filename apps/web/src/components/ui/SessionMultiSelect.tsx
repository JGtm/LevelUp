/**
 * SessionMultiSelect — sélecteur multi-sessions avec fuzzy search et mini-filtre date.
 *
 * Composant contrôlé partagé Squad + Stats. La sélection est propagée au parent
 * via onChange uniquement sur "Valider" (validation différée).
 * Le filtre de date interne filtre UNIQUEMENT la liste visible — pas les matchs analysés.
 */
import { useRef, useState, useEffect } from 'react'
import type { SessionLabelEntry } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'
import { intlLocale as toIntlLocale } from '@/lib/formatters'

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
  reset: string
  search: string
  empty: string
}

function getTexts(locale: ManifestLocale): Texts {
  const isFr = locale === 'fr'
  return {
    all:        isFr ? 'Toutes les sessions'    : 'All sessions',
    count:      (n) => isFr ? `${n} session${n > 1 ? 's' : ''}` : `${n} session${n !== 1 ? 's' : ''}`,
    // Renommé "Filtrer la liste" → "Trouver dans la liste" pour clarifier
    // qu'il filtre seulement la liste visible (vs PeriodePill qui filtre les
    // matchs analysés). Plan smart-filter-counts, Phase 3.
    filterList: isFr ? 'Trouver dans la liste'  : 'Find in list',
    from:       isFr ? 'Du'                     : 'From',
    to:         isFr ? 'Au'                     : 'To',
    validate:   isFr ? 'Valider'                : 'Apply',
    selectAll:  isFr ? 'Tout sélectionner'      : 'Select all',
    deselectAll:isFr ? 'Tout désélectionner'    : 'Deselect all',
    reset:      isFr ? 'Réinitialiser'          : 'Reset',
    search:     isFr ? 'Rechercher…'            : 'Search…',
    empty:      isFr ? 'Aucune session'         : 'No sessions',
  }
}

// ─── Props ───────────────────────────────────────────────────────────────────

export interface SessionMultiSelectProps {
  sessions: SessionLabelEntry[]
  selected: string[]
  onChange: (labels: string[]) => void
  locale: ManifestLocale
  placeholder?: string
  /** Surcharge la classe CSS du bouton déclencheur (ex: taille dans une barre compacte). */
  triggerClassName?: string
  /** Si fourni, retourne le nombre de matchs de la session avec les filtres
   *  actifs. Les sessions retournant 0 sont masquées (sauf si déjà sélectionnées,
   *  pour ne pas les faire disparaître brutalement). Plan smart-filter-counts. */
  getMatchCount?: (label: string) => number | undefined
}

// ─── Composant ───────────────────────────────────────────────────────────────

export function SessionMultiSelect({
  sessions,
  selected,
  onChange,
  locale,
  placeholder,
  triggerClassName,
  getMatchCount,
}: SessionMultiSelectProps) {
  const t = getTexts(locale)
  const intlLocale = toIntlLocale(locale)

  const [isOpen, setIsOpen]     = useState(false)
  const [query, setQuery]       = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo]     = useState('')
  const [pending, setPending]   = useState<string[]>(selected)
  const containerRef = useRef<HTMLDivElement>(null)

  // `pending` est sync sur `selected` au moment de l'ouverture du panel
  // (cf. `handleOpen` ci-dessous). Pas besoin de useEffect : sync manuelle
  // dans l'event handler évite les cascading renders flaggés par
  // react-hooks/set-state-in-effect (équivalent React Compiler).

  // Fermeture click-outside + Escape (alignement UX avec les autres pills
  // de la FilterOmnibar — Escape ferme le popover ouvert).
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setIsOpen(false)
    }
    document.addEventListener('mousedown', handleMouseDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleMouseDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [])

  // Sessions filtrées par recherche + plage de dates (filtre la liste uniquement).
  // Si getMatchCount est fourni, masquer les sessions retournant 0 matchs (sauf
  // si déjà sélectionnée — pour ne pas les faire disparaître brutalement).
  const filtered = sessions.filter((s) => {
    if (query && !s.label.toLowerCase().includes(query.toLowerCase())) return false
    if (dateFrom && new Date(s.ended_at) < new Date(dateFrom)) return false
    if (dateTo   && new Date(s.started_at) > new Date(dateTo))  return false
    if (getMatchCount) {
      const c = getMatchCount(s.label)
      if (c === 0 && !selected.includes(s.label)) return false
    }
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

  function resetPending() {
    setPending([])
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
        className={triggerClassName ?? 'flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-1.5 text-sm hover:bg-accent whitespace-nowrap'}
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
            <div className="text-3xs font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">
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

          {/* Tout sélectionner / désélectionner (visibles) + Réinitialiser (toute la sélection) */}
          {filtered.length > 0 && (
            <div className="flex items-center justify-between px-3 py-1.5 border-b border-border/50">
              <button
                type="button"
                onClick={toggleAllFiltered}
                className="text-xs text-primary hover:underline"
              >
                {allFilteredSelected ? t.deselectAll : t.selectAll}
              </button>
              {pending.length > 0 && (
                <button
                  type="button"
                  onClick={resetPending}
                  className="text-xs text-muted-foreground hover:text-foreground hover:underline"
                >
                  {t.reset}
                </button>
              )}
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
              const count = getMatchCount?.(s.label)
              // Le label backend embarque un suffix " (N)" figé au sync (count
              // brut sans contexte solo/squad ni cascade). Si le caller fournit
              // un count dynamique on retire ce suffix pour éviter "(13) 6".
              const displayLabel =
                count !== undefined ? s.label.replace(/\s*\(\d+\)\s*$/, '') : s.label
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
                  <div className="min-w-0 flex-1">
                    <div className="text-sm truncate">{displayLabel}</div>
                    <div className="text-xs text-muted-foreground">{dateLabel}</div>
                  </div>
                  {count !== undefined && (
                    <span className="shrink-0 self-center text-2xs tabular-nums text-muted-foreground">
                      {count}
                    </span>
                  )}
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
