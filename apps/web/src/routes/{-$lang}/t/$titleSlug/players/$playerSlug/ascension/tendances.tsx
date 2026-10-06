/**
 * Route /players/$playerSlug/ascension/tendances — onglet « Tendances ».
 *
 * Câblage seul : la page vit dans `features/tendances/`. Pas de porte de capacité propre —
 * la page dégrade selon les capacités que l'API déclare dans sa réponse.
 */
import { createFileRoute } from '@tanstack/react-router'

import { TendancesTab } from '@/features/tendances/TendancesTab'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tendances')({
  component: TendancesTab,
})
