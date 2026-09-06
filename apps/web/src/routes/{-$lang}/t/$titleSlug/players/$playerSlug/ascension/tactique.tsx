/**
 * Route /players/$playerSlug/ascension/tactique — onglet « Tactique ».
 *
 * Grille des cartes jouées (phase 4). `validateSearch` déclare `carte` : la carte
 * sélectionnée vit dans l'URL, donc elle survit au retour navigateur et se partage — et
 * c'est l'état que la vue d'analyse par carte (phase 5) consommera.
 *
 * La porte de capability (`replay`) est portée par `TacticalTab`, pas par ce fichier :
 * une route file-based reste un câblage, et le composant monté se teste sans routeur.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { TacticalTab } from '@/features/tactical/TacticalTab'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique',
)({
  validateSearch: z.object({ carte: z.string().optional() }),
  component: TacticalTab,
})
