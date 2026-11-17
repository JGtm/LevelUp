/**
 * TacticalReapparitionsZones — la COUCHE SVG des zones nommées de la lecture « Réapparitions adverses »,
 * posée sur le fond du plan : chaque zone est teintée selon la part des réapparitions adverses qui y
 * tombent, la zone tenue est hachurée, et un clic sur une zone la choisit comme zone tenue.
 *
 * ─── MÊME BOÎTE QUE L'IMAGE DU FOND ────────────────────────────────────────────────────────────
 * Le `<svg>` reçoit le MÊME style de placement que l'`<img>` du fond (`cadrageDuFond` : pourcentages du
 * cadre et rotation de la carte) et son `viewBox` est le cadre monde de cette image (`cadreDeLaCouche`,
 * Y inversé) : les zones, projetées en mètres monde, tombent sur le fond sans autre calcul, et tournent avec
 * lui.
 *
 * ─── ROGNÉES AU PLAN ───────────────────────────────────────────────────────────────────────────
 * Quand la carte a une vue plan, l'alpha de l'image plan rogne les formes (`lib/replay/planMask`, le même
 * masque que le calque des zones du rejeu 2D) : une zone n'est peinte que là où le plan existe, sur le fond
 * plan comme sur le rendu. Carte sans vue plan : zones entières. Tant que la décision n'est pas prise
 * (`plan.decide`), rien n'est peint : les zones ne se montrent pas entières un instant puis rognées.
 *
 * Les couleurs sont des jetons (`team-enemy`, `team-ally`) ou le gris du texte atténué ; la teinte est une
 * OPACITÉ de remplissage (`opaciteDeLaTeinte`), jamais une couleur écrite.
 */
import { useId } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Locale } from '@/lib/i18n/locale'
import { zoneHasShape } from '@/lib/replay/calloutShape'
import { calloutLabel, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import { PlanMaskDef } from '@/lib/replay/PlanMaskDef'
import { planMaskId, planMaskUrl } from '@/lib/replay/planMask'

import type { CadrageDuFond } from './plan.logic'
import type { PlanDuRognage } from './queriesReapparitions'
import { cheminDeLaZone, opaciteDeLaTeinte, type LectureReapparitions } from './reapparitions.logic'

/** Le côté d'un motif de hachure, en fraction de la largeur du cadre monde. */
const HACHURE_FRACTION = 1 / 80

export interface TacticalReapparitionsZonesProps {
  zones: readonly CalloutZoneReady[]
  /** La lecture à teindre ; `null` tant qu'elle n'est pas là (les zones se peignent alors sans teinte). */
  lecture: LectureReapparitions | null
  /** Le cadre monde de l'image du fond (`cadreDeLaCouche`). */
  cadre: { x: number; y: number; width: number; height: number }
  /** La place de l'image dans le cadre du plan, rotation comprise : celle de l'`<img>` du fond. */
  cadrage: CadrageDuFond
  plan: PlanDuRognage
  locale: Locale
  /** L'indice de volume de la zone tenue choisie ; `null` = aucune. */
  choisie: number | null
  onChoisir: (volumeIndex: number) => void
  estompe: string
}

/** Les grandes zones d'abord (dessous), les fines ensuite : une zone fine imbriquée reste visible et cliquable. */
function zonesDansLOrdre(zones: readonly CalloutZoneReady[]): CalloutZoneReady[] {
  return zones
    .filter(zoneHasShape)
    .sort((a, b) => Number(b.big) - Number(a.big) || a.volumeIndex - b.volumeIndex)
}

/** Le style de remplissage d'une zone : hachurée (tenue), teintée (part) ou transparente (cliquable seulement). */
function styleDeLaZone(
  lecture: LectureReapparitions | null,
  volumeIndex: number,
  choisie: number | null,
  motif: string,
) {
  if (volumeIndex === choisie) {
    return { fill: `url(#${motif})`, fillOpacity: 1, stroke: tokenCssVar('team-ally'), strokeOpacity: 1, strokeWidth: 2 }
  }
  const ligne = lecture?.lignes.find((l) => l.volumeIndex === volumeIndex)
  if (!lecture || !ligne) return { fill: 'transparent' }
  if (lecture.etat === 'servie') {
    return { fill: tokenCssVar('team-enemy'), fillOpacity: opaciteDeLaTeinte(ligne.tenue ?? 0, lecture.maxTenue, false) }
  }
  return { fill: 'currentColor', fillOpacity: opaciteDeLaTeinte(ligne.reference, lecture.maxReference, true) }
}

export function TacticalReapparitionsZones({
  zones,
  lecture,
  cadre,
  cadrage,
  plan,
  locale,
  choisie,
  onChoisir,
  estompe,
}: TacticalReapparitionsZonesProps) {
  const base = planMaskId(useId())
  const motif = `${base}-hachure`
  if (!plan.decide) return null
  const cote = cadre.width * HACHURE_FRACTION
  return (
    <svg
      viewBox={`${cadre.x} ${cadre.y} ${cadre.width} ${cadre.height}`}
      preserveAspectRatio="none"
      className={`pointer-events-none absolute max-w-none text-muted-foreground${estompe}`}
      style={cadrage}
      aria-hidden
      data-testid="tactical-reapp-zones"
      data-rogne={plan.href ? 'plan' : 'non'}
    >
      <defs>
        {plan.href && <PlanMaskDef id={base} href={plan.href} cadre={cadre} />}
        <pattern id={motif} width={cote} height={cote} patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
          <rect width={cote} height={cote} style={{ fill: tokenCssVar('team-ally'), fillOpacity: 0.14 }} />
          <rect width={cote * 0.3} height={cote} style={{ fill: tokenCssVar('team-ally'), fillOpacity: 0.6 }} />
        </pattern>
      </defs>
      <g mask={plan.href ? planMaskUrl(base) : undefined}>
        {zonesDansLOrdre(zones).map((z) => (
          <path
            key={z.volumeIndex}
            d={cheminDeLaZone(z)}
            fillRule="evenodd"
            vectorEffect="non-scaling-stroke"
            className="pointer-events-auto cursor-pointer stroke-foreground transition-[fill-opacity] duration-500 [stroke-opacity:0.16] hover:[stroke-opacity:0.7]"
            style={{ strokeWidth: 1, ...styleDeLaZone(lecture, z.volumeIndex, choisie, motif) }}
            data-zone={z.volumeIndex}
            onClick={() => onChoisir(z.volumeIndex)}
          >
            <title>{calloutLabel(z, locale)}</title>
          </path>
        ))}
      </g>
    </svg>
  )
}
