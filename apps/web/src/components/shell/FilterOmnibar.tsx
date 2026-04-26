/**
 * FilterOmnibar — bandeau de filtres en pills (h-9) avec popovers ciblés.
 *
 * Remplace l'ancien FilterPanel expandable. Chaque dimension de filtre
 * (Session, Période, Filtres avancés) est une pill cliquable qui ouvre
 * un popover focused. Un seul popover ouvert à la fois.
 *
 * Ordre d'affichage :
 *   [Session ▾]  [Période ▾]  [Filtres ▾]  ── [N matchs]  [↺ Réinitialiser]
 *
 * Session ↔ Période sont mutuellement exclusifs (auto-derive dans le store) :
 * sélectionner une session vide la période et inversement. La cascade
 * (playlists/modes/cartes/types) est toujours orthogonale.
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { CascadeInput, PeriodInput, SessionsInput } from '@/lib/api/types'

// ─── Helpers période (réutilisés depuis l'ancien FilterPanel) ────────────────

const PERIOD_PRESETS = [
  { id: '7d', label: '7 jours', days: 7 },
  { id: '30d', label: '30 jours', days: 30 },
  { id: '90d', label: '90 jours', days: 90 },
  { id: 'all', label: 'Toutes', days: 0 },
] as const

type PresetId = (typeof PERIOD_PRESETS)[number]['id'] | 'custom'

function isoDate(d: Date): string {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function presetPeriod(days: number): PeriodInput | null {
  if (days <= 0) return null
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - days)
  return { start_date: isoDate(start), end_date: isoDate(end) }
}

function detectActivePreset(period: PeriodInput | undefined): PresetId {
  if (!period) return 'all'
  if (!period.start_date && !period.end_date) return 'all'
  for (const p of PERIOD_PRESETS) {
    const expected = presetPeriod(p.days)
    if (!expected) continue
    if (
      period.start_date === expected.start_date &&
      period.end_date === expected.end_date
    ) {
      return p.id
    }
  }
  return 'custom'
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
function isUUIDLabel(label: string): boolean {
  return UUID_RE.test(label.trim())
}

// ─── Hook : popover ouvert/fermé avec click-outside + Escape ─────────────────

function useDismissable(open: boolean, onClose: () => void) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    function onDocClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, onClose])
  return ref
}

// ─── Composant principal ──────────────────────────────────────────────────────

type ActivePopover = 'session' | 'periode' | 'filtres' | null

export function FilterOmnibar() {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setPeriod = useGlobalFilterStore((s) => s.setPeriod)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const setCascade = useGlobalFilterStore((s) => s.setCascade)
  const resetFilters = useGlobalFilterStore((s) => s.resetFilters)

  const [active, setActive] = useState<ActivePopover>(null)
  const togglePopover = (which: ActivePopover) =>
    setActive((cur) => (cur === which ? null : which))
  const closeAll = () => setActive(null)

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const cascade = (filterContext.cascade ?? {}) as CascadeInput
  const period = filterContext.period
  const pickedId = filterContext.sessions?.picked_sessions?.[0] ?? null
  const currentSession = pickedId
    ? allSessions.find((s) => s.session_id === pickedId)
    : null

  // Comptes pour les pills
  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const)
    .reduce((n, k) => n + ((cascade[k] as string[] | undefined)?.length ?? 0), 0)
  const hasPeriod = !!(period?.start_date || period?.end_date)
  const hasActiveFilters = !!pickedId || hasPeriod || cascadeCount > 0
  const totalAfter = resolvedContext?.counts?.total_matches_after_filters ?? null

  // Available options (cascade) — masque les UUID bruts non résolus
  const rawAvailable = resolvedContext?.available_options
  const available = useMemo(() => {
    if (!rawAvailable) return undefined
    const filterUUIDs = (opts: { label: string; value: string }[]) =>
      opts.filter((o) => !isUUIDLabel(o.label))
    return {
      playlists: filterUUIDs(rawAvailable.playlists),
      modes: filterUUIDs(rawAvailable.modes),
      maps: filterUUIDs(rawAvailable.maps),
      experience_types: filterUUIDs(rawAvailable.experience_types),
    }
  }, [rawAvailable])

  return (
    <div
      className="flex h-9 items-center gap-2 overflow-visible px-4"
      role="toolbar"
      aria-label="Filtres"
    >
      {allSessions.length > 0 && (
        <SessionPill
          open={active === 'session'}
          onToggle={() => togglePopover('session')}
          onClose={closeAll}
          currentLabel={currentSession?.label ?? null}
          allSessions={allSessions}
          pickedId={pickedId}
          onPick={(id: string | null) => {
            setSessions({
              ...(filterContext.sessions ?? {}),
              picked_sessions: id ? [id] : [],
            } as SessionsInput)
            closeAll()
          }}
        />
      )}

      <PeriodePill
        open={active === 'periode'}
        onToggle={() => togglePopover('periode')}
        onClose={closeAll}
        period={period}
        onSetPeriod={(p) => setPeriod(p)}
      />

      {available && (
        <FiltresPill
          open={active === 'filtres'}
          onToggle={() => togglePopover('filtres')}
          onClose={closeAll}
          available={available}
          cascade={cascade}
          cascadeCount={cascadeCount}
          onSetCascade={(c) => setCascade(c)}
        />
      )}

      <div className="flex-1" />

      {totalAfter !== null && (
        <span
          className="shrink-0 text-xs text-muted-foreground"
          aria-live="polite"
          title="Nombre de matchs correspondant aux filtres actifs"
        >
          {totalAfter} match{totalAfter > 1 ? 's' : ''}
        </span>
      )}

      {hasActiveFilters && (
        <button
          type="button"
          onClick={resetFilters}
          className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
          title="Réinitialiser tous les filtres"
        >
          ↺ Réinitialiser
        </button>
      )}
    </div>
  )
}

// ─── Pill : Session ──────────────────────────────────────────────────────────

interface SessionPillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  currentLabel: string | null
  allSessions: { session_id: string; label: string; match_count: number; is_squad: boolean }[]
  pickedId: string | null
  onPick: (id: string | null) => void
}

function SessionPill({
  open,
  onToggle,
  onClose,
  currentLabel,
  allSessions,
  pickedId,
  onPick,
}: SessionPillProps) {
  const ref = useDismissable(open, onClose)
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!search.trim()) return allSessions
    const q = search.toLowerCase()
    return allSessions.filter((s) => s.label.toLowerCase().includes(q))
  }, [allSessions, search])

  const triggerLabel = currentLabel ? `Session : ${currentLabel}` : 'Toutes les sessions'

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={[
          'flex max-w-[18rem] items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          pickedId
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span className="truncate">{triggerLabel}</span>
        <span className="text-[10px] opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="listbox"
          aria-label="Sessions"
          className="absolute left-0 top-full z-40 mt-1 flex w-80 flex-col rounded-md border border-border bg-background shadow-lg"
        >
          <div className="border-b border-border p-2">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Rechercher une session…"
              className="w-full rounded border border-input bg-background px-2 py-1 text-xs focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              autoFocus
            />
          </div>
          <div className="max-h-72 overflow-y-auto py-1">
            <button
              type="button"
              onClick={() => onPick(null)}
              className={[
                'flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs transition-colors hover:bg-muted',
                pickedId === null ? 'bg-primary/10 text-primary' : 'text-foreground',
              ].join(' ')}
            >
              <span className="font-medium">Toutes les sessions</span>
              <span className="text-[10px] text-muted-foreground">{allSessions.length}</span>
            </button>
            {filtered.length === 0 ? (
              <p className="px-3 py-4 text-center text-xs text-muted-foreground">
                Aucune session ne correspond.
              </p>
            ) : (
              filtered.map((s) => {
                const isPicked = s.session_id === pickedId
                return (
                  <button
                    key={s.session_id}
                    type="button"
                    onClick={() => onPick(s.session_id)}
                    className={[
                      'flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs transition-colors hover:bg-muted',
                      isPicked ? 'bg-primary/10 text-primary' : 'text-foreground',
                    ].join(' ')}
                  >
                    <span className="truncate">
                      {s.label}
                      {s.is_squad && (
                        <span className="ml-1 text-[10px] text-muted-foreground">· escouade</span>
                      )}
                    </span>
                    <span className="shrink-0 text-[10px] text-muted-foreground">
                      {s.match_count} match{s.match_count > 1 ? 's' : ''}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      )}
    </div>
  )
}

// ─── Pill : Période ──────────────────────────────────────────────────────────

interface PeriodePillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  period: PeriodInput | undefined
  onSetPeriod: (p: PeriodInput) => void
}

function PeriodePill({ open, onToggle, onClose, period, onSetPeriod }: PeriodePillProps) {
  const ref = useDismissable(open, onClose)
  const detected = detectActivePreset(period)
  const hasPeriod = !!(period?.start_date || period?.end_date)

  let triggerLabel = 'Période'
  if (hasPeriod) {
    const preset = PERIOD_PRESETS.find((p) => p.id === detected)
    triggerLabel = preset && preset.id !== 'all' ? `Période : ${preset.label}` : 'Période : personnalisée'
  }

  function applyPreset(days: number) {
    onSetPeriod(presetPeriod(days) ?? { start_date: null, end_date: null })
  }

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={[
          'flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          hasPeriod
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span>{triggerLabel}</span>
        <span className="text-[10px] opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label="Période"
          className="absolute left-0 top-full z-40 mt-1 flex w-80 flex-col gap-3 rounded-md border border-border bg-background p-3 shadow-lg"
        >
          <div className="flex flex-wrap gap-3">
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Du
              <input
                type="date"
                value={period?.start_date ?? ''}
                max={period?.end_date ?? undefined}
                onChange={(e) =>
                  onSetPeriod({
                    ...(period ?? {}),
                    start_date: e.target.value || null,
                  })
                }
                className="rounded border border-input bg-background px-2 py-1 text-xs text-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </label>
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Au
              <input
                type="date"
                value={period?.end_date ?? ''}
                min={period?.start_date ?? undefined}
                onChange={(e) =>
                  onSetPeriod({
                    ...(period ?? {}),
                    end_date: e.target.value || null,
                  })
                }
                className="rounded border border-input bg-background px-2 py-1 text-xs text-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </label>
          </div>

          <div className="flex flex-wrap gap-2">
            {PERIOD_PRESETS.map((p) => {
              const isActive = detected === p.id
              return (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => applyPreset(p.days)}
                  className={[
                    'rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
                    isActive
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-foreground hover:bg-accent',
                  ].join(' ')}
                >
                  {p.label}
                </button>
              )
            })}
          </div>

          <p className="text-[10px] text-muted-foreground">
            Sélectionner une période vide automatiquement la session active.
          </p>
        </div>
      )}
    </div>
  )
}

// ─── Pill : Filtres avancés (cascade) ────────────────────────────────────────

interface FiltresPillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  available: {
    playlists: { label: string; value: string }[]
    modes: { label: string; value: string }[]
    maps: { label: string; value: string }[]
    experience_types: { label: string; value: string }[]
  }
  cascade: CascadeInput
  cascadeCount: number
  onSetCascade: (c: CascadeInput) => void
}

function FiltresPill({
  open,
  onToggle,
  onClose,
  available,
  cascade,
  cascadeCount,
  onSetCascade,
}: FiltresPillProps) {
  const ref = useDismissable(open, onClose)

  function toggleValue(key: keyof CascadeInput, value: string) {
    const current = (cascade[key] ?? []) as string[]
    const next = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value]
    onSetCascade({ ...cascade, [key]: next })
  }

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={[
          'flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          cascadeCount > 0
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span>Filtres</span>
        {cascadeCount > 0 && (
          <span className="rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-medium text-primary-foreground">
            {cascadeCount}
          </span>
        )}
        <span className="text-[10px] opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label="Filtres avancés"
          className="absolute left-0 top-full z-40 mt-1 grid w-[28rem] grid-cols-2 gap-3 rounded-md border border-border bg-background p-3 shadow-lg"
        >
          <CheckboxGroup
            title="Playlists"
            options={available.playlists}
            selected={(cascade.playlists ?? []) as string[]}
            onToggle={(v) => toggleValue('playlists', v)}
          />
          <CheckboxGroup
            title="Modes"
            options={available.modes}
            selected={(cascade.modes ?? []) as string[]}
            onToggle={(v) => toggleValue('modes', v)}
          />
          <CheckboxGroup
            title="Cartes"
            options={available.maps}
            selected={(cascade.maps ?? []) as string[]}
            onToggle={(v) => toggleValue('maps', v)}
          />
          <CheckboxGroup
            title="Type d'expérience"
            options={available.experience_types}
            selected={(cascade.experience_types ?? []) as string[]}
            onToggle={(v) => toggleValue('experience_types', v)}
          />
        </div>
      )}
    </div>
  )
}

// ─── Sous-composant : groupe de cases à cocher ───────────────────────────────

interface CheckboxGroupProps {
  title: string
  options: { label: string; value: string }[]
  selected: string[]
  onToggle: (value: string) => void
}

function CheckboxGroup({ title, options, selected, onToggle }: CheckboxGroupProps) {
  if (options.length === 0) return null
  return (
    <div className="flex flex-col">
      <h4 className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
        {selected.length > 0 && (
          <span className="ml-1 text-primary">({selected.length})</span>
        )}
      </h4>
      <div className="max-h-44 overflow-y-auto rounded border border-border">
        {options.map((opt) => {
          const checked = selected.includes(opt.value)
          return (
            <label
              key={opt.value}
              className="flex cursor-pointer items-center gap-2 px-2 py-1 text-xs text-foreground transition-colors hover:bg-muted"
            >
              <input
                type="checkbox"
                checked={checked}
                onChange={() => onToggle(opt.value)}
                className="h-3 w-3 cursor-pointer rounded border-input text-primary focus:ring-1 focus:ring-ring"
              />
              <span className="flex-1 truncate">{opt.label}</span>
            </label>
          )
        })}
      </div>
    </div>
  )
}
