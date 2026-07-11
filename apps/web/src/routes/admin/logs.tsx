/**
 * Route /admin/logs — REDIRECTION (A3, DC-8) : l'onglet Logs est remplacé par
 * Détections (triage) ; le viewer de logs bruts vit dans Système, qui porte
 * l'URL-state (module/level/q/n) — les liens partagés restent valides.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/logs')({
  beforeLoad: ({ search }) => {
    const s = search as Record<string, string | undefined>
    throw redirect({
      to: '/admin/system',
      search: { module: s.module ?? 'general', level: s.level, q: s.q, n: s.n },
      replace: true,
    })
  },
})
