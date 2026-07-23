/**
 * toastBridge — relais entre la query useUnreadCount et Sonner.
 *
 * Stratégie : on observe la liste des non-lues pour le joueur courant.
 * À chaque tick de query, on diff sur l'ID max (persisté en useRef) ;
 * tout item nouveau ET dont la pref de catégorie autorise les toasts
 * est envoyé à Sonner.
 *
 * Le bridge est monté une seule fois dans AppShell. Il ne rend rien.
 */
import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { useNavigate } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTitleSlug } from '@/lib/title-routing'
import { useNotificationsList, useNotificationPreferences } from './queries'
import { resolveTitle, resolveBody } from './format'
import { resolveTarget } from './navigation'
import { getNotificationsText } from './i18n'
import { useMarkRead } from './mutations'
import type { Notification, NotificationPreference } from './types'

/**
 * Bridge invisible. Doit être monté à un endroit où le joueur courant existe
 * et qu'un QueryClientProvider est présent au-dessus (cas d'AppShell).
 */
export function NotificationsToastBridge() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const locale = useAppShellStore((s) => s.locale)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const titleSlug = useTitleSlug()
  const lastMaxIdRef = useRef<number | null>(null)
  const initializedRef = useRef(false)
  const navigate = useNavigate()
  const t = getNotificationsText(locale)

  // On poll la liste des non-lues (limit serré : on n'envoie pas 50 toasts).
  const { data } = useNotificationsList(
    playerSlug,
    { unread_only: true, limit: 10 },
    { enabled: !!playerSlug, refetchInterval: 30_000 },
  )
  const { data: prefsData } = useNotificationPreferences(playerSlug, !!playerSlug)
  const markRead = useMarkRead({ playerSlug })

  useEffect(() => {
    if (!data?.items) return

    // Première donnée reçue après navigation : on initialise le pointeur sans toast
    // (évite de flasher 10 toasts au démarrage).
    if (!initializedRef.current) {
      initializedRef.current = true
      lastMaxIdRef.current = data.items.reduce((m, n) => (n.id > m ? n.id : m), 0) || null
      return
    }

    const lastSeen = lastMaxIdRef.current ?? 0
    const fresh = data.items.filter((n) => n.id > lastSeen)
    if (fresh.length === 0) return

    const prefsByCat = indexPrefs(prefsData?.items ?? [])
    let emitted = 0
    for (const n of fresh) {
      if (!shouldToast(n, prefsByCat)) continue
      emitToast(n, t, playerSlug, titleSlug, navigate, () => markRead.mutate([n.id]))
      emitted++
    }

    // Anti-flood : si 4+ nouveaux, regrouper en 1 seul toast info.
    if (emitted === 0 && fresh.length > 0) {
      // Même si toasts désactivés, on track quand même l'ID max.
    } else if (fresh.length > 3) {
      toast.info(`${fresh.length} ${t.dropdownTitle.toLowerCase()}`, {
        description: t.dropdownViewAll,
      })
    }

    lastMaxIdRef.current = fresh.reduce(
      (m, n) => (n.id > m ? n.id : m),
      lastSeen,
    )
  }, [data, prefsData, t, playerSlug, titleSlug, navigate, markRead])

  // Reset quand on switche de joueur
  useEffect(() => {
    initializedRef.current = false
    lastMaxIdRef.current = null
  }, [playerSlug])

  return null
}

function indexPrefs(prefs: NotificationPreference[]): Map<string, NotificationPreference> {
  const m = new Map<string, NotificationPreference>()
  for (const p of prefs) m.set(p.category, p)
  return m
}

function shouldToast(
  n: Notification,
  prefs: Map<string, NotificationPreference>,
): boolean {
  const p = prefs.get(n.category)
  if (!p) return true // default-on
  if (!p.enabled) return false
  return p.delivery === 'both' || p.delivery === 'toast'
}

function emitToast(
  n: Notification,
  t: ReturnType<typeof getNotificationsText>,
  playerSlug: string,
  titleSlug: string,
  navigate: ReturnType<typeof useNavigate>,
  onView: () => void,
) {
  const title = resolveTitle(n, currentLocale(t))
  const body = resolveBody(n, currentLocale(t))
  const target = resolveTarget(n, playerSlug, titleSlug)
  const action = target
    ? {
        label: t.actionView,
        onClick: () => {
          onView()
          // navigate accepte (to/params/search) typés ; en pratique on caste pour l'option
          navigate({
            to: target.to,
            params: target.params,
            search: target.search,
          } as Parameters<typeof navigate>[0])
        },
      }
    : undefined

  switch (n.severity) {
    case 'error':
      toast.error(title, { description: body, action, duration: Infinity })
      return
    case 'warn':
      toast.warning(title, { description: body, action, duration: 8000 })
      return
    case 'success':
      toast.success(title, { description: body, action })
      return
    default:
      toast.info(title, { description: body, action })
  }
}

/** Hack typé : i18n.ts n'expose pas la locale, on la déduit du dict. */
function currentLocale(t: ReturnType<typeof getNotificationsText>): 'fr' | 'en' {
  return t.dropdownEmpty.includes('caught up') ? 'en' : 'fr'
}
