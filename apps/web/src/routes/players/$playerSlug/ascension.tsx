/**
 * Route /players/$playerSlug/ascension — layout Ascension à 2 onglets.
 *
 * Routes enfants :
 *   - /ascension                → tab "Profil & objectifs" (index)
 *   - /ascension/realisations   → tab "Réalisations"
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionLayout } from '@/features/ascension/AscensionLayout'

export const Route = createFileRoute('/players/$playerSlug/ascension')({
  component: AscensionLayout,
})
