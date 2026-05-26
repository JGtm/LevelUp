/**
 * Route /players/$playerSlug/ascension — index (tab "Profil & objectifs").
 *
 * Composé par Phase 6 (AscensionProfileTab). Placeholder ici.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionProfileTab } from '@/features/ascension/AscensionProfileTab'

export const Route = createFileRoute('/players/$playerSlug/ascension/')({
  component: AscensionProfileTab,
})
