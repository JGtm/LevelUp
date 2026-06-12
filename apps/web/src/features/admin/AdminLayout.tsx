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
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { useT, useAdminT } from './useAdminText'
import { useMonitoringOverview } from './monitoring/queries'
import { computeTabBadges, type TabBadge } from './tabBadges'

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
  // Pastilles de compteur : query partagée avec la page Vue d'ensemble
  // (React Query déduplique), zéro I/O DuckDB côté Go.
  const { data: overview } = useMonitoringOverview()
  const badges = computeTabBadges(overview)

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
                className={`flex items-center gap-1.5 whitespace-nowrap px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  active
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                {tA(tab.labelKey)}
                {badges[tab.to] && <TabBadgePill badge={badges[tab.to]} />}
              </Link>
            )
          })}
        </nav>
      </div>

      <Outlet />
    </div>
  )
}

/**
 * Pastille de compteur sur un onglet (flat hard-edge, fond muted + couleur
 * sémantique sur le texte/dot — même grammaire que StatusBadge).
 */
function TabBadgePill({ badge }: { badge: TabBadge }) {
  const color = tokenCssVar(badge.token)
  return (
    <span
      className="inline-flex min-w-[1.125rem] items-center justify-center gap-1 rounded-sm bg-muted px-1 py-0.5 text-[10px] font-semibold leading-none tabular-nums"
      style={{ color }}
    >
      {badge.pulse && (
        <span aria-hidden className="h-1.5 w-1.5 flex-none animate-pulse" style={{ backgroundColor: color }} />
      )}
      {badge.count}
    </span>
  )
}
