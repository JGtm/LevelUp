/**
 * Route _personal — pathless layout pour les onglets de la nouvelle page
 * de stats perso.
 *
 * Le segment `_personal` n'apparaît pas dans l'URL. Les routes enfants
 * (`_personal.summary`, `_personal.maps-modes`, etc.) partagent le
 * PersonalStatsLayout via <Outlet />. Les vieilles routes plates de
 * /stats/{history,sessions,timeseries} sont indépendantes (pas préfixées).
 */
import { createFileRoute } from '@tanstack/react-router'
import { PersonalStatsLayout } from '@/features/personal-stats/PersonalStatsLayout'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal')({
  component: PersonalStatsLayout,
})
