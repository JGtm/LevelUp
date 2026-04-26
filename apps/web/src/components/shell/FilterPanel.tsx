/**
 * FilterPanel — panneau de filtres inline (expandable sous NavL2).
 *
 * Évolution du précédent FilterDrawer (panneau latéral fixe à droite). Sur
 * demande explicite : zéro drawer, partout. Le panneau se déplie / se
 * replie dans le flux du DOM, sans backdrop, avec une transition de
 * `max-height` + `opacity`.
 *
 * Contient : mode d'analyse (Période / Sessions) · sélecteur de session ·
 * filtres cascade (playlists, modes, cartes, types d'expérience).
 *
 * Les options disponibles proviennent du resolvedContext (globalFilterStore) —
 * aucune requête supplémentaire n'est émise.
 */
import { useEffect } from 'react'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { CascadeInput } from '@/lib/api/types'

// ─── Props ────────────────────────────────────────────────────────────────────

interface FilterPanelProps {
  open: boolean
  onClose: () => void
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function FilterPanel({ open, onClose }: FilterPanelProps) {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setFilterMode = useGlobalFilterStore((s) => s.setFilterMode)
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
  const currentSession = filterContext.sessions?.picked_sessions?.[0] ?? ''

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
      // max-height + opacity transition pour le toggle expandable.
      // Pas de fixed/absolute : le panneau participe au flux du DOM et
      // pousse le contenu en dessous.
      className={[
        'overflow-hidden border-b border-border bg-background transition-[max-height,opacity] duration-200 ease-out',
        open ? 'max-h-[40rem] opacity-100' : 'max-h-0 opacity-0',
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
                  filterContext.filter_mode === mode
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'bg-muted text-muted-foreground hover:bg-accent',
                ].join(' ')}
              >
                {mode === 'period' ? 'Période' : 'Sessions'}
              </button>
            ))}
          </div>
        </section>

        {/* ── Sélecteur de session ───────────────────────────────────────── */}
        {sessionOptions.length > 0 && (
          <section>
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Session
            </h3>
            <select
              value={currentSession}
              onChange={(e) =>
                setSessions({
                  ...(filterContext.sessions ?? {}),
                  picked_sessions: e.target.value ? [e.target.value] : [],
                })
              }
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground transition focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="">Toutes les sessions</option>
              {sessionOptions.map((s) => (
                <option key={s.session_id} value={s.session_id}>
                  {s.label} — {s.match_count} matchs
                </option>
              ))}
            </select>
          </section>
        )}

        {/* ── Playlists ──────────────────────────────────────────────────── */}
        {available && available.playlists.length > 0 && (
          <CascadeSection
            title="Playlists"
            options={available.playlists}
            selected={(cascade.playlists ?? []) as string[]}
            onToggle={(v) => toggleCascadeValue('playlists', v)}
          />
        )}

        {/* ── Modes de jeu ───────────────────────────────────────────────── */}
        {available && available.modes.length > 0 && (
          <CascadeSection
            title="Modes de jeu"
            options={available.modes}
            selected={(cascade.modes ?? []) as string[]}
            onToggle={(v) => toggleCascadeValue('modes', v)}
          />
        )}

        {/* ── Cartes ─────────────────────────────────────────────────────── */}
        {available && available.maps.length > 0 && (
          <CascadeSection
            title="Cartes"
            options={available.maps}
            selected={(cascade.maps ?? []) as string[]}
            onToggle={(v) => toggleCascadeValue('maps', v)}
          />
        )}

        {/* ── Types d'expérience ─────────────────────────────────────────── */}
        {available && available.experience_types.length > 0 && (
          <CascadeSection
            title="Type d'expérience"
            options={available.experience_types}
            selected={(cascade.experience_types ?? []) as string[]}
            onToggle={(v) => toggleCascadeValue('experience_types', v)}
          />
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

// ─── Sous-composant : section cascade ─────────────────────────────────────────

interface CascadeSectionProps {
  title: string
  options: { label: string; value: string }[]
  selected: string[]
  onToggle: (value: string) => void
}

function CascadeSection({ title, options, selected, onToggle }: CascadeSectionProps) {
  return (
    <section>
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
      </h3>
      <div className="flex flex-wrap gap-2">
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
    </section>
  )
}
