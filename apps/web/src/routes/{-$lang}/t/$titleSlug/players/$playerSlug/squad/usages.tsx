import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection : l'onglet Usages de l'Escouade s'appelle « Emprise » depuis le 2026-09-27 (D1
// du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Les liens existants restent valides :
// mêmes paramètres de chemin, et la recherche (`session`, `teammates` du layout Escouade) est
// transmise telle quelle.
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/squad/usages')({
  beforeLoad: ({ params, search }) => {
    throw redirect({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/emprise',
      params,
      search,
      replace: true,
    })
  },
})
