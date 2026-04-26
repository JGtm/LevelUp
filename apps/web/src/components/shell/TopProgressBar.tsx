/**
 * TopProgressBar — barre de progression unique de chargement de page.
 *
 * Deux déclencheurs :
 *   1. Changement de pathname (via useRouterState) → apparition immédiate dès le clic.
 *   2. Queries React Query en état `pending` (sans données en cache) → maintien
 *      tant que la page n'a pas encore de données.
 *
 * Les refetch background (stale revalidations) ne prolongent PAS la barre car on
 * filtre sur `q.state.status === 'pending'` (jamais résolu) et pas sur `fetchStatus`.
 *
 * Lifecycle :
 *   - START  : barre à 30 %, monte à 70 % (200 ms), puis 85 % (800 ms).
 *   - SETTLE : 150 ms de grâce après pendingCount→0 (les queries mountent un tick
 *              après le changement de pathname).
 *   - END    : complétion à 100 %, hold court, fade-out 250 ms.
 *   - Durée minimale visible : 450 ms (évite un flash imperceptible).
 *
 * Couleur : `bg-sidebar-primary` — cohérent avec l'onglet actif de la NavL1.
 */
import { useEffect, useRef, useState } from 'react'
import { useIsFetching } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'

const FADE_OUT_MS = 250
const INITIAL_PROGRESS = 30
const MID_PROGRESS = 70
const HIGH_PROGRESS = 85
const MID_AT_MS = 200
const HIGH_AT_MS = 800
const MIN_VISIBLE_MS = 450
const SETTLE_MS = 150

export function TopProgressBar() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  // Uniquement les queries sans données en cache (premier chargement d'une page).
  // Les refetch background ne maintiennent pas la barre.
  const pendingCount = useIsFetching({ predicate: (q) => q.state.status === 'pending' })

  const [progress, setProgress] = useState(0)
  const [visible, setVisible] = useState(false)

  const timersRef = useRef<ReturnType<typeof setTimeout>[]>([])
  const settleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startedAtRef = useRef<number | null>(null)
  const activeRef = useRef(false)
  const prevPathRef = useRef(pathname)

  function clearProgressTimers() {
    for (const t of timersRef.current) clearTimeout(t)
    timersRef.current = []
  }

  function clearSettleTimer() {
    if (settleTimerRef.current) {
      clearTimeout(settleTimerRef.current)
      settleTimerRef.current = null
    }
  }

  function startBar() {
    if (activeRef.current) return
    activeRef.current = true
    startedAtRef.current = Date.now()
    clearProgressTimers()
    clearSettleTimer()
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setVisible(true)
    setProgress(INITIAL_PROGRESS)
    timersRef.current.push(setTimeout(() => setProgress(MID_PROGRESS), MID_AT_MS))
    timersRef.current.push(setTimeout(() => setProgress(HIGH_PROGRESS), HIGH_AT_MS))
  }

  function completeBar() {
    activeRef.current = false
    const elapsed = startedAtRef.current !== null ? Date.now() - startedAtRef.current : MIN_VISIBLE_MS
    const completionDelay = Math.max(0, MIN_VISIBLE_MS - elapsed)
    startedAtRef.current = null
    clearProgressTimers()
    timersRef.current.push(setTimeout(() => setProgress(100), completionDelay))
    timersRef.current.push(
      setTimeout(() => {
        setVisible(false)
        setProgress(0)
      }, completionDelay + FADE_OUT_MS),
    )
  }

  // Déclenchement immédiat sur changement de page.
  useEffect(() => {
    if (pathname !== prevPathRef.current) {
      prevPathRef.current = pathname
      startBar()
    }
  }, [pathname]) // eslint-disable-line react-hooks/exhaustive-deps

  // Maintien tant que des queries pending existent ; extinction avec fenêtre de grâce.
  useEffect(() => {
    if (pendingCount > 0) {
      startBar()
      clearSettleTimer()
    } else if (activeRef.current) {
      // Délai : les queries d'une nouvelle page mountent un tick APRÈS le pathname change.
      clearSettleTimer()
      settleTimerRef.current = setTimeout(() => {
        if (activeRef.current) completeBar()
      }, SETTLE_MS)
    }
  }, [pendingCount]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => () => {
    clearProgressTimers()
    clearSettleTimer()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (!visible && progress === 0) return null

  return (
    <div aria-hidden className="pointer-events-none fixed left-0 right-0 top-12 z-50 h-1">
      <div
        className="h-full bg-sidebar-primary shadow-[0_0_8px_color-mix(in_oklch,var(--sidebar-primary)_60%,transparent)]"
        style={{
          width: `${progress}%`,
          opacity: visible ? 1 : 0,
          transition: visible ? 'width 250ms ease-out' : `opacity ${FADE_OUT_MS}ms ease-out`,
        }}
      />
    </div>
  )
}
