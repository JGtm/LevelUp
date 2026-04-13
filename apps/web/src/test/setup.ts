/**
 * Setup Vitest + Testing Library + MSW.
 * Importé automatiquement via `setupFiles` dans vite.config.ts.
 */
import '@testing-library/jest-dom'
import { setupServer } from 'msw/node'
import { handlers } from './handlers'

// Serveur MSW partagé pour tous les tests
export const server = setupServer(...handlers)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
