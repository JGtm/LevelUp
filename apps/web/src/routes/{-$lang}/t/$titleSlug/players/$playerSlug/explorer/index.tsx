/**
 * Route /players/$playerSlug/explorer — page Explorer.
 *
 * `validateSearch` couvre `mode`/`target` (toggle + cible joueur) ET tous les
 * filtres de scope (dates, playlists, tiers, recherche, tri) — cf.
 * `explorerSearchSchema` dans `@/features/explorer/explorerScope`. Cela rend
 * les filtres durables dans l'URL (survit retour navigateur / F5 / partage).
 */
import { createFileRoute } from '@tanstack/react-router'
import { ExplorerPage } from '@/features/explorer/ExplorerPage'
import { explorerSearchSchema } from '@/features/explorer/explorerScope'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/explorer/')({
  validateSearch: explorerSearchSchema,
  component: ExplorerPage,
})
