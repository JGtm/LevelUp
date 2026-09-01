/**
 * ReplayWeaponPadTip — L'INFOBULLE D'UN EMPLACEMENT D'ARME, au survol de sa pile.
 *
 * DEUX LIGNES AU PLUS, ET C'EST TOUT LE CONTRAT (retour utilisateur du 2026-08-28 : « je veux
 * juste garder le nom de l'arme ou de l'équipement, et à la rigueur son timer ; je ne veux pas
 * de blabla dedans ») : l'ARME (nom bilingue du catalogue, ou son identifiant quand rien ne la
 * nomme — jamais le nom d'une arme voisine), puis le COMPTE À REBOURS quand une source le
 * permet. L'infobulle portait en plus l'ÉTAT du socle et une NOTE DE LECTURE de deux lignes ;
 * l'état se lit déjà sur la carte (le losange s'atténue, la vignette disparaît) et la note
 * répétait à chaque survol une réserve qui vit dans l'aide du calque.
 *
 * UN SOCLE DE POWER-UP N'EST PAS UNE ARME (schéma 17) : son nom vient de la table des familles
 * non-arme (`padNameFor`), jamais de la clé brute du document.
 *
 * CE QU'ELLE NE DIT JAMAIS : QUI a pris l'arme. Le champ existe au contrat (`padPickups[].xuid`)
 * et est PUBLIÉ depuis le schéma 30 (2026-08-31) — l'événement natif le porte —, mais cette
 * infobulle ne l'affiche pas : son dessin n'a pas été repensé (le ramasseur nommé est le sujet
 * du tableau « Contrôle des armes spéciales » de la page match). C'est la clause la
 * plus facile à violer par inadvertance, puisque le champ SE LIT sans qu'on voie qu'il est vide :
 * ce composant ne le lit pas.
 *
 * DEUX COMPTES À REBOURS, ET ELLE DOIT LES SÉPARER (D3, 2026-08-27). Celui qui vise la prochaine
 * apparition VUE DANS LE FILM est EXACT — le rejeu connaît la suite — et se dit tel quel ; celui
 * que le CYCLE prédit, pour le dernier trou qu'aucune apparition ne ferme, garde son « ≈ ». LA
 * RÉSERVE TIENT DANS CE SEUL SIGNE depuis le 2026-08-28 : les deux libellés portaient une
 * parenthèse d'explication (« vue dans le film » / « cycle attendu ») — c'était le blabla que le
 * retour retire, et le « ≈ » dit la même chose en un caractère.
 *
 * PAS DE SOURCE = PAS DE LIGNE. Ni apparition suivante, ni cycle établi : ni chiffre, ni tiret —
 * un tiret suggérerait qu'on saurait.
 *
 * Purement présentationnel : l'état et le compte à rebours sont calculés au survol
 * (useReplayWeaponPads), la géométrie dans weaponPadsLayer.
 */
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { WeaponPadHover } from './useReplayWeaponPads'

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
  const { at, name, respawn } = hover
  const respawnText = respawn
    ? (respawn.measured ? t.padRespawnMeasuredFmt : t.padRespawnExpectedFmt)(respawn.seconds)
    : null
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
      {respawnText !== null && <span className="block text-muted-foreground">{respawnText}</span>}
    </div>
  )
}
