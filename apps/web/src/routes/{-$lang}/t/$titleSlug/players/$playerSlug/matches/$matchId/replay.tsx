/**
 * Route /{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay — Rejeu 2D
 * du match (vue du dessus).
 *
 * Trajectoires des joueurs décodées du film, servies par l'API comme artefact
 * pré-construit (GET .../replay). La disponibilité est PAR MATCH : 404 → état vide.
 * Ce n'est donc pas un drapeau global — le stub `REJEU_2D_ENABLED` qui occupait cette
 * route est remplacé par l'implémentation réelle.
 *
 * DEUX PORTES DE TITRE, ET ELLES NE DISENT PAS LA MÊME CHOSE (2026-09-06, v2 D.14) :
 *  - `matchmaking` : sans elle, le match n'existe pas pour ce titre, donc son rejeu non plus ;
 *  - `replay` : le titre a des matchs, mais aucune chaîne de rejeu (capability title-level du
 *    lot C, servie par le bootstrap). Halo 5 est dans ce cas.
 * La porte est posée AU-DESSUS du composant : la page n'est pas montée, donc aucun de ses
 * hooks de chargement ne tourne — ni l'artefact (1,5 à 2,7 Mio), ni le contexte du match, ni le fond
 * de carte. Un titre sans rejeu ne paie aucune requête, et lit un état qui le dit
 * (`FeatureUnavailable`, libellé `replay` FR/EN).
 *
 * `?t=<instant_ms>&clock=match|film` OUVRE LE REJEU À L'INSTANT EXACT (lot M1b, 2026-09-08,
 * décision utilisateur ferme « corriger le décalage ») — lien posé par la tuile « Rejeu » de la
 * zone choisie de l'onglet Tactique (`MatchReplayLink`). La route attend le document (il porte
 * l'origine du film), convertit avec la fonction pure `resolveTacticalOpenAtFrame`
 * (`lib/replay/replayLogic.ts` : `matchClock(doc).filmMsOfMatchMs` pour l'horloge du match, puis
 * recul de `OUVERTURE_AVANT_EVENEMENT_MS`), et positionne `ReplayCanvas` via `openAtFrame`.
 * `clock=match` sur un artefact sans origine du film : le rejeu s'ouvre au début et affiche un
 * avis — JAMAIS un saut approximatif présenté comme exact.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { ReplayPage } from '@/features/match-replay/ReplayPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
)({
  // `t` (instant_ms) + `clock` : cf. l'en-tête. Une valeur hostile ou mal formée retombe sur
  // `undefined` (`.catch`) plutôt que de faire échouer la route — un lien cassé ouvre le
  // rejeu au début, comme sans paramètre du tout, jamais une page en erreur.
  //
  // `t` RESTE UNE CHAÎNE ICI, PAS UN NOMBRE : `FullSearchSchema` (TanStack Router) fusionne
  // les schémas de TOUTES les routes, et `HelpPage.tsx`/`SettingsPage.tsx` construisent un
  // `URLSearchParams` depuis `routerState.location.search` en supposant CHAQUE valeur déjà
  // une chaîne — un champ numérique ici casse leur build, sans rapport avec ce lot. La
  // conversion en nombre se fait au composant (`Number(search.t)`), une fois validée.
  //
  // ...MAIS L'ENTRÉE ACCEPTE AUSSI UN NOMBRE : le routeur lit chaque valeur de recherche en
  // JSON, donc un lien écrit à la main `?t=612000` arrive comme le NOMBRE 612000. Refusé par
  // `z.string()`, il tombait dans `.catch` et le rejeu s'ouvrait au début sans un mot. Les
  // liens de l'app (`MatchReplayLink`) passent par le sérialiseur, qui cite la chaîne
  // (`?t=%224200%22`) ; les deux formes donnent ici la même chaîne. Le type de SORTIE, seul
  // fusionné dans `FullSearchSchema`, reste `string`.
  validateSearch: z.object({
    t: z
      .union([z.string(), z.number()])
      .transform((v) => String(v))
      .optional()
      .catch(undefined),
    clock: z.enum(['match', 'film']).optional().catch(undefined),
  }),
  component: () => (
    <RouteCapabilityGate capability="matchmaking">
      <RouteCapabilityGate capability="replay">
        <ReplayPage />
      </RouteCapabilityGate>
    </RouteCapabilityGate>
  ),
})
