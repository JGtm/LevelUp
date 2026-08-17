/**
 * ReplayPlacementTip — L'INFOBULLE D'UNE POSE D'ÉQUIPEMENT, au survol du marqueur.
 *
 * CE QU'ELLE DIT, ET DANS CET ORDRE : ce que l'objet EST (« Mur de protection », « Capteur de
 * menaces », ou « objet non identifié » quand le manifeste ne le nomme pas), puis QUI l'a posé.
 * Le poseur est une MESURE — le bipède le plus proche à 250 ms et moins de 3 m — et quand il
 * n'y en a pas, l'infobulle le dit au lieu de laisser un blanc.
 *
 * PAS DE `title` NATIF : le canvas est UNE seule balise, son attribut `title` vaudrait pour
 * toute la carte. L'infobulle est donc un élément posé dans le cadre du canvas, à côté du
 * pointeur, et repoussé à l'intérieur quand elle déborderait du bord (une infobulle coupée par
 * le cadre ne se lit pas).
 *
 * Purement présentationnel : aucune donnée n'est calculée ici (le survol vit dans
 * usePlacementHover, la géométrie dans equipmentPlacementsLayer).
 */
import { PLACEMENT_RENDER } from './equipmentPlacementsLayer'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { PlacementHover } from './usePlacementHover'

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
}

export function ReplayPlacementTip({ locale, hover, ownerName, width }: ReplayPlacementTipProps) {
  const t = REPLAY_TEXT[locale]
  const { placement, at } = hover
  const kind = PLACEMENT_RENDER[placement.family]
  const title =
    kind === 'wall' || kind === 'sensor'
      ? t.placementFamily[kind]
      : t.placementUnnamedLabel
  const owner =
    placement.owner >= 0 && ownerName
      ? t.placementOwnerFmt(ownerName)
      : t.placementOwnerUnknown
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
    </div>
  )
}
