/**
 * ReplayCardFx — LES COUCHES D'EFFETS d'une tuile de joueur, indépendantes de sa forme : celle
 * posée SOUS le contenu (`CardUnderLayers` : éclats de mort et de réapparition, verre, voiles,
 * cadres, puis filigrane de porteur) et celle posée PAR-DESSUS (`ZoneFxOverlay` : nuage et
 * éclairs de l'écran occultant, contour du capteur, croix du champ de réparation, fourreau de
 * translocation).
 *
 * AUCUNE FORME DE TUILE ICI. Les deux couches s'étalent en `inset-0` sur leur parent (qui doit
 * être `relative`) et ne reçoivent que des cotes : le RAYON de la tuile, en classe (un littéral
 * en clair par gabarit, jamais interpolé — cf. `cardGabarit.ts`), le nombre d'éclairs et le côté
 * du filigrane, lus dans le gabarit par l'appelant. Une fiche de 115 × 62 et une ligne de 34 px
 * de haut les posent de la même façon. Ce que les couches montrent vient de `playerCardFx.ts`
 * (`CardFx`) et de `playerCardReadings.ts` (zones, objectif) : elles ne lisent rien d'autre.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { ReplayObjectiveMark } from './ReplayObjectiveMark'
import { SHROUD_BOLT_OFFSETS_S } from '../layers/placementShroud'
import type { ZonePresence } from '../model/equipmentZones'
import type { ObjectiveMarkKind } from '../model/objectiveMark'
import { hasUnderLayer, type CardFx } from '../model/playerCardFx'

/**
 * CardUnderLayers — LES COUCHES SOUS LE CONTENU d'une tuile, dans l'ordre de peinture du DOM :
 * la couche d'effets puis le filigrane, déclarés AVANT les rangées (qui sont `relative`) pour
 * peindre sous elles. `radiusClass` est le rayon de la tuile, en classe ; `watermarkPx` le côté
 * du filigrane (`CardGabarit.watermarkPx`).
 */
export function CardUnderLayers({
  fx,
  objective,
  radiusClass,
  watermarkPx,
}: {
  fx: CardFx
  objective: ObjectiveMarkKind | null
  radiusClass: string
  watermarkPx: number
}) {
  return (
    <>
      {/* LA COUCHE D'EFFETS ÉPOUSE LA TUILE (option 2a) : fonds, voiles, flou et cadres
          vivent sur cette couche `inset-0`, au rayon de la tuile (`radiusClass`), SOUS le
          contenu — les rangées sont en `relative` pour peindre au-dessus d'elle. Les éclats
          de mort/réapparition animent SON fond, jamais celui de la fiche. */}
      {hasUnderLayer(fx) && (
        <div
          aria-hidden
          className={`replay-card-fx pointer-events-none absolute inset-0 ${radiusClass} ${fx.flashClass}`}
          style={fx.underStyle}
        />
      )}
      {/* LE FILIGRANE DE PORTEUR : la couche du seul canal qui restait libre sur la fiche —
          derrière le contenu, sans toucher ni la bordure (trois cadres d'équipement) ni le
          fond (verre, voile, teinte de mort). */}
      {objective && <ReplayObjectiveMark kind={objective} sizePx={watermarkPx} radiusClass={radiusClass} />}
    </>
  )
}

/**
 * Décalages des trois croix du champ de réparation : désynchronisés (délais NÉGATIFS, donc
 * déjà en vol au premier rendu) pour qu'elles ne montent pas au pas. Trois, pas plus : la
 * fiche reste une fiche, l'effet un signe. Abscisses en % pur, glyphe de 10 px : sur une tuile
 * de 115 px elles tombent à 18 / 53 / 85 px, sans chevauchement — les trois restent.
 */
const REPAIR_CROSSES = [
  { left: '16%', delay: '-0.3s' },
  { left: '46%', delay: '-1.1s' },
  { left: '74%', delay: '-1.9s' },
] as const

/**
 * Les TROIS ÉCLAIRS de l'écran occultant (option 2a du handoff 2026-08-27) : abscisse,
 * largeur, et décalage de scintillement dans le cycle de 2,6 s (cf. globals.css,
 * `replay-zone-bolt`). Le décalage se SOUSTRAIT à l'horloge de la pose (`shroudSinceMs`) :
 * à la pose, les délais valent exactement 0 / −1,3 / −2,05 s — la composition de la
 * maquette — et après un saut de lecture chaque éclair reprend à son avancement réel,
 * jamais sur un rythme inventé (même contrat que le capteur et les éclats). Les décalages sont
 * ceux des éclairs de la bulle sur la carte (`placementShroud.SHROUD_BOLT_OFFSETS_S`) : la fiche et
 * la carte s'allument ensemble.
 *
 * LES LARGEURS SONT ABSOLUES (42 / 34 / 28 px) sur des abscisses en % : sur une tuile de
 * 115 px, trois éclairs se chevauchent. Le gabarit dit combien en rendre (`boltCount`) — les
 * deux premiers de la table sur la tuile compacte.
 */
const SHROUD_BOLTS = [
  { left: '10%', width: 42, offsetS: SHROUD_BOLT_OFFSETS_S[0] },
  { left: '56%', width: 34, offsetS: SHROUD_BOLT_OFFSETS_S[1] },
  { left: '33%', width: 28, offsetS: SHROUD_BOLT_OFFSETS_S[2] },
] as const

/**
 * ZoneFxOverlay — l'INCRUSTATION au-dessus du contenu : le nuage noir et les ÉCLAIRS de
 * l'écran occultant (« par-dessus les infos, mais légèrement »), le contour « détecté » du
 * capteur adverse, les mini croix du champ de réparation, et le FOURREAU de translocation —
 * la lumière qui court sur la bordure, en rotation continue jusqu'à son fondu. (Les fonds,
 * voiles et cadres, eux, vivent sur la couche d'effets SOUS le contenu :
 * playerCardFx.underStyle.)
 *
 * SUR SA PROPRE COUCHE, ET C'EST LE POINT : la couche du dessous anime déjà son fond (une
 * seule animation par élément et par propriété) — la pulsation du capteur, les éclairs et
 * le fourreau vivent donc chacun sur leur enfant. MÊME GÉOMÉTRIE que la couche du dessous
 * (`inset-0` au rayon de la tuile, `radiusClass` — option 2a) : les deux épousent la fiche,
 * jamais deux cadres décalés. `pointer-events-none` : l'infobulle et le survol restent
 * ceux de la fiche. `aria-hidden` : tout ce que l'incrustation montre est déjà dit en
 * texte par l'infobulle (title de la fiche).
 *
 * LE DÉLAI NÉGATIF DU CAPTEUR cale la pulsation sur l'horloge des pings du capteur le plus
 * fraîchement pingé (equipmentZones.sensorSincePingMs) : le contour de la fiche bat AVEC
 * l'onde de la carte, à la cadence officielle — jamais un rythme inventé. Ceux des éclairs
 * suivent l'horloge de la POSE de l'écran (shroudSinceMs) ; celui du fourreau, le contrat
 * des éclats (reprise à l'avancement réel après un saut de lecture).
 */
export function ZoneFxOverlay({
  zones,
  translocationDelay,
  boltCount,
  radiusClass,
}: {
  zones: ZonePresence
  translocationDelay: string | null
  /** Nombre d'éclairs à rendre, pris en tête de `SHROUD_BOLTS` (3 en normal, 2 en compact). */
  boltCount: number
  /** Le rayon de la tuile, en classe (un littéral en clair par gabarit) : l'incrustation l'épouse. */
  radiusClass: string
}) {
  const rien =
    !zones.repair && zones.shroudSinceMs === null && zones.sensorSincePingMs === null &&
    translocationDelay === null
  if (rien) return null
  return (
    <div aria-hidden className={`pointer-events-none absolute inset-0 overflow-hidden ${radiusClass}`}>
      {zones.shroudSinceMs !== null && (
        <>
          <div className="replay-zone-cloud absolute inset-0" />
          {SHROUD_BOLTS.slice(0, boltCount).map((b) => (
            <span
              key={b.left}
              className="replay-zone-bolt absolute top-[-8%] h-[116%]"
              style={{
                left: b.left,
                width: b.width,
                animationDelay: `${(-((zones.shroudSinceMs ?? 0) / 1000 + b.offsetS)).toFixed(3)}s`,
              }}
            />
          ))}
        </>
      )}
      {zones.sensorSincePingMs !== null && (
        <div
          className={`replay-zone-sensor absolute inset-0 ${radiusClass} border-[1.5px] border-dashed`}
          style={{
            borderColor: tokenCssVar('destructive'),
            animationDelay: `${(-zones.sensorSincePingMs / 1000).toFixed(3)}s`,
          }}
        />
      )}
      {zones.repair &&
        REPAIR_CROSSES.map((c) => (
          <span
            key={c.left}
            className="replay-zone-cross absolute top-[40%] text-[10px] font-bold leading-none"
            style={{ left: c.left, color: tokenCssVar('success'), animationDelay: c.delay }}
          >
            +
          </span>
        ))}
      {translocationDelay !== null && (
        <div
          className={`replay-flash-translocation absolute inset-0 ${radiusClass}`}
          style={{ animationDelay: translocationDelay }}
        />
      )}
    </div>
  )
}
