/**
 * FilterPanel — panneau de filtres inline (expandable sous NavL2).
 *
 * Évolution du précédent FilterDrawer (panneau latéral fixe à droite). Sur
 * demande explicite : zéro drawer, partout. Le panneau se déplie / se
 * replie dans le flux du DOM, sans backdrop, avec une transition de
 * `max-height` + `opacity`.
 *
 * Architecture des filtres :
 *  - Mode d'analyse : toggle Période / Sessions
 *    - Période : presets 7j/30j/90j/Toutes + Personnalisé (date pickers natifs)
 *    - Sessions : liste déroulante + navigation ◀/▶ pour parcourir
 *      séquentiellement les sessions (alignée sur les boutons NavL2)
 *  - Cascade (Playlists / Modes / Cartes / Types) : visibles dans les deux
 *    modes — orthogonaux au mode période/sessions, ils raffinent le scope
 *    quel qu'il soit.
 *
 * Les options disponibles proviennent du resolvedContext (globalFilterStore) —
 * aucune requête supplémentaire n'est émise.
 */
import { useEffect, useMemo, useState } from 'react'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { CascadeInput, PeriodInput } from '@/lib/api/types'

// ─── Props ────────────────────────────────────────────────────────────────────

interface FilterPanelProps {
  open: boolean
  onClose: () => void
}

// ─── Helpers période ──────────────────────────────────────────────────────────

const PERIOD_PRESETS = [
  { id: '7d', label: '7 jours', days: 7 },
  { id: '30d', label: '30 jours', days: 30 },
  { id: '90d', label: '90 jours', days: 90 },
  { id: 'all', label: 'Toutes', days: 0 }, // 0 = pas de filtre
] as const

type PresetId = (typeof PERIOD_PRESETS)[number]['id'] | 'custom'

/** Format YYYY-MM-DD aligné sur l'input HTML5 type="date". */
function isoDate(d: Date): string {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

/** Construit { start_date, end_date } pour un preset N jours, ou null pour "Toutes". */
function presetPeriod(days: number): PeriodInput | null {
  if (days <= 0) return null
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - days)
  return { start_date: isoDate(start), end_date: isoDate(end) }
}

/** Détecte le preset actif à partir de la période courante (best-effort, à 1 jour près). */
function detectActivePreset(period: PeriodInput | undefined): PresetId {
  if (!period) return 'all'
  if (!period.start_date && !period.end_date) return 'all'
  // Compare au preset attendu pour chaque N
  for (const p of PERIOD_PRESETS) {
    const expected = presetPeriod(p.days)
    if (!expected) {
      if (!period.start_date && !period.end_date) return p.id
      continue
    }
    if (
      period.start_date === expected.start_date &&
      period.end_date === expected.end_date
    ) {
      return p.id
    }
  }
  return 'custom'
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function FilterPanel({ open, onClose }: FilterPanelProps) {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setFilterMode = useGlobalFilterStore((s) => s.setFilterMode)
  const setPeriod = useGlobalFilterStore((s) => s.setPeriod)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const setCascade = useGlobalFilterStore((s) => s.setCascade)
  const resetFilters = useGlobalFilterStore((s) => s.resetFilters)

  // Fermeture par touche Escape (UX inchangée vs ancien drawer).
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  const available = resolvedContext?.available_options
  const sessionOptions = resolvedContext?.session_options?.all_sessions ?? []
  const cascade = filterContext.cascade ?? {}
  const period = filterContext.period
  const filterMode = filterContext.filter_mode

  // Detection du preset actif depuis le store.
  const detectedPreset = useMemo(() => detectActivePreset(period), [period])
  // État local pour basculer en mode 'custom' (révèle les date pickers même
  // si la période courante matche un preset standard).
  const [customMode, setCustomMode] = useState<boolean>(detectedPreset === 'custom')
  const activePreset: PresetId = customMode ? 'custom' : detectedPreset

  function applyPreset(p: (typeof PERIOD_PRESETS)[number]) {
    setCustomMode(false)
    setPeriod(presetPeriod(p.days) ?? {})
  }

  // ── Sessions : navigation séquentielle ──
  const currentSessionId = filterContext.sessions?.picked_sessions?.[0] ?? ''
  const currentSessionIdx = currentSessionId
    ? sessionOptions.findIndex((s) => s.session_id === currentSessionId)
    : -1
  const canGoPrev =
    sessionOptions.length > 0 && currentSessionIdx < sessionOptions.length - 1
  const canGoNext = sessionOptions.length > 0 && currentSessionIdx > 0

  function gotoSessionAt(idx: number) {
    if (idx < 0 || idx >= sessionOptions.length) return
    setSessions({
      ...(filterContext.sessions ?? {}),
      picked_sessions: [sessionOptions[idx].session_id],
    })
  }

  function toggleCascadeValue(key: keyof CascadeInput, value: string) {
    const current = (cascade[key] ?? []) as string[]
    const next = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value]
    setCascade({ ...cascade, [key]: next })
  }

  return (
    <section
      role="region"
      aria-label="Panneau de filtres"
      className={[
        'overflow-hidden border-b border-border bg-background transition-[max-height,opacity] duration-200 ease-out',
        open ? 'max-h-[48rem] opacity-100' : 'max-h-0 opacity-0',
      ].join(' ')}
      aria-hidden={!open}
    >
      <div className="grid grid-cols-1 gap-6 px-4 py-5 md:grid-cols-2 lg:grid-cols-3">
        {/* ── Mode d'analyse ────────────────────────────────────────────── */}
        <section>
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Mode d'analyse
          </h3>
          <div className="flex gap-2">
            {(['period', 'sessions'] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => setFilterMode(mode)}
                className={[
                  'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  filterMode === mode
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'bg-muted text-muted-foreground hover:bg-accent',
                ].join(' ')}
              >
                {mode === 'period' ? 'Période' : 'Sessions'}
              </button>
            ))}
          </div>
        </section>

        {/* ── Période — affiché uniquement en mode 'period' ──────────────── */}
        {filterMode === 'period' && (
          <section className="md:col-span-2 lg:col-span-2">
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Période
            </h3>
            {/* Calendriers TOUJOURS visibles. Les presets en dessous sont des
                raccourcis qui pré-remplissent les inputs. "Toutes" vide les
                deux dates → pas de filtre période. */}
            <div className="flex flex-wrap items-center gap-3">
              <label className="flex items-center gap-2 text-xs text-muted-foreground">
                Du
                <input
                  type="date"
                  value={period?.start_date ?? ''}
                  max={period?.end_date ?? undefined}
                  onChange={(e) => {
                    setCustomMode(true)
                    setPeriod({
                      ...(period ?? {}),
                      start_date: e.target.value || null,
                    })
                  }}
                  className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </label>
              <label className="flex items-center gap-2 text-xs text-muted-foreground">
                Au
                <input
                  type="date"
                  value={period?.end_date ?? ''}
                  min={period?.start_date ?? undefined}
                  onChange={(e) => {
                    setCustomMode(true)
                    setPeriod({
                      ...(period ?? {}),
                      end_date: e.target.value || null,
                    })
                  }}
                  className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </label>
            </div>

            {/* Raccourcis : appliquent un range pré-rempli. activePreset
                surligne celui qui matche les dates courantes (à 1 jour près). */}
            <div className="mt-3 flex flex-wrap gap-2">
              {PERIOD_PRESETS.map((p) => {
                const isActive = activePreset === p.id
                return (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => applyPreset(p)}
                    className={[
                      'rounded-full px-3 py-1 text-xs font-medium transition-colors',
                      isActive
                        ? 'bg-primary text-primary-foreground shadow-sm'
                        : 'bg-muted text-foreground hover:bg-accent',
                    ].join(' ')}
                  >
                    {p.label}
                  </button>
                )
              })}
            </div>
          </section>
        )}

        {/* ── Sessions — affiché uniquement en mode 'sessions' ──────────── */}
        {filterMode === 'sessions' && (
          <section className="md:col-span-2 lg:col-span-2">
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Session
            </h3>
            {sessionOptions.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                Aucune session disponible — synchronise des matchs pour générer des sessions.
              </p>
            ) : (
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  onClick={() => gotoSessionAt(currentSessionIdx + 1)}
                  disabled={!canGoPrev}
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-input text-sm text-muted-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30"
                  title="Session précédente"
                  aria-label="Session précédente"
                >
                  ◀
                </button>
                <select
                  value={currentSessionId}
                  onChange={(e) =>
                    setSessions({
                      ...(filterContext.sessions ?? {}),
                      picked_sessions: e.target.value ? [e.target.value] : [],
                    })
                  }
                  className="min-w-0 flex-1 rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground transition focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring"
                >
                  <option value="">Toutes les sessions</option>
                  {sessionOptions.map((s) => (
                    <option key={s.session_id} value={s.session_id}>
                      {s.label} — {s.match_count} matchs
                      {s.is_squad ? ' · escouade' : ''}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={() => gotoSessionAt(currentSessionIdx - 1)}
                  disabled={!canGoNext}
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-input text-sm text-muted-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30"
                  title="Session suivante (plus récente)"
                  aria-label="Session suivante"
                >
                  ▶
                </button>
                <span className="text-xs text-muted-foreground">
                  {currentSessionIdx >= 0
                    ? `${currentSessionIdx + 1} / ${sessionOptions.length}`
                    : `${sessionOptions.length} disponibles`}
                </span>
              </div>
            )}
          </section>
        )}

        {/* ── Filtres avancés (cascade) ───────────────────────────────────────
            Repliés par défaut : éviter le mur de pastilles quand il y a 50+
            playlists/cartes/modes. Chaque <details> contient un <summary>
            avec un compteur des valeurs sélectionnées.

            Couvre toute la grille (col-span max) pour que chaque accordéon
            occupe la pleine largeur quand ouvert. */}
        {available && (
          <section className="md:col-span-2 lg:col-span-3">
            <CascadeAccordion
              title="Playlists"
              options={available.playlists}
              selected={(cascade.playlists ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('playlists', v)}
            />
            <CascadeAccordion
              title="Modes de jeu"
              options={available.modes}
              selected={(cascade.modes ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('modes', v)}
            />
            <CascadeAccordion
              title="Cartes"
              options={available.maps}
              selected={(cascade.maps ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('maps', v)}
            />
            <CascadeAccordion
              title="Type d'expérience"
              options={available.experience_types}
              selected={(cascade.experience_types ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('experience_types', v)}
            />
          </section>
        )}
      </div>

      <div className="flex items-center justify-end gap-2 border-t border-border px-4 py-3">
        <button
          type="button"
          onClick={resetFilters}
          className="rounded-lg border border-border px-4 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          Réinitialiser les filtres
        </button>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          Fermer
        </button>
      </div>
    </section>
  )
}

// ─── Sous-composant : accordéon cascade ───────────────────────────────────────
//
// Repliable par défaut. Le <summary> affiche le titre + un badge compteur
// quand des valeurs sont sélectionnées. Évite le mur de pastilles quand
// il y a beaucoup d'options (50+ cartes p. ex.). Native <details> = pas
// de JS d'état pour l'open/close.

interface CascadeAccordionProps {
  title: string
  options: { label: string; value: string }[]
  selected: string[]
  onToggle: (value: string) => void
}

function CascadeAccordion({ title, options, selected, onToggle }: CascadeAccordionProps) {
  if (options.length === 0) return null
  return (
    <details className="group border-b border-border/60 py-2 last:border-b-0">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 py-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground">
        <span className="flex items-center gap-2">
          <span className="inline-block transition-transform group-open:rotate-90" aria-hidden="true">
            ▸
          </span>
          {title}
          {selected.length > 0 && (
            <span className="rounded-full bg-primary px-2 py-0.5 text-[10px] text-primary-foreground">
              {selected.length}
            </span>
          )}
        </span>
        <span className="text-[10px] text-muted-foreground/70">{options.length} options</span>
      </summary>
      <div className="mt-2 flex flex-wrap gap-2 pb-1">
        {options.map((opt) => {
          const active = selected.includes(opt.value)
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => onToggle(opt.value)}
              className={[
                'rounded-full px-3 py-1 text-xs font-medium transition-colors',
                active
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'bg-muted text-foreground hover:bg-accent',
              ].join(' ')}
            >
              {opt.label}
            </button>
          )
        })}
      </div>
    </details>
  )
}
