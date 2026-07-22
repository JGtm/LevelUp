/**
 * Route /players/$playerSlug/ascension — layout Ascension à 4 onglets.
 *
 * Routes enfants :
 *   - /ascension                → tab "Profil" (index)
 *   - /ascension/objectifs      → tab "Objectifs" (couche Prestige)
 *   - /ascension/coaching       → tab "Entraînement"
 *   - /ascension/realisations   → tab "Réalisations"
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionLayout } from '@/features/ascension/AscensionLayout'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/ascension')({
  // Ascension (profil LUSR + leviers + coaching + réalisations) dépend du rating
  // LUSR ⇒ gate `lusr` sur le layout (couvre les 3 onglets enfants via l'Outlet).
  component: () => (
    <RouteCapabilityGate capability="lusr">
      <AscensionLayout />
    </RouteCapabilityGate>
  ),
})
