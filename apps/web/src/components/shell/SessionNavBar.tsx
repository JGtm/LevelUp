/**
 * SessionNavBar — barre sticky de hauteur fixe (h-12) au-dessus du corps des stats.
 *
 * Visible uniquement dans les sections Stats et Escouade. Hauteur constante
 * (jamais conditionnelle) pour éviter tout layout reflow quand on bascule
 * entre mode Session et mode Analyse (Période + Filtres).
 *
 * Mode Session (une session pickée) :
 *   [Session: 31/03/2026 21:24…] [◀ Précédente] [⚡ Dernière] [Suivante ▶]   — ↑ auto-snap
 *
 * Mode Analyse (pas de session pickée — Période et/ou Filtres uniquement) :
 *   [Toutes les sessions]      Période · Filtres actifs · N matchs
 *
 * Le contenu change, la hauteur reste à 48px pour stabilité visuelle.
 */
import { useRouterState } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { CascadeInput } from '@/lib/api/types'

// ─── Helpers section ──────────────────────────────────────────────────────────

type ActiveSection = 'stats' | 'squad' | null

function detectSection(pathname: string): ActiveSection {
  if (/\/players\/[^/]+\/stats\//.test(pathname)) return 'stats'
  if (/\/players\/[^/]+\/squad/.test(pathname)) return 'squad'
  return null
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function SessionNavBar() {
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const section = detectSection(pathname)

  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const isAutoSnapping = useGlobalFilterStore((s) => s.isAutoSnappingToLatest)

  // Hidden hors Stats — la section squad gère sa propre barre dans SquadLayout
  if (!section || section === 'squad') return null

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const pickedId = filterContext.sessions?.picked_sessions?.[0] ?? null
  const currentIdx = pickedId
    ? allSessions.findIndex((s) => s.session_id === pickedId)
    : -1
  const currentSession = currentIdx >= 0 ? allSessions[currentIdx] : null
  const isSessionMode = !!currentSession

  // Navigation : sessions ordonnées plus-récente → plus-ancienne (idx 0 = latest).
  // « Précédente » va vers l'ancien (idx + 1), « Suivante » vers le récent (idx - 1).
  const canGoPrev = allSessions.length > 0 && currentIdx < allSessions.length - 1
  const canGoNext = allSessions.length > 0 && currentIdx > 0
  const canGoLatest = allSessions.length > 0 && currentIdx !== 0

  function pickSession(idx: number) {
    if (idx < 0 || idx >= allSessions.length) return
    setSessions({
      ...(filterContext.sessions ?? {}),
      picked_sessions: [allSessions[idx].session_id],
    })
  }

  function goPrev() {
    pickSession(currentIdx === -1 ? allSessions.length - 1 : currentIdx + 1)
  }

  function goNext() {
    pickSession(currentIdx - 1)
  }

  function goLatest() {
    pickSession(0)
  }

  // Indicateurs mode analyse
  const cascade = (filterContext.cascade ?? {}) as CascadeInput
  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const)
    .reduce((n, k) => n + ((cascade[k] as string[] | undefined)?.length ?? 0), 0)
  const period = filterContext.period
  const hasPeriod = !!(period?.start_date || period?.end_date)
  const totalMatchesAfter = resolvedContext?.counts?.total_matches_after_filters ?? null

  return (
    <div
      className="sticky top-0 z-30 flex h-12 shrink-0 items-center gap-3 border-b border-border bg-background px-4"
      role="navigation"
      aria-label="Navigation de session"
    >
      {isSessionMode ? (
        <>
          <div className="flex min-w-0 items-center gap-2">
            <span className="text-xs uppercase tracking-wider text-muted-foreground">Session</span>
            <span
              className="truncate text-sm font-semibold text-foreground"
              title={currentSession.label}
            >
              {currentSession.label}
            </span>
            {isAutoSnapping && (
              <span
                className="shrink-0 rounded-full bg-primary/15 px-2 py-0.5 text-[10px] font-medium text-primary"
                title="Sélection automatique : nouvelle session détectée."
              >
                auto
              </span>
            )}
          </div>

          <div className="flex shrink-0 items-center gap-1.5">
            <button
              type="button"
              onClick={goPrev}
              disabled={!canGoPrev}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30"
              title="Session plus ancienne"
              aria-label="Session précédente"
            >
              ◀ Précédente
            </button>
            <button
              type="button"
              onClick={goNext}
              disabled={!canGoNext}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30"
              title="Session plus récente"
              aria-label="Session suivante"
            >
              Suivante ▶
            </button>
            <button
              type="button"
              onClick={goLatest}
              disabled={!canGoLatest}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
              title="Session la plus récente"
              aria-label="Aller à la dernière session"
            >
              Dernière
            </button>
          </div>

          <div className="flex-1" />

          <span className="shrink-0 text-xs text-muted-foreground" aria-live="polite">
            {currentIdx + 1} / {allSessions.length}
            {currentSession.match_count > 0 && (
              <> · {currentSession.match_count} match{currentSession.match_count > 1 ? 's' : ''}</>
            )}
          </span>
        </>
      ) : (
        <>
          <div className="flex min-w-0 items-center gap-2">
            <span className="text-xs uppercase tracking-wider text-muted-foreground">
              Analyse
            </span>
            <span className="truncate text-sm font-semibold text-foreground">
              {hasPeriod
                ? `Période${period?.start_date ? ` du ${period.start_date}` : ''}${
                    period?.end_date ? ` au ${period.end_date}` : ''
                  }`
                : 'Toutes les sessions'}
            </span>
          </div>

          <div className="flex-1" />

          <div className="flex shrink-0 items-center gap-3 text-xs text-muted-foreground">
            {cascadeCount > 0 && (
              <span title="Filtres avancés actifs (playlists, modes, cartes, types)">
                {cascadeCount} filtre{cascadeCount > 1 ? 's' : ''}
              </span>
            )}
            {totalMatchesAfter !== null && (
              <span aria-live="polite">
                {totalMatchesAfter} match{totalMatchesAfter > 1 ? 's' : ''}
              </span>
            )}
          </div>
        </>
      )}
    </div>
  )
}
