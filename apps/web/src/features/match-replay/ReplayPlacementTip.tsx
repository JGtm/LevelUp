/**
 * ReplayPlacementTip — L'INFOBULLE D'UNE POSE D'ÉQUIPEMENT, au survol du marqueur.
 *
 * CE QU'ELLE DIT, ET DANS CET ORDRE : ce que l'objet EST (« Mur de protection », « Traqueur de
 * menaces », « Champ de réparation »… ou « objet non identifié » quand le manifeste ne le nomme
 * pas), puis QUI l'a posé. Le poseur est une MESURE — le bipède le plus proche à 250 ms et moins
 * de 3 m — et quand il n'y en a pas, l'infobulle le dit au lieu de laisser un blanc.
 *
 * LE NOM VIENT DE LA RÈGLE DE RENDU, PAS DE LA FAMILLE, et c'est la même table que le tracé :
 * une pose qu'on ne dessine pas ne se survole pas, donc son nom n'a jamais à être servi ici.
 *
 * UN OBJET LÂCHÉ SE LIT AUTREMENT, et l'infobulle est le seul endroit où il se nomme (sa forme,
 * elle, est la même pour toutes les familles). Trois différences, et chacune corrige une
 * inexactitude : le NOM vient de la FAMILLE et non de la règle de rendu — un power-up n'a
 * aucune règle de rendu d'objet actif et resterait « non identifié » ; le VERBE devient
 * « lâché par », parce qu'un objet tombé d'un cadavre n'a pas été posé ; et une TROISIÈME
 * ligne donne l'instant du lâcher au chronomètre du rejeu.
 *
 * PAS DE `title` NATIF : le canvas est UNE seule balise, son attribut `title` vaudrait pour
 * toute la carte. L'infobulle est donc un élément posé dans le cadre du canvas, à côté du
 * pointeur, et repoussé à l'intérieur quand elle déborderait du bord (une infobulle coupée par
 * le cadre ne se lit pas).
 *
 * Purement présentationnel : aucune donnée n'est calculée ici (le survol vit dans
 * usePlacementHover, la géométrie dans equipmentPlacementsLayer).
 */
import { PLACEMENT_RENDER, type PlacementKind } from './equipmentPlacementsLayer'
import { PLACEMENT_ORIGIN_DROPPED } from './placementDropped'
import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import { formatClock } from './replayLogic'
import { displayClockMs, type ReplayWindowBounds } from './replayWindow'
import type { PlacementHover } from './usePlacementHover'
import { padEquipmentFamilyOf } from './weaponPadFamilies'

/** Décalage de l'infobulle sous le pointeur, en pixels. */
const TIP_OFFSET = 12
/** Largeur estimée : elle sert UNIQUEMENT à décider du côté, jamais à contraindre le rendu. */
const TIP_WIDTH = 168

interface ReplayPlacementTipProps {
  locale: ReplayLocale
  hover: PlacementHover
  /** Nom du poseur, déjà résolu par slot (useSlotIdentity) ; null = vie sans propriétaire. */
  ownerName: string | null
  /** Largeur du canvas : elle borne l'infobulle du côté droit. */
  width: number
  /** La fenêtre de gameplay : l'instant du lâcher se lit sur l'horloge du match (D-A2). */
  playWindow: ReplayWindowBounds | null
}

/**
 * droppedTitle — le nom d'un objet LÂCHÉ, cherché dans les deux vocabulaires du titre.
 *
 * Un power-up n'a pas de règle de rendu d'objet actif (il n'est pas dans `PLACEMENT_RENDER`) :
 * son nom vient de la table des familles NON-ARME des socles, la même qui nomme le surbouclier
 * sous sa vignette. Un équipement déployable, lui, garde le nom de sa règle. Ce qui n'est ni
 * l'un ni l'autre rend le libellé générique — jamais l'identifiant brut.
 */
function droppedTitle(family: string, t: (typeof REPLAY_TEXT)['fr']): string {
  const powerUp = padEquipmentFamilyOf(family)
  if (powerUp) return t.padEquipmentFamily[powerUp]
  const kind = PLACEMENT_RENDER[family]
  return kind && kind !== 'unnamed' && kind !== 'dropped'
    ? t.placementFamily[kind]
    : t.placementDroppedLabel
}

/** namedKind — la règle de rendu quand elle porte un nom de famille, sinon null. */
function namedKind(kind: PlacementKind | null | undefined) {
  return kind && kind !== 'unnamed' && kind !== 'dropped' ? kind : null
}

export function ReplayPlacementTip({
  locale,
  hover,
  ownerName,
  width,
  playWindow,
}: ReplayPlacementTipProps) {
  const t = REPLAY_TEXT[locale]
  const { placement, at, atMs } = hover
  const kind = PLACEMENT_RENDER[placement.family]
  const dropped = placement.origin === PLACEMENT_ORIGIN_DROPPED
  const named = namedKind(kind)
  const title = dropped
    ? droppedTitle(placement.family, t)
    : named
      ? t.placementFamily[named]
      : t.placementUnnamedLabel
  const ownerFmt = dropped ? t.placementDroppedOwnerFmt : t.placementOwnerFmt
  const owner =
    placement.owner >= 0 && ownerName ? ownerFmt(ownerName) : t.placementOwnerUnknown
  const flip = at.x + TIP_OFFSET + TIP_WIDTH > width
  return (
    <div
      role="tooltip"
      className="pointer-events-none absolute z-10 max-w-[11rem] rounded border border-border bg-card px-2 py-1 text-xs shadow-lg"
      style={{
        left: flip ? undefined : at.x + TIP_OFFSET,
        right: flip ? Math.max(width - at.x + TIP_OFFSET, 0) : undefined,
        top: at.y + TIP_OFFSET,
      }}
    >
      <span className="block font-medium">{title}</span>
      <span className="block text-muted-foreground">{owner}</span>
      {dropped && (
        <span className="block text-muted-foreground">
          {t.placementDroppedAtFmt(formatClock(displayClockMs(atMs, playWindow)))}
        </span>
      )}
    </div>
  )
}
