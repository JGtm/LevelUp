/**
 * Route /onboarding/openspartan — landing after a successful Xbox SSO login.
 * Default CTA is "Continuer →" toward "/"; the OpenSpartan import card is
 * tucked behind an "Options avancées" disclosure.
 */
import { createFileRoute } from '@tanstack/react-router'
import { OnboardingOpenSpartanPage } from '@/features/onboarding/OnboardingOpenSpartanPage'

export const Route = createFileRoute('/onboarding/openspartan')({
  component: OnboardingOpenSpartanPage,
})
