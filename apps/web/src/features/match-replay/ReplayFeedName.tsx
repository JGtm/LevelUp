/**
 * ReplayFeedName — UN NOM DANS LE FIL DES MORTS : sa marque d'identité, son encre.
 *
 * POURQUOI CE FICHIER EXISTE. Le fil écrit cinq noms — tueur, victime, défunt d'une mort
 * neutre, décoré d'une médaille seule, assistant — et les cinq portaient EXACTEMENT le même
 * couple `<PlayerMark/>` + `<span>` coloré. Le retour C1 du 2026-08-18 change la règle des
 * deux à la fois : cinq copies auraient divergé au premier oubli (CLAUDE.md n°6).
 *
 * LES DEUX RÈGLES, ET CE QU'ELLES CORRIGENT :
 *
 *  1. LE GLYPHE « JOUEUR ACTIF » NE S'AFFICHE PLUS ICI. « Il y a un symbole rond dans un
 *     cercle affiché, je sais pas ce que c'est » — le lecteur ne l'a pas reconnu, et l'enquête
 *     du lot R2-V a montré que ce rond dans un cercle était bien ce glyphe (`PlayerMark`,
 *     forme `me`). Sur SON propre fil, savoir lequel des noms est le sien ne vaut pas un
 *     signe de plus par ligne. La marque « ami », elle, RESTE : elle distingue des gens dont
 *     rien d'autre ne dit qu'on les connaît. Les FICHES et la CARTE gardent les deux marques
 *     (grammaire D5) — c'est le fil, et lui seul, qui allège.
 *
 *  2. LE NOM D'UN JOUEUR MARQUÉ PASSE AU TOKEN `success`. C'est le vert demandé le 18/08 pour
 *     le marqueur de carte, porté ici sur le fil : « colorer le joueur actif et ses amis ».
 *     Il remplace la couleur d'ÉQUIPE sur ce nom seulement — le liseré gauche de la ligne et
 *     la teinte de l'icône d'arme continuent de dire le camp, la ligne ne perd donc rien.
 *
 * ACCESSIBILITÉ : retirer un glyphe ne doit pas retirer une information. Le joueur de la page
 * garde son libellé pour les lecteurs d'écran (`sr-only`) — à l'écran il n'y a plus qu'une
 * couleur, dans la lecture il y a toujours le mot.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { PlayerMarkKind } from './playerMarks'
import { PlayerMark } from './PlayerMark'

interface Props {
  /** Marque d'identité du joueur, ou rien. */
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
    <>
      {/* Le glyphe « ami » seulement : cf. règle 1 de l'en-tête. */}
      <PlayerMark kind={kind === 'me' ? undefined : kind} locale={locale} />
      <span className={`truncate ${className ?? ''}`} style={ink ? { color: ink } : undefined}>
        {name}
        {kind === 'me' && <span className="sr-only">{` (${REPLAY_TEXT[locale].markMe})`}</span>}
      </span>
    </>
  )
}
