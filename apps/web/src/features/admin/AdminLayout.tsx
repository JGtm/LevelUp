/**
 * AdminLayout — layout parent de la section admin (guard isAdmin + onglets +
 * Outlet). Remplace l'ancienne AdminPage monolithique : chaque onglet est une
 * vraie sous-route (scoping du polling par page + code-splitting + URL-state).
 *
 * Onglets : Vue d'ensemble (/admin) · Sync & Jobs (/admin/sync) ·
 * Accès (/admin/access) · Système (/admin/system). Les onglets Convergence /
 * Qualité données / Logs arrivent avec leurs phases respectives.
 */
import { Outlet, Link, useNavigate, useMatchRoute } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { useT, useAdminT } from './useAdminText'

interface AdminTab {
  to: string
  labelKey: AdminManifestKey
  /** Match exact pour l'index (/admin), fuzzy pour les sous-routes. */
  exact?: boolean
}

const TABS: AdminTab[] = [
  { to: '/admin', labelKey: 'admin.nav.overview', exact: true },
  { to: '/admin/sync', labelKey: 'admin.nav.sync' },
  { to: '/admin/convergence', labelKey: 'admin.nav.convergence' },
  { to: '/admin/data-quality', labelKey: 'admin.nav.data_quality' },
  { to: '/admin/logs', labelKey: 'admin.nav.logs' },
  { to: '/admin/access', labelKey: 'admin.nav.access' },
  { to: '/admin/system', labelKey: 'admin.nav.system' },
]

export function AdminLayout() {
  const navigate = useNavigate()
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const t = useT()
  const tA = useAdminT()
  const matchRoute = useMatchRoute()

  if (!isAdmin) {
    navigate({ to: '/' })
    return null
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-foreground">{t('common.admin.page_title')}</h1>
        <Button variant="outline" onClick={() => navigate({ to: '/' })}>
          {t('common.admin.back')}
        </Button>
      </div>

      {/* Navigation onglets (pattern SquadLayout) */}
      <div className="border-b">
        <nav className="flex gap-0 overflow-x-auto">
          {TABS.map((tab) => {
            const active = tab.exact
              ? !!matchRoute({ to: tab.to })
              : !!matchRoute({ to: tab.to, fuzzy: true })
            return (
              <Link
                key={tab.to}
                to={tab.to}
                className={`whitespace-nowrap px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  active
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                {tA(tab.labelKey)}
              </Link>
            )
          })}
        </nav>
      </div>

      <Outlet />
    </div>
  )
}
