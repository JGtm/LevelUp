/**
 * Route /players/$playerSlug/ascension/tactique — onglet « Tactique ».
 *
 * Grille des cartes jouées (phase 4) et barre de filtres L2 (phase 4 bis).
 * `validateSearch` déclare TOUT le scope de l'onglet — filtres compris — parce qu'il
 * vit dans l'URL (`usePageScope`) : il survit au retour navigateur, au rechargement,
 * et se partage tel quel. Un paramètre non déclaré serait effacé par le routeur au
 * premier `navigate`, et la barre perdrait son réglage sans rien dire.
 *
 * Les clés courtes (`de`, `a`, `exp`, `pl`, `md`, `ses`, `vue`, `eq`) sont celles de
 * `tacticalScope.ts`, qui est la SEULE à en connaître le sens : ici on ne fait que
 * déclarer leur forme.
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
  validateSearch: z.object({
    carte: z.string().optional(),
    de: z.string().optional(),
    a: z.string().optional(),
    exp: z.string().optional(),
    pl: z.string().optional(),
    md: z.string().optional(),
    ses: z.string().optional(),
    vue: z.string().optional(),
    eq: z.string().optional(),
  }),
  component: TacticalTab,
})
