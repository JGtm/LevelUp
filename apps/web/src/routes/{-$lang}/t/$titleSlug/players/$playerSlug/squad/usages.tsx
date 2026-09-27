import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection : l'onglet Usages de l'Escouade s'appelle « Emprise » depuis le 2026-09-27 (D1
// du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Les liens existants restent valides :
// mêmes paramètres de chemin, et la recherche (`session`, `teammates` du layout Escouade) est
// transmise telle quelle.
// Retrait : cible 2027-01-01 (décision superviseur du 2026-09-27), critère mesurable — aucune
// requête `/squad/usages` dans les journaux HTTP des 30 jours précédents. Ce jour-là, supprimer
// ce fichier, son test et l'assertion de `shellNavigation.test.ts` qui cite l'ancien chemin.
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
