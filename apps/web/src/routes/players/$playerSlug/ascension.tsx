/**
 * Route /players/$playerSlug/ascension — page Ascension (V2 progression).
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.3.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionPage } from '@/features/ascension/AscensionPage'

export const Route = createFileRoute('/players/$playerSlug/ascension')({
  component: AscensionPage,
})
