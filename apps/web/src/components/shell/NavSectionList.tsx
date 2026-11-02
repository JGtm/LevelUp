/**
 * NavSectionList — liste des sections de navigation L1 et de leurs onglets.
 *
 * Rendu partagé par le tiroir mobile (NavL1MobileMenu) et la barre latérale de l'application de
 * bureau (SidebarNav) : une seule liste, deux contenants. Les sections viennent de `L1_SECTIONS`
 * (filtrées par `useVisibleNavSections`).
 *
 * - `variant="drawer"` : menu ARIA, tous les onglets visibles sous chaque section (tiroir mobile).
 * - `variant="sidebar"` : navigation à liens, les onglets ne se déplient que sous la section active
 *   ou celle que l'utilisateur a dépliée (bouton chevron).
 */
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { playerRelativePath, routeTemplateSuffix } from '@/lib/title-routing'
import type { L1Section } from './navL1Sections'

export interface NavSectionListProps {
  variant: 'drawer' | 'sidebar'
  /** Sections visibles (déjà filtrées par capability de titre). */
  sections: L1Section[]
  /** Pathname courant pour l'état actif. */
  pathname: string
  /** Slug du titre actif (param de route des liens title-scoped). */
  titleSlug: string
  /** Slug du joueur actif (param de route des liens title-scoped). */
  playerSlug: string
  /** Appelé après un clic sur un lien (ex. fermer le tiroir). */
  onNavigate?: () => void
}

export function NavSectionList({
  variant,
  sections,
  pathname,
  titleSlug,
  playerSlug,
  onNavigate,
}: NavSectionListProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  // Sections dépliées à la main (barre latérale) ; la section active l'est toujours.
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set())
  const drawer = variant === 'drawer'
  const suffix = playerRelativePath(pathname)
  const itemRole = drawer ? 'menuitem' : undefined

  function toggle(key: string) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  return (
    <>
      {sections.map((section) => {
        const isActive = section.matchPathname(pathname)
        const tabsShown = !!section.tabs && (drawer || isActive || expanded.has(section.key))
        const label = t(section.labelKey)
        return (
          <div key={section.key} className="px-2">
            <div className="flex items-stretch">
              <Link
                to={section.defaultPath}
                params={{ titleSlug, playerSlug }}
                onClick={onNavigate}
                role={itemRole}
                aria-current={isActive ? 'page' : undefined}
                className={[
                  'flex min-w-0 flex-1 items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                    : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                ].join(' ')}
              >
                {section.icon}
                {label}
              </Link>
              {!drawer && section.tabs && (
                <button
                  type="button"
                  onClick={() => toggle(section.key)}
                  aria-expanded={tabsShown}
                  aria-label={t('common.nav.section_tabs_aria', { section: label })}
                  className="ml-0.5 shrink-0 rounded-md px-1.5 text-sidebar-foreground/60 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className={`h-3 w-3 transition-transform ${tabsShown ? 'rotate-90' : ''}`}
                    viewBox="0 0 12 12"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path d="M3 1l6 5-6 5z" />
                  </svg>
                </button>
              )}
            </div>

            {tabsShown && section.tabs && (
              <div className="mb-1 ml-3 mt-0.5 flex flex-col border-l border-border pl-2">
                {section.tabs.map((tab) => {
                  const tabActive = suffix !== null && suffix === routeTemplateSuffix(tab.path)
                  return (
                    <Link
                      key={tab.key}
                      to={tab.path}
                      params={{ titleSlug, playerSlug }}
                      onClick={onNavigate}
                      role={itemRole}
                      aria-current={tabActive ? 'page' : undefined}
                      className={[
                        'rounded-md px-3 py-1.5 text-sm transition-colors',
                        tabActive
                          ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                          : 'text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                      ].join(' ')}
                    >
                      {t(tab.labelKey)}
                    </Link>
                  )
                })}
              </div>
            )}
          </div>
        )
      })}
    </>
  )
}
