/**
 * ReplayWeaponPadTip — L'INFOBULLE D'UN EMPLACEMENT D'ARME, au survol de son anneau.
 *
 * CE QU'ELLE DIT, ET DANS CET ORDRE : l'ARME (nom bilingue du catalogue, ou son identifiant
 * quand rien ne la nomme — jamais le nom d'une arme voisine), son ÉTAT à cet instant, le
 * COMPTE À REBOURS quand un cycle établi le permet, et la réserve de lecture : la mesure ne
 * distingue pas un socle au sol d'un râtelier mural.
 *
 * UN SOCLE DE POWER-UP N'EST PAS UNE ARME (schéma 17) : son nom vient de la table des familles
 * non-arme (`padNameFor`), jamais de la clé brute du document, et sa réserve de lecture change
 * avec lui — le râtelier mural n'a pas de sens pour un objet qu'on ramasse au sol.
 *
 * CE QU'ELLE NE DIT JAMAIS : QUI a pris l'arme. Le champ existe au contrat (`padPickups[].xuid`)
 * et vaut `null` partout — l'oracle plafonne à 79,7 % contre 90 % exigés. C'est la clause la
 * plus facile à violer par inadvertance, puisque le champ SE LIT sans qu'on voie qu'il est vide :
 * ce composant ne le lit pas.
 *
 * PAS DE CYCLE = PAS DE LIGNE. Une famille sans cycle établi n'affiche ni chiffre ni tiret : un
 * tiret suggérerait qu'on saurait.
 *
 * NI MÉDIANE NI ÉCARTS (verdict du 2026-08-18) : l'infobulle portait aussi la médiane du cycle
 * et ses deux dénominateurs (écarts mesurés, écarts manqués). Ces trois nombres disaient la
 * CONFIANCE dans le cycle — une lecture d'analyse, pas un repère de carte. Le compte à rebours,
 * lui, répond à la seule question qu'on se pose devant un socle vide : dans combien de temps.
 * La réserve n'est pas perdue pour autant : elle vit dans le « ≈ » du libellé, et le compte ne
 * s'affiche QUE quand le cycle est établi.
 *
 * Purement présentationnel : l'état et le compte à rebours sont calculés au survol
 * (useReplayWeaponPads), la géométrie dans weaponPadsLayer.
 */
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { WeaponPadHover } from './useReplayWeaponPads'
import { padEquipmentFamilyOf } from './weaponPadFamilies'

/** Décalage de l'infobulle sous le pointeur, en pixels (même valeur que celle des poses). */
const TIP_OFFSET = 12
/** Largeur estimée : elle sert UNIQUEMENT à décider du côté, jamais à contraindre le rendu. */
const TIP_WIDTH = 192

interface ReplayWeaponPadTipProps {
  locale: ReplayLocale
  hover: WeaponPadHover
  /** Largeur du canvas : elle borne l'infobulle du côté droit. */
  width: number
}

export function ReplayWeaponPadTip({ locale, hover, width }: ReplayWeaponPadTipProps) {
  const t = REPLAY_TEXT[locale]
  const { at, name, state, respawnS } = hover
  // LA RÉSERVE SUIT LA NATURE DU SOCLE : « socle au sol ou râtelier mural » n'a aucun sens
  // pour un power-up — il ne s'accroche pas à un mur. Même table que la taille et le nom.
  const note = padEquipmentFamilyOf(hover.pad.weapon)
    ? t.padPlacementNotePowerUp
    : t.padPlacementNote
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
      <span className="block font-medium">{name}</span>
      <span className="block text-muted-foreground">
        {t.padState[state]}
        {respawnS !== null ? ` · ${t.padRespawnFmt(respawnS)}` : ''}
      </span>
      <span className="mt-0.5 block text-[0.65rem] text-muted-foreground opacity-80">
        {note}
      </span>
    </div>
  )
}
