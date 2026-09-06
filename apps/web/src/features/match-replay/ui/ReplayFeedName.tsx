/**
 * ReplayFeedName — UN NOM DANS LE FIL DES MORTS : sa marque d'identité, son encre.
 *
 * POURQUOI CE FICHIER EXISTE. Le fil écrit cinq noms — tueur, victime, défunt d'une mort
 * neutre, décoré d'une médaille seule, assistant — et les cinq portaient EXACTEMENT le même
 * `<span>` coloré. Centraliser leur rendu évite que cinq copies divergent au premier oubli
 * (CLAUDE.md n°6).
 *
 * PLUS AUCUN GLYPHE DANS LE FIL (décision D5, 2026-09-02). Le fil a porté deux marques
 * successives devant un nom : le rond du joueur actif — retiré le 2026-08-18 (retour C1,
 * « il y a un symbole rond dans un cercle affiché, je sais pas ce que c'est ») — puis le
 * glyphe « ami » (deux silhouettes, ex-`PlayerMark.tsx`) qui lui survivait seul. Le
 * 2026-09-02 le user tranche : plus un seul glyphe au fil, quelle que soit la marque. Les
 * FICHES et la CARTE gardent leur PROPRE grammaire de formes (losange/anneau/cercle sur le
 * marqueur de carte, `replayMarkers.ts`) — une grammaire distincte, jamais partagée avec le
 * fil, qui ne bouge pas.
 *
 * L'ENCRE `success`, ELLE, RESTE. C'est le vert demandé le 18/08 pour le marqueur de carte,
 * porté ici sur le fil : « colorer le joueur actif et ses amis ». Sans glyphe, c'est
 * désormais le SEUL signe qui distingue un nom marqué — retirer aussi l'encre effacerait une
 * information plutôt que l'alléger. Elle remplace la couleur d'ÉQUIPE sur ce nom seulement —
 * la teinte de l'icône d'arme continue de dire le camp, la ligne ne perd donc rien.
 *
 * ACCESSIBILITÉ : retirer un glyphe ne doit pas retirer une information. Le joueur de la page
 * garde son libellé pour les lecteurs d'écran (`sr-only`) — à l'écran il n'y a plus qu'une
 * couleur, dans la lecture il y a toujours le mot.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PlayerMarkKind } from '../playerMarks'

interface Props {
  /** Marque d'identité du joueur, ou rien — décide de l'encre (`feedNameInk`), plus d'un glyphe. */
  kind: PlayerMarkKind | undefined
  name: string
  /**
   * Couleur d'équipe du joueur. Absente = le nom prend l'encre de son parent (c'est le cas
   * de l'assistant, écrit en encre atténuée dans la rangée). Un joueur MARQUÉ ignore les
   * deux et passe au token `success`.
   */
  color?: string
  locale: ReplayLocale
  /** Classes de mise en page propres à l'emplacement (`font-medium` sur un acteur). */
  className?: string
}

/** feedNameInk — l'encre d'un nom du fil : `success` s'il est marqué, sa couleur sinon. */
export function feedNameInk(
  kind: PlayerMarkKind | undefined,
  color: string | undefined,
): string | undefined {
  return kind ? tokenCssVar('success') : color
}

export function FeedName({ kind, name, color, locale, className }: Props) {
  const ink = feedNameInk(kind, color)
  return (
    <span className={`truncate ${className ?? ''}`} style={ink ? { color: ink } : undefined}>
      {name}
      {kind === 'me' && <span className="sr-only">{` (${REPLAY_TEXT[locale].markMe})`}</span>}
    </span>
  )
}
