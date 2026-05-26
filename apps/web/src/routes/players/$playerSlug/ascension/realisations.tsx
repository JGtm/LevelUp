/**
 * Route /players/$playerSlug/ascension/realisations — tab "Réalisations".
 *
 * Composé par Phase 7 (AscensionRealisationsTab).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionRealisationsTab } from '@/features/ascension/AscensionRealisationsTab'

export const Route = createFileRoute('/players/$playerSlug/ascension/realisations')({
  component: AscensionRealisationsTab,
})
