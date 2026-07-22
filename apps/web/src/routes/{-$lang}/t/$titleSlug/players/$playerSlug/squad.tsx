/**
 * Route /players/$playerSlug/squad — layout Escouade.
 * Rôle : layout parent avec Outlet. Les pages enfants
 * (synergies, contributions) sont montées dans l'Outlet.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SquadLayout } from '@/features/squad/SquadLayout'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/squad')({
  // Deep-link depuis l'accueil (card session escouade) : `session` cible la
  // session à afficher, `teammates` la composition à pré-sélectionner (gamertags
  // joints par virgule — un gamertag Xbox n'en contient jamais). SquadLayout les
  // consomme au montage. String (pas array) pour rester compatible avec les
  // utilitaires qui traitent les search params comme Record<string,string>.
  validateSearch: z.object({
    session: z.string().optional(),
    teammates: z.string().optional(),
  }),
  component: SquadLayout,
})
