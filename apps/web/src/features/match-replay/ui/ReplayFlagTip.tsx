/**
 * ReplayFlagTip — L'INFOBULLE D'UN DRAPEAU DE CTF, au survol de son glyphe.
 *
 * CE QU'ELLE DIT, ET DANS CET ORDRE : de QUEL drapeau il s'agit (allié, adverse — jamais une
 * couleur d'équipe nommée, la page n'en connaît pas), son ÉTAT à cet instant, QUI le porte quand
 * quelqu'un le porte, et DEPUIS QUAND cet état dure.
 *
 * LA RÉSERVE DE `carried_open` EST À L'ÉCRAN, pas dans un commentaire : quand rien ne date la fin
 * du portage, l'intervalle court jusqu'à la fin du film et c'est une BORNE HAUTE. Le glyphe la
 * porte par sa FORME — un fanion CREUX — pendant que son battement dit tout autre chose (il est
 * hors de sa base, comme tout objet sorti depuis le lot du 2026-08-27, qui a retiré l'atténuation
 * de cet état au profit du clignotement). L'infobulle, elle, la dit EN TOUTES LETTRES, parce
 * qu'une forme seule se lit comme un effet de style.
 *
 * LA JAUGE DE RETOUR N'EST PAS UN COMPTE À REBOURS, ET LA LIGNE LE DIT COMME TELLE (schéma 63).
 * Un drapeau au sol rentre chez lui quand une JAUGE se remplit — au taux `1/reset + H(n)/solo`,
 * série harmonique du nombre de défenseurs — et cette jauge SE VIDE quand plus personne n'est
 * dans la zone. Afficher « retour dans N s » serait donc un mensonge mesurable : sur le corpus,
 * deux lâchers de même durée montent à 1,0000 et à 0,3209. La ligne affiche donc le REMPLISSAGE,
 * en pourcentage, et rien d'autre.
 *
 * ELLE NE S'AFFICHE QUE SUR UN DRAPEAU AU SOL, et seulement quand l'artefact porte la jauge :
 * un artefact antérieur à 63, ou un intervalle que le film n'a pas couvert, n'a pas de ligne —
 * pas de tiret, qui suggérerait qu'on saurait.
 *
 * UN ÉTAT INCONNU N'INVENTE PAS DE LIBELLÉ : un artefact plus récent que ce client peut publier
 * un état que la table de texte ne couvre pas — la ligne est alors omise plutôt que remplie d'un
 * identifiant brut ou d'un état voisin.
 *
 * Purement présentationnel : l'état, le porteur et la durée sont calculés au survol
 * (useReplayFlagCarries), la géométrie dans flagCarriesLayer.
 */
import type { FlagState } from '../layers/flagCarriesLayer'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { FlagHover } from '../layers/useReplayFlagCarries'

/** Décalage de l'infobulle sous le pointeur, en pixels (même valeur que celle des socles). */
const TIP_OFFSET = 12
/** Largeur estimée : elle sert UNIQUEMENT à décider du côté, jamais à contraindre le rendu. */
const TIP_WIDTH = 192

interface ReplayFlagTipProps {
  locale: ReplayLocale
  hover: FlagHover
  /** Largeur du canvas : elle borne l'infobulle du côté droit. */
  width: number
}

export function ReplayFlagTip({ locale, hover, width }: ReplayFlagTipProps) {
  const t = REPLAY_TEXT[locale]
  const { at, now, side, carrier, sinceMs } = hover
  const state = t.flagState[now.state as FlagState] as string | undefined
  const carried = now.state === 'carried' || now.state === 'carried_open'
  const flip = at.x + TIP_OFFSET + TIP_WIDTH > width
  return (
    <div
      role="tooltip"
      className="pointer-events-none absolute z-10 max-w-[13rem] rounded border border-border bg-card px-2 py-1 text-xs shadow-lg"
      style={{
        left: flip ? undefined : at.x + TIP_OFFSET,
        right: flip ? Math.max(width - at.x + TIP_OFFSET, 0) : undefined,
        top: at.y + TIP_OFFSET,
      }}
    >
      <span className="block font-medium">{t.flagSide[side]}</span>
      {state && (
        <span className="block text-muted-foreground">
          {state}
          {carried ? ` · ${carrier ?? t.flagCarrierUnknown}` : ''}
        </span>
      )}
      <span className="block text-muted-foreground">{t.flagSinceFmt(sinceMs / 1000)}</span>
      {now.state === 'dropped' && now.returnProgress !== null && (
        <span className="block text-muted-foreground">{t.flagReturnFmt(now.returnProgress)}</span>
      )}
      {now.state === 'carried_open' && (
        <span className="mt-0.5 block text-[0.65rem] text-muted-foreground opacity-80">
          {t.flagOpenNote}
        </span>
      )}
    </div>
  )
}
