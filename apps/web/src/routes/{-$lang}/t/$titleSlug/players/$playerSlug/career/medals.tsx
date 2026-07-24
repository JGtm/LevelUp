/**
 * Route /players/$playerSlug/career/medals — page Médailles (section Carrière).
 *
 * Gate `career` (parité Citations) : Halo 5 déclare cette capability et possède
 * des médailles, donc l'onglet apparaît aussi pour lui.
 */
import { createFileRoute } from '@tanstack/react-router'
import { MedalsPage } from '@/features/medals/MedalsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/career/medals')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <MedalsPage />
    </RouteCapabilityGate>
  ),
})
