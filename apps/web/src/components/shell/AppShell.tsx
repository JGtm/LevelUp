/**
 * AppShell — enveloppe principale de l'application.
 *
 * Layout vertical : NavL1 (top bar globale, h-12) + zone de contenu principale.
 * La NavL2 (bandeau contextuel Stats/Escouade) et la KPIBar sont gérées
 * au niveau du layout joueur ($playerSlug.tsx).
 */
import { Outlet } from '@tanstack/react-router'
import { Toaster } from 'sonner'
import { NavL1 } from './NavL1'
import { ReauthBanner } from './ReauthBanner'
import { TopProgressBar } from './TopProgressBar'
import { ErrorBoundary } from './ErrorBoundary'
import { AppFooter } from './AppFooter'
import { NotificationsToastBridge } from '@/features/notifications/toastBridge'
import { AssetDrawer } from '@/features/asset-drawer'
import { FeedbackDrawer } from '@/features/feedback-drawer'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

export function AppShell() {
  // Lie le thème Sonner au thème app pour que les toasts respectent dark/light.
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  return (
    <div className="flex h-screen flex-col gap-3 overflow-hidden bg-background text-foreground sm:gap-4">
      {/* Barre de navigation L1 (fixe en haut) */}
      <div className="shrink-0">
        <NavL1 />
        {/* Bannière reconnexion Xbox (refresh_token mort) — PR-B */}
        <ReauthBanner />
      </div>

      {/* Barre de progression — flottante, hors du flux, collée sous NavL1 */}
      <TopProgressBar />

      {/* Toasts globaux (Sonner) — décalés sous la NavL1 (h-12 = 48px + 8px marge).
          theme={theme} synchronise les toasts avec le thème dark/light de l'app. */}
      <Toaster
        position="top-right"
        offset={56}
        theme={theme}
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

      {/* Asset Drawer — panneau latéral fixe, toujours monté, état géré par le store */}
      <AssetDrawer />

      {/* Feedback Drawer — second panneau latéral sous AssetDrawer (mini-tab + form) */}
      <FeedbackDrawer />

      {/* Contenu principal scrollable — protégé contre les crashs de composants */}
      <ErrorBoundary>
        <main className="flex-1 overflow-y-auto">
          <div className="app-shell-width mx-auto w-full">
            <Outlet />
            {/* Pied de page — dans le flux scrollable, jamais fixé : la hauteur
                verticale est comptée pour les tableaux denses. */}
            <AppFooter />
          </div>
        </main>
      </ErrorBoundary>
    </div>
  )
}
