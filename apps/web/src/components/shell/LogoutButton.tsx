/**
 * LogoutButton — bouton de déconnexion dans la NavL1.
 *
 * Visible uniquement quand une session est ouverte (currentUsername non nul).
 * Appelle POST /auth/logout puis force un rechargement complet (window.location)
 * pour que l'auth soit ré-évaluée côté serveur (redirection vers login).
 */
import { toast } from 'sonner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useLogout } from '@/features/auth/queries'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { log } from './_logger'

function LogoutIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className="h-3.5 w-3.5"
      aria-hidden="true"
    >
      <path d="M3 4.25A2.25 2.25 0 015.25 2h4.5a.75.75 0 010 1.5h-4.5a.75.75 0 00-.75.75v11.5c0 .414.336.75.75.75h4.5a.75.75 0 010 1.5h-4.5A2.25 2.25 0 013 15.75V4.25z" />
      <path d="M13.06 6.22a.75.75 0 011.06 0l3.25 3.25a.75.75 0 010 1.06l-3.25 3.25a.75.75 0 11-1.06-1.06l1.97-1.97H8.75a.75.75 0 010-1.5h6.28l-1.97-1.97a.75.75 0 010-1.06z" />
    </svg>
  )
}

export function LogoutButton() {
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const logout = useLogout()

  // Pas de session locale (mode none/demo) → pas de bouton.
  if (!currentUsername) return null

  function handleLogout() {
    logout.mutate(undefined, {
      onSuccess: () => window.location.assign('/'),
      onError: (err) => {
        const msg = err instanceof Error ? err.message : t('common.shell.logout_failed')
        log.error('logout:failed', `logout failed: ${msg}`, err)
        toast.error(msg)
      },
    })
  }

  return (
    <button
      type="button"
      onClick={handleLogout}
      disabled={logout.isPending}
      title={t('common.shell.logout')}
      aria-label={t('common.shell.logout')}
      className="ml-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-sidebar-border bg-sidebar-accent/40 text-sidebar-foreground transition-colors hover:bg-sidebar-accent/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 disabled:opacity-50"
    >
      <LogoutIcon />
    </button>
  )
}
