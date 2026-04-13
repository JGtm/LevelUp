/**
 * Route /players/$playerSlug/synthesis — page Synthèse.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SynthesisPage } from '@/features/synthesis/SynthesisPage'

export const Route = createFileRoute('/players/$playerSlug/synthesis')({
  component: SynthesisPage,
})
