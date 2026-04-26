/**
 * Setup Vitest + Testing Library + MSW.
 * Importé automatiquement via `setupFiles` dans vite.config.ts.
 */
import '@testing-library/jest-dom'
import { setupServer } from 'msw/node'
import { handlers } from './handlers'
import { vi } from 'vitest'

// react-plotly.js requiert canvas/WebGL absent en jsdom — on le mocke globalement
vi.mock('react-plotly.js', () => ({
  default: () => null,
}))

// Serveur MSW partagé pour tous les tests
export const server = setupServer(...handlers)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  // Nettoyage localStorage entre tests pour éviter les fuites d'état des
  // stores Zustand persistés (globalFilterStore notamment).
  if (typeof localStorage !== 'undefined') localStorage.clear()
})
afterAll(() => server.close())
