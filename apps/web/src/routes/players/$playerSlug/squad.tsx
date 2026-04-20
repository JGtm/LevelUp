/**
 * Route /players/$playerSlug/squad — layout Escouade.
 * Rôle : layout parent avec Outlet. Les pages enfants
 * (synergies, contributions) sont montées dans l'Outlet.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SquadLayout } from '@/features/squad/SquadLayout'

export const Route = createFileRoute('/players/$playerSlug/squad')({
  component: SquadLayout,
})
