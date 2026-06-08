/**
 * Route /players/$playerSlug/ascension/coaching — tab "Entraînement".
 *
 * Couche coaching d'amélioration (proposals + campagne + profil + patterns),
 * scindée de l'onglet "Profil & objectifs" (2026-06-08).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionCoachingTab } from '@/features/ascension/AscensionCoachingTab'

export const Route = createFileRoute('/players/$playerSlug/ascension/coaching')({
  component: AscensionCoachingTab,
})
