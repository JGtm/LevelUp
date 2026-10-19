/**
 * Route /players/$playerSlug/career/citations — page Citations CALCULÉES (section Carrière).
 * Gardée par `career` ET `citations` : un titre qui sert des commendations natives (Halo 5)
 * obtient FeatureUnavailable (et aucune requête /pages/citations) au lieu de la page du
 * moteur d'Infinite.
 */
import { createFileRoute } from '@tanstack/react-router'
import { UnifiedCitationsPage } from '@/features/citations/UnifiedCitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <RouteCapabilityGate capability="citations">
        <UnifiedCitationsPage source="infinite" />
      </RouteCapabilityGate>
    </RouteCapabilityGate>
  ),
})
