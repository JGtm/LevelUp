/**
 * AdminLayout — layout parent de la section admin (guard isAdmin + onglets +
 * Outlet). Chaque onglet est une vraie sous-route (scoping du polling par page
 * + code-splitting + URL-state).
 *
 * Architecture DC-8 (A3) — chaque onglet répond à UNE question opérateur :
 * État (/admin) « tout va bien ? » · Détections « que dois-je traiter ? » ·
 * Données « mon warehouse est-il intègre ? » · Sync « le moteur tourne-t-il ? » ·
 * Système (bas niveau) · Gestion (administration). Tous les onglets vivent dans
 * le même flux horizontal (observation d'abord, gestion en fin de barre).
 */
import { useEffect } from 'react'
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
  { to: '/admin/detections', labelKey: 'admin.nav.detections' },
  { to: '/admin/data', labelKey: 'admin.nav.data' },
  { to: '/admin/sync', labelKey: 'admin.nav.sync' },
  { to: '/admin/system', labelKey: 'admin.nav.system' },
  { to: '/admin/management', labelKey: 'admin.nav.management' },
]

export function AdminLayout() {
  const navigate = useNavigate()
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const t = useT()
  const tA = useAdminT()
  const matchRoute = useMatchRoute()

  // Garde admin : redirection dans un EFFET (post-commit), jamais pendant le
  // render. Un navigate() en cours de render met à jour le Transitioner du
  // routeur PENDANT le rendu d'AdminLayout → « Cannot update a component while
  // rendering a different component » + boucle de re-render (page figée). On
  // attend aussi la fin du bootstrap : avant hydratation isAdmin=false par
  // défaut, sans ce garde un admin serait rebondi vers '/' au premier render.
  useEffect(() => {
    if (isBootstrapped && !isAdmin) {
      navigate({ to: '/' })
    }
  }, [isBootstrapped, isAdmin, navigate])

  // Pastilles de compteur : query partagée avec la page État (React Query
  // déduplique), zéro I/O DuckDB côté Go.
  const { data: overview } = useMonitoringOverview()
  const badges = computeTabBadges(overview)

  // Non-admin (ou bootstrap en cours) : ne rien rendre — l'effet ci-dessus
  // redirige dès que le statut est connu.
  if (!isAdmin) {
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

      {/* Navigation onglets (pattern SquadLayout) — tous les onglets dans le
          flux horizontal normal, Gestion en fin de barre comme les autres. */}
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
                    ? 'border-b-primary text-primary'
                    : 'border-b-transparent text-muted-foreground hover:text-foreground'
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
