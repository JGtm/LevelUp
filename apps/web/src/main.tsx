import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
import { AppProviders } from '@/app/providers'
import { installGlobalCapture } from '@/lib/global-capture/install'
import { initTitleFromLocation } from '@/lib/title-routing'

// D-9 (chantier D7 — titre dans l'URL) : affirmer le titre (et la langue) depuis
// le segment d'URL AVANT tout — createRoot, installGlobalCapture, et surtout la
// première requête (bootstrap). X-LevelUp-Title est ainsi posé dès le premier
// fetch pour les pages title-scoped (routes déplacées en Phase 2). No-op runtime
// tant qu'aucune route ne porte de segment (Phase 1).
initTitleFromLocation(window.location.pathname)

// Side-effects globaux (wrap console / fetch / window errors) pour le drawer
// feedback. Idempotent + HMR-safe via flag globalThis.
installGlobalCapture()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProviders />
  </StrictMode>,
)
