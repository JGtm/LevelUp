/**
 * AppShell — enveloppe principale de l'application.
 *
 * Layout vertical : NavL1 (top bar globale, h-12) + zone de contenu principale.
 * La NavL2 (bandeau contextuel Stats/Escouade) et la KPIBar sont gérées
 * au niveau du layout joueur ($playerSlug.tsx).
 */
import { Outlet } from '@tanstack/react-router'
import { NavL1 } from './NavL1'
import { ErrorBoundary } from './ErrorBoundary'

export function AppShell() {
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      {/* Barre de navigation L1 — toujours visible */}
      <NavL1 />

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
