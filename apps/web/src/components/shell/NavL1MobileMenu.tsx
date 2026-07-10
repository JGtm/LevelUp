/**
 * NavL1MobileMenu — navigation L1 pour petits écrans (`< md`).
 *
 * Sur desktop (`≥ md`), les sections L1 sont rendues inline dans NavL1.
 * Sous le breakpoint `md`, cet inline est masqué (`hidden md:flex`) car il
 * déborde hors viewport : on le remplace par un bouton hamburger qui ouvre un
 * panneau latéral gauche listant toutes les sections + leurs onglets.
 *
 * Source des sections : `L1_SECTIONS` (module partagé navL1Sections), donc
 * aucune duplication de structure avec le rendu desktop.
 */
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { log } from './_logger'
import type { L1Section } from './navL1Sections'

interface NavL1MobileMenuProps {
  /** Sections visibles (déjà filtrées, ex: Ascension masquée si progression off). */
  sections: L1Section[]
  /** Pathname courant pour l'état actif. */
  pathname: string
  /** Résout un template `$playerSlug` en chemin concret. */
  resolvePath: (templatePath: string) => string
}

export function NavL1MobileMenu({ sections, pathname, resolvePath }: NavL1MobileMenuProps) {
  const [open, setOpen] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Fermeture au clavier Escape + verrou du scroll body tant que le panneau est ouvert.
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])

  return (
    <div className="md:hidden">
      <button
        type="button"
        onClick={() => {
          setOpen(true)
          log.debug('nav:menu_open')
        }}
        aria-label={t('common.shell.nav_menu_open')}
        aria-expanded={open}
        aria-haspopup="menu"
        className="flex items-center justify-center rounded-md p-1.5 text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
      </button>

      {/* Overlay — clic hors panneau ferme */}
      {open && (
        <div
          className="fixed inset-0 z-[59] bg-black/40"
          onClick={() => setOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Panneau latéral gauche */}
      <div
        role="menu"
        aria-label={t('common.shell.nav_main_aria')}
        aria-hidden={!open}
        className="fixed left-0 top-0 z-[60] flex h-full w-[78vw] max-w-[20rem] flex-col border-r border-border bg-sidebar shadow-xl transition-transform duration-200 ease-out"
        style={{ transform: open ? 'translateX(0)' : 'translateX(-100%)' }}
      >
        {/* Header : titre + fermer */}
        <div className="flex shrink-0 items-center justify-between border-b border-border px-4 py-3">
          <span className="text-sm font-bold text-sidebar-foreground">{t('common.shell.nav_menu_title')}</span>
          <button
            type="button"
            onClick={() => setOpen(false)}
            aria-label={t('common.shell.nav_menu_close')}
            className="rounded p-1 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Liste des sections (scrollable) */}
        <nav className="flex-1 overflow-y-auto py-2" aria-label={t('common.nav.sections_aria')}>
          {sections.map((section) => {
            const isActive = section.matchPathname(pathname)
            const resolvedDefaultPath = resolvePath(section.defaultPath)
            return (
              <div key={section.key} className="px-2">
                <Link
                  to={resolvedDefaultPath as never}
                  onClick={() => setOpen(false)}
                  role="menuitem"
                  aria-current={isActive ? 'page' : undefined}
                  className={[
                    'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                      : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                  ].join(' ')}
                >
                  {section.icon}
                  {t(section.labelKey)}
                </Link>

                {section.tabs && (
                  <div className="mb-1 ml-3 mt-0.5 flex flex-col border-l border-border pl-2">
                    {section.tabs.map((tab) => {
                      const resolvedTabPath = resolvePath(tab.path)
                      const tabActive = pathname === resolvedTabPath
                      return (
                        <Link
                          key={tab.key}
                          to={resolvedTabPath as never}
                          onClick={() => setOpen(false)}
                          role="menuitem"
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
        </nav>
      </div>
    </div>
  )
}
