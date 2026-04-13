/**
 * AppShell — enveloppe principale de l'application.
 *
 * Compose le layout Sidebar + zone de contenu principale.
 * Utilisé dans __root.tsx une fois le bootstrap terminé.
 */
import { Outlet } from '@tanstack/react-router'
import { NavBar } from './NavBar'

export function AppShell() {
  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      {/* Sidebar fixe */}
      <NavBar />

      {/* Zone de contenu principale */}
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
