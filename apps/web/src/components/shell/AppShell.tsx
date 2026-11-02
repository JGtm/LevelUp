/**
 * AppShell — enveloppe principale de l'application.
 *
 * Deux dispositions, choisies par la capacité `can_use_compact_chrome` de /bootstrap (ADR 0044 :
 * jamais le nom du profil) :
 * - site : NavL1 (barre haute, h-12) au-dessus de la zone de contenu, pied de page dans le flux ;
 * - application de bureau : barre latérale (SidebarNav) à gauche, zone de contenu au centre, volet
 *   des référentiels ancré à droite (il pousse le contenu, rien ne se superpose), sans pied de page
 *   (la fenêtre « À propos » en porte le contenu).
 *
 * La NavL2 (bandeau contextuel Stats/Escouade) et la KPIBar sont gérées
 * au niveau du layout joueur ($playerSlug.tsx).
 */
import { Outlet, useMatches } from '@tanstack/react-router'
import { Toaster } from 'sonner'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { NavL1 } from './NavL1'
import { SidebarNav } from './SidebarNav'
import { ReauthBanner } from './ReauthBanner'
import { OAuthRefusalBanner } from './OAuthRefusalBanner'
import { PoolSharingBanner } from './PoolSharingBanner'
import { TopProgressBar } from './TopProgressBar'
import { ErrorBoundary } from './ErrorBoundary'
import { AppFooter } from './AppFooter'
import { piedDePageAffiche } from './piedDePage'
import { NotificationsToastBridge } from '@/features/notifications/toastBridge'
import { AssetDockedPanel, AssetDrawer } from '@/features/asset-drawer'
import { FeedbackDrawer } from '@/features/feedback-drawer'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

/** Bandeaux d'état de la session, au-dessus du contenu. */
function SessionBanners() {
  return (
    <>
      {/* Bannière reconnexion Xbox (refresh_token mort) — PR-B */}
      <ReauthBanner />
      {/* Motif du refus d'une reconnexion Xbox (retour de /login?oauth_error=) */}
      <OAuthRefusalBanner />
      {/* Question de consentement : mise en commun du jeton (première connexion Xbox) */}
      <PoolSharingBanner />
    </>
  )
}

/** Éléments hors flux : progression, toasts, tiroirs. `compact` : disposition de l'application de bureau. */
function ShellOverlays({ compact }: { compact: boolean }) {
  // Lie le thème Sonner au thème app pour que les toasts respectent dark/light.
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const locale = useAppShellStore((s) => s.locale)
  return (
    <>
      {/* Barre de progression — flottante, hors du flux, collée sous NavL1 (site) ou en tête du contenu (bureau) */}
      <TopProgressBar placement={compact ? 'left-60 right-0 top-0' : undefined} />

      {/* Toasts globaux (Sonner) — décalés sous la NavL1 (h-12 = 48px + 8px marge) sur le site, contre le
          bord haut en bureau. theme={theme} synchronise les toasts avec le thème dark/light de l'app. */}
      <Toaster
        position="top-right"
        offset={compact ? 16 : 56}
        theme={theme}
        containerAriaLabel={formatMessage(commonManifest, 'common.notifications.container_aria', locale)}
        toastOptions={{
          classNames: {
            toast:
              'bg-popover text-popover-foreground border border-border shadow-lg rounded-md',
            description: 'text-muted-foreground',
            actionButton: 'bg-primary text-primary-foreground rounded px-2 py-1',
          },
        }}
      />

      {/* Bridge invisible : observe la query unread et déclenche les toasts */}
      <NotificationsToastBridge />

      {/* Référentiels du site : panneau latéral fixe, toujours monté, état géré par le store. Le bureau
          a son volet ancré (rendu par la disposition), pas ce panneau superposé. */}
      {!compact && <AssetDrawer />}

      {/* Feedback Drawer — second panneau latéral sous AssetDrawer (mini-tab + form) */}
      <FeedbackDrawer />
    </>
  )
}

/** Contenu principal scrollable — protégé contre les crashs de composants. */
function ShellMain({ footer }: { footer: boolean }) {
  return (
    <ErrorBoundary>
      <main className="flex-1 overflow-y-auto">
        {/* Colonne pleine hauteur : le contenu prend au MINIMUM la hauteur visible
            (`flex-1`), ce qui pousse le pied de page sous la ligne de flottaison.
            Sans cela (retour utilisateur 2026-09-09), une page dont le contenu se
            charge encore ne rend qu'un placeholder d'une ligne : le pied de page
            remontait alors en haut de l'écran, seul visible, puis se faisait
            repousser à l'arrivée des données — un saut de mise en page à chaque
            navigation. */}
        <div className="app-shell-width mx-auto flex min-h-full w-full flex-col">
          <div className="flex-1">
            <Outlet />
          </div>
          {/* Pied de page — dans le flux scrollable, jamais fixé : la hauteur
              verticale est comptée pour les tableaux denses. */}
          {footer && <AppFooter />}
        </div>
      </main>
    </ErrorBoundary>
  )
}

export function AppShell() {
  const compact = useDeploymentCapability('can_use_compact_chrome')
  // Une vue « cockpit » (onglet Tactique) tient dans l'écran : pas de pied de page sous elle.
  const avecPiedDePage = useMatches({ select: (matches) => piedDePageAffiche(matches) })

  if (compact) {
    return (
      <div className="flex h-screen flex-row overflow-hidden bg-background text-foreground">
        <SidebarNav />
        <div className="flex min-w-0 flex-1 flex-col gap-3 overflow-hidden">
          <div className="shrink-0">
            <SessionBanners />
          </div>
          <ShellMain footer={false} />
        </div>
        <AssetDockedPanel />
        <ShellOverlays compact />
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col gap-3 overflow-hidden bg-background text-foreground sm:gap-4">
      {/* Barre de navigation L1 (fixe en haut) */}
      <div className="shrink-0">
        <NavL1 />
        <SessionBanners />
      </div>
      <ShellOverlays compact={false} />
      <ShellMain footer={avecPiedDePage} />
    </div>
  )
}
