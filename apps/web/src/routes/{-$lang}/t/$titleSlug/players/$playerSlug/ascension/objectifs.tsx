/**
 * Route /players/$playerSlug/ascension/objectifs — onglet "Objectifs".
 *
 * Couche Prestige (objectifs + arcs), extraite de l'index lors de la
 * restructuration 4 onglets (2026-07, DEC-3).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionObjectivesTab } from '@/features/ascension/AscensionObjectivesTab'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/objectifs')({
  component: AscensionObjectivesTab,
})
