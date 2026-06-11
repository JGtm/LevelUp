/**
 * Route /admin/logs — viewer de logs par module. Filtres en search params
 * (URL partageable) : module, level (seuil), q (contains), n (limite).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminLogsPage } from '@/features/admin/logs/AdminLogsPage'

const LEVELS = new Set(['debug', 'info', 'warn', 'error'])
// Valeurs string : la convention du routeur est search params string-only
// (HelpPage/SettingsPage passent l'objet search parsé à URLSearchParams).
const LIMITS = new Set(['50', '200', '500'])

export interface AdminLogsSearch {
  module: string
  level?: string
  q?: string
  n?: string
}

export const Route = createFileRoute('/admin/logs')({
  validateSearch: (search: Record<string, unknown>): AdminLogsSearch => {
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
  component: AdminLogsPage,
})
