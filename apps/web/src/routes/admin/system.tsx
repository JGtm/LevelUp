/**
 * Route /admin/system — Système : bas niveau (contention DB, perf persist,
 * tail de logs bruts, backup). Porte l'URL-state du viewer de logs (ex-onglet
 * Logs, A3) : module, level (seuil), q (contains), n (limite).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminSystemPage } from '@/features/admin/system/AdminSystemPage'

const LEVELS = new Set(['debug', 'info', 'warn', 'error'])
// Valeurs string : la convention du routeur est search params string-only
// (HelpPage/SettingsPage passent l'objet search parsé à URLSearchParams).
const LIMITS = new Set(['50', '200', '500'])

export interface AdminSystemSearch {
  module: string
  level?: string
  q?: string
  n?: string
}

export const Route = createFileRoute('/admin/system')({
  validateSearch: (search: Record<string, unknown>): AdminSystemSearch => {
    const module =
      typeof search.module === 'string' && /^[a-z0-9_-]{1,64}$/.test(search.module)
        ? search.module
        : 'general'
    const level =
      typeof search.level === 'string' && LEVELS.has(search.level) ? search.level : undefined
    const q = typeof search.q === 'string' && search.q ? search.q.slice(0, 128) : undefined
    const n = LIMITS.has(String(search.n)) ? String(search.n) : undefined
    return { module, level, q, n }
  },
  component: AdminSystemPage,
})
