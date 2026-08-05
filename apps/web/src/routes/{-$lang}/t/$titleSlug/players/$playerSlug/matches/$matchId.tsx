/**
 * Route /{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId — LAYOUT du match.
 *
 * Rôle : layout parent avec Outlet + schéma de recherche (onglet). La vue match
 * (index) et le rejeu 2D (`replay`) se montent dans l'Outlet. Le schéma `tab` reste
 * porté ici pour que MatchViewPage (index) le lise via useSearch(from: '…/$matchId').
 *
 * SANS CE LAYOUT, LA SOUS-ROUTE NE S'AFFICHE PAS : un fichier `$matchId.tsx` qui rend
 * directement la page est un parent sans `Outlet`, et `$matchId/replay` rendait alors
 * la vue match. La bascule en layout + `index.tsx` est la contrepartie exacte de
 * l'existence du dossier `$matchId/`.
 */
import { createFileRoute, Outlet } from '@tanstack/react-router'
import { z } from 'zod'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
)({
  validateSearch: z.object({
    tab: z.enum(['summary', 'details']).optional().catch('summary'),
  }),
  component: MatchLayout,
})

function MatchLayout() {
  return <Outlet />
}
