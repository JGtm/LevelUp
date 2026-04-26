/**
 * NavL2 — bandeau de contexte analytique (niveau 2).
 *
 * Visible uniquement dans les sections Stats et Escouade.
 *
 * Stats    : sous-onglets (Historique · Séries · Sessions)
 *            + bandeau filtres avec navigation de session.
 * Escouade : bandeau filtres + navigation de session uniquement.
 *
 * Bandeau filtres (ligne compacte h-9) :
 *   [scope session]  [◀ Précédente]  [⚡ Dernière]  │  [chips actifs ×]  ── [Filtres]  [Réinitialiser]
 */
import { useState } from 'react'
import { Link, useRouterState, useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { FilterPanel } from './FilterPanel'

// ─── Sous-onglets de la section Stats ─────────────────────────────────────────

const STATS_TABS = [
  { label: 'Historique', path: '/players/$playerSlug/stats/history' },
  { label: 'Séries', path: '/players/$playerSlug/stats/timeseries' },
  { label: 'Sessions', path: '/players/$playerSlug/stats/sessions' },
] as const

// ─── Helpers ──────────────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | null

function detectSection(pathname: string): ActiveSection {
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  return null
}

// ─── Chip filtre ──────────────────────────────────────────────────────────────

interface FilterChipProps {
  label: string
  onRemove: () => void
}

function FilterChip({ label, onRemove }: FilterChipProps) {
  return (
    <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-primary/20 px-2.5 py-0.5 text-xs font-medium text-primary">
      {label}
      <button
        type="button"
        onClick={onRemove}
        className="flex h-3.5 w-3.5 items-center justify-center rounded-full text-primary transition-colors hover:bg-primary/10 hover:text-primary"
        aria-label={`Retirer le filtre ${label}`}
      >
        ×
      </button>
    </span>
  )
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function NavL2() {
  const [panelOpen, setPanelOpen] = useState(false)

  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = params.playerSlug ?? ''

  const section = detectSection(pathname)

  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const setCascade = useGlobalFilterStore((s) => s.setCascade)
  const resetFilters = useGlobalFilterStore((s) => s.resetFilters)

  // N'afficher que sur Stats et Escouade
  if (!section) return null

  // ── Navigation de sessions ─────────────────────────────────────────────────

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const pickedId = filterContext.sessions?.picked_sessions?.[0] ?? null
  const currentIdx = pickedId
    ? allSessions.findIndex((s) => s.session_id === pickedId)
    : -1
  const currentLabel =
    pickedId && currentIdx >= 0 ? allSessions[currentIdx].label : 'Toutes les sessions'

  const canGoPrev = allSessions.length > 0 && currentIdx < allSessions.length - 1
  const canGoLatest = allSessions.length > 0 && currentIdx !== 0

  function goPrev() {
    const idx = currentIdx === -1 ? allSessions.length - 1 : currentIdx + 1
    if (idx >= allSessions.length) return
    setSessions({
      ...(filterContext.sessions ?? {}),
      picked_sessions: [allSessions[idx].session_id],
    })
  }

  function goLatest() {
    if (!allSessions.length) return
    setSessions({
      ...(filterContext.sessions ?? {}),
      picked_sessions: [allSessions[0].session_id],
    })
  }

  // ── Chips filtres cascade actifs ───────────────────────────────────────────

  type CascadeKey = 'experience_types' | 'playlists' | 'modes' | 'maps'
  const cascade = filterContext.cascade ?? {}
  const chips: { key: CascadeKey; value: string }[] = []
  ;(['experience_types', 'playlists', 'modes', 'maps'] as CascadeKey[]).forEach((key) => {
    ;(cascade[key] ?? []).forEach((v) => chips.push({ key, value: v }))
  })

  function removeChip(key: CascadeKey, value: string) {
    setCascade({ ...cascade, [key]: (cascade[key] ?? []).filter((v) => v !== value) })
  }

  const hasActiveFilters = chips.length > 0 || !!pickedId

  // ── Résolution des chemins (substitution $playerSlug) ─────────────────────

  function resolvePath(tpl: string): string {
    return tpl.replace('$playerSlug', playerSlug)
  }

  return (
    <>
      <div
        className="shrink-0 border-b border-border bg-background"
        role="navigation"
        aria-label="Navigation analytique"
      >
        {/* ── Sous-onglets Stats ──────────────────────────────────────────── */}
        {section === 'stats' && (
          <div className="flex items-center gap-0 border-b border-border px-4">
            {STATS_TABS.map((tab) => {
              const resolved = resolvePath(tab.path)
              const isActive = pathname === resolved
              return (
                <Link
                  key={tab.label}
                  to={resolved}
                  className={[
                    'border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
                    isActive
                      ? 'border-primary text-primary'
                      : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
                  ].join(' ')}
                  aria-current={isActive ? 'page' : undefined}
                >
                  {tab.label}
                </Link>
              )
            })}
          </div>
        )}

        {/* ── Bandeau filtres ─────────────────────────────────────────────── */}
        <div className="flex h-9 items-center gap-2 overflow-x-auto px-4">
          {/* Scope session + navigation */}
          {allSessions.length > 0 && (
            <>
              <span className="shrink-0 max-w-[12rem] truncate text-xs font-medium text-muted-foreground">
                {currentLabel}
              </span>
              <div className="flex shrink-0 items-center gap-0.5">
                <button
                  type="button"
                  onClick={goPrev}
                  disabled={!canGoPrev}
                  className="rounded px-1.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
                  title="Session précédente"
                >
                  ◀ Précédente
                </button>
                <button
                  type="button"
                  onClick={goLatest}
                  disabled={!canGoLatest}
                  className="rounded px-1.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
                  title="Session la plus récente"
                >
                  ⚡ Dernière
                </button>
              </div>
            </>
          )}

          {/* Séparateur visuel avant les chips */}
          {chips.length > 0 && (
            <div className="mx-0.5 h-4 w-px shrink-0 bg-border" aria-hidden="true" />
          )}

          {/* Chips filtres actifs */}
          {chips.map((chip) => (
            <FilterChip
              key={`${chip.key}-${chip.value}`}
              label={chip.value}
              onRemove={() => removeChip(chip.key, chip.value)}
            />
          ))}

          {/* Spacer */}
          <div className="flex-1" />

          {/* Bouton Filtres — toggle expandable inline (plus de drawer) */}
          <button
            type="button"
            onClick={() => setPanelOpen((v) => !v)}
            aria-expanded={panelOpen}
            aria-controls="filter-panel"
            className="shrink-0 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors hover:border-border hover:bg-muted"
          >
            {panelOpen ? 'Masquer les filtres' : 'Filtres'}
          </button>

          {/* Réinitialiser (seulement si filtres actifs) */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={resetFilters}
              className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
            >
              Réinitialiser
            </button>
          )}
        </div>

        {/* Panneau de filtres inline (zéro drawer, déplie/replie en place) */}
        <div id="filter-panel">
          <FilterPanel open={panelOpen} onClose={() => setPanelOpen(false)} />
        </div>
      </div>
    </>
  )
}
