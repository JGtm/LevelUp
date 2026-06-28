/**
 * NavL1MobileActions — menu « compte & outils » pour petits écrans (`< md`).
 *
 * Pendant mobile du cluster droit de NavL1, qui est masqué sous `md`
 * (`hidden md:flex`). Bouton kebab `⋯` ouvrant un panneau latéral droit avec :
 * - Compte : thème · paramètres (onglets) · déconnexion
 * - Outils : référentiels · feedback/issue · aide
 *
 * Les outils (référentiels, feedback) sont des panneaux latéraux propres, ouverts
 * via leur store (rendus responsives en bottom-sheet plein écran sous `sm`). Le
 * sélecteur de joueur et la cloche restent visibles dans le bandeau (hors de ce menu).
 */
import { Link } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useLogout } from '@/features/auth/queries'
import { useAssetDrawerStore } from '@/features/asset-drawer/assetDrawer.store'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { feedbackDrawerManifest } from '@/lib/i18n/generated/feedback_drawer'
import { ThemeToggle } from './ThemeToggle'
import { TitleSwitcher } from './TitleSwitcher'
import { log } from './_logger'
import type { SettingsTab } from '@/features/settings/tabs'

export interface SettingsTabItem {
  key: string
  label: string
  tab: SettingsTab
}

interface NavL1MobileActionsProps {
  /** Onglets de Paramètres (construits dans NavL1 avec les capabilities). */
  settingsTabs: SettingsTabItem[]
  /** Pathname courant pour l'état actif. */
  pathname: string
  /** Affiche le lien Administration (réservé admin). */
  isAdmin: boolean
}

export function NavL1MobileActions({ settingsTabs, pathname, isAdmin }: NavL1MobileActionsProps) {
  const [open, setOpen] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const logout = useLogout()
  const openAssets = useAssetDrawerStore((s) => s.open)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])

  function openMenu() {
    setOpen(true)
    log.debug('nav:actions_open')
  }

  function runTool(tool: 'references' | 'feedback', openFn: () => void) {
    setOpen(false)
    log.debug(`nav:tool_open:${tool}`)
    openFn()
  }

  function handleLogout() {
    setOpen(false)
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
    <div className="md:hidden">
      <button
        type="button"
        onClick={openMenu}
        aria-label={t('common.shell.nav_actions_open')}
        aria-expanded={open}
        aria-haspopup="menu"
        className="flex items-center justify-center rounded-md p-1.5 text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
          viewBox="0 0 24 24"
          fill="currentColor"
          aria-hidden="true"
        >
          <circle cx="12" cy="5" r="1.6" />
          <circle cx="12" cy="12" r="1.6" />
          <circle cx="12" cy="19" r="1.6" />
        </svg>
      </button>

      {open && (
        <div
          className="fixed inset-0 z-[59] bg-black/40"
          onClick={() => setOpen(false)}
          aria-hidden="true"
        />
      )}

      <div
        role="menu"
        aria-label={t('common.shell.nav_actions_title')}
        aria-hidden={!open}
        className="fixed right-0 top-0 z-[60] flex h-full w-[78vw] max-w-[20rem] flex-col border-l border-border bg-sidebar shadow-xl transition-transform duration-200 ease-out"
        style={{ transform: open ? 'translateX(0)' : 'translateX(100%)' }}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-border px-4 py-3">
          <span className="text-sm font-bold text-sidebar-foreground">
            {t('common.shell.nav_actions_title')}
          </span>
          <button
            type="button"
            onClick={() => setOpen(false)}
            aria-label={t('common.shell.nav_menu_close')}
            className="rounded p-1 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <CloseIcon />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto py-2">
          {/* ── Compte ───────────────────────────────────────────────── */}
          <SectionLabel>{t('common.shell.nav_account')}</SectionLabel>

          {/* Sélecteur de jeu (titre) — parité avec le dropdown desktop (NavL1).
              S'auto-masque en mono-titre ; visible dès qu'un 2e titre est présent
              (ex. Demo). Corrige son absence en mobile. */}
          <TitleSwitcher onSwitched={() => setOpen(false)} />

          <div className="flex items-center justify-between px-5 py-2">
            <span className="text-sm text-sidebar-foreground/80">{t('common.shell.nav_theme')}</span>
            <ThemeToggle variant="menu" />
          </div>

          <RowLabel>{t('common.shell.nav_settings')}</RowLabel>
          {settingsTabs.map((item) => (
            <MenuLink
              key={item.key}
              to="/settings"
              search={{ tab: item.tab }}
              active={pathname.startsWith('/settings')}
              onNavigate={() => setOpen(false)}
              indent
            >
              {item.label}
            </MenuLink>
          ))}

          {isAdmin && (
            <MenuLink
              to="/admin"
              search={{}}
              active={pathname.startsWith('/admin')}
              onNavigate={() => setOpen(false)}
              indent
            >
              {t('common.admin.page_title')}
            </MenuLink>
          )}

          {currentUsername && (
            <button
              type="button"
              onClick={handleLogout}
              disabled={logout.isPending}
              role="menuitem"
              className="mt-1 flex w-full items-center gap-2 px-5 py-2 text-left text-sm text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground disabled:opacity-50"
            >
              {t('common.shell.logout')}
            </button>
          )}

          <Divider />

          {/* ── Outils ───────────────────────────────────────────────── */}
          <SectionLabel>{t('common.shell.nav_tools')}</SectionLabel>

          <MenuButton onClick={() => runTool('references', openAssets)}>
            {formatMessage(assetDrawerManifest, 'asset_drawer.mini_tab', locale)}
          </MenuButton>
          <MenuButton onClick={() => runTool('feedback', openFeedback)}>
            {formatMessage(feedbackDrawerManifest, 'feedback_drawer.title', locale)}
          </MenuButton>

          <RowLabel>{t('common.shell.nav_help')}</RowLabel>
          <MenuLink to="/help" search={{ tab: 'glossary' }} active={pathname.startsWith('/help')} onNavigate={() => setOpen(false)} indent>
            {t('common.help.glossary')}
          </MenuLink>
          <MenuLink to="/help" search={{ tab: 'release-notes' }} active={pathname.startsWith('/help')} onNavigate={() => setOpen(false)} indent>
            {t('common.help.release_notes')}
          </MenuLink>
        </div>
      </div>
    </div>
  )
}

// ─── Sous-composants de présentation ──────────────────────────────────────────

function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <div className="px-4 pb-1 pt-2 text-2xs font-semibold uppercase tracking-widest text-sidebar-foreground/50">
      {children}
    </div>
  )
}

function RowLabel({ children }: { children: ReactNode }) {
  return <div className="px-5 pb-0.5 pt-2 text-xs font-medium text-sidebar-foreground/60">{children}</div>
}

function Divider() {
  return <div role="separator" className="my-2 h-px bg-foreground/20" />
}

function MenuButton({ onClick, children }: { onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      role="menuitem"
      className="flex w-full items-center gap-2 px-5 py-2 text-left text-sm text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
    >
      {children}
    </button>
  )
}

function MenuLink({
  to,
  search,
  active,
  indent,
  onNavigate,
  children,
}: {
  to: string
  search: Record<string, string>
  active: boolean
  indent?: boolean
  onNavigate: () => void
  children: ReactNode
}) {
  return (
    <Link
      to={to as never}
      search={search as never}
      role="menuitem"
      onClick={onNavigate}
      aria-current={active ? 'page' : undefined}
      className={[
        'block py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
        indent ? 'px-7' : 'px-5',
        active ? 'text-sidebar-foreground' : 'text-sidebar-foreground/70',
      ].join(' ')}
    >
      {children}
    </Link>
  )
}

function CloseIcon() {
  return (
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
  )
}
