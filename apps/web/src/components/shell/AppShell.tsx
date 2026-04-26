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
import { TopProgressBar } from './TopProgressBar'
import { ErrorBoundary } from './ErrorBoundary'
import { NotificationsToastBridge } from '@/features/notifications/toastBridge'

export function AppShell() {
  return (
    <div className="flex h-screen flex-col gap-3 overflow-hidden bg-background text-foreground sm:gap-4">
      {/* Barre de navigation L1 (fixe en haut) */}
      <div className="shrink-0">
        <NavL1 />
      </div>

      {/* Barre de progression — flottante, hors du flux, collée sous NavL1 */}
      <TopProgressBar />

      {/* Toasts globaux (Sonner) — décalés sous la NavL1 (h-12 = 48px + 8px marge) */}
      <Toaster
        position="top-right"
        offset={56}
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

      {/* Contenu principal scrollable — protégé contre les crashs de composants */}
      <ErrorBoundary>
        <main className="flex-1 overflow-y-auto">
          <div className="app-shell-width mx-auto w-full">
            <Outlet />
          </div>
        </main>
      </ErrorBoundary>
    </div>
  )
}
