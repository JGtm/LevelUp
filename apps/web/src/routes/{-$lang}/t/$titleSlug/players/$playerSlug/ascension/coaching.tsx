/**
 * Route /players/$playerSlug/ascension/coaching — tab "Entraînement".
 *
 * Couche coaching d'amélioration (cap du moment + proposals + campagne +
 * pistes de progression + leviers calibrés). Restructuration 4 onglets
 * (2026-07, DEC-3).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionCoachingTab } from '@/features/ascension/AscensionCoachingTab'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/coaching')({
  component: AscensionCoachingTab,
})
