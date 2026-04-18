/**
 * FilterDrawer — panneau de filtres latéral (drawer depuis la droite).
 *
 * Ouvert depuis le bouton "Filtres" de la NavL2.
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

interface FilterDrawerProps {
  open: boolean
  onClose: () => void
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function FilterDrawer({ open, onClose }: FilterDrawerProps) {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setFilterMode = useGlobalFilterStore((s) => s.setFilterMode)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const setCascade = useGlobalFilterStore((s) => s.setCascade)
  const resetFilters = useGlobalFilterStore((s) => s.resetFilters)

  // Fermeture par touche Escape
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
    <>
      {/* ── Backdrop ────────────────────────────────────────────────────── */}
      <div
        className={[
          'fixed inset-0 z-40 bg-black/20 transition-opacity duration-200',
          open ? 'opacity-100' : 'pointer-events-none opacity-0',
        ].join(' ')}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* ── Drawer ──────────────────────────────────────────────────────── */}
      <aside
        role="dialog"
        aria-modal="true"
        aria-label="Panneau de filtres"
        className={[
          'fixed inset-y-0 right-0 z-50 flex w-80 flex-col bg-white shadow-xl',
          'transition-transform duration-200',
          open ? 'translate-x-0' : 'translate-x-full',
        ].join(' ')}
      >
        {/* En-tête */}
        <div className="flex shrink-0 items-center justify-between border-b border-gray-200 px-5 py-4">
          <h2 className="text-sm font-semibold text-gray-900">Filtres</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
            aria-label="Fermer le panneau"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Corps scrollable */}
        <div className="flex-1 space-y-6 overflow-y-auto px-5 py-5">
          {/* ── Mode d'analyse ──────────────────────────────────────────── */}
          <section>
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-400">
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
                      ? 'bg-purple-600 text-white shadow-sm'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                  ].join(' ')}
                >
                  {mode === 'period' ? 'Période' : 'Sessions'}
                </button>
              ))}
            </div>
          </section>

          {/* ── Sélecteur de session ─────────────────────────────────────── */}
          {sessionOptions.length > 0 && (
            <section>
              <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-400">
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
                className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-700 transition focus:border-purple-400 focus:outline-none focus:ring-2 focus:ring-purple-200"
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

          {/* ── Playlists ────────────────────────────────────────────────── */}
          {available && available.playlists.length > 0 && (
            <CascadeSection
              title="Playlists"
              options={available.playlists}
              selected={(cascade.playlists ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('playlists', v)}
            />
          )}

          {/* ── Modes de jeu ─────────────────────────────────────────────── */}
          {available && available.modes.length > 0 && (
            <CascadeSection
              title="Modes de jeu"
              options={available.modes}
              selected={(cascade.modes ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('modes', v)}
            />
          )}

          {/* ── Cartes ───────────────────────────────────────────────────── */}
          {available && available.maps.length > 0 && (
            <CascadeSection
              title="Cartes"
              options={available.maps}
              selected={(cascade.maps ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('maps', v)}
            />
          )}

          {/* ── Types d'expérience ───────────────────────────────────────── */}
          {available && available.experience_types.length > 0 && (
            <CascadeSection
              title="Type d'expérience"
              options={available.experience_types}
              selected={(cascade.experience_types ?? []) as string[]}
              onToggle={(v) => toggleCascadeValue('experience_types', v)}
            />
          )}
        </div>

        {/* Pied de page */}
        <div className="shrink-0 border-t border-gray-200 px-5 py-4">
          <button
            type="button"
            onClick={() => {
              resetFilters()
              onClose()
            }}
            className="w-full rounded-lg border border-gray-200 py-2 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-800"
          >
            Réinitialiser les filtres
          </button>
        </div>
      </aside>
    </>
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
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-400">
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
                  ? 'bg-purple-600 text-white shadow-sm'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
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
