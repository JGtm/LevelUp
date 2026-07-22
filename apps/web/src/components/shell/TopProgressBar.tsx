/**
 * TopProgressBar — barre de progression unique de chargement de page.
 *
 * Inspiré de NProgress : trickle continu avec incrément décroissant pour donner
 * un mouvement perpétuel jusqu'à la résolution, plutôt que des paliers figés.
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
 *   - START   : barre à INITIAL_PROGRESS (30 %), trickle continu vers 99 % max.
 *   - SETTLE  : 150 ms de grâce après pendingCount→0 OU navigation sans queries
 *               (les queries d'une nouvelle page mountent un tick après le pathname change).
 *   - END     : complétion à 100 %, hold court, fade-out 250 ms.
 *   - SAFETY  : timeout max MAX_VISIBLE_MS qui force la complétion (filet contre query infinie).
 *   - Durée minimale visible : 450 ms (évite un flash imperceptible).
 *
 * Couleur : `bg-sidebar-primary` — cohérent avec l'onglet actif de la NavL1.
 */
import { useEffect, useRef, useState } from 'react'
import { useIsFetching } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'

const FADE_OUT_MS = 250
const INITIAL_PROGRESS = 30
const MAX_TRICKLE_PROGRESS = 99
const TRICKLE_INTERVAL_MS = 200
const MIN_VISIBLE_MS = 450
const SETTLE_MS = 150
const MAX_VISIBLE_MS = 8000

// Incrément du trickle : grand au début, infinitésimal à l'approche de 100 %.
// Calqué sur la courbe NProgress.
function trickleIncrement(current: number): number {
  if (current < 20) return 10
  if (current < 50) return 4
  if (current < 80) return 2
  if (current < 99) return 0.5
  return 0
}

export function TopProgressBar() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  // Uniquement les queries sans données en cache (premier chargement d'une page).
  // Les refetch background ne maintiennent pas la barre.
  const pendingCount = useIsFetching({ predicate: (q) => q.state.status === 'pending' })

  const [progress, setProgress] = useState(0)
  const [visible, setVisible] = useState(false)

  const trickleTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const completionTimersRef = useRef<ReturnType<typeof setTimeout>[]>([])
  const settleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startedAtRef = useRef<number | null>(null)
  const activeRef = useRef(false)
  const prevPathRef = useRef(pathname)
  // Ref miroir pour vérifier la valeur courante depuis un closure de timer.
  // Affectée dans un effet (pas pendant le rendu) : lue uniquement de façon
  // asynchrone par les closures de timers (l.147), jamais synchronement.
  const pendingCountRef = useRef(pendingCount)
  useEffect(() => {
    pendingCountRef.current = pendingCount
  }, [pendingCount])

  function clearTrickleTimer() {
    if (trickleTimerRef.current) {
      clearInterval(trickleTimerRef.current)
      trickleTimerRef.current = null
    }
  }

  function clearCompletionTimers() {
    for (const t of completionTimersRef.current) clearTimeout(t)
    completionTimersRef.current = []
  }

  function clearSettleTimer() {
    if (settleTimerRef.current) {
      clearTimeout(settleTimerRef.current)
      settleTimerRef.current = null
    }
  }

  function clearMaxTimer() {
    if (maxTimerRef.current) {
      clearTimeout(maxTimerRef.current)
      maxTimerRef.current = null
    }
  }

  function startBar() {
    if (activeRef.current) {
      // Déjà actif : annuler un settle en attente (nouvelle navigation pendant
      // la fenêtre de grâce) pour éviter de fermer prématurément.
      clearSettleTimer()
      return
    }
    activeRef.current = true
    // eslint-disable-next-line react-hooks/purity -- appelé depuis un useEffect, pas pendant le render
    startedAtRef.current = Date.now()
    clearTrickleTimer()
    clearCompletionTimers()
    clearSettleTimer()
    clearMaxTimer()
    setVisible(true)
    setProgress(INITIAL_PROGRESS)

    trickleTimerRef.current = setInterval(() => {
      setProgress((prev) => {
        const inc = trickleIncrement(prev)
        if (inc === 0) return prev
        // Léger jitter pour un mouvement plus naturel (entre 50 % et 100 % de l'incrément).
        const next = prev + inc * (0.5 + Math.random() * 0.5)
        return Math.min(next, MAX_TRICKLE_PROGRESS)
      })
    }, TRICKLE_INTERVAL_MS)

    // Filet de sécurité : si une query reste en pending indéfiniment (timeout réseau,
    // fetch sans résolution), on force la complétion après MAX_VISIBLE_MS.
    maxTimerRef.current = setTimeout(() => {
      if (activeRef.current) completeBar()
    }, MAX_VISIBLE_MS)
  }

  function completeBar() {
    activeRef.current = false
    clearTrickleTimer()
    clearMaxTimer()
    const elapsed = startedAtRef.current !== null ? Date.now() - startedAtRef.current : MIN_VISIBLE_MS
    const completionDelay = Math.max(0, MIN_VISIBLE_MS - elapsed)
    startedAtRef.current = null
    clearCompletionTimers()
    completionTimersRef.current.push(setTimeout(() => setProgress(100), completionDelay))
    completionTimersRef.current.push(
      setTimeout(() => {
        setVisible(false)
        setProgress(0)
      }, completionDelay + FADE_OUT_MS),
    )
  }

  function scheduleSettle() {
    clearSettleTimer()
    settleTimerRef.current = setTimeout(() => {
      if (activeRef.current && pendingCountRef.current === 0) completeBar()
    }, SETTLE_MS)
  }

  // Effet unifié — démarrage et complétion. Réagit à pathname ET pendingCount, ce qui
  // garantit que la barre se ferme même si pendingCount reste à 0 après navigation
  // (page sans queries ou queries déjà mises en cache → status `success`, pas `pending`).
  useEffect(() => {
    const navigated = pathname !== prevPathRef.current
    if (navigated) prevPathRef.current = pathname

    if (navigated || pendingCount > 0) startBar()

    if (pendingCount > 0) {
      clearSettleTimer()
    } else if (activeRef.current) {
      scheduleSettle()
    }
  }, [pathname, pendingCount]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => () => {
    clearTrickleTimer()
    clearCompletionTimers()
    clearSettleTimer()
    clearMaxTimer()
  }, [])

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
