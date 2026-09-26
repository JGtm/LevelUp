/**
 * presenceWording — LES MOTS D'UN ÉVÉNEMENT DE PRÉSENCE, choisis par sa SOURCE.
 *
 * # DEUX VOCABULAIRES ET PAS UN
 *
 * L'API AFFIRME (« a rejoint », « a quitté ») parce qu'elle porte un horodatage de
 * participation. Le repli film reste au FAIT (« entre en partie », « ne reviendra plus ») parce
 * qu'il DÉDUIT d'une borne de vie et ne distingue pas un départ d'une élimination définitive sur
 * un mode à manches. L'infobulle (`hint`) porte cette réserve. C'est la même règle que le bord de
 * l'ombre de la frise dessine : franc quand la source affirme, dégradé quand elle déduit.
 *
 * # UN SEUL FOYER, DEUX LECTEURS
 *
 * La ligne du fil (`ui/ReplayPresenceLine`) et la porte de la frise (`ui/ReplayPresenceShade`)
 * disent la MÊME chose du même événement. Écrite deux fois, la règle aurait fini corrigée d'un
 * seul côté — et l'infobulle de la frise aurait affirmé ce que le fil mettait au conditionnel.
 *
 * # POURQUOI DANS `model/` ET NON À CÔTÉ DU GLYPHE
 *
 * Elle a d'abord vécu dans `ui/PresenceGlyph.tsx`, avec le composant qu'elle sert (2026-09-07,
 * lot L4). Un module qui exporte un composant ET autre chose casse le rafraîchissement à chaud
 * de Vite (`react-refresh/only-export-components`), et la dette lint du dépôt est GELÉE : on ne
 * l'accroît pas d'un avertissement. Le lot L1 avait résolu exactement ce cas en sortant les
 * constantes de grille vers `replayTimelineGrid.ts` ; même remède ici. La frontière est d'ailleurs
 * la bonne : choisir des mots d'après une source de données est une décision de MODÈLE, pas un
 * détail de rendu — elle se teste sans monter le moindre composant.
 */
import type { ReplayText } from '../i18n/i18nContract'
import type { PresenceEvent } from './presenceFeed'

export function presenceWording(
  presence: Pick<PresenceEvent, 'kind' | 'source'>,
  t: ReplayText,
): { label: string; hint: string } {
  const joined = presence.kind === 'joined'
  const api = presence.source === 'api'
  return {
    label: joined
      ? (api ? t.presenceJoined : t.presenceJoinedDerived)
      : (api ? t.presenceLeft : t.presenceLeftDerived),
    hint: joined
      ? (api ? t.presenceJoinedHint : t.presenceJoinedDerivedHint)
      : (api ? t.presenceLeftHint : t.presenceLeftDerivedHint),
  }
}
