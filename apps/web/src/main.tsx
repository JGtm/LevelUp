import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
import { AppProviders } from '@/app/providers'
import { installGlobalCapture } from '@/lib/global-capture/install'

// Side-effects globaux (wrap console / fetch / window errors) pour le drawer
// feedback. Idempotent + HMR-safe via flag globalThis.
installGlobalCapture()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProviders />
  </StrictMode>,
)
