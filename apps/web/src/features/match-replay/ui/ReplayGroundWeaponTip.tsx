/**
 * ReplayGroundWeaponTip — L'INFOBULLE D'UNE ARME AU SOL, au survol de sa vignette (lot 6.5,
 * 2026-09-10).
 *
 * UNE SEULE LIGNE, ET C'EST TOUT LE CONTRAT — même règle que `ReplayWeaponPadTip` (retour
 * utilisateur du 2026-08-28 : « je ne veux pas de blabla dedans »), et le rapport de ce lot le
 * redit noir sur blanc : « Une ligne, même mécanique que les autres calques de hoverLayers.ts. »
 * Le geste se lit : « <arme> · lâchée par X » puis « · reprise par Y » quand un ramasseur est
 * mesuré.
 *
 * LES TROIS FRAGMENTS SONT DÉJÀ COMPOSÉS PAR LE HOOK (`useReplayGroundWeapons`), qui seul
 * connaît la mesure (origine, résolution de joueur, `t0`/`t1`) : ce composant ne fait
 * qu'assembler `weaponName`, `originLine` et l'optionnel `pickupLine` avec le séparateur
 * visuel, exactement comme `ReplayCanvasTips` assemble les infobulles entre elles sans savoir
 * ce qu'elles affichent.
 */
import type { GroundWeaponHover } from '../layers/useReplayGroundWeapons'
import type { ReplayLocale } from '../i18n/i18n'

/** Décalage de l'infobulle sous le pointeur, en pixels — même valeur que les socles. */
const TIP_OFFSET = 12
/** Largeur estimée : elle sert UNIQUEMENT à décider du côté, jamais à contraindre le rendu. */
const TIP_WIDTH = 224

interface ReplayGroundWeaponTipProps {
  locale: ReplayLocale
  hover: GroundWeaponHover
  /** Largeur du canvas : elle borne l'infobulle du côté droit. */
  width: number
}

export function ReplayGroundWeaponTip({ hover, width }: ReplayGroundWeaponTipProps) {
  const { at, weaponName, originLine, pickupLine, ammoLine } = hover
  const flip = at.x + TIP_OFFSET + TIP_WIDTH > width
  // L'ORDRE DES FRAGMENTS EST CELUI DU GESTE : ce qu'est l'objet, qui l'a lâché, ce qu'il
  // restait dedans, qui l'a repris. Chaque fragment absent DISPARAÎT — jamais un séparateur
  // orphelin, jamais un « · ? ».
  const line = [weaponName, originLine, ammoLine, pickupLine].filter(Boolean).join(' · ')
  return (
    <div
      role="tooltip"
      className="pointer-events-none absolute z-10 max-w-[15rem] rounded border border-border bg-card px-2 py-1 text-xs shadow-lg"
      style={{
        left: flip ? undefined : at.x + TIP_OFFSET,
        right: flip ? Math.max(width - at.x + TIP_OFFSET, 0) : undefined,
        top: at.y + TIP_OFFSET,
      }}
    >
      <span className="block font-medium">{line}</span>
    </div>
  )
}
