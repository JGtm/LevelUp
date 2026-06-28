/**
 * Route /players/$playerSlug/stats/synthesis — page Synthèse (section Solo).
 */
import { createFileRoute } from '@tanstack/react-router'
import { SynthesisPage } from '@/features/synthesis/SynthesisPage'

export const Route = createFileRoute('/players/$playerSlug/stats/synthesis')({
  component: SynthesisPage,
})
